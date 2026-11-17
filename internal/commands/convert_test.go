package commands

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"doc-html-translate/internal/config"
	"doc-html-translate/internal/fdsec"
	"doc-html-translate/internal/i18n"
	"doc-html-translate/internal/outputpath"
)

// document makes a small input file in a fresh folder and returns its path.
func document(t *testing.T, name string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, []byte("book"), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// runner stands in for the converter: it records that it ran and with what, and leaves the output a finished run leaves
// (an entry page and a completion record) - or nothing, for a run that failed.
type runner struct {
	ran   int
	cfg   config.Config
	exit  int
	state string // the translation state to record; "" leaves no record
	reuse bool   // the pipeline's R4: a finished, current output is opened as it is and nothing is written
}

func (r *runner) run(cfg config.Config) int {
	r.ran++
	r.cfg = cfg
	if r.reuse || r.state == "" {
		return r.exit
	}
	abs, _ := filepath.Abs(cfg.InputFile)
	target, err := outputpath.Resolve(abs, cfg.OutputFolder)
	if err != nil {
		panic(err)
	}
	if err := os.MkdirAll(target.Dir, 0o755); err != nil {
		panic(err)
	}
	if err := os.WriteFile(filepath.Join(target.Dir, "index.html"), []byte("<html></html>"), 0o644); err != nil {
		panic(err)
	}
	c := outputpath.Completion{ToolVersion: testVersion, Options: outputpath.OptionsFor(cfg), Translation: r.state}
	if err := outputpath.MarkComplete(target.Dir, abs, c); err != nil {
		panic(err)
	}
	return r.exit
}

func (c *capture) convertEnv(r *runner) Env {
	e := c.env()
	e.Convert = r.run
	return e
}

// ---- the grammar's positional word and required options ----

func TestGrammarPositionalWord(t *testing.T) {
	m, _ := Load()
	cmd := m.Find("convert")
	word := func(args ...string) (string, *UsageError) {
		inv, err := Parse(cmd, args)
		if err != nil {
			return "", err
		}
		return inv.String("path")
	}
	for _, c := range []struct {
		args []string
		want string
	}{
		{[]string{"book.epub"}, "book.epub"},
		{[]string{"book.epub", "--noopen"}, "book.epub"},
		{[]string{"--noopen", "book.epub"}, "book.epub"},
		{[]string{"--folder", "out", "book.epub", "--dst=de"}, "book.epub"},
		{[]string{"--noopen", "--", "-weird.epub"}, "-weird.epub"},
		{[]string{"--", "C:\\My Books\\a b.epub"}, "C:\\My Books\\a b.epub"},
		{[]string{"BOOK.EPUB", "--NOOPEN"}, "BOOK.EPUB"}, // options without case, values never
	} {
		if got, err := word(c.args...); err != nil || got != c.want {
			t.Errorf("%q: path %q, error %v; want %q", c.args, got, err, c.want)
		}
	}
	for _, c := range []struct {
		name string
		args []string
		want string
	}{
		{"a second word", []string{"a.epub", "b.epub"}, `unexpected value "b.epub" for "convert"`},
		{"two words after --", []string{"--", "a.epub", "b.epub"}, `unexpected value "b.epub" for "convert"`},
		{"the word is not an option", []string{"--path", "a.epub"}, `unknown option "--path" for "convert"`},
		{"no word", []string{"--noopen"}, `missing required <path> for "convert"`},
		{"nothing at all", nil, `missing required <path> for "convert"`},
		{"an empty word", []string{""}, `unexpected value "" for "<path>"`},
		{"a legacy flag spelling", []string{"-noopen", "a.epub"}, `unknown option "-noopen" for "convert"`},
	} {
		if _, err := word(c.args...); err == nil || err.Reason != ReasonBadArgument || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: got %v, want a refusal containing %q", c.name, err, c.want)
		}
	}
}

func TestGrammarRequiredOptions(t *testing.T) {
	m, _ := Load()
	cmd := m.Find("output status")
	if _, err := Parse(cmd, []string{"--json"}); err == nil || err.Error() != `missing required --in for "output status"` {
		t.Errorf("a missing --in: %v", err)
	}
	if inv, err := Parse(cmd, []string{"--in", "a.epub"}); err != nil {
		t.Errorf("--in given: %v", err)
	} else if v, _ := inv.String("in"); v != "a.epub" {
		t.Errorf("--in = %q", v)
	}
	// A required option that is a bool or whose value is empty is still a refusal, not a silent default.
	if _, err := Parse(cmd, []string{"--in="}); err == nil {
		t.Error("an empty --in was accepted")
	}

	var c capture
	r := &runner{}
	handled, code := Front([]string{"output", "status"}, c.convertEnv(r))
	if !handled || code != ExitUsage || !strings.Contains(c.err.String(), `missing required --in for "output status"`) || r.ran != 0 {
		t.Errorf("handled=%v exit=%d stderr=%q ran=%d", handled, code, c.err.String(), r.ran)
	}
}

