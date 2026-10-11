package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"doc-html-translate/internal/outputpath"
)

// runS16MemberAgainstStub runs one member function against the stub CLI and judges it with the scene's own
// judgement, so a wiring or assertion defect fails here and not on the first live run.
func runS16MemberAgainstStub(t *testing.T, mode string, run func(*runEnv, *record, string) (*injectionResult, error)) outcomeCheck {
	t.Helper()
	t.Setenv("AUDITSTAND_STUB_CLI", "1")
	t.Setenv("AUDITSTAND_STUB_MODE", mode)
	bin, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	env := &runEnv{
		artifactsDir: filepath.Join(root, "artifacts"),
		fixturesDir:  filepath.Join(root, "fixtures"),
		appPath:      bin,
		isolation:    "B-env",
		sceneRunID:   "RUN",
	}
	if err := os.MkdirAll(env.fixturesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(env.fixturesDir, "epub-ocr-halfimage.epub"), []byte("not read by the stub"), 0o644); err != nil {
		t.Fatal(err)
	}
	app, err := env.resolveApp()
	if err != nil {
		t.Fatal(err)
	}
	rec := &record{Canary: "dht-audit-canary-test", Binaries: map[string]binaryID{"app": app}}
	runDir := filepath.Join(env.runDir("RUN"), "c1")
	inj, err := run(env, rec, runDir)
	if err != nil {
		t.Fatalf("member: %v", err)
	}
	return judgeInjection("member", inj, rec.Canary)
}

func TestS16OcrImageFailureHoldsWhenTheBookContinues(t *testing.T) {
	oc := runS16MemberAgainstStub(t, "", runOcrImageFailure)
	if !oc.NamedOutcome || !oc.BoundaryLogFound || oc.CanaryHits != 0 {
		t.Fatalf("a best-effort OCR run that continued must pass: %+v", oc)
	}
	if !strings.Contains(oc.Detail, "image(s) overlaid around the failed one") {
		t.Errorf("the continuation proof is missing from the detail: %q", oc.Detail)
	}
}

func TestS16OcrImageFailureAbortingTheBookIsAViolation(t *testing.T) {
	oc := runS16MemberAgainstStub(t, "ocr-aborts", runOcrImageFailure)
	if oc.NamedOutcome {
		t.Fatalf("exit 3 for a single failed image breaks the best-effort decision: %+v", oc)
	}
}

func TestS16OcrSilentFailureIsNotAcceptedAsNamed(t *testing.T) {
	oc := runS16MemberAgainstStub(t, "ocr-silent", runOcrImageFailure)
	if oc.BoundaryLogFound {
		t.Fatalf("a run that never names the failed image must not satisfy the boundary: %+v", oc)
	}
}

func TestS16MissingFileDOIsNamedAndAsksNoPassword(t *testing.T) {
	oc := runS16MemberAgainstStub(t, "", runMissingFileDO)
	if !oc.NamedOutcome || !oc.BoundaryLogFound || !oc.StateConsistent {
		t.Fatalf("missing FileDO must end in a named, consistent outcome: %+v", oc)
	}
	if oc.ExitCode == 0 {
		t.Errorf("exit code 0 for a missing helper reads as success")
	}
}

func TestS16FileDOQuestionBeforeLookupIsCaught(t *testing.T) {
	oc := runS16MemberAgainstStub(t, "filedo-asks-password", runMissingFileDO)
	if oc.BoundaryLogFound {
		t.Fatalf("a message that does not promise 'no password was asked' must fail the boundary: %+v", oc)
	}
}

func TestOverlaySummaryRegexpReadsTheRealLineShapes(t *testing.T) {
	for line, want := range map[string][2]string{
		"  OCR overlay: 2 image(s) overlaid, 1 failed":                       {"2", "1"},
		"  OCR overlay: 3 image(s) overlaid, 2 with no text found, 1 failed": {"3", "1"},
		"[21:41:48]   OCR overlay: 0 image(s) overlaid, 5 failed":            {"0", "5"},
	} {
		m := overlaidRe.FindStringSubmatch(line)
		if m == nil || m[1] != want[0] || m[2] != want[1] {
			t.Errorf("%q -> %v, want %v", line, m, want)
		}
	}
	// The pipeline builds the line from these fragments; a rewording must fail here.
	b, err := os.ReadFile(filepath.Join("..", "..", "internal", "pipeline", "ocrstep.go"))
	if err != nil {
		t.Skip("input absent:", err)
	}
	src := string(b)
	for _, frag := range []string{`"%d image(s) overlaid"`, `", %d with no text found"`, `", %d failed"`, `"  WARNING: OCR failed on %s: %v\n"`} {
		if !strings.Contains(src, frag) {
			t.Errorf("ocrstep.go no longer contains %s; update scene_s16.go's overlaidRe and boundary", frag)
		}
	}
}

// A failed run legitimately owns its folder (the ownership record exists from the moment the
// folder is claimed); what it must not carry is the record's "complete" section.
func TestNoCompletionRecordOnFailureTellsOwnershipFromCompletion(t *testing.T) {
	out := t.TempDir()
	book := filepath.Join(out, "book")
	if err := os.MkdirAll(book, 0o755); err != nil {
		t.Fatal(err)
	}
	failed := &injectionResult{res: &execResult{ExitCode: 3}, outDir: out}

	if err := os.WriteFile(filepath.Join(book, outputpath.MarkerName), []byte(`{"tool":"doc-html-translate","source":"x"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if !noCompletionRecordOnFailure(failed) {
		t.Error("an ownership record without a completion section is the normal state of a failed run")
	}

	src := filepath.Join(out, "book.epub")
	if err := os.WriteFile(src, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := outputpath.MarkComplete(book, src, outputpath.Completion{Translation: "none"}); err != nil {
		t.Fatal(err)
	}
	if noCompletionRecordOnFailure(failed) {
		t.Error("a failed run that left a completion record would let the output reopen as finished")
	}
}
