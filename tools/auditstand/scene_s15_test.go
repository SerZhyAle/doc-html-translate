package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPeakVerdictAgainstTicket12sLine(t *testing.T) {
	if v := peakVerdict(0); v.Verdict != "advisory" {
		t.Errorf("an unseen process is no evidence: %+v", v)
	}
	if v := peakVerdict(300 << 20); v.Verdict != "advisory" || !strings.Contains(v.Observed, "300.0 MiB") {
		t.Errorf("a modest peak is recorded, not closed: %+v", v)
	}
	if v := peakVerdict(1 << 30); v.Verdict != "fail" {
		t.Errorf("the line itself is a fail: %+v", v)
	}
	if v := peakVerdict(1.6 * (1 << 30)); v.Verdict != "fail" {
		t.Errorf("above the line is a fail: %+v", v)
	}
}

func TestPeakPrivateBytesReadsTheLargestSample(t *testing.T) {
	samples := []sample{
		{Fields: map[string]interface{}{"private_bytes": 10.0}},
		{Fields: map[string]interface{}{"private_bytes": 700.0}},
		{Fields: map[string]interface{}{"private_bytes": 20.0, "working_set_bytes": 9999.0}},
		{Fields: map[string]interface{}{"handles": int64(3)}},
	}
	if got := peakPrivateBytes(samples); got != 700 {
		t.Errorf("peak = %v, want 700 (private bytes, not working set)", got)
	}
}

func TestS15SmallMembersRunAndBigOnesArePendingNotSkipped(t *testing.T) {
	t.Setenv("AUDITSTAND_STUB_CLI", "1")
	bin, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	fixtures := filepath.Join(root, "fixtures")
	if err := os.MkdirAll(fixtures, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"rtf-nested.rtf", "tiff-multiframe.tif", "tiff-over-cap.tif"} {
		if err := os.WriteFile(filepath.Join(fixtures, f), []byte("not read by the stub"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	env := &runEnv{artifactsDir: filepath.Join(root, "artifacts"), fixturesDir: fixtures, appPath: bin, isolation: "B-env", sceneRunID: "RUN"}
	rec, err := runSceneS15(env, context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(rec.OutcomeChecks) != 3 {
		t.Fatalf("outcome checks = %d, want the 3 small members (notes: %s)", len(rec.OutcomeChecks), rec.Notes)
	}
	for _, oc := range rec.OutcomeChecks {
		if !oc.NamedOutcome || !oc.BoundaryLogFound || !oc.StateConsistent || oc.CanaryHits != 0 {
			t.Errorf("%s: %+v", oc.Injection, oc)
		}
	}
	if !strings.Contains(rec.Notes, "comic-entry-over-cap pending") || !strings.Contains(rec.Notes, "comic-2gb-pages pending") {
		t.Errorf("the big members must be named as pending: %q", rec.Notes)
	}
	if !strings.Contains(rec.Notes, "subset run") {
		t.Errorf("a subset run must say so: %q", rec.Notes)
	}
	for _, c := range rec.Cells {
		if c.State == "measured" {
			t.Errorf("a subset run closed a cell: %+v", c)
		}
	}
}

// The big members' expectations are copies of the pipeline's wording and the generator's size.
func TestS15ExpectationsMatchTheApp(t *testing.T) {
	read := func(parts ...string) string {
		b, err := os.ReadFile(filepath.Join(append([]string{"..", ".."}, parts...)...))
		if err != nil {
			t.Skip("input absent:", err)
		}
		return string(b)
	}
	if !strings.Contains(read("internal", "limits", "limits.go"), "the limit for one file") {
		t.Error("limits.go no longer says \"the limit for one file\"; update comic-entry-over-cap's boundary")
	}
	if !strings.Contains(read("internal", "limits", "limits.go"), "above the limit of %d megapixels") {
		t.Error("limits.go no longer says \"above the limit of N megapixels\"; update tiff-over-cap's boundary")
	}
	if !strings.Contains(read("internal", "comic", "extract.go"), `"  Pages: %d\n"`) {
		t.Error("comic/extract.go no longer prints the page count line; update comic-2gb-pages' boundary")
	}
	if !strings.Contains(read("internal", "comic", "extract.go"), "const maxPageBytes = 200 << 20") {
		t.Error("the comic page cap moved; the 8 MB pages of comic-2gb-pages and the 2 GiB entry of comic-big-2gb must be re-judged")
	}
	if !strings.Contains(read("internal", "limits", "limits.go"), "MaxArchiveTotalBytes int64 = 4 << 30") {
		t.Error("the archive total cap moved; comic-2gb-pages must stay under it")
	}
}

func TestNoiseComicIsTheSizeItClaims(t *testing.T) {
	// A small instance of the -Big generator: page bytes are incompressible, so the archive is
	// about n * side^2 bytes and every entry is a decodable PNG.
	dir := t.TempDir()
	path := filepath.Join(dir, "noise.cbz")
	const n, side = 3, 64
	if err := writeNoiseComic(path, n, side); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	want := int64(n * side * side)
	if fi.Size() < want || fi.Size() > want+int64(n)*2048 {
		t.Errorf("archive is %d bytes, want ~%d (noise must not compress)", fi.Size(), want)
	}
	// Deterministic: a second generation is byte-identical.
	again := filepath.Join(dir, "again.cbz")
	if err := writeNoiseComic(again, n, side); err != nil {
		t.Fatal(err)
	}
	a, _ := sha256File(path)
	b, _ := sha256File(again)
	if a != b {
		t.Error("the generator is not deterministic")
	}
	// The catalog numbers describe a 2 GiB archive within every budget.
	total := int64(comicPagesCount) * int64(comicPagesSide) * int64(comicPagesSide)
	if total < 2000<<20 || total > 4<<30 {
		t.Errorf("full-size archive would be %d bytes; ticket 12 asks for ~2 GB and the cap is 4 GiB", total)
	}
	if int64(comicPagesSide)*int64(comicPagesSide) > 100_000_000 {
		t.Error("a page would exceed the pixel budget; the fixture would be refused instead of converted")
	}
}