// ---- convert is the bare call ----

// The verb and the bare call run the same code with the same defaults: the argument list the verb hands to the
// conversion parser parses to the very configuration the bare call's flags do, for every option the manifest lists.
func TestConvertBuildsTheLegacyArguments(t *testing.T) {
	m, _ := Load()
	cmd := m.Find("convert")
	verb := []string{
		"book.epub", "--noopen", "--folder", "out dir", "--dst=de", "--ollama", "--ollama-model", "m:1", "--max-cost", "2.5",
		"--ocr", "--ocr-lang", "eng+rus", "--multipage", "--split", "100", "--toc-depth", "2", "--force", "--v",
		"--ui-lang", "ru", "--ollama-parallel", "3", "--ollama-ctx", "1024", "--src", "fr", "--notranslate", "--google",
		"--fdsec-password-env", "PW",
	}
	bare := []string{
		"-noopen", "-folder", "out dir", "-dst", "de", "-free", "-ollama-model", "m:1", "-max-cost", "2.5",
		"-ocr", "-ocr-lang", "eng+rus", "-multipage", "-split", "100", "-toc-depth", "2", "-force", "-v",
		"-ui-lang", "ru", "-ollama-parallel", "3", "-ollama-ctx", "1024", "-src", "fr", "-notranslate", "-google",
		"-fdsec-password-env", "PW", "book.epub",
	}
	same := func(verb, bare []string) {
		t.Helper()
		inv, uerr := Parse(cmd, verb)
		if uerr != nil {
			t.Fatalf("%q: %v", verb, uerr)
		}
		path, _ := inv.String("path")
		args, uerr := LegacyArgs(inv, path, "result")
		if uerr != nil {
			t.Fatal(uerr)
		}
		got, err := config.ParseArgs(args)
		if err != nil {
			t.Fatalf("the legacy parser refused %q: %v", args, err)
		}
		want, err := config.ParseArgs(bare)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("the verb and the bare call differ:\nverb %+v\nbare %+v", got, want)
		}
	}
	same(verb, bare)
	same([]string{"book.epub"}, []string{"book.epub"})                                        // the same defaults, the browser stays on
	same([]string{"--noopen", "--", "-weird.epub"}, []string{"-noopen", "--", "-weird.epub"}) // a path that looks like a flag
	if _, uerr := Parse(cmd, []string{"book.epub", "--free"}); uerr == nil || uerr.Reason != ReasonBadArgument {
		t.Errorf("convert --free: uerr = %v, want ReasonBadArgument", uerr)
	}
	if c, _ := config.ParseArgs([]string{"book.epub"}); c.NoOpen {
		t.Error("the default of the bare call is no longer to open the browser")
	}
	// A bool given false is the default and adds nothing.
	inv, _ := Parse(cmd, []string{"a.epub", "--noopen=false"})
	if args, _ := LegacyArgs(inv, "a.epub"); !reflect.DeepEqual(args, []string{"--", "a.epub"}) {
		t.Errorf("--noopen=false produced %q", args)
	}
}

// Every option the manifest lists for the two verbs is a flag the conversion parser takes: a verb option the parser
// does not know would be a refusal in a run, found here instead.
func TestEveryOptionOfTheConvertGroupReachesTheLegacyParser(t *testing.T) {
	m, _ := Load()
	for _, verb := range []string{"convert", "output status"} {
		cmd := m.Find(verb)
		for _, a := range cmd.Args {
			if a.Positional || a.Name == "result" || a.Name == "json" || a.Name == "in" {
				continue
			}
			var toks []string
			switch a.Kind {
			case "bool":
				toks = []string{"--" + a.Name}
			case "int":
				toks = []string{"--" + a.Name, "7"}
			case "enum":
				toks = []string{"--" + a.Name, a.Values[0]}
			default:
				val := "abc"
				if a.Name == "max-cost" {
					val = "2"
				}
				toks = []string{"--" + a.Name, val}
			}
			head := []string{"doc.epub"}
			if verb == "output status" {
				head = []string{"--in", "doc.epub"}
			}
			inv, uerr := Parse(cmd, append(head, toks...))
			if uerr != nil {
				t.Errorf("%s --%s: %v", verb, a.Name, uerr)
				continue
			}
			args, uerr := LegacyArgs(inv, "doc.epub", "in", "json", "result")
			if uerr != nil {
				t.Errorf("%s --%s: %v", verb, a.Name, uerr)
				continue
			}
			if _, err := config.ParseArgs(args); err != nil {
				t.Errorf("%s --%s: the conversion parser refused %q: %v", verb, a.Name, args, err)
			}
			if !strings.Contains(strings.Join(args, " "), "--"+a.Name) {
				t.Errorf("%s --%s was dropped on the way to the parser: %q", verb, a.Name, args)
			}
		}
	}
}

