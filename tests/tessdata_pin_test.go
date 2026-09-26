package tests

// The bundled English OCR data is pinned in three places: scripts/lib/tessdata.ps1 (every desktop
// package), internal/ocr/download.go (the in-app download) and extension/build.mjs (the extension's
// vendored copy). Ticket 38: the desktop scripts once fetched raw/main unpinned and shipped a package
// without the data when that failed. These tests hold the pins together and drive the helper through
// each refusal it documents, against a local server - never the network.

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
)

func TestEngTessdataPinAgrees(t *testing.T) {
	lib := readRepoFile(t, "scripts", "lib", "tessdata.ps1")
	psSize := regexp.MustCompile(`\$script:EngTessdataSize\s*=\s*(\d+)`).FindStringSubmatch(lib)
	psSha := regexp.MustCompile(`\$script:EngTessdataSha256\s*=\s*"([0-9a-f]{64})"`).FindStringSubmatch(lib)
	psURL := regexp.MustCompile(`\$script:EngTessdataUrl\s*=\s*"([^"]+)"`).FindStringSubmatch(lib)
	if psSize == nil || psSha == nil || psURL == nil {
		t.Fatal("scripts/lib/tessdata.ps1: pin variables not found")
	}

	goSrc := readRepoFile(t, "internal", "ocr", "download.go")
	goEng := regexp.MustCompile(`"eng":\s*\{(\d+),\s*"([0-9a-f]{64})"\}`).FindStringSubmatch(goSrc)
	if goEng == nil {
		t.Fatal(`internal/ocr/download.go: packDigests["eng"] not found`)
	}
	if psSize[1] != goEng[1] || psSha[1] != goEng[2] {
		t.Errorf("tessdata.ps1 pins %s/%s, download.go pins %s/%s", psSize[1], psSha[1], goEng[1], goEng[2])
	}

	mjs := readRepoFile(t, "extension", "build.mjs")
	mjsSha := regexp.MustCompile(`ENG_TRAINEDDATA_SHA256\s*=\s*"([0-9a-f]{64})"`).FindStringSubmatch(mjs)
	if mjsSha == nil || mjsSha[1] != psSha[1] {
		t.Errorf("extension/build.mjs ENG_TRAINEDDATA_SHA256 = %v, tessdata.ps1 pins %s", mjsSha, psSha[1])
	}

	cdn := regexp.MustCompile(`const cdnBase = "([^"]+)"`).FindStringSubmatch(readRepoFile(t, "internal", "ocr", "tessdata.go"))
	if cdn == nil || psURL[1] != cdn[1]+"/eng.traineddata" {
		t.Errorf("tessdata.ps1 URL %s is not cdnBase %v + /eng.traineddata", psURL[1], cdn)
	}

	// No packaging path may fetch the data on its own again.
	for _, f := range [][]string{
		{"scripts", "build.ps1"}, {"scripts", "build-ui.ps1"}, {"scripts", "build-installer.ps1"},
		{"msix", "build-msix.ps1"}, {".github", "workflows", "release.yml"},
	} {
		src := readRepoFile(t, f...)
		if strings.Contains(src, "tessdata_fast/raw/") {
			t.Errorf("%s fetches tessdata directly; use Install-EngTessdata from scripts/lib/tessdata.ps1", filepath.Join(f...))
		}
		if !strings.Contains(src, "Install-EngTessdata") {
			t.Errorf("%s does not provision the bundled English OCR data through Install-EngTessdata", filepath.Join(f...))
		}
	}
}

// pinnedEngBytes returns a local copy of the pinned pinned file when one is at hand (the extension's
// vendored copy or a previous build's), or nil.
func pinnedEngBytes(t *testing.T) []byte {
	t.Helper()
	lib := readRepoFile(t, "scripts", "lib", "tessdata.ps1")
	want := regexp.MustCompile(`\$script:EngTessdataSha256\s*=\s*"([0-9a-f]{64})"`).FindStringSubmatch(lib)[1]
	for _, p := range [][]string{
		{"extension", "vendor", "tesseract", "lang", "eng.traineddata"},
		{"build", "tessdata", "eng.traineddata"},
	} {
		b, err := os.ReadFile(repoPath(t, p...))
		if err != nil {
			continue
		}
		sum := sha256.Sum256(b)
		if hex.EncodeToString(sum[:]) == want {
			return b
		}
	}
	return nil
}

