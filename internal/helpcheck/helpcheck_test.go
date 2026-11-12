package helpcheck

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// The gate is only worth having if it fails when it should. Each case copies the files the gate reads into a scratch
// tree, breaks one thing the contract names (CONTEXT-HELP rule 19), and expects exactly that to be reported.

const repoRoot = "../.."

var gateFiles = []string{
	"cmd/doc-html-ui/help", "help", "cmd/doc-html-ui/ui.html", "cmd/doc-html-ui/main.go", "cmd/doc-html-ui/i18n.js",
	"configs/help-baseline.txt", "tools/helpgen", "assets/sza-kit.css", "index.html", "docs.html", "docs.ru.html", "docs.uk.html",
}

func scratch(t *testing.T) string {
	t.Helper()
	dst := t.TempDir()
	for _, rel := range gateFiles {
		src := filepath.Join(repoRoot, filepath.FromSlash(rel))
		err := filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			r, _ := filepath.Rel(repoRoot, p)
			if strings.Contains(r, "node_modules") {
				return fs.SkipDir
			}
			to := filepath.Join(dst, r)
			if d.IsDir() {
				return os.MkdirAll(to, 0o755)
			}
			raw, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
				return err
			}
			return os.WriteFile(to, raw, 0o644)
		})
		if err != nil {
			t.Fatalf("copy %s: %v", rel, err)
		}
	}
	return dst
}

func edit(t *testing.T, root, rel string, f func(string) string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	raw, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	out := f(string(raw))
	if out == string(raw) {
		t.Fatalf("the edit of %s changed nothing - the case tests nothing", rel)
	}
	if err := os.WriteFile(p, []byte(out), 0o644); err != nil {
		t.Fatal(err)
	}
}

