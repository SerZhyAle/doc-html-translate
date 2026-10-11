package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFixturesAreDeterministic(t *testing.T) {
	a := t.TempDir()
	b := t.TempDir()
	if _, err := generateFixtures(a, false); err != nil {
		t.Fatal(err)
	}
	if _, err := generateFixtures(b, false); err != nil {
		t.Fatal(err)
	}
	for _, f := range fixtureCatalog {
		if f.big {
			continue
		}
		ha := mustSHA(t, filepath.Join(a, f.file))
		hb := mustSHA(t, filepath.Join(b, f.file))
		if ha != hb {
			t.Fatalf("fixture %s is not deterministic: %s vs %s", f.id, ha, hb)
		}
	}
}

func TestPDFMetadataUsesFixtureClock(t *testing.T) {
	dir := t.TempDir()
	for name, gen := range map[string]func(string) error{"pdf-text.pdf": genPdfText, "pdf-scan.pdf": genPdfScan} {
		if err := gen(dir); err != nil {
			t.Fatal(err)
		}
		b, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		for _, field := range []string{"/CreationDate (D:20260101000000)", "/ModDate (D:20260101000000)"} {
			if !bytes.Contains(b, []byte(field)) {
				t.Errorf("%s: metadata is not pinned: %s", name, field)
			}
		}
	}
}

func TestEpubFixtureIsAReadableEpub(t *testing.T) {
	dir := t.TempDir()
	if _, err := generateFixtures(dir, false); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"epub-small.epub", "epub-nested.epub"} {
		r, err := zip.OpenReader(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		defer r.Close()
		var hasMimetype, hasNcx bool
		for _, zf := range r.File {
			switch zf.Name {
			case "mimetype":
				hasMimetype = true
				f, err := zf.Open()
				if err != nil {
					t.Fatal(err)
				}
				var buf [32]byte
				n, _ := io.ReadFull(f, buf[:20])
				f.Close()
				if string(buf[:n]) != "application/epub+zip" {
					t.Fatalf("%s: mimetype content %q", name, buf[:n])
				}
			case "OEBPS/toc.ncx":
				hasNcx = true
			}
		}
		if !hasMimetype || !hasNcx {
			t.Fatalf("%s: missing mimetype(%v) or toc.ncx(%v)", name, hasMimetype, hasNcx)
		}
	}
}

func TestCorruptFixtureIsCorrupt(t *testing.T) {
	dir := t.TempDir()
	if _, err := generateFixtures(dir, false); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "comic-corrupt.cbz"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := zip.NewReader(bytes.NewReader(data), int64(len(data))); err == nil {
		t.Fatal("comic-corrupt.cbz must not open as a zip")
	}
}

func mustSHA(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func TestRecordRoundtripAndFormatRefusal(t *testing.T) {
	dir := t.TempDir()
	rec := &record{
		RunID:     "20260101-000000-S03",
		Kind:      "scene",
		CreatedAt: fixtureTime,
		Scene:     "S03",
		Isolation: "B-env",
		Valid:     true,
		Cells: []cellResult{{
			Scene: "S03", Property: 1, State: "open",
			Budgets: []budgetVerdict{{ID: "H5", Verdict: "pass", Observed: "900 ms", Limit: "<= 1500 ms"}},
		}},
	}
	p, err := saveRecord(dir, rec)
	if err != nil {
		t.Fatal(err)
	}
	got, err := loadRecord(p)
	if err != nil {
		t.Fatal(err)
	}
	if got.Scene != "S03" || len(got.Cells) != 1 || got.Cells[0].Budgets[0].ID != "H5" {
		t.Fatalf("roundtrip mismatch: %+v", got)
	}

	// saveRecord stamps the current format, so the refusal is driven by hand-written JSON.
	bad := *rec
	bad.Format = 99
	bb, _ := json.Marshal(bad)
	if err := os.MkdirAll(filepath.Join(dir, "bad"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "bad", "record.json"), bb, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := loadRecord(filepath.Join(dir, "bad", "record.json")); err == nil {
		t.Fatal("an unknown record format must be refused")
	}
}

func TestPercentileNearestRank(t *testing.T) {
	xs := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	if got := percentile(xs, 0.5); got != 5 {
		t.Fatalf("p50 = %v, want 5", got)
	}
	if got := percentile(xs, 0.95); got != 10 {
		t.Fatalf("p95 = %v, want 10 (nearest-rank on 10 samples)", got)
	}
	if got := percentile(nil, 0.95); got != 0 {
		t.Fatalf("p95 of empty = %v, want 0", got)
	}
}

func TestBudgetCheckTwoFormRule(t *testing.T) {
	ctrl := &record{Measurements: []measurement{{Probe: "P2.host", N: 10, P95Ms: 10}}}

	// Absolute pass + within 2x control: pass (the relative form compares the same probe).
	if v := budgetCheck(findBudget("H5"), "15 ms", 15, ctrl, "P2.host"); v.Verdict != "pass" {
		t.Fatalf("H5 15ms with control p95 10ms = %s, want pass", v.Verdict)
	}
	// Absolute fail: fail even when within 2x control.
	if v := budgetCheck(findBudget("H5"), "2000 ms", 2000, ctrl, "P2.host"); v.Verdict != "fail" {
		t.Fatalf("H5 2000ms = %s, want fail", v.Verdict)
	}
	// Relative fail: 3x control p95 breaches the two-form rule even under the absolute bar.
	if v := budgetCheck(findBudget("H5"), "900 ms", 900, ctrl, "P2.host"); v.Verdict != "fail" {
		t.Fatalf("H5 900ms with control p95 10ms = %s, want fail (relative form)", v.Verdict)
	}
	// No control number for the probe: an advisory, never a pass.
	if v := budgetCheck(findBudget("H5"), "15 ms", 15, ctrl, "P6.fsync-writer"); v.Verdict != "advisory" {
		t.Fatalf("H5 with no control number for the probe = %s, want advisory", v.Verdict)
	}
	// The hard bar is absolute-only by definition.
	if v := budgetCheck(findBudget("H1"), "5100 ms", 5100, ctrl, "P2.host"); v.Verdict != "fail" {
		t.Fatalf("H1 above the watchdog tick = %s, want fail", v.Verdict)
	}
}

func TestSceneS03LogPhrasesMatchThePipeline(t *testing.T) {
	// The reuse verdicts read the pipeline's own log lines; a wording change in
	// internal/pipeline must fail here, not silently pass every future S03 run.
	b, err := os.ReadFile(filepath.Join("..", "..", "internal", "pipeline", "pipeline.go"))
	if err != nil {
		t.Skip("input absent:", err)
	}
	src := string(b)
	if !strings.Contains(src, logReused) || !strings.Contains(src, logForcedRebuild) {
		t.Fatalf("pipeline no longer logs %q / %q as written; update scene_s03.go's pins", logReused, logForcedRebuild)
	}
}
