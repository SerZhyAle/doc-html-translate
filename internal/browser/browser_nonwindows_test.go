//go:build !windows

package browser

import "testing"

// Off Windows Open must fail rather than silently do nothing, so the caller can tell the
// user where the output is instead of claiming it was opened.
func TestOpenUnsupportedOffWindows(t *testing.T) {
	if err := Open("/tmp/book/index.html"); err == nil {
		t.Error("Open succeeded off Windows")
	}
}
