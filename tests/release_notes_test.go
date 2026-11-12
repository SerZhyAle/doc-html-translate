package tests

// Drives scripts/release-notes.ps1 (ticket 103, SITE-STRUCTURE rules 2 and 14) to every outcome it
// documents, over a small scratch repository built here, and holds the real source and page of this
// repository to the same check. A rule no test turns red is a rule nobody knows still works.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
	w(".sza-canon.json", `{"site":{"pages":["release-notes.html"]}}`)
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

	// The label's {version}, the landing's block and the extension's version (ticket E0111, F29, F67, F15).
	w("docs/release-notes.json", strings.Replace(scratchNotesSource(newer, older), `"en":"Downloads"`, `"en":"Downloads for {version}"`, 1))
	run("-Render names the version in each link", 0, "release-notes: PASS", "", "-Render")
	if page := readScratch(t, dir, "release-notes.html"); !strings.Contains(page, "Downloads for 26.0201.1000") || !strings.Contains(page, "Downloads for 26.0101.0900") {
		t.Errorf("the link label does not name its record's version:\n%s", page)
	}
	notesWith := func(extra string) string {
		return strings.Replace(scratchNotesSource(newer, older), `"releases":`, extra+`"releases":`, 1)
	}
	w("docs/release-notes.json", scratchNotesSource(newer, older))
	run("-Render restores the page", 0, "release-notes: PASS", "", "-Render")
	w(".sza-canon.json", `{"site":{"pages":["release-notes.html","landing.html"]}}`)
	landing := "<main>\n    <!-- release-notes:begin whatsnew -->\n    <!-- release-notes:end whatsnew -->\n<a>Chrome Web Store<span data-rf=\"extension\"></span></a>\n</main>\n"
	w("landing.html", landing)
	w("docs/release-notes.json", notesWith(`"whatsNew":{"en":"New in {version}","ru":"Новое в {version}","uk":"Нове в {version}"},`))
	run("a page shows the extension's version and the source records none", 1, "release-notes: FAIL", "records none")
	full := notesWith(`"whatsNew":{"en":"New in {version}","ru":"Новое в {version}","uk":"Нове в {version}"},"extension":{"version":"26.1021.41"},`)
	w("docs/release-notes.json", full)
	run("the landing is not rendered yet", 1, "release-notes: FAIL", "its release facts are not the render")
	run("-Render fills the landing", 0, "release-notes: PASS", "", "-Render")
	rendered := readScratch(t, dir, "landing.html")
	for _, want := range []string{`<span data-rf="extension"> v26.1021.41</span>`, `New in 26.0201.1000`, `<ul data-l="ua">`, `<li>Новіше.</li>`} {
		if !strings.Contains(rendered, want) {
			t.Errorf("the landing's render lacks %q:\n%s", want, rendered)
		}
	}
	if strings.Contains(rendered, "Older.") {
		t.Errorf("the landing's block holds an older record:\n%s", rendered)
	}
	run("a clean second run of the landing", 0, "release-notes: PASS", "")
	w("landing.html", strings.Replace(rendered, "v26.1021.41", "v9.9.9", 1))
	run("a version typed by hand beside a store button", 1, "release-notes: FAIL", "its release facts are not the render")
	w("landing.html", strings.Replace(rendered, "<li>Newer.</li>", "<li>Newer, by hand.</li>", 1))
	run("the landing's block edited by hand", 1, "release-notes: FAIL", "its release facts are not the render")
	w("landing.html", rendered)
	w("docs/release-notes.json", strings.Replace(full, "26.1021.41", "26.0702.1342", 1))
	run("an extension version the stores refuse", 1, "release-notes: FAIL", "integer shape")
	w("docs/release-notes.json", notesWith(`"extension":{"version":"26.1021.41"},`))
	w("landing.html", landing)
	run("a landing block with no heading in the source", 1, "release-notes: FAIL", "whatsNew.en")
	w(".sza-canon.json", `{"site":{"pages":["release-notes.html"]}}`)
	if err := os.Remove(filepath.Join(dir, "landing.html")); err != nil {
		t.Fatal(err)
	}

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
	var src struct {
		Releases []struct {
			Version string `json:"version"`
		} `json:"releases"`
	}
	if err := json.Unmarshal([]byte(readRepoFile(t, "docs", "release-notes.json")), &src); err != nil {
		t.Fatal(err)
	}
	if len(src.Releases) == 0 {
		t.Fatal("docs/release-notes.json holds no release record")
	}
	page := readRepoFile(t, "release-notes.html")
	for _, r := range src.Releases {
		id := `id="v` + strings.ReplaceAll(r.Version, ".", "-") + `"`
		if !strings.Contains(page, id) {
			t.Errorf("release-notes.html lacks the section %s of docs/release-notes.json; run scripts/release-notes.ps1 -Render", id)
		}
	}
	if got := strings.Count(page, `<section class="release"`); got != len(src.Releases) {
		t.Errorf("release-notes.html has %d release sections, the source has %d records; run scripts/release-notes.ps1 -Render", got, len(src.Releases))
	}
}
