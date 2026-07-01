//go:build !windows

package syslocale

// IsRussian always returns false on non-Windows platforms.
func IsRussian() bool { return false }

// Lang returns "en" on non-Windows platforms (the app targets Windows).
func Lang() string { return "en" }
