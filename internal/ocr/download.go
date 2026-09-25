package ocr

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"doc-html-translate/internal/i18n"
)

// packDigest pins one catalogue pack: its exact size and SHA-256. The download URL is pinned to the
// 4.0.0 tag, so the bytes never legitimately change; anything else - a truncated transfer, a proxy
// rewriting the body, a mirror serving another build - is refused before it becomes visible.
type packDigest struct {
	size   int64
	sha256 string
}

// packDigests covers every code in Available (TestEveryCataloguePackHasADigest). The values were
// taken from tessdata_fast 4.0.0 as served by cdnBase (its raw.githubusercontent.com redirect
// target) on 2026-09-25. The extension loads a gzipped build of the same files from another host
// through tesseract.js, so it cannot share this table - see docs/PARITY.md "OCR".
var packDigests = map[string]packDigest{
	"eng":      {4113088, "7d4322bd2a7749724879683fc3912cb542f19906c83bcc1a52132556427170b2"},
	"rus":      {3861738, "e16e5e036cce1d9ec2b00063cf8b54472625b9e14d893a169e2b0dedeb4df225"},
	"ukr":      {3825102, "d59e53e2bded32f4445f124b4b00240fcac7e8044c003ab822ccb94f0b3db59b"},
	"jpn":      {2471260, "1f5de9236d2e85f5fdf4b3c500f2d4926f8d9449f28f5394472d9e8d83b91b4d"},
	"jpn_vert": {3037480, "bf1e2640954691797e2dc14f38533e601b59ee37958698ae0f0b81dc6f09c71b"},
	"deu":      {1525436, "19d219bbb6672c869d20a9636c6816a81eb9a71796cb93ebe0cb1530e2cdb22d"},
	"fra":      {1130365, "ced037562e8c80c13122dece28dd477d399af80911a28791a66a63ac1e3445ca"},
	"spa":      {2294433, "6f2e04d02774a18f01bed44b1111f2cd7f3ba7ac9dc4373cd3f898a40ea6b464"},
	"ita":      {2701314, "b8f89e1e785118dac4d51ae042c029a64edb5c3ee42ef73027a6d412748d8827"},
	"por":      {1982756, "c4932b937207a9514b7514d518b931a99938c02a28a5a5a553f8599ed58b7deb"},
	"pol":      {4765518, "c4476cdbc0e33d898d32345122b7be1cbf85ace15f920f06c7714756e1ef79b2"},
	"chi_sim":  {2469156, "a5fcb6f0db1e1d6d8522f39db4e848f05984669172e584e8d76b6b3141e1f730"},
	"kor":      {1677415, "6b85e11d9bbf07863b97b3523b1b112844c43e713df8b66418a081fd1060b3b2"},
}

// downloadBase and downloadClient are variables so a test can serve fixtures from httptest.
var (
	downloadBase   = cdnBase
	downloadClient = &http.Client{Timeout: 10 * time.Minute}
)

// staleTmpAge is how old a leftover temp file must be before a later download removes it. A day
// is far beyond any live transfer, so a file that old was left by a process that died mid-way.
const staleTmpAge = 24 * time.Hour

// installLocks holds one mutex per language code. Two requests for one pack in this process
// (a double click in the GUI) then run one after the other, and the second finds the pack
// installed; another process racing us is handled by the rename and the re-check after it.
var installLocks sync.Map

// ErrUnknownLang and ErrPackMismatch classify a refused download for callers and tests; the
// user-facing text is the localized message of the error that wraps them.
var (
	ErrUnknownLang  = errors.New("not a catalogue OCR language")
	ErrPackMismatch = errors.New("downloaded OCR data does not match the pinned digest")
)

// downloadError carries its message as an i18n key plus arguments, so the CLI renders it in the
// process language (Error) and the GUI in the language its page is shown in (ErrorText).
type downloadError struct {
	kind   error
	format string
	args   []any
}

func (e *downloadError) Error() string { return i18n.S(e.format, e.args...) }
func (e *downloadError) Unwrap() error { return e.kind }

// ErrorText renders err in the interface language lang when it came from this package's
// download path, and falls back to its plain message otherwise.
func ErrorText(err error, lang string) string {
	var de *downloadError
	if errors.As(err, &de) {
		return i18n.T(lang, de.format, de.args...)
	}
	return err.Error()
}

// CheckLang refuses a code outside the fixed catalogue. The code becomes both a URL path and a
// file name, so this gate - not the HTTP status - is what keeps "../../x" from writing outside
// the data folder. Every entry point calls it before anything touches the network or the disk.
func CheckLang(code string) error {
	for _, l := range Available {
		if l.Code == code {
			return nil
		}
	}
	codes := make([]string, 0, len(Available))
	for _, l := range Available {
		codes = append(codes, l.Code)
	}
	return &downloadError{kind: ErrUnknownLang,
		format: "%q is not an OCR language this app offers. Choose one of: %s",
		args:   []any{code, strings.Join(codes, " ")}}
}