func run(t *testing.T, root string) *Report {
	t.Helper()
	r, err := Run(root)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func mustFail(t *testing.T, r *Report, want string) {
	t.Helper()
	for _, f := range r.Failures {
		if strings.Contains(f, want) {
			return
		}
	}
	t.Errorf("want a failure naming %q, got failures %q (advisories %q)", want, r.Failures, r.Advisories)
}

func TestAPageGitIgnoresFails(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not on PATH")
	}
	root := scratch(t)
	if out, err := exec.Command("git", "-C", root, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte("*secret*\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "help", "en", "secret-file-password.html"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	mustFail(t, run(t, root), "help/en/secret-file-password.html is ignored by git")
}

func TestTheRepositoryPasses(t *testing.T) {
	r := run(t, repoRoot)
	for _, f := range r.Failures {
		t.Error(f)
	}
}

func TestACleanCopyIsClean(t *testing.T) {
	root := scratch(t)
	r := run(t, root)
	if len(r.Failures) != 0 {
		t.Fatalf("a copy of the tree fails: %q", r.Failures)
	}
	if r.Surfaces < 8 || r.Pages != 5 || r.Locales != 3 || r.Links < 100 {
		t.Errorf("the counts look wrong: %+v", r)
	}
}

func TestASurfaceWithNoRowFails(t *testing.T) {
	root := scratch(t)
	// the dialog is still in ui.html; its row is gone
	edit(t, root, MapPath, func(s string) string {
		i := strings.Index(s, `"id": "dlg-cost"`)
		j := strings.Index(s[i:], "},\n")
		return s[:i-len("    {\n      ")] + s[i+j+len("},\n"):]
	})
	mustFail(t, run(t, root), `surface: "dlg-cost" but the map has no such surface`)
}

func TestADialogThatNamesNoSurfaceFails(t *testing.T) {
	root := scratch(t)
	edit(t, root, UIPath, func(s string) string {
		return strings.Replace(s, "title: q.title, message: q.message, tag: run, run, surface: 'dlg-cost',", "title: q.title, message: q.message, tag: run, run,", 1)
	})
	mustFail(t, run(t, root), "an openDialog call names no surface")
}

func TestARowWithNoSurfaceFails(t *testing.T) {
	root := scratch(t)
	edit(t, root, UIPath, func(s string) string { return strings.Replace(s, ` data-help="main.queue"`, "", 1) })
	mustFail(t, run(t, root), "main.queue, which no element of the window carries")
}

func TestARegionWithNoEntryFails(t *testing.T) {
	root := scratch(t)
	edit(t, root, UIPath, func(s string) string {
		return strings.Replace(s, `aria-labelledby="tab-ocr" data-help="settings.ocr" hidden>`, `aria-labelledby="tab-ocr" data-help="settings.ocr2" hidden>`, 1)
	})
	r := run(t, root)
	mustFail(t, r, "settings.ocr2 is in the window but not among the map's settings rows")
	mustFail(t, r, "settings.ocr, which no element of the window carries")
}

func TestNoneAndInheritNeedAReason(t *testing.T) {
	root := scratch(t)
	edit(t, root, MapPath, func(s string) string {
		i := strings.Index(s, `"id": "dlg-notice"`)
		j := strings.Index(s[i:], `"reason"`)
		k := strings.Index(s[i+j:], "\"\n")
		return s[:i+j] + `"reason": ""` + s[i+j+k+1:]
	})
	mustFail(t, run(t, root), "surface dlg-notice is none without a reason")
}

func TestAMissingPageInAShippedLocaleFails(t *testing.T) {
	root := scratch(t)
	for _, base := range []string{HelpDir, MirrorDir} {
		if err := os.Remove(filepath.Join(root, filepath.FromSlash(base), "uk", "run-failed.html")); err != nil {
			t.Fatal(err)
		}
	}
	r := run(t, root)
	mustFail(t, r, "the page run-failed is missing in locale uk")
}

func TestAnUnresolvedLinkFails(t *testing.T) {
	root := scratch(t)
	for _, base := range []string{HelpDir, MirrorDir} {
		edit(t, root, base+"/en/settings.html", func(s string) string {
			return strings.Replace(s, `href="converter.html"`, `href="no-such-page.html"`, 1)
		})
	}
	mustFail(t, run(t, root), "no-such-page.html")
}

func TestARequestToAnotherOriginFails(t *testing.T) {
	root := scratch(t)
	for _, base := range []string{HelpDir, MirrorDir} {
		edit(t, root, base+"/en/index.html", func(s string) string {
			return strings.Replace(s, `<link rel="stylesheet" href="../help.css">`, `<link rel="stylesheet" href="https://fonts.example/outfit.css">`, 1)
		})
	}
	mustFail(t, run(t, root), "makes a request to another origin")
}

func TestAnInlineScriptFails(t *testing.T) {
	root := scratch(t)
	for _, base := range []string{HelpDir, MirrorDir} {
		edit(t, root, base+"/ru/converter.html", func(s string) string { return strings.Replace(s, "</body>", "<script>1</script></body>", 1) })
	}
	mustFail(t, run(t, root), "an inline <script>")
}

func TestALongDashInProseFails(t *testing.T) {
	root := scratch(t)
	dash := string(rune(0x2014))
	for _, base := range []string{HelpDir, MirrorDir} {
		edit(t, root, base+"/uk/settings.html", func(s string) string { return strings.Replace(s, "<h1>", "<h1>"+dash+" ", 1) })
	}
	mustFail(t, run(t, root), "a long dash")
}

func TestAnEntryTheMapPointsAtMustExist(t *testing.T) {
	root := scratch(t)
	for _, base := range []string{HelpDir, MirrorDir} {
		edit(t, root, base+"/en/converter.html", func(s string) string {
			return strings.Replace(s, `<details class="el" id="languages">`, `<details class="el" id="languages-x">`, 1)
		})
	}
	mustFail(t, run(t, root), "the entry #languages (main.languages) is missing")
}

func TestTheTwoCopiesMustAgree(t *testing.T) {
	root := scratch(t)
	edit(t, root, MirrorDir+"/en/index.html", func(s string) string { return s + "\n<!-- drift -->\n" })
	mustFail(t, run(t, root), "must be the same bytes")
}

func TestTheBaselineMayFallAndNeverRise(t *testing.T) {
	root := scratch(t)
	edit(t, root, BaselinePath, func(s string) string { return strings.Replace(s, "locale:zh\n", "", 1) })
	mustFail(t, run(t, root), "the GUI ships zh but the help has no pages in it")

	root = scratch(t)
	edit(t, root, BaselinePath, func(s string) string { return s + "locale:uk\n" })
	r := run(t, root)
	if r.Code() != 3 && len(r.Failures) == 0 {
		t.Errorf("a pruned-able line is an advisory (exit 3), got code %d", r.Code())
	}
	found := false
	for _, a := range r.Advisories {
		found = found || strings.Contains(a, "locale:uk")
	}
	if !found {
		t.Errorf("no advisory about locale:uk: %q", r.Advisories)
	}
}

// committedBaseline is the committed state the scratch tree is compared with: the repository's own file, as it is.
func committedBaseline(t *testing.T) Options {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(BaselinePath)))
	if err != nil {
		t.Fatal(err)
	}
	return Options{Committed: func(rel string) ([]byte, bool, error) { return raw, rel == BaselinePath, nil }}
}

