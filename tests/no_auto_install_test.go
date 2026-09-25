package tests

// Nothing is installed on the user's machine without their explicit consent (ticket
// bugfix-external-process-bounds, done criterion 2). The converter once started a system-wide
// `winget install` with auto-accepted agreements when the bundled pdftotext was blocked. This
// guard parses every non-test Go file under internal/ and cmd/ and fails on a string literal
// that could only be part of such a call: the bare program name "winget" (advice text naming
// the command inside a sentence is fine) or winget's agreement-accepting flags.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestNoPackageManagerIsInvoked(t *testing.T) {
	forbidden := map[string]bool{
		"winget":                      true,
		"winget.exe":                  true,
		"--accept-package-agreements": true,
		"--accept-source-agreements":  true,
	}
	for _, root := range []string{"../internal", "../cmd"} {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			fset := token.NewFileSet()
			f, perr := parser.ParseFile(fset, path, nil, 0)
			if perr != nil {
				return perr
			}
			ast.Inspect(f, func(n ast.Node) bool {
				lit, ok := n.(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					return true
				}
				v, uerr := strconv.Unquote(lit.Value)
				if uerr == nil && forbidden[strings.ToLower(v)] {
					t.Errorf("%s: %q - the converter must not drive a package manager", fset.Position(lit.Pos()), v)
				}
				return true
			})
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}
