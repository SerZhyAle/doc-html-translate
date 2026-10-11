package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// No path option may put raw artifacts outside the repository's disposable tree.
func containedPath(root, path string) error {
	r, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	p, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(r, p)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return fmt.Errorf("path must be strictly inside %s: %s", r, p)
	}
	// Inspect every existing component, including parents above root; reject junctions/symlinks.
	for cur := p; ; cur = filepath.Dir(cur) {
		info, err := os.Lstat(cur)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		if err == nil && info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("redirected path contains a link: %s", cur)
		}
		if filepath.Dir(cur) == cur {
			break
		}
	}
	return nil
}

func (e *runEnv) prepareProfile(path string) error {
	if err := containedPath(e.runDir(e.sceneRunID), path); err != nil {
		return err
	}
	if e.recovery != nil {
		if err := e.recovery.validate(); err != nil {
			return err
		}
	}
	for _, p := range e.profiles {
		if p == path {
			return nil
		}
	}
	if _, err := os.Lstat(path); err == nil {
		return fmt.Errorf("refusing a pre-existing profile: %s", path)
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	if err := os.Mkdir(path, 0o700); err != nil {
		return err
	}
	e.profiles = append(e.profiles, path)
	if e.recovery != nil {
		if err := e.recovery.addFolder(path); err != nil {
			return err
		}
		recPath := filepath.Join(e.runDir(e.sceneRunID), "record.json")
		rec, err := loadRecord(recPath)
		if err != nil {
			return err
		}
		rec.RecoverySHA256 = e.recovery.hash
		if _, err := saveRecord(e.runDir(e.sceneRunID), rec); err != nil {
			return err
		}
	}
	return nil
}

func (e *runEnv) cleanupProfiles() error {
	for _, path := range e.profiles {
		if err := containedPath(e.runDir(e.sceneRunID), path); err != nil {
			return err
		}
		if err := os.RemoveAll(path); err != nil {
			return err
		}
		if _, err := os.Lstat(path); !os.IsNotExist(err) {
			return fmt.Errorf("profile residue: %s", path)
		}
	}
	return nil
}

func treeFingerprint() (string, error) {
	cmd := exec.Command("git", "ls-files", "--cached", "--others", "--exclude-standard", "-z")
	cmd.Dir = repoRoot()
	list, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("tree file inventory: %w", err)
	}
	paths := strings.Split(strings.TrimSuffix(string(list), "\x00"), "\x00")
	sort.Strings(paths)
	h := sha256.New()
	last := ""
	for _, p := range paths {
		if p == "" || p == last {
			continue
		}
		last = p
		b, err := os.ReadFile(filepath.Join(repoRoot(), p))
		if err != nil {
			return "", err
		}
		sum := sha256.Sum256(b)
		fmt.Fprintf(h, "%s\x00%x\x00", p, sum)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
