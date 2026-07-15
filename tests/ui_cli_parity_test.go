package tests

// Enforces the ui-cli-parity invariant by code: the GUI (cmd/doc-html-ui) must expose
// every CLI flag defined in internal/config/flags.go, either by forwarding it in
// assembleArgs or through a GUI-native endpoint/button (allow-listed below with a
// reason). See docs/PARITY.md "Settings defaults".

import (
	"regexp"
	"strings"
	"testing"
)

func TestParityGUIExposesEveryCLIFlag(t *testing.T) {
	flagsSrc := readRepoFile(t, "internal", "config", "flags.go")
	guiSrc := readRepoFile(t, "cmd", "doc-html-ui", "main.go")

	// Flags the GUI handles through its own endpoints/buttons rather than by forwarding
	// the CLI flag to the converter.
	guiNative := map[string]string{
		"register":          `"Make default handler" toggle (on) + /api/register`,
		"unregister":        `"Make default handler" toggle (off) + /api/unregister`,
		"register-openwith": `auto-run on GUI startup via ensureRightClickRegistered (adds "Open with" + "Convert to HTML" right-click entry without setting a default)`,
		"version":           `/api/version`,
		"ocr-langs":         `/api/ocr-langs`,
		"ocr-download":      `/api/ocr-download`,
		"free":              `alias of -ollama`,
	}

	re := regexp.MustCompile(`fs\.(?:Bool|String|Int|Float64)\("([a-z0-9-]+)"`)
	ms := re.FindAllStringSubmatch(flagsSrc, -1)
	if len(ms) == 0 {
		t.Fatal("no flags parsed from flags.go")
	}
	for _, m := range ms {
		name := m[1]
		if _, ok := guiNative[name]; ok {
			continue
		}
		if !strings.Contains(guiSrc, `"-`+name+`"`) {
			t.Errorf("CLI flag -%s is not surfaced by the GUI (cmd/doc-html-ui/main.go). "+
				"The GUI must expose every CLI flag; forward it in assembleArgs or add it to guiNative "+
				"with a reason - see docs/PARITY.md.", name)
		}
	}
}
