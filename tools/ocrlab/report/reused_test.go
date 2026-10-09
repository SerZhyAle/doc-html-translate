package report

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"doc-html-translate/tools/ocrlab/evidence"
	"doc-html-translate/tools/ocrlab/metrics"
)

func reuseData() *Data {
	from := &evidence.ReusedFrom{
		Bundle: "temp/ocrlab/earlier", DeclarationSHA256: "d", ProducerDigest: "p",
		CollectedAt: "2026-10-09T10:00:00Z", ReusedAt: "2026-10-09T11:00:00Z", Integrity: "sizes",
	}
	return &Data{
		Run: &evidence.Run{
			RunID: "later", Edition: evidence.EditionDesktop,
			Scenes: []evidence.Scene{{SceneID: "copied", ReusedFrom: from}, {SceneID: "fresh"}},
		},
		Summary: &metrics.Summary{},
	}
}

func TestReportSaysWhichScenesWereReusedAndFromWhere(t *testing.T) {
	dir := t.TempDir()
	d := reuseData()
	if err := WriteMarkdown(dir, d); err != nil {
		t.Fatal(err)
	}
	if err := WriteHTML(dir, d); err != nil {
		t.Fatal(err)
	}
	md, _ := os.ReadFile(filepath.Join(dir, "report.md"))
	page, _ := os.ReadFile(filepath.Join(dir, "report.html"))

	for name, got := range map[string]string{"report.md": string(md), "report.html": string(page)} {
		if !strings.Contains(got, "1 scene(s) collected in this run, 1 reused from earlier runs.") {
			t.Errorf("%s: the collected/reused counts are missing", name)
		}
		if !strings.Contains(got, "temp/ocrlab/earlier") {
			t.Errorf("%s: the origin of a reused scene is missing", name)
		}
	}
	if !strings.Contains(string(md), "| `copied` | `temp/ocrlab/earlier` | 2026-10-09T10:00:00Z | 2026-10-09T11:00:00Z | sizes |") {
		t.Errorf("the reused-scene table is missing or malformed:\n%s", md)
	}
	if strings.Contains(string(md), "`fresh` | `temp") {
		t.Error("a collected scene must not be listed as reused")
	}
	if n := strings.Count(string(page), `class="reused"`); n != 1 || !strings.Contains(string(page), "not a new measurement") {
		t.Errorf("exactly the reused scene's card carries the provenance note, got %d", n)
	}
}

func TestReportOfAFullyCollectedRunHasNoReuseNoise(t *testing.T) {
	dir := t.TempDir()
	d := reuseData()
	d.Run.Scenes[0].ReusedFrom = nil
	if err := WriteMarkdown(dir, d); err != nil {
		t.Fatal(err)
	}
	if err := WriteHTML(dir, d); err != nil {
		t.Fatal(err)
	}
	md, _ := os.ReadFile(filepath.Join(dir, "report.md"))
	page, _ := os.ReadFile(filepath.Join(dir, "report.html"))
	if strings.Contains(string(md), "Reused scene") || strings.Contains(string(page), `class="reused"`) {
		t.Error("nothing was reused, so nothing should be marked")
	}
	if !strings.Contains(string(md), "2 scene(s) collected in this run, 0 reused") {
		t.Error("the counts still state that everything was collected")
	}
}
