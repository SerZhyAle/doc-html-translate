package commands

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"doc-html-translate/internal/ocr"
	"doc-html-translate/internal/readiness"
)

// machine installs probes for a fake machine: no Calibre, 7-Zip and FileDO present, Tesseract found with every language,
// no Google key, and an Ollama probe that fails the test when it is called. It puts the probe set back at the end.
func machine(t *testing.T) {
	t.Helper()
	saved := readiness.Probes
	t.Cleanup(func() { readiness.Probes = saved })
	readiness.Probes.Calibre = func() bool { return false }
	readiness.Probes.SevenZip = func() bool { return true }
	readiness.Probes.FileDO = func() bool { return true }
	readiness.Probes.PDFToText = func() bool { return false }
	readiness.Probes.OCRLocate = func() (string, error) { return `C:\Tools\tesseract.exe`, nil }
	readiness.Probes.OCRMissing = func(context.Context, string, string) []string { return nil }
	readiness.Probes.OCRLangs = func() []string { return []string{"eng"} }
	readiness.Probes.GoogleKey = func() (string, error) { return "", errors.New("no key") }
	readiness.Probes.Ollama = func(context.Context, string) (bool, error) {
		t.Error("the Ollama probe was called")
		return false, errors.New("must not be called")
	}
	readiness.Probes.Version = func(context.Context, string) string { return "1.2.3" }
	readiness.Probes.Packaged = func() bool { return false }
}

// run runs the front on args with the console converter's settings and returns what it wrote.
func run(t *testing.T, args ...string) (c *capture, handled bool, code int) {
	t.Helper()
	c = &capture{}
	handled, code = Front(args, c.env())
	return c, handled, code
}

func writeDoc(t *testing.T, name string, size int) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, make([]byte, size), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestDoctorVerb(t *testing.T) {
	machine(t)

	c, handled, code := run(t, "doctor", "--json")
	if !handled || code != ExitDone || c.err.Len() != 0 || c.opened != 1 {
		t.Fatalf("handled=%v exit=%d opened=%d stderr=%q", handled, code, c.opened, c.err.String())
	}
	if n := strings.Count(c.out.String(), "\n"); n != 1 {
		t.Errorf("--json prints one line, got %d: %q", n, c.out.String())
	}
	var rep readiness.Report
	if err := json.Unmarshal(c.out.Bytes(), &rep); err != nil {
		t.Fatalf("not JSON: %v: %s", err, c.out.String())
	}
	if rep.Version != testVersion || rep.Tools.Calibre.Found || !rep.Tools.SevenZip.Found || rep.Tools.SevenZip.Version != "1.2.3" || rep.Ollama != nil {
		t.Errorf("report: %+v", rep)
	}
	mobi := false
	for _, f := range rep.Formats {
		if f.Ext == ".mobi" {
			mobi = f.Tool == "calibre" && !f.Ready
		}
	}
	if !mobi {
		t.Errorf(".mobi does not name the missing Calibre: %+v", rep.Formats)
	}

	// Without --json: key: value lines.
	c, _, code = run(t, "doctor")
	for _, want := range []string{
		"version: " + testVersion + "\n", "packaged: false\n", "tool tesseract: found 1.2.3\n", "tool calibre: missing\n",
		"tool pdftotext: missing\n", "ocrLangs: eng\n", "googleKeyPresent: false\n", "format .mobi: not ready, needs calibre\n",
		"format .epub: ready\n", "format .cbr: ready, needs sevenzip, optional tesseract\n",
	} {
		if code != ExitDone || !strings.Contains(c.out.String(), want) {
			t.Errorf("exit %d: the readable answer has no %q:\n%s", code, want, c.out.String())
		}
	}
	if strings.Contains(c.out.String(), "ollama") {
		t.Errorf("the readable answer names Ollama without --ollama:\n%s", c.out.String())
	}

	// --result is the same data, written whole; standard output still answers.
	res := filepath.Join(t.TempDir(), "doctor.json")
	c, _, code = run(t, "doctor", "--json", "--result", res)
	b, err := os.ReadFile(res)
	if code != ExitDone || err != nil || string(b) != c.out.String() {
		t.Errorf("exit %d, read %v: the result file is not the printed data", code, err)
	}
	// A result that cannot be written is exit 1 with nothing on standard output.
	c, _, code = run(t, "doctor", "--result", filepath.Join(t.TempDir(), "no", "such", "dir", "d.json"))
	if code != ExitFailed || c.out.Len() != 0 || !strings.Contains(c.err.String(), "the result file could not be written") {
		t.Errorf("unwritable result: exit %d stdout=%q stderr=%q", code, c.out.String(), c.err.String())
	}

	// --ollama asks the local server for the model named, and only then.
	asked := ""
	readiness.Probes.Ollama = func(_ context.Context, model string) (bool, error) { asked = model; return true, nil }
	c, _, code = run(t, "doctor", "--ollama", "--ollama-model", "m1", "--json")
	if err := json.Unmarshal(c.out.Bytes(), &rep); err != nil || code != ExitDone ||
		rep.Ollama == nil || !rep.Ollama.Reachable || !rep.Ollama.ModelPresent || rep.Ollama.Model != "m1" || asked != "m1" {
		t.Errorf("exit %d asked %q ollama=%+v err=%v", code, asked, rep.Ollama, err)
	}
	c, _, _ = run(t, "doctor", "--ollama")
	if !strings.Contains(c.out.String(), "ollama: model "+readiness.DefaultOllamaModel+", reachable true, modelPresent true\n") {
		t.Errorf("the readable Ollama line: %s", c.out.String())
	}
}

