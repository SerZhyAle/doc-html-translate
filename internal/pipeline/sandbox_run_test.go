package pipeline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"doc-html-translate/internal/config"
	"doc-html-translate/internal/outputpath"
)

const sandboxMarkdown = "# One\n\nFirst chapter text.\n\n# Two\n\nSecond chapter text.\n"

// A finished run leaves an index with a TOC, the pages, and a completion record that names this
// source and the options it was built with - the three things the next run's reuse check reads.
func TestSandboxRunWritesIndexAndCompletionRecord(t *testing.T) {
	for _, tc := range []struct{ name, content string }{
		{"book.md", sandboxMarkdown},
		{"book.txt", "First paragraph.\n\nSecond paragraph.\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := newPipelineSandbox(t)
			cfg := s.config(s.input(tc.name, tc.content))
			s.mustRun(cfg)

			out := s.outputDir(cfg.InputFile)
			index, err := os.ReadFile(filepath.Join(out, "index.html"))
			if err != nil {
				t.Fatalf("index.html: %v", err)
			}
			if !strings.Contains(string(index), "</html>") {
				t.Fatal("index.html is truncated")
			}
			if s.exists(filepath.Join(out, outputpath.LockName)) {
				t.Error("the run left its lock behind")
			}
			m, err := outputpath.ReadMarker(out)
			if err != nil {
				t.Fatalf("marker: %v", err)
			}
			if m.Source != cfg.InputFile || m.Complete == nil {
				t.Fatalf("marker = %+v", m)
			}
			if m.Complete.Translation != outputpath.TranslationNone {
				t.Errorf("translation = %q, want %q", m.Complete.Translation, outputpath.TranslationNone)
			}
			if m.Complete.Options != outputpath.OptionsFor(cfg) {
				t.Errorf("recorded options = %+v, want %+v", m.Complete.Options, outputpath.OptionsFor(cfg))
			}
			if r := s.reuse(cfg); r != outputpath.ReuseOK {
				t.Errorf("finished output is not reusable: reason %v", r)
			}
		})
	}
}

// The two Markdown headings become two pages, and the generated index links both.
func TestSandboxMarkdownBuildsMultiPageTOC(t *testing.T) {
	s := newPipelineSandbox(t)
	cfg := s.config(s.input("book.md", sandboxMarkdown))
	s.mustRun(cfg)

	out := s.outputDir(cfg.InputFile)
	index, err := os.ReadFile(filepath.Join(out, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	for _, page := range []string{"page_001.html", "page_002.html"} {
		if !s.exists(filepath.Join(out, page)) {
			t.Errorf("%s missing", page)
		}
		if !strings.Contains(string(index), page) {
			t.Errorf("index does not link %s", page)
		}
	}
}

// A first run that fails must not leave a folder behind: an output that exists without a
// completion record is at best clutter, and at worst a half-built book a later tool opens.
func TestSandboxFailedFirstRunLeavesNoOutput(t *testing.T) {
	for _, tc := range []struct {
		name, content string
		code          int
	}{
		// A ZIP under an unknown extension is refused by the binary sniff.
		{"report.dat", "PK\x03\x04garbage", ExitParse},
		// Whitespace-only Markdown passes the size check and fails in the extractor.
		{"blank.md", "   \n\n  \n", ExitParse},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := newPipelineSandbox(t)
			cfg := s.config(s.input(tc.name, tc.content))
			code, err := s.run(cfg)
			if code != tc.code || err == nil {
				t.Fatalf("code = %d, err = %v; want %d with an error", code, err, tc.code)
			}
			if out := s.outputDir(cfg.InputFile); s.exists(out) {
				t.Errorf("failed run left %s behind", out)
			}
			if r := s.reuse(cfg); r != outputpath.ReuseNoRecord {
				t.Errorf("reuse after a failed run = %v, want ReuseNoRecord", r)
			}
		})
	}
}

