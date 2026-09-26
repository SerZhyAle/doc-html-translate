// Package ocr recognizes text baked into a document's images (via the Tesseract CLI) and
// overlays it as real, translatable HTML text positioned over each image. It mirrors the
// browser extension's OCR-overlay feature for the desktop app.
//
// The engine is the external `tesseract` binary: we shell out and parse its TSV output,
// which gives per-word/line/block bounding boxes we turn into positioned overlay plates.
// English data ships with the app; other languages are downloaded on demand (see
// tessdata.go). OCR is best-effort - a missing binary or a failed image never aborts the
// conversion.
package ocr

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/draw"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"doc-html-translate/internal/procrun"

	xdraw "golang.org/x/image/draw"
)

// Block is a recognized text block with its bounding box in image pixels. LineH is the
// representative (median) line height in the block, used to size the overlay font.
//
// Lines carries the block's own line boxes, in reading order. The overlay needs them because a
// block's bounding box is not the shape of the text it covers: centred copy narrows on its last
// line, and a plate drawn as the bounding rectangle paints the picture on both sides of it.
// Measured on a phone screenshot whose caption is 984 px wide on its first two lines and 759 on
// its third: the single rectangle covered 941 px throughout and put 91 px of paper over the
// photograph on either side of the last line.
type Block struct {
	Text           string
	X0, Y0, X1, Y1 int
	LineH          int
	Lines          []LineBox
	// Conf is the mean confidence of the block's lines. Only the discard record reads it: a plate
	// the screen merge rejects has no line left to take a confidence from.
	Conf float64
}

// LineBox is one recognized line's rectangle in image pixels.
type LineBox struct {
	X0, Y0, X1, Y1 int
}

// DroppedLine is a line the recognizer read and a gate threw away. It never reaches the page - it
// exists so the decision can be looked at.
//
// The gates are the places in the overlay where the app silently decides a reader does not get
// words the engine did read, and until this existed nothing said so: a scene where the poster's
// first word came back correctly at 69.2 and was discarded at a floor of 80 looked, from every
// output the app or the lab produced, exactly like a scene where the recognizer found nothing.
// Mirrors ocr-cluster.js `dropped` (docs/PARITY.md).
type DroppedLine struct {
	Text string
	Conf float64
	// Floor is the confidence floor of the pass that read the line. One image can be read several
	// times at two floors - the ordinary pass at ocrMinLineConf, the rescue ladder and the screen
	// sweep at ocrRescueLineConf - and a distribution derived from them mixed together would be a
	// distribution of nothing.
	Floor float64
	// Gate names the test that dropped it (gateConfidence, gateTranslatable, gateScreenMerge):
	// OCR-OVERLAY rule 12 asks which threshold failed, and three gates can drop a line that
	// cleared the same floor.
	Gate           string
	X0, Y0, X1, Y1 int
}

// The gates a DroppedLine names.
const (
	gateConfidence   = "confidence"   // the line's mean confidence is under the pass's floor (keepLine)
	gateTranslatable = "translatable" // its cluster has nothing to translate (isTranslatable)
	gateScreenMerge  = "screen-merge" // a screen-sweep plate over lettering already plated (mergeScreenBlocks)
)

// Result is the OCR output for a single image.
type Result struct {
	Width, Height int
	Blocks        []Block
	// Dropped carries the lines the confidence floor rejected, for diagnostics only. Nothing in
	// the rendering path reads it, and strictlyBetter does not weigh it - a rung's strength is
	// still the words it placed.
	Dropped []DroppedLine
}

// ErrNoTesseract is returned by Locate when no tesseract binary can be found.
var ErrNoTesseract = errors.New("tesseract executable not found (set DOCHT_TESSERACT, place it next to the app, or add it to PATH)")

func tesseractExeName() string {
	if runtime.GOOS == "windows" {
		return "tesseract.exe"
	}
	return "tesseract"
}

