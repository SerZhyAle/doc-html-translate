package tests

// Drives the three checks ticket 31 added - scripts/doc-registry.ps1, scripts/security-posture.ps1
// and scripts/contract-gate.ps1 - to every outcome they document, each over a small scratch tree
// built here, never over the repository itself. A rule no test turns red is a rule nobody knows
// still works (CHECK-VERDICT section 7, rung 1).

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// scratchRepo makes a git repository in a temp directory holding copies of the named scripts (paths
// relative to scripts/), and returns its root plus a git runner.
func scratchRepo(t *testing.T, scripts ...string) (string, func(args ...string)) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	dir := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-c", "user.name=t", "-c", "user.email=t@t", "-c", "core.autocrlf=false"}, args...)...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	for _, s := range scripts {
		writeFile(t, filepath.Join(dir, "scripts", filepath.FromSlash(s)), readRepoFile(t, append([]string{"scripts"}, strings.Split(s, "/")...)...))
	}
	git("init", "-q")
	return dir, git
}

// expectRun runs a script and checks the exit code, a prefix of the verdict line, and optionally a
// fragment of the output that names the finding.
func expectRun(t *testing.T, pwsh, dir, script, what string, wantCode int, wantPrefix, wantFragment string, args ...string) {
	t.Helper()
	code, last, out := runScript(t, pwsh, dir, filepath.Join(dir, "scripts", script), args...)
	if code != wantCode || !strings.HasPrefix(last, wantPrefix) {
		t.Fatalf("%s: exit %d, last line %q; want exit %d, prefix %q\n%s", what, code, last, wantCode, wantPrefix, out)
	}
	if wantFragment != "" && !strings.Contains(out, wantFragment) {
		t.Fatalf("%s: the output does not name %q\n%s", what, wantFragment, out)
	}
}

const scratchPage = `<!doctype html>
<html lang="en"><head>
<link rel="canonical" href="https://example.test/site/">
<link rel="alternate" hreflang="en" href="https://example.test/site/">
<link rel="alternate" hreflang="ru" href="https://example.test/site/?l=ru">
<link rel="alternate" hreflang="x-default" href="https://example.test/site/">
<meta property="og:title" content="T"><meta property="og:description" content="D">
<meta property="og:image" content="https://example.test/site/shot.png"><meta property="og:url" content="https://example.test/site/">
<meta name="twitter:card" content="summary_large_image">
<script type="application/ld+json">{"@type":"SoftwareApplication"}</script>
<title>T</title>
<meta name="description" content="D">
</head><body><h1>T</h1></body></html>
`

