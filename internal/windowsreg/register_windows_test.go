//go:build windows

package windowsreg

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"

	"golang.org/x/sys/windows/registry"
)

// fakeHKCU stands in for the real hive so no test touches the user's file associations.
// Key paths are compared case-insensitively, as the registry does.
type fakeHKCU struct {
	keys       map[string]map[string]string
	failCreate func(path string) bool
	failSet    func(path, name string) bool
	failDelete func(path, name string) bool
	notified   int
	settings   int
}

type fakeKey struct {
	hive *fakeHKCU
	path string
}

func (k fakeKey) SetStringValue(name, value string) error {
	if k.hive.failSet != nil && k.hive.failSet(k.path, name) {
		return errors.New("access denied")
	}
	k.hive.keys[k.path][name] = value
	return nil
}

func (k fakeKey) GetStringValue(name string) (string, uint32, error) {
	v, ok := k.hive.keys[k.path][name]
	if !ok {
		return "", 0, registry.ErrNotExist
	}
	return v, registry.SZ, nil
}

func (k fakeKey) DeleteValue(name string) error {
	if k.hive.failDelete != nil && k.hive.failDelete(k.path, name) {
		return errors.New("access denied")
	}
	delete(k.hive.keys[k.path], name)
	return nil
}

func (k fakeKey) Close() error { return nil }

// useFakeHKCU swaps the registry seams for an empty fake for the duration of the test.
func useFakeHKCU(t *testing.T) *fakeHKCU {
	t.Helper()
	h := &fakeHKCU{keys: map[string]map[string]string{}}
	origCreate, origOpen, origDelete := createKey, openKey, deleteKey
	origNotify, origSettings := notifyAssocChanged, openSettings
	t.Cleanup(func() {
		createKey, openKey, deleteKey = origCreate, origOpen, origDelete
		notifyAssocChanged, openSettings = origNotify, origSettings
	})
	notifyAssocChanged = func() { h.notified++ }
	openSettings = func() error { h.settings++; return nil }

	createKey = func(path string) (regKey, error) {
		p := strings.ToLower(path)
		if h.failCreate != nil && h.failCreate(p) {
			return nil, errors.New("access denied")
		}
		if h.keys[p] == nil {
			h.keys[p] = map[string]string{}
		}
		return fakeKey{hive: h, path: p}, nil
	}
	openKey = func(path string, _ uint32) (regKey, error) {
		p := strings.ToLower(path)
		if h.keys[p] == nil {
			return nil, registry.ErrNotExist
		}
		return fakeKey{hive: h, path: p}, nil
	}
	deleteKey = func(path string) error {
		p := strings.ToLower(path)
		if h.keys[p] == nil {
			return registry.ErrNotExist
		}
		delete(h.keys, p)
		return nil
	}
	return h
}

func (h *fakeHKCU) value(path, name string) (string, bool) {
	v, ok := h.keys[strings.ToLower(path)][name]
	return v, ok
}

func (h *fakeHKCU) set(path, name, value string) {
	p := strings.ToLower(path)
	if h.keys[p] == nil {
		h.keys[p] = map[string]string{}
	}
	h.keys[p][name] = value
}