// Locate finds the tesseract executable: the DOCHT_TESSERACT env var, then a copy shipped
// next to the running executable (tesseract/tesseract.exe), then PATH. This order is pinned by
// OCR-INVOCATION.md section 2 - an embedder sets the variable and expects it to win.
func Locate() (string, error) {
	if p := os.Getenv("DOCHT_TESSERACT"); p != "" {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	if exe, err := os.Executable(); err == nil {
		cand := filepath.Join(filepath.Dir(exe), "tesseract", tesseractExeName())
		if _, err := os.Stat(cand); err == nil {
			return cand, nil
		}
	}
	if p, err := exec.LookPath("tesseract"); err == nil {
		return p, nil
	}
	return "", ErrNoTesseract
}

// EngineLangs asks the engine which languages it can actually load. The app's own tessdata
// directory is not the whole answer: a system Tesseract installed with its own language packs
// carries them elsewhere, and --tessdata-dir is pinned only when our directory can satisfy the
// request (see tesseractArgs). An error means "no idea" - never "none installed" - so a caller
// must not turn a failed probe into a claim about the user's machine.
func EngineLangs(ctx context.Context, bin string) ([]string, error) {
	// The list goes to stdout on some builds and to stderr on others, so both are read.
	res, err := runTesseract(ctx, procrun.TesseractProbe, bin, "", []string{"--list-langs"})
	if err != nil {
		return nil, err
	}
	return parseLangList(string(res.Stdout) + "\n" + string(res.Stderr)), nil
}

// parseLangList pulls the codes out of "tesseract --list-langs". Its first line names the
// tessdata directory and the count, so it carries spaces; every other line is one code, and
// codes may hold an underscore (jpn_vert) or a slash (script/Cyrillic).
func parseLangList(out string) []string {
	var langs []string
	for _, line := range strings.Split(out, "\n") {
		code := strings.TrimSpace(line)
		if code == "" || strings.ContainsAny(code, " \t") {
			continue
		}
		langs = append(langs, code)
	}
	return langs
}

// MissingLangs reports which codes of a "+"-joined language string the engine cannot load.
// A probe that failed returns nothing: a wrong "not installed" would send the user to
// download data they already have, which is worse than saying nothing.
func MissingLangs(ctx context.Context, bin, lang string) []string {
	engine, err := EngineLangs(ctx, bin)
	if err != nil {
		return nil
	}
	have := make(map[string]bool, len(engine)+4)
	for _, c := range engine {
		have[c] = true
	}
	// The app's own directory counts too: it is passed as --tessdata-dir precisely when it
	// holds the request, and the engine's own list does not mention it.
	for _, c := range Installed() {
		have[c] = true
	}
	var missing []string
	for _, code := range strings.Split(lang, "+") {
		if code = strings.TrimSpace(code); code != "" && !have[code] {
			missing = append(missing, code)
		}
	}
	return missing
}

// ocrPageSegMode pins Tesseract's page-segmentation mode. 3 = fully automatic page segmentation
// (no OSD) - the tesseract CLI default, made explicit so it stays a guarded shared value. The
// extension's tesseract.js otherwise defaults to PSM 6 (single block), which reads an illustrated or
// scanned page as one text block and folds scene edges into the recognized text; PSM 3 runs layout
// analysis and isolates real text regions. Mirrored by ocr-overlay.js OCR_PSM (see docs/PARITY.md).
const ocrPageSegMode = 3

// ocrSparsePageSegMode is Tesseract's PSM 11, "sparse text": find as much text as possible in no
// particular order, with no layout analysis behind it. It is a rescue rung rather than a default,
// because on a page it is worse than PSM 3 - it has no columns, no reading order and no notion of a
// paragraph, and wave 4 of the halftone work measured PSM 3 as the right mode for pages and comic
// balloons. What it is right for is the input that is not a page: a poster is a few large words
// placed for effect, and the layout analysis PSM 3 runs finds no page in it and drops them.
// Measured on poster-display-type-on-flat-colour with rus data and the app's own staging: the grey
// rungs at PSM 3 return one line above the rescue floor ("| МОЖЕМ", mean 85.5) while the same
// rendition at PSM 11 returns six (ТРАХАТЬСЯ: 80.7, МЫ ЖЕ 96.1, ЛЮДИ, 92.6, МОЖЕМ 95.9, ПРОСТО
// 87.2, ПОГОВОРИТЬ 95.0) with no debris. Mirrored by ocr-overlay.js (see docs/PARITY.md).
const ocrSparsePageSegMode = 11

// Recognize runs tesseract on imgPath for the given language and returns the recognized
// blocks plus the image's pixel dimensions. dataDir is passed as --tessdata-dir only when
// it actually contains the requested language, so a system tesseract still works when the
// app ships no bundled data.
//
// prepareForOCR decides how to feed the image to tesseract: it estimates the image's DPI,
// upscales genuinely low-res scans so recognition is legible, declares the resolution so
// Tesseract's layout analysis separates regions (e.g. speech balloons) instead of merging
// them, and always hands tesseract an ASCII path.
//
// A colour image that yields nothing is retried through the greyRescuePasses ladder and then the
// halftone-screen pass - see there for why a picture with plainly legible lettering can come back
// empty. An image that *did* read goes through screenSweep instead, which is the same low-pass spent
// on the parts of the picture no plate covers.
func Recognize(ctx context.Context, bin, imgPath, lang, dataDir string) (Result, error) {
	if lang == "" {
		lang = "eng"
	}

	warnOverBudget(imgPath)
	frame, scale, dpi, cleanup := prepareForOCR(imgPath)
	defer cleanup()

	// frame.grey is the luminance the line split reads for its stroke test (strokeBetween), handed
	// over unbuilt: recognizePass builds it once the recognizer has returned, which is the moment
	// the rescue ladder or screenSweep built it before the test existed - every image reaches it
	// either way - so the worker holds no more memory, for no longer, than it did.
	res, err := recognizePass(ctx, bin, frame.path, lang, dataDir, dpi, thresholdEngineDefault, ocrPageSegMode, ocrMinLineConf, frame.grey)
	if err != nil {
		return Result{}, err
	}
	if len(res.Blocks) == 0 {
		// The record is the ordinary pass's drops followed by the ladder's, whether or not the
		// ladder placed anything: the ordinary pass ran and its rejections are the reader's loss
		// either way, and the two stay separable because each line names its floor. When the
		// ladder read text and rejected all of it, that record is the only thing telling "the
		// floor rejected everything" from "there was nothing here".
		primary := res.Dropped
		alt, ok := greyRescue(ctx, bin, frame, lang, dataDir, dpi)
		if ok {
			res = alt
		}
		res.Dropped = append(primary, alt.Dropped...)
	} else {
		var swept []DroppedLine
		res.Blocks, swept = screenSweep(ctx, bin, frame, lang, dataDir, dpi, res.Blocks)
		res.Dropped = append(res.Dropped, swept...)
	}
	if scale > 1 {
		scaleDown(&res, scale)
	}
	return res, nil
}

// Tesseract's `thresholding_method`: 0 is its own Otsu (the engine default), 1 is Leptonica's
// tiled Otsu, which thresholds locally rather than picking one cut-off for the whole picture.
const (
	thresholdEngineDefault = 0
	thresholdLeptonicaOtsu = 1
)

// recognizePass runs one recognition attempt: a prepared image, a language, a declared DPI and a
// thresholding method. Splitting it out is what lets Recognize retry a picture that came back
// empty without re-deciding how the image was staged.
//
// ink supplies the luminance of the picture the passes read, for the line split's stroke test; it
// is asked for only after the recognizer has returned, and nil or a nil answer leaves the split to
// its ratio rule. Every pass reads the picture itself, not the grey or low-passed copy it may have
// handed the recognizer.
func recognizePass(ctx context.Context, bin, ocrPath, lang, dataDir string, dpi, thresholding, psm int, minConf float64, ink func() *image.Gray) (Result, error) {
	res, err := runTesseract(ctx, procrun.Tesseract, bin, ocrPath, tesseractArgs(ocrPath, lang, dataDir, dpi, thresholding, psm))
	if err != nil {
		return Result{}, err
	}
	// A truncated TSV parses into a page missing its lower half; better no plates than wrong ones.
	if res.StdoutTruncated {
		return Result{}, fmt.Errorf("tesseract: output larger than %d MB", procrun.DefaultMaxStdout>>20)
	}
	var plane *image.Gray
	if ink != nil {
		plane = ink()
	}
	return parseTSV(res.Stdout, minConf, plane)
}

// known hands recognizePass a luminance plane that already exists.
func known(g *image.Gray) func() *image.Gray { return func() *image.Gray { return g } }

// tesseractArgs builds one pass's command line.
func tesseractArgs(ocrPath, lang, dataDir string, dpi, thresholding, psm int) []string {
	args := []string{ocrPath, "stdout"}
	if dataDir != "" && hasLangFile(dataDir, lang) {
		args = append(args, "--tessdata-dir", dataDir)
	}
	// Request TSV via -c, not the `tsv` config file. That config lives in <tessdata>/configs/tsv,
	// but the app's bundled tessdata dir ships only traineddata (no configs/), and --tessdata-dir
	// redirects config lookup there - so `tsv` is not found ("read_params_file: Can't open tsv")
	// and tesseract falls back to plain text, which parseTSV can't read (zero blocks -> no overlay).
	// Setting the renderer flag directly is independent of any configs/ directory.
	args = append(args, "--psm", strconv.Itoa(psm), "-l", lang, "-c", "tessedit_create_tsv=1")
	// Declaring the resolution lets layout analysis separate regions Tesseract otherwise merges
	// (adjacent balloons read as one plate): its own estimate on a bare page scan runs far below
	// reality (~70 DPI for a ~180 DPI page). dpi==0 means we could not estimate one - leave it to guess.
	if dpi > 0 {
		args = append(args, "-c", "user_defined_dpi="+strconv.Itoa(dpi))
	}
	// Only named when it differs from the engine's own default, so the ordinary pass keeps the
	// exact command line it had before the rescue ladder existed.
	if thresholding != thresholdEngineDefault {
		args = append(args, "-c", "thresholding_method="+strconv.Itoa(thresholding))
	}
	return args
}

// greyRescuePasses are the retries for an image the ordinary pass could not read at all, in the
// order they are tried. Each hands tesseract an 8-bit luminance copy; they differ in who decides
// where ink ends and paper begins.
//
// Why greyscale rescues a picture whose lettering is perfectly legible to a person: Tesseract's
// default thresholder runs Otsu on each RGB channel separately and a pixel counts as ink only
// where every channel agrees. On flat paper the three channels agree and this is harmless, but on
// saturated artwork - a brick-red comic panel behind a white balloon, a coloured poster - each
// channel splits the picture somewhere else, the channels disagree over the lettering, and the
// mask that reaches recognition has no text in it. Feeding luminance makes it one decision instead
// of three. Measured on the lab's diagnostic scenes: the two speech-balloon scenes go from nothing
// recognized to their full transcript.
//
// The second pass then changes who thresholds. A single global cut-off cannot survive a background
// that varies across the image: on a sky gradient, Otsu splits the gradient itself, and the caption
// - drawn in white over the bright half - comes out the same value as its background and vanishes.
// Leptonica's tiled Otsu decides locally and reads it.
//
// The third rung changes neither the pixels nor the thresholder but what Tesseract is told to
// look for: sparse text instead of a page (see ocrSparsePageSegMode). It comes last of the three
// because it is the one that gives up layout analysis, which is what holds a real page's columns
// and reading order together.
//
// This is a ladder rather than a replacement because none of them is best everywhere: the colour
// pass wins on scenes where the lettering is separated by hue rather than brightness (a poster
// whose text and ground share a luminance), and greyscale throws that away. Retrying only after
// the ordinary pass returned no plates at all keeps every scene that works today byte-identical,
// and spends the extra passes only where the first one produced nothing to lose.
var greyRescuePasses = []rescueRung{
	{thresholding: thresholdEngineDefault, psm: ocrPageSegMode},
	{thresholding: thresholdLeptonicaOtsu, psm: ocrPageSegMode},
	{thresholding: thresholdEngineDefault, psm: ocrSparsePageSegMode},
}

// rescueRung is one retry: who decides where ink ends and paper begins, and what Tesseract is
// asked to find. Kept as a pair rather than as two parallel lists so a rung cannot half-exist.
type rescueRung struct {
	thresholding int
	psm          int
}

// greyRescue walks greyRescuePasses over a greyscale copy of the prepared image, then - for a
// picture whose lettering is printed on a halftone screen, which no thresholder can see through -
// tries the screen pass. It returns the strongest result any rung produced. ok=false when the copy
// cannot be built or nothing reads anything, and the caller then keeps the empty colour result,
// which is the honest answer.
//
// Strongest, not first-non-empty, and that distinction is the whole of the reported defect. The
// ladder used to stop at the first rung that returned any plate at all, so a rung that recovered
// one word ended the search before a later rung could recover six. Measured on
// poster-display-type-on-flat-colour: rung 1 returns "| МОЖЕМ" and used to win outright, while the
// sparse rung behind it returns six lines of the poster's lettering. The cost is that every rung
// now runs - up to two more shell-outs on an image the ordinary pass could not read at all, and
// none at all on an image that read.
//
// The screen pass is last because it is the only rung that changes the picture rather than the
// reading of it: a low-pass costs a little accuracy on lettering the earlier rungs can already
// read, so it is spent only after they have all failed.
func greyRescue(ctx context.Context, bin string, frame *ocrFrame, lang, dataDir string, dpi int) (Result, bool) {
	grey := frame.grey()
	if grey == nil {
		return Result{}, false
	}
	greyPath, cleanup, ok := writeTempPNG(grey)
	if !ok {
		return Result{}, false
	}
	defer cleanup()
	var best Result
	for _, rung := range greyRescuePasses {
		res, err := recognizePass(ctx, bin, greyPath, lang, dataDir, dpi, rung.thresholding, rung.psm, ocrRescueLineConf, known(grey))
		if err != nil {
			continue
		}
		switch {
		case strictlyBetter(res, best):
			best = res // the record travels with the rung that won, blocks and drops together
		case len(best.Blocks) == 0 && len(res.Dropped) > len(best.Dropped):
			// No rung has placed anything yet, so there is no winner to attach the record to.
			// The honest record is then the rung that *read* the most and had it all rejected -
			// which is the case this whole record exists for, and the one that used to be lost.
			best.Dropped = res.Dropped
		}
	}
	if len(best.Blocks) > 0 {
		return best, true
	}
	screened, ok := screenRescue(ctx, bin, grey, lang, dataDir, dpi)
	if ok {
		return screened, true
	}
	// Nothing read. Hand back whatever the ladder saw and had to throw away, so the caller can
	// tell "the floor rejected everything" from "there was nothing here".
	screened.Dropped = append(screened.Dropped, best.Dropped...)
	return screened, false
}

// screenRescue is the ladder's last rung: measure the halftone screen the picture is printed with
// and, if there is one, hand tesseract a copy low-passed just wide enough to dissolve it.
//
// Why this is a rung and not a filter applied to every image: the same blur that recovers text
// printed on a screen merges neighbouring words on clean lettering (measured - `THAT IS` came back
// as `THATIS` on a clean balloon), and on real screened material the ordinary passes usually read
// the text unaided. Firing only after every other rung returned nothing keeps that cost where
// there is nothing left to lose. The sigma is derived from the screen's own measured period rather
// than fixed, because a screen's pitch depends on the press and on the scan resolution, and a
// kernel tuned to one pitch is a fix for one pitch (see screen.go for the measurements).
func screenRescue(ctx context.Context, bin string, grey *image.Gray, lang, dataDir string, dpi int) (Result, bool) {
	pitch := screenPitch(grey)
	if pitch == 0 {
		return Result{}, false
	}
	path, cleanup, ok := writeTempPNG(gaussBlurGray(grey, float64(pitch)/ocrScreenSigmaDivisor))
	if !ok {
		return Result{}, false
	}
	defer cleanup()
	res, err := recognizePass(ctx, bin, path, lang, dataDir, dpi, thresholdEngineDefault, ocrPageSegMode, ocrRescueLineConf, known(grey))
	if err != nil {
		return Result{}, false
	}
	if len(res.Blocks) == 0 {
		// No plates, but possibly text the floor rejected: hand the record back, not the plates.
		return Result{Dropped: res.Dropped}, false
	}
	return res, true
}

// screenSweep is the screen pass for a page that already read. The rescue ladder cannot reach that
// page: it fires only for an image that produced no plates at all, and on a real comic the dialogue
// on clean white balloons reads fine while the caption printed as a tint does not. The page is never
// "unread", the rung never runs, and the caption stays untranslated with no explanation.
//
// So the pass is additive rather than a replacement. It has to be: measured over the corpus, the
// screen pass wins on screened material (+47% and +18% confident words on two real scenes, and a
// caption sentence that finally completes) and loses badly where there is no screen (16 confident
// words down to 0 on one cover, 71 down to 49 on a poster). Keeping every plate the ordinary pass
// produced and merging only what does not overlap one is what takes the gain without the loss.
// Measurements: DEV/research/ocr_halftone_2026-08-12.md.
//
// The trigger is screenPitchOutside rather than screenPitch, so the second recognition is spent only
// where there is screened area no plate covers. A screen under lettering that is already plated has
// nothing left to give, and the detector costs one sweep over at most ocrScreenMaxTiles tiles - far
// less than the recognition it decides against.
//
// Every failure returns the input untouched. A sweep that cannot run must leave the page exactly as
// it was, because the page was already good enough to show.
//
// The confidence floor is ocrRescueLineConf, inherited rather than re-derived, and the argument is
// that it is a lower bound: the local prior is the rescue prior - nothing was found *in this region*
// - while a wrong plate costs more here, since it lands on a page the reader is otherwise happy
// with. Turning it into a measured number needs annotated whole pages, which the corpus does not yet
// have (see the ticket's human-owned gate).
//
// No relaxed word rule applies here: the 2026-08-15 one was reverted, and the 2026-09-25 size rule
// never reached this pass - on the corpus its candidates here are mostly half-read masthead lines of
// a French page (DEV/research/ocr_rescue_third_axis_2026-09-25.md).
//
// The second result is the sweep's own discard record: the lines its floor and its translatability
// test rejected, and the plates the merge refused as duplicates.
func screenSweep(ctx context.Context, bin string, frame *ocrFrame, lang, dataDir string, dpi int, kept []Block) ([]Block, []DroppedLine) {
	grey := frame.grey()
	if grey == nil {
		return kept, nil
	}
	// Both this and the ladder run before scaleDown, so every rectangle here is already in
	// prepared-image coordinates and nothing needs rescaling.
	pitch := screenPitchOutside(grey, blockRects(kept))
	if pitch == 0 {
		return kept, nil
	}
	path, cleanup, ok := writeTempPNG(gaussBlurGray(grey, float64(pitch)/ocrScreenSigmaDivisor))
	if !ok {
		return kept, nil
	}
	defer cleanup()
	res, err := recognizePass(ctx, bin, path, lang, dataDir, dpi, thresholdEngineDefault, ocrPageSegMode, ocrRescueLineConf, known(grey))
	if err != nil {
		return kept, nil
	}
	merged, rejected := mergeScreenBlocks(kept, res.Blocks)
	return merged, append(res.Dropped, blockDrops(rejected, ocrRescueLineConf, gateScreenMerge)...)
}

// blockDrops records whole plates a gate refused, one entry per plate: by then the lines are
// merged into it and the plate is what was lost.
func blockDrops(blocks []Block, floor float64, gate string) []DroppedLine {
	out := make([]DroppedLine, 0, len(blocks))
	for _, b := range blocks {
		out = append(out, DroppedLine{
			Text: b.Text, Conf: b.Conf, Floor: floor, Gate: gate,
			X0: b.X0, Y0: b.Y0, X1: b.X1, Y1: b.Y1,
		})
	}
	return out
}

// ocrRescueLineConf is the line-confidence floor for a rescue pass, higher than ocrMinLineConf
// because a rescue is a second guess: the ordinary pass already looked at this image and found
// nothing, so the prior that there is text here at all is weaker, and a plate of invented words
// painted over artwork is worse for the reader than no overlay.
//
// Anchored on the same scale ocrMinLineConf uses. That gate sits at 50, the top of the band
// Tesseract hallucinates in; this one sits at the bottom of the band real text occupies (~80-97).
// Measured over the scenes the ladder newly reads: genuine rescued lettering scored 93.1-97.0,
// while the two Cyrillic posters read with English data - where the correct answer is no text -
// scored 50.8. The floor separates them with margin on both sides rather than splitting a gap.
const ocrRescueLineConf = 80

// **The floor was re-measured on 2026-08-15 and did not move, and the reason is worth keeping.**
// The 93.1 / 50.8 band above is two points from one cycle, and the gap it sits in is not empty any
// more. Measured with the floor's own discard record (Result.Dropped) over all 46 lab scenes plus
// the 13 annotated ones - DEV/research/ocr_rescue_floor_2026-08-15.md - the two populations
// **overlap on confidence**: genuine rescued lettering runs 32.8-69.2 and invented lettering runs
// 8.4-73.9. `ЗАЧЕМ`, the poster's correctly read first word, scores 69.2; `ОБ ЗЛОМ`, a misread,
// scores 73.9 above it. No value of a single floor separates them.
//
// The rule the distribution *did* support - keep a line under the floor when it carries a run of
// four letters and clears the middle of the empty band those lines bracket (36.1 / 58.3) - was
// implemented and run over the corpus, and **the corpus rejected it**. It recovers the poster's
// headline when the language is right, but under the app's default `eng` a Cyrillic poster then
// gets one plate of transliterated debris (`TPAXATBCR: 4 y`) 782x310 px over its own lettering,
// where it previously got none. That is the regression this floor exists to prevent, so the floor
// stays at 80 and the gap stays open.
//
// **A third axis was measured on 2026-09-25 and rejected too.** Keeping a sub-floor word only when
// its ink height matches a line of the same pass that cleared the floor (sameTypeSize) left every
// scene byte-identical under `eng` - under the wrong alphabet no real lettering clears the floor, so
// there is nothing to match - and brought the poster to recall 1.00 under `rus`. It still failed the
// lab's hard gates under `rus`: the admitted ОБ ЗЛОМ arrives out of order from the sparse rung and
// splits the body into two overlapping plates, and where the floor already trusts debris (an English
// balloon read with `rus`) the rule extends it across the next balloon onto a protected outline.
// DEV/research/ocr_rescue_third_axis_2026-09-25.md.
//

// greyRendition returns an 8-bit luminance copy of the image, or nil when it cannot be decoded.
// What matters is that the channels agree, not the depth: measured, an 8-bit grey PNG and an RGB
// one whose three channels hold the same luminance recognize identically, because per-channel Otsu
// over three equal channels is one decision. 8-bit is simply the cheaper way to say it here - the
// extension gets there through a canvas filter and stays RGBA (docs/PARITY.md). The pixels are
// kept rather than only written out because the screen rung measures them (see screen.go).
func greyRendition(imgPath string) *image.Gray { return greyOf(decodeImage(imgPath)) }

// greyOf is greyRendition for a picture already in memory; nil in, nil out.
func greyOf(src image.Image) *image.Gray {
	if src == nil {
		return nil
	}
	b := src.Bounds()
	if b.Dx() <= 0 || b.Dy() <= 0 {
		return nil
	}
	grey := image.NewGray(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(grey, grey.Bounds(), src, b.Min, draw.Src)
	return grey
}

// OCR resolution constants, shared with the extension's ocr-overlay.js (docs/PARITY.md). We do not
// gate on raw pixel count - a page scan is over 1000 px tall even at a poor ~100 DPI, so a pixel
// threshold either upscales everything (4x the OCR cost on a clean render that gains nothing) or
// nothing (the low-res scan that needs it most). Instead we estimate DPI from the long side against
// an assumed page height and act on that: below ocrUpscaleDPIFloor the image is enlarged
// ocrUpscaleFactor-fold before recognition (and coordinates divided back after), and in every case
// the resolution is declared to Tesseract (clamped to >= ocrMinDeclaredDPI). Measured: a ~90-DPI
// newsprint scan gains hugely from the upscale, while a ~150-DPI scan only needs the DPI declared -
// the upscale over-segments it for no benefit.
const (
	ocrUpscaleFactor     = 2
	ocrAssumedPageInches = 11.0 // assumed long-side page size (US Letter) for the DPI estimate
	ocrUpscaleDPIFloor   = 120  // estimated DPI below which an image is upscaled before OCR
	ocrMinDeclaredDPI    = 70   // never declare a DPI below this (Tesseract ignores sub-70 anyway)
)

// estimateDPI approximates an image's resolution from its long side, treating it as one
// ocrAssumedPageInches-tall page. 0 when the size is unknown. Crude, but enough to tell a low-res
// scan that needs enlarging from a mid-res one that only needs its DPI declared.
func estimateDPI(longSidePx int) int {
	if longSidePx <= 0 {
		return 0
	}
	return int(math.Round(float64(longSidePx) / ocrAssumedPageInches))
}

func clampDeclaredDPI(d int) int {
	if d > 0 && d < ocrMinDeclaredDPI {
		return ocrMinDeclaredDPI
	}
	return d
}

// prepareForOCR returns the frame to hand tesseract (its path, plus the staged picture when one
// was made in memory, so later passes need not decode it again), the scale factor applied (to map
// coordinates back), the DPI to declare (0 = none), and a cleanup func (never nil). It upscales images whose
// estimated DPI is below the floor, and always resolves to an ASCII path: tesseract/leptonica open
// a path with the Windows ANSI codepage and mangle any byte outside it, so a book under a Cyrillic
// name would otherwise fail recognition silently. Best-effort: any failure recognizes the original.
func prepareForOCR(imgPath string) (frame *ocrFrame, scale, dpi int, cleanup func()) {
	dpi = estimateDPI(imageLongSide(imgPath))
	upscale := dpi > 0 && dpi < ocrUpscaleDPIFloor
	// A photo carries its rotation in a tag rather than in its pixels, and recognition must be
	// given the picture the reader sees: a page shot in portrait stores its lettering on its side,
	// where PSM 3 (no OSD) reads none of it. Applying it here also means every coordinate comes
	// back in the space the plates are positioned in, so nothing downstream has to know (exif.go).
	orientation := exifOrientation(imgPath)
	if upscale || orientation != orientNormal {
		if p, img, cl, ok := stageForOCR(imgPath, orientation, upscale); ok {
			frame = &ocrFrame{path: p, img: img, tried: true}
			if upscale {
				return frame, ocrUpscaleFactor, clampDeclaredDPI(dpi * ocrUpscaleFactor), cl
			}
			return frame, 1, clampDeclaredDPI(dpi), cl
		}
	}
	staged, cl := stageASCIIPath(imgPath)
	return &ocrFrame{path: staged}, 1, clampDeclaredDPI(dpi), cl
}

// imageLongSide reads only the image header (cheap, no full decode) and returns its longer pixel
// dimension, or 0 if the size can't be read. Go opens the non-ASCII path fine - only the external
// tesseract binary can't, which stageASCIIPath handles.
func imageLongSide(path string) int {
	f, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer f.Close()
	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return 0
	}
	return max(cfg.Width, cfg.Height)
}

// stageForOCR writes the copy tesseract actually reads to a temp PNG (ASCII path), returning it
// with the picture itself (the later passes read that, not the PNG) and a cleanup func: the picture turned the way a reader sees it, and - for a genuinely low-res
// scan - enlarged ocrUpscaleFactor-fold so the lettering is legible to the engine.
//
// One decode for both, and the rotation first: enlarging is the expensive half, and doing it to
// pixels that are about to be moved wastes the work either way round. Best-effort - any
// decode/scale/encode failure returns ok=false and the caller recognizes the original.
func stageForOCR(imgPath string, orientation int, upscale bool) (path string, staged image.Image, cleanup func(), ok bool) {
	src := decodeImage(imgPath)
	if src == nil {
		return "", nil, nil, false
	}
	img := orientImage(src, orientation)
	if upscale {
		b := img.Bounds()
		if b.Dx() <= 0 || b.Dy() <= 0 {
			return "", nil, nil, false
		}
		dst := image.NewRGBA(image.Rect(0, 0, b.Dx()*ocrUpscaleFactor, b.Dy()*ocrUpscaleFactor))
		xdraw.CatmullRom.Scale(dst, dst.Bounds(), img, b, xdraw.Over, nil)
		img = dst
	}
	path, cleanup, ok = writeTempPNG(img)
	if !ok {
		return "", nil, nil, false
	}
	return path, img, cleanup, true
}

// scaleDown maps a Result recognized on an s-fold enlarged image back to the original pixel
// space, dividing every coordinate (and the reported dimensions) by s with rounding.
func scaleDown(res *Result, s int) {
	div := func(v int) int { return (v + s/2) / s }
	res.Width, res.Height = div(res.Width), div(res.Height)
	for i := range res.Blocks {
		bl := &res.Blocks[i]
		bl.X0, bl.Y0, bl.X1, bl.Y1 = div(bl.X0), div(bl.Y0), div(bl.X1), div(bl.Y1)
		bl.LineH = div(bl.LineH)
		for j := range bl.Lines {
			ln := &bl.Lines[j]
			ln.X0, ln.Y0, ln.X1, ln.Y1 = div(ln.X0), div(ln.Y0), div(ln.X1), div(ln.Y1)
		}
	}
	// The discard record is in the same prepared-image space, and a box left there would put a
	// rejected line at twice its place on the picture it is reported against (OCR-OVERLAY rule 2).
	for i := range res.Dropped {
		d := &res.Dropped[i]
		d.X0, d.Y0, d.X1, d.Y1 = div(d.X0), div(d.Y0), div(d.X1), div(d.Y1)
	}
}

// hasLangFile reports whether every code in a "+"-joined lang string has a traineddata
// file in dir (so we only pin --tessdata-dir when it can actually satisfy the request).
func hasLangFile(dir, lang string) bool {
	for _, code := range strings.Split(lang, "+") {
		if fi, err := os.Stat(filepath.Join(dir, code+".traineddata")); err != nil || !fi.Mode().IsRegular() {
			return false
		}
	}
	return true
}

// Overlay grouping constants, shared verbatim with the extension (see docs/PARITY.md and
// extension/src/ocr-cluster.js OCR_MIN_LINE_CONF / OCR_CLUSTER_PITCH_FACTOR / OCR_MAX_LEADING_RATIO).
//
// ocrMinLineConf drops a recognized line whose mean word confidence is below this. Real text
// scores ~80-97, while the "text" Tesseract hallucinates out of a drawing scores ~0-50, so a
// line-level confidence gate removes the noise that would otherwise become an opaque plate
// covering the figure (and whose oversized boxes inflate the plate font).
//
// ocrClusterPitchFactor then groups surviving lines into one plate while the next line keeps the
// running line pitch - the distance from one line's top to the next line's top - within this many
// reference pitches; a larger step (a figure, a section break, a new column) starts a new plate.
//
// The factor multiplies the *pitch* and not the height of the recognized ink box, because the ink
// box is not the line. All-caps lettering with no descenders boxes far shorter than the line it
// came from, and the difference is not marginal: measured on the lab scene synth-balloon-on-panel
// (2026-08-11), a balloon drawn with 36 px leading recognizes as 14-17 px ink boxes with 19-22 px
// between them, so a gap read against 1.2 x 14 px broke one balloon into three plates - three
// sentence fragments handed to a translator that has no sentence to work with. The same balloon's
// pitch is a flat 36 px, well inside 1.2 x 36. Pitch is also what keeps the opposite failure from
// being traded in: on synth-adjacent-balloons the pitch inside a balloon is 26 px and the step
// across to the next balloon is 42 px, so the two stay two plates.
//
// ocrMaxLeadingRatio bounds what may count as a line pitch when the reference is estimated: a step
// of more than this many median ink heights is a section break, not leading. Typography sets the
// scale - all-caps ink runs about 0.7 em and loose leading about 2 em, so real leading tops out
// near 2.9 ink heights - and the bound is what stops one big jump on a two-line page (a heading and
// a body paragraph far below it) from becoming the page's "typical" pitch and merging them.
//
// ocrTypeSizeRatio is the second half of that question, and it is the half pitch cannot answer. A
// page with separated regions - a poster, a form, a comic cover - hands the page-wide pitch estimate
// steps that belong to no single text, and a headline can then sit closer to the body than the body's
// own missing lines do. Measured on poster-display-type-on-flat-colour: the headline stands 168 px
// from the next recognized line while two body lines that a reader takes as one text stand 190 px
// apart, so no pitch bound separates them in the right place. Their *type sizes* do - the headline's
// ink is 281 px against the body's 155 px median (1.81x) - which is the distinction a reader uses.
//
// The value has to clear the spread a single text shows on its own and stay under the step between
// two texts. Both are measured on the corpus's hand-drawn line boxes, never on a recognizer's output:
// the widest within-group spread is samson-and-delilah-03-scroll, 19 lines of one caption whose ink
// runs 23-34 px, worst line 1.42x its own group's median; the narrowest across-group step is this
// poster's 1.86x (and synth-display-lettering's headline-to-footnote is 2.57x). 1.6 is the geometric
// middle of 1.42 and 1.81 - about 13% of margin on each side - rather than a round number chosen next
// to one of them.
//
// ocrMaxPlateCoverage and ocrMinPlateLineFill are the last pair, and together they answer a
// question none of the others can. Pitch and type size compare a line with its neighbours, so a
// page whose separated regions carry one type at one pitch - a form, a list, an application window
// - offers nothing in its typography to cut on, and the whole page arrives as a single plate. What
// separates that plate from a real one is that it is at once **too big and too loose**: it covers
// more of the picture than a text block plausibly can, and its own line boxes account for too
// little of the height it spans, because most of that height is the gaps between regions.
//
// Both conditions are required, and each is there because the other is not enough on its own. Size
// alone would release samson-and-delilah-03-scroll, a crop whose one hand-annotated caption covers
// 0.6087 of it. Looseness alone would release synth-uniform-paper, three lines of ordinary body
// text whose 0.6667 line fill is just the leading a paragraph has.
//
// The response is to release the cluster into its own line boxes rather than to drop it: a plate
// per line still carries every word (nothing recognized is lost, and no scene loses a plate), while
// what each one covers is the line it read instead of the rectangle spanning them all.
//
// Each bound is bracketed from opposite directions, measured over the 46 lab scenes and the 13
// hand-drawn annotations (DEV/research/ocr_plate_coverage_2026-08-13.md):
//
//   - coverage 0.52 - the largest legitimate plate the recognizer produces is that scroll caption
//     at 0.4004 of its image; the defect is accounts.jpg's window at 0.6829, six list rows and
//     their icons in one plate. 0.52 is the geometric middle, ~30% of margin each way.
//   - line fill 0.72 - the same scroll caption is the only annotated reading group whose own bounds
//     pass the coverage bound, and its hand-drawn lines fill 0.7921 of its height (0.9936 as the
//     recognizer boxes it); accounts.jpg's merged plate fills 0.6608. 0.72 is the geometric middle,
//     ~9% each way.
//
// The measure this was expected to be - how much of a plate's box *area* its lines fill - was
// measured on the same run and does not separate anything: accounts.jpg's merged plate fills 0.5891
// of its box while a legitimate balloon fills 0.4582 and a legitimate cartoon caption 0.3621. Area
// fill is a property of ragged right edges as much as of merging, so the rule is stated on the
// vertical axis, which is the axis separated regions are separated on.
//
// ocrMaxWordGapRatio is the one rule that runs before all of the above, because it repairs their
// *input* rather than their output: a "line" the recognizer handed us that is not one line.
//
// Every constant above compares one line with another, and none of them can do anything about the
// case where layout analysis walks across a picture and stitches a phrase from the left of the page
// to a phrase from the right into a single line box. Nothing downstream recovers: the stitched box
// genuinely spans both columns, so clusterLines' own column test (overlap*10 >= narrower) sees a
// real overlap and joins the columns, and ocrMaxPlateCoverage does not fire either, because the
// plate that comes out is wide but short - measured on test_doc/1.png, a 1593x105 px bar is 4% of a
// 2048x2048 image against a 0.52 bound. The reader gets a grey band across the artwork carrying two
// speakers' words in one sentence.
//
// So a line is cut between two consecutive words whose boxes stand more than this many times the
// line's own median word height apart. Word height rather than the line box, for the same reason
// inkHeight uses it: the box is the union of its words and one artefact sets it for the whole line.
//
// Bracketed over the 46 lab scenes plus test_doc/1.png, on the 199 multi-word lines that clear
// ocrMinLineConf - the only lines that can reach a plate (DEV/research/ocr_word_gap_2026-09-12.md):
//
//   - The widest gap inside a line that really is one line is 2.57x (the slogan on
//     join-the-ranks-of-the-red-army-russian-propaganda-poster-1920, 18 px over a 7 px median),
//     then 1.88x (the Olympiad poster's committee line) and 1.69x (le-petit-journal's headline).
//   - The narrowest cross-region stitch above it is 4.80x (samson-and-delilah-15, two balloons), and
//     the stitches this rule exists for are an order of magnitude clear of both: 12.4-17.6x on
//     synth-two-columns - the corpus's one known merge, named in thresholds.json - 19.9x on a
//     newspaper masthead, 38.6x on a dateline joined to a masthead, and 36-100x on test_doc/1.png.
//
// 3.5 is the geometric middle of 2.57 and 4.80, so every legitimate line keeps 36% of margin.
//
// The band 1.87-2.57x overlaps and no ratio separates it: comic balloons drawn side by side stitch
// at 1.87-3.04x while real lines run up to 2.57x, and a geometric rule must not pretend otherwise.
// Those are cut on evidence from the pixels between the two words instead - see strokeBetween and
// ocrBoundaryReach below.
//
// ocrBoundaryReach is how far past the words' band, above and below, a stroke has to run before the
// gap it crosses counts as a boundary - a fraction of the line's median word height, like the ratio
// above. It is what tells a balloon outline, which is drawn on past the line, from a letter the
// recognizer left out of its word box, which stays inside it.
//
// Bracketed over all 1043 word gaps of the lab corpus plus test_doc/1.png with the desktop engine,
// each gap the test's verdict depends on labelled by eye as a stitch between two regions or one real
// line (DEV/research/ocr_balloon_boundary_2026-09-25.md):
//
//   - At 0.07 and below a real line is cut: the J of le-petit-journal's "Petit Journal" masthead,
//     left out of its word box, descends that far below the line.
//   - From 0.08 to 0.25 no real line is cut and all 11 stitches under ocrMaxWordGapRatio are.
//   - At 0.30 the first stitch is lost (two paper notices side by side on a photographed wall), at
//     0.50 a balloon stitch, at 0.75 two more: an outline beside the first or last line of its
//     balloon turns away before it reaches that far.
//
// 0.14 is the geometric middle of 0.07 and 0.30, about 2x of margin each way.
//
// Shared invariant - see docs/PARITY.md and ocr-cluster.js OCR_MAX_WORD_GAP_RATIO.
const (
	ocrMinLineConf        = 50
	ocrClusterPitchFactor = 1.2
	ocrMaxLeadingRatio    = 3
	ocrTypeSizeRatio      = 1.6
	ocrMaxPlateCoverage   = 0.52
	ocrMinPlateLineFill   = 0.72
	ocrMaxWordGapRatio    = 3.5
	ocrBoundaryReach      = 0.14
)

// ocrLine is one recognized text line: its bounding box, the concatenated word text, and the
// running mean of its word confidences (used to reject noise).
type ocrLine struct {
	x0, y0, x1, y1 int
	text           strings.Builder
	confSum        float64
	confN          int
	wordH          []int     // each word's own box height, for inkHeight
	words          []ocrWord // the words themselves, for trimOutlierWords
	// The line's box with a non-text artefact trimmed off (see trimOutlierWords). It is the box a
	// plate is *drawn* from; every clustering decision still reads the untrimmed one, so trimming
	// can never change what reaches the page - only how far the paper spreads. Measured on
	// poster-display-type-on-flat-colour (2026-08-15): letting the trimmed box into the decisions
	// put two plates of transliterated debris ("NPOCTO", "0b 3TOM") over a legible Russian poster,
	// which is what the rescue floor exists to prevent.
	inkX0, inkY0, inkX1, inkY1 int
	// orphan marks a run a stroke cut off that cannot be a plate on its own (see splitWideGaps):
	// orderColumns parks it with the lines the confidence floor will drop.
	orphan bool
}

// ocrWord is one recognized word's box, text and confidence, kept only long enough to decide
// whether the line box it contributed to is really the line's - and, when it is not, to rebuild the
// lines it should have been (see splitWideGaps, which needs each run's own mean confidence).
//
// hasConf distinguishes "the engine scored this word 0" from "the engine gave no score": only a
// word that carries one may move the mean, exactly as parseTSV has always counted them.
type ocrWord struct {
	x0, y0, x1, y1 int
	text           string
	conf           float64
	hasConf        bool
}

// splitWideGaps cuts one recognizer line into the runs of words that belong to one another, at any
// horizontal step wider than ocrMaxWordGapRatio times the line's own median word height, and at any
// narrower step a stroke crosses (strokeBetween). It returns the line itself when there is nothing to
// cut, which is the answer on every ordinary line.
//
// The step is measured between the two boxes and not from left to right, so a right-to-left line is
// read the same way round as a left-to-right one instead of producing a negative gap on every pair
// and silently opting out of the rule. Overlapping boxes give a negative step and never cut.
//
// ink is the luminance of the picture the recognizer read, in its coordinates; nil (a picture that
// could not be decoded, or a test fixture with no pixels) leaves only the ratio rule.
// Mirrors ocr-cluster.js splitWideGaps (docs/PARITY.md).
func (l *ocrLine) splitWideGaps(ink *image.Gray) []*ocrLine {
	if len(l.words) < 2 {
		return []*ocrLine{l}
	}
	med := median(l.wordH, 0)
	if med <= 0 {
		return []*ocrLine{l}
	}
	maxGap := float64(med) * ocrMaxWordGapRatio
	reach := int(float64(med) * ocrBoundaryReach)

	var runs [][]ocrWord
	var byStroke []bool // one per cut: true when a stroke made it and the ratio would not have
	cur := []ocrWord{l.words[0]}
	for _, w := range l.words[1:] {
		prev := cur[len(cur)-1]
		wide := float64(max(w.x0-prev.x1, prev.x0-w.x1)) > maxGap
		if wide || strokeBetween(ink, prev, w, reach) {
			runs = append(runs, cur)
			byStroke = append(byStroke, !wide)
			cur = nil
		}
		cur = append(cur, w)
	}
	runs = append(runs, cur)
	if len(runs) < 2 {
		return []*ocrLine{l}
	}
	out := make([]*ocrLine, 0, len(runs))
	for i, run := range runs {
		r := lineFromWords(run)
		// A run a stroke cut off that could not be a plate on its own is the outline or the artwork
		// next to it, read as text: every one of the 8 such runs on the lab corpus is ("|", "}", "gs",
		// the speck "A" beside a balloon on atomicwar0401). Handed to the clustering where it stands,
		// it sits in its column at the height of the line it was cut from and breaks that line's plate
		// from the one above - measured on atomicwar0401, one balloon became two plates. So it is
		// marked, orderColumns parks it at the end, and the translatability gate records it there.
		if (i > 0 && byStroke[i-1] || i < len(byStroke) && byStroke[i]) && !isTranslatable(r.text.String()) {
			r.orphan = true
		}
		out = append(out, r)
	}
	return out
}

// lineFromWords rebuilds one line from a run of words. Its box is the union of the run's own words,
// because the box the engine gave is the stitch itself and would hand every run the full width back.
func lineFromWords(words []ocrWord) *ocrLine {
	l := &ocrLine{x0: words[0].x0, y0: words[0].y0, x1: words[0].x1, y1: words[0].y1}
	for _, w := range words {
		l.x0, l.y0 = min(l.x0, w.x0), min(l.y0, w.y0)
		l.x1, l.y1 = max(l.x1, w.x1), max(l.y1, w.y1)
		if l.text.Len() > 0 {
			l.text.WriteByte(' ')
		}
		l.text.WriteString(w.text)
		if h := w.y1 - w.y0; h > 0 {
			l.wordH = append(l.wordH, h)
		}
		if w.hasConf {
			l.confSum += w.conf
			l.confN++
		}
		l.words = append(l.words, w)
	}
	return l
}

// orderColumns is the other half of splitWideGaps, and without it the split trades one defect for
// another. clusterLines walks its input once and closes the open plate the moment a line does not
// belong to it, so it needs a column's lines to arrive together - which is exactly what the engine
// stops providing once a stitched line is cut in two. The runs then interleave left, right, left,
// right down the page, and measured on test_doc/1.png in the extension edition the left balloon that
// used to be one plate (inside one oversized bar) came back as three.
//
// So the page's runs are regrouped into columns - runs whose x-ranges overlap, by the same test
// clusterLines itself uses - and handed over column by column, each in vertical order. The caller
// runs this only for a page the split actually cut somewhere, so a page with no stitch on it keeps
// the engine's own order and cannot move at all.
//
// The scope is the page and not the paragraph, which is where this was first written. clusterLines
// deliberately merges across the paragraph boundaries the engine invents, and the engine invents
// them in the middle of a column: measured on synth-two-columns, the third row of both columns lands
// in a second paragraph, so regrouping each paragraph on its own still handed the clustering
// L1 L2 R1 R2 L3 R3 and produced four plates over two columns instead of two.
//
// Columns are formed from the runs that can actually reach a plate - the same confidence floor
// clusterLines applies - and a run that cannot is parked at the end, where nothing reads it but the
// discard record. A line the floor will drop must not decide where a column is: measured on
// test_doc/1.png, the engine also returns one empty 1982x1864 px "line" across the whole picture,
// and letting it into the grouping chains both columns into a single column, which sorts the page
// straight back into the interleaving this exists to undo. An orphan - the outline or artwork a
// stroke cut off a line, too little text to be a plate (see splitWideGaps) - is parked the same way.
//
// Columns go left to right, which is reading order for every script the corpus stitches. A
// right-to-left page would want them the other way round, and nothing measured says whether its
// layout analysis ever produces the stitch this repairs - so it is left as it is rather than guessed
// at. Mirrors ocr-cluster.js orderColumns (docs/PARITY.md).
func orderColumns(runs []*ocrLine, minConf float64) []*ocrLine {
	if len(runs) < 2 {
		return runs
	}
	type column struct {
		x0, x1 int
		runs   []*ocrLine
	}
	var byX, parked []*ocrLine
	for _, r := range runs {
		if keepLine(r, minConf) && !r.orphan {
			byX = append(byX, r)
		} else {
			parked = append(parked, r)
		}
	}
	sort.SliceStable(byX, func(i, j int) bool { return byX[i].x0 < byX[j].x0 })

	var cols []*column
	for _, r := range byX {
		var hit *column
		for _, c := range cols {
			overlap := min(r.x1, c.x1) - max(r.x0, c.x0)
			narrower := min(r.x1-r.x0, c.x1-c.x0)
			if overlap*10 >= narrower {
				hit = c
				break
			}
		}
		if hit == nil {
			cols = append(cols, &column{x0: r.x0, x1: r.x1, runs: []*ocrLine{r}})
			continue
		}
		hit.x0, hit.x1 = min(hit.x0, r.x0), max(hit.x1, r.x1)
		hit.runs = append(hit.runs, r)
	}
	sort.SliceStable(cols, func(i, j int) bool { return cols[i].x0 < cols[j].x0 })

	out := make([]*ocrLine, 0, len(runs))
	for _, c := range cols {
		sort.SliceStable(c.runs, func(i, j int) bool { return c.runs[i].y0 < c.runs[j].y0 })
		out = append(out, c.runs...)
	}
	return append(out, parked...)
}

// hasLetterOrDigit reports whether a token carries any actual content. A token that does not is
// punctuation, and punctuation is what a mis-read rule or outline comes back as.
func hasLetterOrDigit(s string) bool {
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return true
		}
	}
	return false
}