func TestDocRegistryOutcomes(t *testing.T) {
	pwsh := findPwsh(t)
	dir, git := scratchRepo(t, "doc-registry.ps1", "lib/verdict.ps1", "lib/docregistry.ps1")
	w := func(rel, content string) { writeFile(t, filepath.Join(dir, filepath.FromSlash(rel)), content) }
	w(".sza-canon.json", `{"docRegistryShape":1,"docRegistryFile":"docs/DOCUMENT_REGISTRY.jsonl"}`)
	w("robots.txt", "User-agent: *\nAllow: /\n\nSitemap: https://example.test/site/sitemap.xml\n")
	w("index.html", scratchPage)
	w("README.md", "# scratch\n")
	w("docs/DOCUMENT_REGISTRY.jsonl", strings.Join([]string{
		`{"id":"site","title":"Site","category":"site","audience":"user","paths":["index.html"],"published":true,"indexable":true,"url":"/","languages":["en","ru"],"localized_urls":{"en":"/","ru":"/?l=ru"},"product_areas":["site"],"update_triggers":["site-page"],"generated":false}`,
		`{"id":"crawl","title":"Crawl files","category":"site","audience":"crawler","paths":["robots.txt","sitemap.xml"],"published":true,"indexable":false,"product_areas":["site"],"update_triggers":["site-page"],"generated":false}`,
		`{"id":"readme","title":"Readme","category":"public","audience":"user","paths":["README.md"],"published":false,"indexable":false,"product_areas":["product"],"update_triggers":["user-feature"],"generated":false,"notes":"the repository front page, not a site page"}`,
		`{"id":"registry","title":"Registry","category":"engineering","audience":"developer","paths":["docs/DOCUMENT_REGISTRY.jsonl"],"published":false,"indexable":false,"product_areas":["documentation"],"update_triggers":["document-added"],"generated":false,"notes":"tooling input kept in the repository"}`,
	}, "\n")+"\n")
	w("sitemap.xml", "stale\n")
	git("add", "-A")
	git("commit", "-q", "-m", "init")
	run := func(what string, code int, prefix, fragment string, args ...string) {
		t.Helper()
		expectRun(t, pwsh, dir, "doc-registry.ps1", what, code, prefix, fragment, args...)
	}

	run("a hand-kept sitemap", 1, "doc-registry: FAIL", "sitemap.xml differs")
	run("-Generate writes the sitemap, then the check passes", 0, "doc-registry: PASS (4 record(s)", "", "-Generate")
	sitemap := readScratch(t, dir, "sitemap.xml")
	for _, want := range []string{"<loc>https://example.test/site/</loc>", "<loc>https://example.test/site/?l=ru</loc>", `hreflang="x-default"`} {
		if !strings.Contains(sitemap, want) {
			t.Fatalf("the generated sitemap lacks %q:\n%s", want, sitemap)
		}
	}
	run("the generated sitemap is current", 0, "doc-registry: PASS", "")

	w("notes/todo.md", "# not registered\n")
	run("a new document in no record", 1, "doc-registry: FAIL", "notes/todo.md is in no record")
	w("configs/doc-registry-exclusions.jsonl", `{"path":"notes/*.md","reason":"scratch notes kept for this test only"}`+"\n")
	run("an exclusion with a reason covers it", 0, "doc-registry: PASS", "")
	w("configs/doc-registry-exclusions.jsonl", `{"path":"notes/*.md","reason":"internal"}`+"\n")
	run("an exclusion reason under four words", 1, "doc-registry: FAIL", "under four words")
	w("configs/doc-registry-exclusions.jsonl", `{"path":"notes/*.md","reason":"scratch notes kept for this test only"}`+"\n")

	w("page2.html", scratchPage)
	run("a publishable page nobody decided about", 1, "doc-registry: FAIL", "page2.html is in no record")
	if err := os.Remove(filepath.Join(dir, "page2.html")); err != nil {
		t.Fatal(err)
	}

	w("index.html", strings.Replace(scratchPage, `<meta property="og:image" content="https://example.test/site/shot.png">`, "", 1))
	run("an announced page without og:image", 1, "doc-registry: FAIL", "'og:image'")
	w("index.html", scratchPage)

	w(".sza-canon.json", `{"ledgerShape":2}`)
	run("the stamp does not declare the record shape", 1, "doc-registry: FAIL", "docRegistryShape")
	w(".sza-canon.json", `{"docRegistryShape":1,"docRegistryFile":"docs/DOCUMENT_REGISTRY.jsonl"}`)
	run("back to clean", 0, "doc-registry: PASS", "")

	outside := t.TempDir()
	writeFile(t, filepath.Join(outside, "scripts", "doc-registry.ps1"), readRepoFile(t, "scripts", "doc-registry.ps1"))
	writeFile(t, filepath.Join(outside, "scripts", "lib", "verdict.ps1"), readRepoFile(t, "scripts", "lib", "verdict.ps1"))
	writeFile(t, filepath.Join(outside, "scripts", "lib", "docregistry.ps1"), readRepoFile(t, "scripts", "lib", "docregistry.ps1"))
	expectRun(t, pwsh, outside, "doc-registry.ps1", "not a git checkout", 2, "doc-registry: COULD NOT VERIFY", "")
}

func readScratch(t *testing.T, dir, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return strings.ReplaceAll(string(b), "\r\n", "\n")
}

