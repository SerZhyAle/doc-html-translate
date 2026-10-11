// stateprobe.go is P8 (strategic section 6, property 5): the state diffs around a GUI toggle. A
// toggle is understandable only if flipping it changes something an outside observer can see and
// flipping it back restores the pre-state byte for byte (H14). This file holds the primitives -
// a registry-and-folder snapshot, a diff, and the two-snapshot toggle evaluation - and nothing
// that drives a GUI; the scenes that own a real toggle (S11, S18, phase 04) supply the three
// snapshots.
//
// Registry values are compared on their type and raw bytes; a report names a value, never its
// data, so a run record carries no handler paths or user choices.

package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// hashSizeCap is the largest file whose content is hashed. Larger files are compared on size
// alone and recorded as such: a toggle's state files are settings and markers, and a capped log
// store is only ever compared by size.
const hashSizeCap = 64 << 20

type folderEntry struct {
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"` // "" for a directory, "size-only" past hashSizeCap
}

// folderTree maps a slash-separated path relative to the folder root to its entry. Directories
// are present with an empty hash so that an emptied folder differs from an absent one.
type folderTree map[string]folderEntry

// snapshotFolder records every file under root. A missing root is nil (absent), distinct from an
// existing empty folder (a non-nil empty tree).
func snapshotFolder(root string) (folderTree, error) {
	info, err := os.Stat(root)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a folder", root)
	}
	tree := folderTree{}
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, werr error) error {
		if werr != nil {
			return werr
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			tree[rel] = folderEntry{}
			return nil
		}
		fi, err := d.Info()
		if err != nil {
			return err
		}
		entry := folderEntry{Size: fi.Size(), SHA256: "size-only"}
		if fi.Size() <= hashSizeCap {
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			h := sha256.New()
			_, err = io.Copy(h, f)
			f.Close()
			if err != nil {
				return err
			}
			entry.SHA256 = hex.EncodeToString(h.Sum(nil))
		}
		tree[rel] = entry
		return nil
	})
	return tree, err
}

// stateSnapshot is everything a toggle can change that the stand watches: the named registry
// roots, the named folders, and behavioural facts the scene read from the running program (for
// example "update check: on" as the settings API reports it).
type stateSnapshot struct {
	Registry map[string]regTree    `json:"registry,omitempty"`
	Folders  map[string]folderTree `json:"folders,omitempty"`
	Facts    map[string]string     `json:"facts,omitempty"`
}

// captureState reads the watched registry roots and folders. The folder keys are the caller's
// stable names (not paths), so a snapshot is comparable across profiles.
func captureState(store registryStore, regRoots []string, folders map[string]string, facts map[string]string) (*stateSnapshot, error) {
	s := &stateSnapshot{Registry: map[string]regTree{}, Folders: map[string]folderTree{}, Facts: map[string]string{}}
	for _, root := range regRoots {
		tree, err := store.snapshot(root)
		if err != nil {
			return nil, fmt.Errorf("registry %s: %w", root, err)
		}
		s.Registry[root] = tree
	}
	for name, dir := range folders {
		tree, err := snapshotFolder(dir)
		if err != nil {
			return nil, fmt.Errorf("folder %s: %w", name, err)
		}
		s.Folders[name] = tree
	}
	for k, v := range facts {
		s.Facts[k] = v
	}
	return s, nil
}

// stateChange is one difference between two snapshots.
type stateChange struct {
	Kind  string `json:"kind"`  // "added" | "removed" | "changed"
	Where string `json:"where"` // "registry" | "folder" | "fact"
	Item  string `json:"item"`
}

type stateDiff []stateChange

func (d stateDiff) empty() bool { return len(d) == 0 }

func (d stateDiff) String() string {
	parts := make([]string, 0, len(d))
	for _, c := range d {
		parts = append(parts, fmt.Sprintf("%s %s %s", c.Kind, c.Where, c.Item))
	}
	return strings.Join(parts, "; ")
}

// diffState lists what differs between two snapshots, sorted so two diffs of the same pair are
// byte-identical.
func diffState(before, after *stateSnapshot) stateDiff {
	var out stateDiff
	add := func(kind, where, item string) { out = append(out, stateChange{kind, where, item}) }

	roots := map[string]bool{}
	for r := range before.Registry {
		roots[r] = true
	}
	for r := range after.Registry {
		roots[r] = true
	}
	for root := range roots {
		b, a := before.Registry[root], after.Registry[root]
		switch {
		case b == nil && a == nil:
		case b == nil:
			add("added", "registry", `key `+root)
		case a == nil:
			add("removed", "registry", `key `+root)
		default:
			diffRegTree(root, b, a, add)
		}
	}

	names := map[string]bool{}
	for n := range before.Folders {
		names[n] = true
	}
	for n := range after.Folders {
		names[n] = true
	}
	for name := range names {
		b, a := before.Folders[name], after.Folders[name]
		switch {
		case b == nil && a == nil:
		case b == nil:
			add("added", "folder", name)
		case a == nil:
			add("removed", "folder", name)
		default:
			for rel, eb := range b {
				ea, ok := a[rel]
				switch {
				case !ok:
					add("removed", "folder", name+"/"+rel)
				case ea != eb:
					add("changed", "folder", name+"/"+rel)
				}
			}
			for rel := range a {
				if _, ok := b[rel]; !ok {
					add("added", "folder", name+"/"+rel)
				}
			}
		}
	}

	for k, vb := range before.Facts {
		va, ok := after.Facts[k]
		switch {
		case !ok:
			add("removed", "fact", k)
		case va != vb:
			add("changed", "fact", k)
		}
	}
	for k := range after.Facts {
		if _, ok := before.Facts[k]; !ok {
			add("added", "fact", k)
		}
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Where != out[j].Where {
			return out[i].Where < out[j].Where
		}
		if out[i].Item != out[j].Item {
			return out[i].Item < out[j].Item
		}
		return out[i].Kind < out[j].Kind
	})
	return out
}

