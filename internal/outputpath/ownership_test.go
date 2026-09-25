package outputpath

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func mkdir(t *testing.T, p string) {
	t.Helper()
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatal(err)
	}
}

func write(t *testing.T, p, content string) {
	t.Helper()
	mkdir(t, filepath.Dir(p))
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestResolveFreshLocation(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "book.epub")
	got, err := Resolve(in, "")
	if err != nil {
		t.Fatal(err)
	}
	if got.Dir != filepath.Join(dir, "book") || got.State != StateAbsent {
		t.Fatalf("Resolve = %+v", got)
	}
}

// book.epub and book.pdf in one folder used to share "book/": one opened the other's
// content, or -force destroyed it.
func TestResolveSeparatesSameNamedDocuments(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "book")
	mkdir(t, out)
	if err := WriteMarker(out, filepath.Join(dir, "book.epub")); err != nil {
		t.Fatal(err)
	}

	own, _ := Resolve(filepath.Join(dir, "book.epub"), "")
	if own.Dir != out || own.State != StateOwned {
		t.Fatalf("epub: %+v", own)
	}
	other, _ := Resolve(filepath.Join(dir, "book.pdf"), "")
	if other.Dir != filepath.Join(dir, "book (pdf)") || other.Preferred != out {
		t.Fatalf("pdf: %+v", other)
	}
}

func TestResolveSkipsForeignFolder(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "Tolkien", "notes.txt"), "mine")
	got, _ := Resolve(filepath.Join(dir, "Tolkien.pdf"), "")
	if got.Dir != filepath.Join(dir, "Tolkien (pdf)") {
		t.Fatalf("foreign folder not skipped: %+v", got)
	}
}

func TestResolveKeepsCountingPastTakenSuffixes(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "book", "x"), "")
	write(t, filepath.Join(dir, "book (pdf)", "x"), "")
	got, _ := Resolve(filepath.Join(dir, "book.pdf"), "")
	if got.Dir != filepath.Join(dir, "book (pdf) 2") {
		t.Fatalf("got %+v", got)
	}
}

func TestResolveUsesEmptyFolder(t *testing.T) {
	dir := t.TempDir()
	mkdir(t, filepath.Join(dir, "book"))
	got, _ := Resolve(filepath.Join(dir, "book.pdf"), "")
	if got.Dir != filepath.Join(dir, "book") || got.State != StateEmpty {
		t.Fatalf("got %+v", got)
	}
}

// -folder C:\x with input C:\x\book\book.epub derives C:\x\book - the folder holding the
// source. Cleaning that up deleted the book itself.
func TestResolveNeverPicksFolderContainingInput(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "book", "book.epub")
	write(t, in, "x")
	got, _ := Resolve(in, dir)
	if got.Dir == filepath.Join(dir, "book") {
		t.Fatalf("picked the folder that contains the input: %+v", got)
	}
}

// A file without an extension derives an output path equal to itself.
func TestResolveNeverPicksInputPath(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "README")
	write(t, in, "x")
	got, _ := Resolve(in, "")
	if got.Dir == in {
		t.Fatalf("picked the input path itself")
	}
	if got.Dir != filepath.Join(dir, "README (file)") {
		t.Fatalf("got %+v", got)
	}
}

func TestInspectLegacyOutput(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "book")
	write(t, filepath.Join(out, "index.html"),
		`<html><head><style id="dht-single-css"></style></head><body><div class="page-header"><span class="nav-file" title="book.epub">book.epub</span></div></body></html>`)

	if st := Inspect(out, filepath.Join(dir, "book.epub")); st != StateLegacy {
		t.Fatalf("own legacy output: %v", st)
	}
	if st := Inspect(out, filepath.Join(dir, "book.pdf")); st != StateOwnedByOther {
		t.Fatalf("legacy output of book.epub claimed by book.pdf: %v", st)
	}
}

func TestInspectLegacyFollowsRedirectStub(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "book")
	write(t, filepath.Join(out, "index.html"), `<script>location.replace("OEBPS/index.html");</script>`)
	write(t, filepath.Join(out, "OEBPS", "index.html"), `<div class="dht-navbar"></div>`)
	if st := Inspect(out, filepath.Join(dir, "book.epub")); st != StateLegacy {
		t.Fatalf("got %v", st)
	}
}

func TestInspectPlainIndexIsForeign(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "site")
	write(t, filepath.Join(out, "index.html"), `<html><body>my site</body></html>`)
	if st := Inspect(out, filepath.Join(dir, "site.html")); st != StateForeign {
		t.Fatalf("got %v", st)
	}
}

func TestInspectCorruptMarkerIsForeign(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "book")
	write(t, filepath.Join(out, MarkerName), "{not json")
	if st := Inspect(out, filepath.Join(dir, "book.pdf")); st != StateForeign {
		t.Fatalf("got %v", st)
	}
}

func TestClearContentsKeepsNamedEntries(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "a", "b.html"), "")
	write(t, filepath.Join(dir, LockName), "1")
	if err := ClearContents(dir, LockName); err != nil {
		t.Fatal(err)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 || entries[0].Name() != LockName {
		t.Fatalf("left %v", entries)
	}
}

func TestLockIsExclusive(t *testing.T) {
	dir := t.TempDir()
	l, err := AcquireLock(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !Locked(dir) {
		t.Fatal("held lock not reported")
	}
	if _, err := AcquireLock(dir); err != ErrLocked {
		t.Fatalf("second acquire: %v", err)
	}
	l.Release()
	l2, err := AcquireLock(dir)
	if err != nil {
		t.Fatalf("re-acquire after release: %v", err)
	}
	l2.Release()
}

func TestLockLeftByDeadProcessIsCleared(t *testing.T) {
	dir := t.TempDir()
	// A pid far above any real one stands in for a crashed run.
	write(t, filepath.Join(dir, LockName), strconv.Itoa(1<<30)+"\n")
	if Locked(dir) {
		t.Fatal("dead owner's lock reported as live")
	}
	l, err := AcquireLock(dir)
	if err != nil {
		t.Fatalf("stale lock not cleared: %v", err)
	}
	l.Release()
}
