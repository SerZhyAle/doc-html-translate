package fdsec

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"doc-html-translate/internal/fdsec/fdsectest"
	"doc-html-translate/internal/outputpath"
)

func TestMain(m *testing.M) {
	fdsectest.RunFakeIfAsked()
	os.Exit(m.Run())
}

// seqAsker answers from a list and records how it was asked.
type seqAsker struct {
	secrets  []string
	calls    int
	retries  []bool
	canRetry bool
	err      error
}

func (a *seqAsker) Secret(_ string, retry bool) ([]byte, error) {
	a.retries = append(a.retries, retry)
	if a.err != nil {
		return nil, a.err
	}
	i := a.calls
	a.calls++
	if i >= len(a.secrets) {
		i = len(a.secrets) - 1
	}
	return []byte(a.secrets[i]), nil
}

func (a *seqAsker) CanRetry() bool { return a.canRetry }

func convertible(ext string) bool { return ext == ".txt" || ext == ".png" }

// fixture points the package at the fake FileDO and a private work root, and returns a
// container of a size that passes the length screen.
func fixture(t *testing.T, mode, build string) (container, root string) {
	t.Helper()
	t.Setenv(fdsectest.Env, mode)
	t.Setenv(fdsectest.BuildEnv, build)
	t.Setenv(OverrideEnv, os.Args[0])
	root = filepath.Join(t.TempDir(), "root")
	workRootOverride = root
	t.Cleanup(func() { workRootOverride = "" })
	container = filepath.Join(t.TempDir(), "report.fd-sec")
	if err := os.WriteFile(container, make([]byte, 12288), 0o600); err != nil {
		t.Fatal(err)
	}
	return container, root
}

func assertRootEmpty(t *testing.T, root string) {
	t.Helper()
	entries, err := os.ReadDir(root)
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		var names []string
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("the work root should be empty, holds %v", names)
	}
}

func classOf(t *testing.T, err error) Class {
	t.Helper()
	var fe *Error
	if !errors.As(err, &fe) {
		t.Fatalf("want *fdsec.Error, got %T %v", err, err)
	}
	return fe.Class
}

func TestScreenLength(t *testing.T) {
	cases := []struct {
		size int64
		want bool
		why  string
	}{
		{0, false, "empty"},
		{5631, false, "one short of suite 1/3"},
		{5632, true, "suite 1/3 minimum"},
		{5633, false, "suite 1/3 needs a multiple of 512"},
		{6144, true, "multiple of 512"},
		{9307, false, "one short of suite 2"},
		{9308, true, "suite 2 minimum"},
		{40 + 65552 + 16, false, "suite 2 last frame of 16 bytes"},
		{40 + 65552 + 17, true, "suite 2 last frame of 17 bytes"},
	}
	for _, c := range cases {
		if got := ScreenLength(c.size); got != c.want {
			t.Errorf("ScreenLength(%d) = %v, want %v (%s)", c.size, got, c.want, c.why)
		}
	}
}

func TestIsContainer(t *testing.T) {
	for ext, want := range map[string]bool{".fd-sec": true, ".FD-SEC": true, ".txt": false, "": false, "fd-sec": false} {
		if IsContainer(ext) != want {
			t.Errorf("IsContainer(%q) = %v", ext, !want)
		}
	}
}

func TestUnwrapOK(t *testing.T) {
	container, root := fixture(t, "ok-txt", "")
	p, err := Unwrap(context.Background(), container, &seqAsker{secrets: []string{"pw"}}, convertible)
	if err != nil {
		t.Fatal(err)
	}
	if p.Ext != ".txt" {
		t.Errorf("Ext = %q", p.Ext)
	}
	if got := filepath.Base(p.Path); got != "report.txt" {
		t.Errorf("the plain copy must carry the container's stem and the inner extension, got %q", got)
	}
	if strings.Contains(p.Path, fdsectest.TrueName) {
		t.Errorf("the sealed true name must not survive in the path: %s", p.Path)
	}
	if _, err := os.Stat(p.Path); err != nil {
		t.Errorf("the plain copy should exist until Remove: %v", err)
	}
	// A second copy must wait for the first: only one plain copy at a time.
	ctx, cancel := context.WithTimeout(context.Background(), 600*time.Millisecond)
	defer cancel()
	if _, err := Unwrap(ctx, container, &seqAsker{secrets: []string{"pw"}}, convertible); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("a second concurrent copy must be refused by waiting, got %v", err)
	}
	p.Remove()
	p.Remove() // idempotent
	assertRootEmpty(t, root)
}

