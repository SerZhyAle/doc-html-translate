//go:build windows

package browser

import (
	"fmt"
	"os/exec"
	"syscall"
)

// Open opens the given file or URL in the default Windows browser.
func Open(target string) error {
	target = normalizeTarget(target)
	cmd := exec.Command("cmd", "/c", "start", "", target)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("open browser: %w", err)
	}
	return nil
}
