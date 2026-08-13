package ocr

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// The OCR language is chosen from -ocr-lang, or - when that is empty - from -src, a flag about
// translation that most readers never relate to recognition. A reader who converts a Russian
// screenshot with the defaults therefore gets an English recognizer pointed at Cyrillic, and
// Tesseract does not fail on it: it transliterates, and the page comes back covered in plates
// reading "Katanoru-nonyyarenn" over an interface that was perfectly readable before. The output
// is strictly worse than the input, and nothing in the log says so
// (DEV/research/ocr_sweep_2026-08-13.md defect 3).
//
// So when - and only when - the language was not chosen by the reader, the book's first image is
// put through Tesseract's orientation-and-script pass and the answer is allowed to correct the
// default. Two things keep that from becoming a new way to break a book that works today.
//
// The first is what the correction does. Where data for the detected script is installed the
// language becomes `<script language>+<the default>` rather than a replacement, so a wrong verdict
// costs a slower pass and not an unreadable one; Tesseract reads both. Where it is not installed
// there is no way to produce correct plates, so the book gets none and a line naming the script and
// the download - the ticket's own criterion: correct plates, or none, never debris.
//
// The second is the floor, and it exists because the detector is much worse than its name suggests.
// Measured over the 46 lab scenes (DEV/research/ocr_plate_coverage_2026-08-13.md §4): it calls an
// English Archie cover "Cyrillic" at 3.81, an 1887 English cartoon "Arabic" at 5.00, and a genuinely
// Cyrillic poster "Latin" at 3.97. Only two scenes get a non-Latin script right, and both are far
// above that noise - the Russian UI screenshot at 8.24 and a Soviet poster at 8.15. 6.4 is the
// geometric middle of the worst wrong answer (5.00) and the weaker right one (8.15), about 28% of
// margin each way, and no scene in the corpus sits between them.
const ocrScriptConfidenceFloor = 6.4

// osdScriptLine reads the two lines of `--psm 0` output the script rule needs. Everything else
// tesseract prints there (page number, orientation, rotation) belongs to the orientation half,
// which the app does not use - it turns the picture by its EXIF tag instead (see exif.go).
var osdScriptLine = regexp.MustCompile(`(?m)^Script:\s*(\S+)\s*$[\s\S]*?^Script confidence:\s*([\d.]+)\s*$`)

// scriptOfLang says which script each catalog language reads. Only the catalog is listed, because
// a language outside it cannot be installed and so cannot be switched to; anything unknown counts
// as Latin, which is the conservative answer (it makes the rule fire less, never more).
var scriptOfLang = map[string]string{
	"rus": "Cyrillic", "ukr": "Cyrillic",
	"jpn": "Japanese", "jpn_vert": "Japanese",
	"chi_sim": "Han",
	"kor":     "Hangul",
}

// osdScriptAliases folds the several names Tesseract's OSD gives one writing system onto the names
// scriptOfLang uses. Japanese text is reported as "Japanese", but a page of only kana comes back as
// "Katakana" or "Hiragana", and Chinese as "Han" or "HanS"/"HanT" depending on the model.
var osdScriptAliases = map[string]string{
	"Katakana": "Japanese", "Hiragana": "Japanese",
	"HanS": "Han", "HanT": "Han", "HanS_vert": "Han", "HanT_vert": "Han",
	"Korean": "Hangul",
}

// scriptOf returns the script a "+"-joined language string reads: the first code's, since that is
// the one Tesseract leads with.
func scriptOf(lang string) string {
	first, _, _ := strings.Cut(lang, "+")
	if s, ok := scriptOfLang[first]; ok {
		return s
	}
	return "Latin"
}

// installedForScript lists the catalog languages for a script that are actually present in dataDir,
// in catalog order, so the choice is repeatable rather than map-ordered.
func installedForScript(dataDir, script string) []string {
	var out []string
	for _, l := range Available {
		if scriptOfLang[l.Code] == script && hasLangFile(dataDir, l.Code) {
			out = append(out, l.Code)
		}
	}
	return out
}

// DetectScript runs Tesseract's orientation-and-script pass over one image and returns the script
// it named and how sure it was. ok=false for every failure - no osd.traineddata (the app bundles
// only eng, so this is the common case on a fresh install), too few characters on the page, a
// broken engine - because the rule this feeds must never act on an answer that was not given.
func DetectScript(bin, imgPath, dataDir string) (script string, conf float64, ok bool) {
	args := []string{imgPath, "stdout", "--psm", "0"}
	if dataDir != "" && hasLangFile(dataDir, "osd") {
		args = append(args, "--tessdata-dir", dataDir)
	}
	out, err := exec.Command(bin, args...).CombinedOutput()
	if err != nil {
		return "", 0, false
	}
	m := osdScriptLine.FindSubmatch(out)
	if m == nil {
		return "", 0, false
	}
	c, err := strconv.ParseFloat(string(m[2]), 64)
	if err != nil {
		return "", 0, false
	}
	s := string(m[1])
	if alias, ok := osdScriptAliases[s]; ok {
		s = alias
	}
	return s, c, true
}

// resolveScript decides what the book's OCR language should be once the detector has spoken. It is
// pure so the decision is tested without an engine: the caller supplies what was detected and what
// is installed, and gets back the language to use, a line for the log, and whether to give up.
//
// Order matters. A language the reader chose is never second-guessed; a detector that said nothing,
// said Latin, or was not sure enough changes nothing; and a script the default already reads is not
// a mismatch. Only a confident disagreement gets an answer.
func resolveScript(lang string, langFixed bool, script string, conf float64, detected bool, installed []string) (use, note string, stop bool) {
	if langFixed || !detected || conf < ocrScriptConfidenceFloor {
		return lang, "", false
	}
	if script == "" || script == "Latin" || script == scriptOf(lang) {
		return lang, "", false
	}
	if len(installed) > 0 {
		use = installed[0] + "+" + lang
		return use, fmt.Sprintf(
			"the pages are written in the %s script, which the %s data cannot read - recognizing with %s instead "+
				"(pass -ocr-lang to choose for yourself)", script, LangLabel(lang), LangLabel(use)), false
	}
	want := ""
	for _, l := range Available {
		if scriptOfLang[l.Code] == script {
			want = l.Code
			break
		}
	}
	msg := fmt.Sprintf("the pages are written in the %s script, which the %s data cannot read, so no text plates "+
		"were added", script, LangLabel(lang))
	if want != "" {
		msg += fmt.Sprintf(" - install the data with -ocr-download %s, or pass -ocr-lang to choose for yourself", want)
	} else {
		msg += " - this app has no data for that script; pass -ocr-lang to recognize anyway"
	}
	return lang, msg, true
}
