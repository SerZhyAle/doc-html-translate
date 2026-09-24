package tests

// The structural drift check (scripts/parity-check.ps1) watches the pairs in configs/parity-map.json;
// the humans read the port map table at the top of docs/PARITY.md. The two were once a hand mirror
// with nothing comparing them, so the script silently stopped watching files the doc said were paired
// (ocr-plates.js, the GUI settings page). This test is the comparison (BUILD-EVIDENCE rule 7: a
// hand-kept twin ships with the check that proves it agrees).

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

type parityMap struct {
	Pairs []struct {
		Name string   `json:"name"`
		Go   []string `json:"go"`
		JS   []string `json:"js"`
	} `json:"pairs"`
	NotWatched []struct {
		Path   string `json:"path"`
		Reason string `json:"reason"`
	} `json:"notWatched"`
	Acknowledge []string `json:"acknowledge"`
}

// portMapLink captures a repo-relative link target inside a table cell: [text](../path).
var portMapLink = regexp.MustCompile(`\]\(\.\./([^)#\s]+)\)`)

// portMapRows returns the Go and JS link targets of every row of the port map table in which both
// sides link at least one file - a one-sided row ("(none - extension-only by design)") is not a pair.
func portMapRows(t *testing.T) [][2][]string {
	t.Helper()
	doc := readRepoFile(t, "docs", "PARITY.md")
	start := strings.Index(doc, "## The port map")
	if start < 0 {
		t.Fatal(`docs/PARITY.md: no "## The port map" section`)
	}
	section := doc[start+len("## The port map"):]
	if end := strings.Index(section, "\n## "); end >= 0 {
		section = section[:end]
	}
	var rows [][2][]string
	for _, line := range strings.Split(section, "\n") {
		cells := strings.Split(line, "|")
		// "| capability | go | js |" splits into 5 pieces with empty ends.
		if len(cells) != 5 || strings.HasPrefix(strings.TrimSpace(cells[1]), "---") {
			continue
		}
		var row [2][]string
		for side, cell := range []string{cells[2], cells[3]} {
			for _, m := range portMapLink.FindAllStringSubmatch(cell, -1) {
				row[side] = append(row[side], m[1])
			}
		}
		if len(row[0]) > 0 && len(row[1]) > 0 {
			rows = append(rows, row)
		}
	}
	if len(rows) == 0 {
		t.Fatal("docs/PARITY.md: the port map table has no paired rows - the parser no longer matches the table")
	}
	return rows
}

func covers(pattern, path string) bool {
	if strings.HasSuffix(pattern, "/") {
		return strings.HasPrefix(path, pattern)
	}
	return pattern == path
}

func TestParityMapMatchesPortMap(t *testing.T) {
	raw := readRepoFile(t, "configs", "parity-map.json")
	var pm parityMap
	if err := json.Unmarshal([]byte(raw), &pm); err != nil {
		t.Fatalf("configs/parity-map.json: %v", err)
	}
	notWatched := map[string]bool{}
	for _, n := range pm.NotWatched {
		if strings.TrimSpace(n.Reason) == "" {
			t.Errorf("configs/parity-map.json: notWatched %q carries no reason", n.Path)
		}
		notWatched[n.Path] = true
	}
	rows := portMapRows(t)

	// Doc -> map: every file the port map pairs is watched on its side, or excused with a reason.
	for _, row := range rows {
		for side, paths := range row {
			for _, p := range paths {
				if notWatched[p] {
					continue
				}
				found := false
				for _, pair := range pm.Pairs {
					patterns := pair.Go
					if side == 1 {
						patterns = pair.JS
					}
					for _, pat := range patterns {
						if covers(pat, p) {
							found = true
						}
					}
				}
				if !found {
					t.Errorf("docs/PARITY.md pairs %s, but configs/parity-map.json does not watch it (add it to a pair, or to notWatched with a reason)", p)
				}
			}
		}
	}

	// Map -> doc: every watched path is named by the port map on the same side and exists on disk.
	for _, pair := range pm.Pairs {
		for side, patterns := range [][]string{pair.Go, pair.JS} {
			for _, pat := range patterns {
				if _, err := os.Stat(filepath.Join("..", filepath.FromSlash(strings.TrimSuffix(pat, "/")))); err != nil {
					t.Errorf("configs/parity-map.json %q: %s does not exist", pair.Name, pat)
				}
				named := false
				for _, row := range rows {
					for _, p := range row[side] {
						if covers(pat, p) {
							named = true
						}
					}
				}
				if !named {
					t.Errorf("configs/parity-map.json %q watches %s, which the docs/PARITY.md port map does not pair on that side", pair.Name, pat)
				}
			}
		}
	}

	for _, ack := range pm.Acknowledge {
		if _, err := os.Stat(filepath.Join("..", filepath.FromSlash(ack))); err != nil {
			t.Errorf("configs/parity-map.json acknowledge %s: %v", ack, err)
		}
	}
}
