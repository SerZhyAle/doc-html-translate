// Package exchange writes the OCR-OVERLAY section 7 comparison record - the one format two
// implementations exchange when they compare results on the same scene - from a lab run's
// diagnostics sidecar (runner.DiagFile). OCR-OVERLAY 1.2 item C lets an implementation emit it from
// any instrument it owns and never from its user output; this is that instrument for this product,
// and OCR-PIPELINE amendment 1.4 D records it.
//
// It reads what the app wrote and nothing else: the display-space size and block boxes of the
// diagnostic line, the block's mean line confidence, and the source file's EXIF orientation. Both
// editions write the same diagnostic line, so one reader serves both.
package exchange

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Record is one OCR-OVERLAY section 7 document: one image and its blocks.
type Record struct {
	Image  Image   `json:"image"`
	Blocks []Block `json:"blocks"`
}

// Image is the picture the boxes are measured on. Width and Height are display-space pixels
// (OCR-OVERLAY rule 1); Orientation is the source file's EXIF tag, 1 when it carries none.
type Image struct {
	Width       int `json:"width"`
	Height      int `json:"height"`
	Orientation int `json:"orientation"`
}

// Block is one recognized block. Translation is optional (OCR-OVERLAY 1.2 item B) and this product
// never has it at recognition time, so it is absent - "not translated in this record" - rather than
// an empty string, which a reader would take as translated to nothing.
type Block struct {
	ID           string  `json:"id"`
	Text         string  `json:"text"`
	Translation  *string `json:"translation,omitempty"`
	Confidence   float64 `json:"confidence"`
	Box          Box     `json:"box"`
	ReadingOrder int     `json:"readingOrder"`
}

// Box is top-left anchored, in display-space pixels.
type Box struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

// diagLine is the part of the diagnostic line (internal/ocr diagImage, extension makeDiagRecord)
// this package reads. Fields are matched by name; anything else on the line is ignored.
type diagLine struct {
	File   string `json:"file"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Blocks []struct {
		Text string  `json:"text"`
		X0   int     `json:"x0"`
		Y0   int     `json:"y0"`
		X1   int     `json:"x1"`
		Y1   int     `json:"y1"`
		Conf float64 `json:"conf"`
	} `json:"blocks"`
}

// SceneID is the scene a diagnostic line belongs to: the image file's stem. Both editions name the
// image after its scene - the desktop run under pages/<scene>/<scene>.<ext>, the extension run at the
// corpus media path - so the stem is the one key they share.
func SceneID(file string) string {
	base := filepath.Base(strings.ReplaceAll(file, `\`, "/"))
	return strings.TrimSuffix(base, filepath.Ext(base))
}

// FromLine turns one diagnostic line into a scene id and its record. orientation reads the source
// file's EXIF tag; it is a parameter so a test needs no image on disk.
func FromLine(line []byte, orientation func(file string) int) (string, Record, error) {
	var d diagLine
	if err := json.Unmarshal(line, &d); err != nil {
		return "", Record{}, fmt.Errorf("diagnostic line: %w", err)
	}
	if d.File == "" {
		return "", Record{}, errors.New("diagnostic line names no file")
	}
	o := 1
	if orientation != nil {
		if v := orientation(d.File); v >= 1 && v <= 8 {
			o = v
		}
	}
	rec := Record{Image: Image{Width: d.Width, Height: d.Height, Orientation: o}, Blocks: []Block{}}
	// The line lists blocks in the order the plates are drawn, which is reading order.
	for i, b := range d.Blocks {
		rec.Blocks = append(rec.Blocks, Block{
			ID:           fmt.Sprintf("b-%03d", i+1),
			Text:         b.Text,
			Confidence:   b.Conf,
			Box:          Box{X: b.X0, Y: b.Y0, Width: b.X1 - b.X0, Height: b.Y1 - b.Y0},
			ReadingOrder: i + 1,
		})
	}
	return SceneID(d.File), rec, nil
}

// WriteDir reads runDir/<diagFile> and writes runDir/exchange/<scene>.json, one per scene. When a
// scene has more than one line - a re-run appended to the same sidecar - the last one wins, because
// it describes the plates the run's evidence was recorded from. It returns the scenes written.
func WriteDir(runDir, diagFile string, orientation func(file string) int) ([]string, error) {
	raw, err := os.ReadFile(filepath.Join(runDir, diagFile))
	if err != nil {
		return nil, err
	}
	recs := map[string]Record{}
	sc := bufio.NewScanner(bytes.NewReader(raw))
	sc.Buffer(make([]byte, 0, 1<<20), 64<<20)
	for n := 1; sc.Scan(); n++ {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 {
			continue
		}
		id, rec, err := FromLine(line, orientation)
		if err != nil {
			return nil, fmt.Errorf("%s line %d: %w", diagFile, n, err)
		}
		recs[id] = rec
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if len(recs) == 0 {
		return nil, fmt.Errorf("%s holds no diagnostic line", diagFile)
	}
	out := filepath.Join(runDir, "exchange")
	if err := os.MkdirAll(out, 0o755); err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(recs))
	for id := range recs {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		b, err := json.MarshalIndent(recs[id], "", "  ")
		if err != nil {
			return nil, err
		}
		if err := os.WriteFile(filepath.Join(out, id+".json"), append(b, '\n'), 0o644); err != nil {
			return nil, err
		}
	}
	return ids, nil
}