func diffRegTree(root string, b, a regTree, add func(kind, where, item string)) {
	for rel, vb := range b {
		path := root
		if rel != "" {
			path += `\` + rel
		}
		va, ok := a[rel]
		if !ok {
			add("removed", "registry", "key "+path)
			continue
		}
		for name, x := range vb {
			y, ok := va[name]
			switch {
			case !ok:
				add("removed", "registry", fmt.Sprintf("value %s :: %q", path, name))
			case x.Type != y.Type || !bytes.Equal(x.Data, y.Data):
				add("changed", "registry", fmt.Sprintf("value %s :: %q", path, name))
			}
		}
		for name := range va {
			if _, ok := vb[name]; !ok {
				add("added", "registry", fmt.Sprintf("value %s :: %q", path, name))
			}
		}
	}
	for rel := range a {
		if _, ok := b[rel]; !ok {
			path := root
			if rel != "" {
				path += `\` + rel
			}
			add("added", "registry", "key "+path)
		}
	}
}

// toggleCheck is the H14 record of one GUI toggle: did flipping it change something observable
// from outside, and did flipping it back restore the pre-state byte-identically.
type toggleCheck struct {
	Toggle     string   `json:"toggle"`
	Channels   []string `json:"channels"`   // what was watched: registry roots, folder names, fact keys
	Observable bool     `json:"observable"` // the flip changed at least one watched item
	Reverted   bool     `json:"reverted"`   // the revert returned every watched item to its pre-state
	Changes    int      `json:"changes"`    // items the flip changed
	Leftover   int      `json:"leftover"`   // items still different from the pre-state after the revert
	Detail     string   `json:"detail,omitempty"`
}

func (c toggleCheck) ok() bool { return c.Observable && c.Reverted }

// evaluateToggle compares the three snapshots of one toggle: the pre-state, the state after the
// flip and the state after the revert. Only item names reach the Detail - never registry data.
func evaluateToggle(name string, before, flipped, reverted *stateSnapshot) toggleCheck {
	channels := map[string]bool{}
	for r := range before.Registry {
		channels["registry:"+r] = true
	}
	for f := range before.Folders {
		channels["folder:"+f] = true
	}
	for k := range before.Facts {
		channels["fact:"+k] = true
	}
	list := make([]string, 0, len(channels))
	for c := range channels {
		list = append(list, c)
	}
	sort.Strings(list)

	flip := diffState(before, flipped)
	back := diffState(before, reverted)
	c := toggleCheck{
		Toggle: name, Channels: list,
		Observable: !flip.empty(), Reverted: back.empty(),
		Changes: len(flip), Leftover: len(back),
	}
	switch {
	case !c.Observable:
		c.Detail = "the flip changed nothing the stand watched"
	case !c.Reverted:
		c.Detail = "after the revert still differing: " + back.String()
	}
	return c
}

// h14ToggleVerdict is the externally measurable half of H14. It can fail (a toggle that changed
// nothing observable, or a revert that left a difference) but it never passes the budget alone:
// locale presence, accessibility announcements and the keyboard-only flows are other probes'
// halves, so a clean toggle half is an advisory until those report.
func h14ToggleVerdict(checks []toggleCheck) budgetVerdict {
	v := budgetVerdict{ID: "H14", Limit: findBudget("H14").limit}
	if len(checks) == 0 {
		v.Verdict = "advisory"
		v.Observed = "no toggle was exercised"
		return v
	}
	var bad []string
	for _, c := range checks {
		if !c.ok() {
			bad = append(bad, c.Toggle)
		}
	}
	if len(bad) > 0 {
		v.Verdict = "fail"
		v.Observed = fmt.Sprintf("%d of %d toggle(s) unobservable or not reverted: %s", len(bad), len(checks), strings.Join(bad, ", "))
		return v
	}
	v.Verdict = "advisory"
	v.Observed = fmt.Sprintf("%d of %d toggle(s) observable and byte-identical after revert; locale, announcement and keyboard halves are other probes", len(checks), len(checks))
	return v
}
