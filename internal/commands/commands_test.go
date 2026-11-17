package commands

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"doc-html-translate/internal/config"
	"doc-html-translate/internal/i18n"
)

const testVersion = "26.1114.0001"

// capture is the front's output channels in a test, and a count of how often they were opened: a front that does not
// answer must not open (attach to) a console at all.
type capture struct {
	out, err bytes.Buffer
	opened   int
}

func (c *capture) env() Env {
	return Env{
		Version: testVersion,
		Lang:    "en",
		Usage:   config.UsageText,
		Open: func() (io.Writer, io.Writer) {
			c.opened++
			return &c.out, &c.err
		},
	}
}

func repoFile(t *testing.T, parts ...string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(append([]string{"..", ".."}, parts...)...))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// stdoutOf runs fn with the process's standard output redirected and returns what it wrote.
func stdoutOf(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = old }()
	fn()
	_ = w.Close()
	b, _ := io.ReadAll(r)
	return string(b)
}

func golden(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "help.golden"))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// ---- the manifest ----

// C1: go:embed cannot reach docs/, so the executable's copy lives beside the code and must be the same bytes.
func TestEmbeddedManifestEqualsDocsCopy(t *testing.T) {
	if !bytes.Equal(Embedded(), repoFile(t, "docs", "cli-commands.json")) {
		t.Fatal("internal/commands/cli-commands.json differs from docs/cli-commands.json: run `go run ./tools/clidoc -render`")
	}
}

func TestManifestIsValid(t *testing.T) {
	if problems := Check(repoFile(t, "docs", "cli-commands.json")); len(problems) > 0 {
		t.Fatalf("docs/cli-commands.json breaks APP-COMMANDS rule 18:\n%s", strings.Join(problems, "\n"))
	}
	m, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if m.Product != Product || m.Find("commands") == nil || m.Find("version") == nil {
		t.Fatalf("manifest product %q, base commands %v %v", m.Product, m.Find("commands"), m.Find("version"))
	}
	// D1, D2, D5 (APP-COMMANDS 0.2 items A.4, E.1-E.4, F.1): every shipped alias is legacy:true, and those whose
	// behavior or interactive exit code does not match the verb carry answers:false.
	wantAnswersFalse := map[string]bool{"": true, "-ocr-download": true, "-register-openwith": true, "-register": true}
	var seenEmpty bool
	for _, c := range m.Commands {
		for _, a := range c.Aliases {
			if !a.Legacy {
				t.Errorf("%s: alias %q lacks legacy:true", c.Name, a.Spelling)
			}
			if wantAnswersFalse[a.Spelling] && (a.Answers == nil || *a.Answers) {
				t.Errorf("%s: alias %q lacks answers:false", c.Name, a.Spelling)
			}
			if c.Name == "integration add" && a.Spelling == "" {
				seenEmpty = true
			}
		}
	}
	if !seenEmpty {
		t.Error(`"integration add" aliases lack "" (no-argument first-run prompt, APP-COMMANDS 0.2 item A.4)`)
	}
}

