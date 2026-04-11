//go:build windows

package windowsreg

import (
	"fmt"
	"os"

	"golang.org/x/sys/windows/registry"
)

// SupportedExtensions lists all file extensions registered by this program.
var SupportedExtensions = []string{".epub", ".pdf", ".txt", ".md", ".fb2", ".rtf", ".html", ".htm", ".mobi", ".azw3"}

// legacyProgIDs are old ProgID names left from previous versions; cleaned up on every registration.
var legacyProgIDs = []string{"epub2html"}

// RegisterHandler registers the program as the HKCU handler for all SupportedExtensions.
// Returns the list of successfully registered extensions.
func RegisterHandler() ([]string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("resolve executable path: %w", err)
	}

	// Remove stale ProgID keys from previous versions.
	for _, legacy := range legacyProgIDs {
		_ = registry.DeleteKey(registry.CURRENT_USER, `Software\Classes\`+legacy+`\shell\open\command`)
		_ = registry.DeleteKey(registry.CURRENT_USER, `Software\Classes\`+legacy+`\shell\open`)
		_ = registry.DeleteKey(registry.CURRENT_USER, `Software\Classes\`+legacy+`\shell`)
		_ = registry.DeleteKey(registry.CURRENT_USER, `Software\Classes\`+legacy+`\DefaultIcon`)
		_ = registry.DeleteKey(registry.CURRENT_USER, `Software\Classes\`+legacy)
	}

	progID := "doc-html-translate"
	command := fmt.Sprintf("\"%s\" \"%%1\"", exePath)
	// Icon is embedded in the exe — reference it directly as resource index 0.
	defaultIconValue := fmt.Sprintf("\"%s\",0", exePath)

	// Create ProgID key once (shared across all extensions)
	progKeyPath := `Software\Classes\` + progID
	progKey, _, err := registry.CreateKey(registry.CURRENT_USER, progKeyPath, registry.SET_VALUE)
	if err != nil {
		return nil, fmt.Errorf("create progid key: %w", err)
	}
	defer progKey.Close()

	if err := progKey.SetStringValue("", "DOC-HTML-TRANSLATE Document"); err != nil {
		return nil, fmt.Errorf("set progid description: %w", err)
	}

	iconKey, _, err := registry.CreateKey(registry.CURRENT_USER, progKeyPath+`\DefaultIcon`, registry.SET_VALUE)
	if err != nil {
		return nil, fmt.Errorf("create default icon key: %w", err)
	}
	defer iconKey.Close()

	if err := iconKey.SetStringValue("", defaultIconValue); err != nil {
		return nil, fmt.Errorf("set default icon value: %w", err)
	}

	commandKey, _, err := registry.CreateKey(registry.CURRENT_USER, progKeyPath+`\shell\open\command`, registry.SET_VALUE)
	if err != nil {
		return nil, fmt.Errorf("create open command key: %w", err)
	}
	defer commandKey.Close()

	if err := commandKey.SetStringValue("", command); err != nil {
		return nil, fmt.Errorf("set open command value: %w", err)
	}

	// Register each extension
	var registered []string
	for _, ext := range SupportedExtensions {
		if err := registerExtension(ext, progID); err != nil {
			fmt.Fprintf(os.Stderr, "WARNING: failed to register %s: %v\n", ext, err)
			continue
		}
		registered = append(registered, ext)
	}

	if len(registered) == 0 {
		return nil, fmt.Errorf("failed to register any extensions")
	}

	return registered, nil
}

// registerExtension associates a file extension with the given ProgID in HKCU.
func registerExtension(ext, progID string) error {
	extKey, _, err := registry.CreateKey(registry.CURRENT_USER, `Software\Classes\`+ext, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("create extension key: %w", err)
	}
	defer extKey.Close()

	return extKey.SetStringValue("", progID)
}
