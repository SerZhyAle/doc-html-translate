package main

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// s08Run runs the S08 scene against the stub CLI (the test binary itself) in the given mode and
// returns the outcome checks by member name.
func s08Run(t *testing.T, mode string) map[string]outcomeCheck {
	t.Helper()
	t.Setenv("AUDITSTAND_STUB_CLI", "1")
	t.Setenv("AUDITSTAND_STUB_MODE", mode)
	bin, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	fixtures := filepath.Join(root, "fixtures")
	if err := os.MkdirAll(fixtures, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fixtures, "epub-small.epub"), []byte("not read by the stub"), 0o644); err != nil {
		t.Fatal(err)
	}
	env := &runEnv{
		artifactsDir: filepath.Join(root, "artifacts"),
		fixturesDir:  fixtures,
		appPath:      bin,
		isolation:    "B-env",
		sceneRunID:   "RUN",
	}
	rec, err := runSceneS08(env, context.Background(), 1)
	if err != nil {
		t.Fatalf("scene: %v", err)
	}
	out := map[string]outcomeCheck{}
	for _, oc := range rec.OutcomeChecks {
		out[strings.TrimPrefix(oc.Injection, "S08-")] = oc
	}
	if len(out) != 4 {
		t.Fatalf("members recorded = %d, want 4 (notes: %s)", len(out), rec.Notes)
	}
	return out
}

func okCheck(oc outcomeCheck) bool {
	return oc.NamedOutcome && oc.BoundaryLogFound && oc.CanaryHits == 0 && oc.StateConsistent && (!oc.LocaleChecked || oc.MessageInLocale)
}

func TestS08RefusalPathsPassAgainstACorrectCLI(t *testing.T) {
	checks := s08Run(t, "")
	for name, oc := range checks {
		if !okCheck(oc) {
			t.Errorf("%s: %+v", name, oc)
		}
	}
	if ru := checks["google-cost-guard-ru"]; !ru.LocaleChecked || !ru.MessageInLocale {
		t.Errorf("the ru member must assert the locale part: %+v", ru)
	}
	if checks["google-cost-guard"].LocaleChecked {
		t.Error("the default-language member must not claim a locale verdict")
	}
}

func TestS08LeakedKeyIsACanaryHit(t *testing.T) {
	checks := s08Run(t, "leak-key")
	for _, name := range []string{"google-cost-guard", "google-cost-guard-ru"} {
		oc := checks[name]
		if oc.CanaryHits == 0 || okCheck(oc) {
			t.Errorf("%s: a key echoed into the log and console must be a canary hit: %+v", name, oc)
		}
		if !strings.Contains(oc.CanaryWhere, "console output") {
			t.Errorf("%s: where = %q, want the console named", name, oc.CanaryWhere)
		}
	}
}

func TestS08EnglishMessageUnderRuFailsTheLocalePart(t *testing.T) {
	oc := s08Run(t, "english-ru")["google-cost-guard-ru"]
	if !oc.LocaleChecked || oc.MessageInLocale {
		t.Fatalf("an English refusal under -ui-lang ru must fail H8's locale part: %+v", oc)
	}
	if !oc.NamedOutcome || !oc.StateConsistent {
		t.Errorf("only the locale part may fail here: %+v", oc)
	}
}

func TestS08FailureReportedAsSuccessIsNotANamedOutcome(t *testing.T) {
	oc := s08Run(t, "exit0-nokey")["google-no-key"]
	if oc.NamedOutcome {
		t.Fatalf("exit 0 with no key must not be a named outcome: %+v", oc)
	}
}

func TestS08RecordClaimingFullTranslationIsInconsistentState(t *testing.T) {
	oc := s08Run(t, "claims-full")["google-cost-guard"]
	if oc.StateConsistent {
		t.Fatalf("a completion record saying full after a refusal must be inconsistent: %+v", oc)
	}
}

func TestS08PlaintextKeyLeftBehindIsInconsistentState(t *testing.T) {
	oc := s08Run(t, "no-migrate")["google-cost-guard"]
	if oc.StateConsistent {
		t.Fatalf("a plaintext key that survives the run must be inconsistent: %+v", oc)
	}
}

func TestS08RefusesWhenAKeyFileSitsBesideTheBinary(t *testing.T) {
	t.Setenv("AUDITSTAND_STUB_CLI", "1")
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	bin := filepath.Join(root, "bin", "doc-html-translate.exe")
	if err := os.MkdirAll(filepath.Dir(bin), 0o755); err != nil {
		t.Fatal(err)
	}
	in, err := os.Open(self)
	if err != nil {
		t.Fatal(err)
	}
	defer in.Close()
	dst, err := os.Create(bin)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(dst, in); err != nil {
		t.Fatal(err)
	}
	dst.Close()
	keyPath := filepath.Join(filepath.Dir(bin), "google_api.key")
	if err := os.WriteFile(keyPath, []byte("someone's real key"), 0o600); err != nil {
		t.Fatal(err)
	}
	env := &runEnv{artifactsDir: filepath.Join(root, "artifacts"), fixturesDir: root, appPath: bin, isolation: "B-env", sceneRunID: "RUN"}
	_, err = runSceneS08(env, context.Background(), 1)
	if err == nil || !strings.Contains(err.Error(), "next to the app binary") {
		t.Fatalf("err = %v, want the beside-the-binary refusal", err)
	}
	if b, _ := os.ReadFile(keyPath); string(b) != "someone's real key" {
		t.Fatal("the stand let the app consume a key that was not its own")
	}
}

func TestS08ScriptTestSeparatesLocalizedFromEnglish(t *testing.T) {
	en := "[3/4] Translation skipped - estimated cost $0.05 USD exceeds -max-cost $0.01 limit"
	ru := "[3/4] Перевод пропущен - стоимость $0.05 USD превышает -max-cost $0.01"
	if lineInScript(en, "-max-cost", "ru") {
		t.Error("the English literal must not pass as Cyrillic")
	}
	if !lineInScript(ru, "-max-cost", "ru") {
		t.Error("a Cyrillic line must pass")
	}
	if lineInScript(ru, "-max-cost", "de") {
		t.Error("Latin-script languages have no script verdict; the stand must not invent one")
	}
}

// The scene's expectations are copies of the pipeline's contract; a change there must fail here.
func TestS08ExpectationsMatchThePipeline(t *testing.T) {
	read := func(parts ...string) string {
		b, err := os.ReadFile(filepath.Join(append([]string{"..", ".."}, parts...)...))
		if err != nil {
			t.Skip("input absent:", err)
		}
		return string(b)
	}
	pipelineGo := read("internal", "pipeline", "pipeline.go")
	if !strings.Contains(pipelineGo, "ExitAPI       = 4") {
		t.Errorf("pipeline.ExitAPI is no longer 4; update exitAPI in scene_s08.go")
	}
	costGo := read("internal", "pipeline", "cost.go")
	if !strings.Contains(costGo, "exceeds -max-cost") {
		t.Errorf("cost.go no longer says \"exceeds -max-cost\"; update the google-cost-guard boundary")
	}
	translateGo := read("internal", "pipeline", "translate.go")
	for _, phrase := range []string{"Google Translate skipped - API key not available"} {
		if !strings.Contains(translateGo, phrase) {
			t.Errorf("translate.go no longer says %q; update the google-no-key boundary", phrase)
		}
	}
}
