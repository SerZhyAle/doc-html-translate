package ocr

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWingetAliasUsesRealExecutableDirectory(t *testing.T) {
	root := t.TempDir()
	install := filepath.Join(root, "installed")
	links := filepath.Join(root, "WinGet", "Links")
	for _, dir := range []string{install, links, filepath.Join(install, "tessdata"), filepath.Join(install, "tesseract")} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	realExe := filepath.Join(install, "doc-html-translate.exe")
	alias := filepath.Join(links, "doc-html-translate.exe")
	for _, path := range []string{realExe, langFile(filepath.Join(install, "tessdata"), "eng"), filepath.Join(install, "tesseract", tesseractExeName())} {
		if err := os.WriteFile(path, []byte("test"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(realExe, alias); err != nil {
		t.Skipf("cannot create a winget-style executable symlink: %v", err)
	}
	saved := executablePath
	executablePath = func() (string, error) { return alias, nil }
	t.Cleanup(func() { executablePath = saved })
	savedUserDataDir := userDataDir
	userDataDir = func() string { return filepath.Join(root, "user", "tessdata") }
	t.Cleanup(func() { userDataDir = savedUserDataDir })

	if got := bundledDataDir(); got != filepath.Join(install, "tessdata") {
		t.Errorf("bundledDataDir = %q, want data next to the real executable", got)
	}
	if !IsInstalled("eng") {
		t.Error("bundled English data was not found through the alias")
	}
	if got, err := DataDirFor("eng"); err != nil || got != filepath.Join(install, "tessdata") {
		t.Errorf("DataDirFor(eng) = %q, %v", got, err)
	}
	if got := stagingCandidates(); got[len(got)-1] != filepath.Join(install, stagingDirName) {
		t.Errorf("last staging candidate = %q, want directory next to the real executable", got[len(got)-1])
	}
	if got, err := Locate(); err != nil || got != filepath.Join(install, "tesseract", tesseractExeName()) {
		t.Errorf("Locate = %q, %v", got, err)
	}
}
