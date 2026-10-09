package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func exitCode(n int) *int { return &n }

func baseCase(class string) Case {
	return Case{ID: "en-x-case", Class: class, Language: "en", Role: "primary", Split: "dev",
		SHA256: strings.Repeat("a", 64), Bytes: 1, File: "x/y.pdf", Expect: "expect/en-x-case.json",
		Rights: Rights{Licence: "PD"}}
}

func resultWith(t *testing.T, p *Probe, text, plates string) *Result {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "text.txt"), []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "plates.txt"), []byte(plates), 0o644); err != nil {
		t.Fatal(err)
	}
	return &Result{CaseID: "en-x-case", Edition: "windows", OCRMode: "default", Source: "ok",
		Convert: &Convert{ExitCode: exitCode(0), Index: "html/index.html"}, Probe: p, dir: dir}
}

func TestJudgeCleanPageIsCappedByTheHumanGates(t *testing.T) {
	c := baseCase("text-pdf")
	e := &Expectation{Origin: "draft", TextLayer: "full", SourceTextChars: 100, Snippets: []string{"The General Assembly"}}
	r := resultWith(t, &Probe{Lang: "en", TextChars: 98, TocEntries: 3}, "Preamble. The General\nAssembly, meeting", "")
	j := Judge(c, e, r)
	if j.Auto != Pass {
		t.Fatalf("auto = %s (%v %v %v), want PASS", j.Auto, j.Fails, j.Gaps, j.Advisories)
	}
	if j.Campaign != CouldNotVerify {
		t.Fatalf("campaign = %s, want COULD NOT VERIFY while rights and expectation are unreviewed", j.Campaign)
	}

	c.Rights.ReviewedBy, c.Rights.ReviewedOn = "A Person", "2026-09-30"
	e.Origin, e.ReviewedBy, e.ReviewedOn = "human", "A Person", "2026-09-30"
	if j = Judge(c, e, r); j.Campaign != Pass {
		t.Fatalf("campaign = %s with both gates closed, want PASS", j.Campaign)
	}
}

func TestJudgeFailures(t *testing.T) {
	cases := []struct {
		name  string
		class string
		exp   *Expectation
		res   func(*Result)
		want  string // substring of a failure
	}{
		{"conversion exit", "text-pdf", &Expectation{}, func(r *Result) { r.Convert.ExitCode = exitCode(3) }, "conversion failed (exit 3)"},
		{"broken image", "epub", &Expectation{}, func(r *Result) { r.Probe.ImagesBroken, r.Probe.Images = 2, 5 }, "2 of 5 images"},
		{"dangling link", "epub", &Expectation{}, func(r *Result) { r.Probe.InternalLinksMissing, r.Probe.InternalLinks = 1, 9 }, "in-page links"},
		{"lost pages", "comic-archive", &Expectation{Pages: 11}, func(r *Result) { r.Probe.ImagesLoaded = 9 }, "9 of 11 pages"},
		{"text loss", "text-pdf", &Expectation{TextLayer: "full", SourceTextChars: 1000}, func(r *Result) { r.Probe.TextChars = 300 }, "30% of the source text"},
		{"wrong language", "epub", &Expectation{}, func(r *Result) { r.Probe.Lang = "zh-CN" }, `document language "zh-CN"`},
		{"unplated lettering", "comic-image", &Expectation{Pages: 1, Lettering: []string{"WHAT?!!"}}, func(r *Result) { r.Probe.ImagesLoaded = 1 }, "no OCR plate"},
		{"refusal notice", "epub", &Expectation{}, func(r *Result) { r.Probe.Notice = "Cannot open this file" }, "notice instead of the document"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := resultWith(t, &Probe{Lang: "en", TocEntries: 1}, "", "")
			tc.res(r)
			j := Judge(baseCase(tc.class), tc.exp, r)
			if j.Auto != Fail || j.Campaign != Fail {
				t.Fatalf("auto %s campaign %s, want FAIL both (fails %v)", j.Auto, j.Campaign, j.Fails)
			}
			if !strings.Contains(strings.Join(j.Fails, "\n"), tc.want) {
				t.Fatalf("fails %q do not mention %q", j.Fails, tc.want)
			}
		})
	}
}

