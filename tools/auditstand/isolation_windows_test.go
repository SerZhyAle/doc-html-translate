//go:build windows

package main

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"golang.org/x/sys/windows/registry"
)

func TestWindowsRegistryRecoveryOnScratchKey(t *testing.T) {
	// Only this unique scratch subtree is writable; registration keys are never touched.
	root := fmt.Sprintf(`Software\AuditstandTest-%d`, time.Now().UnixNano())
	s := windowsRegistry{}
	if err := s.create(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		tree, _ := s.snapshot(root)
		if tree != nil {
			_ = restoreTree(s, root, nil)
		}
	})
	for name, value := range map[string]rawRegValue{"": {registry.SZ, []byte{65, 0, 0, 0}}, "blob": {registry.BINARY, []byte{0, 255}}, "empty": {registry.BINARY, []byte{}}} {
		if err := s.set(root, name, value); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.create(root + `\Child`); err != nil {
		t.Fatal(err)
	}
	before, err := s.snapshot(root)
	if err != nil {
		t.Fatal(err)
	}
	r, err := backupProfile(s, t.TempDir(), []string{root}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.set(root, "blob", rawRegValue{registry.BINARY, []byte{42}}); err != nil {
		t.Fatal(err)
	}
	if err := s.create(root + `\New`); err != nil {
		t.Fatal(err)
	}
	if err := s.deleteKey(root + `\Child`); err != nil {
		t.Fatal(err)
	}
	if err := r.restore(); err != nil {
		t.Fatal(err)
	}
	after, err := s.snapshot(root)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatal("scratch registry restore not byte-identical")
	}
}
