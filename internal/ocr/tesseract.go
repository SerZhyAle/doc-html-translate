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
type Block struct {
	Text           string
	X0, Y0, X1, Y1 int
	LineH          int
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

// ocrPageSegMode pins Tesseract's page-segmentation mode. 3 = fully automatic page segmentation
// (no OSD) - the tesseract CLI default, made explicit so it stays a guarded shared value. The
// extension's tesseract.js otherwise defaults to PSM 6 (single block), which reads an illustrated or
// scanned page as one text block and folds scene edges into the recognized text; PSM 3 runs layout
// analysis and isolates real text regions. Mirrored by ocr-overlay.js OCR_PSM (see docs/PARITY.md).
const ocrPageSegMode = 3

// Recognize runs tesseract on imgPath for the given language and returns the recognized
// blocks plus the image's pixel dimensions. dataDir is passed as --tessdata-dir only when
// it actually contains the requested language, so a system tesseract still works when the
// app ships no bundled data.
//
// prepareForOCR decides how to feed the image to tesseract: it estimates the image's DPI,
// upscales genuinely low-res scans so recognition is legible, declares the resolution so
// Tesseract's layout analysis separates regions (e.g. speech balloons) instead of merging
// them, and always hands tesseract an ASCII path.
func Recognize(bin, imgPath, lang, dataDir string) (Result, error) {
	if lang == "" {
		lang = "eng"
	}

	ocrPath, scale, dpi, cleanup := prepareForOCR(imgPath)
	defer cleanup()

	args := []string{ocrPath, "stdout"}
	if dataDir != "" && hasLangFile(dataDir, lang) {
		args = append(args, "--tessdata-dir", dataDir)
	}
	// Request TSV via -c, not the `tsv` config file. That config lives in <tessdata>/configs/tsv,
	// but the app's bundled tessdata dir ships only traineddata (no configs/), and --tessdata-dir
	// redirects config lookup there - so `tsv` is not found ("read_params_file: Can't open tsv")
	// and tesseract falls back to plain text, which parseTSV can't read (zero blocks -> no overlay).
	// Setting the renderer flag directly is independent of any configs/ directory.
	args = append(args, "--psm", strconv.Itoa(ocrPageSegMode), "-l", lang, "-c", "tessedit_create_tsv=1")
	// Declaring the resolution lets layout analysis separate regions Tesseract otherwise merges
	// (adjacent balloons read as one plate): its own estimate on a bare page scan runs far below
	// reality (~70 DPI for a ~180 DPI page). dpi==0 means we could not estimate one - leave it to guess.
	if dpi > 0 {
		args = append(args, "-c", "user_defined_dpi="+strconv.Itoa(dpi))
	}

	cmd := exec.Command(bin, args...)
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		return Result{}, fmt.Errorf("tesseract: %w: %s", err, strings.TrimSpace(errb.String()))
	}
	res, err := parseTSV(out.Bytes())
	if err != nil {
		return Result{}, err
	}
	if scale > 1 {
		scaleDown(&res, scale)
	}
	return res, nil
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
	if dpi > 0 && dpi < ocrUpscaleDPIFloor {
		if up, cl, ok := upscaleForOCR(imgPath); ok {
			return up, ocrUpscaleFactor, clampDeclaredDPI(dpi * ocrUpscaleFactor), cl
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

// upscaleForOCR writes an ocrUpscaleFactor-enlarged copy of the image to a temp PNG (ASCII path) so
// Tesseract reads it better, returning the temp path and a cleanup func. Best-effort: any
// decode/scale/encode failure returns ok=false (recognize the original).
func upscaleForOCR(imgPath string) (path string, cleanup func(), ok bool) {
	src := decodeImage(imgPath)
	if src == nil {
		return "", nil, false
	}
	b := src.Bounds()
	if b.Dx() <= 0 || b.Dy() <= 0 {
		return "", nil, false
	}
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx()*ocrUpscaleFactor, b.Dy()*ocrUpscaleFactor))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, b, xdraw.Over, nil)
	f, err := os.CreateTemp("", "docht-ocr-*.png")
	if err != nil {
		return "", nil, false
	}
	if err := png.Encode(f, dst); err != nil {
		f.Close()
		os.Remove(f.Name())
		return "", nil, false
	}
	f.Close()
	name := f.Name()
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
// extension/src/ocr-overlay.js OCR_MIN_LINE_CONF / OCR_CLUSTER_GAP_FACTOR).
//
// ocrMinLineConf drops a recognized line whose mean word confidence is below this. Real text
// scores ~80-97, while the "text" Tesseract hallucinates out of a drawing scores ~0-50, so a
// line-level confidence gate removes the noise that would otherwise become an opaque plate
// covering the figure (and whose oversized boxes inflate the plate font). ocrClusterGapFactor
// then groups surviving lines into one plate while the vertical gap to the next line stays
// within this many median line heights; a larger gap - a figure, a section break, a new column
// - starts a new plate. Together they turn Tesseract's unstable block/paragraph boxes into
// plates that (a) never cover imagery and (b) don't splinter one uniform text column.
const (
	ocrMinLineConf      = 50
	ocrClusterGapFactor = 1.2
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
func parseTSV(data []byte) (Result, error) {
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

	res.Blocks = clusterLines(lines)
	return res, nil
}

// clusterLines drops low-confidence noise lines, then groups the survivors (in reading order)
// into one Block per run of vertically-adjacent, horizontally-overlapping lines. A plate's box
// is the union of its line boxes and its font tracks the median line height, so a plate covers a
// coherent text column without spanning the imagery or blank gaps between columns/sections.
// Mirrors the extension's ocr-overlay.js clusterLines - keep the two in sync (docs/PARITY.md).
func clusterLines(lines []*ocrLine) []Block {
	var kept []*ocrLine
	var heights []int
	for _, l := range lines {
		if l.text.Len() == 0 || l.meanConf() < ocrMinLineConf {
			continue
		}
		kept = append(kept, l)
		heights = append(heights, l.y1-l.y0)
	}
	if len(kept) == 0 {
		return nil
	}
	medianH := median(heights, 1)
	gapMax := float64(medianH) * ocrClusterGapFactor

	var blocks []Block
	var (
		cx0, cy0, cx1, cy1 int
		ctext              strings.Builder
		cheights           []int
		open               bool
	)
	flush := func() {
		if !open {
			return
		}
		if txt := strings.TrimSpace(ctext.String()); isTranslatable(txt) {
			blocks = append(blocks, Block{
				Text: txt, X0: cx0, Y0: cy0, X1: cx1, Y1: cy1,
				LineH: median(cheights, cy1-cy0),
			})
		}
		ctext.Reset()
		cheights = cheights[:0]
		open = false
	}
	for _, l := range kept {
		if open {
			gap := float64(l.y0 - cy1)
			overlap := min(l.x1, cx1) - max(l.x0, cx0)
			narrower := min(l.x1-l.x0, cx1-cx0)
			// same column (share x-extent), and vertically adjacent (small forward gap; a
			// small negative gap tolerates overlapping boxes, a big one means a new column).
			if gap <= gapMax && gap >= -float64(medianH) && overlap*10 >= narrower {
				cx0, cy0 = min(cx0, l.x0), min(cy0, l.y0)
				cx1, cy1 = max(cx1, l.x1), max(cy1, l.y1)
				ctext.WriteByte(' ')
				ctext.WriteString(strings.TrimSpace(l.text.String()))
				cheights = append(cheights, l.y1-l.y0)
				continue
			}
			flush()
		}
		cx0, cy0, cx1, cy1 = l.x0, l.y0, l.x1, l.y1
		ctext.WriteString(strings.TrimSpace(l.text.String()))
		cheights = append(cheights, l.y1-l.y0)
		open = true
	}
	flush()
	return blocks
}

func median(vals []int, fallback int) int {
	if len(vals) == 0 {
		return fallback
	}
	s := append([]int(nil), vals...)
	sort.Ints(s)
	return s[len(s)/2]
}
