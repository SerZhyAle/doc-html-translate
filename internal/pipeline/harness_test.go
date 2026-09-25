package pipeline

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"doc-html-translate/internal/config"
	"doc-html-translate/internal/outputpath"
	"doc-html-translate/internal/translator"
)

// pipelineSandbox runs the real pipeline against inputs in a private temp directory. Every
// place the app could reach outside that directory - the per-user app data and temp folders,
// the browser, the translation engines, the cost dialog - is redirected or stubbed, so a test
// that forgets a flag fails instead of touching the developer's machine or the network.
// Later tickets extend it rather than growing another ad-hoc runner.
type pipelineSandbox struct {
	t   *testing.T
	dir string
}

var errSandboxEngine = errors.New("sandbox: translation engines are disabled")

// sandboxEngine fails every request, so a run that reaches an engine without a test asking for
// one ends partial instead of calling a network service.
type sandboxEngine struct{}

func (sandboxEngine) Translate(context.Context, []string, string, string) ([]string, error) {
	return nil, errSandboxEngine
}

func newPipelineSandbox(t *testing.T) *pipelineSandbox {
	t.Helper()
	root := t.TempDir()
	home := filepath.Join(root, "home")
	tmp := filepath.Join(root, "tmp")
	for _, d := range []string{home, tmp} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	// Covers both platforms' lookups: os.UserCacheDir, os.TempDir and the explicit
	// %LOCALAPPDATA% reads of the key and report stores.
	for k, v := range map[string]string{
		"LOCALAPPDATA":   home,
		"APPDATA":        home,
		"USERPROFILE":    home,
		"HOME":           home,
		"XDG_CACHE_HOME": home,
		"TEMP":           tmp,
		"TMP":            tmp,
		"TMPDIR":         tmp,
	} {
		t.Setenv(k, v)
	}
	docs := filepath.Join(root, "docs")
	if err := os.MkdirAll(docs, 0o755); err != nil {
		t.Fatal(err)
	}
	return &pipelineSandbox{t: t, dir: docs}
}

// input writes an input document into the sandbox and returns its absolute path.
func (s *pipelineSandbox) input(name, content string) string {
	s.t.Helper()
	p := filepath.Join(s.dir, name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		s.t.Fatal(err)
	}
	return p
}

// rewrite replaces an input's content and moves its mtime forward, so the change is visible to
// the completion record even on a filesystem with coarse timestamps.
func (s *pipelineSandbox) rewrite(path, content string) {
	s.t.Helper()
	fi, err := os.Stat(path)
	if err != nil {
		s.t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		s.t.Fatal(err)
	}
	later := fi.ModTime().Add(2 * time.Second)
	if err := os.Chtimes(path, later, later); err != nil {
		s.t.Fatal(err)
	}
}

// config is the baseline a sandbox run starts from: multi-page, no translation, no browser.
func (s *pipelineSandbox) config(in string) config.Config {
	return config.Config{
		InputFile: in, NoOpen: true, NoTranslate: true,
		SourceLang: "en", TargetLang: "de", TOCDepth: 3,
	}
}

// runner builds the runner for cfg with the sandbox's guards in place. NoOpen is forced, not
// defaulted: a test mutating cfg must never be one field away from launching a browser.
func (s *pipelineSandbox) runner(cfg config.Config) Runner {
	cfg.NoOpen = true
	r := NewRunner(cfg)
	r.engines = engines{
		googleKey: func() (string, error) { return "", errSandboxEngine },
		google:    func(string) translator.Client { return sandboxEngine{} },
		ollama:    func(config.Config) translator.Client { return sandboxEngine{} },
		confirm:   func(string, string) bool { return false },
	}
	return r
}

func (s *pipelineSandbox) run(cfg config.Config) (int, error) {
	return s.runner(cfg).Run()
}

// mustRun runs cfg and fails the test unless the run succeeded.
func (s *pipelineSandbox) mustRun(cfg config.Config) {
	s.t.Helper()
	if code, err := s.run(cfg); code != ExitOK || err != nil {
		s.t.Fatalf("run %s: code = %d, err = %v", filepath.Base(cfg.InputFile), code, err)
	}
}

// outputDir is where a run of in writes when nothing else claims the name.
func (s *pipelineSandbox) outputDir(in string) string {
	return outputpath.OutputDirFor(in, "")
}

// record returns the completion record of in's output, or nil when there is none.
func (s *pipelineSandbox) record(in string) *outputpath.Completion {
	m, err := outputpath.ReadMarker(s.outputDir(in))
	if err != nil {
		return nil
	}
	return m.Complete
}

// reuse is the decision the next run of cfg would take on the existing output.
func (s *pipelineSandbox) reuse(cfg config.Config) outputpath.Reason {
	r, _ := outputpath.CheckReuse(s.outputDir(cfg.InputFile), cfg.InputFile, outputpath.OptionsFor(cfg))
	return r
}

// plantSentinel drops a file no conversion writes into in's output. It survives a reuse and is
// gone after a rebuild, which clears the folder - the proof of which of the two happened.
func (s *pipelineSandbox) plantSentinel(in string) string {
	s.t.Helper()
	p := filepath.Join(s.outputDir(in), "sandbox-sentinel")
	if err := os.WriteFile(p, nil, 0o644); err != nil {
		s.t.Fatal(err)
	}
	return p
}

func (s *pipelineSandbox) exists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
