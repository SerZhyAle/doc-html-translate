package pipeline

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"doc-html-translate/internal/config"
	"doc-html-translate/internal/fdsec"
	"doc-html-translate/internal/fdsec/fdsectest"
	"doc-html-translate/internal/logging"
)

func TestMain(m *testing.M) {
	fdsectest.RunFakeIfAsked()
	os.Exit(m.Run())
}

const (
	secretVar = "DOCHT_TEST_SECRET_PW"
	// A password no log, report, error or page may contain.
	knownPassword = "Tr0ub4dor&3-known-password"
)

// secretSandbox is a sandbox whose FileDO is the stand-in, with a container the length screen
// accepts and the password in a variable named by cfg.FdsecPasswordEnv.
func secretSandbox(t *testing.T, mode string) (*pipelineSandbox, config.Config) {
	t.Helper()
	s := newPipelineSandbox(t)
	t.Setenv(fdsectest.Env, mode)
	t.Setenv(fdsec.OverrideEnv, os.Args[0])
	t.Setenv(secretVar, knownPassword)
	in := filepath.Join(s.dir, "report.fd-sec")
	if err := os.WriteFile(in, make([]byte, 12288), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := s.config(in)
	cfg.FdsecPasswordEnv = secretVar
	return s, cfg
}

// runLogged runs cfg and returns the run log with the result.
func runLogged(s *pipelineSandbox, cfg config.Config) (code int, log string, err error) {
	var buf bytes.Buffer
	logging.StartRunLog(&buf)
	defer logging.StopRunLog()
	code, err = s.run(cfg)
	return code, buf.String(), err
}

func workRootEntries(t *testing.T) []string {
	t.Helper()
	root := filepath.Join(os.TempDir(), "doc-html-translate-fdsec")
	entries, err := os.ReadDir(root)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

// everyOutputText is the names and contents of everything a run left in a folder.
func everyOutputText(t *testing.T, dir string) string {
	t.Helper()
	var sb strings.Builder
	_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		sb.WriteString(p + "\n")
		if !d.IsDir() {
			b, _ := os.ReadFile(p)
			sb.Write(b)
		}
		return nil
	})
	return sb.String()
}

func TestSecretFileConvertsAndLeavesNothingBehind(t *testing.T) {
	s, cfg := secretSandbox(t, "ok-txt")
	code, log, err := runLogged(s, cfg)
	if code != ExitOK || err != nil {
		t.Fatalf("code = %d, err = %v\nlog:\n%s", code, err, log)
	}
	out := s.outputDir(cfg.InputFile)
	if filepath.Base(out) != "report" {
		t.Errorf("the result is named after the container, got %s", out)
	}
	if !s.exists(filepath.Join(out, "index.html")) {
		t.Fatal("no index.html")
	}
	rec := s.record(cfg.InputFile)
	if rec == nil {
		t.Fatal("no completion record")
	}
	if names := workRootEntries(t); len(names) != 0 {
		t.Errorf("a plain copy was left behind: %v", names)
	}
	if !strings.Contains(log, "are not encrypted") {
		t.Errorf("the run must say the converted pages are not encrypted:\n%s", log)
	}
	for _, text := range []string{log, everyOutputText(t, out)} {
		for _, secret := range []string{knownPassword, fdsectest.TrueName} {
			if strings.Contains(text, secret) {
				t.Errorf("%q leaked into the log or the result", secret)
			}
		}
	}
	// The container is the marker's source, so every GUI action keeps working on it.
	if r := s.reuse(cfg); r != 0 {
		t.Errorf("a finished result of the unchanged container must be reusable, reason %v", r)
	}
}

func TestFinishedResultIsReusedWithoutAskingForTheSecret(t *testing.T) {
	s, cfg := secretSandbox(t, "ok-txt")
	s.mustRun(cfg)
	sentinel := s.plantSentinel(cfg.InputFile)

	// Nothing can answer a question now: no variable, and the fake would fail the call.
	cfg.FdsecPasswordEnv = ""
	t.Setenv(fdsectest.Env, "damaged")
	if code, err := s.run(cfg); code != ExitOK || err != nil {
		t.Fatalf("a reuse must not need FileDO or a password: code = %d, err = %v", code, err)
	}
	if !s.exists(sentinel) {
		t.Error("the second run rebuilt instead of reusing")
	}

	// A forced rebuild asks again - and with nobody to ask it stops, at once.
	cfg.Force = true
	code, err := s.run(cfg)
	if code != ExitArgsError && code != ExitParse {
		t.Errorf("a forced rebuild must need the secret again, code = %d err = %v", code, err)
	}
}

