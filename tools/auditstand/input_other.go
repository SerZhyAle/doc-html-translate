//go:build !windows

package main

func injectAuditTab(uintptr) error { return errNotWindows }