func TestConvertReturnsTheRunsExitCode(t *testing.T) {
	for _, exit := range []int{0, 1, 2, 3, 4, 130} {
		var c capture
		r := &runner{exit: exit}
		handled, code := Front([]string{"convert", "book.epub", "--noopen"}, c.convertEnv(r))
		if !handled || code != exit || r.ran != 1 {
			t.Errorf("run exit %d: handled=%v exit=%d ran=%d", exit, handled, code, r.ran)
		}
		if !r.cfg.NoOpen || r.cfg.InputFile != "book.epub" {
			t.Errorf("the run got %+v", r.cfg)
		}
		if c.out.Len() != 0 || c.err.Len() != 0 {
			t.Errorf("the front itself printed: %q %q", c.out.String(), c.err.String())
		}
	}
	// A value the grammar accepts and the converter's parser refuses is "bad arguments", exit 1, and runs nothing.
	var c capture
	r := &runner{}
	handled, code := Front([]string{"convert", "book.epub", "--split", "-5"}, c.convertEnv(r))
	if !handled || code != ExitFailed || r.ran != 0 || !strings.Contains(c.err.String(), "-split") {
		t.Errorf("handled=%v exit=%d ran=%d stderr=%q", handled, code, r.ran, c.err.String())
	}
}

// ---- rule 4: a command never waits on a person ----