func TestUnwrapEmptyPassword(t *testing.T) {
	container, root := fixture(t, "ok-txt", "")
	p, err := Unwrap(context.Background(), container, &seqAsker{secrets: []string{""}}, convertible)
	if err != nil {
		t.Fatal(err)
	}
	p.Remove()
	assertRootEmpty(t, root)
}

func TestUnwrapPicture(t *testing.T) {
	container, _ := fixture(t, "ok-png", "")
	p, err := Unwrap(context.Background(), container, &seqAsker{secrets: []string{"pw"}}, convertible)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Remove()
	if p.Ext != ".png" || filepath.Base(p.Path) != "report.png" {
		t.Errorf("got %s (%s)", p.Path, p.Ext)
	}
}

func TestOutcomeClasses(t *testing.T) {
	cases := []struct {
		mode string
		want Class
	}{
		{"wrong", Credential},
		{"damaged", Damaged},
		{"io", IO},
		{"unsupported", Unsupported},
		{"usage", RefusedCall},
		{"crash", Failed},
		{"ok-folder", Folder},
		{"ok-nested", Nested},
		{"ok-docx", InnerType},
	}
	for _, c := range cases {
		t.Run(c.mode, func(t *testing.T) {
			container, root := fixture(t, c.mode, "")
			_, err := Unwrap(context.Background(), container, &seqAsker{secrets: []string{"not-the-right-one"}}, convertible)
			if err == nil {
				t.Fatal("want an error")
			}
			if got := classOf(t, err); got != c.want {
				t.Errorf("class = %v, want %v (%v)", got, c.want, err)
			}
			assertRootEmpty(t, root)
			msg := err.Error()
			for _, bad := range []string{fdsectest.TrueName, "not-the-right-one", root, os.TempDir()} {
				if strings.Contains(msg, bad) {
					t.Errorf("error text %q contains %q", msg, bad)
				}
			}
			if !strings.Contains(msg, "report.fd-sec") && c.want != Failed {
				t.Errorf("error text should name the container: %q", msg)
			}
		})
	}
}

func TestInnerTypeNamesExtensionOnly(t *testing.T) {
	container, _ := fixture(t, "ok-docx", "")
	_, err := Unwrap(context.Background(), container, &seqAsker{secrets: []string{"pw"}}, convertible)
	var fe *Error
	if !errors.As(err, &fe) || fe.Inner != ".docx" {
		t.Fatalf("want InnerType with .docx, got %v", err)
	}
	if strings.Contains(err.Error(), fdsectest.TrueName) {
		t.Errorf("true name leaked: %v", err)
	}
}

func TestOutdatedBuildSuffix(t *testing.T) {
	for _, mode := range []string{"damaged", "unsupported"} {
		container, _ := fixture(t, mode, "2609241700")
		_, err := Unwrap(context.Background(), container, &seqAsker{secrets: []string{"pw"}}, convertible)
		if err == nil || !strings.Contains(err.Error(), "2609241700") || !strings.Contains(err.Error(), "winget upgrade") {
			t.Errorf("%s with an old build must say to update FileDO, got %v", mode, err)
		}
		container, _ = fixture(t, mode, "2610040306")
		_, err = Unwrap(context.Background(), container, &seqAsker{secrets: []string{"pw"}}, convertible)
		if err == nil || strings.Contains(err.Error(), "winget upgrade") {
			t.Errorf("%s with a current build must not blame FileDO's age, got %v", mode, err)
		}
	}
}

func TestRetryAsksAgainWithRetryFlag(t *testing.T) {
	container, root := fixture(t, "wrong", "")
	ask := &seqAsker{secrets: []string{"nope", fdsectest.RightPassword}, canRetry: true}
	p, err := Unwrap(context.Background(), container, ask, convertible)
	if err != nil {
		t.Fatal(err)
	}
	p.Remove()
	if len(ask.retries) != 2 || ask.retries[0] || !ask.retries[1] {
		t.Errorf("retry flags = %v, want [false true]", ask.retries)
	}
	assertRootEmpty(t, root)
}

func TestNoRetryWhenTheSourceCannotChange(t *testing.T) {
	container, _ := fixture(t, "wrong", "")
	ask := &seqAsker{secrets: []string{"nope"}}
	_, err := Unwrap(context.Background(), container, ask, convertible)
	if classOf(t, err) != Credential || len(ask.retries) != 1 {
		t.Errorf("one question, one Credential: %v after %d asks", err, len(ask.retries))
	}
}

func TestCancelledQuestionRemovesEverything(t *testing.T) {
	container, root := fixture(t, "ok-txt", "")
	ask := &seqAsker{err: &Error{Class: Cancelled}}
	_, err := Unwrap(context.Background(), container, ask, convertible)
	var fe *Error
	if !errors.As(err, &fe) || fe.Class != Cancelled || fe.Name != "report.fd-sec" {
		t.Errorf("got %v", err)
	}
	assertRootEmpty(t, root)
}

