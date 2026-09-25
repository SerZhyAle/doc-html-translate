//go:build windows

package windowsreg

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// SupportedExtensions lists all file extensions registered by this program.
var SupportedExtensions = []string{".epub", ".pdf", ".txt", ".md", ".fb2", ".rtf", ".html", ".htm", ".mobi", ".azw3", ".cbz", ".cbr", ".cb7", ".cbt"}

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

// regKey is the part of registry.Key this package uses; a test substitutes an in-memory
// key so it never writes to the real HKCU hive.
type regKey interface {
	SetStringValue(name, value string) error
	GetStringValue(name string) (string, uint32, error)
	DeleteValue(name string) error
	Close() error
}

// createKey, openKey and deleteKey are indirected so a test can run registration
// against a fake HKCU. Paths are relative to HKCU.
var (
	createKey = func(path string) (regKey, error) {
		// QUERY_VALUE too: a write reads the old value first, to save it or to skip a no-op.
		k, _, err := registry.CreateKey(registry.CURRENT_USER, path, registry.SET_VALUE|registry.QUERY_VALUE)
		return k, err
	}
	openKey = func(path string, access uint32) (regKey, error) {
		return registry.OpenKey(registry.CURRENT_USER, path, access)
	}
	deleteKey = func(path string) error {
		return registry.DeleteKey(registry.CURRENT_USER, path)
	}
)

// notifyAssocChanged and openSettings are indirected so a test neither refreshes the real
// Explorer nor opens a Settings window.
var (
	// Without SHCNE_ASSOCCHANGED Explorer keeps showing the old icons and handler until it
	// restarts, so a correct write still looks like a failure.
	notifyAssocChanged = func() {
		_, _, _ = procSHChangeNotify.Call(shcneAssocChanged, shcnfIDList, 0, 0)
	}
	openSettings = func() error {
		verb, _ := windows.UTF16PtrFromString("open")
		uri, _ := windows.UTF16PtrFromString("ms-settings:defaultapps")
		return windows.ShellExecute(0, verb, uri, nil, nil, windows.SW_SHOWNORMAL)
	}
)

var procSHChangeNotify = windows.NewLazySystemDLL("shell32.dll").NewProc("SHChangeNotify")

const (
	shcneAssocChanged = 0x08000000
	shcnfIDList       = 0x0000
)

// backupValue holds, inside Software\Classes\<ext>, the per-user handler that was the
// default before RegisterHandler replaced it, so Unregister can hand the type back.
const backupValue = "doc-html-translate.previous"

// cliExeName is the converter's file name. "Open with -> Always" on it records the choice as
// Applications\<exe>, which is this app as much as progID is.
const cliExeName = "doc-html-translate.exe"