// trimOutlierWords shrinks a line box that a non-text artefact stretched.
//
// A balloon outline or a panel rule recognized beside real lettering comes back as a tall
// punctuation token - "|" - and the line box, being the union of its words, grows to hold it. That
// box is then the plate's box, so the plate reaches past the lettering onto artwork the annotation
// protects: measured in the extension edition on synth-adjacent-balloons (2026-08-15), the plate
// started at x=78 where its lettering starts at x=116, and the overhang scored 148 px of
// protected-area damage.
//
// Only a token with no letter or digit may be dropped, and only when it is taller than
// ocrTypeSizeRatio times the line's own median word height. Both conditions are needed: without the
// first this would delete short words on a line of display type, and without the second it would
// delete ordinary punctuation. A line that is nothing but such tokens keeps its box - there is then
// no lettering to measure against and nothing to shrink towards. Mirrors ocr-overlay.js
// trimOutlierWords (docs/PARITY.md).
func (l *ocrLine) trimOutlierWords() {
	l.inkX0, l.inkY0, l.inkX1, l.inkY1 = l.x0, l.y0, l.x1, l.y1
	if len(l.words) < 2 {
		return
	}
	med := median(l.wordH, 0)
	if med <= 0 {
		return
	}
	var kept []ocrWord
	for _, w := range l.words {
		h := w.y1 - w.y0
		if !hasLetterOrDigit(w.text) && float64(h) > float64(med)*ocrTypeSizeRatio {
			continue
		}
		kept = append(kept, w)
	}
	if len(kept) == 0 || len(kept) == len(l.words) {
		return
	}
	x0, y0, x1, y1 := kept[0].x0, kept[0].y0, kept[0].x1, kept[0].y1
	for _, w := range kept[1:] {
		x0, y0 = min(x0, w.x0), min(y0, w.y0)
		x1, y1 = max(x1, w.x1), max(y1, w.y1)
	}
	l.inkX0, l.inkY0, l.inkX1, l.inkY1 = x0, y0, x1, y1
}

