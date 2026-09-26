package pdf

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	pdflib "github.com/ledongthuc/pdf"
)

// The same fixture drives extension/test/reflow.test.mjs, so the two editions drop exactly the
// same rows (docs/PARITY.md, "PDF reflow heuristics").
func TestIsLigaturesArtifactSharedCases(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "tests", "testdata", "ligature_artifact_cases.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Cases []struct {
			Text     string `json:"text"`
			Artifact bool   `json:"artifact"`
			Why      string `json:"why"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	if len(fixture.Cases) == 0 {
		t.Fatal("no cases in the shared fixture")
	}
	for _, c := range fixture.Cases {
		if got := isLigaturesArtifact(c.Text); got != c.Artifact {
			t.Errorf("isLigaturesArtifact(%q) = %v, want %v (%s)", c.Text, got, c.Artifact, c.Why)
		}
	}
}

// shortLinesPage is one page of pdftotext -layout output: real short lines, each its own block,
// and one block of split-ligature garbage.
const shortLinesPage = "   Text of page 2\n\n" +
	"   Is it so? I do.\n\n" +
	"   is it so I do\n\n" +
	"   fi fl fi fi fl fi\n\n" +
	"   I am as I am\n"

// The pdftotext path used to drop every block of four or more words averaging under three
// letters, which is most short dialogue. Only the garbage block may go.
func TestParsePDFLayoutPage_KeepsShortRealLines(t *testing.T) {
	var got []string
	for _, it := range parsePDFLayoutPage(shortLinesPage) {
		got = append(got, it.text)
	}
	want := []string{"Text of page 2", "Is it so? I do.", "is it so I do", "I am as I am"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("kept blocks = %q, want %q", got, want)
	}
}

// The pure-Go reader's rows go through the same filter, per row as in the extension.
func TestRowsToText_DropsOnlyLigatureRows(t *testing.T) {
	rows := pdflib.Rows{
		{Content: pdflib.TextHorizontal{{S: "Is it so? I do.", X: 10, Y: 100}}},
		{Content: pdflib.TextHorizontal{{S: "if lf if if if if", X: 10, Y: 88}}},
		{Content: pdflib.TextHorizontal{{S: "Text of page 2", X: 10, Y: 76}}},
	}
	got := strings.TrimSpace(rowsToText(rows))
	if got != "Is it so? I do. Text of page 2" {
		t.Errorf("rowsToText = %q, want the two real rows and not the fragments", got)
	}
}

// End to end through extractWithPDFToText with a stub standing in for pdftotext, so the path
// that failed on the owner's machine (TestExtract_Volume3ImagesOnTheirPages with the vendored
// pdftotext) is covered where Poppler is not installed: every short text page is emitted.
func TestExtractWithPDFToText_KeepsShortTextPages(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the stub pdftotext is a shell script")
	}
	tmp := t.TempDir()
	pdfPath := filepath.Join(tmp, "Volume_3.pdf")
	buildFixturePDF(t, pdfPath, 3, allPages(3), nil)

	stub := filepath.Join(tmp, "pdftotext")
	script := "#!/bin/sh\nprintf 'Text of page 1\\r\\n\\fText of page 2\\r\\n\\fText of page 3\\r\\n\\f'\n"
	if err := os.WriteFile(stub, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(tmp, "out")
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatal(err)
	}
	book, err := extractWithPDFToText(context.Background(), stub, pdfPath, out)
	if err != nil {
		t.Fatalf("extractWithPDFToText: %v", err)
	}
	hrefs := book.SpineHrefs()
	if len(hrefs) != 3 {
		t.Fatalf("got %d pages, want 3", len(hrefs))
	}
	for i, href := range hrefs {
		data, err := os.ReadFile(filepath.Join(out, href))
		if err != nil {
			t.Fatal(err)
		}
		if want := "Text of page " + string(rune('1'+i)); !strings.Contains(string(data), want) {
			t.Errorf("page %d lost its text %q", i+1, want)
		}
	}
}
