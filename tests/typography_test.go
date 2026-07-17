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

// badTypography returns a reason if v breaks the rule, or "" if it is clean. A literal that
// contains "<title>" is allowed to carry an em dash (the page-title exception); everything
// else must use a short hyphen. Nobody may use "..." or a "..." ellipsis.
func badTypography(v string) string {
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
				if reason := badTypography(v); reason != "" {
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
		for key, m := range msgs {
			if reason := badTypography(m.Message); reason != "" {
				t.Errorf("%s [%s]: %s in %q", filepath.ToSlash(path), key, reason, m.Message)
			}
		}
	}
	if seen == 0 {
		t.Fatal("no _locales/*/messages.json scanned - path may have moved")
	}
}