// inkHeight is the line's type size, measured as the median of its words' box heights rather than
// as the height of the line box.
//
// The line box is the union of its words, so one tall artefact sets it for the whole line - and a
// balloon outline recognized as "|" beside real lettering is exactly that. Measured on
// synth-adjacent-balloons in the extension edition, 2026-08-15: the line "| NOT EVEN" boxes 37 px
// against 13 px for "SLIGHTLY." below it, a 2.85x step that ocrTypeSizeRatio (1.6) reads as two
// different type sizes, so one balloon became two plates and the taller box reached past the
// lettering onto the protected outline. The median of the word heights is 13 px, which is the size
// a reader sees. No constant moves: this changes what the ratio is measured on, the way
// ocrClusterPitchFactor changed from a gap to a pitch. Mirrors ocr-cluster.js lineInkHeight
// (docs/PARITY.md).
func (l *ocrLine) inkHeight() int {
	if h := median(l.wordH, 0); h > 0 {
		return h
	}
	return l.y1 - l.y0
}

func (l *ocrLine) meanConf() float64 {
	if l.confN == 0 {
		return 0
	}
	return l.confSum / float64(l.confN)
}

// tsvCols says where each field parseTSV reads sits in a TSV row. need is one past the highest index,
// so a row shorter than that is skipped rather than read out of range.
type tsvCols struct {
	level, left, top, width, height, conf, text, need int
}

