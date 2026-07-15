//go:build windows

package windowsreg

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows/registry"
)

// SupportedExtensions lists all file extensions registered by this program.
var SupportedExtensions = []string{".epub", ".pdf", ".txt", ".md", ".fb2", ".rtf", ".html", ".htm", ".mobi", ".azw3"}

// legacyProgIDs are old ProgID names left from previous versions; cleaned up on every registration.
var legacyProgIDs = []string{"epub2html"}

// progID is the shared ProgID this app registers itself under when it becomes the
// default handler (RegisterHandler). Unregister and IsDefaultHandler compare against it.
const progID = "doc-html-translate"

// contextMenuVerb is the HKCU shell-verb key name for the non-destructive
// "Convert to HTML" right-click command written by RegisterContextMenu. It lives under
// SystemFileAssociations so it attaches to the file *type* and is available no matter
// which app owns that type's default association.
const contextMenuVerb = "dochtmltranslate.convert"

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
		if err := registerExtension(ext); err != nil {
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

// RegisterOpenWith advertises the current executable in the Windows "Open with"
// list for all SupportedExtensions, without making it the default handler.
func RegisterOpenWith() ([]string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("resolve executable path: %w", err)
	}
	return RegisterOpenWithFor(exePath)
}

// RegisterOpenWithFor advertises exePath in the Windows "Open with" list for all
// SupportedExtensions without touching any extension's default handler. Unlike
// RegisterHandler this is non-destructive: it never sets a default ProgID, so each
// file type's existing default association is left alone - the app just becomes one
// more choice in the right-click "Open with" menu. It writes, under HKCU (no admin
// rights needed):
//
//	Software\Classes\Applications\<exe>\
//	    FriendlyAppName              = "DOC-HTML-TRANSLATE"
//	    DefaultIcon\(default)        = "<exe>",0
//	    shell\open\command\(default) = "<exe>" "%1"
//	    SupportedTypes\<ext>         = ""   (one empty value per extension)
//
// The SupportedTypes list is exactly what makes Explorer offer the app under "Open
// with" for those file types. The call is idempotent - safe to run on every launch,
// and it re-points the command when a portable exe moves. Returns the advertised
// extensions.
func RegisterOpenWithFor(exePath string) ([]string, error) {
	appKeyPath := `Software\Classes\Applications\` + filepath.Base(exePath)

	appKey, _, err := registry.CreateKey(registry.CURRENT_USER, appKeyPath, registry.SET_VALUE)
	if err != nil {
		return nil, fmt.Errorf("create application key: %w", err)
	}
	defer appKey.Close()
	if err := appKey.SetStringValue("FriendlyAppName", "DOC-HTML-TRANSLATE"); err != nil {
		return nil, fmt.Errorf("set FriendlyAppName: %w", err)
	}

	iconKey, _, err := registry.CreateKey(registry.CURRENT_USER, appKeyPath+`\DefaultIcon`, registry.SET_VALUE)
	if err != nil {
		return nil, fmt.Errorf("create default icon key: %w", err)
	}
	defer iconKey.Close()
	if err := iconKey.SetStringValue("", fmt.Sprintf("\"%s\",0", exePath)); err != nil {
		return nil, fmt.Errorf("set default icon value: %w", err)
	}

	commandKey, _, err := registry.CreateKey(registry.CURRENT_USER, appKeyPath+`\shell\open\command`, registry.SET_VALUE)
	if err != nil {
		return nil, fmt.Errorf("create open command key: %w", err)
	}
	defer commandKey.Close()
	if err := commandKey.SetStringValue("", fmt.Sprintf("\"%s\" \"%%1\"", exePath)); err != nil {
		return nil, fmt.Errorf("set open command value: %w", err)
	}

	typesKey, _, err := registry.CreateKey(registry.CURRENT_USER, appKeyPath+`\SupportedTypes`, registry.SET_VALUE)
	if err != nil {
		return nil, fmt.Errorf("create supported types key: %w", err)
	}
	defer typesKey.Close()

	var advertised []string
	for _, ext := range SupportedExtensions {
		if err := typesKey.SetStringValue(ext, ""); err != nil {
			fmt.Fprintf(os.Stderr, "WARNING: failed to advertise %s: %v\n", ext, err)
			continue
		}
		advertised = append(advertised, ext)
	}
	if len(advertised) == 0 {
		return nil, fmt.Errorf("failed to advertise any extensions")
	}
	return advertised, nil
}

// RegisterContextMenu adds the "Convert to HTML" right-click verb for the current
// executable across all SupportedExtensions. See RegisterContextMenuFor.
func RegisterContextMenu() ([]string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("resolve executable path: %w", err)
	}
	return RegisterContextMenuFor(exePath)
}

// RegisterContextMenuFor adds a per-user "Convert to HTML" right-click command for
// exePath to every SupportedExtensions type, WITHOUT changing any default handler. It
// writes, under HKCU (no admin rights needed):
//
//	Software\Classes\SystemFileAssociations\<ext>\shell\dochtmltranslate.convert\
//	    MUIVerb                = "Convert to HTML"
//	    Icon                   = "<exe>",0
//	    command\(default)      = "<exe>" "%1"
//
// SystemFileAssociations verbs attach to the file *type*, so the entry appears in the
// right-click menu regardless of which app owns the default association - exactly the
// "always reachable, never the default" behaviour we want. The call is idempotent - safe
// to run on every launch, and it re-points the command when a portable exe moves. Returns
// the extensions the verb was added to.
func RegisterContextMenuFor(exePath string) ([]string, error) {
	command := fmt.Sprintf("\"%s\" \"%%1\"", exePath)
	icon := fmt.Sprintf("\"%s\",0", exePath)

	var added []string
	for _, ext := range SupportedExtensions {
		verbPath := `Software\Classes\SystemFileAssociations\` + ext + `\shell\` + contextMenuVerb
		verbKey, _, err := registry.CreateKey(registry.CURRENT_USER, verbPath, registry.SET_VALUE)
		if err != nil {
			fmt.Fprintf(os.Stderr, "WARNING: failed to add context menu for %s: %v\n", ext, err)
			continue
		}
		_ = verbKey.SetStringValue("MUIVerb", "Convert to HTML")
		_ = verbKey.SetStringValue("Icon", icon)
		verbKey.Close()

		cmdKey, _, err := registry.CreateKey(registry.CURRENT_USER, verbPath+`\command`, registry.SET_VALUE)
		if err != nil {
			fmt.Fprintf(os.Stderr, "WARNING: failed to add context menu command for %s: %v\n", ext, err)
			continue
		}
		err = cmdKey.SetStringValue("", command)
		cmdKey.Close()
		if err != nil {
			fmt.Fprintf(os.Stderr, "WARNING: failed to set context menu command for %s: %v\n", ext, err)
			continue
		}
		added = append(added, ext)
	}
	if len(added) == 0 {
		return nil, fmt.Errorf("failed to add any context menu entries")
	}
	return added, nil
}

// Unregister releases the default-handler association created by RegisterHandler, leaving
// the non-destructive right-click verb and "Open with" advertisement in place. For each
// SupportedExtensions type whose HKCU default ProgID is ours, it deletes that default
// value so the file type falls back to its previous handler; it never touches an extension
// pointed at some other app, nor the ProgID definition or the "Convert to HTML" verb.
// Returns the extensions that were released.
func Unregister() ([]string, error) {
	var released []string
	for _, ext := range SupportedExtensions {
		k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Classes\`+ext, registry.QUERY_VALUE|registry.SET_VALUE)
		if err != nil {
			continue // never associated - nothing to release
		}
		cur, _, err := k.GetStringValue("")
		if err == nil && cur == progID {
			_ = k.DeleteValue("")
			released = append(released, ext)
		}
		k.Close()
	}
	return released, nil
}

// IsDefaultHandler reports whether this program is currently the HKCU default handler for
// every SupportedExtensions type (RegisterHandler in effect and not undone by Unregister).
// The GUI uses it to reflect the association toggle's on/off state.
func IsDefaultHandler() bool {
	for _, ext := range SupportedExtensions {
		k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Classes\`+ext, registry.QUERY_VALUE)
		if err != nil {
			return false
		}
		cur, _, err := k.GetStringValue("")
		k.Close()
		if err != nil || cur != progID {
			return false
		}
	}
	return true
}

// registerExtension associates a file extension with our ProgID in HKCU.
func registerExtension(ext string) error {
	extKey, _, err := registry.CreateKey(registry.CURRENT_USER, `Software\Classes\`+ext, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("create extension key: %w", err)
	}
	defer extKey.Close()

	return extKey.SetStringValue("", progID)
}