// Download installs a catalogue language pack into the per-user tessdata folder. The body is
// streamed into a unique temp file, bounded to the pinned size, checked against the pinned
// SHA-256 and only then renamed into place, so a partial, oversized or altered file is never
// visible as installed and a failure leaves nothing behind.
func Download(code string) error {
	return DownloadContext(context.Background(), code, nil)
}

// Progress is told how many bytes of a pack have arrived and how many the pack has in all.
type Progress func(done, total int64)

// DownloadContext is Download that stops when ctx ends and reports its progress. A cancelled
// download leaves nothing behind, like any other failed one, and its error wraps ctx.Err().
func DownloadContext(ctx context.Context, code string, progress Progress) error {
	code = strings.TrimSpace(code)
	if err := CheckLang(code); err != nil {
		return err
	}
	want, ok := packDigests[code]
	if !ok {
		return fmt.Errorf("ocr: no pinned digest for %q", code)
	}
	dir := UserDataDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return &downloadError{kind: err, format: "Cannot write to the OCR language folder %s: %v", args: []any{dir, err}}
	}
	lock, _ := installLocks.LoadOrStore(code, &sync.Mutex{})
	mu := lock.(*sync.Mutex)
	mu.Lock()
	defer mu.Unlock()

	final := langFile(dir, code)
	if verifyPack(final, want) == nil {
		return nil
	}
	removeStaleTemps(dir)
	return fetchPack(ctx, code, dir, want, progress)
}

// fetchPack downloads one pack into dir through a verified temp file.
func fetchPack(ctx context.Context, code, dir string, want packDigest, progress Progress) error {
	label := LangLabel(code)
	failed := func(err error) error {
		return &downloadError{kind: err, format: "Could not download the %s language data: %v", args: []any{label, err}}
	}
	mismatch := &downloadError{kind: ErrPackMismatch,
		format: "The downloaded %s language data does not match the published file, so it was not installed. Try again later.",
		args:   []any{label}}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadBase+"/"+code+".traineddata", nil)
	if err != nil {
		return failed(err)
	}
	resp, err := downloadClient.Do(req)
	if err != nil {
		return failed(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return failed(fmt.Errorf("HTTP %d", resp.StatusCode))
	}
	if resp.ContentLength > want.size {
		return mismatch
	}

	tmp, err := os.CreateTemp(dir, code+".traineddata-*.tmp")
	if err != nil {
		return &downloadError{kind: err, format: "Cannot write to the OCR language folder %s: %v", args: []any{dir, err}}
	}
	keep := false
	defer func() {
		if !keep {
			_ = os.Remove(tmp.Name())
		}
	}()
	h := sha256.New()
	// One byte past the pinned size is enough to know the body is too long without reading it all.
	var sink io.Writer = io.MultiWriter(tmp, h)
	if progress != nil {
		sink = io.MultiWriter(sink, &progressWriter{total: want.size, report: progress})
	}
	n, err := io.Copy(sink, io.LimitReader(resp.Body, want.size+1))
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return failed(err)
	}
	if n != want.size || hex.EncodeToString(h.Sum(nil)) != want.sha256 {
		return mismatch
	}
	final := langFile(dir, code)
	if err := os.Rename(tmp.Name(), final); err != nil {
		// Windows refuses to replace a file another process has open; if that process was
		// installing the same pack, what it left in place is as good as ours.
		if verifyPack(final, want) == nil {
			return nil
		}
		return failed(err)
	}
	keep = true
	return nil
}

// progressWriter counts the bytes that pass through it and reports each step.
type progressWriter struct {
	done, total int64
	report      Progress
}

func (p *progressWriter) Write(b []byte) (int, error) {
	p.done += int64(len(b))
	p.report(p.done, p.total)
	return len(b), nil
}

// verifyPack reports whether path holds exactly the pinned pack.
func verifyPack(path string, want packDigest) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	st, err := f.Stat()
	if err != nil {
		return err
	}
	if st.Size() != want.size {
		return ErrPackMismatch
	}
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	if hex.EncodeToString(h.Sum(nil)) != want.sha256 {
		return ErrPackMismatch
	}
	return nil
}

// removeStaleTemps deletes download and staging temp files a dead process left in dir. Only old
// ones: a young file may be another process's live transfer.
func removeStaleTemps(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.Contains(name, ".traineddata") || !strings.HasSuffix(name, ".tmp") {
			continue
		}
		if st, err := e.Info(); err == nil && time.Since(st.ModTime()) > staleTmpAge {
			_ = os.Remove(filepath.Join(dir, name))
		}
	}
}
