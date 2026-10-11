package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

// Registry data retains its type and uninterpreted bytes, including unnamed values.
// Restore changes only differences; unchanged Windows UserChoice keys are never rewritten.
type rawRegValue struct {
	Type uint32 `json:"type"`
	Data []byte `json:"data"`
}
type regTree map[string]map[string]rawRegValue // relative key -> value name -> data

type registryStore interface {
	exists(string) (bool, error)
	snapshot(string) (regTree, error) // nil means absent; an empty key has a "" entry
	export(string, string) error
	create(string) error
	set(string, string, rawRegValue) error
	deleteValue(string, string) error
	deleteKey(string) error // empty keys only
}

type recoveryKey struct {
	Hive    string   `json:"hive"`
	Path    string   `json:"path"`
	Absent  bool     `json:"absent_by_design"`
	Before  regTree  `json:"before"`
	Export  string   `json:"export,omitempty"`
	SHA256  string   `json:"sha256,omitempty"`
	Restore []string `json:"restore_command"`
	Verify  []string `json:"verify_command"`
}
type recoveryManifest struct {
	Format        int           `json:"format"`
	Keys          []recoveryKey `json:"keys"`
	AbsentParents []string      `json:"absent_parents"`
	Folders       []string      `json:"disposable_folders"`
	Restore       []string      `json:"restore_command"`
	Verify        []string      `json:"verify_command"`
}
type profileRecovery struct {
	store    registryStore
	manifest recoveryManifest
	path     string
	hash     string
	roots    []string
}

