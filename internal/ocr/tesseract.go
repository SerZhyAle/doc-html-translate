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
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
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

// Recognize runs tesseract on imgPath for the given language and returns the recognized
// blocks plus the image's pixel dimensions. dataDir is passed as --tessdata-dir only when
// it actually contains the requested language, so a system tesseract still works when the
// app ships no bundled data.
func Recognize(bin, imgPath, lang, dataDir string) (Result, error) {
	if lang == "" {
		lang = "eng"
	}
	args := []string{imgPath, "stdout"}
	if dataDir != "" && hasLangFile(dataDir, lang) {
		args = append(args, "--tessdata-dir", dataDir)
	}
	args = append(args, "-l", lang, "tsv")

	cmd := exec.Command(bin, args...)
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		return Result{}, fmt.Errorf("tesseract: %w: %s", err, strings.TrimSpace(errb.String()))
	}
	return parseTSV(out.Bytes())
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

// parseTSV turns tesseract's TSV output into a Result. Columns are:
// level page block par line word left top width height conf text
// level 1 = page (its size is the image size), 2 = block, 4 = line, 5 = word.
func parseTSV(data []byte) (Result, error) {
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)

	var res Result
	type box struct{ x0, y0, x1, y1 int }
	blockBox := map[int]box{}
	blockText := map[int]*strings.Builder{}
	blockLineH := map[int][]int{}
	var order []int

	firstLine := true
	for sc.Scan() {
		line := sc.Text()
		if firstLine {
			firstLine = false
			if strings.HasPrefix(line, "level\t") || strings.HasPrefix(line, "level ") {
				continue // header row
			}
		}
		cols := strings.Split(line, "\t")
		if len(cols) < 12 {
			continue
		}
		level, _ := strconv.Atoi(cols[0])
		blockNum, _ := strconv.Atoi(cols[2])
		left, _ := strconv.Atoi(cols[6])
		top, _ := strconv.Atoi(cols[7])
		w, _ := strconv.Atoi(cols[8])
		h, _ := strconv.Atoi(cols[9])
		text := cols[11]

		switch level {
		case 1:
			res.Width, res.Height = w, h
		case 2:
			blockBox[blockNum] = box{left, top, left + w, top + h}
			if _, ok := blockText[blockNum]; !ok {
				blockText[blockNum] = &strings.Builder{}
				order = append(order, blockNum)
			}
		case 4:
			blockLineH[blockNum] = append(blockLineH[blockNum], h)
		case 5:
			if text == "" {
				continue
			}
			b, ok := blockText[blockNum]
			if !ok {
				b = &strings.Builder{}
				blockText[blockNum] = b
				order = append(order, blockNum)
			}
			if b.Len() > 0 {
				b.WriteByte(' ')
			}
			b.WriteString(text)
		}
	}
	if err := sc.Err(); err != nil {
		return Result{}, err
	}

	for _, bn := range order {
		txt := strings.TrimSpace(blockText[bn].String())
		if txt == "" {
			continue
		}
		bx := blockBox[bn]
		res.Blocks = append(res.Blocks, Block{
			Text: txt,
			X0:   bx.x0, Y0: bx.y0, X1: bx.x1, Y1: bx.y1,
			LineH: median(blockLineH[bn], bx.y1-bx.y0),
		})
	}
	return res, nil
}

func median(vals []int, fallback int) int {
	if len(vals) == 0 {
		return fallback
	}
	s := append([]int(nil), vals...)
	sort.Ints(s)
	return s[len(s)/2]
}