// tsvFixedCols is the layout every Tesseract build to date writes. It is the fallback, not the
// primary read: a row is located through the header whenever the header names every field.
var tsvFixedCols = tsvCols{level: 0, left: 6, top: 7, width: 8, height: 9, conf: 10, text: 11, need: 12}

// tsvColsFromHeader maps the header's column names to indices. It reports false when any field is
// missing, and the caller keeps the fixed layout - a recognizer that reads a stable layout beats one
// that stops working because a header was renamed.
func tsvColsFromHeader(row string) (tsvCols, bool) {
	idx := map[string]int{}
	for i, name := range strings.Split(row, "\t") {
		name = strings.ToLower(strings.TrimSpace(name))
		if _, dup := idx[name]; !dup {
			idx[name] = i
		}
	}
	var c tsvCols
	for _, f := range []struct {
		name string
		dst  *int
	}{
		{"level", &c.level}, {"left", &c.left}, {"top", &c.top}, {"width", &c.width},
		{"height", &c.height}, {"conf", &c.conf}, {"text", &c.text},
	} {
		i, ok := idx[f.name]
		if !ok {
			return tsvCols{}, false
		}
		*f.dst = i
		c.need = max(c.need, i+1)
	}
	return c, true
}

// isTSVHeader tells the header row from a data row: every data row starts with a numeric level, so
// a first field that is not a number is a header, whatever column an engine build put first.
func isTSVHeader(row string) bool {
	first, _, _ := strings.Cut(row, "\t")
	first = strings.TrimSpace(first)
	if first == "" {
		return false
	}
	_, err := strconv.Atoi(first)
	return err != nil
}