func runWith(t *testing.T, root string, opt Options) *Report {
	t.Helper()
	r, err := RunWith(root, opt)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestAHandAddedBaselineLineIsAnAbsorption(t *testing.T) {
	opt := committedBaseline(t)

	// the line a hand adds so that a new GUI locale without pages passes: it names the amount and the identifier
	root := scratch(t)
	edit(t, root, BaselinePath, func(s string) string { return s + "locale:xx\n" })
	r := runWith(t, root, opt)
	mustFail(t, r, "gains 1 line(s) over the committed file with no recorded reason: locale:xx")

	// a journal line with no reason is no record
	root = scratch(t)
	edit(t, root, BaselinePath, func(s string) string { return "# accepted locale:xx on 2026-10-21:\n" + s + "locale:xx\n" })
	mustFail(t, runWith(t, root, opt), "no recorded reason: locale:xx")

	// pruning is always welcome, and the run without a committed state judges the file by its own rules only
	root = scratch(t)
	edit(t, root, BaselinePath, func(s string) string { return strings.Replace(s, "locale:zh\n", "", 1) })
	r = runWith(t, root, opt)
	for _, f := range r.Failures {
		if strings.Contains(f, "gains") {
			t.Errorf("a pruned line is not an absorption: %s", f)
		}
	}
	root = scratch(t)
	edit(t, root, BaselinePath, func(s string) string { return s + "locale:xx\n" })
	for _, f := range run(t, root).Failures {
		if strings.Contains(f, "gains") {
			t.Errorf("a run with no committed state compares nothing: %s", f)
		}
	}
}

func TestAcceptRecordsTheReasonAndTheGateReadsIt(t *testing.T) {
	opt := committedBaseline(t)
	root := scratch(t)
	// a GUI locale the help has no pages in and the baseline does not hold: the help map drops "uk"
	edit(t, root, MapPath, func(s string) string { return strings.Replace(s, `"uk"`, `"ukx"`, 1) })

	if _, err := Accept(root, "  ", time.Now()); err == nil {
		t.Error("an empty reason must be refused")
	}
	if _, err := Accept(root, "one\ntwo", time.Now()); err == nil {
		t.Error("a reason of two lines must be refused")
	}

	ids, err := Accept(root, "owner tier decision, test", time.Date(2026, 10, 21, 9, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 1 || ids[0] != "locale:uk" {
		t.Fatalf("Accept returned %q, want [locale:uk]", ids)
	}
	raw, _ := os.ReadFile(filepath.Join(root, filepath.FromSlash(BaselinePath)))
	if !strings.Contains(string(raw), "# accepted locale:uk on 2026-10-21: owner tier decision, test\n") {
		t.Errorf("the journal line is not in the file:\n%s", raw)
	}
	b := parseBaseline(raw)
	if len(b.ids) != 11 || !sortedStrings(b.ids) {
		t.Errorf("the identifiers must stay sorted and complete: %q", b.ids)
	}

	// the gate takes the recorded line as an advisory to review, not as a failure
	r := runWith(t, root, opt)
	for _, f := range r.Failures {
		if strings.Contains(f, "gains") {
			t.Errorf("an accepted line is no absorption: %s", f)
		}
	}
	named := false
	for _, a := range r.Advisories {
		named = named || strings.Contains(a, "gains 1 accepted line(s)") && strings.Contains(a, "locale:uk")
	}
	if !named {
		t.Errorf("no advisory about the accepted line: %q", r.Advisories)
	}

	// a second call has nothing left to accept
	if ids, err := Accept(root, "again", time.Now()); err != nil || len(ids) != 0 {
		t.Errorf("a second Accept must be a no-op, got %q, %v", ids, err)
	}
}

func sortedStrings(s []string) bool {
	for i := 1; i < len(s); i++ {
		if s[i-1] > s[i] {
			return false
		}
	}
	return true
}

func TestABaselineNotInHeadYetIsAnAdvisory(t *testing.T) {
	root := scratch(t)
	opt := Options{Committed: func(string) ([]byte, bool, error) { return nil, false, nil }}
	r := runWith(t, root, opt)
	if len(r.Failures) != 0 {
		t.Fatalf("a baseline with no committed state to compare with must not fail: %q", r.Failures)
	}
	found := false
	for _, a := range r.Advisories {
		found = found || strings.Contains(a, "not in HEAD yet")
	}
	if !found {
		t.Errorf("no advisory about the missing committed state: %q", r.Advisories)
	}
}

func TestAnUnreadableCommittedStateIsNotAPass(t *testing.T) {
	opt := Options{Committed: func(string) ([]byte, bool, error) { return nil, false, os.ErrPermission }}
	if _, err := RunWith(scratch(t), opt); err == nil {
		t.Fatal("a committed state that cannot be read must end the run as COULD NOT VERIFY, not as a pass")
	}
}

func TestAnInputChangedAfterGenerationIsAnAdvisoryNotAFailure(t *testing.T) {
	base := run(t, scratch(t))

	root := scratch(t)
	edit(t, root, UIPath, func(s string) string {
		return strings.Replace(s, `<div class="head" data-help="main.header">`, `<div class="head" data-help="main.header" lang="en">`, 1)
	})
	r := run(t, root)
	if len(r.Failures) != 0 {
		t.Fatalf("a changed input must not fail the gate: %q", r.Failures)
	}
	named := false
	for _, a := range r.Advisories {
		named = named || strings.Contains(a, "ui.html (markup and style)")
	}
	if r.Code() != 3 || !named {
		t.Errorf("want an advisory naming ui.html (exit 3), got code %d, %q", r.Code(), r.Advisories)
	}

	// a change to the script alone is not a change of what the window looks like
	root = scratch(t)
	edit(t, root, UIPath, func(s string) string {
		return strings.Replace(s, "const SHORTCUTS = [", "const SHORTCUTS = [ /* x */", 1)
	})
	if r := run(t, root); len(r.Failures) != 0 || len(r.Advisories) != len(base.Advisories) {
		t.Errorf("a script-only edit must keep the digest: failures %q, advisories %q (before: %q)", r.Failures, r.Advisories, base.Advisories)
	}
}

// The full gate and the release pre-flight hold the set to its inputs (ticket E0125): the same difference that is an
// advisory between releases is a failure under StrictFreshness, and the message names the command that clears it - the
// full generator when a window input moved, the fast wording form when only text did.
func TestAStaleSetFailsUnderStrictFreshness(t *testing.T) {
	strict := func(t *testing.T, root string) *Report {
		t.Helper()
		r, err := RunWith(root, Options{StrictFreshness: true})
		if err != nil {
			t.Fatal(err)
		}
		return r
	}

	if r := strict(t, scratch(t)); r.Code() != 0 {
		t.Fatalf("a fresh set must pass under StrictFreshness: failures %q, advisories %q", r.Failures, r.Advisories)
	}

	root := scratch(t)
	edit(t, root, UIPath, func(s string) string {
		return strings.Replace(s, `<div class="head" data-help="main.header">`, `<div class="head" data-help="main.header" lang="en">`, 1)
	})
	r := strict(t, root)
	mustFail(t, r, "ui.html (markup and style)")
	mustFail(t, r, "node tools/helpgen/generate.mjs")
	if len(r.Advisories) != 0 {
		t.Errorf("a strict run reports the stale set as a failure, not also as an advisory: %q", r.Advisories)
	}

	root = scratch(t)
	edit(t, root, CopyDir+"/converter.json", func(s string) string { return s + " " })
	r = strict(t, root)
	mustFail(t, r, "copy/converter.json")
	mustFail(t, r, "node tools/helpgen/repage.mjs")
	for _, f := range r.Failures {
		if strings.Contains(f, "generate.mjs") {
			t.Errorf("a wording-only change must not send the author to the full generator: %q", f)
		}
	}
}