const scratchInventory = `{
  "shape": 1,
  "lastReconciled": "2026-01-01",
  "permissions": [
    {"id": "ext-storage", "edition": "extension", "declares": "permission:storage", "declaredIn": "extension/manifest.json",
     "consumers": ["the settings"], "evidence": [{"file": "extension/src/app.js", "contains": "chrome.storage.local.set"}],
     "shownAtRequest": false, "shownWhere": "the browser shows no prompt for it",
     "public": {"en": "**storage** - your settings, *here*.", "ru": "**storage** - настройки.", "uk": "**storage** - налаштування."},
     "storeLabel": "storage", "storeJustification": "to remember the user's settings on this device and nothing else at all"},
    {"id": "app-rft", "edition": "app", "declares": "msix:runFullTrust", "declaredIn": "msix/AppxManifest.xml",
     "consumers": ["the whole app"], "evidence": [{"file": "msix/AppxManifest.xml", "contains": "runFullTrust"}],
     "shownAtRequest": false, "shownWhere": "certification only", "public": null,
     "publicNote": "a packaging capability, not user data",
     "storeLabel": "runFullTrust", "storeJustification": "a full-trust desktop app needs runFullTrust to run as a normal desktop process"}
  ],
  "networkSurfaces": [
    {"id": "net-doc", "edition": "extension", "kind": "outbound", "surface": "a fetch of the document", "defaultOn": true,
     "turnedOnBy": "opening a document", "lifetime": "the tab", "leaves": "a request to the site that serves it",
     "evidence": [{"file": "extension/src/app.js", "contains": "fetch(url)"}],
     "public": {"en": "**The document** - fetched from ` + "`its site`" + `.", "ru": "**Документ** - загружается.", "uk": "**Документ** - завантажується."}}
  ],
  "networkCallSites": {"roots": [{"glob": "extension/src/*.js", "skip": null, "patterns": ["\\bfetch\\("]}], "notNetwork": []},
  "telemetry": {
    "claim": "No telemetry.",
    "statedIn": [{"file": "page.html", "contains": "no telemetry"}],
    "dependencySets": [{"file": "go.sum", "kind": "go-sum"}, {"file": "extension/package-lock.json", "kind": "npm-lock"}],
    "denylist": ["(^|/)getsentry(/|$)", "^@sentry(/|$)"]
  },
  "renders": [
    {"target": "page.html", "block": "blk", "format": "html-lists", "rows": ["ext-storage", "net-doc"]},
    {"target": "msix.md", "block": "j", "format": "md-fence", "rows": ["app-rft"]},
    {"target": "docs/SECURITY_POSTURE.md", "block": null, "format": "posture-doc", "rows": []}
  ]
}
`