func TestRegisterHandlerMakesAppTheDefault(t *testing.T) {
	h := useFakeHKCU(t)
	for _, k := range []string{`epub2html`, `epub2html\shell`, `epub2html\shell\open`, `epub2html\shell\open\command`, `epub2html\DefaultIcon`} {
		h.set(`Software\Classes\`+k, "", "stale")
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}

	got, err := RegisterHandler()
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got.Default, SupportedExtensions) || !got.Complete() {
		t.Errorf("registered %+v, want every extension default", got)
	}
	if h.notified != 1 {
		t.Errorf("Explorer notified %d times, want once", h.notified)
	}
	for _, ext := range SupportedExtensions {
		if v, _ := h.value(`Software\Classes\`+ext, ""); v != progID {
			t.Errorf("%s default = %q, want %q", ext, v, progID)
		}
	}
	// The quoted %1 is what lets Explorer pass a path with spaces as one argument.
	if v, _ := h.value(`Software\Classes\`+progID+`\shell\open\command`, ""); v != fmt.Sprintf(`"%s" "%%1"`, exe) {
		t.Errorf("open command = %q", v)
	}
	if v, _ := h.value(`Software\Classes\`+progID+`\DefaultIcon`, ""); v != fmt.Sprintf(`"%s",0`, exe) {
		t.Errorf("icon = %q", v)
	}
	for p := range h.keys {
		if strings.Contains(p, "epub2html") {
			t.Errorf("legacy key %s survived registration", p)
		}
	}
	if !IsDefaultHandler() {
		t.Error("IsDefaultHandler = false right after RegisterHandler")
	}
}

func TestRegisterHandlerFailures(t *testing.T) {
	t.Run("progid key cannot be created", func(t *testing.T) {
		h := useFakeHKCU(t)
		h.failCreate = func(p string) bool { return p == strings.ToLower(`Software\Classes\`+progID) }
		got, err := RegisterHandler()
		if err == nil || len(got.Default) != 0 {
			t.Fatalf("got %+v, %v; want an error and nothing registered", got, err)
		}
		if _, ok := h.value(`Software\Classes\.epub`, ""); ok {
			t.Error(".epub was associated although the ProgID it points at was never written")
		}
	})
	t.Run("one extension fails", func(t *testing.T) {
		h := useFakeHKCU(t)
		h.failCreate = func(p string) bool { return p == `software\classes\.pdf` }
		got, err := RegisterHandler()
		if err != nil {
			t.Fatal(err)
		}
		if slices.Contains(got.Default, ".pdf") || len(got.Default) != len(SupportedExtensions)-1 {
			t.Errorf("registered %+v, want every extension but .pdf", got)
		}
		if !slices.Equal(got.Failed, []string{".pdf"}) || got.Complete() {
			t.Errorf("failed = %v, want [.pdf] and an incomplete result", got.Failed)
		}
		if IsDefaultHandler() {
			t.Error("IsDefaultHandler = true with .pdf unassociated")
		}
	})
	t.Run("every extension fails", func(t *testing.T) {
		h := useFakeHKCU(t)
		h.failCreate = func(p string) bool { return strings.HasPrefix(p, `software\classes\.`) }
		got, err := RegisterHandler()
		if err == nil || !slices.Equal(got.Failed, SupportedExtensions) {
			t.Fatalf("got %+v, %v; want an error with every extension failed", got, err)
		}
	})
}

// Unregister must give back only what is ours: a type another app owns keeps its handler,
// and the ProgID and the non-destructive integrations stay.
func TestUnregisterReleasesOnlyOurAssociations(t *testing.T) {
	h := useFakeHKCU(t)
	if _, err := RegisterHandler(); err != nil {
		t.Fatal(err)
	}
	if _, err := RegisterContextMenuFor(`C:\app\doc-html-translate.exe`); err != nil {
		t.Fatal(err)
	}
	h.set(`Software\Classes\.pdf`, "", "AcroExch.Document")

	released, err := Unregister()
	if err != nil {
		t.Fatal(err)
	}
	if slices.Contains(released, ".pdf") || len(released) != len(SupportedExtensions)-1 {
		t.Errorf("released %v, want every extension but .pdf", released)
	}
	if v, _ := h.value(`Software\Classes\.pdf`, ""); v != "AcroExch.Document" {
		t.Errorf(".pdf default = %q, want the other app's ProgID kept", v)
	}
	if _, ok := h.value(`Software\Classes\.epub`, ""); ok {
		t.Error(".epub still points at us")
	}
	if _, ok := h.value(`Software\Classes\`+progID+`\shell\open\command`, ""); !ok {
		t.Error("the ProgID definition was removed")
	}
	if _, ok := h.value(`Software\Classes\SystemFileAssociations\.epub\shell\`+contextMenuVerb+`\command`, ""); !ok {
		t.Error("the context menu verb was removed")
	}
	if IsDefaultHandler() {
		t.Error("IsDefaultHandler = true after Unregister")
	}
}

func TestUnregisterWithNothingRegistered(t *testing.T) {
	useFakeHKCU(t)
	released, err := Unregister()
	if err != nil || len(released) != 0 {
		t.Errorf("got %v, %v; want nothing released and no error", released, err)
	}
	if IsDefaultHandler() {
		t.Error("IsDefaultHandler = true on an empty hive")
	}
}

func TestRegisterOpenWithForAdvertisesWithoutTakingDefaults(t *testing.T) {
	h := useFakeHKCU(t)
	h.set(`Software\Classes\.epub`, "", "Calibre.epub")
	exe := `C:\Program Files\DHT\doc-html-translate.exe`

	got, err := RegisterOpenWithFor(exe)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got, SupportedExtensions) {
		t.Errorf("advertised %v, want %v", got, SupportedExtensions)
	}
	app := `Software\Classes\Applications\doc-html-translate.exe`
	if v, _ := h.value(app, "FriendlyAppName"); v != "DOC-HTML-TRANSLATE" {
		t.Errorf("FriendlyAppName = %q", v)
	}
	if v, _ := h.value(app+`\shell\open\command`, ""); v != `"`+exe+`" "%1"` {
		t.Errorf("open command = %q", v)
	}
	for _, ext := range SupportedExtensions {
		if _, ok := h.value(app+`\SupportedTypes`, ext); !ok {
			t.Errorf("%s missing from SupportedTypes", ext)
		}
	}
	if v, _ := h.value(`Software\Classes\.epub`, ""); v != "Calibre.epub" {
		t.Errorf(".epub default = %q, want it left alone", v)
	}
}

func TestRegisterOpenWithForFailures(t *testing.T) {
	t.Run("application key cannot be created", func(t *testing.T) {
		h := useFakeHKCU(t)
		h.failCreate = func(p string) bool { return strings.HasPrefix(p, `software\classes\applications\`) }
		if got, err := RegisterOpenWithFor(`C:\a\app.exe`); err == nil || got != nil {
			t.Fatalf("got %v, %v; want an error", got, err)
		}
	})
	t.Run("no type can be advertised", func(t *testing.T) {
		h := useFakeHKCU(t)
		h.failSet = func(p, _ string) bool { return strings.HasSuffix(p, `\supportedtypes`) }
		if got, err := RegisterOpenWithFor(`C:\a\app.exe`); err == nil || got != nil {
			t.Fatalf("got %v, %v; want an error", got, err)
		}
	})
}

func TestRegisterContextMenuFor(t *testing.T) {
	h := useFakeHKCU(t)
	h.failCreate = func(p string) bool { return strings.Contains(p, `\.rtf\`) }
	exe := `D:\portable\doc-html-translate.exe`

	got, err := RegisterContextMenuFor(exe)
	if err != nil {
		t.Fatal(err)
	}
	if slices.Contains(got, ".rtf") || len(got) != len(SupportedExtensions)-1 {
		t.Errorf("added %v, want every extension but .rtf", got)
	}
	verb := `Software\Classes\SystemFileAssociations\.epub\shell\` + contextMenuVerb
	if v, _ := h.value(verb, "MUIVerb"); v != "Convert to HTML" {
		t.Errorf("MUIVerb = %q", v)
	}
	if v, _ := h.value(verb+`\command`, ""); v != `"`+exe+`" "%1"` {
		t.Errorf("command = %q", v)
	}
	if _, ok := h.value(`Software\Classes\.epub`, ""); ok {
		t.Error("the context menu set a default handler")
	}
}

func TestRegisterContextMenuForFailsWhenNothingIsAdded(t *testing.T) {
	h := useFakeHKCU(t)
	// The verb key is created but its command cannot be written: the entry would do nothing,
	// so it must not count as added.
	h.failSet = func(p, name string) bool { return name == "" && strings.HasSuffix(p, `\command`) }
	if got, err := RegisterContextMenuFor(`C:\a\app.exe`); err == nil || got != nil {
		t.Fatalf("got %v, %v; want an error", got, err)
	}
}

// The GUI's shell-entry toggle is a yes the user can take back: removal undoes exactly what
// registration wrote and leaves the Applications key a user's "Always" choice may point at.
func TestRemoveShellEntriesUndoesRegistration(t *testing.T) {
	h := useFakeHKCU(t)
	exe := `D:\portable\doc-html-translate.exe`
	if HasShellEntries() {
		t.Fatal("an empty hive reports shell entries")
	}
	if _, err := RegisterOpenWithFor(exe); err != nil {
		t.Fatal(err)
	}
	if _, err := RegisterContextMenuFor(exe); err != nil {
		t.Fatal(err)
	}
	if !HasShellEntries() {
		t.Fatal("HasShellEntries = false right after registering")
	}
	h.notified = 0

	removed, err := RemoveShellEntries()
	if err != nil {
		t.Fatal(err)
	}
	if len(removed) != len(SupportedExtensions) {
		t.Errorf("removed %v, want every extension", removed)
	}
	if HasShellEntries() {
		t.Error("HasShellEntries = true after removal")
	}
	app := `Software\Classes\Applications\doc-html-translate.exe`
	if _, ok := h.value(app+`\SupportedTypes`, ".epub"); ok {
		t.Error(".epub is still advertised under Open with")
	}
	if v, _ := h.value(app+`\shell\open\command`, ""); v == "" {
		t.Error("the Applications command was deleted; an Always choice pointing at it would break")
	}
	if h.notified != 1 {
		t.Errorf("Explorer notified %d times, want 1", h.notified)
	}

	// Nothing left to remove is not an error and changes nothing.
	h.notified = 0
	if removed, err := RemoveShellEntries(); err != nil || len(removed) != 0 || h.notified != 0 {
		t.Errorf("second removal = %v, %v, notified %d; want nothing", removed, err, h.notified)
	}
}

func userChoiceKey(ext string) string { return fileExtsPath + ext + `\UserChoice` }

// Since Windows 8 the user's own choice wins over the class key: writing it must not be
// reported as having become the default.
func TestRegisterHandlerReportsAUserChoiceThatWins(t *testing.T) {
	h := useFakeHKCU(t)
	h.set(userChoiceKey(".epub"), "ProgId", "Calibre.epub")
	h.set(userChoiceKey(".pdf"), "ProgId", `Applications\`+cliExeName)

	got, err := RegisterHandler()
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got.Blocked, []string{".epub"}) || got.Complete() {
		t.Errorf("blocked = %v, want [.epub] and an incomplete result", got.Blocked)
	}
	if !slices.Contains(got.Default, ".pdf") {
		t.Error(`"Open with -> Always" on the converter was not counted as the default`)
	}
	st := HandlerStatus()
	if st.Summary() != "blocked" || IsDefaultHandler() {
		t.Errorf("status %+v (%s), want blocked and not default", st, st.Summary())
	}
}

func TestHandlerStatusReadsTheWindows11Choice(t *testing.T) {
	h := useFakeHKCU(t)
	if _, err := RegisterHandler(); err != nil {
		t.Fatal(err)
	}
	// UserChoiceLatest wins over a stale UserChoice that still names us.
	h.set(userChoiceKey(".epub"), "ProgId", progID)
	h.set(fileExtsPath+`.epub\UserChoiceLatest`, "Hash", "x")
	h.set(fileExtsPath+`.epub\UserChoiceLatest\ProgId`, "ProgId", "Calibre.epub")
	// A choice key with no readable ProgId is unknown, never "yes".
	h.set(userChoiceKey(".pdf"), "Hash", "x")

	st := HandlerStatus()
	if !slices.Equal(st.Blocked, []string{".epub"}) {
		t.Errorf("blocked = %v, want [.epub]", st.Blocked)
	}
	if !slices.Equal(st.Unknown, []string{".pdf"}) {
		t.Errorf("unknown = %v, want [.pdf]", st.Unknown)
	}
	if st.IsDefault() {
		t.Error("IsDefault = true with an unreadable choice")
	}
}

func TestRegisterThenUnregisterRestoresThePreviousHandler(t *testing.T) {
	h := useFakeHKCU(t)
	h.set(`Software\Classes\.epub`, "", "Calibre.epub")

	// Registering twice must not replace the saved handler with our own ProgID.
	for range 2 {
		if _, err := RegisterHandler(); err != nil {
			t.Fatal(err)
		}
	}
	if v, _ := h.value(`Software\Classes\.epub`, backupValue); v != "Calibre.epub" {
		t.Fatalf("backup = %q, want Calibre.epub", v)
	}
	h.notified = 0

	released, err := Unregister()
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(released, SupportedExtensions) {
		t.Errorf("released %v, want %v", released, SupportedExtensions)
	}
	if v, _ := h.value(`Software\Classes\.epub`, ""); v != "Calibre.epub" {
		t.Errorf(".epub default = %q, want Calibre.epub restored", v)
	}
	if _, ok := h.value(`Software\Classes\.epub`, backupValue); ok {
		t.Error("the backup outlived the restore")
	}
	if _, ok := h.value(`Software\Classes\.pdf`, ""); ok {
		t.Error(".pdf had no previous handler but still has a default")
	}
	if h.notified != 1 {
		t.Errorf("Explorer notified %d times, want once", h.notified)
	}
}

func TestUnregisterDropsABackupForATypeSomeoneElseTook(t *testing.T) {
	h := useFakeHKCU(t)
	h.set(`Software\Classes\.epub`, "", "Calibre.epub")
	if _, err := RegisterHandler(); err != nil {
		t.Fatal(err)
	}
	h.set(`Software\Classes\.epub`, "", "Other.epub")

	if _, err := Unregister(); err != nil {
		t.Fatal(err)
	}
	if v, _ := h.value(`Software\Classes\.epub`, ""); v != "Other.epub" {
		t.Errorf(".epub default = %q, want Other.epub kept", v)
	}
	if _, ok := h.value(`Software\Classes\.epub`, backupValue); ok {
		t.Error("a stale backup survived")
	}
}

func TestUnregisterReportsAFailedRelease(t *testing.T) {
	h := useFakeHKCU(t)
	if _, err := RegisterHandler(); err != nil {
		t.Fatal(err)
	}
	h.failDelete = func(p, name string) bool { return p == `software\classes\.pdf` && name == "" }

	released, err := Unregister()
	if err == nil || !strings.Contains(err.Error(), ".pdf") {
		t.Fatalf("err = %v, want one naming .pdf", err)
	}
	if slices.Contains(released, ".pdf") || len(released) != len(SupportedExtensions)-1 {
		t.Errorf("released %v, want every extension but .pdf", released)
	}
}

// The launch-time integrations run on every GUI start: an unchanged rewrite must not make
// Explorer rebuild its icon cache.
func TestIdempotentRewriteDoesNotNotifyExplorer(t *testing.T) {
	h := useFakeHKCU(t)
	exe := `C:\a\doc-html-translate.exe`
	for range 2 {
		if _, err := RegisterOpenWithFor(exe); err != nil {
			t.Fatal(err)
		}
		if _, err := RegisterContextMenuFor(exe); err != nil {
			t.Fatal(err)
		}
	}
	if h.notified != 2 {
		t.Errorf("Explorer notified %d times, want 2 (the first run of each only)", h.notified)
	}
}

func TestOpenDefaultAppsSettings(t *testing.T) {
	h := useFakeHKCU(t)
	if err := OpenDefaultAppsSettings(); err != nil || h.settings != 1 {
		t.Errorf("err = %v, opened %d times; want one open", err, h.settings)
	}
}
