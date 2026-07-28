//go:build !windows

package syslocale

// Lang returns "en" on non-Windows platforms (the app targets Windows).
func Lang() string { return "en" }
