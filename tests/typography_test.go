package tests

// Typography drift guard. CLAUDE.md's rule - "use short hyphens (no long dashes) and '..'
// not '...'" - applies to generated output, but nothing stopped it drifting back into new
// string literals. This test parses every non-test Go source file under internal/ and cmd/
// and fails on a long dash, a three-dot ellipsis, or a '...' literal in any string it emits.
//
// One deliberate exception (settled 2026-07-18): the converted page <title>Book - Page N</title>
// keeps an em dash, because that is conventional book typography rather than app chatter. The
// guard therefore allows an em dash only inside a literal that also contains "<title>".
//
// The extension's user-visible text lives in _locales/*/messages.json; those message values are
// checked too. Extension JS source is not scanned here: the spread operator (`...x`) and em
// dashes in `//` comments make a reliable text scan impractical, and the UI strings are in
// _locales - see docs/PARITY.md and the ticket DEV/plan/done/2026-07-17_output-typography.md.

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// houseStyleLangs are the languages the author writes and proofreads. CLAUDE.md's typography
// rule is an author-language rule: it says how *we* write, not how every script punctuates.
// Chinese uses "——" as its dash and "……" as its ellipsis, German and French use "–", and the
// Arabic-script languages have their own marks - enforcing the house style on those would mean
// shipping mistranslated punctuation, so they are checked by the weaker rule below.
var houseStyleLangs = map[string]bool{"en": true, "ru": true, "uk": true}

// badTypography returns a reason if v breaks the rule for lang, or "" if it is clean.
//
// For an author language: no ellipsis character, no three-dot ellipsis, and no em dash - except
// inside a literal that also contains "<title>", which is the page-title exception settled
// 2026-07-18. For every other language only an ASCII three-dot "..." is rejected, and not even
// that in Chinese, whose own ellipsis is a doubled U+2026.
func badTypography(lang, v string) string {
	if !houseStyleLangs[lang] {
		if lang != "zh" && strings.Contains(v, "...") {
			return "ASCII three-dot ellipsis (...) - use the script's own ellipsis"
		}
		return ""
	}
	if strings.Contains(v, "…") {
		return "ellipsis character (…) - use '..'"
	}
	if strings.Contains(v, "...") {
		return "three-dot ellipsis (...) - use '..'"
	}
	if strings.Contains(v, "—") && !strings.Contains(v, "<title>") {
		return "long dash (—) - use a short hyphen '-'"
	}
	return ""
}

// TestTypographyGoOutput scans every non-test Go string literal under internal/ and cmd/.
func TestTypographyGoOutput(t *testing.T) {
	roots := []string{filepath.Join("..", "internal"), filepath.Join("..", "cmd")}
	fset := token.NewFileSet()
	for _, root := range roots {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			file, perr := parser.ParseFile(fset, path, nil, 0)
			if perr != nil {
				t.Errorf("parse %s: %v", path, perr)
				return nil
			}
			ast.Inspect(file, func(n ast.Node) bool {
				lit, ok := n.(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					return true
				}
				v, uerr := strconv.Unquote(lit.Value)
				if uerr != nil {
					return true
				}
				// Go source literals are English by canon, so they are always an author language.
				if reason := badTypography("en", v); reason != "" {
					pos := fset.Position(lit.Pos())
					t.Errorf("%s:%d: %s in %q", filepath.ToSlash(path), pos.Line, reason, v)
				}
				return true
			})
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", root, err)
		}
	}
}

// TestTypographyLocaleMessages scans the extension's user-visible message strings.
func TestTypographyLocaleMessages(t *testing.T) {
	base := filepath.Join("..", "extension", "_locales")
	langs, err := os.ReadDir(base)
	if err != nil {
		t.Fatalf("read %s: %v", base, err)
	}
	seen := 0
	for _, lang := range langs {
		if !lang.IsDir() {
			continue
		}
		path := filepath.Join(base, lang.Name(), "messages.json")
		raw, rerr := os.ReadFile(path)
		if rerr != nil {
			continue
		}
		var msgs map[string]struct {
			Message string `json:"message"`
		}
		if jerr := json.Unmarshal(raw, &msgs); jerr != nil {
			t.Errorf("parse %s: %v", path, jerr)
			continue
		}
		seen++
		// The language is the directory name; Chrome's regional dirs ("pt_BR", "zh_CN") carry
		// the variant after an underscore. An unknown directory falls through to the weaker
		// rule rather than to the house style - a new locale must never fail the build for
		// punctuating in its own script.
		code, _, _ := strings.Cut(lang.Name(), "_")
		for key, m := range msgs {
			if reason := badTypography(code, m.Message); reason != "" {
				t.Errorf("%s [%s]: %s in %q", filepath.ToSlash(path), key, reason, m.Message)
			}
		}
	}
	if seen == 0 {
		t.Fatal("no _locales/*/messages.json scanned - path may have moved")
	}
}

// TestBadTypographyLanguageScope pins the rule itself: the house style binds the author
// languages, and a translated string may punctuate the way its script does.
func TestBadTypographyLanguageScope(t *testing.T) {
	cases := []struct {
		name   string
		lang   string
		value  string
		reject bool
	}{
		{"en ascii ellipsis", "en", "wait...", true},
		{"en short hyphen", "en", "a - b", false},
		{"en title em dash", "en", "<title>Book — Page 1</title>", false},
		{"ru ellipsis char", "ru", "ждём…", true},
		{"uk em dash", "uk", "текст — далі", true},
		{"zh double dash", "zh", "等待——完成", false},
		{"zh own ellipsis", "zh", "等待……", false},
		{"de en dash", "de", "Zeit – Ort", false},
		{"ar question mark", "ar", "انتظر؟", false},
		{"hi ascii ellipsis", "hi", "रुकिए...", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			reason := badTypography(c.lang, c.value)
			if c.reject && reason == "" {
				t.Errorf("%q in %s: expected a rejection, got none", c.value, c.lang)
			}
			if !c.reject && reason != "" {
				t.Errorf("%q in %s: unexpected rejection: %s", c.value, c.lang, reason)
			}
		})
	}
}

// TestTypographySplashResources scans the embedded per-language console splash text. These are
// resources rather than Go literals, so the AST walk above cannot see them. Matching nothing is
// a pass: the files arrive with the localized CLI.
func TestTypographySplashResources(t *testing.T) {
	paths, err := filepath.Glob(filepath.Join("..", "internal", "app", "splash", "*.txt"))
	if err != nil {
		t.Fatalf("glob splash resources: %v", err)
	}
	for _, path := range paths {
		raw, rerr := os.ReadFile(path)
		if rerr != nil {
			t.Errorf("read %s: %v", path, rerr)
			continue
		}
		code := strings.TrimSuffix(filepath.Base(path), ".txt")
		if reason := badTypography(code, string(raw)); reason != "" {
			t.Errorf("%s: %s", filepath.ToSlash(path), reason)
		}
	}
}