func TestJudgeMissingInputIsNeverAPass(t *testing.T) {
	for _, src := range []string{"missing", "hash-changed"} {
		j := Judge(baseCase("epub"), &Expectation{}, &Result{Source: src})
		if j.Auto != CouldNotVerify || j.Campaign != CouldNotVerify {
			t.Fatalf("%s: auto %s campaign %s, want COULD NOT VERIFY", src, j.Auto, j.Campaign)
		}
	}
	r := resultWith(t, &Probe{Lang: "zh", ImagesLoaded: 1}, "", "")
	r.EnvironmentGap = "OCR language chi_tra is not offered by either edition"
	c := baseCase("comic-image")
	c.Language = "zh"
	if j := Judge(c, &Expectation{Pages: 1}, r); j.Auto != CouldNotVerify {
		t.Fatalf("an environment gap graded %s, want COULD NOT VERIFY", j.Auto)
	}
}

func TestJudgeDraftSnippetsAdviseAndReviewedOnesFail(t *testing.T) {
	c := baseCase("epub")
	e := &Expectation{Origin: "draft", Snippets: []string{"a sentence that is not on the page"}}
	r := resultWith(t, &Probe{Lang: "en", TocEntries: 2}, "something else entirely", "")
	if j := Judge(c, e, r); j.Auto != PassAdvisories {
		t.Fatalf("a missing draft snippet graded %s, want PASS WITH ADVISORIES", j.Auto)
	}
	e.Origin, e.ReviewedBy, e.ReviewedOn = "human", "A Person", "2026-09-30"
	if j := Judge(c, e, r); j.Auto != Fail {
		t.Fatalf("a missing reviewed snippet graded %s, want FAIL", j.Auto)
	}
}

func TestJudgeIntentionalDifference(t *testing.T) {
	r := resultWith(t, &Probe{Notice: "This comic needs the desktop app"}, "", "")
	r.Edition, r.Convert = "extension", nil
	r.IntentionalDifference = "the extension declines CBR/CB7"
	if j := Judge(baseCase("comic-archive"), &Expectation{Pages: 37}, r); j.Auto != PassAdvisories {
		t.Fatalf("the documented CB7 notice graded %s (%v), want PASS WITH ADVISORIES", j.Auto, j.Fails)
	}
	r.Probe.Notice = ""
	if j := Judge(baseCase("comic-archive"), &Expectation{Pages: 37}, r); j.Auto != Fail {
		t.Fatalf("a missing CB7 notice graded %s, want FAIL", j.Auto)
	}
}

func TestSnippetsIgnoreSpacingAndCheckOrder(t *testing.T) {
	found, ordered, _ := snippetsIn("第一段 こんにちは\n世界 。第二段", []string{"こんにちは世界", "第二段"})
	if found != 2 || !ordered {
		t.Fatalf("found %d ordered %v", found, ordered)
	}
	_, ordered, _ = snippetsIn("B then A", []string{"A", "B"})
	if ordered {
		t.Fatal("reversed snippets reported in order")
	}
}

func TestCharRecall(t *testing.T) {
	if got := charRecall("제 6 화", "제6화 포션"); got != 1 {
		t.Fatalf("recall %v, want 1", got)
	}
	if got := charRecall("ABCD", "ab"); got != 0.5 {
		t.Fatalf("recall %v, want 0.5 (case-folded)", got)
	}
	if got := charRecall("", "x"); got != 0 {
		t.Fatalf("empty reference recall %v", got)
	}
}

func TestManifestValidate(t *testing.T) {
	good := baseCase("epub")
	m := &Manifest{SchemaVersion: SchemaVersion, Root: "test_doc", Cases: []Case{good}}
	if bad := m.Validate(); len(bad) != 0 {
		t.Fatalf("valid manifest reported %v", bad)
	}
	dup := good
	dup.ID = "en-x-other"
	dup.Expect = "expect/en-x-other.json"
	selfSigned := good
	selfSigned.ID, selfSigned.File, selfSigned.Expect = "en-x-third", "x/z.pdf", "expect/en-x-third.json"
	selfSigned.Rights.PermitsCommit = true
	m.Cases = append(m.Cases, dup, selfSigned)
	bad := strings.Join(m.Validate(), "\n")
	for _, want := range []string{"same file as en-x-case", "permitsCommit without a human rights review"} {
		if !strings.Contains(bad, want) {
			t.Errorf("validation %q does not report %q", bad, want)
		}
	}
}