// parseTSV turns tesseract's TSV output into a Result. Columns are, in every build to date:
// level page block par line word left top width height conf text
// They are read by the names in the header row (see tsvColsFromHeader), so an inserted column
// does not shift every field; without a usable header the layout above is assumed.
// level 1 = page (its size is the image size), 2 = block, 3 = paragraph, 4 = line, 5 = word.
//
// We read only the page size (level 1), the line boxes (level 4) and the words (level 5): the
// words give each line its text and confidence, and clusterLines groups the confident lines
// into plates by proximity. Block/paragraph boxes are ignored - trusting them makes an opaque
// plate span imagery the engine folded into a text paragraph (see clusterLines, docs/PARITY.md).
func parseTSV(data []byte, minConf float64, ink *image.Gray) (Result, error) {
	lines, w, h, err := tsvLines(data, minConf, ink)
	if err != nil {
		return Result{}, err
	}
	res := Result{Width: w, Height: h}

	// Same predicate clusterLines uses, so the record cannot drift away from the decision: a line
	// that carried text and did not clear the floor is one the reader lost.
	for _, l := range lines {
		if l.text.Len() > 0 && !keepLine(l, minConf) {
			res.Dropped = append(res.Dropped, lineDrop(l, minConf, gateConfidence))
		}
	}
	res.Blocks = clusterLinesRecording(lines, minConf, res.Width, res.Height, &res.Dropped)
	return res, nil
}

