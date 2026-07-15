//go:build !windows

package windowsreg

import "errors"

// SupportedExtensions mirrors the Windows implementation.
var SupportedExtensions = []string{".epub", ".pdf", ".txt", ".md", ".fb2", ".rtf", ".html", ".htm", ".mobi", ".azw3"}

func RegisterHandler() ([]string, error) {
	return nil, errors.New("windows registry registration is supported only on Windows")
}

func RegisterOpenWith() ([]string, error) {
	return nil, errors.New("windows registry registration is supported only on Windows")
}

func RegisterOpenWithFor(string) ([]string, error) {
	return nil, errors.New("windows registry registration is supported only on Windows")
}

func RegisterContextMenu() ([]string, error) {
	return nil, errors.New("windows registry registration is supported only on Windows")
}

func RegisterContextMenuFor(string) ([]string, error) {
	return nil, errors.New("windows registry registration is supported only on Windows")
}

func Unregister() ([]string, error) {
	return nil, errors.New("windows registry registration is supported only on Windows")
}

func IsDefaultHandler() bool {
	return false
}
