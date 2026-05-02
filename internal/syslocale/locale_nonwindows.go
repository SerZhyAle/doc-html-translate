//go:build !windows

package syslocale

// IsRussian always returns false on non-Windows platforms.
func IsRussian() bool { return false }
