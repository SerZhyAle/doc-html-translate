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
	"errors"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"

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
}

// LineBox is one recognized line's rectangle in image pixels.
type LineBox struct {
	X0, Y0, X1, Y1 int
}

// Result is the OCR output for a single image.
type Result struct {
	Width, Height int
	Blocks        []Block
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
// next to the running executable (tesseract/tesseract.exe), then PATH.
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
func EngineLangs(bin string) ([]string, error) {
	// The list goes to stdout on some builds and to stderr on others, so both are read.
	out, err := exec.Command(bin, "--list-langs").CombinedOutput()
	if err != nil {
		return nil, err
	}
	return parseLangList(string(out)), nil
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
func MissingLangs(bin, lang string) []string {
	engine, err := EngineLangs(bin)
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
func Recognize(bin, imgPath, lang, dataDir string) (Result, error) {
	if lang == "" {
		lang = "eng"
	}

	ocrPath, scale, dpi, cleanup := prepareForOCR(imgPath)
	defer cleanup()

	res, err := recognizePass(bin, ocrPath, lang, dataDir, dpi, thresholdEngineDefault, ocrPageSegMode, ocrMinLineConf)
	if err != nil {
		return Result{}, err
	}
	if len(res.Blocks) == 0 {
		if alt, ok := greyRescue(bin, ocrPath, lang, dataDir, dpi); ok {
			res = alt
		}
	} else {
		res.Blocks = screenSweep(bin, ocrPath, lang, dataDir, dpi, res.Blocks)
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
func recognizePass(bin, ocrPath, lang, dataDir string, dpi, thresholding, psm int, minConf float64) (Result, error) {
	cmd := exec.Command(bin, tesseractArgs(ocrPath, lang, dataDir, dpi, thresholding, psm)...)
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		return Result{}, fmt.Errorf("tesseract: %w: %s", err, strings.TrimSpace(errb.String()))
	}
	return parseTSV(out.Bytes(), minConf)
}

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
func greyRescue(bin, ocrPath, lang, dataDir string, dpi int) (Result, bool) {
	grey := greyRendition(ocrPath)
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
		res, err := recognizePass(bin, greyPath, lang, dataDir, dpi, rung.thresholding, rung.psm, ocrRescueLineConf)
		if err == nil && strictlyBetter(res, best) {
			best = res
		}
	}
	if len(best.Blocks) > 0 {
		return best, true
	}
	return screenRescue(bin, grey, lang, dataDir, dpi)
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
func screenRescue(bin string, grey *image.Gray, lang, dataDir string, dpi int) (Result, bool) {
	pitch := screenPitch(grey)
	if pitch == 0 {
		return Result{}, false
	}
	path, cleanup, ok := writeTempPNG(gaussBlurGray(grey, float64(pitch)/ocrScreenSigmaDivisor))
	if !ok {
		return Result{}, false
	}
	defer cleanup()
	res, err := recognizePass(bin, path, lang, dataDir, dpi, thresholdEngineDefault, ocrPageSegMode, ocrRescueLineConf)
	if err != nil || len(res.Blocks) == 0 {
		return Result{}, false
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
func screenSweep(bin, ocrPath, lang, dataDir string, dpi int, kept []Block) []Block {
	grey := greyRendition(ocrPath)
	if grey == nil {
		return kept
	}
	// Both this and the ladder run before scaleDown, so every rectangle here is already in
	// prepared-image coordinates and nothing needs rescaling.
	pitch := screenPitchOutside(grey, blockRects(kept))
	if pitch == 0 {
		return kept
	}
	path, cleanup, ok := writeTempPNG(gaussBlurGray(grey, float64(pitch)/ocrScreenSigmaDivisor))
	if !ok {
		return kept
	}
	defer cleanup()
	res, err := recognizePass(bin, path, lang, dataDir, dpi, thresholdEngineDefault, ocrPageSegMode, ocrRescueLineConf)
	if err != nil {
		return kept
	}
	return mergeScreenBlocks(kept, res.Blocks)
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

// greyRendition returns an 8-bit luminance copy of the image, or nil when it cannot be decoded.
// What matters is that the channels agree, not the depth: measured, an 8-bit grey PNG and an RGB
// one whose three channels hold the same luminance recognize identically, because per-channel Otsu
// over three equal channels is one decision. 8-bit is simply the cheaper way to say it here - the
// extension gets there through a canvas filter and stays RGBA (docs/PARITY.md). The pixels are
// kept rather than only written out because the screen rung measures them (see screen.go).
func greyRendition(imgPath string) *image.Gray {
	src := decodeImage(imgPath)
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

// prepareForOCR returns the path to hand tesseract, the scale factor applied (to map coordinates
// back), the DPI to declare (0 = none), and a cleanup func (never nil). It upscales images whose
// estimated DPI is below the floor, and always resolves to an ASCII path: tesseract/leptonica open
// a path with the Windows ANSI codepage and mangle any byte outside it, so a book under a Cyrillic
// name would otherwise fail recognition silently. Best-effort: any failure recognizes the original.
func prepareForOCR(imgPath string) (path string, scale, dpi int, cleanup func()) {
	dpi = estimateDPI(imageLongSide(imgPath))
	upscale := dpi > 0 && dpi < ocrUpscaleDPIFloor
	// A photo carries its rotation in a tag rather than in its pixels, and recognition must be
	// given the picture the reader sees: a page shot in portrait stores its lettering on its side,
	// where PSM 3 (no OSD) reads none of it. Applying it here also means every coordinate comes
	// back in the space the plates are positioned in, so nothing downstream has to know (exif.go).
	orientation := exifOrientation(imgPath)
	if upscale || orientation != orientNormal {
		if p, cl, ok := stageForOCR(imgPath, orientation, upscale); ok {
			if upscale {
				return p, ocrUpscaleFactor, clampDeclaredDPI(dpi * ocrUpscaleFactor), cl
			}
			return p, 1, clampDeclaredDPI(dpi), cl
		}
	}
	staged, cl := stageASCIIPath(imgPath)
	return staged, 1, clampDeclaredDPI(dpi), cl
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
// with a cleanup func: the picture turned the way a reader sees it, and - for a genuinely low-res
// scan - enlarged ocrUpscaleFactor-fold so the lettering is legible to the engine.
//
// One decode for both, and the rotation first: enlarging is the expensive half, and doing it to
// pixels that are about to be moved wastes the work either way round. Best-effort - any
// decode/scale/encode failure returns ok=false and the caller recognizes the original.
func stageForOCR(imgPath string, orientation int, upscale bool) (path string, cleanup func(), ok bool) {
	src := decodeImage(imgPath)
	if src == nil {
		return "", nil, false
	}
	img := orientImage(src, orientation)
	if upscale {
		b := img.Bounds()
		if b.Dx() <= 0 || b.Dy() <= 0 {
			return "", nil, false
		}
		dst := image.NewRGBA(image.Rect(0, 0, b.Dx()*ocrUpscaleFactor, b.Dy()*ocrUpscaleFactor))
		xdraw.CatmullRom.Scale(dst, dst.Bounds(), img, b, xdraw.Over, nil)
		img = dst
	}
	return writeTempPNG(img)
}

// writeTempPNG encodes an image to a temp PNG on an ASCII path and returns it with a cleanup
// func, removing a half-written file on failure. Shared by the passes that hand tesseract a
// derived image rather than the user's file.
func writeTempPNG(img image.Image) (path string, cleanup func(), ok bool) {
	f, err := os.CreateTemp("", "docht-ocr-*.png")
	if err != nil {
		return "", nil, false
	}
	name := f.Name()
	if err := png.Encode(f, img); err != nil {
		f.Close()
		os.Remove(name)
		return "", nil, false
	}
	f.Close()
	return name, func() { os.Remove(name) }, true
}

// stageASCIIPath returns a path safe to hand tesseract. A path with non-ASCII bytes is copied to an
// ASCII temp file (tesseract/leptonica open it via the Windows ANSI codepage and mangle any byte
// outside it, failing recognition silently); an all-ASCII path is returned unchanged. The cleanup
// func removes any copy and is never nil. Mirrors internal/pdf's stagePDFForPDFToText.
func stageASCIIPath(imgPath string) (string, func()) {
	noop := func() {}
	if isASCIIPath(imgPath) {
		return imgPath, noop
	}
	ext := filepath.Ext(imgPath)
	if !isASCIIPath(ext) {
		ext = "" // tesseract sniffs the format from content; a mangled ext is worse than none
	}
	f, err := os.CreateTemp("", "docht-ocr-*"+ext)
	if err != nil {
		return imgPath, noop
	}
	name := f.Name()
	src, err := os.Open(imgPath)
	if err != nil {
		f.Close()
		os.Remove(name)
		return imgPath, noop
	}
	_, cerr := io.Copy(f, src)
	src.Close()
	f.Close()
	if cerr != nil {
		os.Remove(name)
		return imgPath, noop
	}
	return name, func() { os.Remove(name) }
}

func isASCIIPath(p string) bool {
	for _, r := range p {
		if r > 127 {
			return false
		}
	}
	return true
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
}

// hasLangFile reports whether every code in a "+"-joined lang string has a traineddata
// file in dir (so we only pin --tessdata-dir when it can actually satisfy the request).
func hasLangFile(dir, lang string) bool {
	for _, code := range strings.Split(lang, "+") {
		if _, err := os.Stat(filepath.Join(dir, code+".traineddata")); err != nil {
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
const (
	ocrMinLineConf        = 50
	ocrClusterPitchFactor = 1.2
	ocrMaxLeadingRatio    = 3
	ocrTypeSizeRatio      = 1.6
	ocrMaxPlateCoverage   = 0.52
	ocrMinPlateLineFill   = 0.72
)

// ocrLine is one recognized text line: its bounding box, the concatenated word text, and the
// running mean of its word confidences (used to reject noise).
type ocrLine struct {
	x0, y0, x1, y1 int
	text           strings.Builder
	confSum        float64
	confN          int
}

func (l *ocrLine) meanConf() float64 {
	if l.confN == 0 {
		return 0
	}
	return l.confSum / float64(l.confN)
}

// parseTSV turns tesseract's TSV output into a Result. Columns are:
// level page block par line word left top width height conf text
// level 1 = page (its size is the image size), 2 = block, 3 = paragraph, 4 = line, 5 = word.
//
// We read only the page size (level 1), the line boxes (level 4) and the words (level 5): the
// words give each line its text and confidence, and clusterLines groups the confident lines
// into plates by proximity. Block/paragraph boxes are ignored - trusting them makes an opaque
// plate span imagery the engine folded into a text paragraph (see clusterLines, docs/PARITY.md).
func parseTSV(data []byte, minConf float64) (Result, error) {
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)

	var res Result
	var lines []*ocrLine
	var cur *ocrLine

	firstLine := true
	for sc.Scan() {
		row := sc.Text()
		if firstLine {
			firstLine = false
			if strings.HasPrefix(row, "level\t") || strings.HasPrefix(row, "level ") {
				continue // header row
			}
		}
		cols := strings.Split(row, "\t")
		if len(cols) < 12 {
			continue
		}
		level, _ := strconv.Atoi(cols[0])
		left, _ := strconv.Atoi(cols[6])
		top, _ := strconv.Atoi(cols[7])
		w, _ := strconv.Atoi(cols[8])
		h, _ := strconv.Atoi(cols[9])

		switch level {
		case 1: // page: the image dimensions
			res.Width, res.Height = w, h
		case 4: // line: start a fresh accumulator
			cur = &ocrLine{x0: left, y0: top, x1: left + w, y1: top + h}
			lines = append(lines, cur)
		case 5: // word: fold text + confidence into the current line
			if cur == nil || strings.TrimSpace(cols[11]) == "" {
				continue
			}
			if cur.text.Len() > 0 {
				cur.text.WriteByte(' ')
			}
			cur.text.WriteString(cols[11])
			if conf, err := strconv.ParseFloat(cols[10], 64); err == nil {
				cur.confSum += conf
				cur.confN++
			}
		}
	}
	if err := sc.Err(); err != nil {
		return Result{}, err
	}

	res.Blocks = clusterLines(lines, minConf, res.Width, res.Height)
	return res, nil
}

// clusterLines drops low-confidence noise lines, then groups the survivors (in reading order)
// into one Block per run of vertically-adjacent, horizontally-overlapping lines. A plate's box
// is the union of its line boxes and its font tracks the median line height, so a plate covers a
// coherent text column without spanning the imagery or blank gaps between columns/sections.
// Mirrors the extension's ocr-cluster.js clusterLines - keep the two in sync (docs/PARITY.md).
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
	var kept []*ocrLine
	var heights []int
	for _, l := range lines {
		if l.text.Len() == 0 || l.meanConf() < minConf {
			continue
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
		cx0, cy0, cx1, cy1 int
		clastY0            int // top of the cluster's last line - the pitch is measured from it
		ctext              strings.Builder
		ctexts             []string // the same text, still split by line, for the coverage release
		cheights           []int
		clines             []LineBox
		open               bool
	)
	flush := func() {
		if !open {
			return
		}
		if txt := strings.TrimSpace(ctext.String()); isTranslatable(txt) {
			if over := releaseOversized(cx0, cy0, cx1, cy1, ctexts, clines, imgW, imgH); over != nil {
				blocks = append(blocks, over...)
			} else {
				blocks = append(blocks, Block{
					Text: txt, X0: cx0, Y0: cy0, X1: cx1, Y1: cy1,
					LineH: median(cheights, cy1-cy0),
					Lines: append([]LineBox(nil), clines...),
				})
			}
		}
		ctext.Reset()
		ctexts = ctexts[:0]
		cheights = cheights[:0]
		clines = clines[:0]
		open = false
	}
	for _, l := range kept {
		if open {
			gap := float64(l.y0 - cy1)
			pitch := float64(l.y0 - clastY0)
			overlap := min(l.x1, cx1) - max(l.x0, cx0)
			narrower := min(l.x1-l.x0, cx1-cx0)
			adjacent := gap <= gapMax
			if havePitch {
				adjacent = pitch <= pitchMax
			}
			// same column (share x-extent), same type size, and vertically adjacent (the line
			// keeps the page's pitch; a small negative gap tolerates overlapping boxes, a big one
			// means a new column).
			sameSize := sameTypeSize(l.y1-l.y0, median(cheights, 0))
			if adjacent && sameSize && gap >= -float64(medianH) && overlap*10 >= narrower {
				cx0, cy0 = min(cx0, l.x0), min(cy0, l.y0)
				cx1, cy1 = max(cx1, l.x1), max(cy1, l.y1)
				clastY0 = l.y0
				ctext.WriteByte(' ')
				ctext.WriteString(strings.TrimSpace(l.text.String()))
				ctexts = append(ctexts, strings.TrimSpace(l.text.String()))
				cheights = append(cheights, l.y1-l.y0)
				clines = append(clines, LineBox{X0: l.x0, Y0: l.y0, X1: l.x1, Y1: l.y1})
				continue
			}
			flush()
		}
		cx0, cy0, cx1, cy1 = l.x0, l.y0, l.x1, l.y1
		clastY0 = l.y0
		ctext.WriteString(strings.TrimSpace(l.text.String()))
		ctexts = append(ctexts, strings.TrimSpace(l.text.String()))
		cheights = append(cheights, l.y1-l.y0)
		clines = append(clines, LineBox{X0: l.x0, Y0: l.y0, X1: l.x1, Y1: l.y1})
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