func TestInstallEngTessdataOutcomes(t *testing.T) {
	pwsh := findPwsh(t)
	dir := t.TempDir()
	driver := filepath.Join(dir, "drive.ps1")
	writeFile(t, driver, `param($Lib, $Dest, $Vendored, $Cache, $Url)
$ErrorActionPreference = "Stop"
. $Lib
Install-EngTessdata -DestDir $Dest -Vendored $Vendored -CacheDir $Cache -Url $Url -TimeoutSec 20
Write-Host "PROVISIONED"
`)
	lib := repoPath(t, "scripts", "lib", "tessdata.ps1")

	var body []byte
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if body == nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Length", strconv.Itoa(len(body)))
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	run := func(what, dest, vendored, cache string, wantCode int, wantFragment string) {
		t.Helper()
		code, _, out := runScript(t, pwsh, dir, driver, "-Lib", lib, "-Dest", dest, "-Vendored", vendored,
			"-Cache", cache, "-Url", srv.URL+"/eng.traineddata")
		if code != wantCode || !strings.Contains(out, wantFragment) {
			t.Fatalf("%s: exit %d, want %d with %q\n%s", what, code, wantCode, wantFragment, out)
		}
	}
	noVendored := filepath.Join(dir, "no-vendored", "eng.traineddata")

	// The server has nothing: the build fails rather than shipping without the data.
	run("download fails", filepath.Join(dir, "d1"), noVendored, filepath.Join(dir, "c1"), 1, "could not download")
	if _, err := os.Stat(filepath.Join(dir, "d1", "tessdata", "eng.traineddata")); err == nil {
		t.Fatal("download fails: a file was left in tessdata/")
	}

	// The server serves other bytes: refused by the digest, and nothing is cached.
	body = []byte("not the pinned model")
	run("tampered download", filepath.Join(dir, "d2"), noVendored, filepath.Join(dir, "c2"), 1, "does not match the pinned digest")
	if _, err := os.Stat(filepath.Join(dir, "c2", "eng.traineddata")); err == nil {
		t.Fatal("tampered download: the bad file reached the cache")
	}

	// A tampered vendored copy is an error, not something to route around.
	badVendored := filepath.Join(dir, "vendor", "eng.traineddata")
	writeFile(t, badVendored, "tampered")
	run("tampered vendored", filepath.Join(dir, "d3"), badVendored, filepath.Join(dir, "c3"), 1, "does not match the pinned tessdata_fast 4.0.0 digest")

	pinned := pinnedEngBytes(t)
	if pinned == nil {
		t.Log("no local copy of the pinned eng.traineddata; the success paths are not driven")
		return
	}

	// A tampered cache is discarded and fetched again; the result is the pinned file.
	body = pinned
	cache := filepath.Join(dir, "c4")
	writeFile(t, filepath.Join(cache, "eng.traineddata"), "tampered cache")
	hits.Store(0)
	run("tampered cache", filepath.Join(dir, "d4"), noVendored, cache, 0, "PROVISIONED")
	if n := hits.Load(); n != 1 {
		t.Fatalf("tampered cache: %d downloads, want 1", n)
	}
	got, _ := os.ReadFile(filepath.Join(dir, "d4", "tessdata", "eng.traineddata"))
	if string(got) != string(pinned) {
		t.Fatal("tampered cache: the provisioned file is not the pinned model")
	}

	// A good cache is reused without the network; a tampered destination is replaced.
	dest := filepath.Join(dir, "d5")
	writeFile(t, filepath.Join(dest, "tessdata", "eng.traineddata"), "stale")
	hits.Store(0)
	run("tampered destination", dest, noVendored, cache, 0, "replacing it")
	if n := hits.Load(); n != 0 {
		t.Fatalf("verified cache: %d downloads, want 0", n)
	}
	got, _ = os.ReadFile(filepath.Join(dest, "tessdata", "eng.traineddata"))
	if string(got) != string(pinned) {
		t.Fatal("tampered destination: not replaced by the pinned model")
	}
}
