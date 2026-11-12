package tests

// The README trio's flag tables are hand-written against internal/config/flags.go
// (DOC-INTERNAL-QUALITY rule 4: a document derived from code is held to it). A flag added to the
// parser without a row, or a row left behind by a removed flag, fails here in every language.

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

var (
	// fs.Bool("register", ..), fs.String("ocr-lang", ..) and the like.
	flagDefinition = regexp.MustCompile(`\bfs\.(?:Bool|String|Int|Int64|Uint|Uint64|Float64|Duration)(?:Var)?\(\s*(?:&[\w.]+,\s*)?"([^"]+)"`)
	// A flag table row: | `-name` | default | description |
	readmeFlagRow = regexp.MustCompile("(?m)^\\|\\s*`-([a-z0-9-]+)`\\s*\\|")
)

func cliFlags(t *testing.T) map[string]bool {
	t.Helper()
	flags := map[string]bool{}
	for _, m := range flagDefinition.FindAllStringSubmatch(readRepoFile(t, "internal", "config", "flags.go"), -1) {
		flags[m[1]] = true
	}
	if len(flags) == 0 {
		t.Fatal("internal/config/flags.go: no flag definition found - the parser may have moved")
	}
	return flags
}

// The three guides carry the same flag table on the site (ticket E0111, site audit F21): a row per flag
// internal/config/flags.go defines, none it does not.
var guideFlagRow = regexp.MustCompile(`<tr><td><code>-([a-z0-9-]+)</code></td>`)

func TestGuideFlagTables(t *testing.T) {
	flags := cliFlags(t)
	for _, guide := range []string{"docs.html", "docs.ru.html", "docs.uk.html"} {
		rows := map[string]bool{}
		for _, m := range guideFlagRow.FindAllStringSubmatch(readRepoFile(t, guide), -1) {
			rows[m[1]] = true
		}
		var missing, stale []string
		for f := range flags {
			if !rows[f] {
				missing = append(missing, "-"+f)
			}
		}
		for r := range rows {
			if !flags[r] {
				stale = append(stale, "-"+r)
			}
		}
		sort.Strings(missing)
		sort.Strings(stale)
		if len(missing) > 0 || len(stale) > 0 {
			t.Errorf("%s: the flag table lacks %v and lists %v beyond internal/config/flags.go", guide, missing, stale)
		}
	}
}

// The guides' #cli-reference (ticket E0111, site audit F22). A DOCHT_ variable in the product's code is either on those
// pages or named here as not a user's, so a new one cannot stay undocumented by accident.
var dochtNotForUsers = map[string]string{
	"DOCHT_DIALOG_HOST":       "the GUI tells its converter child to ask over its pipes",
	"DOCHT_FAKE_FILEDO":       "a test double (internal/fdsec/fdsectest)",
	"DOCHT_FAKE_FILEDO_BUILD": "a test double (internal/fdsec/fdsectest)",
	"DOCHT_FDSEC_SECRET":      "the password, in the FileDO child's environment only",
	"DOCHT_GUI_NO_WINDOW":     "the help screenshot generator and UI automation",
	"DOCHT_OCR_DIAG":          "the OCR lab's corpus runner",
}

func TestGuideCLIReference(t *testing.T) {
	variable := regexp.MustCompile(`\bDOCHT_[A-Z_]+\b`)
	inCode := map[string]bool{}
	for _, tree := range []string{"cmd", "internal"} {
		root := filepath.Join("..", tree)
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return err
			}
			src, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			for _, v := range variable.FindAllString(string(src), -1) {
				inCode[v] = true
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	section := regexp.MustCompile(`(?s)<details class="sec" id="cli-reference">.*?</details>`)
	for _, guide := range []string{"docs.html", "docs.ru.html", "docs.uk.html"} {
		ref := section.FindString(readRepoFile(t, guide))
		if ref == "" {
			t.Errorf("%s: no #cli-reference section", guide)
			continue
		}
		for _, code := range []string{"0", "1", "2", "3", "4", "130"} {
			if !strings.Contains(ref, "<tr><td><code>"+code+"</code></td>") {
				t.Errorf("%s: #cli-reference has no row for exit code %s", guide, code)
			}
		}
		for v := range inCode {
			if _, internal := dochtNotForUsers[v]; !internal && !strings.Contains(ref, "<code>"+v+"</code>") {
				t.Errorf("%s: #cli-reference does not name %s; add it, or name it in dochtNotForUsers if no user sets it", guide, v)
			}
		}
		for _, fact := range []string{".doc-html-translate.json", "100", "32768", "20000", "4 "} {
			if !strings.Contains(ref, fact) {
				t.Errorf("%s: #cli-reference lacks %q (the input limits and the ownership record)", guide, fact)
			}
		}
	}
	for v := range dochtNotForUsers {
		if !inCode[v] {
			t.Errorf("dochtNotForUsers names %s, which cmd/ and internal/ no longer read", v)
		}
	}
}

func TestReadmeFlagTables(t *testing.T) {
	flags := cliFlags(t)
	for _, readme := range []string{"README.md", "README_RU.md", "README_UK.md"} {
		rows := map[string]bool{}
		for _, m := range readmeFlagRow.FindAllStringSubmatch(readRepoFile(t, readme), -1) {
			rows[m[1]] = true
		}
		var missing, stale []string
		for f := range flags {
			if !rows[f] {
				missing = append(missing, "-"+f)
			}
		}
		for r := range rows {
			if !flags[r] {
				stale = append(stale, "-"+r)
			}
		}
		sort.Strings(missing)
		sort.Strings(stale)
		if len(missing) > 0 {
			t.Errorf("%s: the flag table has no row for %v (defined in internal/config/flags.go)", readme, missing)
		}
		if len(stale) > 0 {
			t.Errorf("%s: the flag table lists %v, which internal/config/flags.go does not define", readme, stale)
		}
	}
}
