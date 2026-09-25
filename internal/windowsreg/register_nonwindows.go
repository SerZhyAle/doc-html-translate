//go:build !windows

package windowsreg

import "errors"

// SupportedExtensions mirrors the Windows implementation.
var SupportedExtensions = []string{".epub", ".pdf", ".txt", ".md", ".fb2", ".rtf", ".html", ".htm", ".mobi", ".azw3", ".cbz", ".cbr", ".cb7", ".cbt"}

var errUnsupported = errors.New("windows registry registration is supported only on Windows")

func RegisterHandler() (Registration, error) {
	return Registration{}, errUnsupported
}

func RegisterOpenWith() ([]string, error) {
	return nil, errUnsupported
}

func RegisterOpenWithFor(string) ([]string, error) {
	return nil, errUnsupported
}

func RegisterContextMenu() ([]string, error) {
	return nil, errUnsupported
}

func RegisterContextMenuFor(string) ([]string, error) {
	return nil, errUnsupported
}

func HasShellEntries() bool {
	return false
}

func RemoveShellEntries() ([]string, error) {
	return nil, errUnsupported
}

func Unregister() ([]string, error) {
	return nil, errUnsupported
}

func HandlerStatus() Status {
	return Status{Other: append([]string(nil), SupportedExtensions...)}
}

func IsDefaultHandler() bool {
	return false
}

func OpenDefaultAppsSettings() error {
	return errUnsupported
}