func TestConvertRefusesWhatWouldWaitOnAPerson(t *testing.T) {
	refused := []struct {
		name string
		args []string
		want string
	}{
		{"google with no cap", []string{"convert", "a.epub", "--google", "--noopen"}, `"--google" needs "--max-cost" with a positive amount in USD`},
		{"google with a zero cap", []string{"convert", "a.epub", "--google", "--max-cost", "0"}, `"--google" needs "--max-cost"`},
		{"google with a zero cap, spelled 0.0", []string{"convert", "a.epub", "--google", "--max-cost=0.0"}, `"--google" needs "--max-cost"`},
		{"a negative cap", []string{"convert", "a.epub", "--google", "--max-cost", "-1"}, `unexpected value "-1" for "--max-cost"`},
		{"a negative cap without google", []string{"convert", "a.epub", "--max-cost=-2"}, `unexpected value "-2" for "--max-cost"`},
		{"a cap that is NaN", []string{"convert", "a.epub", "--google", "--max-cost", "NaN"}, `unexpected value "NaN" for "--max-cost"`},
		{"a cap that is infinite", []string{"convert", "a.epub", "--google", "--max-cost=Inf"}, `unexpected value "Inf" for "--max-cost"`},
		{"a cap that is not a number", []string{"convert", "a.epub", "--max-cost", "two"}, `unexpected value "two" for "--max-cost"`},
		{"a secret file with no variable", []string{"convert", "a.fd-sec", "--noopen"}, `a FileDO secret file needs "--fdsec-password-env"`},
		{"a secret file in capitals", []string{"convert", "C:\\x\\A.FD-SEC"}, `a FileDO secret file needs "--fdsec-password-env"`},
		{"a secret file with an empty variable name", []string{"convert", "a.fd-sec", "--fdsec-password-env", ""}, `unexpected value "" for "--fdsec-password-env"`},
	}
	for _, c := range refused {
		t.Run(c.name, func(t *testing.T) {
			var got capture
			r := &runner{state: outputpath.TranslationNone}
			dir := t.TempDir()
			t.Chdir(dir)
			handled, code := Front(c.args, got.convertEnv(r))
			if !handled || code != ExitUsage {
				t.Fatalf("handled=%v exit=%d, want true and 10", handled, code)
			}
			if r.ran != 0 || got.out.Len() != 0 {
				t.Errorf("a refusal ran the converter (%d) or printed %q", r.ran, got.out.String())
			}
			if !strings.Contains(got.err.String(), c.want) {
				t.Errorf("standard error %q does not say %q", got.err.String(), c.want)
			}
			if entries, _ := os.ReadDir(dir); len(entries) != 0 {
				t.Errorf("a refusal wrote files: %v", entries)
			}
		})
	}

	// The refusal speaks the language the run asked for with --ui-lang.
	var ru capture
	if handled, code := Front([]string{"convert", "a.epub", "--google", "--ui-lang", "ru"}, ru.convertEnv(&runner{})); !handled || code != ExitUsage ||
		!strings.Contains(ru.err.String(), `"--google" требует "--max-cost"`) {
		t.Errorf("handled=%v exit=%d stderr=%q", handled, code, ru.err.String())
	}

	// What is not refused runs, with the cap and the variable name reaching the converter.
	for _, c := range []struct {
		name string
		args []string
		cost float64
		env  string
	}{
		{"google with a cap", []string{"convert", "a.epub", "--google", "--max-cost", "2"}, 2, ""},
		{"google with a cap, spelled 1e0", []string{"convert", "a.epub", "--google", "--max-cost=1e0"}, 1, ""},
		{"a zero cap without google", []string{"convert", "a.epub", "--max-cost", "0"}, 0, ""},
		{"a secret file with its variable", []string{"convert", "a.fd-sec", "--fdsec-password-env", "DOCHT_PW"}, 0, "DOCHT_PW"},
		{"an ordinary document", []string{"convert", "a.epub"}, 0, ""},
	} {
		var got capture
		r := &runner{}
		handled, code := Front(c.args, got.convertEnv(r))
		if !handled || code != 0 || r.ran != 1 || r.cfg.MaxCost != c.cost || r.cfg.FdsecPasswordEnv != c.env {
			t.Errorf("%s: handled=%v exit=%d ran=%d cfg=%+v stderr=%q", c.name, handled, code, r.ran, r.cfg, got.err.String())
		}
	}

	// The bare call keeps its dialog and its prompt: the front does not read it, so the parser's own rules stand.
	for _, args := range [][]string{{"a.epub", "-google"}, {"-google", "a.epub"}, {"a.fd-sec"}} {
		var got capture
		if handled, _ := Front(args, got.convertEnv(&runner{})); handled {
			t.Errorf("the bare call %q was taken by the front", args)
		}
	}
}

// The extension is the one FileDO's package owns; the front compares against a copy so it need not import it.
func TestSecretExtensionIsFdsecs(t *testing.T) {
	if secretExt != fdsec.Ext {
		t.Fatalf("secretExt %q, fdsec.Ext %q", secretExt, fdsec.Ext)
	}
}

func TestUILangValuesAreTheCatalogCodes(t *testing.T) {
	m, _ := Load()
	for _, verb := range []string{"convert"} {
		for _, a := range m.Find(verb).Args {
			if a.Name == "ui-lang" && !reflect.DeepEqual(a.Values, i18n.Codes) {
				t.Errorf("%s --ui-lang lists %v, the catalog has %v", verb, a.Values, i18n.Codes)
			}
		}
	}
}

// ---- the launcher does not serve the group ----

func TestConvertGroupIsNotServedByTheLauncher(t *testing.T) {
	for _, args := range [][]string{
		{"convert", "a.epub"}, {"Convert", "--nope"}, {"output", "status", "--in", "a.epub"}, {"output", "status"},
	} {
		var c capture
		env := c.env() // no Convert hook: the launcher
		handled, code := Front(args, env)
		verb := "convert"
		if strings.EqualFold(args[0], "output") {
			verb = "output status"
		}
		if !handled || code != ExitUsage || c.out.Len() != 0 ||
			!strings.Contains(c.err.String(), `"`+verb+`" is run by doc-html-translate.exe, not by this program`) {
			t.Errorf("%q: handled=%v exit=%d stdout=%q stderr=%q", args, handled, code, c.out.String(), c.err.String())
		}
	}
	// The launcher still answers the two base verbs, and its list names the group.
	var c capture
	env := c.env()
	env.Usage = nil
	if handled, code := Front([]string{"commands"}, env); !handled || code != 0 ||
		!strings.Contains(c.out.String(), "convert") || !strings.Contains(c.out.String(), "output status") {
		t.Errorf("commands: handled=%v exit=%d\n%s", handled, code, c.out.String())
	}
}

