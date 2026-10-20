package ocr

// THROWAWAY (ticket 121 phase A). Not committed: measures the word-gap distribution at PSM 3 and
// PSM 11 with the shipped staging. Gated by DOCHT_PSM11_PROBE=<out dir>; deleted after the run.

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"

	"doc-html-translate/internal/procrun"

	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

type probeGap struct {
	Ratio  float64 `json:"r"`
	Stroke bool    `json:"s"`
	A      string  `json:"a"`
	B      string  `json:"b"`
}

type probeLine struct {
	Scene   string     `json:"scene"`
	Variant string     `json:"variant"`
	Lang    string     `json:"lang"`
	Idx     int        `json:"idx"`
	Conf    float64    `json:"conf"`
	N       int        `json:"n"`
	Text    string     `json:"text"`
	Med     int        `json:"med"`
	Box     [4]int     `json:"box"`
	Gaps    []probeGap `json:"gaps"`
	MaxR    float64    `json:"maxR"`
	TallN   int        `json:"tallN"`
	MaxTall float64    `json:"maxTall"`
	AnyWide bool       `json:"wide"`
	AnyStr  bool       `json:"stroke"`
	Crop    string     `json:"crop,omitempty"`
}

func probeParse(data []byte) (lines []*ocrLine) {
	sc := bufio.NewScanner(strings.NewReader(string(data)))
	sc.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	var cur *ocrLine
	first := true
	for sc.Scan() {
		row := sc.Text()
		if first {
			first = false
			if isTSVHeader(row) {
				continue
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
		text := cols[11]
		switch level {
		case 4:
			cur = &ocrLine{x0: left, y0: top, x1: left + w, y1: top + h}
			lines = append(lines, cur)
		case 5:
			if cur == nil || strings.TrimSpace(text) == "" {
				continue
			}
			if h > 0 {
				cur.wordH = append(cur.wordH, h)
			}
			word := ocrWord{x0: left, y0: top, x1: left + w, y1: top + h, text: text}
			if conf, err := strconv.ParseFloat(cols[10], 64); err == nil {
				word.conf, word.hasConf = conf, true
				cur.confSum += conf
				cur.confN++
			}
			cur.words = append(cur.words, word)
		}
	}
	for _, l := range lines {
		l.text.WriteString(joinLineWords(l.words))
	}
	return lines
}

func probeMeasure(scene, variant, lang string, lines []*ocrLine, ink *image.Gray) []probeLine {
	var out []probeLine
	for i, l := range lines {
		if len(l.words) < 2 {
			continue
		}
		med := median(l.wordH, 0)
		if med <= 0 {
			continue
		}
		reach := int(float64(med) * ocrBoundaryReach)
		pl := probeLine{Scene: scene, Variant: variant, Lang: lang, Idx: i, Conf: l.meanConf(), N: len(l.words),
			Text: strings.TrimSpace(l.text.String()), Med: med, Box: [4]int{l.x0, l.y0, l.x1, l.y1}}
		for j := 1; j < len(l.words); j++ {
			prev, w := l.words[j-1], l.words[j]
			gap := float64(max(w.x0-prev.x1, prev.x0-w.x1)) / float64(med)
			st := strokeBetween(ink, prev, w, reach)
			pl.Gaps = append(pl.Gaps, probeGap{Ratio: gap, Stroke: st, A: prev.text, B: w.text})
			if gap > pl.MaxR {
				pl.MaxR = gap
			}
			if gap > ocrMaxWordGapRatio {
				pl.AnyWide = true
			}
			if st {
				pl.AnyStr = true
			}
			if t := float64(w.y1-w.y0) / float64(med); t > ocrTypeSizeRatio {
				pl.TallN++
				if t > pl.MaxTall {
					pl.MaxTall = t
				}
			}
		}
		out = append(out, pl)
	}
	return out
}

func probeCrop(ink *image.Gray, pl *probeLine, path string) bool {
	pad := pl.Med
	r := image.Rect(pl.Box[0]-pad, pl.Box[1]-pad, pl.Box[2]+pad, pl.Box[3]+pad).Intersect(ink.Bounds())
	if r.Empty() {
		return false
	}
	sub := ink.SubImage(r).(*image.Gray)
	f, err := os.Create(path)
	if err != nil {
		return false
	}
	defer f.Close()
	return png.Encode(f, sub) == nil
}

func TestPSM11Probe(t *testing.T) {
	outDir := os.Getenv("DOCHT_PSM11_PROBE")
	if outDir == "" {
		t.Skip("probe off")
	}
	bin, err := Locate()
	if err != nil {
		t.Fatal(err)
	}
	root := `p:\WINDOWS\EPUB_2_HTML`
	var manifest struct {
		Scenes []struct {
			ID   string `json:"id"`
			File string `json:"file"`
		} `json:"scenes"`
	}
	raw, err := os.ReadFile(filepath.Join(root, "DEV", "ocrlab", "corpus.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatal(err)
	}
	type job struct{ id, path string }
	var jobs []job
	for _, s := range manifest.Scenes {
		jobs = append(jobs, job{s.ID, filepath.Join(root, "test_doc", "ocrlab", filepath.FromSlash(s.File))})
	}
	jobs = append(jobs, job{"test_doc-1.png", filepath.Join(root, "test_doc", "1.png")})
	if only := os.Getenv("DOCHT_PSM11_ONLY"); only != "" {
		var keep []job
		for _, j := range jobs {
			if strings.Contains(j.id, only) {
				keep = append(keep, j)
			}
		}
		jobs = keep
	}
	lang := os.Getenv("DOCHT_PSM11_LANG")
	if lang == "" {
		lang = "eng"
	}
	_ = os.MkdirAll(filepath.Join(outDir, "crops"), 0o755)

	var mu sync.Mutex
	var all []probeLine
	var wg sync.WaitGroup
	sem := make(chan struct{}, 5)
	for _, j := range jobs {
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			if _, err := os.Stat(j.path); err != nil {
				t.Logf("missing %s", j.path)
				return
			}
			frame, _, dpi, cleanup := prepareForOCR(j.path)
			defer cleanup()
			ctx := context.Background()
			run := func(path string, psm int) []*ocrLine {
				res, err := runTesseract(ctx, procrun.Tesseract, bin, path, tesseractArgs(path, lang, "", dpi, thresholdEngineDefault, psm))
				if err != nil {
					t.Logf("%s psm %d: %v", j.id, psm, err)
					return nil
				}
				return probeParse(res.Stdout)
			}
			p3 := run(frame.path, 3)
			p11c := run(frame.path, 11)
			grey := frame.grey()
			greyPath, gcl, ok := writeTempPNG(grey)
			var p11g []*ocrLine
			if ok {
				defer gcl()
				p11g = run(greyPath, 11)
			}
			var rows []probeLine
			for _, v := range []struct {
				name  string
				lines []*ocrLine
			}{{"p3", p3}, {"p11c", p11c}, {"p11g", p11g}} {
				ms := probeMeasure(j.id, v.name, lang, v.lines, grey)
				for k := range ms {
					pl := &ms[k]
					// A crop for every line a verdict could depend on.
					if pl.Conf >= ocrMinLineConf && (pl.AnyWide || pl.AnyStr || pl.TallN > 0 || pl.MaxR >= 1.4) {
						name := fmt.Sprintf("%s__%s__%d.png", pl.Variant, pl.Scene, pl.Idx)
						if probeCrop(grey, pl, filepath.Join(outDir, "crops", name)) {
							pl.Crop = name
						}
					}
				}
				rows = append(rows, ms...)
			}
			mu.Lock()
			all = append(all, rows...)
			mu.Unlock()
		}()
	}
	wg.Wait()
	sort.SliceStable(all, func(a, b int) bool {
		if all[a].Variant != all[b].Variant {
			return all[a].Variant < all[b].Variant
		}
		if all[a].Scene != all[b].Scene {
			return all[a].Scene < all[b].Scene
		}
		return all[a].Idx < all[b].Idx
	})
	f, err := os.Create(filepath.Join(outDir, "desktop-"+lang+".jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	for _, pl := range all {
		_ = enc.Encode(pl)
	}
	t.Logf("wrote %d multi-word lines", len(all))
}

// probeSheet is used by the labelling step: stacks crops with a caption into one PNG.
func TestPSM11Sheets(t *testing.T) {
	spec := os.Getenv("DOCHT_PSM11_SHEET") // file: out.png \t caption \t crop path ... one block per sheet, blank line between
	if spec == "" {
		t.Skip("sheets off")
	}
	data, err := os.ReadFile(spec)
	if err != nil {
		t.Fatal(err)
	}
	for _, block := range strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n\n") {
		rowsTxt := strings.Split(strings.TrimSpace(block), "\n")
		if len(rowsTxt) < 2 {
			continue
		}
		outPath := rowsTxt[0]
		const sheetW = 1100
		type item struct {
			cap string
			img image.Image
		}
		var items []item
		for _, r := range rowsTxt[1:] {
			parts := strings.SplitN(r, "\t", 2)
			if len(parts) < 2 {
				continue
			}
			fh, err := os.Open(parts[1])
			if err != nil {
				continue
			}
			im, err := png.Decode(fh)
			fh.Close()
			if err != nil {
				continue
			}
			items = append(items, item{parts[0], im})
		}
		total := 0
		scaled := make([]image.Image, len(items))
		for i, it := range items {
			b := it.img.Bounds()
			w, h := b.Dx(), b.Dy()
			sc := 1.0
			if w < 500 {
				sc = float64(sheetW-20) / float64(w)
				if sc > 3 {
					sc = 3
				}
			} else if w > sheetW-20 {
				sc = float64(sheetW-20) / float64(w)
			}
			nw, nh := max(1, int(float64(w)*sc)), max(1, int(float64(h)*sc))
			dst := image.NewGray(image.Rect(0, 0, nw, nh))
			xdraw.CatmullRom.Scale(dst, dst.Bounds(), it.img, b, xdraw.Over, nil)
			scaled[i] = dst
			total += nh + 22
		}
		sheet := image.NewRGBA(image.Rect(0, 0, sheetW, total+4))
		xdraw.Draw(sheet, sheet.Bounds(), image.NewUniform(color.RGBA{255, 255, 255, 255}), image.Point{}, xdraw.Src)
		y := 2
		d := &font.Drawer{Dst: sheet, Src: image.NewUniform(color.RGBA{200, 0, 0, 255}), Face: basicfont.Face7x13}
		for i, it := range items {
			d.Dot = fixed.P(4, y+13)
			d.DrawString(it.cap)
			y += 18
			b := scaled[i].Bounds()
			xdraw.Draw(sheet, image.Rect(10, y, 10+b.Dx(), y+b.Dy()), scaled[i], image.Point{}, xdraw.Src)
			y += b.Dy() + 4
		}
		fo, err := os.Create(outPath)
		if err != nil {
			t.Fatal(err)
		}
		_ = png.Encode(fo, sheet)
		fo.Close()
	}
}
