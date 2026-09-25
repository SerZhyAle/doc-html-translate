package main

import (
	"testing"

	"doc-html-translate/internal/config"
)

// An input whose name looks like a flag must be converted, not obeyed.
func TestAssembleArgsInputStartingWithDashIsAFile(t *testing.T) {
	for _, in := range []string{"-force.epub", "-folder", "--", "-"} {
		args := assembleArgs(runRequest{Input: in, SrcLang: "en", DstLang: "ru", SinglePage: true,
			OllamaModel: "gemma3:12b", OllamaParallel: "1", OllamaCtx: "8192"})
		cfg, err := config.ParseArgs(args)
		if err != nil {
			t.Fatalf("input %q: %v (args %v)", in, err, args)
		}
		if cfg.InputFile != in {
			t.Errorf("input %q parsed as InputFile %q (args %v)", in, cfg.InputFile, args)
		}
		if cfg.Force {
			t.Errorf("input %q switched -force on", in)
		}
	}
}

func TestTrimTrailingSeparators(t *testing.T) {
	cases := map[string]string{
		`C:\out dir\`:  `C:\out dir`,
		`C:\out dir\\`: `C:\out dir`,
		`C:\`:          `C:\`,
		`/`:            `/`,
		`C:\a.epub`:    `C:\a.epub`,
	}
	for in, want := range cases {
		if got := trimTrailingSeparators(in); got != want {
			t.Errorf("trimTrailingSeparators(%q) = %q, want %q", in, got, want)
		}
	}
}

// The displayed command must be valid PowerShell: metacharacters sit inside single
// quotes, where PowerShell expands nothing, and a quote in the value is doubled.
func TestQuoteArgForPowerShell(t *testing.T) {
	cases := map[string]string{
		"-src":                   "-src",
		`C:\books\story.epub`:    `C:\books\story.epub`,
		`C:\books\Книга.epub`:    `C:\books\Книга.epub`,
		`C:\books\My Story.epub`: `'C:\books\My Story.epub'`,
		`C:\a&calc&b.epub`:       `'C:\a&calc&b.epub'`,
		`C:\Q&A.epub`:            `'C:\Q&A.epub'`,
		`C:\50%.epub`:            `'C:\50%.epub'`,
		`C:\$env:x.epub`:         `'C:\$env:x.epub'`,
		`C:\a^b!(c),d;e.epub`:    `'C:\a^b!(c),d;e.epub'`,
		"C:\\it's.epub":          "'C:\\it''s.epub'",
		"C:\\it\u2019s.epub":     "'C:\\it\u2019\u2019s.epub'",
		`gemma3:12b`:             `gemma3:12b`,
		`say "hi"`:               `'say "hi"'`,
		"":                       "''",
		"--":                     "'--'",
		"a`b":                    "'a`b'",
	}
	for in, want := range cases {
		if got := quoteArg(in); got != want {
			t.Errorf("quoteArg(%q) = %s, want %s", in, got, want)
		}
	}
}

func TestFormatCommandLineUsesCallOperator(t *testing.T) {
	got := formatCommandLine(`C:\Program Files\app\doc-html-translate.exe`, []string{"-src", "en", "--", `C:\Q&A.epub`})
	want := `& 'C:\Program Files\app\doc-html-translate.exe' -src en '--' 'C:\Q&A.epub'`
	if got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}