// The dangerous failure is the rebuild of a finished output: if the old completion record
// survived, the next run would reopen the previous book as though the new source converted.
func TestSandboxFailedRebuildDropsPreviousRecord(t *testing.T) {
	s := newPipelineSandbox(t)
	in := s.input("notes.dat", "Plain text notes.\n\nSecond paragraph.\n")
	cfg := s.config(in)
	s.mustRun(cfg)
	if s.record(in) == nil {
		t.Fatal("first run left no completion record")
	}

	s.rewrite(in, "PK\x03\x04 now a zip archive, not text")
	if code, err := s.run(cfg); code != ExitParse || err == nil {
		t.Fatalf("rebuild of a binary: code = %d, err = %v; want ExitParse", code, err)
	}
	out := s.outputDir(in)
	if s.exists(filepath.Join(out, "index.html")) {
		t.Error("the failed rebuild left the previous index.html in place")
	}
	if rec := s.record(in); rec != nil {
		t.Errorf("the failed rebuild left a completion record: %+v", rec)
	}
	if s.exists(filepath.Join(out, outputpath.LockName)) {
		t.Error("the failed rebuild left its lock behind")
	}

	// Once the source is text again, the next run converts it rather than refusing the folder.
	s.rewrite(in, "Fixed text.\n")
	s.mustRun(cfg)
	// A one-page book's index.html only redirects, so the text is looked for in the page.
	if data, _ := os.ReadFile(filepath.Join(out, "page_001.html")); !strings.Contains(string(data), "Fixed text.") {
		t.Error("the run after a failed rebuild did not convert the current source")
	}
}

// Reuse is decided by the source and the result-affecting options only. The sentinel survives
// exactly when the output was reused, so each case proves which path the run took.
func TestSandboxReuseAndRebuildTriggers(t *testing.T) {
	cases := []struct {
		name    string
		change  func(s *pipelineSandbox, cfg *config.Config)
		reused  bool
		comment string
	}{
		{"same source and options", func(*pipelineSandbox, *config.Config) {}, true, ""},
		{"execution-only switches", func(_ *pipelineSandbox, c *config.Config) {
			c.Verbose, c.OllamaParallel, c.MaxCost, c.UILang = true, 4, 1.5, "ru"
		}, true, "verbose, parallelism, cost limit and UI language do not change the book"},
		{"split size", func(_ *pipelineSandbox, c *config.Config) { c.SplitSize = 3000 }, false, ""},
		{"toc depth", func(_ *pipelineSandbox, c *config.Config) { c.TOCDepth = 1 }, false, ""},
		{"single page", func(_ *pipelineSandbox, c *config.Config) { c.SinglePage = true }, false, ""},
		{"source edited", func(s *pipelineSandbox, c *config.Config) {
			s.rewrite(c.InputFile, "# One\n\nEdited chapter text.\n")
		}, false, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := newPipelineSandbox(t)
			cfg := s.config(s.input("book.md", sandboxMarkdown))
			s.mustRun(cfg)
			index := filepath.Join(s.outputDir(cfg.InputFile), "index.html")
			before, err := os.Stat(index)
			if err != nil {
				t.Fatal(err)
			}
			sentinel := s.plantSentinel(cfg.InputFile)

			tc.change(s, &cfg)
			s.mustRun(cfg)

			if got := s.exists(sentinel); got != tc.reused {
				t.Fatalf("output reused = %v, want %v %s", got, tc.reused, tc.comment)
			}
			after, err := os.Stat(index)
			if err != nil {
				t.Fatalf("index.html after the second run: %v", err)
			}
			if tc.reused && !after.ModTime().Equal(before.ModTime()) {
				t.Error("a reused output had its index.html rewritten")
			}
			if rec := s.record(cfg.InputFile); rec == nil || rec.Options != outputpath.OptionsFor(cfg) {
				t.Errorf("record after the second run = %+v, want options %+v", rec, outputpath.OptionsFor(cfg))
			}
		})
	}
}

// -force rebuilds even when the output is reusable, and keeps doing so on every run.
func TestSandboxForceAlwaysRebuilds(t *testing.T) {
	s := newPipelineSandbox(t)
	cfg := s.config(s.input("book.md", sandboxMarkdown))
	s.mustRun(cfg)

	forced := cfg
	forced.Force = true
	for i := 1; i <= 2; i++ {
		if r := s.reuse(cfg); r != outputpath.ReuseOK {
			t.Fatalf("pass %d: output not reusable before -force (reason %v), the test proves nothing", i, r)
		}
		sentinel := s.plantSentinel(cfg.InputFile)
		s.mustRun(forced)
		if s.exists(sentinel) {
			t.Fatalf("pass %d: -force reused the output", i)
		}
		if !s.exists(filepath.Join(s.outputDir(cfg.InputFile), "index.html")) {
			t.Fatalf("pass %d: -force left no index.html", i)
		}
	}
	// -force is an execution switch: the rebuilt output stays reusable without it.
	if r := s.reuse(cfg); r != outputpath.ReuseOK {
		t.Errorf("output after -force is not reusable without it: reason %v", r)
	}
}
