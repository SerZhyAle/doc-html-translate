//go:build !windows

package main

var raiseLauncherWindow = func() (found bool, focused bool) {
	return false, false
}
