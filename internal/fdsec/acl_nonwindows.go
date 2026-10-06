//go:build !windows

package fdsec

import "os"

// restrictToUser creates dir readable by its owner only.
func restrictToUser(dir string) error { return os.Mkdir(dir, 0o700) }
