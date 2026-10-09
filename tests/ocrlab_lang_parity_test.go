package tests

import (
	"reflect"
	"regexp"
	"strings"
	"testing"

	"doc-html-translate/internal/ocr"
	"doc-html-translate/tools/ocrlab/evidence"
)

// jsTable reads `export const <name> = { key: "pack", "key-2": "pack" };` out of the extension lab's
// language module, as a map.
func jsTable(t *testing.T, src, name string) map[string]string {
	t.Helper()
	body := between(src, "export const "+name+" = {", "};")
	if body == "" {
		t.Fatalf("%s not found in _ocrlab-lang.mjs", name)
	}
	out := map[string]string{}
	for _, m := range regexp.MustCompile(`"?([a-z][a-z-]*)"?:\s*"([a-z_]+)"`).FindAllStringSubmatch(body, -1) {
		out[m[1]] = m[2]
	}
	if len(out) == 0 {
		t.Fatalf("%s holds no entries", name)
	}
	return out
}

// TestParityOCRLabLangTable: the lab reads each scene with its declared language, and both editions
// must derive the same Tesseract pack from the same declaration or the two runs grade different
// measurements. The Go lab's table, the extension lab's table and the app's own mapping
// (internal/ocr TessLang) are compared entry for entry, the constants that name a run's language are
// compared by value, and the scene fields that record the choice are compared by name. See
// docs/PARITY.md "OCR lab evidence schema".
func TestParityOCRLabLangTable(t *testing.T) {
	jsSrc := readRepoFile(t, "extension", "scripts", "_ocrlab-lang.mjs")
	goISO, goRegion := evidence.LangTables()

	if jsISO := jsTable(t, jsSrc, "ISO_TO_TESS"); !reflect.DeepEqual(goISO, jsISO) {
		t.Errorf("language table drift:\n  evidence/lang.go isoToTess : %v\n  _ocrlab-lang.mjs          : %v", goISO, jsISO)
	}
	if jsRegion := jsTable(t, jsSrc, "REGION_TO_TESS"); !reflect.DeepEqual(goRegion, jsRegion) {
		t.Errorf("region table drift:\n  evidence/lang.go regionToTess : %v\n  _ocrlab-lang.mjs             : %v", goRegion, jsRegion)
	}

	// Neither lab table may become a rival of the app's mapping, and every pack must be one the app
	// can download.
	catalogue := map[string]bool{}
	for _, l := range ocr.Available {
		catalogue[l.Code] = true
	}
	for _, table := range []map[string]string{goISO, goRegion} {
		for declared, pack := range table {
			if !catalogue[pack] {
				t.Errorf("%s -> %s: not an OCR catalogue language", declared, pack)
			}
			if got := ocr.TessLang(declared); got != pack {
				t.Errorf("%s: the lab maps to %s, internal/ocr TessLang to %s", declared, pack, got)
			}
		}
	}

	// The names that label a run and say where a scene's language came from.
	goSrc := readRepoFile(t, "tools", "ocrlab", "evidence", "lang.go")
	for _, c := range []struct{ goName, jsName string }{
		{"DefaultLang", "DEFAULT_LANG"},
		{"LangPerScene", "LANG_PER_SCENE"},
		{"LangSourceFlag", "LANG_SOURCE_FLAG"},
		{"LangSourceTruth", "LANG_SOURCE_TRUTH"},
		{"LangSourceCorpus", "LANG_SOURCE_CORPUS"},
		{"LangSourceDefault", "LANG_SOURCE_DEFAULT"},
	} {
		goVal := regexp.MustCompile(`\b` + c.goName + `\s*=\s*"([^"]+)"`).FindStringSubmatch(goSrc)
		jsVal := regexp.MustCompile(`export const ` + c.jsName + ` = "([^"]+)";`).FindStringSubmatch(jsSrc)
		if goVal == nil || jsVal == nil {
			t.Errorf("%s / %s not found (Go=%v JS=%v)", c.goName, c.jsName, goVal != nil, jsVal != nil)
			continue
		}
		if goVal[1] != jsVal[1] {
			t.Errorf("%s drift: Go %q, JS %q", c.goName, goVal[1], jsVal[1])
		}
	}

	// The scene fields that record the choice exist on both producers.
	goScene := jsonTags(between(readRepoFile(t, "tools", "ocrlab", "evidence", "evidence.go"), "type Scene struct {", "\n}"))
	jsScene := readRepoFile(t, "extension", "scripts", "_ocrlab-evidence.mjs")
	for _, field := range []string{"lang", "langSource", "unmeasured"} {
		found := false
		for _, tag := range goScene {
			found = found || tag == field
		}
		if !found {
			t.Errorf("evidence.Scene has no %q field", field)
		}
		if !strings.Contains(jsScene, "out."+field+" = str(s."+field+")") {
			t.Errorf("makeScene does not emit %q", field)
		}
	}
}