func TestInputCheckVerb(t *testing.T) {
	machine(t)
	mobi := writeDoc(t, "book.mobi", 10)
	epub := writeDoc(t, "book.epub", 10)
	empty := writeDoc(t, "empty.epub", 0)

	check := func(args ...string) (readiness.Result, int, *capture) {
		t.Helper()
		c, handled, code := run(t, append([]string{"input", "check"}, args...)...)
		if !handled {
			t.Fatalf("%q was not handled", args)
		}
		var res readiness.Result
		if code == ExitDone && contains(args, "--json") {
			if err := json.Unmarshal(c.out.Bytes(), &res); err != nil {
				t.Fatalf("not JSON: %v: %s", err, c.out.String())
			}
		}
		return res, code, c
	}

	res, code, c := check("--in", mobi, "--json")
	if code != ExitDone || res.State != "action" || len(res.Issues) != 1 || res.Issues[0].Code != "calibre" || res.Issues[0].Level != "action" {
		t.Errorf("a .mobi with no Calibre: exit %d %+v", code, res)
	}
	if strings.Contains(c.out.String(), "googleLimit") {
		t.Errorf("the window's display member is part of the data: %s", c.out.String())
	}
	res, code, c = check("--in", epub, "--json")
	if code != ExitDone || res.State != "ready" || res.Issues == nil || len(res.Issues) != 0 || !strings.Contains(c.out.String(), `"issues":[]`) {
		t.Errorf("a plain .epub: exit %d %+v %s", code, res, c.out.String())
	}
	dir := t.TempDir()
	for name, in := range map[string]string{"an empty file": empty, "a folder": dir} {
		res, code, _ = check("--in", in, "--json")
		if code != ExitDone || res.State != "action" || len(res.Issues) != 1 || res.Issues[0].Code != "input" {
			t.Errorf("%s: exit %d %+v", name, code, res)
		}
	}

	// A path that does not exist is exit 11 with the sentence, and nothing on standard output.
	_, code, c = check("--in", filepath.Join(dir, "none.epub"), "--json")
	if code != ExitNotFound || c.out.Len() != 0 || !strings.Contains(c.err.String(), "the document does not exist") {
		t.Errorf("missing file: exit %d stdout=%q stderr=%q", code, c.out.String(), c.err.String())
	}
	// No --in, an unknown option and a stray word are usage.
	for _, args := range [][]string{{}, {"--google"}, {"--in", epub, "--nope"}, {"--in", epub, "extra"}, {"--in"}} {
		_, code, c = check(args...)
		if code != ExitUsage || c.out.Len() != 0 {
			t.Errorf("%q: exit %d stdout=%q, want 10 and nothing", args, code, c.out.String())
		}
	}
	_, _, c = check()
	if !strings.Contains(c.err.String(), `missing required --in for "input check"`) {
		t.Errorf("no --in: %q", c.err.String())
	}

	// Without --json: the state, then one line per issue.
	_, _, c = check("--in", mobi)
	if c.out.String() != "state: action\nissue: calibre action\n" {
		t.Errorf("readable answer: %q", c.out.String())
	}
	_, _, c = check("--in", epub)
	if c.out.String() != "state: ready\n" {
		t.Errorf("readable answer for a ready document: %q", c.out.String())
	}
	// --result is the data, whole.
	out := filepath.Join(t.TempDir(), "check.json")
	_, code, c = check("--in", mobi, "--json", "--result", out)
	if b, err := os.ReadFile(out); code != ExitDone || err != nil || string(b) != c.out.String() {
		t.Errorf("exit %d read %v: the result file is not the printed data", code, err)
	}
}