func TestManifestNamesTheDefaultVerbOfTheConsoleExecutable(t *testing.T) {
	m, _ := Load()
	c := m.Find("convert")
	if c == nil || c.DefaultFor != m.Exe || c.Forward != "never" || c.Risk != "leaves-machine" {
		t.Fatalf("convert: %+v", c)
	}
	if c.TimeoutMs == nil || *c.TimeoutMs != 0 {
		t.Error("convert declares a time bound of its own: it never forwards, so it has none (proposal item 1)")
	}
	if s := m.Find("output", "status"); s == nil || s.Risk != "reads-user-data" || s.DefaultFor != "" {
		t.Errorf("output status: %+v", s)
	}
	var u capture
	Front([]string{"commands"}, u.env())
	if !strings.Contains(u.out.String(), "the call with no verb word (a path and flags) is this command") {
		t.Error("the readable list does not say which command the bare call is")
	}
}

// ---- --result (rule 9, C11) ----

func readResult(t *testing.T, path string) (ConvertResult, map[string]json.RawMessage) {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var res ConvertResult
	if err := json.Unmarshal(b, &res); err != nil {
		t.Fatalf("the result file is not JSON: %v\n%s", err, b)
	}
	var members map[string]json.RawMessage
	_ = json.Unmarshal(b, &members)
	return res, members
}