// Check is the gate's judge, so it has to say no: a misspelt member a lenient reader would drop, a value outside a
// closed list and a missing required member each produce a finding.
func TestCheckCatchesBadManifests(t *testing.T) {
	good := string(repoFile(t, "docs", "cli-commands.json"))
	cases := []struct{ name, from, to, want string }{
		{"risk outside the list", `"risk": "safe"`, `"risk": "bogus"`, "risk"},
		{"misspelt member", `"timeoutMs": 5000`, `"timeoutMS": 5000`, "timeoutMS"},
		{"missing idempotent", `"idempotent": true,`, ``, "idempotent"},
		{"timeout over the ceiling", `"timeoutMs": 5000`, `"timeoutMs": 600001`, "timeoutMs"},
		{"cold start outside the list", `"coldStart": "oneshot"`, `"coldStart": "sometimes"`, "coldStart"},
		{"verb with capitals", `"name": "version"`, `"name": "Version"`, "verb"},
		{"argument kind outside the list", `"kind": "bool"`, `"kind": "boolean"`, "kind"},
		{"bad cliVersion", `"cliVersion": "1.0"`, `"cliVersion": "one"`, "cliVersion"},
		{"two positional values",
			`{ "name": "src", "kind": "string", "required": false, "default": "en", "description": "Source language code." }`,
			`{ "name": "src", "kind": "string", "required": false, "positional": true, "default": "en", "description": "Source language code." }`,
			"positional values"},
		{"a positional value that is a bool",
			`{ "name": "notranslate", "kind": "bool", "required": false, "default": false, "description": "Convert only, skip translation." }`,
			`{ "name": "notranslate", "kind": "bool", "required": false, "positional": true, "default": false, "description": "Convert only, skip translation." }`,
			"positional value"},
		{"two default verbs for one executable", `"name": "output status",`, `"name": "output status", "defaultFor": "doc-html-translate.exe",`, "defaultFor"},
		{"alias object without spelling", `"spelling": "-report",`, ``, "spelling"},
		{"unknown alias member", `{ "spelling": "-report", "legacy": true }`, `{ "spelling": "-report", "legacy": true, "extra": 1 }`, "extra"},
	}
	for _, c := range cases {
		bad := strings.Replace(good, c.from, c.to, 1)
		if bad == good {
			t.Fatalf("%s: the fixture text %q is not in the manifest", c.name, c.from)
		}
		problems := Check([]byte(bad))
		if len(problems) == 0 || !strings.Contains(strings.Join(problems, "\n"), c.want) {
			t.Errorf("%s: Check said %v, want a finding naming %q", c.name, problems, c.want)
		}
	}
	if p := Check([]byte(`{`)); len(p) == 0 {
		t.Error("Check accepted text that is not JSON")
	}
}

// Rule 20: docs/CLI.md is a render target; the gate child holds the same equality.
func TestCLIDocIsTheRender(t *testing.T) {
	m, err := ParseManifest(repoFile(t, "docs", "cli-commands.json"))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(repoFile(t, "docs", "CLI.md")), Markdown(m); got != want {
		t.Fatal("docs/CLI.md is not the render of docs/cli-commands.json: run `go run ./tools/clidoc -render`, never edit the page")
	}
}

func TestMarkdownCarriesEveryCommandAndNoLongDash(t *testing.T) {
	m, _ := Load()
	md := Markdown(m)
	for _, c := range m.Commands {
		for _, want := range []string{"## `" + c.Name + "`", c.Synopsis} {
			if !strings.Contains(md, want) {
				t.Errorf("the render lacks %q", want)
			}
		}
		for _, a := range c.Aliases {
			if a.Spelling != "" && !strings.Contains(md, "`"+a.Spelling+"`") {
				t.Errorf("the render lacks the alias %q of %s", a.Spelling, c.Name)
			}
		}
	}
	for _, bad := range []string{"—", "–", "…", "..."} {
		if strings.Contains(md, bad) {
			t.Errorf("the render contains %q: house text style", bad)
		}
	}
	if md != Markdown(m) {
		t.Error("the render is not deterministic")
	}
}

// ---- the front ----

// C1: commands --json is the manifest, byte for byte, with nothing on standard error.
func TestCommandsJSONIsTheManifestBytes(t *testing.T) {
	var c capture
	handled, code := Front([]string{"commands", "--json"}, c.env())
	if !handled || code != 0 {
		t.Fatalf("handled=%v exit=%d", handled, code)
	}
	if !bytes.Equal(c.out.Bytes(), repoFile(t, "docs", "cli-commands.json")) {
		t.Error("standard output is not the bytes of docs/cli-commands.json")
	}
	if c.err.Len() != 0 {
		t.Errorf("standard error: %q", c.err.String())
	}
	if c.opened != 1 {
		t.Errorf("the output was opened %d times, want once", c.opened)
	}
	// The manifest the front prints is itself valid JSON that a script reads.
	if _, err := ParseManifest(c.out.Bytes()); err != nil {
		t.Error(err)
	}
}

