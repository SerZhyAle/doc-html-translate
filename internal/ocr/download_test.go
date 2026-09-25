package ocr

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// packFixture is the body the test server serves for rus, with its digest pinned in place of the
// real one - the download path is under test, not the upstream bytes.
var packFixture = []byte(strings.Repeat("fake traineddata ", 4096))

// dataDirs points both tessdata layers at fresh temp folders for one test.
func dataDirs(t *testing.T) (user, bundled string) {
	t.Helper()
	root := t.TempDir()
	user, bundled = filepath.Join(root, "user", "tessdata"), filepath.Join(root, "exe", "tessdata")
	savedUser, savedBundled := userDataDir, bundledDataDir
	userDataDir = func() string { return user }
	bundledDataDir = func() string { return bundled }
	t.Cleanup(func() { userDataDir, bundledDataDir = savedUser, savedBundled })
	return user, bundled
}

// packServer serves body for every request and pins rus to fixture's digest. It returns a counter
// of the requests it saw.
func packServer(t *testing.T, body []byte, handler http.HandlerFunc) *atomic.Int32 {
	t.Helper()
	var hits atomic.Int32
	if handler == nil {
		handler = func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(body) }
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		handler(w, r)
	}))
	t.Cleanup(srv.Close)

	sum := sha256.Sum256(packFixture)
	savedBase, savedClient, savedDigests := downloadBase, downloadClient, packDigests
	downloadBase, downloadClient = srv.URL, srv.Client()
	packDigests = map[string]packDigest{"rus": {int64(len(packFixture)), hex.EncodeToString(sum[:])}}
	t.Cleanup(func() { downloadBase, downloadClient, packDigests = savedBase, savedClient, savedDigests })
	return &hits
}

// listTree names every file under root, so a test can say "nothing was written" exactly.
func listTree(t *testing.T, root string) []string {
	t.Helper()
	var out []string
	_ = filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			out = append(out, p)
		}
		return nil
	})
	return out
}

func TestEveryCataloguePackHasADigest(t *testing.T) {
	for _, l := range Available {
		d, ok := packDigests[l.Code]
		if !ok || d.size <= 0 || len(d.sha256) != 64 {
			t.Errorf("%s: no usable pinned digest (%+v)", l.Code, d)
		}
	}
}

// Done criterion 1: the code is a URL path and a file name, so a code outside the catalogue must
// be refused before anything reaches the network or the disk.
func TestDownloadRefusesTraversalCode(t *testing.T) {
	user, _ := dataDirs(t)
	hits := packServer(t, packFixture, nil)
	root := filepath.Dir(filepath.Dir(user))

	for _, code := range []string{"../../x", "..\\..\\x", "rus/../../x", "x", ""} {
		err := Download(code)
		if !errors.Is(err, ErrUnknownLang) {
			t.Errorf("Download(%q) = %v, want ErrUnknownLang", code, err)
		}
	}
	if hits.Load() != 0 {
		t.Errorf("a refused code reached the server %d times", hits.Load())
	}
	if files := listTree(t, root); len(files) != 0 {
		t.Errorf("a refused code wrote files: %v", files)
	}
}

// Done criterion 2: two simultaneous requests for one pack end with one valid pack and no temp
// file; the second waits for the first and finds the pack installed.
func TestConcurrentDownloadsInstallOnePack(t *testing.T) {
	user, _ := dataDirs(t)
	hits := packServer(t, packFixture, func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(50 * time.Millisecond) // keep the first transfer open while the second arrives
		_, _ = w.Write(packFixture)
	})

	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i := range errs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs[i] = Download("rus")
		}()
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Errorf("download %d: %v", i, err)
		}
	}
	if err := verifyPack(langFile(user, "rus"), packDigests["rus"]); err != nil {
		t.Errorf("installed pack does not verify: %v", err)
	}
	if files := listTree(t, user); len(files) != 1 {
		t.Errorf("want exactly the pack in the folder, got %v", files)
	}
	if hits.Load() != 1 {
		t.Errorf("server hit %d times, want 1 - the second request should find the pack installed", hits.Load())
	}
}

func TestDownloadRefusesChecksumMismatch(t *testing.T) {
	user, _ := dataDirs(t)
	altered := append([]byte(nil), packFixture...)
	altered[100] ^= 0xff
	packServer(t, altered, nil)

	if err := Download("rus"); !errors.Is(err, ErrPackMismatch) {
		t.Fatalf("Download = %v, want ErrPackMismatch", err)
	}
	if files := listTree(t, user); len(files) != 0 {
		t.Errorf("a refused pack left files behind: %v", files)
	}
}

