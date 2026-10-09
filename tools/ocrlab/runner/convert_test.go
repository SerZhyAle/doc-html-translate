package runner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The engine identity must move with the executable's bytes: two builds that print the same
// version line are different engines, and the reuse key holds this string equal.
func TestBinaryIdentityFollowsTheBytes(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(body), 0o755); err != nil {
			t.Fatal(err)
		}
		return p
	}
	a := binaryIdentity(write("a.exe", "build one"))
	b := binaryIdentity(write("b.exe", "build two"))
	if !strings.HasPrefix(a, " (exe sha256:") || len(a) != len(" (exe sha256:)")+12 {
		t.Fatalf("identity = %q, want a 12-hex prefix", a)
	}
	if a == b {
		t.Fatalf("different bytes gave the same identity %q", a)
	}
	if again := binaryIdentity(filepath.Join(dir, "a.exe")); again != a {
		t.Fatalf("same bytes gave %q then %q", a, again)
	}
	if got := binaryIdentity(filepath.Join(dir, "missing.exe")); got != "" {
		t.Fatalf("an unreadable executable must add nothing, got %q", got)
	}
}
