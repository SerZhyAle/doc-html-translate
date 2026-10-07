package tests

// Drives scripts/release-notes.ps1 (ticket 103, SITE-STRUCTURE rules 2 and 14) to every outcome it
// documents, over a small scratch repository built here, and holds the real source and page of this
// repository to the same check. A rule no test turns red is a rule nobody knows still works.

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

const scratchNotesPage = `<!doctype html>
<html lang="en"><body><main>
    <!-- release-notes:begin releases (rendered) -->
    <!-- release-notes:end releases -->
</main></body></html>
`

func scratchNotesRecord(version, date, en, ru, uk string) string {
	return fmt.Sprintf(`{"version":%q,"date":%q,"notes":{"en":[%q],"ru":[%q],"uk":[%q]}}`, version, date, en, ru, uk)
}

func scratchNotesSource(records ...string) string {
	return `{"schema":1,"link":{"en":"Downloads","ru":"Файлы","uk":"Файли"},"releases":[` + strings.Join(records, ",") + `]}`
}

func TestReleaseNotesOutcomes(t *testing.T) {
	pwsh := findPwsh(t)
	dir, git := scratchRepo(t, "release-notes.ps1", "lib/verdict.ps1")
	w := func(rel, content string) { writeFile(t, filepath.Join(dir, filepath.FromSlash(rel)), content) }
	run := func(what string, code int, prefix, fragment string, args ...string) {
		expectRun(t, pwsh, dir, "release-notes.ps1", what, code, prefix, fragment, args...)
	}
	newer := scratchNotesRecord("26.0201.1000", "2026-02-01", "Newer.", "Новее.", "Новіше.")
	older := scratchNotesRecord("26.0101.0900", "2026-01-01", "Older.", "Старее.", "Старіше.")

	w("docs/release-notes.json", scratchNotesSource(newer, older))
	w("release-notes.html", scratchNotesPage)
	git("add", "-A")
	git("commit", "-q", "-m", "x")
	run("a clone with no release tags", 2, "release-notes: COULD NOT VERIFY", "no app release tags")

	git("tag", "v26.0101.0900")
	git("tag", "v26.0201.1000")
	git("tag", "ext-cws-v26.0301.0000") // the extension's tags are not releases of the app
	run("the page is not rendered yet", 1, "release-notes: FAIL", "is not the render")
	run("-Render fills the block, then the check passes", 0, "release-notes: PASS", "", "-Render")
	page := readScratch(t, dir, "release-notes.html")
	for _, want := range []string{`id="v26-0201-1000"`, `<ul data-l="ua">`, `<li>Новіше.</li>`, `releases/tag/v26.0101.0900`, "Файли"} {
		if !strings.Contains(page, want) {
			t.Errorf("the render lacks %q:\n%s", want, page)
		}
	}
	if strings.Index(page, "26.0201.1000") > strings.Index(page, "26.0101.0900") {
		t.Error("the render does not keep the source order, newest first")
	}
	run("a clean second run", 0, "release-notes: PASS", "")
	w("release-notes.html", strings.ReplaceAll(page, "\n", "\r\n"))
	run("a Windows CRLF checkout is the same render", 0, "release-notes: PASS", "")
	w("release-notes.html", strings.ReplaceAll(strings.Replace(page, "<li>Newer.</li>", "<li>Changed on Windows.</li>", 1), "\n", "\r\n"))
	run("CRLF does not hide a content edit", 1, "release-notes: FAIL", "is not the render")

	w("release-notes.html", strings.Replace(page, "<li>Newer.</li>", "<li>Newer, by hand.</li>", 1))
	run("a rendered block edited by hand", 1, "release-notes: FAIL", "is not the render")
	run("-Render restores it", 0, "release-notes: PASS", "", "-Render")

	git("tag", "v26.0215.1200")
	run("a tag with no notes", 1, "release-notes: FAIL", "tag v26.0215.1200 has no record")
	git("tag", "-d", "v26.0215.1200")

	run("the newest record may be the release in preparation", 0, "release-notes: PASS", "")
	git("tag", "-d", "v26.0201.1000")
	run("the newest record without its tag", 0, "release-notes: PASS", "no tag yet")
	git("tag", "-d", "v26.0101.0900")
	git("tag", "v26.0201.1000")
	run("an older record without its tag", 1, "release-notes: FAIL", "26.0101.0900 has no tag v26.0101.0900")
	git("tag", "v26.0101.0900")

	broken := func(what, source, fragment string) {
		w("docs/release-notes.json", source)
		run(what, 1, "release-notes: FAIL", fragment)
	}
	broken("a date that disagrees with the version",
		scratchNotesSource(newer, scratchNotesRecord("26.0101.0900", "2026-01-02", "Older.", "Старее.", "Старіше.")), "disagrees with the version")
	broken("a record out of order", scratchNotesSource(older, newer), "not strictly descending")
	broken("a record twice", scratchNotesSource(newer, newer, older), "not strictly descending")
	broken("a long dash in a bullet",
		scratchNotesSource(scratchNotesRecord("26.0201.1000", "2026-02-01", "Newer \u2014 and more.", "Новее.", "Новіше."), older), "long dash")
	broken("three dots in a bullet",
		scratchNotesSource(scratchNotesRecord("26.0201.1000", "2026-02-01", "Newer.", "Новее...", "Новіше."), older), "three dots")
	broken("a language with no notes",
		scratchNotesSource(strings.Replace(newer, `"uk":["Новіше."]`, `"uk":[]`, 1), older), "no uk notes")
	broken("a language that is not mirrored",
		scratchNotesSource(strings.Replace(newer, `"ru":["Новее."]`, `"ru":["Новее.","Ещё."]`, 1), older), "not mirrored")
	broken("a version of the wrong shape",
		scratchNotesSource(scratchNotesRecord("26.201.1000", "2026-02-01", "Newer.", "Новее.", "Новіше."), older), "not of the shape YY.MMDD.HHmm")
	w("docs/release-notes.json", strings.Replace(scratchNotesSource(newer, older), `"link":{"en":"Downloads","ru":"Файлы","uk":"Файли"},`, "", 1))
	run("no link label", 1, "release-notes: FAIL", "link.en")
	w("docs/release-notes.json", `{`)
	run("a source that is not JSON", 1, "release-notes: FAIL", "")

	w("docs/release-notes.json", scratchNotesSource(newer, older))
	w("release-notes.html", "<main></main>\n")
	run("a page with no markers", 1, "release-notes: FAIL", "must carry exactly one")
	if err := os.Remove(filepath.Join(dir, "release-notes.html")); err != nil {
		t.Fatal(err)
	}
	run("no page", 2, "release-notes: COULD NOT VERIFY", "")

	outside := t.TempDir()
	writeFile(t, filepath.Join(outside, "scripts", "release-notes.ps1"), readRepoFile(t, "scripts", "release-notes.ps1"))
	writeFile(t, filepath.Join(outside, "scripts", "lib", "verdict.ps1"), readRepoFile(t, "scripts", "lib", "verdict.ps1"))
	expectRun(t, pwsh, outside, "release-notes.ps1", "not a git checkout", 2, "release-notes: COULD NOT VERIFY", "")
}

// The page this repository serves is the render of the source it carries, and its source keeps the
// shape the page and the release flow rely on. The tag half of the check needs the clone's tags, so
// that half runs in scripts/check.ps1, not here.
func TestReleaseNotesPageMatchesSource(t *testing.T) {
	src := readRepoFile(t, "docs", "release-notes.json")
	versions := regexp.MustCompile(`"version":\s*"([^"]+)"`).FindAllStringSubmatch(src, -1)
	if len(versions) == 0 {
		t.Fatal("docs/release-notes.json holds no release record")
	}
	page := readRepoFile(t, "release-notes.html")
	for _, m := range versions {
		id := `id="v` + strings.ReplaceAll(m[1], ".", "-") + `"`
		if !strings.Contains(page, id) {
			t.Errorf("release-notes.html lacks the section %s of docs/release-notes.json; run scripts/release-notes.ps1 -Render", id)
		}
	}
	if got := strings.Count(page, `<section class="release"`); got != len(versions) {
		t.Errorf("release-notes.html has %d release sections, the source has %d records; run scripts/release-notes.ps1 -Render", got, len(versions))
	}
}