func TestDownloadRefusesOversizeBody(t *testing.T) {
	for name, chunked := range map[string]bool{"with length": false, "chunked": true} {
		t.Run(name, func(t *testing.T) {
			user, _ := dataDirs(t)
			big := append(append([]byte(nil), packFixture...), make([]byte, 1<<20)...)
			packServer(t, big, func(w http.ResponseWriter, _ *http.Request) {
				if chunked {
					w.(http.Flusher).Flush() // no Content-Length: the bound has to hold while reading
				}
				_, _ = w.Write(big)
			})
			if err := Download("rus"); !errors.Is(err, ErrPackMismatch) {
				t.Fatalf("Download = %v, want ErrPackMismatch", err)
			}
			if files := listTree(t, user); len(files) != 0 {
				t.Errorf("an oversize body left files behind: %v", files)
			}
		})
	}
}

// A pack another process finished installing is accepted without a second transfer.
func TestDownloadAcceptsAnInstalledVerifiedPack(t *testing.T) {
	user, _ := dataDirs(t)
	hits := packServer(t, packFixture, nil)
	if err := os.MkdirAll(user, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(langFile(user, "rus"), packFixture, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Download("rus"); err != nil {
		t.Fatal(err)
	}
	if hits.Load() != 0 {
		t.Errorf("an installed, verified pack was downloaded again")
	}
}

func TestDownloadRemovesOnlyStaleTemps(t *testing.T) {
	user, _ := dataDirs(t)
	packServer(t, packFixture, nil)
	if err := os.MkdirAll(user, 0o755); err != nil {
		t.Fatal(err)
	}
	stale := filepath.Join(user, "deu.traineddata-123.tmp")
	live := filepath.Join(user, "deu.traineddata-456.tmp")
	for _, p := range []string{stale, live} {
		if err := os.WriteFile(p, []byte("partial"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	old := time.Now().Add(-2 * staleTmpAge)
	if err := os.Chtimes(stale, old, old); err != nil {
		t.Fatal(err)
	}
	if err := Download("rus"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Error("a day-old temp file was not cleaned up")
	}
	if _, err := os.Stat(live); err != nil {
		t.Error("a young temp file - possibly another process's transfer - was removed")
	}
}

// The per-user folder is looked up first and the bundled one still counts; when the requested
// languages are split across the two, Tesseract gets one folder that holds both.
func TestLayeredDataDirs(t *testing.T) {
	user, bundled := dataDirs(t)
	if err := os.MkdirAll(bundled, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(langFile(bundled, "eng"), []byte("eng data"), 0o644); err != nil {
		t.Fatal(err)
	}

	if got := DataDirs(); len(got) != 2 || got[0] != user || got[1] != bundled {
		t.Errorf("DataDirs = %v, want [user bundled]", got)
	}
	// Only bundled data: an existing install keeps working from next to the exe, nothing copied.
	if got := DataDir(); got != bundled {
		t.Errorf("DataDir with only bundled data = %q, want %q", got, bundled)
	}
	if _, err := os.Stat(user); !os.IsNotExist(err) {
		t.Error("the per-user folder was created although nothing needed it")
	}

	if err := os.MkdirAll(user, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(langFile(user, "rus"), []byte("rus data"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(Installed(), ","); got != "eng,rus" {
		t.Errorf("Installed = %q, want eng,rus", got)
	}
	if !IsInstalled("eng") || !IsInstalled("rus") || IsInstalled("deu") {
		t.Error("IsInstalled does not see both layers")
	}
	dir := DataDir()
	if dir != user || !hasLangFile(dir, "rus+eng") {
		t.Errorf("DataDir = %q, want the per-user folder holding rus+eng", dir)
	}
	if files := listTree(t, user); len(files) != 2 {
		t.Errorf("staging left extra files: %v", files)
	}
}

func TestDownloadErrorsAreLocalized(t *testing.T) {
	err := CheckLang("xx")
	if got := ErrorText(err, "ru"); !strings.Contains(got, "не язык OCR") {
		t.Errorf("ru text = %q", got)
	}
	if got := ErrorText(err, "en"); !strings.Contains(got, `"xx" is not an OCR language`) || !strings.Contains(got, "rus") {
		t.Errorf("en text = %q", got)
	}
}