// The committed manifest must stay valid - the one check that needs no media.
func TestCommittedManifestIsValid(t *testing.T) {
	m, err := LoadManifest(filepath.Join("..", "..", "DEV", "doccorpus", "cases.json"))
	if err != nil {
		t.Fatal(err)
	}
	if bad := m.Validate(); len(bad) > 0 {
		t.Fatalf("DEV/doccorpus/cases.json: %v", bad)
	}
	for _, c := range m.Cases {
		e, err := LoadExpectation(filepath.Join("..", "..", "DEV", "doccorpus", c.Expect))
		if err != nil {
			t.Errorf("%s: %v", c.ID, err)
			continue
		}
		if e.CaseID != c.ID {
			t.Errorf("%s: expectation names %s", c.ID, e.CaseID)
		}
		if e.Origin == "human" && !e.IsTruth() {
			t.Errorf("%s: origin human without a signed review", c.ID)
		}
	}
}

func intp(n int) *int { return &n }

func TestJudgeUnsettledCollectionIsIncompleteEvidenceNotPageLoss(t *testing.T) {
	c := baseCase("scanned-pdf")
	e := &Expectation{Pages: 184, TextLayer: "none"}
	r := resultWith(t, &Probe{Lang: "en", PageUnits: 12, Notice: ""}, "", "")
	r.Truncated = true
	r.Collection = &Collection{Outcome: "timeout", Stage: "rendering-active", Rendered: 12, Total: intp(184), Polls: 90}
	j := Judge(c, e, r)
	if len(j.Fails) != 0 {
		t.Fatalf("an unsettled run failed: %v", j.Fails)
	}
	if j.Auto != CouldNotVerify {
		t.Fatalf("auto = %s, want COULD NOT VERIFY", j.Auto)
	}
	if !hasGap(j, "incomplete evidence: stage rendering-active, 12/184") {
		t.Fatalf("gaps %v lack the incomplete-evidence line", j.Gaps)
	}

	// The same page count from a settled run is a real loss.
	r.Truncated = false
	r.Collection = &Collection{Outcome: "settled", Stage: "settled", Rendered: 12, Total: intp(184)}
	j = Judge(c, e, r)
	if j.Auto != Fail || !strings.Contains(strings.Join(j.Fails, "|"), "12 of 184 pages reached the reader") {
		t.Fatalf("settled run with 12/184 pages: auto %s fails %v, want the page-loss FAIL", j.Auto, j.Fails)
	}
}

func TestJudgeIncompleteEvidenceProgress(t *testing.T) {
	cases := []struct {
		name string
		c    Collection
		want string
	}{
		{"ocr", Collection{Outcome: "timeout", Stage: "ocr-active", Rendered: 184, Total: intp(184), OCRDone: intp(55), OCRTotal: intp(184)}, "stage ocr-active, 55/184"},
		{"extracting", Collection{Outcome: "timeout", Stage: "extracting", Rendered: 100, Total: intp(184), ExtractionPending: 30}, "stage extracting, 70/100"},
		{"no total", Collection{Outcome: "timeout", Stage: "starting"}, "stage starting, 0/?"},
		{"producer error", Collection{Outcome: "producer-error", Stage: "producer-error", Rendered: 3, Total: intp(9)}, "stage producer-error, 3/9"},
	}
	for _, tc := range cases {
		r := resultWith(t, &Probe{Lang: "en"}, "", "")
		r.Collection = &tc.c
		if j := Judge(baseCase("epub"), &Expectation{}, r); !hasGap(j, "incomplete evidence: "+tc.want) {
			t.Errorf("%s: gaps %v, want %q", tc.name, j.Gaps, tc.want)
		}
	}
}

func TestJudgeSettledCollectionAddsNoGap(t *testing.T) {
	r := resultWith(t, &Probe{Lang: "en", TocEntries: 1}, "", "")
	r.Collection = &Collection{Outcome: "settled", Stage: "settled", Rendered: 5, Total: intp(5)}
	j := Judge(baseCase("epub"), &Expectation{}, r)
	if hasGap(j, "incomplete evidence") || j.Auto != Pass {
		t.Fatalf("settled run graded %s with gaps %v", j.Auto, j.Gaps)
	}
}

func TestJudgeLegacyTruncatedResultStillCouldNotVerify(t *testing.T) {
	r := resultWith(t, &Probe{Lang: "en", PageUnits: 2}, "", "")
	r.Truncated = true
	j := Judge(baseCase("scanned-pdf"), &Expectation{Pages: 9}, r)
	if j.Auto != CouldNotVerify || len(j.Fails) != 0 {
		t.Fatalf("legacy truncated result: auto %s fails %v, want COULD NOT VERIFY", j.Auto, j.Fails)
	}
}

func hasGap(j Judgement, sub string) bool {
	for _, g := range j.Gaps {
		if strings.Contains(g, sub) {
			return true
		}
	}
	return false
}
