//go:build !windows

package main

import "fmt"

func newRegistryStore() (registryStore, error) {
	return nil, fmt.Errorf("Level B registry isolation needs Windows")
}
