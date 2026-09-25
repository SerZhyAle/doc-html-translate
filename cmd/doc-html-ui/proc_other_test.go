//go:build !windows

package main

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"syscall"
)

// processGone treats a zombie as gone: it has exited and only waits to be reaped, which in a
// container can be nobody's job.
func processGone(pid int) bool {
	if data, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat"); err == nil {
		s := string(data)
		if i := strings.LastIndexByte(s, ')'); i >= 0 && i+2 < len(s) {
			return s[i+2] == 'Z'
		}
	} else if errors.Is(err, os.ErrNotExist) {
		if _, statErr := os.Stat("/proc/self"); statErr == nil {
			return true
		}
	}
	return syscall.Kill(pid, 0) != nil
}
