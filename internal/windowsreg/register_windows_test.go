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
	delete(k.hive.keys[k.path], name)
	return nil
}

func (k fakeKey) Close() error { return nil }

// useFakeHKCU swaps the registry seams for an empty fake for the duration of the test.
func useFakeHKCU(t *testing.T) *fakeHKCU {
	t.Helper()
	h := &fakeHKCU{keys: map[string]map[string]string{}}
	origCreate, origOpen, origDelete := createKey, openKey, deleteKey
	t.Cleanup(func() { createKey, openKey, deleteKey = origCreate, origOpen, origDelete })

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
	if !slices.Equal(got, SupportedExtensions) {
		t.Errorf("registered %v, want %v", got, SupportedExtensions)
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
		if err == nil || got != nil {
			t.Fatalf("got %v, %v; want an error and nothing registered", got, err)
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
		if slices.Contains(got, ".pdf") || len(got) != len(SupportedExtensions)-1 {
			t.Errorf("registered %v, want every extension but .pdf", got)
		}
		if IsDefaultHandler() {
			t.Error("IsDefaultHandler = true with .pdf unassociated")
		}
	})
	t.Run("every extension fails", func(t *testing.T) {
		h := useFakeHKCU(t)
		h.failCreate = func(p string) bool { return strings.HasPrefix(p, `software\classes\.`) }
		if got, err := RegisterHandler(); err == nil || got != nil {
			t.Fatalf("got %v, %v; want an error", got, err)
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