// The readable form begins with exactly what -h prints (assumption X1) and then lists the commands.
func TestCommandsReadableBeginsWithTheFrozenUsage(t *testing.T) {
	var c capture
	if handled, code := Front([]string{"Commands"}, c.env()); !handled || code != 0 {
		t.Fatalf("handled=%v exit=%d", handled, code)
	}
	got := c.out.String()
	if !strings.HasPrefix(got, golden(t)) {
		t.Error("the readable list does not begin with the bytes -h prints")
	}
	for _, want := range []string{"Commands of doc-html-translate (command set 1.0):", "  commands  ", "aliases: -h -help --help", "aliases: -version --version"} {
		if !strings.Contains(got, want) {
			t.Errorf("the readable list lacks %q:\n%s", want, got)
		}
	}
	// The launcher has no flag usage: its list is the verbs alone.
	var u capture
	env := u.env()
	env.Usage = nil
	Front([]string{"commands"}, env)
	if strings.Contains(u.out.String(), "Usage:") || !strings.HasPrefix(u.out.String(), "Commands of ") {
		t.Errorf("without a usage block the list should start at the commands:\n%s", u.out.String())
	}
}

// C2 (the version half): one line, the frozen one.
func TestVersionLineIsFrozen(t *testing.T) {
	var c capture
	if handled, code := Front([]string{"version"}, c.env()); !handled || code != 0 {
		t.Fatalf("handled=%v exit=%d", handled, code)
	}
	const want = "doc-html-translate " + testVersion + "\n"
	if c.out.String() != want || c.err.Len() != 0 {
		t.Errorf("version printed %q (stderr %q), want %q and nothing else (OCR-INVOCATION F)", c.out.String(), c.err.String(), want)
	}
	if VersionLine(testVersion) != want {
		t.Errorf("VersionLine = %q", VersionLine(testVersion))
	}

	var j capture
	Front([]string{"version", "--json"}, j.env())
	if got, want := j.out.String(), `{"product":"doc-html-translate","version":"`+testVersion+`","cliVersion":"1.0"}`+"\n"; got != want {
		t.Errorf("version --json = %q, want %q", got, want)
	}
}

// Rule 2 and C13: -version and --version behave as their verb. The conversion parser serves them (assumption X5) and
// the console converter prints VersionLine for both paths, so the two lines are one function's output.
func TestVersionAliasesPrintTheSameLine(t *testing.T) {
	m, _ := Load()
	for _, alias := range m.Find("version").Aliases {
		var c capture
		if handled, _ := Front([]string{alias.Spelling}, c.env()); handled || c.opened != 0 {
			t.Errorf("%s: the front took a frozen alias (handled=%v opened=%d)", alias.Spelling, handled, c.opened)
		}
		if _, err := config.ParseArgs([]string{alias.Spelling}); !errors.Is(err, config.ErrVersion) {
			t.Errorf("%s: the conversion parser answered %v, want ErrVersion", alias.Spelling, err)
		}
	}
}

// -h, -help and --help print the usage exactly as before the ticket: the golden was captured from the code before
// internal/config was touched.
func TestUsageEqualsGolden(t *testing.T) {
	if got := config.UsageText(); got != golden(t) {
		t.Errorf("config.UsageText() differs from testdata/help.golden:\n%s", got)
	}
	var err error
	got := stdoutOf(t, func() { _, err = config.ParseArgs([]string{"-h"}) })
	if !errors.Is(err, config.ErrHelp) || got != golden(t) {
		t.Errorf("-h printed %q (err %v), want the golden", got, err)
	}
	// UsageText must leave the sink as it found it, so a later -h still reaches standard output.
	if again := stdoutOf(t, func() { _, _ = config.ParseArgs([]string{"-help"}) }); again != golden(t) {
		t.Error("-help after UsageText did not print to standard output")
	}
}

func TestHelpAliasesPassThrough(t *testing.T) {
	m, _ := Load()
	for _, alias := range m.Find("commands").Aliases {
		var c capture
		if handled, _ := Front([]string{alias.Spelling}, c.env()); handled || c.opened != 0 {
			t.Errorf("%s: the front took a frozen alias (handled=%v opened=%d)", alias.Spelling, handled, c.opened)
		}
		var err error
		got := stdoutOf(t, func() { _, err = config.ParseArgs([]string{alias.Spelling}) })
		if !errors.Is(err, config.ErrHelp) || got != golden(t) {
			t.Errorf("%s: the conversion parser printed %q (err %v), want the golden usage", alias.Spelling, got, err)
		}
	}
}

