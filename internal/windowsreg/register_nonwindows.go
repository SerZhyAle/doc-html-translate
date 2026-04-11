//go:build !windows

package windowsreg

import "errors"

// SupportedExtensions mirrors the Windows implementation.
var SupportedExtensions = []string{".epub", ".pdf", ".txt", ".md", ".fb2", ".rtf", ".html", ".htm"}

func RegisterHandler() ([]string, error) {
	return nil, errors.New("windows registry registration is supported only on Windows")
}