// tsvLines reads a TSV page into its lines - split, reordered and trimmed, before any floor is
// applied - plus the page's own size. minConf only steers orderColumns, which parks the lines the
// floor will drop.
func tsvLines(data []byte, minConf float64, ink *image.Gray) (lines []*ocrLine, width, height int, err error) {
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)

	var cur *ocrLine

	// A line is only complete once the next one starts, so this is where the split runs. Whether
	// anything was cut decides, at the end, if the page's reading order has to be rebuilt - see
	// splitWideGaps and orderColumns.
	split := false
	closeLine := func() {
		if cur == nil {
			return
		}
		runs := cur.splitWideGaps(ink)
		if len(runs) > 1 {
			split = true
		}
		lines = append(lines, runs...)
		cur = nil
	}

	col := tsvFixedCols
	firstLine := true
	for sc.Scan() {
		row := sc.Text()
		if firstLine {
			firstLine = false
			if isTSVHeader(row) {
				if c, ok := tsvColsFromHeader(row); ok {
					col = c
				}
				continue
			}
		}
		cols := strings.Split(row, "\t")
		if len(cols) < col.need {
			continue
		}
		level, _ := strconv.Atoi(cols[col.level])
		left, _ := strconv.Atoi(cols[col.left])
		top, _ := strconv.Atoi(cols[col.top])
		w, _ := strconv.Atoi(cols[col.width])
		h, _ := strconv.Atoi(cols[col.height])
		text := cols[col.text]

		switch level {
		case 1: // page: the image dimensions
			width, height = w, h
		case 4: // line: close the one before it, then start a fresh accumulator
			closeLine()
			cur = &ocrLine{x0: left, y0: top, x1: left + w, y1: top + h}
		case 5: // word: fold text + confidence into the current line
			if cur == nil || strings.TrimSpace(text) == "" {
				continue
			}
			if cur.text.Len() > 0 {
				cur.text.WriteByte(' ')
			}
			cur.text.WriteString(text)
			if h > 0 {
				cur.wordH = append(cur.wordH, h)
			}
			word := ocrWord{x0: left, y0: top, x1: left + w, y1: top + h, text: text}
			if conf, err := strconv.ParseFloat(cols[col.conf], 64); err == nil {
				word.conf, word.hasConf = conf, true
				cur.confSum += conf
				cur.confN++
			}
			cur.words = append(cur.words, word)
		}
	}
	if err := sc.Err(); err != nil {
		return nil, 0, 0, err
	}
	closeLine()
	if split {
		lines = orderColumns(lines, minConf)
	}
	for _, l := range lines {
		l.trimOutlierWords()
	}
	return lines, width, height, nil
}

func lineDrop(l *ocrLine, floor float64, gate string) DroppedLine {
	return DroppedLine{
		Text: strings.TrimSpace(l.text.String()), Conf: l.meanConf(), Floor: floor, Gate: gate,
		X0: l.x0, Y0: l.y0, X1: l.x1, Y1: l.y1,
	}
}

// keepLine is the confidence floor, in one place. clusterLines applies it and parseTSV records
// what it rejected; if the two ever stated it separately, the record would stop describing the
// decision the first time either moved.
func keepLine(l *ocrLine, minConf float64) bool {
	if l.text.Len() == 0 {
		return false
	}
	return l.meanConf() >= minConf
}

// clusterLines drops low-confidence noise lines, then groups the survivors (in reading order)
// into one Block per run of vertically-adjacent, horizontally-overlapping lines. A plate's box
// is the union of its line boxes and its font tracks the median line height, so a plate covers a
// coherent text column without spanning the imagery or blank gaps between columns/sections.
// Mirrors the extension's ocr-cluster.js clusterLines - keep the two in sync (docs/PARITY.md).
// Grouping by line pitch rather than by the gap between ink boxes is OCR-OVERLAY rule 6.
//
// Vertical adjacency is judged on the line pitch (see ocrClusterPitchFactor). A page whose lines
// yield no measurable pitch at all - one line, or nothing but lines that share no column - has no
// reference to compare against, and falls back to the ink-box gap this test used before.
//
// imgW/imgH are the page's own dimensions, used only by the coverage rule (see
// ocrMaxPlateCoverage): a finished cluster that covers more of the picture than a text block
// plausibly can is released into its own lines. Zero on either means the page size is unknown and
// the rule does not run.
func clusterLines(lines []*ocrLine, minConf float64, imgW, imgH int) []Block {
	return clusterLinesRecording(lines, minConf, imgW, imgH, nil)
}