// The options of the verb are read by the converter's own parser and reach the shared check as the window's request
// would: the source language derives the OCR data, --free is --ollama, the default model is the converter's, and --google
// wins over --ollama.
func TestInputCheckOptionsReachTheSharedCheck(t *testing.T) {
	machine(t)
	epub := writeDoc(t, "book.epub", 10)
	var lang, model string
	var keyAsked, ollamaAsked bool
	readiness.Probes.OCRMissing = func(_ context.Context, _, l string) []string { lang = l; return []string{"x"} }
	readiness.Probes.GoogleKey = func() (string, error) { keyAsked = true; return "", errors.New("none") }
	readiness.Probes.Ollama = func(_ context.Context, m string) (bool, error) { ollamaAsked = true; model = m; return true, nil }

	reset := func() { lang, model, keyAsked, ollamaAsked = "", "", false, false }
	for _, tc := range []struct {
		args        []string
		wantLang    string
		wantModel   string
		key, ollama bool
	}{
		{[]string{"--ocr"}, "eng", "", false, false},
		{[]string{"--ocr", "--src", "ru"}, "rus", "", false, false},
		{[]string{"--ocr", "--ocr-lang", "deu+fra", "--src", "ru"}, "deu+fra", "", false, false},
		{[]string{"--ollama"}, "", readiness.DefaultOllamaModel, false, true},
		{[]string{"--ollama", "--ollama-model", "m2"}, "", "m2", false, true},
		{[]string{"--google", "--ollama"}, "", "", true, false},
		{[]string{"--ollama-model", "m3"}, "", "", false, false},
		{nil, "", "", false, false},
	} {
		reset()
		_, handled, code := run(t, append([]string{"input", "check", "--in", epub, "--json"}, tc.args...)...)
		if !handled || code != ExitDone {
			t.Errorf("%q: handled=%v exit=%d", tc.args, handled, code)
			continue
		}
		if lang != tc.wantLang || model != tc.wantModel || keyAsked != tc.key || ollamaAsked != tc.ollama {
			t.Errorf("%q: OCR language %q (want %q), Ollama model %q (want %q), key asked %v (want %v), Ollama asked %v (want %v)",
				tc.args, lang, tc.wantLang, model, tc.wantModel, keyAsked, tc.key, ollamaAsked, tc.ollama)
		}
	}
	if c, handled, code := run(t, "input", "check", "--in", epub, "--free"); !handled || code != ExitUsage || !strings.Contains(c.err.String(), `unknown option "--free"`) {
		t.Errorf("input check --free: handled=%v exit=%d err=%q", handled, code, c.err.String())
	}
}

