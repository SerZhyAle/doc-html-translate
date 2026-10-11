//go:build !windows

package main

func processCPUTimes() (map[int]cpuCounter, error) { return nil, errNotWindows }
func desktopUnlocked() error                       { return errNotWindows }
