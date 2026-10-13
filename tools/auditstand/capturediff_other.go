//go:build !windows

package main

import (
	"time"
)

func captureWindowGrid(uintptr) ([]byte, error) { return nil, errNotWindows }

func gridMaxDiff(a, b []byte) int {
	if len(a) != len(b) || len(a) == 0 {
		return 1 << 30
	}
	max := 0
	for i := range a {
		d := int(a[i]) - int(b[i])
		if d < 0 {
			d = -d
		}
		if d > max {
			max = d
		}
	}
	return max
}

func findWindowsByTitle(string) ([]uintptr, error) { return nil, errNotWindows }

func bringWindowToFront(uintptr) error { return errNotWindows }

func waitWindowPaint(uintptr, time.Duration) (float64, bool, error) { return 0, false, errNotWindows }
