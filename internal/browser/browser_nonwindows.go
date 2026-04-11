//go:build !windows

package browser

import "errors"

// Open is not supported on non-Windows platforms.
func Open(_ string) error {
	return errors.New("browser open is supported only on Windows")
}
