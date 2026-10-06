//go:build !windows

package fdsec

// isFileDO has nothing to read on other systems: FileDO is a Windows program, and the
// override variable or PATH is the only way to name one.
func isFileDO(string) bool { return true }