// Read the writer's declarations rather than maintaining a second extension roster.
func registrationRoots(root string, exeNames ...string) ([]string, error) {
	f, err := parser.ParseFile(token.NewFileSet(), filepath.Join(root, "internal", "windowsreg", "register_windows.go"), nil, 0)
	if err != nil {
		return nil, err
	}
	lists := map[string][]string{}
	constants := map[string]string{}
	for _, d := range f.Decls {
		g, ok := d.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, s := range g.Specs {
			v, ok := s.(*ast.ValueSpec)
			if !ok || len(v.Names) != 1 || len(v.Values) != 1 {
				continue
			}
			name := v.Names[0].Name
			switch value := v.Values[0].(type) {
			case *ast.BasicLit:
				if value.Kind == token.STRING {
					constants[name], err = strconv.Unquote(value.Value)
				}
			case *ast.CompositeLit:
				for _, element := range value.Elts {
					lit, ok := element.(*ast.BasicLit)
					if !ok || lit.Kind != token.STRING {
						continue
					}
					x, e := strconv.Unquote(lit.Value)
					if e != nil {
						return nil, e
					}
					lists[name] = append(lists[name], x)
				}
			}
			if err != nil {
				return nil, err
			}
		}
	}
	for _, n := range []string{"SupportedExtensions", "ShellOnlyExtensions", "legacyProgIDs"} {
		if len(lists[n]) == 0 {
			return nil, fmt.Errorf("registration inventory %s unreadable", n)
		}
	}
	for _, n := range []string{"progID", "contextMenuVerb", "fileExtsPath", "cliExeName"} {
		if constants[n] == "" {
			return nil, fmt.Errorf("registration inventory %s unreadable", n)
		}
	}
	roots := []string{`Software\Classes\` + constants["progID"]}
	for _, id := range lists["legacyProgIDs"] {
		roots = append(roots, `Software\Classes\`+id)
	}
	for _, ext := range lists["SupportedExtensions"] {
		roots = append(roots, `Software\Classes\`+ext, constants["fileExtsPath"]+ext)
	}
	for _, ext := range append(lists["SupportedExtensions"], lists["ShellOnlyExtensions"]...) {
		roots = append(roots, `Software\Classes\SystemFileAssociations\`+ext+`\shell\`+constants["contextMenuVerb"])
	}
	for _, exe := range append(exeNames, constants["cliExeName"], "doc-html-ui.exe") {
		if exe != "" {
			roots = append(roots, `Software\Classes\Applications\`+filepath.Base(exe))
		}
	}
	seen := map[string]bool{}
	var out []string
	for _, p := range roots {
		if strings.ContainsAny(p, "\r\n\x00") || strings.Contains(p, "..") {
			return nil, fmt.Errorf("invalid registry root %q", p)
		}
		if !seen[strings.ToLower(p)] {
			out = append(out, p)
			seen[strings.ToLower(p)] = true
		}
	}
	sort.Strings(out)
	return out, nil
}

func backupProfile(store registryStore, dir string, roots, folders []string) (*profileRecovery, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	r := &profileRecovery{store: store, path: filepath.Join(dir, "recovery.json"), roots: roots,
		manifest: recoveryManifest{Format: 1, Folders: folders}}
	r.manifest.Restore = []string{"auditstand", "-Recover", r.path, "-recovery-sha256", "<recovery_sha256 from record.json>"}
	r.manifest.Verify = []string{"auditstand", "-VerifyRecovery", r.path, "-recovery-sha256", "<recovery_sha256 from record.json>"}
	parents := map[string]bool{}
	for i, root := range roots {
		before, err := store.snapshot(root)
		if err != nil {
			return nil, err
		}
		k := recoveryKey{Hive: "HKCU", Path: root, Before: before, Absent: before == nil,
			Verify: []string{"reg.exe", "query", `HKCU\` + root, "/s"}}
		if before != nil {
			k.Export = fmt.Sprintf("key-%03d.reg", i)
			p := filepath.Join(dir, k.Export)
			if err := store.export(root, p); err != nil {
				return nil, err
			}
			// An export taken during a concurrent edit is not a recoverable pre-state.
			after, err := store.snapshot(root)
			if err != nil {
				return nil, err
			}
			if !reflect.DeepEqual(before, after) {
				return nil, fmt.Errorf("registry changed during backup: %s", root)
			}
			k.SHA256, err = sha256File(p)
			if err != nil {
				return nil, err
			}
			k.Restore = []string{"reg.exe", "import", p}
		} else {
			k.Restore = []string{"reg.exe", "delete", `HKCU\` + root, "/f"}
		}
		r.manifest.Keys = append(r.manifest.Keys, k)
		for p := root; strings.Contains(p, `\`); {
			p = p[:strings.LastIndex(p, `\`)]
			if parents[p] {
				continue
			}
			parents[p] = true
			present, err := store.exists(p)
			if err != nil {
				return nil, err
			}
			if !present {
				r.manifest.AbsentParents = append(r.manifest.AbsentParents, p)
			}
		}
	}
	// Parent existence needs only a shallow read on Windows; snapshot above is intentionally
	// not used to restore ancestors (they contain unrelated applications).
	sort.Slice(r.manifest.AbsentParents, func(i, j int) bool { return len(r.manifest.AbsentParents[i]) > len(r.manifest.AbsentParents[j]) })
	b, err := json.MarshalIndent(r.manifest, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(r.path, b, 0o600); err != nil {
		return nil, err
	}
	r.hash, err = sha256File(r.path)
	if err != nil {
		return nil, err
	}
	return r, r.validate()
}

func (r *profileRecovery) validate() error {
	b, err := os.ReadFile(r.path)
	if err != nil {
		return err
	}
	h := sha256.Sum256(b)
	if hex.EncodeToString(h[:]) != r.hash {
		return fmt.Errorf("recovery manifest hash mismatch")
	}
	var m recoveryManifest
	if err := json.Unmarshal(b, &m); err != nil {
		return err
	}
	if m.Format != 1 || len(m.Keys) != len(r.roots) {
		return fmt.Errorf("incomplete recovery manifest")
	}
	for i, root := range r.roots {
		k := m.Keys[i]
		if k.Path != root || k.Hive != "HKCU" || k.Absent != (k.Before == nil) {
			return fmt.Errorf("uncovered registry key: %s", root)
		}
		if k.Absent {
			continue
		}
		if k.Export != filepath.Base(k.Export) || k.Export == "" || k.SHA256 == "" {
			return fmt.Errorf("missing export for %s", root)
		}
		hash, err := sha256File(filepath.Join(filepath.Dir(r.path), k.Export))
		if err != nil {
			return err
		}
		if hash != k.SHA256 {
			return fmt.Errorf("export hash mismatch for %s", root)
		}
	}
	return nil
}

func regPath(root, relative string) string {
	if relative == "" {
		return root
	}
	if root == "" {
		return relative
	}
	return root + `\` + relative
}

func (r *profileRecovery) addFolder(path string) error {
	for _, p := range r.manifest.Folders {
		if p == path {
			return nil
		}
	}
	r.manifest.Folders = append(r.manifest.Folders, path)
	b, err := json.MarshalIndent(r.manifest, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(r.path, b, 0o600); err != nil {
		return err
	}
	r.hash, err = sha256File(r.path)
	return err
}

func restoreTree(store registryStore, root string, before regTree) error {
	current, err := store.snapshot(root)
	if err != nil {
		return err
	}
	var keys []string
	for k := range current {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return len(keys[i]) > len(keys[j]) })
	for _, key := range keys {
		want, existed := before[key]
		for name := range current[key] {
			if _, ok := want[name]; !ok {
				if err := store.deleteValue(regPath(root, key), name); err != nil {
					return err
				}
			}
		}
		if !existed {
			if err := store.deleteKey(regPath(root, key)); err != nil {
				return err
			}
		}
	}
	keys = nil
	for k := range before {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return len(keys[i]) < len(keys[j]) })
	for _, key := range keys {
		if _, ok := current[key]; !ok {
			if err := store.create(regPath(root, key)); err != nil {
				return err
			}
		}
		for name, value := range before[key] {
			old, exists := current[key][name]
			if !exists || old.Type != value.Type || !bytes.Equal(old.Data, value.Data) {
				if err := store.set(regPath(root, key), name, value); err != nil {
					return err
				}
			}
		}
	}
	actual, err := store.snapshot(root)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(before, actual) {
		return fmt.Errorf("byte-level registry restore mismatch: %s", root)
	}
	return nil
}

func (r *profileRecovery) restore() error {
	// Attempt every backed-up root even when one restoration fails. Keep exports for recovery.
	var failures []string
	for _, k := range r.manifest.Keys {
		if err := restoreTree(r.store, k.Path, k.Before); err != nil {
			failures = append(failures, k.Path+": "+err.Error())
		}
	}
	for _, p := range r.manifest.AbsentParents {
		tree, err := r.store.snapshot(p)
		if err != nil {
			failures = append(failures, err.Error())
			continue
		}
		if tree == nil {
			continue
		}
		if len(tree) != 1 || len(tree[""]) != 0 {
			failures = append(failures, "new parent not empty: "+p)
			continue
		}
		if err := r.store.deleteKey(p); err != nil {
			failures = append(failures, err.Error())
		}
	}
	if len(failures) != 0 {
		return fmt.Errorf("P0/T profile restoration failed: %s", strings.Join(failures, "; "))
	}
	return nil
}

// Crash recovery refuses untrusted manifests; the expected hash comes from the run record.
func loadProfileRecovery(path, hash string, store registryStore, roots []string) (*profileRecovery, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(hash) != 64 {
		return nil, fmt.Errorf("the recovery SHA-256 from the run record is required")
	}
	var m recoveryManifest
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	r := &profileRecovery{store: store, manifest: m, path: path, hash: hash, roots: roots}
	if err := r.validate(); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *profileRecovery) verifyRestored() error {
	for _, k := range r.manifest.Keys {
		actual, err := r.store.snapshot(k.Path)
		if err != nil {
			return err
		}
		if !reflect.DeepEqual(k.Before, actual) {
			return fmt.Errorf("registry differs from pre-state: %s", k.Path)
		}
	}
	for _, p := range r.manifest.AbsentParents {
		present, err := r.store.exists(p)
		if err != nil {
			return err
		}
		if present {
			return fmt.Errorf("new registry parent remains: %s", p)
		}
	}
	return nil
}
