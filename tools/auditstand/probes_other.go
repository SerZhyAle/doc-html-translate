//go:build !windows

// probes_other.go keeps the tree compiling on every platform a vet can reach. Every platform
// probe refuses to measure: the audit's subject is the Windows desktop, and a stub that returned
// numbers would be a lie the CHECK-VERDICT contract forbids.

package main

import (
	"errors"
	"time"
)

// platformProbes returns nothing: the stand runs only on Windows (strategic section 1).
func platformProbes() []latencyProbe { return nil }

var errNotWindows = errors.New("auditstand is Windows-only (strategic section 1: the subject is the Windows desktop edition)")

// The Windows-only helpers below are declared so the scene code compiles; every call fails.
func findAuditWindows() ([]uintptr, error) { return nil, errNotWindows }

func sampleProcess(pid int) (procSample, bool) { return procSample{PID: pid}, false }

func findProcessesByName(name string) []int { return nil }

func listeningPortsFor(pid int) ([]int, error) { return nil, errNotWindows }

func hostPing(baseURL string, n int, timeout time.Duration) (measurement, error) {
	return measurement{}, errNotWindows
}

func takeFocusTransitions() []focusEvent { return nil }

// setKeyInject is a no-op off Windows: the hook probe does not exist there either.
func setKeyInject(bool) {}

// freeBytes refuses off Windows.
func freeBytes(string) (uint64, error) { return 0, errNotWindows }