// fileExtsPath is where Explorer keeps the user's own per-extension choice.
const fileExtsPath = `Software\Microsoft\Windows\CurrentVersion\Explorer\FileExts\`

// changeTracker remembers whether a write actually changed a value, so an idempotent rewrite
// on every launch does not make Explorer rebuild its icon cache each time.
type changeTracker struct{ changed bool }

func (c *changeTracker) set(k regKey, name, value string) error {
	if cur, _, err := k.GetStringValue(name); err == nil && cur == value {
		return nil
	}
	if err := k.SetStringValue(name, value); err != nil {
		return err
	}
	c.changed = true
	return nil
}

// notifyIfChanged is deferred by every writer: a partial write still changed something.
func (c *changeTracker) notifyIfChanged() {
	if c.changed {
		notifyAssocChanged()
	}
}

// RegisterHandler registers the program as the HKCU handler for all SupportedExtensions and
// reports, per extension, whether Windows now actually uses it. The error is set only when
// nothing could be registered; a partial result is described by the Registration.
func RegisterHandler() (Registration, error) {
	exePath, err := os.Executable()
	if err != nil {
		return Registration{}, fmt.Errorf("resolve executable path: %w", err)
	}
	var w changeTracker
	defer w.notifyIfChanged()

	// Remove stale ProgID keys from previous versions.
	for _, legacy := range legacyProgIDs {
		for _, sub := range []string{`\shell\open\command`, `\shell\open`, `\shell`, `\DefaultIcon`, ``} {
			if deleteKey(`Software\Classes\`+legacy+sub) == nil {
				w.changed = true
			}
		}
	}

	command := fmt.Sprintf("\"%s\" \"%%1\"", exePath)
	// Icon is embedded in the exe - reference it directly as resource index 0.
	defaultIconValue := fmt.Sprintf("\"%s\",0", exePath)

	// Create ProgID key once (shared across all extensions)
	progKeyPath := `Software\Classes\` + progID
	progKey, err := createKey(progKeyPath)
	if err != nil {
		return Registration{}, fmt.Errorf("create progid key: %w", err)
	}
	defer progKey.Close()

	if err := w.set(progKey, "", "DOC-HTML-TRANSLATE Document"); err != nil {
		return Registration{}, fmt.Errorf("set progid description: %w", err)
	}

	iconKey, err := createKey(progKeyPath + `\DefaultIcon`)
	if err != nil {
		return Registration{}, fmt.Errorf("create default icon key: %w", err)
	}
	defer iconKey.Close()

	if err := w.set(iconKey, "", defaultIconValue); err != nil {
		return Registration{}, fmt.Errorf("set default icon value: %w", err)
	}

	commandKey, err := createKey(progKeyPath + `\shell\open\command`)
	if err != nil {
		return Registration{}, fmt.Errorf("create open command key: %w", err)
	}
	defer commandKey.Close()

	if err := w.set(commandKey, "", command); err != nil {
		return Registration{}, fmt.Errorf("set open command value: %w", err)
	}

	var reg Registration
	for _, ext := range SupportedExtensions {
		if err := registerExtension(&w, ext); err != nil {
			reg.Failed = append(reg.Failed, ext)
			continue
		}
		switch extState(ext) {
		case stateDefault:
			reg.Default = append(reg.Default, ext)
		case stateBlocked:
			reg.Blocked = append(reg.Blocked, ext)
		default:
			reg.Unknown = append(reg.Unknown, ext)
		}
	}

	if len(reg.Failed) == len(SupportedExtensions) {
		return reg, fmt.Errorf("failed to register any extensions")
	}
	return reg, nil
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
	var w changeTracker
	defer w.notifyIfChanged()

	appKey, err := createKey(appKeyPath)
	if err != nil {
		return nil, fmt.Errorf("create application key: %w", err)
	}
	defer appKey.Close()
	if err := w.set(appKey, "FriendlyAppName", "DOC-HTML-TRANSLATE"); err != nil {
		return nil, fmt.Errorf("set FriendlyAppName: %w", err)
	}

	iconKey, err := createKey(appKeyPath + `\DefaultIcon`)
	if err != nil {
		return nil, fmt.Errorf("create default icon key: %w", err)
	}
	defer iconKey.Close()
	if err := w.set(iconKey, "", fmt.Sprintf("\"%s\",0", exePath)); err != nil {
		return nil, fmt.Errorf("set default icon value: %w", err)
	}

	commandKey, err := createKey(appKeyPath + `\shell\open\command`)
	if err != nil {
		return nil, fmt.Errorf("create open command key: %w", err)
	}
	defer commandKey.Close()
	if err := w.set(commandKey, "", fmt.Sprintf("\"%s\" \"%%1\"", exePath)); err != nil {
		return nil, fmt.Errorf("set open command value: %w", err)
	}

	typesKey, err := createKey(appKeyPath + `\SupportedTypes`)
	if err != nil {
		return nil, fmt.Errorf("create supported types key: %w", err)
	}
	defer typesKey.Close()

	var advertised []string
	for _, ext := range SupportedExtensions {
		if err := w.set(typesKey, ext, ""); err != nil {
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
	var w changeTracker
	defer w.notifyIfChanged()

	var added []string
	for _, ext := range SupportedExtensions {
		verbPath := contextMenuVerbPath(ext)
		verbKey, err := createKey(verbPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "WARNING: failed to add context menu for %s: %v\n", ext, err)
			continue
		}
		// The label and icon are cosmetic: Explorer falls back to the verb name and no icon.
		_ = w.set(verbKey, "MUIVerb", "Convert to HTML")
		_ = w.set(verbKey, "Icon", icon)
		verbKey.Close()

		cmdKey, err := createKey(verbPath + `\command`)
		if err != nil {
			fmt.Fprintf(os.Stderr, "WARNING: failed to add context menu command for %s: %v\n", ext, err)
			continue
		}
		err = w.set(cmdKey, "", command)
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

// HasShellEntries reports whether the "Convert to HTML" right-click verb is registered for
// at least one SupportedExtensions type. It is the on/off state of the GUI's shell-entry
// toggle and what decides whether the first-run question is still worth asking.
func HasShellEntries() bool {
	for _, ext := range SupportedExtensions {
		k, err := openKey(contextMenuVerbPath(ext)+`\command`, registry.QUERY_VALUE)
		if err == nil {
			k.Close()
			return true
		}
	}
	return false
}

// RemoveShellEntries takes back what RegisterOpenWithFor and RegisterContextMenuFor wrote for
// the converter: the "Convert to HTML" verb of every type and the converter's SupportedTypes
// list, which is what puts it under "Open with". The Applications\<exe> key itself stays,
// because a user who picked the app with "Open with -> Always" chose that key as the handler,
// and deleting it would break a choice that is theirs. Returns the extensions whose verb was
// removed; the error names the ones that could not be.
func RemoveShellEntries() ([]string, error) {
	var w changeTracker
	defer w.notifyIfChanged()

	var removed, failed []string
	for _, ext := range SupportedExtensions {
		verbPath := contextMenuVerbPath(ext)
		cmdErr := deleteKey(verbPath + `\command`)
		verbErr := deleteKey(verbPath)
		gone := func(err error) bool { return err == nil || errors.Is(err, registry.ErrNotExist) }
		switch {
		case !gone(cmdErr) || !gone(verbErr):
			failed = append(failed, ext)
		case cmdErr == nil || verbErr == nil:
			w.changed = true
			removed = append(removed, ext)
		}
	}

	for _, exe := range openWithExeNames() {
		k, err := openKey(`Software\Classes\Applications\`+exe+`\SupportedTypes`, registry.SET_VALUE|registry.QUERY_VALUE)
		if errors.Is(err, registry.ErrNotExist) {
			continue
		}
		if err != nil {
			failed = append(failed, `Applications\`+exe)
			continue
		}
		for _, ext := range SupportedExtensions {
			if _, _, err := k.GetStringValue(ext); err != nil {
				continue
			}
			if err := k.DeleteValue(ext); err != nil {
				failed = append(failed, `Applications\`+exe+` `+ext)
				continue
			}
			w.changed = true
		}
		k.Close()
	}

	if len(failed) > 0 {
		return removed, fmt.Errorf("could not remove %s", strings.Join(failed, ", "))
	}
	return removed, nil
}

// contextMenuVerbPath is the HKCU key of the "Convert to HTML" verb for one extension.
func contextMenuVerbPath(ext string) string {
	return `Software\Classes\SystemFileAssociations\` + ext + `\shell\` + contextMenuVerb
}

// openWithExeNames are the Applications\<exe> names the converter may have advertised itself
// under: its shipped name, and the name of this executable when it was renamed.
func openWithExeNames() []string {
	names := []string{cliExeName}
	if exe, err := os.Executable(); err == nil {
		if base := filepath.Base(exe); !strings.EqualFold(base, cliExeName) {
			names = append(names, base)
		}
	}
	return names
}

// Unregister releases the default-handler association created by RegisterHandler, leaving
// the non-destructive right-click verb and "Open with" advertisement in place. For each
// SupportedExtensions type whose HKCU default ProgID is ours, it puts back the handler saved
// at registration, or deletes the value when none was saved (nothing was there, or the
// registration predates the backup). It never touches an extension pointed at some other app,
// nor the ProgID definition or the "Convert to HTML" verb, nor the user's own choice in
// Windows, which only the user may change. Returns the extensions that were released; the
// error names the ones that could not be.
func Unregister() ([]string, error) {
	var w changeTracker
	defer w.notifyIfChanged()

	var released, failed []string
	for _, ext := range SupportedExtensions {
		k, err := openKey(`Software\Classes\`+ext, registry.QUERY_VALUE|registry.SET_VALUE)
		if errors.Is(err, registry.ErrNotExist) {
			continue // never associated - nothing to release
		}
		if err != nil {
			failed = append(failed, ext)
			continue
		}
		ok, err := releaseExtension(k)
		k.Close()
		switch {
		case err != nil:
			failed = append(failed, ext)
		case ok:
			w.changed = true
			released = append(released, ext)
		}
	}
	if len(failed) > 0 {
		return released, fmt.Errorf("could not release %s", strings.Join(failed, ", "))
	}
	return released, nil
}

// releaseExtension hands one extension key back and reports whether it was ours.
func releaseExtension(k regKey) (bool, error) {
	prev, _, prevErr := k.GetStringValue(backupValue)
	hasBackup := prevErr == nil
	cur, _, err := k.GetStringValue("")
	if err != nil || cur != progID {
		// Someone else owns the type now. A kept backup would, on some later unregister,
		// resurrect a handler the user has since moved away from.
		if hasBackup {
			return false, k.DeleteValue(backupValue)
		}
		return false, nil
	}
	if hasBackup && prev != "" && prev != progID {
		err = k.SetStringValue("", prev)
	} else {
		err = k.DeleteValue("")
	}
	if err != nil {
		return false, err
	}
	if hasBackup {
		if err := k.DeleteValue(backupValue); err != nil {
			return false, err
		}
	}
	return true, nil
}

// IsDefaultHandler reports whether Windows opens every SupportedExtensions type with this
// program. The GUI uses it to reflect the association toggle's on/off state.
func IsDefaultHandler() bool {
	return HandlerStatus().IsDefault()
}

// HandlerStatus reports, per SupportedExtensions type, which handler Windows actually uses.
func HandlerStatus() Status {
	var s Status
	for _, ext := range SupportedExtensions {
		switch extState(ext) {
		case stateDefault:
			s.Default = append(s.Default, ext)
		case stateBlocked:
			s.Blocked = append(s.Blocked, ext)
		case stateUnknown:
			s.Unknown = append(s.Unknown, ext)
		default:
			s.Other = append(s.Other, ext)
		}
	}
	return s
}

// OpenDefaultAppsSettings opens Settings > Apps > Default apps, the only place the user's own
// choice may be changed: Windows protects it with a hash no other program may forge.
func OpenDefaultAppsSettings() error {
	return openSettings()
}

type handlerState int

const (
	stateOther handlerState = iota
	stateDefault
	stateBlocked
	stateUnknown
)

// extState decides which handler Windows uses for ext. The user's choice, when recorded,
// wins; without one the per-user class key does.
func extState(ext string) handlerState {
	registered := false
	if k, err := openKey(`Software\Classes\`+ext, registry.QUERY_VALUE); err == nil {
		cur, _, err := k.GetStringValue("")
		k.Close()
		registered = err == nil && cur == progID
	}
	choice, found, err := userChoice(ext)
	switch {
	case err != nil:
		return stateUnknown
	case !found:
		if registered {
			return stateDefault
		}
		return stateOther
	case isOurProgID(choice):
		return stateDefault
	case registered:
		return stateBlocked
	}
	return stateOther
}

// userChoice reads the ProgID the user picked for ext. Windows 11 24H2 added UserChoiceLatest
// and prefers it over UserChoice; its ProgId sits either on the key or in a ProgId subkey,
// depending on the build.
func userChoice(ext string) (id string, found bool, err error) {
	base := fileExtsPath + ext
	if id, found, err := readProgID(base+`\UserChoiceLatest`, base+`\UserChoiceLatest\ProgId`); found || err != nil {
		return id, found, err
	}
	return readProgID(base + `\UserChoice`)
}

// readProgID reads the ProgId value of key, then of each alt key. A missing key means no
// choice; a key that exists but yields no ProgId is an error, so the caller says "unknown"
// rather than guessing.
func readProgID(key string, alt ...string) (string, bool, error) {
	k, err := openKey(key, registry.QUERY_VALUE)
	if errors.Is(err, registry.ErrNotExist) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	id, _, err := k.GetStringValue("ProgId")
	k.Close()
	if err == nil && id != "" {
		return id, true, nil
	}
	for _, p := range alt {
		k, err := openKey(p, registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		id, _, err := k.GetStringValue("ProgId")
		k.Close()
		if err == nil && id != "" {
			return id, true, nil
		}
	}
	return "", false, fmt.Errorf("%s holds no readable ProgId", key)
}

func isOurProgID(id string) bool {
	if strings.EqualFold(id, progID) || strings.EqualFold(id, `Applications\`+cliExeName) {
		return true
	}
	// A renamed portable exe advertises itself under its own name.
	exe, err := os.Executable()
	return err == nil && strings.EqualFold(id, `Applications\`+filepath.Base(exe))
}

// registerExtension associates a file extension with our ProgID in HKCU, first saving the
// handler it replaces so Unregister can put it back.
func registerExtension(w *changeTracker, ext string) error {
	extKey, err := createKey(`Software\Classes\` + ext)
	if err != nil {
		return fmt.Errorf("create extension key: %w", err)
	}
	defer extKey.Close()

	cur, _, err := extKey.GetStringValue("")
	switch {
	case errors.Is(err, registry.ErrNotExist):
	case err != nil:
		// Overwriting a value that cannot be read would lose it for good.
		return fmt.Errorf("read current handler: %w", err)
	case cur != "" && cur != progID:
		if err := w.set(extKey, backupValue, cur); err != nil {
			return fmt.Errorf("save previous handler: %w", err)
		}
	}
	return w.set(extKey, "", progID)
}