// C4: an unknown option and an oversized request exit 10 and run nothing (unreserved first words pass to defaultFor).
func TestRefusalsExitTenAndRunNothing(t *testing.T) {
	big := strings.Repeat("a", 70000)
	mid := strings.Repeat("b", 30000)
	cases := []struct {
		name string
		args []string
		want string
	}{
		{"unknown option", []string{"commands", "--nope"}, `unknown option "--nope" for "commands"`},
		{"unknown option of version", []string{"version", "--nope=1"}, `unknown option "--nope" for "version"`},
		{"single-dash option", []string{"commands", "-json"}, `unknown option "-json"`},
		{"a word the verb does not take", []string{"commands", "extra"}, `unexpected value "extra" for "commands"`},
		{"a word after the end of options", []string{"version", "--", "extra"}, `unexpected value "extra"`},
		{"bool with a bad value", []string{"commands", "--json=maybe"}, `unexpected value "maybe" for "--json"`},
		{"missing value", []string{"commands", "--result"}, `option "--result" needs a value`},
		{"empty value", []string{"commands", "--result", ""}, `unexpected value "" for "--result"`},
		{"repeated option", []string{"version", "--result", "a", "--result", "b"}, `option "--result" was given more than once`},
		{"argument over 32 KiB as the first token", []string{big}, "too large"},
		{"argument over 32 KiB as a value", []string{"commands", "--result", big}, "too large"},
		{"request over 64 KiB in arguments of 30 KiB", []string{"book.epub", mid, mid, mid}, "too large"},
		{"output alone", []string{"output"}, `unknown command "output"`},
		{"uppercase verb prefix", []string{"OCR"}, `unknown command "OCR"`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var got capture
			dir := t.TempDir()
			t.Chdir(dir)
			handled, code := Front(c.args, got.env())
			if !handled || code != ExitUsage {
				t.Fatalf("handled=%v exit=%d, want true and 10", handled, code)
			}
			if got.out.Len() != 0 {
				t.Errorf("a refusal wrote to standard output: %q", got.out.String())
			}
			if !strings.Contains(got.err.String(), c.want) {
				t.Errorf("standard error %q does not say %q", got.err.String(), c.want)
			}
			if entries, _ := os.ReadDir(dir); len(entries) != 0 {
				t.Errorf("a refusal wrote files: %v", entries)
			}
		})
	}
	// APP-COMMANDS 0.2 item A.1: a reserved first word wins over a file of the same name on both executables.
	t.Run("reserved prefix wins over same-named file on both executables", func(t *testing.T) {
		dir := t.TempDir()
		t.Chdir(dir)
		if err := os.WriteFile(filepath.Join(dir, "ocr"), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
		for _, launcher := range []bool{false, true} {
			var got capture
			env := got.env()
			if launcher {
				env.Launcher = &LauncherHooks{BypassSingleInstance: true}
			}
			handled, code := Front([]string{"ocr"}, env)
			if !handled || code != ExitUsage || got.out.Len() != 0 || !strings.Contains(got.err.String(), `unknown command "ocr"`) {
				t.Errorf("launcher=%v: handled=%v exit=%d stdout=%q stderr=%q, want 10 and unknown command", launcher, handled, code, got.out.String(), got.err.String())
			}
		}
	})
	// "a following command still works" (C4): the refusal left nothing behind.
	var after capture
	if handled, code := Front([]string{"version"}, after.env()); !handled || code != 0 {
		t.Errorf("version after the refusals: handled=%v exit=%d", handled, code)
	}
}

// The sentence is in the interface language.
func TestRefusalIsInTheInterfaceLanguage(t *testing.T) {
	var c capture
	env := c.env()
	env.Lang = "ru"
	Front([]string{"commands", "--nope"}, env)
	if !strings.Contains(c.err.String(), `неизвестный параметр "--nope" для "commands"`) {
		t.Errorf("standard error %q is not the Russian sentence", c.err.String())
	}
}