func TestConvertResultFile(t *testing.T) {
	type scenario struct {
		name       string
		extra      []string
		state      string
		exit       int
		outcome    string
		translated string
	}
	for _, s := range []scenario{
		{"converted", nil, outputpath.TranslationNone, 0, OutcomeConverted, "none"},
		{"translated", []string{"--ollama"}, outputpath.TranslationFull, 0, OutcomeTranslated, "full"},
		{"translated, labels not (exit 4)", []string{"--ollama"}, outputpath.TranslationFull, 4, OutcomeTranslated, "full"},
		{"partial", []string{"--ollama"}, outputpath.TranslationPartial, 4, OutcomePartial, "partial"},
		{"skipped: the engine never ran", []string{"--ollama"}, outputpath.TranslationNone, 0, OutcomeSkipped, "none"},
		{"skipped: the key was unavailable (exit 4)", []string{"--google", "--max-cost", "1"}, outputpath.TranslationNone, 4, OutcomeSkipped, "none"},
	} {
		t.Run(s.name, func(t *testing.T) {
			src := document(t, "book.epub")
			work := t.TempDir()
			out := filepath.Join(work, "out")
			result := filepath.Join(work, "result.json")
			var c capture
			r := &runner{state: s.state, exit: s.exit}
			args := append([]string{"convert", src, "--noopen", "--folder", out, "--result", result}, s.extra...)
			handled, code := Front(args, c.convertEnv(r))
			if !handled || code != s.exit || r.ran != 1 {
				t.Fatalf("handled=%v exit=%d ran=%d stderr=%q", handled, code, r.ran, c.err.String())
			}
			res, members := readResult(t, result)
			for _, key := range []string{"outcome", "outputDir", "entryFile", "translation", "complete", "exitCode"} {
				if _, ok := members[key]; !ok {
					t.Errorf("the result lacks %q: %v", key, members)
				}
			}
			if len(members) != 6 {
				t.Errorf("the result has %d members, the manifest names 6: %v", len(members), members)
			}
			dir := filepath.Join(out, "book")
			if res.Outcome != s.outcome || res.Translation != s.translated || res.ExitCode != s.exit {
				t.Errorf("outcome %q translation %q exit %d; want %q %q %d", res.Outcome, res.Translation, res.ExitCode, s.outcome, s.translated, s.exit)
			}
			if !samePathForTest(res.OutputDir, dir) || !samePathForTest(res.EntryFile, filepath.Join(dir, "index.html")) {
				t.Errorf("the result names %q and %q, want the child %q of the folder", res.OutputDir, res.EntryFile, dir)
			}
			if res.Complete == nil || res.Complete.Translation != s.translated || res.Complete.ToolVersion != testVersion || res.Complete.Completed.IsZero() {
				t.Errorf("complete is not the record's: %+v", res.Complete)
			}
			// The record the result copies is the one on disk.
			m, err := outputpath.ReadMarker(dir)
			if err != nil || m.Complete == nil || !m.Complete.Completed.Equal(res.Complete.Completed) {
				t.Errorf("the result's complete differs from %s: %v", outputpath.MarkerName, err)
			}
			// Whole or not at all: only the result and the output are there, no temporary file.
			entries, _ := os.ReadDir(work)
			if len(entries) != 2 {
				t.Errorf("the working folder holds %d entries, want out and result.json: %v", len(entries), entries)
			}
		})
	}

	t.Run("reused", func(t *testing.T) {
		src := document(t, "book.epub")
		work := t.TempDir()
		out := filepath.Join(work, "out")
		r := &runner{state: outputpath.TranslationNone}
		run := func(result string) ConvertResult {
			var c capture
			args := []string{"convert", src, "--noopen", "--folder", out, "--result", result}
			if handled, code := Front(args, c.convertEnv(r)); !handled || code != 0 {
				t.Fatalf("handled=%v exit=%d stderr=%q", handled, code, c.err.String())
			}
			res, _ := readResult(t, result)
			return res
		}
		first := run(filepath.Join(work, "first.json"))
		if first.Outcome != OutcomeConverted {
			t.Fatalf("the first run is %q", first.Outcome)
		}
		r.reuse = true // the second run opens the finished output as it is and writes nothing
		second := run(filepath.Join(work, "second.json"))
		if second.Outcome != OutcomeReused || second.Complete == nil || !second.Complete.Completed.Equal(first.Complete.Completed) {
			t.Errorf("the second run is %q, want reused with the first run's record: %+v", second.Outcome, second)
		}
		// A rebuild (--force, a changed option) writes a new record and is not "reused".
		r.reuse = false
		var c capture
		result := filepath.Join(work, "third.json")
		Front([]string{"convert", src, "--noopen", "--folder", out, "--force", "--result", result}, c.convertEnv(r))
		if third, _ := readResult(t, result); third.Outcome != OutcomeConverted || third.Complete.Completed.Equal(first.Complete.Completed) {
			t.Errorf("a rebuild reads as %q", third.Outcome)
		}
	})

	t.Run("a run that failed writes no file", func(t *testing.T) {
		for _, exit := range []int{1, 2, 3, 130} {
			src := document(t, "book.epub")
			work := t.TempDir()
			result := filepath.Join(work, "result.json")
			var c capture
			r := &runner{exit: exit}
			handled, code := Front([]string{"convert", src, "--noopen", "--folder", filepath.Join(work, "out"), "--result", result}, c.convertEnv(r))
			if !handled || code != exit {
				t.Errorf("exit %d: handled=%v exit=%d", exit, handled, code)
			}
			if entries, _ := os.ReadDir(work); len(entries) != 0 {
				t.Errorf("exit %d left %v", exit, entries)
			}
		}
		// Even a record that happens to exist (an earlier run's) is not turned into the answer of a run that failed.
		src := document(t, "book.epub")
		work := t.TempDir()
		out := filepath.Join(work, "out")
		var c capture
		Front([]string{"convert", src, "--noopen", "--folder", out}, c.convertEnv(&runner{state: outputpath.TranslationNone}))
		result := filepath.Join(work, "result.json")
		Front([]string{"convert", src, "--noopen", "--folder", out, "--result", result}, c.convertEnv(&runner{exit: 3}))
		if _, err := os.Stat(result); err == nil {
			t.Error("a run that ended 3 wrote a result from the earlier run's record")
		}
	})

	t.Run("a run that finished but left no record is not done", func(t *testing.T) {
		src := document(t, "book.epub")
		work := t.TempDir()
		result := filepath.Join(work, "result.json")
		var c capture
		r := &runner{exit: 0} // no state: the completion record was never written
		handled, code := Front([]string{"convert", src, "--noopen", "--folder", filepath.Join(work, "out"), "--result", result}, c.convertEnv(r))
		if !handled || code != ExitIO || !strings.Contains(c.err.String(), "the result file could not be written") {
			t.Errorf("handled=%v exit=%d stderr=%q", handled, code, c.err.String())
		}
		if _, err := os.Stat(result); err == nil {
			t.Error("a result file appeared with no record behind it")
		}
	})

	t.Run("a target that cannot be written is refused before the run", func(t *testing.T) {
		src := document(t, "book.epub")
		work := t.TempDir()
		out := filepath.Join(work, "out")
		for _, result := range []string{filepath.Join(work, "no-such-folder", "result.json"), work} {
			var c capture
			r := &runner{state: outputpath.TranslationNone}
			handled, code := Front([]string{"convert", src, "--noopen", "--folder", out, "--result", result}, c.convertEnv(r))
			if !handled || code != ExitFailed || r.ran != 0 || !strings.Contains(c.err.String(), "the result file could not be written") {
				t.Errorf("%s: handled=%v exit=%d ran=%d stderr=%q", result, handled, code, r.ran, c.err.String())
			}
			if _, err := os.Stat(out); err == nil {
				t.Error("the conversion started although the result could not land")
			}
		}
		if entries, _ := os.ReadDir(work); len(entries) != 0 {
			t.Errorf("the probe left %v", entries)
		}
	})

	t.Run("two --result options are refused", func(t *testing.T) {
		var c capture
		r := &runner{}
		handled, code := Front([]string{"convert", "a.epub", "--result", "a", "--result", "b"}, c.convertEnv(r))
		if !handled || code != ExitUsage || r.ran != 0 || !strings.Contains(c.err.String(), `option "--result" was given more than once`) {
			t.Errorf("handled=%v exit=%d ran=%d stderr=%q", handled, code, r.ran, c.err.String())
		}
	})
}

