package helpcheck

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