// Everything that works today is not the front's: the front answers false, opens no output and writes nothing.
func TestLegacyCallsPassThrough(t *testing.T) {
	pass := [][]string{
		{},
		{"book.epub"},
		{"book.epub", "-noopen", "-folder", "out"},
		{"-noopen", "book.epub"},
		{"-google", "-max-cost", "2", "C:\\books\\a.epub"},
		{".\\book"},
		{".\\ocr"},
		{"./report"},
		{"..\\book.txt"},
		{"-register"},
		{"-ocr-langs"},
		{"-h"}, {"-help"}, {"--help"}, {"-version"}, {"--version"},
		{"frobnicate"},
		{"Frobnicate", "--json"},
		{"notes"},
		{"--", "-weird-name.epub"},
	}
	for _, args := range pass {
		var c capture
		if handled, code := Front(args, c.env()); handled || code != 0 || c.opened != 0 || c.out.Len()+c.err.Len() != 0 {
			t.Errorf("%q: handled=%v exit=%d opened=%d output=%q%q", args, handled, code, c.opened, c.out.String(), c.err.String())
		}
	}

	// Both executables answer manifest verbs directly without a conversion hook.
	var l capture
	if handled, code := Front([]string{"version"}, l.env()); !handled || code != 0 {
		t.Errorf("launcher version: handled=%v exit=%d", handled, code)
	}
}

// C11 for --result: the file is whole or absent, and it holds the answer.
func TestResultFileIsWrittenWholeOrNotAtAll(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.json")

	var c capture
	if handled, code := Front([]string{"commands", "--json", "--result=" + path}, c.env()); !handled || code != 0 {
		t.Fatalf("handled=%v exit=%d (%s)", handled, code, c.err.String())
	}
	got, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(got, Embedded()) || !bytes.Equal(c.out.Bytes(), Embedded()) {
		t.Fatalf("the result file is not the manifest bytes (err %v)", err)
	}

	vpath := filepath.Join(dir, "v.json")
	var v capture
	Front([]string{"version", "--result", vpath}, v.env())
	got, err = os.ReadFile(vpath)
	if err != nil || !bytes.Equal(got, VersionJSON(testVersion, "1.0")) {
		t.Fatalf("version --result wrote %q (err %v)", got, err)
	}
	if v.out.String() != VersionLine(testVersion) {
		t.Errorf("--result changed what version prints: %q", v.out.String())
	}

	// A target that cannot be written: exit 1, standard output empty, nothing created.
	missing := filepath.Join(dir, "no-such-folder", "out.json")
	var f capture
	handled, code := Front([]string{"commands", "--json", "--result", missing}, f.env())
	if !handled || code != ExitFailed || f.out.Len() != 0 || !strings.Contains(f.err.String(), "the result file could not be written") {
		t.Errorf("handled=%v exit=%d stdout=%q stderr=%q", handled, code, f.out.String(), f.err.String())
	}
	if _, err := os.Stat(missing); err == nil {
		t.Error("a result file appeared in a folder that does not exist")
	}
	// No temp file is left beside a good result either.
	entries, _ := os.ReadDir(dir)
	if len(entries) != 2 {
		t.Errorf("the folder holds %d entries after two results, want 2", len(entries))
	}
}

// ---- the grammar (rules 1 and 3) on a synthetic manifest, so two-word verbs and every option kind are exercised ----

func grammarManifest() *Manifest {
	yes := true
	return &Manifest{Product: "x", Commands: []Command{
		{Name: "output status", Args: []Arg{
			{Name: "in", Kind: "path", Required: &yes},
			{Name: "json", Kind: "bool"},
			{Name: "ollama", Kind: "bool", Aliases: []string{"free"}},
			{Name: "split", Kind: "int"},
			{Name: "ui-lang", Kind: "enum", Values: []string{"en", "ru"}},
		}},
		{Name: "ocr", Args: nil},
	}}
}

func TestGrammarMatchesVerbsOfOneOrTwoWordsWithoutCase(t *testing.T) {
	m := grammarManifest()
	if c, n := m.MatchVerb([]string{"Output", "STATUS", "--in", "x"}); c == nil || n != 2 || c.Name != "output status" {
		t.Errorf("two-word verb: %v %d", c, n)
	}
	if c, n := m.MatchVerb([]string{"ocr", "langs"}); c == nil || n != 1 || c.Name != "ocr" {
		t.Errorf("one-word verb followed by a word: %v %d", c, n)
	}
	if c, _ := m.MatchVerb([]string{"output"}); c != nil {
		t.Errorf("half of a two-word verb matched: %v", c)
	}
	if c, _ := m.MatchVerb([]string{"book.epub"}); c != nil {
		t.Errorf("a document matched a verb: %v", c)
	}
}

