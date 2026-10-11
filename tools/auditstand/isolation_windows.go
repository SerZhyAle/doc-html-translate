//go:build windows

package main

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

type windowsRegistry struct{}

func newRegistryStore() (registryStore, error) { return windowsRegistry{}, nil }

func (windowsRegistry) exists(path string) (bool, error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, path, registry.READ)
	if errors.Is(err, registry.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	k.Close()
	return true, nil
}

func (windowsRegistry) snapshot(path string) (regTree, error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, path, registry.READ)
	if errors.Is(err, registry.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer k.Close()
	tree := regTree{}
	var walk func(registry.Key, string) error
	walk = func(k registry.Key, relative string) error {
		values := map[string]rawRegValue{}
		names, err := k.ReadValueNames(-1)
		if err != nil {
			return err
		}
		for _, name := range names {
			n, _, err := k.GetValue(name, nil)
			if err != nil {
				return err
			}
			data := make([]byte, n)
			n, typ, err := k.GetValue(name, data)
			if err != nil {
				return err
			}
			values[name] = rawRegValue{Type: typ, Data: data[:n]}
		}
		tree[relative] = values
		subs, err := k.ReadSubKeyNames(-1)
		if err != nil {
			return err
		}
		for _, sub := range subs {
			child, err := registry.OpenKey(k, sub, registry.READ)
			if err != nil {
				return err
			}
			err = walk(child, regPath(relative, sub))
			child.Close()
			if err != nil {
				return err
			}
		}
		return nil
	}
	if err := walk(k, ""); err != nil {
		return nil, err
	}
	return tree, nil
}

func (windowsRegistry) export(path, file string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "reg.exe", "export", `HKCU\`+path, file, "/y").CombinedOutput()
	if err != nil {
		return fmt.Errorf("reg export %s: %w (%s)", path, err, out)
	}
	return nil
}
func (windowsRegistry) create(path string) error {
	k, _, err := registry.CreateKey(registry.CURRENT_USER, path, registry.WRITE)
	if err == nil {
		k.Close()
	}
	return err
}
func (windowsRegistry) set(path, name string, value rawRegValue) error {
	k, err := registry.OpenKey(registry.CURRENT_USER, path, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	n, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return err
	}
	var p *byte
	if len(value.Data) > 0 {
		p = &value.Data[0]
	}
	r, _, _ := windows.NewLazySystemDLL("advapi32.dll").NewProc("RegSetValueExW").Call(
		uintptr(k), uintptr(unsafe.Pointer(n)), 0, uintptr(value.Type), uintptr(unsafe.Pointer(p)), uintptr(len(value.Data)))
	if r != 0 {
		return windows.Errno(r)
	}
	return nil
}
func (windowsRegistry) deleteValue(path, name string) error {
	k, err := registry.OpenKey(registry.CURRENT_USER, path, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	return k.DeleteValue(name)
}
func (windowsRegistry) deleteKey(path string) error {
	return registry.DeleteKey(registry.CURRENT_USER, path)
}