func TestSecurityPostureOutcomes(t *testing.T) {
	pwsh := findPwsh(t)
	dir, git := scratchRepo(t, "security-posture.ps1", "lib/verdict.ps1", "lib/docregistry.ps1")
	w := func(rel, content string) { writeFile(t, filepath.Join(dir, filepath.FromSlash(rel)), content) }
	manifest := `{"manifest_version": 3, "permissions": ["storage"]}`
	app := "chrome.storage.local.set({a: 1});\nconst r = await fetch(url);\n"
	page := "<p>We ship no telemetry.</p>\n    <!-- security-posture:begin blk (rendered) -->\n    <!-- security-posture:end blk -->\n"
	gosum := "golang.org/x/text v0.3.0 h1:abc=\ngolang.org/x/text v0.3.0/go.mod h1:def=\n"
	w("docs/security-posture.json", scratchInventory)
	w("extension/manifest.json", manifest)
	w("extension/src/app.js", app)
	w("msix/AppxManifest.xml", `<Capabilities><rescap:Capability Name="runFullTrust" /></Capabilities>`)
	w("go.sum", gosum)
	w("extension/package-lock.json", `{"packages": {"": {}, "node_modules/marked": {}}}`)
	w("page.html", page)
	w("msix.md", "### justification\n<!-- security-posture:begin j -->\n<!-- security-posture:end j -->\n")
	git("add", "-A")
	git("commit", "-q", "-m", "init")
	run := func(what string, code int, prefix, fragment string, args ...string) {
		t.Helper()
		expectRun(t, pwsh, dir, "security-posture.ps1", what, code, prefix, fragment, args...)
	}

	run("empty blocks are not the render", 1, "security-posture: FAIL", "is not the render")
	run("-Render fills every form, then the check passes", 0, "security-posture: PASS (2 permission row(s), 1 network surface(s)", "", "-Render")
	rendered := readScratch(t, dir, "page.html")
	for _, want := range []string{`<ul data-l="ua">`, "<strong>storage</strong> - your settings, <em>here</em>.", "<code>its site</code>"} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("page.html lacks %q:\n%s", want, rendered)
		}
	}
	if !strings.Contains(readScratch(t, dir, "msix.md"), "```\na full-trust desktop app needs runFullTrust") {
		t.Fatalf("msix.md was not rendered:\n%s", readScratch(t, dir, "msix.md"))
	}

	w("extension/manifest.json", `{"manifest_version": 3, "permissions": ["storage", "tabs"]}`)
	run("a declared permission without a row", 1, "security-posture: FAIL", "permission:tabs is declared in a manifest and has no row")
	w("extension/manifest.json", `{"manifest_version": 3, "permissions": []}`)
	run("a row nothing declares", 1, "security-posture: FAIL", "which no manifest declares")
	w("extension/manifest.json", manifest)

	w("extension/src/beacon.js", "fetch('https://collector.example/');\n")
	run("a network call site no row covers", 1, "security-posture: FAIL", "extension/src/beacon.js contains a network primitive")
	if err := os.Remove(filepath.Join(dir, "extension", "src", "beacon.js")); err != nil {
		t.Fatal(err)
	}

	w("go.sum", gosum+"github.com/getsentry/sentry-go v0.20.0 h1:xyz=\n")
	run("a crash-reporting dependency against a no-telemetry claim", 1, "security-posture: FAIL", "dependency 'github.com/getsentry/sentry-go'")
	w("go.sum", gosum)

	w("extension/src/app.js", "const r = await fetch(url);\n")
	run("a consumer the code no longer has", 1, "security-posture: FAIL", "no longer contains 'chrome.storage.local.set'")
	w("extension/src/app.js", app)

	edited := strings.Replace(readScratch(t, dir, "page.html"), "your settings", "your settings and more", 1)
	w("page.html", edited)
	run("a rendered block edited by hand", 1, "security-posture: FAIL", "page.html block 'blk' is not the render")
	run("-Render restores it", 0, "security-posture: PASS", "", "-Render")

	if err := os.Remove(filepath.Join(dir, "docs", "security-posture.json")); err != nil {
		t.Fatal(err)
	}
	run("no inventory", 2, "security-posture: COULD NOT VERIFY", "")
}