func TestOcrLangsVerb(t *testing.T) {
	c, handled, code := run(t, "ocr", "langs", "--json")
	if !handled || code != ExitDone || c.err.Len() != 0 {
		t.Fatalf("handled=%v exit=%d stderr=%q", handled, code, c.err.String())
	}
	var data struct {
		Languages []ocr.LangRow `json:"languages"`
	}
	if err := json.Unmarshal(c.out.Bytes(), &data); err != nil {
		t.Fatalf("not JSON: %v: %s", err, c.out.String())
	}
	if len(data.Languages) != len(ocr.Available) {
		t.Fatalf("%d languages for a catalogue of %d", len(data.Languages), len(ocr.Available))
	}
	have := map[string]bool{}
	for _, code := range ocr.Installed() {
		have[code] = true
	}
	for i, l := range data.Languages {
		if l.Code != ocr.Available[i].Code || l.Name != ocr.Available[i].Name || l.Installed != have[l.Code] {
			t.Errorf("row %d = %+v, want %+v installed %v", i, l, ocr.Available[i], have[l.Code])
		}
	}
	if strings.Contains(c.out.String(), `:\`) || strings.Contains(c.out.String(), "tessdata") {
		t.Errorf("the data names a folder: %s", c.out.String())
	}

	// Without --json it is the listing the frozen alias prints, byte for byte.
	c, _, code = run(t, "ocr", "langs")
	if code != ExitDone || c.out.String() != ocr.LangsText() {
		t.Errorf("exit %d: the readable answer is not the listing of -ocr-langs:\n%s", code, c.out.String())
	}
	// --result is the data, whole.
	out := filepath.Join(t.TempDir(), "langs.json")
	c, _, code = run(t, "ocr", "langs", "--json", "--result", out)
	if b, err := os.ReadFile(out); code != ExitDone || err != nil || string(b) != c.out.String() {
		t.Errorf("exit %d read %v: the result file is not the printed data", code, err)
	}
	// The legacy spelling is not the front's: the converter prints the listing.
	if _, handled, _ := run(t, "-ocr-langs"); handled {
		t.Error("the frozen spelling -ocr-langs was taken by the front")
	}
}

func TestInspectGroupRefusals(t *testing.T) {
	machine(t)
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{"doctor unknown option", []string{"doctor", "--nope"}, `unknown option "--nope" for "doctor"`},
		{"doctor stray word", []string{"doctor", "extra"}, `unexpected value "extra" for "doctor"`},
		{"doctor single dash", []string{"doctor", "-json"}, `unknown option "-json"`},
		{"doctor bad bool", []string{"doctor", "--json=maybe"}, `unexpected value "maybe" for "--json"`},
		{"doctor result without a value", []string{"doctor", "--result"}, `option "--result" needs a value`},
		{"doctor repeated", []string{"doctor", "--ollama-model", "a", "--ollama-model", "b"}, `option "--ollama-model" was given more than once`},
		{"ocr langs unknown option", []string{"ocr", "langs", "--nope"}, `unknown option "--nope" for "ocr langs"`},
		{"ocr langs stray word", []string{"ocr", "langs", "rus"}, `unexpected value "rus" for "ocr langs"`},
		{"ocr alone", []string{"ocr"}, `unknown command "ocr"`},
		{"input alone", []string{"input", "--in", "a.epub"}, `unknown command "input"`},
		{"input check without a document", []string{"input", "check"}, `missing required --in for "input check"`},
		{"input check unknown option", []string{"input", "check", "--in", "a.epub", "--force"}, `unknown option "--force" for "input check"`},
		{"input check empty document", []string{"input", "check", "--in", ""}, `unexpected value "" for "--in"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			t.Chdir(dir)
			c, handled, code := run(t, tc.args...)
			if !handled || code != ExitUsage || c.out.Len() != 0 || !strings.Contains(c.err.String(), tc.want) {
				t.Errorf("handled=%v exit=%d stdout=%q stderr=%q, want 10 and %q", handled, code, c.out.String(), c.err.String(), tc.want)
			}
			if entries, _ := os.ReadDir(dir); len(entries) != 0 {
				t.Errorf("a refusal wrote files: %v", entries)
			}
		})
	}
}

// Both executables answer the group from the same code: the launcher has no conversion hook and needs none.
func TestInspectGroupIsServedByTheLauncher(t *testing.T) {
	machine(t)
	epub := writeDoc(t, "book.epub", 10)
	for _, args := range [][]string{
		{"doctor", "--json"}, {"input", "check", "--in", epub, "--json"}, {"ocr", "langs", "--json"},
	} {
		var c capture
		env := c.env() // no Convert hook: the launcher
		handled, code := Front(args, env)
		if !handled || code != ExitDone || c.err.Len() != 0 || c.out.Len() == 0 || c.opened != 1 {
			t.Errorf("%q: handled=%v exit=%d opened=%d stdout=%q stderr=%q", args, handled, code, c.opened, c.out.String(), c.err.String())
		}
	}
}

func TestManifestNamesTheInspectionGroup(t *testing.T) {
	m, _ := Load()
	for _, want := range []struct {
		name, risk, alias string
		timeout           int
	}{
		{"doctor", "safe", "", 10000}, {"input check", "reads-user-data", "", 10000}, {"ocr langs", "safe", "-ocr-langs", 5000},
	} {
		c := m.Find(strings.Fields(want.name)...)
		if c == nil {
			t.Errorf("%s is not in the manifest", want.name)
			continue
		}
		if c.Risk != want.risk || c.Forward != "never" || c.DefaultFor != "" || c.Since != "1.0" || c.TimeoutMs == nil || *c.TimeoutMs != want.timeout {
			t.Errorf("%s: risk %q forward %q since %q timeout %v", want.name, c.Risk, c.Forward, c.Since, c.TimeoutMs)
		}
		if want.alias != "" {
			found := false
			for _, a := range c.Aliases {
				if a.Spelling == want.alias {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("%s: aliases %v miss %s", want.name, c.Aliases, want.alias)
			}
		}
		hasExit := map[int]bool{}
		for _, e := range c.ExitCodes {
			hasExit[e.Code] = true
			if e.Code == ExitNotFound && !strings.Contains(e.Meaning, ReasonTargetMissing) {
				t.Errorf("%s: exit 11 does not name the reason %s", want.name, ReasonTargetMissing)
			}
		}
		if !hasExit[0] || !hasExit[1] || !hasExit[10] || (want.name == "input check") != hasExit[11] {
			t.Errorf("%s: exit codes %v", want.name, hasExit)
		}
	}
}