func samePathForTest(a, b string) bool {
	return strings.EqualFold(filepath.Clean(a), filepath.Clean(b))
}

// ---- output status ----

func TestOutputStatus(t *testing.T) {
	src := document(t, "book.epub")
	work := t.TempDir()
	out := filepath.Join(work, "out")
	status := func(extra ...string) (outputpath.Status, int, string) {
		t.Helper()
		var c capture
		args := append([]string{"output", "status", "--in", src, "--folder", out, "--json"}, extra...)
		handled, code := Front(args, c.convertEnv(&runner{}))
		if !handled {
			t.Fatal("not handled")
		}
		var st outputpath.Status
		if code == 0 {
			if err := json.Unmarshal(c.out.Bytes(), &st); err != nil {
				t.Fatalf("not the data object: %v\n%s", err, c.out.String())
			}
		}
		return st, code, c.err.String()
	}

	// A fresh input: nothing exists.
	if st, code, _ := status(); code != 0 || st != (outputpath.Status{}) {
		t.Fatalf("fresh input: exit %d, %+v", code, st)
	}
	var fresh capture
	Front([]string{"output", "status", "--in", src, "--folder", out, "--json"}, fresh.convertEnv(&runner{}))
	if fresh.out.String() != `{"exists":false,"complete":false,"stale":false}`+"\n" {
		t.Errorf("the data of a fresh input is %q", fresh.out.String())
	}

	// After a run with an engine: exists, complete, current.
	var c capture
	run := &runner{state: outputpath.TranslationFull}
	if _, code := Front([]string{"convert", src, "--noopen", "--folder", out, "--ollama", "--dst", "ru"}, c.convertEnv(run)); code != 0 {
		t.Fatalf("the run: %d", code)
	}
	st, code, _ := status("--ollama", "--dst", "ru")
	if code != 0 || !st.Exists || !st.Complete || st.Stale || st.Reason != "" || st.Translation != "full" || st.ToolVersion != testVersion {
		t.Fatalf("after a run: exit %d, %+v", code, st)
	}
	if !samePathForTest(st.OutputDir, filepath.Join(out, "book")) || !samePathForTest(st.EntryFile, filepath.Join(out, "book", "index.html")) {
		t.Errorf("paths: %+v", st)
	}

	// The same input asked for with another target language (and an engine): a run would rebuild it.
	st, code, _ = status("--ollama", "--dst", "de")
	if code != 0 || !st.Exists || !st.Complete || !st.Stale || st.Reason != "options-changed" {
		t.Errorf("another --dst with an engine: exit %d, %+v", code, st)
	}
	// An engine turned off is a change too: the record was built with Ollama.
	if st, _, _ = status("--notranslate", "--dst", "ru"); !st.Stale || st.Reason != "options-changed" {
		t.Errorf("an engine turned off: %+v", st)
	}
	// A result-affecting option the record lacks.
	if st, _, _ = status("--ollama", "--dst", "ru", "--ocr"); !st.Stale || st.Reason != "options-changed" {
		t.Errorf("--ocr: %+v", st)
	}
	// An execution-only switch is no change; the parser's defaults apply as in a run.
	if st, _, _ = status("--ollama", "--dst", "ru", "--v"); st.Stale {
		t.Errorf("--v: %+v", st)
	}
}