func TestContractGateOutcomes(t *testing.T) {
	pwsh := findPwsh(t)
	dir, git := scratchRepo(t, "contract-gate.ps1", "lib/verdict.ps1")
	catalog := t.TempDir()
	w := func(rel, content string) { writeFile(t, filepath.Join(dir, filepath.FromSlash(rel)), content) }
	today := time.Now().Format("2006-01-02")
	pointer := func(version string) string {
		return "# Pointer: FOO-FORMAT\n\n- **Id:** `FOO-FORMAT`\n- **Version:** " + version + " (draft)\n- **Role:** consumer - reads it\n"
	}
	registry := func(catalogVersion, adoption, exception string) string {
		return "# Registry\n\n## 1. Contracts\n\n| Id | Folder | Version | Status | Owner | Wire carrier | Conformance artifacts |\n| --- | --- | --- | --- | --- | --- | --- |\n" +
			"| `FOO-FORMAT` | [`foo/`](../foo/README.md) | " + catalogVersion + " | active | other | none | none yet |\n\n" +
			"## 2. Adoption\n\n| Contract | Product | Role | Implements | Reads | Verified | Notes |\n| --- | --- | --- | --- | --- | --- | --- |\n" + adoption + "\n" +
			"## 3. Exceptions\n\n| Contract | Product | Deviation | Reason | Until |\n| --- | --- | --- | --- | --- |\n" + exception + "\n## 4. Planned and reserved\n"
	}
	row := func(version, verified string) string {
		return "| `FOO-FORMAT` | testprod | C | " + version + " | " + version + " | " + verified + " | read \\| checked |\n"
	}
	setCatalog := func(content string) { writeFile(t, filepath.Join(catalog, "_meta", "REGISTRY.md"), content) }
	w("docs/contracts/README.md", "# pointers\n")
	w("docs/contracts/FOO-FORMAT.md", pointer("1.2"))
	w("AGENTS.md", "The shared contracts catalog is at `"+filepath.ToSlash(catalog)+"` on this machine.\n")
	setCatalog(registry("1.2", row("1.2", today), ""))
	git("add", "-A")
	git("commit", "-q", "-m", "init")
	run := func(what string, code int, prefix, fragment string, args ...string) {
		t.Helper()
		expectRun(t, pwsh, dir, "contract-gate.ps1", what, code, prefix, fragment, append([]string{"-Product", "testprod"}, args...)...)
	}

	run("the catalog is where AGENTS.md says, and every row holds", 0, "contract-gate: PASS (1 contract(s)", "verdict = PASS")
	run("an unreachable catalog is never a pass", 2, "contract-gate: COULD NOT VERIFY", "verdict = UNVERIFIED", "-Catalog", filepath.Join(catalog, "missing"))

	w("docs/contracts/FOO-FORMAT.md", pointer("1.3"))
	run("the repository ahead of the catalog", 1, "contract-gate: FAIL", "the contract change goes into the catalog first")
	w("docs/contracts/FOO-FORMAT.md", pointer("1.1"))
	setCatalog(registry("1.2", row("1.1", today), ""))
	run("behind within one MAJOR", 3, "contract-gate: PASS WITH ADVISORIES", "verdict = WARN")
	setCatalog(registry("3.0", row("1.1", today), ""))
	run("two MAJOR versions behind", 1, "contract-gate: FAIL", "2 MAJOR versions behind")

	w("docs/contracts/FOO-FORMAT.md", pointer("1.2"))
	setCatalog(registry("1.2", "", ""))
	run("no adoption row", 1, "contract-gate: FAIL", "no adoption row for testprod")
	setCatalog(registry("1.2", row("1.2", "pending"), ""))
	run("a row nobody verified", 1, "contract-gate: FAIL", "is not verified ('pending')")

	setCatalog(registry("1.2", row("-", today), ""))
	run("absent, with no exception", 1, "contract-gate: FAIL", "absent is not allowed")
	setCatalog(registry("1.2", row("-", today), "| `FOO-FORMAT` | testprod | not adopted yet | a dated reason | 2026-12-31 |\n"))
	run("absent, covered by an open exception", 3, "contract-gate: PASS WITH ADVISORIES", "covered by an open exception", "-Today", "2026-06-01")
	run("the exception has expired", 1, "contract-gate: FAIL", "expired on 2026-12-31", "-Today", "2027-01-15")

	setCatalog(registry("1.2", row("1.2", today), ""))
	w("docs/contracts/FOO-FORMAT.md", strings.Replace(pointer("1.2"), "consumer - reads it", "not applicable - never read here", 1))
	run("zero contracts checked is never a pass", 2, "contract-gate: COULD NOT VERIFY (0 contracts checked", "verdict = UNVERIFIED")
	w("docs/contracts/FOO-FORMAT.md", pointer("1.2"))

	setCatalog(registry("1.2", row("1.2", "2000-01-01"), ""))
	git("tag", "v26.0101.0000")
	run("verified before the last release", 1, "contract-gate: FAIL", "before the last release v26.0101.0000")
}