func TestRefusalsBeforeTheQuestion(t *testing.T) {
	t.Run("FileDO missing", func(t *testing.T) {
		container, root := fixture(t, "ok-txt", "")
		t.Setenv(OverrideEnv, filepath.Join(t.TempDir(), "missing.exe"))
		ask := &seqAsker{secrets: []string{"pw"}}
		_, err := Unwrap(context.Background(), container, ask, convertible)
		if classOf(t, err) != NotFound || len(ask.retries) != 0 {
			t.Errorf("want NotFound with no question, got %v after %d asks", err, len(ask.retries))
		}
		if !strings.Contains(err.Error(), "winget install SerZhyAle.FileDO") {
			t.Errorf("the message must say how to get FileDO: %v", err)
		}
		assertRootEmpty(t, root)
	})
	t.Run("length screen", func(t *testing.T) {
		container, root := fixture(t, "ok-txt", "")
		if err := os.WriteFile(container, make([]byte, 100), 0o600); err != nil {
			t.Fatal(err)
		}
		ask := &seqAsker{secrets: []string{"pw"}}
		_, err := Unwrap(context.Background(), container, ask, convertible)
		if classOf(t, err) != Screened || len(ask.retries) != 0 {
			t.Errorf("want Screened with no question, got %v after %d asks", err, len(ask.retries))
		}
		assertRootEmpty(t, root)
	})
}

func TestCancelDuringRestore(t *testing.T) {
	container, root := fixture(t, "ok-txt", "")
	ctx, cancel := context.WithCancel(context.Background())
	ask := &seqAsker{secrets: []string{"pw"}}
	cancel()
	_, err := Unwrap(ctx, container, ask, convertible)
	if err == nil {
		t.Fatal("a cancelled run must not succeed")
	}
	assertRootEmpty(t, root)
}

func TestSweepStale(t *testing.T) {
	root := filepath.Join(t.TempDir(), "root")
	workRootOverride = root
	t.Cleanup(func() { workRootOverride = "" })
	old := time.Now().Add(-time.Hour)

	stale := filepath.Join(root, "aaaaaaaa")
	locked := filepath.Join(root, "bbbbbbbb")
	fresh := filepath.Join(root, "cccccccc")
	for _, d := range []string{stale, locked, fresh} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(d, "plain.txt"), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	lock, err := outputpath.AcquireLock(locked)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Release()
	for _, d := range []string{stale, locked} {
		if err := os.Chtimes(d, old, old); err != nil {
			t.Fatal(err)
		}
	}
	// A file in the root (the one-copy lock's neighbour) is ignored.
	if err := os.WriteFile(filepath.Join(root, "note.txt"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	SweepStale()

	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Error("an old unlocked folder must be removed")
	}
	if _, err := os.Stat(locked); err != nil {
		t.Error("a locked folder must stay")
	}
	if _, err := os.Stat(fresh); err != nil {
		t.Error("a fresh folder must stay")
	}
	if _, err := os.Stat(filepath.Join(root, "note.txt")); err != nil {
		t.Error("files in the root must be left alone")
	}
}

func TestSweepStaleWithoutRoot(t *testing.T) {
	workRootOverride = filepath.Join(t.TempDir(), "never-created")
	t.Cleanup(func() { workRootOverride = "" })
	SweepStale() // must not panic or create anything
	if _, err := os.Stat(workRootOverride); !os.IsNotExist(err) {
		t.Error("the sweep must not create the work root")
	}
}

func TestNeutralStem(t *testing.T) {
	long := strings.Repeat("x", 100)
	for in, want := range map[string]string{
		"report.fd-sec":  "report",
		"a:b*c.fd-sec":   "a_b_c",
		"CON.fd-sec":     "_CON",
		".fd-sec":        "document",
		"dots...fd-sec":  "dots",
		long + ".fd-sec": strings.Repeat("x", 64),
	} {
		if got := neutralStem(in); got != want {
			t.Errorf("neutralStem(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestLocateOverrideAndMissing(t *testing.T) {
	t.Setenv(OverrideEnv, os.Args[0])
	if p, err := Locate(); err != nil || p != os.Args[0] {
		t.Errorf("override: %q %v", p, err)
	}
	t.Setenv(OverrideEnv, filepath.Join(t.TempDir(), "nope"))
	if _, err := Locate(); classOf(t, err) != NotFound {
		t.Errorf("a missing override must be NotFound, got %v", err)
	}
	if Available() {
		t.Error("Available must follow Locate")
	}
}
