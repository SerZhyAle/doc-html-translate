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
// not a runtime verdict. Since 0.11 (section 16 A) its MinTarget is one toolkit's snapshot and
// not a floor for any product's targets - UI-SIZING decides those - so the probe only reports
// the two numbers side by side, without importing another toolkit's row widths or claiming
// the section 9 frame budgets.
//
// The launcher no longer has a fixed floor: UI-SIZING (ticket 119) replaced it with one token
// resolver, where the pointer-profile target is --tgt = base * --s and the smallest declared
// scale is the lower bound of the --s clamp. The compact pointer target is their product.
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
		base := regexp.MustCompile(`--tgt: calc\((\d+)px \* var\(--s\)\);`).FindStringSubmatch(uiHTML)
		scale := regexp.MustCompile(`--s: clamp\(([0-9.]+),`).FindStringSubmatch(uiHTML)
		if base == nil || scale == nil {
			t.Fatal("launcher target token (--tgt / --s) is missing")
		}
		px, err := strconv.Atoi(base[1])
		if err != nil {
			t.Fatalf("invalid --tgt base %q", base[1])
		}
		lowest, err := strconv.ParseFloat(scale[1], 64)
		if err != nil || lowest <= 0 {
			t.Fatalf("invalid --s lower bound %q", scale[1])
		}
		got := int(float64(px)*lowest + 0.5)
		t.Logf("WINDOWS-UI reference: MinTarget=%d, launcher compact pointer target=%d; runtime acceptance remains separate", floor, got)
		return
	}
	t.Fatal("reference lacks ChromeMetrics MinTarget")
}