func TestOutputStatusWithoutAnEngineIgnoresTheTargetLanguage(t *testing.T) {
	src := document(t, "book.epub")
	out := filepath.Join(t.TempDir(), "out")
	var c capture
	Front([]string{"convert", src, "--noopen", "--folder", out}, c.convertEnv(&runner{state: outputpath.TranslationNone}))
	var s capture
	Front([]string{"output", "status", "--in", src, "--folder", out, "--dst", "de", "--json"}, s.convertEnv(&runner{}))
	var st outputpath.Status
	if err := json.Unmarshal(s.out.Bytes(), &st); err != nil || !st.Exists || !st.Complete || st.Stale {
		t.Errorf("another --dst on a plain conversion: %+v %v", st, err)
	}
}

func TestOutputStatusReadableAndResult(t *testing.T) {
	src := document(t, "book.epub")
	work := t.TempDir()
	out := filepath.Join(work, "out")
	var c capture
	Front([]string{"convert", src, "--noopen", "--folder", out}, c.convertEnv(&runner{state: outputpath.TranslationNone}))

	var s capture
	result := filepath.Join(work, "status.json")
	handled, code := Front([]string{"output", "status", "--in", src, "--folder", out, "--result", result}, s.convertEnv(&runner{}))
	if !handled || code != 0 {
		t.Fatalf("handled=%v exit=%d stderr=%q", handled, code, s.err.String())
	}
	for _, want := range []string{"exists: true\n", "complete: true\n", "stale: false\n", "translation: none\n", "toolVersion: " + testVersion + "\n"} {
		if !strings.Contains(s.out.String(), want) {
			t.Errorf("the readable form lacks %q:\n%s", want, s.out.String())
		}
	}
	if strings.Contains(s.out.String(), "reason:") {
		t.Errorf("a current result has a reason:\n%s", s.out.String())
	}
	b, err := os.ReadFile(result)
	var st outputpath.Status
	if err != nil || json.Unmarshal(b, &st) != nil || !st.Exists || !st.Complete {
		t.Errorf("the result file: %q %v", b, err)
	}

	// An unwritable result file: exit 1 and nothing on standard output.
	var f capture
	handled, code = Front([]string{"output", "status", "--in", src, "--result", filepath.Join(work, "no-folder", "x.json")}, f.convertEnv(&runner{}))
	if !handled || code != ExitFailed || f.out.Len() != 0 {
		t.Errorf("handled=%v exit=%d stdout=%q", handled, code, f.out.String())
	}
}

func TestOutputStatusRefusals(t *testing.T) {
	src := document(t, "book.epub")
	missing := filepath.Join(t.TempDir(), "nothing.epub")
	for _, c := range []struct {
		name string
		args []string
		exit int
		want string
	}{
		{"an input that does not exist", []string{"output", "status", "--in", missing}, ExitNotFound, "the document does not exist"},
		{"no --in", []string{"output", "status", "--json"}, ExitUsage, `missing required --in for "output status"`},
		{"an unknown option", []string{"output", "status", "--in", src, "--nope"}, ExitUsage, `unknown option "--nope" for "output status"`},
		{"a word", []string{"output", "status", src}, ExitUsage, `unexpected value`},
		{"a value the converter refuses", []string{"output", "status", "--in", src, "--split", "-5"}, ExitUsage, "-split"},
		{"an option of convert only", []string{"output", "status", "--in", src, "--noopen"}, ExitUsage, `unknown option "--noopen" for "output status"`},
	} {
		var got capture
		handled, code := Front(c.args, got.convertEnv(&runner{}))
		if !handled || code != c.exit || got.out.Len() != 0 || !strings.Contains(got.err.String(), c.want) {
			t.Errorf("%s: handled=%v exit=%d stdout=%q stderr=%q", c.name, handled, code, got.out.String(), got.err.String())
		}
	}
	// The refusal is in the interface language, like every other.
	var ru capture
	env := ru.convertEnv(&runner{})
	env.Lang = "ru"
	Front([]string{"output", "status", "--in", missing}, env)
	if !strings.Contains(ru.err.String(), "документ не существует") {
		t.Errorf("stderr %q", ru.err.String())
	}
}

// A folder that holds an index.html and is not the converter's is not a result, whatever the options.
func TestOutputStatusIgnoresAForeignFolder(t *testing.T) {
	src := document(t, "book.epub")
	foreign := filepath.Join(filepath.Dir(src), "book")
	if err := os.MkdirAll(foreign, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(foreign, "index.html"), []byte("my site"), 0o644); err != nil {
		t.Fatal(err)
	}
	var c capture
	Front([]string{"output", "status", "--in", src, "--json"}, c.convertEnv(&runner{}))
	if c.out.String() != `{"exists":false,"complete":false,"stale":false}`+"\n" {
		t.Errorf("a foreign index.html was reported: %q", c.out.String())
	}
}
