package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"doc-html-translate/internal/app"
	"doc-html-translate/internal/config"
	"doc-html-translate/internal/fdsec"
)

// TestFdsecRealFileDO is the one end-to-end run against the real, installed FileDO: it seals a
// text file, converts the container with the password handed over through a variable, and checks
// the result is named after the container and no plain copy is left in the work root. Without a
// FileDO on the machine it skips - and a skip is "COULD NOT VERIFY", never a pass.
func TestFdsecRealFileDO(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping the real FileDO run in short mode")
	}
	filedo, err := fdsec.Locate()
	if err != nil {
		t.Skip("FileDO not installed - COULD NOT VERIFY the real hand-off")
	}

	const password = "e2e-test-password-94"
	work := t.TempDir()
	// The work root lives under the temp folder: point it at the test's own folder so a leftover
	// is visible here and nothing touches the machine's real temp.
	t.Setenv("TMP", work)
	t.Setenv("TEMP", work)
	t.Setenv("TMPDIR", work)
	t.Setenv("DOCHT_E2E_FDSEC_PW", password)

	src := filepath.Join(work, "doc.txt")
	const body = "A short document sealed by the real FileDO.\n\nIt has two paragraphs, so the converter has something to wrap navigation around.\n"
	if err := os.WriteFile(src, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	// --no-history: FileDO must not drop a history.json into this package's folder.
	if out, err := exec.Command(filedo, "--no-history", src, "secure", "p:"+password, "-y").CombinedOutput(); err != nil {
		t.Skipf("this FileDO could not make a container (%v): COULD NOT VERIFY the real hand-off\n%s", err, out)
	}
	sealed, _ := filepath.Glob(filepath.Join(work, "*.fd-sec"))
	if len(sealed) != 1 {
		t.Skipf("expected one .fd-sec after sealing, found %v: COULD NOT VERIFY the real hand-off", sealed)
	}
	container := filepath.Join(work, "probe.fd-sec")
	if err := os.Rename(sealed[0], container); err != nil {
		t.Fatal(err)
	}

	cfg := config.Config{
		InputFile:        container,
		NoTranslate:      true,
		NoOpen:           true,
		SinglePage:       true,
		SourceLang:       "en",
		TargetLang:       "ru",
		FdsecPasswordEnv: "DOCHT_E2E_FDSEC_PW",
	}
	if code, err := app.New(cfg).Run(); err != nil || code != 0 {
		t.Fatalf("convert: exit=%d err=%v", code, err)
	}

	index := filepath.Join(work, "probe", "index.html")
	raw, err := os.ReadFile(index)
	if err != nil {
		t.Fatalf("the result must be named after the container: %v", err)
	}
	if !strings.Contains(string(raw), "two paragraphs") {
		t.Error("the converted page does not carry the sealed document's text")
	}
	if entries, _ := os.ReadDir(filepath.Join(work, "doc-html-translate-fdsec")); len(entries) != 0 {
		t.Errorf("a plain copy was left in the work root: %d entries", len(entries))
	}
}
