package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type memoryRegistry struct {
	keys       map[string]map[string]rawRegValue
	failExport bool
	failSet    bool
}

func (m *memoryRegistry) exists(p string) (bool, error) { _, ok := m.keys[p]; return ok, nil }
func (m *memoryRegistry) snapshot(p string) (regTree, error) {
	var out regTree
	for k, v := range m.keys {
		if k == p || strings.HasPrefix(k, p+`\`) {
			if out == nil {
				out = regTree{}
			}
			rel := strings.TrimPrefix(strings.TrimPrefix(k, p), `\`)
			out[rel] = map[string]rawRegValue{}
			for n, x := range v {
				out[rel][n] = rawRegValue{x.Type, append([]byte{}, x.Data...)}
			}
		}
	}
	return out, nil
}
func (m *memoryRegistry) export(p, f string) error {
	if m.failExport {
		return fmt.Errorf("export failed")
	}
	tree, _ := m.snapshot(p)
	b, _ := json.Marshal(tree)
	return os.WriteFile(f, b, 0o600)
}
func (m *memoryRegistry) create(p string) error {
	if m.keys[p] == nil {
		m.keys[p] = map[string]rawRegValue{}
	}
	return nil
}
func (m *memoryRegistry) set(p, n string, v rawRegValue) error {
	if m.failSet {
		return fmt.Errorf("set failed")
	}
	m.keys[p][n] = v
	return nil
}
func (m *memoryRegistry) deleteValue(p, n string) error { delete(m.keys[p], n); return nil }
func (m *memoryRegistry) deleteKey(p string) error {
	for k := range m.keys {
		if strings.HasPrefix(k, p+`\`) {
			return fmt.Errorf("not empty")
		}
	}
	delete(m.keys, p)
	return nil
}

func TestProfileRecoveryRestoresRawValuesAndAbsence(t *testing.T) {
	m := &memoryRegistry{keys: map[string]map[string]rawRegValue{
		`Software`: {}, `Software\Audit`: {"": {1, []byte{65, 0, 0, 0}}, "binary": {3, []byte{0, 255}}, "empty": {3, []byte{}}},
		`Software\Audit\Child`: {"number": {4, []byte{1, 0, 0, 0}}},
	}}
	before, _ := m.snapshot(`Software\Audit`)
	r, err := backupProfile(m, t.TempDir(), []string{`Software\Audit`, `Software\Absent`}, nil)
	if err != nil {
		t.Fatal(err)
	}
	m.keys[`Software\Audit`][""] = rawRegValue{1, []byte{66, 0, 0, 0}}
	m.keys[`Software\Audit`]["extra"] = rawRegValue{3, []byte{1}}
	delete(m.keys, `Software\Audit\Child`)
	m.keys[`Software\Audit\New`] = map[string]rawRegValue{"x": {3, []byte{9}}}
	m.keys[`Software\Absent`] = map[string]rawRegValue{}
	if err := r.restore(); err != nil {
		t.Fatal(err)
	}
	after, _ := m.snapshot(`Software\Audit`)
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("restore differs: %#v", after)
	}
	if ok, _ := m.exists(`Software\Absent`); ok {
		t.Fatal("absent key survived")
	}
}

func TestProfileRecoveryRefusesIncompleteOrTamperedBackup(t *testing.T) {
	m := &memoryRegistry{keys: map[string]map[string]rawRegValue{`Audit`: {"x": {3, []byte{1}}}}}
	for _, what := range []string{"manifest", "export"} {
		t.Run(what, func(t *testing.T) {
			r, err := backupProfile(m, t.TempDir(), []string{"Audit"}, nil)
			if err != nil {
				t.Fatal(err)
			}
			p := r.path
			if what == "export" {
				p = filepath.Join(filepath.Dir(p), r.manifest.Keys[0].Export)
			}
			if err := os.WriteFile(p, []byte("tampered"), 0o600); err != nil {
				t.Fatal(err)
			}
			if r.validate() == nil {
				t.Fatal("tampering accepted")
			}
		})
	}
	m.failExport = true
	if _, err := backupProfile(m, t.TempDir(), []string{"Audit"}, nil); err == nil {
		t.Fatal("incomplete export accepted")
	}
}

func TestHashRequiredCrashRecoveryReload(t *testing.T) {
	m := &memoryRegistry{keys: map[string]map[string]rawRegValue{"Audit": {"v": {3, []byte{1}}}}}
	r, err := backupProfile(m, t.TempDir(), []string{"Audit"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := loadProfileRecovery(r.path, "", m, r.roots); err == nil {
		t.Fatal("crash recovery accepted no expected digest")
	}
	loaded, err := loadProfileRecovery(r.path, r.hash, m, r.roots)
	if err != nil {
		t.Fatal(err)
	}
	m.keys["Audit"]["v"] = rawRegValue{3, []byte{2}}
	if err := loaded.verifyRestored(); err == nil {
		t.Fatal("verification accepted changed pre-state")
	}
	if err := loaded.restore(); err != nil {
		t.Fatal(err)
	}
	if err := loaded.verifyRestored(); err != nil {
		t.Fatal(err)
	}
}

func TestProfileRestoreFailureIsP0AndOtherRootsStillRestore(t *testing.T) {
	m := &memoryRegistry{keys: map[string]map[string]rawRegValue{"A": {"v": {3, []byte{1}}}, "B": {}}}
	r, err := backupProfile(m, t.TempDir(), []string{"A", "B", "C"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	m.keys["A"]["v"] = rawRegValue{3, []byte{2}}
	m.keys["C"] = map[string]rawRegValue{}
	m.failSet = true
	if err := r.restore(); err == nil || !strings.Contains(err.Error(), "P0/T") {
		t.Fatalf("want P0 restore failure: %v", err)
	}
	if _, ok := m.keys["C"]; ok {
		t.Fatal("one failed root prevented restoration of others")
	}
}

func TestRegistryPlanCoversWriterAndRenamedExecutables(t *testing.T) {
	root := filepath.Join("..", "..")
	roots, err := registrationRoots(root, "renamed-cli.exe", "renamed-ui.exe")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`Software\Classes\doc-html-translate`, `Software\Classes\epub2html`,
		`Software\Classes\Applications\renamed-cli.exe`, `Software\Classes\Applications\renamed-ui.exe`,
		`Software\Classes\.epub`, `Software\Microsoft\Windows\CurrentVersion\Explorer\FileExts\.epub`,
		`Software\Classes\SystemFileAssociations\.fd-sec\shell\dochtmltranslate.convert`} {
		found := false
		for _, p := range roots {
			found = found || p == want
		}
		if !found {
			t.Errorf("missing %s", want)
		}
	}
}

func TestContainedPathRefusesEscapesAndLinks(t *testing.T) {
	root := t.TempDir()
	for _, p := range []string{root, filepath.Join(root, "..", "foreign"), filepath.Join(root, "..")} {
		if containedPath(root, p) == nil {
			t.Fatalf("escape accepted: %s", p)
		}
	}
	if err := containedPath(root, filepath.Join(root, "owned", "profile")); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link")
	if err := os.Symlink(t.TempDir(), link); err == nil {
		if containedPath(root, filepath.Join(link, "profile")) == nil {
			t.Fatal("link accepted")
		}
	}
}
