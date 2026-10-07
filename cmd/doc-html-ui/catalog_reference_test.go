package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"
)

// WINDOWS-UI section 5: the catalog reference records source-derived baseline metrics,
// not a runtime verdict. Compare its common target floor with the launcher, without
// importing another toolkit's row widths or claiming the new section 9 frame budgets.
func TestWindowsUIReferenceTargetIfAvailable(t *testing.T) {
	catalog := os.Getenv("SZA_CONTRACTS_ROOT")
	if catalog == "" {
		t.Skip("optional catalog probe: set SZA_CONTRACTS_ROOT to compare the Windows UI reference")
	}
	raw, err := os.ReadFile(filepath.Join(catalog, "desktop-app-ux", "reference", "windows-ui-reference.json"))
	if err != nil {
		t.Fatal(err)
	}
	var ref struct {
		Contract string `json:"contract"`
		Sources  []struct {
			Path      string            `json:"path"`
			Constants map[string]string `json:"constants"`
		} `json:"sources"`
	}
	if err := json.Unmarshal(raw, &ref); err != nil {
		t.Fatal(err)
	}
	if ref.Contract != "WINDOWS-UI" {
		t.Fatalf("unexpected reference contract %q", ref.Contract)
	}
	for _, source := range ref.Sources {
		if source.Path != "src/Chrome/ChromeMetrics.vb" {
			continue
		}
		floor, err := strconv.Atoi(source.Constants["MinTarget"])
		if err != nil || floor <= 0 {
			t.Fatalf("invalid reference MinTarget %q", source.Constants["MinTarget"])
		}
		match := regexp.MustCompile(`\.inline-btn \{ min-height: (\d+)px;`).FindStringSubmatch(uiHTML)
		if match == nil {
			t.Fatal("launcher inline button target rule is missing")
		}
		got, err := strconv.Atoi(match[1])
		if err != nil || got < floor {
			t.Fatalf("launcher button target %s px is below the reference %d px", match[1], floor)
		}
		t.Logf("WINDOWS-UI reference: MinTarget=%d, launcher=%d; runtime acceptance remains separate", floor, got)
		return
	}
	t.Fatal("reference lacks ChromeMetrics MinTarget")
}