func TestGrammarOptionsAndValues(t *testing.T) {
	m := grammarManifest()
	cmd := m.Find("output status")
	parse := func(args ...string) (*Invocation, *UsageError) { return Parse(cmd, args) }

	inv, err := parse("--IN=Book.EPUB", "--Json", "--OLLAMA", "--split", "-5", "--ui-lang=ru")
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := inv.String("in"); v != "Book.EPUB" {
		t.Errorf("a value was changed: %q (values are never compared without case)", v)
	}
	if b, _ := inv.Bool("json"); !b {
		t.Error("--Json is not true")
	}
	if b, _ := inv.Bool("ollama"); !b {
		t.Error("--OLLAMA is not true")
	}
	if v, _ := inv.String("split"); v != "-5" {
		t.Errorf("--split -5 = %q", v)
	}
	// D4 (APP-COMMANDS 0.2 item E.5): option aliases (a.Aliases) belong to bare legacy calls, never canonical verbs.
	if _, err := parse("--in", "x", "--FREE"); err == nil || err.Reason != ReasonBadArgument {
		t.Errorf("--FREE on canonical verb: err = %v, want ReasonBadArgument", err)
	}

	if inv, err = parse("--in", "x", "--json=FALSE"); err != nil {
		t.Fatal(err)
	} else if b, _ := inv.Bool("json"); b {
		t.Error("--json=FALSE is true")
	}

	// A value is the next argument verbatim, even one that looks like an option.
	if inv, err = parse("--in", "--json"); err != nil {
		t.Fatal(err)
	} else if v, _ := inv.String("in"); v != "--json" {
		t.Errorf("--in --json took %q", v)
	}

	// Repeats are collected; a verb that wants one value refuses two.
	if inv, err = parse("--in", "a", "--in", "b"); err != nil {
		t.Fatal(err)
	} else if got := inv.Options["in"]; len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Errorf("repeated --in = %v", got)
	} else if _, e := inv.String("in"); e == nil || e.Reason != ReasonBadArgument {
		t.Errorf("a repeated single-valued option was accepted: %v", e)
	}

	// "--" ends the options: nothing may follow it for a verb with no positional word.
	if _, err = parse("--in", "x", "--"); err != nil {
		t.Errorf("a trailing -- was refused: %v", err)
	}

	for _, bad := range [][]string{
		{"--split=12x"}, {"--split="}, {"--ui-lang=EN"}, {"--ui-lang=de"}, {"--json=maybe"}, {"--in"}, {"--in="},
		{"--nope"}, {"--=x"}, {"-x"}, {"word"}, {"--", "word"},
	} {
		if _, e := parse(bad...); e == nil || e.Reason != ReasonBadArgument {
			t.Errorf("%q was accepted (%v)", bad, e)
		}
	}
}

func TestBoundsAreExact(t *testing.T) {
	arg := func(n int) string { return strings.Repeat("a", n) }
	if CheckBounds([]string{arg(MaxArgBytes)}) != nil {
		t.Error("an argument of exactly 32 KiB was refused")
	}
	if e := CheckBounds([]string{arg(MaxArgBytes + 1)}); e == nil || e.Reason != ReasonTooLarge {
		t.Errorf("an argument of 32 KiB + 1 was accepted: %v", e)
	}
	// 32768 + 1 separator + 32767 = 65536, exactly the request bound.
	if CheckBounds([]string{arg(MaxArgBytes), arg(MaxArgBytes - 1)}) != nil {
		t.Error("a request of exactly 64 KiB was refused")
	}
	if CheckBounds([]string{arg(MaxArgBytes), arg(MaxArgBytes)}) == nil {
		t.Error("a request of 64 KiB + 1 was accepted")
	}
	// Bytes, not characters: 11 000 three-byte characters are over the argument bound.
	if CheckBounds([]string{strings.Repeat("\u20ac", 11000)}) == nil {
		t.Error("the argument bound counted characters, not UTF-8 bytes")
	}
}

// ---- C13: every shipped spelling is in the manifest ----