func TestSecretFileFailureClasses(t *testing.T) {
	cases := []struct {
		mode     string
		password string
		wantCode int
		wantText string
	}{
		{"wrong", "not-the-right-password", ExitParse, "the password is wrong, or the file is not a FileDO secret file, or it was altered"},
		{"damaged", knownPassword, ExitParse, "damaged or cut short"},
		{"unsupported", knownPassword, ExitParse, "kind of secret file it does not support"},
		{"usage", knownPassword, ExitParse, "did not accept the call"},
		{"crash", knownPassword, ExitParse, "stopped unexpectedly"},
		{"io", knownPassword, ExitIOError, "could not read or write a file"},
		{"ok-folder", knownPassword, ExitParse, "holds a folder"},
		{"ok-nested", knownPassword, ExitParse, "holds another secret file"},
		{"ok-docx", knownPassword, ExitParse, "which this app does not convert"},
	}
	for _, c := range cases {
		t.Run(c.mode, func(t *testing.T) {
			s, cfg := secretSandbox(t, c.mode)
			t.Setenv(secretVar, c.password)
			code, log, err := runLogged(s, cfg)
			if code != c.wantCode || err == nil || !strings.Contains(err.Error(), c.wantText) {
				t.Fatalf("code = %d (want %d), err = %v (want %q)", code, c.wantCode, err, c.wantText)
			}
			if s.exists(s.outputDir(cfg.InputFile)) {
				t.Error("a refused container must leave no output folder")
			}
			if names := workRootEntries(t); len(names) != 0 {
				t.Errorf("a plain copy was left behind: %v", names)
			}
			for _, text := range []string{err.Error(), log} {
				for _, secret := range []string{c.password, fdsectest.TrueName} {
					if strings.Contains(text, secret) {
						t.Errorf("%q leaked into %q", secret, text)
					}
				}
			}
		})
	}
}

func TestSecretFileRefusalsComeBeforeTheSecret(t *testing.T) {
	t.Run("FileDO missing", func(t *testing.T) {
		s, cfg := secretSandbox(t, "ok-txt")
		t.Setenv(fdsec.OverrideEnv, filepath.Join(s.dir, "no-such-filedo.exe"))
		os.Unsetenv(secretVar) // a read of the variable would be ExitArgsError instead
		code, _, err := runLogged(s, cfg)
		if code != ExitParse || err == nil || !strings.Contains(err.Error(), "winget install SerZhyAle.FileDO") {
			t.Fatalf("code = %d, err = %v", code, err)
		}
		if s.exists(s.outputDir(cfg.InputFile)) {
			t.Error("no output folder may be left")
		}
	})
	t.Run("length screen", func(t *testing.T) {
		s, cfg := secretSandbox(t, "ok-txt")
		if err := os.WriteFile(cfg.InputFile, make([]byte, 100), 0o644); err != nil {
			t.Fatal(err)
		}
		os.Unsetenv(secretVar)
		code, _, err := runLogged(s, cfg)
		if code != ExitParse || err == nil || !strings.Contains(err.Error(), "its size rules it out") {
			t.Fatalf("code = %d, err = %v", code, err)
		}
		if s.exists(s.outputDir(cfg.InputFile)) {
			t.Error("no output folder may be left")
		}
	})
}

func TestSecretFileNamedVariableUnset(t *testing.T) {
	s, cfg := secretSandbox(t, "ok-txt")
	os.Unsetenv(secretVar)
	code, _, err := runLogged(s, cfg)
	if code != ExitArgsError || err == nil || !strings.Contains(err.Error(), secretVar) {
		t.Fatalf("code = %d, err = %v", code, err)
	}
}

func TestSecretFilePicture(t *testing.T) {
	s, cfg := secretSandbox(t, "ok-png")
	code, log, err := runLogged(s, cfg)
	if code != ExitOK || err != nil {
		t.Fatalf("code = %d, err = %v\nlog:\n%s", code, err, log)
	}
	rec := s.record(cfg.InputFile)
	if rec == nil || !rec.OCRForced {
		t.Errorf("a picture inside a secret file is converted as a picture is, with the OCR overlay forced: %+v", rec)
	}
}
