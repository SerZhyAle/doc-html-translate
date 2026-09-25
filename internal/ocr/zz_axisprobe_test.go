package ocr

// Throwaway measurement probe for ticket 29 (rescue third axis). Not committed.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode"

	"doc-html-translate/internal/procrun"
)


type probeLine struct {
	Pass    string
	Text    string
	Conf    float64
	InkH    int
	Run     int
	Cleared bool
}

func TestAxisProbe(t *testing.T) {
	out := os.Getenv("DOCHT_AXIS_PROBE")
	if out == "" {
		t.Skip("probe")
	}
	lang := os.Getenv("DOCHT_AXIS_LANG")
	only := os.Getenv("DOCHT_AXIS_SCENES")
	root := `P:\WINDOWS\EPUB_2_HTML`
	raw, err := os.ReadFile(filepath.Join(root, "DEV", "ocrlab", "corpus.json"))
	if err != nil {
		t.Fatal(err)
	}
	var man struct {
		Scenes []struct{ ID, File, Split string }
	}
	if err := json.Unmarshal(raw, &man); err != nil {
		t.Fatal(err)
	}
	bin, err := Locate()
	if err != nil {
		t.Fatal(err)
	}
	dataDir := DataDir()
	f, err := os.Create(out)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	fmt.Fprintln(f, "scene\tsplit\tlang\tosd\tosdConf\tpath\tpass\tconf\tinkH\trun\tcleared\tfitSame\tfitAny\tagree\ttext")
	for _, sc := range man.Scenes {
		if only != "" && !strings.Contains(","+only+",", ","+sc.ID+",") {
			continue
		}
		img := filepath.Join(root, "test_doc", "ocrlab", filepath.FromSlash(sc.File))
		if _, err := os.Stat(img); err != nil {
			t.Logf("skip %s: %v", sc.ID, err)
			continue
		}
		script, sconf, ok := DetectScript(bin, img, dataDir)
		if !ok {
			script = "-"
		}
		frame, _, dpi, cleanup := prepareForOCR(img)
		run := func(path string, thr, psm int, minConf float64) []*ocrLine {
			res, err := runTesseract(procrun.Tesseract, bin, path, tesseractArgs(path, lang, dataDir, dpi, thr, psm))
			if err != nil {
				return nil
			}
			ls, _, _, _ := tsvLines(res.Stdout, minConf, frame.grey())
			return ls
		}
		passes := map[string][]*ocrLine{}
		var order []string
		readOK := false
		if res, err := runTesseract(procrun.Tesseract, bin, frame.path, tesseractArgs(frame.path, lang, dataDir, dpi, thresholdEngineDefault, ocrPageSegMode)); err == nil {
			if pr, err := parseTSV(res.Stdout, ocrMinLineConf, frame.grey()); err == nil && len(pr.Blocks) > 0 {
				readOK = true
			}
		}
		path := "ladder"
		if readOK {
			path = "read"
		}
		grey := frame.grey()
		if grey != nil {
			gp, gcl, ok := writeTempPNG(grey)
			if ok {
				if !readOK {
					for i, rung := range greyRescuePasses {
						name := fmt.Sprintf("rung%d", i+1)
						passes[name] = run(gp, rung.thresholding, rung.psm, ocrRescueLineConf)
						order = append(order, name)
					}
				}
				gcl()
			}
			if pitch := screenPitch(grey); pitch != 0 {
				sp, scl, ok := writeTempPNG(gaussBlurGray(grey, float64(pitch)/ocrScreenSigmaDivisor))
				if ok {
					passes["screen"] = run(sp, thresholdEngineDefault, ocrPageSegMode, ocrRescueLineConf)
					order = append(order, "screen")
					scl()
				}
			}
		}
		cleanup()
		norm := func(s string) string {
			return strings.ToLower(strings.Join(strings.FieldsFunc(s, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) }), " "))
		}
		var anyCleared []int
		for _, name := range order {
			for _, l := range passes[name] {
				if keepLine(l, ocrRescueLineConf) {
					anyCleared = append(anyCleared, l.inkHeight())
				}
			}
		}
		for _, name := range order {
			var same []int
			for _, l := range passes[name] {
				if keepLine(l, ocrRescueLineConf) {
					same = append(same, l.inkHeight())
				}
			}
			fits := func(h int, set []int) bool {
				for _, c := range set {
					if h > 0 && c > 0 && sameTypeSize(h, c) {
						return true
					}
				}
				return false
			}
			for _, l := range passes[name] {
				txt := strings.TrimSpace(l.text.String())
				if txt == "" {
					continue
				}
				agree := 0
				for _, other := range order {
					if other == name {
						continue
					}
					for _, o := range passes[other] {
						if n := norm(txt); n != "" && n == norm(o.text.String()) {
							agree++
							break
						}
					}
				}
				fmt.Fprintf(f, "%s\t%s\t%s\t%s\t%.2f\t%s\t%s\t%.1f\t%d\t%d\t%v\t%v\t%v\t%d\t%s\n",
					sc.ID, sc.Split, lang, script, sconf, path, name, l.meanConf(), l.inkHeight(), longestLetterRun(txt),
					keepLine(l, ocrRescueLineConf), fits(l.inkHeight(), same), fits(l.inkHeight(), anyCleared), agree,
					strings.ReplaceAll(txt, "\t", " "))
			}
		}
		t.Logf("%s %s osd=%s %.2f path=%s", sc.ID, lang, script, sconf, path)
	}
}
