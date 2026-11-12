package tests

// WINDOWS-STORE rules 7 to 9, 12 and 13 and conformance rungs 1 and 2 (ticket E0135). The cases of the
// Partner Center listing builder - every refusal and every kept dialect - run in one pwsh process,
// tools/store/test-store-listing-csv.ps1 (3 s on 2026-11-12, so the fast tier). What a script cannot say
// about its own files is held here, in Go: the -text pin, the measured dialect of both committed CSVs, the
// emptied copy cells of the fixture, and the name of the authoritative listing file.

import (
	"bytes"
	"encoding/csv"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var storeLocales = []string{"en-us", "ru", "uk", "de", "it", "es", "fr", "pt-br", "ar", "hi", "bn", "ur", "zh-hans"}

// storeCopyField is a row the builder fills from the decks, or the contract says is copy.
var storeCopyField = regexp.MustCompile(`^(Title|ShortDescription|Description|ReleaseNotes|CopyrightTrademarkInformation|Feature\d+|SearchTerm\d+)$`)

func TestStoreListingBuilderCases(t *testing.T) {
	pwsh := findPwsh(t)
	script := repoPath(t, "tools", "store", "test-store-listing-csv.ps1")
	code, last, out := runScript(t, pwsh, repoPath(t), script, "-Work", t.TempDir())
	if code != 0 {
		t.Fatalf("test-store-listing-csv.ps1 exit %d:\n%s", code, out)
	}
	if !regexp.MustCompile(`^cases=\d+ failed=0$`).MatchString(last) {
		t.Fatalf("last line %q, want cases=<N> failed=0:\n%s", last, out)
	}
	if strings.Contains(out, "FAIL ") {
		t.Fatalf("a case failed:\n%s", out)
	}
}

func TestStoreListingExportsArePinned(t *testing.T) {
	attrs := readRepoFile(t, ".gitattributes")
	for _, f := range []string{"tools/store/listingData.csv", "tools/store/listingData.sample.csv"} {
		if !regexp.MustCompile(`(?m)^` + regexp.QuoteMeta(f) + `\s+-text\s*$`).MatchString(attrs) {
			t.Errorf(".gitattributes has no \"%s -text\" line: a checkout could rewrite the export's line endings (rule 13)", f)
		}
	}
}

// csvDialect measures a Partner Center export the way the builder does: quotes open and close a cell, a break
// outside quotes ends a record, a break inside is part of the cell.
type csvDialectFacts struct {
	bom                  bool
	recordCRLF, recordLF int
	recordCR, cellCR     int
	cellCRLF, cellLF     int
	open, endsWithBreak  bool
}

func measureStoreCSV(b []byte) csvDialectFacts {
	var f csvDialectFacts
	if bytes.HasPrefix(b, []byte{0xEF, 0xBB, 0xBF}) {
		f.bom = true
		b = b[3:]
	}
	in := false
	for i := 0; i < len(b); i++ {
		c := b[i]
		if in {
			switch {
			case c == '"' && i+1 < len(b) && b[i+1] == '"':
				i++
			case c == '"':
				in = false
			case c == '\r' && i+1 < len(b) && b[i+1] == '\n':
				f.cellCRLF++
				i++
			case c == '\r':
				f.cellCR++
			case c == '\n':
				f.cellLF++
			}
			continue
		}
		switch {
		case c == '"':
			in = true
		case c == '\r' && i+1 < len(b) && b[i+1] == '\n':
			f.recordCRLF++
			i++
		case c == '\r':
			f.recordCR++
		case c == '\n':
			f.recordLF++
		}
	}
	f.open = in
	f.endsWithBreak = len(b) > 0 && (b[len(b)-1] == '\n' || b[len(b)-1] == '\r')
	return f
}

func readStoreCSV(t *testing.T, name string) ([][]string, csvDialectFacts) {
	t.Helper()
	p := filepath.Join("..", "tools", "store", name)
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read %s: %v", p, err)
	}
	facts := measureStoreCSV(b)
	r := csv.NewReader(bytes.NewReader(bytes.TrimPrefix(b, []byte{0xEF, 0xBB, 0xBF})))
	r.FieldsPerRecord = -1
	rows, err := r.ReadAll()
	if err != nil {
		t.Fatalf("parse %s: %v", p, err)
	}
	return rows, facts
}

func TestStoreListingExportDialect(t *testing.T) {
	for _, name := range []string{"listingData.csv", "listingData.sample.csv"} {
		rows, f := readStoreCSV(t, name)
		if !f.bom || f.recordLF != 0 || f.recordCR != 0 || f.cellCR != 0 || f.cellCRLF != 0 || f.open || f.endsWithBreak || f.recordCRLF != len(rows)-1 {
			t.Errorf("%s is not in the export's dialect (BOM, CRLF between records, LF inside cells, no trailing newline): %+v over %d records", name, f, len(rows))
		}
		header := strings.Join(rows[0], "|")
		for _, l := range storeLocales {
			if !strings.Contains("|"+header+"|", "|"+l+"|") {
				t.Errorf("%s has no %s column (rule 7)", name, l)
			}
		}
	}
}

// Rule 13: the fixture is an export - header, all 13 language columns, asset rows with their cells - whose copy
// cells are empty in every language. It is derived from a fresh export by emptying those cells, so it is held
// on its own shape here and not against the working export, which a fresh export replaces before each import.
func TestStoreListingFixtureHasEmptyCopyCells(t *testing.T) {
	sample, _ := readStoreCSV(t, "listingData.sample.csv")
	lang := map[int]bool{}
	for i, h := range sample[0] {
		for _, l := range storeLocales {
			if h == l {
				lang[i] = true
			}
		}
	}
	copyRows, assetCells := 0, 0
	for r := 1; r < len(sample); r++ {
		field := sample[r][0]
		switch {
		case storeCopyField.MatchString(field):
			copyRows++
			for c := range sample[r] {
				if lang[c] && sample[r][c] != "" {
					t.Errorf("fixture %s %s is not empty", field, sample[0][c])
				}
			}
		case strings.HasPrefix(field, "DesktopScreenshot"):
			for c := range sample[r] {
				if lang[c] && sample[r][c] != "" {
					assetCells++
				}
			}
		}
	}
	if copyRows < 20 {
		t.Errorf("the fixture has %d copy rows, an export has Title, ShortDescription, Description, ReleaseNotes, Feature* and SearchTerm*", copyRows)
	}
	if assetCells == 0 {
		t.Error("the fixture has no asset cell, so it is not an export patched by -FillNothing's rules (rule 13)")
	}
}

// Rule 12: a product with more than one listing file names the authoritative one in its agent-rules file.
func TestAgentsNamesTheAuthoritativeListingFile(t *testing.T) {
	agents := readRepoFile(t, "AGENTS.md")
	ok := false
	for _, line := range strings.Split(agents, "\n") {
		if strings.Contains(line, "tools/store/listingData.csv") && strings.Contains(line, "authoritative") &&
			strings.Contains(line, "msix/out/store-import/") && strings.Contains(line, "gitignored") {
			ok = true
		}
	}
	if !ok {
		t.Error("AGENTS.md has no line naming tools/store/listingData.csv as the authoritative listing file and msix/out/store-import/ as its gitignored copy (WINDOWS-STORE rule 12)")
	}
}