var usageFlag = regexp.MustCompile(`(?m)^  -([a-z][a-z0-9-]*)`)

// shippedFlags are the flags of internal/config, read from the usage text the parser prints.
func shippedFlags(t *testing.T) []string {
	t.Helper()
	var out []string
	for _, m := range usageFlag.FindAllStringSubmatch(config.UsageText(), -1) {
		out = append(out, m[1])
	}
	if len(out) < 20 {
		t.Fatalf("read only %d flags from the usage text: %v", len(out), out)
	}
	return out
}

// covered reports whether the manifest lists the spelling -flag: as an option of a verb (--flag, or an option alias) or
// as an alias of a verb.
func covered(m *Manifest, flag string) bool {
	for _, c := range m.Commands {
		for _, a := range c.Aliases {
			if a.Spelling == "-"+flag || a.Spelling == "--"+flag {
				return true
			}
		}
		for _, a := range c.Args {
			if a.Name == flag {
				return true
			}
			for _, al := range a.Aliases {
				if al == flag {
					return true
				}
			}
		}
	}
	return false
}

// pendingFlags reads testdata/flags-pending.txt: "<flag> <ticket>" per line, # comments.
func pendingFlags(t *testing.T) map[string]string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "flags-pending.txt"))
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]string{}
	for i, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		f := strings.Fields(line)
		if len(f) != 2 {
			t.Fatalf("flags-pending.txt line %d: want \"<flag> <ticket>\", got %q", i+1, line)
		}
		// Every verb is in the manifest (ticket E0147 was the last), so no ticket is left that could add one: a line here
		// is a flag that ships with no verb, and the file may not take it.
		t.Fatalf("flags-pending.txt line %d: %q names ticket %q, but no ticket adds a verb any more; list the flag in docs/cli-commands.json instead", i+1, f[0], f[1])
		if _, dup := out[f[0]]; dup {
			t.Fatalf("flags-pending.txt lists %q twice", f[0])
		}
		out[f[0]] = f[1]
	}
	return out
}

// C13: the product keeps every spelling it ever shipped. A flag of internal/config must be an option of a verb, an
// alias, or - for a verb a later ticket adds - a line of flags-pending.txt, which may only shrink: a flag the manifest
// covers has to leave the file, and a new flag cannot enter it (no ticket is left that adds a verb; the file is empty).
func TestEveryShippedSpellingIsInTheManifest(t *testing.T) {
	m, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	pending := pendingFlags(t)
	shipped := map[string]bool{}
	for _, f := range shippedFlags(t) {
		shipped[f] = true
		_, isPending := pending[f]
		switch {
		case covered(m, f) && isPending:
			t.Errorf("-%s is in the manifest now: delete it from testdata/flags-pending.txt", f)
		case !covered(m, f) && !isPending:
			t.Errorf("-%s is shipped by internal/config and is neither an option of a verb, an alias, nor a pending line: list it in docs/cli-commands.json", f)
		}
	}
	for f := range pending {
		if !shipped[f] {
			t.Errorf("flags-pending.txt names -%s, which internal/config no longer ships", f)
		}
	}
}

// ---- messages ----

func TestEverySentenceIsTranslatedWithTheSameVerbs(t *testing.T) {
	verbs := regexp.MustCompile(`%[a-z]`)
	for _, key := range []string{
		msgUnknownCommand, msgUnknownOption, msgNeedsValue, msgUnexpected, msgRepeated, msgTooLarge, msgResultFailed,
		msgMissing, msgGoogleNeedsCap, msgSecretNeedsEnv, msgNotServed, msgNoSuchDocument,
		msgPackaged, msgOutExists, msgOutFolderMissing, msgNoSpace, msgOpenWithFailed, msgMenuFailed, msgSetFailed,
	} {
		tr, ok := i18n.All()[key]
		if !ok {
			t.Errorf("no translations for %q", key)
			continue
		}
		want := verbs.FindAllString(key, -1)
		for i, v := range tr {
			if got := verbs.FindAllString(v, -1); strings.Join(got, "") != strings.Join(want, "") {
				t.Errorf("%s translation of %q has verbs %v, want %v", i18n.Codes[i+1], key, got, want)
			}
		}
	}
}
