package tests

// The README trio's flag tables are hand-written against internal/config/flags.go
// (DOC-INTERNAL-QUALITY rule 4: a document derived from code is held to it). A flag added to the
// parser without a row, or a row left behind by a removed flag, fails here in every language.

import (
	"regexp"
	"sort"
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
