package ocr

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// stageLangs points the two tessdata folders at temp folders holding the given packs.
func stageLangs(t *testing.T, user, bundled []string) (userDir, bundledDir string) {
	t.Helper()
	userDir, bundledDir = t.TempDir(), t.TempDir()
	for dir, codes := range map[string][]string{userDir: user, bundledDir: bundled} {
		for _, c := range codes {
			if err := os.WriteFile(filepath.Join(dir, c+".traineddata"), []byte("x"), 0o600); err != nil {
				t.Fatal(err)
			}
		}
	}
	savedUser, savedBundled := userDataDir, bundledDataDir
	t.Cleanup(func() { userDataDir, bundledDataDir = savedUser, savedBundled })
	userDataDir = func() string { return userDir }
	bundledDataDir = func() string { return bundledDir }
	return userDir, bundledDir
}

// `-ocr-langs` is a frozen spelling (OCR-INVOCATION section F): its listing is the one it printed before the verb
// `ocr langs` existed. The expected text is written out here from the catalogue, row by row.
func TestLangsTextIsTheLegacyListing(t *testing.T) {
	userDir, bundledDir := stageLangs(t, []string{"rus"}, []string{"eng"})

	var want strings.Builder
	want.WriteString("OCR languages (tessdata: " + userDir + "; " + bundledDir + ")\n")
	for _, l := range Available {
		pad := func(s string, width int) string {
			if n := width - len([]rune(s)); n > 0 {
				return s + strings.Repeat(" ", n)
			}
			return s
		}
		mark := "  available - download with: -ocr-download " + l.Code
		if l.Code == "eng" || l.Code == "rus" {
			mark = "  installed"
		}
		want.WriteString("  " + pad(l.Code, 9) + " " + pad(l.Name, 22) + mark + "\n")
	}
	if got := LangsText(); got != want.String() {
		t.Errorf("LangsText differs from the legacy listing\n got: %q\nwant: %q", got, want.String())
	}

	// Three rows, literally.
	got := LangsText()
	for _, row := range []string{
		"  eng       English" + strings.Repeat(" ", 15) + "  installed\n",
		"  rus       Русский" + strings.Repeat(" ", 15) + "  installed\n",
	} {
		if !strings.Contains(got, row) {
			t.Errorf("the listing has no row %q:\n%s", row, got)
		}
	}
	if !strings.Contains(got, "  available - download with: -ocr-download ") {
		t.Errorf("a language that is not installed says how to download it:\n%s", got)
	}
}

func TestLangRowsMarkTheInstalledAndNameNoFolder(t *testing.T) {
	userDir, _ := stageLangs(t, []string{"rus"}, []string{"eng"})
	rows := LangRows()
	if len(rows) != len(Available) {
		t.Fatalf("%d rows for a catalogue of %d", len(rows), len(Available))
	}
	installed := 0
	for i, r := range rows {
		if r.Code != Available[i].Code || r.Name != Available[i].Name {
			t.Errorf("row %d = %+v, catalogue has %+v", i, r, Available[i])
		}
		if r.Installed != (r.Code == "eng" || r.Code == "rus") {
			t.Errorf("%s installed = %v", r.Code, r.Installed)
		}
		if r.Installed {
			installed++
		}
		if strings.Contains(r.Name, userDir) {
			t.Errorf("a row names a folder: %+v", r)
		}
	}
	if installed != 2 {
		t.Errorf("%d installed rows, want 2", installed)
	}
}