// clusterLinesRecording is clusterLines that also appends to *dropped, when given, every line of a
// cluster the translatability test refused - recorded where the decision is taken, so the record
// is the decision and not a second copy of it.
func clusterLinesRecording(lines []*ocrLine, minConf float64, imgW, imgH int, dropped *[]DroppedLine) []Block {
	var kept []*ocrLine
	var heights []int
	for _, l := range lines {
		if !keepLine(l, minConf) {
			continue
		}
		// A line that never went through trimOutlierWords (a rescue path, a test fixture) has no
		// trimmed box; its own box is then the box a plate is drawn from.
		if l.inkX1 <= l.inkX0 || l.inkY1 <= l.inkY0 {
			l.inkX0, l.inkY0, l.inkX1, l.inkY1 = l.x0, l.y0, l.x1, l.y1
		}
		kept = append(kept, l)
		heights = append(heights, l.y1-l.y0)
	}
	if len(kept) == 0 {
		return nil
	}
	medianH := median(heights, 1)
	refPitch, havePitch := medianLinePitch(kept, medianH)
	pitchMax := float64(refPitch) * ocrClusterPitchFactor
	gapMax := float64(medianH) * ocrClusterPitchFactor

	var blocks []Block
	var (
		cx0, cy0, cx1, cy1 int // the cluster's drawn box, built from trimmed line boxes
		sx0, sy0, sx1, sy1 int // the same lines' untrimmed span, which every decision below reads
		clastY0            int // top of the cluster's last line - the pitch is measured from it
		ctext              strings.Builder
		ctexts             []string // the same text, still split by line, for the coverage release
		cheights           []int
		cink               []int // the same lines' ink heights, for the type-size test only
		clines             []LineBox
		cmembers           []*ocrLine // the lines themselves, for their confidences
		open               bool
	)
	flush := func() {
		if !open {
			return
		}
		if txt := strings.TrimSpace(ctext.String()); isTranslatable(txt) {
			if over := releaseOversized(cx0, cy0, cx1, cy1, ctexts, clines, imgW, imgH); over != nil {
				// A released plate is one member line; its box finds which (releaseOversized skips
				// textless lines, so the positions do not line up).
				for i := range over {
					if j := slices.Index(clines, over[i].Lines[0]); j >= 0 {
						over[i].Conf = cmembers[j].meanConf()
					}
				}
				blocks = append(blocks, over...)
			} else {
				conf := 0.0
				for _, m := range cmembers {
					conf += m.meanConf()
				}
				blocks = append(blocks, Block{
					Text: txt, X0: cx0, Y0: cy0, X1: cx1, Y1: cy1,
					LineH: median(cheights, cy1-cy0),
					Lines: append([]LineBox(nil), clines...),
					Conf:  conf / float64(len(cmembers)),
				})
			}
		} else if dropped != nil {
			for _, m := range cmembers {
				*dropped = append(*dropped, lineDrop(m, minConf, gateTranslatable))
			}
		}
		ctext.Reset()
		ctexts = ctexts[:0]
		cheights = cheights[:0]
		cink = cink[:0]
		clines = clines[:0]
		cmembers = cmembers[:0]
		open = false
	}
	for _, l := range kept {
		if open {
			gap := float64(l.y0 - sy1)
			pitch := float64(l.y0 - clastY0)
			overlap := min(l.x1, sx1) - max(l.x0, sx0)
			narrower := min(l.x1-l.x0, sx1-sx0)
			adjacent := gap <= gapMax
			if havePitch {
				adjacent = pitch <= pitchMax
			}
			// same column (share x-extent), same type size, and vertically adjacent (the line
			// keeps the page's pitch; a small negative gap tolerates overlapping boxes, a big one
			// means a new column).
			sameSize := sameTypeSize(l.inkHeight(), median(cink, 0))
			if adjacent && sameSize && gap >= -float64(medianH) && overlap*10 >= narrower {
				// The plate's box grows by the trimmed line box, never by the artefact's reach; the
				// span grows by the untrimmed one so the next line is judged as it was before.
				cx0, cy0 = min(cx0, l.inkX0), min(cy0, l.inkY0)
				cx1, cy1 = max(cx1, l.inkX1), max(cy1, l.inkY1)
				sx0, sy0 = min(sx0, l.x0), min(sy0, l.y0)
				sx1, sy1 = max(sx1, l.x1), max(sy1, l.y1)
				clastY0 = l.y0
				ctext.WriteByte(' ')
				ctext.WriteString(strings.TrimSpace(l.text.String()))
				ctexts = append(ctexts, strings.TrimSpace(l.text.String()))
				cheights = append(cheights, l.y1-l.y0)
				cink = append(cink, l.inkHeight())
				clines = append(clines, LineBox{X0: l.inkX0, Y0: l.inkY0, X1: l.inkX1, Y1: l.inkY1})
				cmembers = append(cmembers, l)
				continue
			}
			flush()
		}
		cx0, cy0, cx1, cy1 = l.inkX0, l.inkY0, l.inkX1, l.inkY1
		sx0, sy0, sx1, sy1 = l.x0, l.y0, l.x1, l.y1
		clastY0 = l.y0
		ctext.WriteString(strings.TrimSpace(l.text.String()))
		ctexts = append(ctexts, strings.TrimSpace(l.text.String()))
		cheights = append(cheights, l.y1-l.y0)
		cink = append(cink, l.inkHeight())
		clines = append(clines, LineBox{X0: l.inkX0, Y0: l.inkY0, X1: l.inkX1, Y1: l.inkY1})
		cmembers = append(cmembers, l)
		open = true
	}
	flush()
	return blocks
}

// releaseOversized turns a cluster that is both too big and too loose (see ocrMaxPlateCoverage and
// ocrMinPlateLineFill) into one block per line, and returns nil when the cluster passes either
// test - the ordinary case, and the case on every scene of the lab corpus.
//
// Releasing rather than refusing is the whole point: every recognized word still reaches a plate,
// so no scene loses text, while the area the overlay paints drops from the rectangle spanning the
// regions to the lines themselves. The released blocks skip isTranslatable deliberately - the
// cluster's assembled text has already passed it, and re-testing each line separately would drop
// the short ones and turn a composition fix into a recall regression.
//
// A single-line cluster is never released: its box is its line, so there is nothing to release it
// into and the coverage is a fact about the picture rather than about the grouping.
func releaseOversized(x0, y0, x1, y1 int, texts []string, lines []LineBox, imgW, imgH int) []Block {
	if imgW <= 0 || imgH <= 0 || len(lines) < 2 || len(texts) != len(lines) {
		return nil
	}
	boxH := y1 - y0
	box := float64(x1-x0) * float64(boxH)
	if box <= 0 || boxH <= 0 || box <= float64(imgW)*float64(imgH)*ocrMaxPlateCoverage {
		return nil
	}
	ink := 0
	for _, ln := range lines {
		ink += ln.Y1 - ln.Y0
	}
	if float64(ink) >= float64(boxH)*ocrMinPlateLineFill {
		return nil
	}
	out := make([]Block, 0, len(lines))
	for i, ln := range lines {
		txt := strings.TrimSpace(texts[i])
		if txt == "" {
			continue
		}
		out = append(out, Block{
			Text: txt, X0: ln.X0, Y0: ln.Y0, X1: ln.X1, Y1: ln.Y1,
			LineH: ln.Y1 - ln.Y0,
			Lines: []LineBox{ln},
		})
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// sameTypeSize reports whether a line's ink height is close enough to a cluster's own to be part of
// the same text (see ocrTypeSizeRatio). The comparison is against the cluster's median rather than
// its previous line, so one odd box - a line of caps with no descenders, a line that is a single
// short word - cannot end a plate by itself.
//
// A missing height is not evidence of a break: a zero on either side keeps the line, because the
// clustering must never become stricter through an unmeasured quantity.
func sameTypeSize(h, clusterH int) bool {
	if h <= 0 || clusterH <= 0 {
		return true
	}
	return float64(max(h, clusterH)) <= float64(min(h, clusterH))*ocrTypeSizeRatio
}

// medianLinePitch estimates the page's line pitch from the tops of successive kept lines.
//
// The unit is the image, not the cluster: a cluster's own pitch is unavailable exactly when the
// decision is hardest - joining its second line, where there is no pitch yet - and that is the very
// join that split the balloon. Taking the median over the page instead makes the estimate available
// from the first decision, and the median is what keeps a page carrying two type sizes honest: the
// steps between the sizes are outliers around the body text's pitch, not the middle of it.
//
// A pair contributes only when it could plausibly be one text's leading: same column (the test the
// clustering itself uses), moving forward, and no further than ocrMaxLeadingRatio ink heights. Both
// bounds are there to keep a step that is really a section break out of the estimate - without them
// a page holding one heading and one distant paragraph would make that single jump the median and
// merge the two.
func medianLinePitch(kept []*ocrLine, medianH int) (int, bool) {
	maxPitch := float64(medianH) * ocrMaxLeadingRatio
	var pitches []int
	for i := 1; i < len(kept); i++ {
		prev, l := kept[i-1], kept[i]
		dy := l.y0 - prev.y0
		if dy <= 0 || float64(dy) > maxPitch {
			continue
		}
		overlap := min(l.x1, prev.x1) - max(l.x0, prev.x0)
		narrower := min(l.x1-l.x0, prev.x1-prev.x0)
		if overlap*10 < narrower {
			continue
		}
		pitches = append(pitches, dy)
	}
	if len(pitches) == 0 {
		return 0, false
	}
	return median(pitches, 0), true
}

func median(vals []int, fallback int) int {
	if len(vals) == 0 {
		return fallback
	}
	s := append([]int(nil), vals...)
	sort.Ints(s)
	return s[len(s)/2]
}
