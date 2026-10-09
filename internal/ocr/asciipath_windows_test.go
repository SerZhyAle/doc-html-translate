//go:build windows

package ocr

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The real Windows limit against a real directory tree: the path exceeds MAX_PATH, so what comes
// back must be readable under it - an 8.3 alias where the volume keeps short names, else a copy.
func TestStageASCIIPathHandlesARealOver260CharPath(t *testing.T) {
	dir := t.TempDir()
	for len(dir) <= 270 {
		dir = filepath.Join(dir, strings.Repeat("n", 60))
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Skipf("cannot create a path over MAX_PATH on this file system: %v", err)
	}
	data := []byte("windows long path payload")
	src := filepath.Join(dir, "page.png")
	if err := os.WriteFile(src, data, 0o644); err != nil {
		t.Skipf("cannot write under a path over MAX_PATH: %v", err)
	}
	if maxEnginePath != 259 {
		t.Fatalf("maxEnginePath = %d, want 259 on Windows", maxEnginePath)
	}
	useStagingRoot(t, []string{t.TempDir()}, shortPath)

	staged, cleanup := stageASCIIPath(src)
	defer cleanup()
	if len(staged) > maxEnginePath {
		t.Fatalf("stageASCIIPath returned %d characters, over the engine limit", len(staged))
	}
	if got, err := os.ReadFile(staged); err != nil || !bytes.Equal(got, data) {
		t.Errorf("staged file = %q, %v; want the original bytes", got, err)
	}
}
