package commands

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	"doc-html-translate/internal/report"
	"doc-html-translate/internal/windowsreg"
)

// stubEffects replaces what the verbs do to the machine with recorders that answer a quiet success, and puts the real
// effects back at the end. The returned slice names every effect that was called, in order.
func stubEffects(t *testing.T) *[]string {
	t.Helper()
	saved := fx
	t.Cleanup(func() { fx = saved })
	var log []string
	fx = effects{
		ensureInstalled: func(_ context.Context, code string) (bool, error) {
			log = append(log, "ensure "+code)
			return true, nil
		},
		buildReport: func(report.BuildOptions) (string, int, error) {
			log = append(log, "report")
			return `C:\Reports\logs.zip`, 2, nil
		},
		registerOpenWith: func() ([]string, error) {
			log = append(log, "openwith")
			return []string{".epub", ".pdf"}, nil
		},
		registerContextMenu: func() ([]string, error) {
			log = append(log, "menu")
			return []string{".epub", ".pdf"}, nil
		},
		registerHandler: func() (windowsreg.Registration, error) {
			log = append(log, "handler")
			return windowsreg.Registration{Default: []string{".epub"}, Blocked: []string{".pdf"}}, nil
		},
		unregister: func() ([]string, error) {
			log = append(log, "unregister")
			return []string{".epub"}, nil
		},
		handlerStatus: func() windowsreg.Status {
			log = append(log, "status")
			return windowsreg.Status{Default: []string{".pdf"}}
		},
		packaged: func() bool { return false },
	}
	return &log
}

// runMaintain runs the front as the console converter does, with the maintenance group served.
func runMaintain(t *testing.T, args ...string) (c *capture, handled bool, code int) {
	t.Helper()
	c = &capture{}
	env := c.env()
	env.Maintain = true
	handled, code = Front(args, env)
	return c, handled, code
}

// answered checks the one shape every successful verb has: handled, exit 0, an answer on standard output, nothing on
// standard error.
func answered(t *testing.T, c *capture, handled bool, code int) {
	t.Helper()
	if !handled || code != ExitDone || c.err.Len() != 0 || c.out.Len() == 0 {
		t.Fatalf("handled=%v exit=%d stdout=%q stderr=%q, want an answer and exit 0", handled, code, c.out.String(), c.err.String())
	}
}

// failed checks the one shape every failed verb has: the exit code, an empty standard output, the sentence on stderr.
func failed(t *testing.T, c *capture, handled bool, code, want int, sentence string) {
	t.Helper()
	if !handled || code != want || c.out.Len() != 0 || !strings.Contains(c.err.String(), sentence) {
		t.Errorf("handled=%v exit=%d stdout=%q stderr=%q, want exit %d, nothing on stdout and %q on stderr", handled, code, c.out.String(), c.err.String(), want, sentence)
	}
}

func TestOcrDownloadVerb(t *testing.T) {
	log := stubEffects(t)

	// The pack is there: exit 0, installed, and the answer says nothing was downloaded.
	fx.ensureInstalled = func(_ context.Context, code string) (bool, error) {
		*log = append(*log, "ensure "+code)
		return false, nil
	}
	c, handled, code := runMaintain(t, "ocr", "download", "--lang", "rus", "--json")
	answered(t, c, handled, code)
	if want := `{"lang":"rus","installed":true,"downloaded":false}` + "\n"; c.out.String() != want {
		t.Errorf("stdout = %q, want %q", c.out.String(), want)
	}
	if !reflect.DeepEqual(*log, []string{"ensure rus"}) {
		t.Errorf("effects = %v, want one ensure of rus", *log)
	}

	// The pack was fetched: downloaded is true; the readable form is the data names and values.
	fx.ensureInstalled = func(context.Context, string) (bool, error) { return true, nil }
	c, handled, code = runMaintain(t, "ocr", "download", "--lang", "rus")
	answered(t, c, handled, code)
	if want := "lang: rus\ninstalled: true\ndownloaded: true\n"; c.out.String() != want {
		t.Errorf("readable = %q, want %q", c.out.String(), want)
	}

	// --result is the printed JSON, written whole.
	res := filepath.Join(t.TempDir(), "r.json")
	c, handled, code = runMaintain(t, "ocr", "download", "--lang", "rus", "--json", "--result", res)
	answered(t, c, handled, code)
	if b, err := os.ReadFile(res); err != nil || string(b) != c.out.String() {
		t.Errorf("result file = %q (%v), want %q", b, err, c.out.String())
	}
	// A result that cannot be written is exit 1 with nothing on standard output.
	c, handled, code = runMaintain(t, "ocr", "download", "--lang", "rus", "--result", filepath.Join(t.TempDir(), "no", "r.json"))
	failed(t, c, handled, code, ExitFailed, "the result file could not be written")

	// A code outside the catalogue is a wrong value: exit 10, the sentence names the catalogue, nothing is fetched.
	*log = nil
	for _, bad := range []string{"xx", "../../x", `..\..\x`, "RUS", " rus"} {
		c, handled, code = runMaintain(t, "ocr", "download", "--lang", bad)
		failed(t, c, handled, code, ExitUsage, "is not an OCR language this app offers")
		if !strings.Contains(c.err.String(), "rus") {
			t.Errorf("%q: the sentence does not name the catalogue: %q", bad, c.err.String())
		}
	}
	c, handled, code = runMaintain(t, "ocr", "download")
	failed(t, c, handled, code, ExitUsage, `missing required --lang for "ocr download"`)
	if len(*log) != 0 {
		t.Errorf("a refusal reached the effects: %v", *log)
	}

	// A download that fails is exit 1.
	fx.ensureInstalled = func(context.Context, string) (bool, error) { return false, errors.New("the server said no") }
	c, handled, code = runMaintain(t, "ocr", "download", "--lang", "rus")
	failed(t, c, handled, code, ExitFailed, "the server said no")
}

func TestReportBuildVerb(t *testing.T) {
	log := stubEffects(t)
	var got report.BuildOptions
	fx.buildReport = func(o report.BuildOptions) (string, int, error) {
		got = o
		*log = append(*log, "report")
		return `C:\Reports\logs.zip`, 2, nil
	}

	c, handled, code := runMaintain(t, "report", "build", "--json")
	answered(t, c, handled, code)
	if want := `{"path":"C:\\Reports\\logs.zip","droppedLogs":2}` + "\n"; c.out.String() != want {
		t.Errorf("stdout = %q, want %q", c.out.String(), want)
	}
	if got.Out != "" || got.Overwrite || got.AppVersion != testVersion || got.At.IsZero() {
		t.Errorf("default options = %+v, want no Out, no Overwrite, version %s and a time", got, testVersion)
	}
	c, handled, code = runMaintain(t, "report", "build", "--out", `D:\x\logs.zip`, "--overwrite")
	answered(t, c, handled, code)
	if got.Out != `D:\x\logs.zip` || !got.Overwrite {
		t.Errorf("options = %+v, want Out and Overwrite as given", got)
	}
	if want := "path: C:\\Reports\\logs.zip\ndroppedLogs: 2\n"; c.out.String() != want {
		t.Errorf("readable = %q, want %q", c.out.String(), want)
	}
	// --overwrite alone is accepted and means nothing (assumption W8).
	c, handled, code = runMaintain(t, "report", "build", "--overwrite")
	answered(t, c, handled, code)
	if got.Out != "" || !got.Overwrite {
		t.Errorf("options = %+v", got)
	}

	// Each way the archive cannot be written has its own exit and sentence; nothing is printed on standard output.
	diskFull := syscall.ENOSPC
	if runtime.GOOS == "windows" {
		diskFull = syscall.Errno(112)
	}
	for _, tc := range []struct {
		name     string
		err      error
		exit     int
		sentence string
	}{
		{"exists", report.ErrOutExists, ExitNotFound, `already exists: add "--overwrite" to replace it`},
		{"missing folder", report.ErrOutFolderMissing, ExitNotFound, `the folder of "--out" does not exist`},
		{"disk full", &os.PathError{Op: "write", Path: "x", Err: diskFull}, ExitNoSpace, "no space left for the archive"},
		{"anything else", errors.New("list run logs: access denied"), ExitFailed, "list run logs: access denied"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx.buildReport = func(report.BuildOptions) (string, int, error) { return "", 0, tc.err }
			c, handled, code := runMaintain(t, "report", "build", "--out", `D:\x`)
			failed(t, c, handled, code, tc.exit, tc.sentence)
		})
	}
}

// D6 (APP-COMMANDS 0.3 item V): --out naming a folder writes DIAGNOSTIC-REPORT 8A's file inside it, and a missing
// folder ending in a separator is refused exit 11 without creating it.
func TestReportBuildVerbFolderTarget(t *testing.T) {
	stubEffects(t)
	fx.buildReport = report.Build
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)
	t.Setenv("LOCALAPPDATA", filepath.Join(home, "AppData", "Local"))

	dir := t.TempDir()
	c, handled, code := runMaintain(t, "report", "build", "--out", dir, "--json")
	answered(t, c, handled, code)
	var data ReportBuildData
	if err := json.Unmarshal(c.out.Bytes(), &data); err != nil {
		t.Fatalf("json %q: %v", c.out.String(), err)
	}
	if filepath.Dir(data.Path) != dir || !strings.HasPrefix(filepath.Base(data.Path), "doc-html-translate-logs-") {
		t.Errorf("folder target wrote %q outside %q or wrong name", data.Path, dir)
	}
	missing := filepath.Join(dir, "missing") + string(os.PathSeparator)
	c2, handled2, code2 := runMaintain(t, "report", "build", "--out", missing)
	failed(t, c2, handled2, code2, ExitNotFound, `the folder of "--out" does not exist`)
	if _, err := os.Stat(filepath.Join(dir, "missing")); !os.IsNotExist(err) {
		t.Errorf("missing folder was created: %v", err)
	}
}

// The verb on the real archive builder: a fresh file, the same line refused, then replaced whole. The archive opens and
// carries the environment summary.
func TestReportBuildVerbWritesTheFileItNames(t *testing.T) {
	stubEffects(t)
	fx.buildReport = report.Build
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)
	t.Setenv("LOCALAPPDATA", filepath.Join(home, "AppData", "Local"))
	out := filepath.Join(t.TempDir(), "logs.zip")

	c, handled, code := runMaintain(t, "report", "build", "--out", out, "--json")
	answered(t, c, handled, code)
	if !strings.Contains(c.out.String(), `"droppedLogs":0`) {
		t.Errorf("stdout = %q", c.out.String())
	}
	zr, err := zip.OpenReader(out)
	if err != nil {
		t.Fatalf("the named file does not open as an archive: %v", err)
	}
	names := map[string]bool{}
	for _, f := range zr.File {
		names[f.Name] = true
	}
	_ = zr.Close()
	if !names["environment.txt"] || !names["report.txt"] {
		t.Errorf("archive entries = %v", names)
	}
	if _, err := os.Stat(filepath.Join(home, "AppData", "Local", "doc-html-translate", "reports")); !os.IsNotExist(err) {
		t.Errorf("the default reports folder was made for a named target: %v", err)
	}

	c, handled, code = runMaintain(t, "report", "build", "--out", out)
	failed(t, c, handled, code, ExitNotFound, `already exists: add "--overwrite" to replace it`)

	if err := os.WriteFile(out, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	c, handled, code = runMaintain(t, "report", "build", "--out", out, "--overwrite")
	answered(t, c, handled, code)
	if zr, err := zip.OpenReader(out); err != nil {
		t.Errorf("the replaced file does not open as an archive: %v", err)
	} else {
		_ = zr.Close()
	}
}

func TestIntegrationVerbs(t *testing.T) {
	log := stubEffects(t)

	// Add: "Open with" first, then the right-click entry, as the flag does; no default handler call.
	c, handled, code := runMaintain(t, "integration", "add", "--json")
	answered(t, c, handled, code)
	if want := `{"extensions":[".epub",".pdf"]}` + "\n"; c.out.String() != want {
		t.Errorf("stdout = %q, want %q", c.out.String(), want)
	}
	if !reflect.DeepEqual(*log, []string{"openwith", "menu"}) {
		t.Errorf("effects = %v, want openwith then menu", *log)
	}
	c, handled, code = runMaintain(t, "integration", "add")
	answered(t, c, handled, code)
	if want := "extensions: .epub .pdf\n"; c.out.String() != want {
		t.Errorf("readable = %q, want %q", c.out.String(), want)
	}

	// "Open with" fails: the flag stops there, so the verb does; exit 1 and the console's own sentence.
	*log = nil
	fx.registerOpenWith = func() ([]string, error) { *log = append(*log, "openwith"); return nil, errors.New("access denied") }
	c, handled, code = runMaintain(t, "integration", "add")
	failed(t, c, handled, code, ExitFailed, `Could not add the "Open with" entry: access denied`)
	if !reflect.DeepEqual(*log, []string{"openwith"}) {
		t.Errorf("effects after a failed Open with = %v", *log)
	}
	// The right-click entry fails after it: exit 1 for the verb (the flag only warns).
	fx.registerOpenWith = func() ([]string, error) { return []string{".epub"}, nil }
	fx.registerContextMenu = func() ([]string, error) { return nil, errors.New("key is locked") }
	c, handled, code = runMaintain(t, "integration", "add")
	failed(t, c, handled, code, ExitFailed, `Could not add the right-click "Convert to HTML" entry: key is locked`)
	if strings.HasPrefix(strings.TrimPrefix(c.err.String(), "Error: "), " ") {
		t.Errorf("the sentence keeps the console's leading spaces: %q", c.err.String())
	}
}

func TestDefaultHandlerVerbs(t *testing.T) {
	log := stubEffects(t)

	// Set: the handler, then the right-click entry; exit 0 although Windows keeps one type (complete says so).
	c, handled, code := runMaintain(t, "default-handler", "set", "--json")
	answered(t, c, handled, code)
	want := `{"complete":false,"default":[".epub"],"blocked":[".pdf"],"unknown":[],"failed":[]}` + "\n"
	if c.out.String() != want {
		t.Errorf("stdout = %q, want %q", c.out.String(), want)
	}
	if !reflect.DeepEqual(*log, []string{"handler", "menu"}) {
		t.Errorf("effects = %v, want handler then menu", *log)
	}
	c, handled, code = runMaintain(t, "default-handler", "set")
	answered(t, c, handled, code)
	if want := "complete: false\ndefault: .epub\nblocked: .pdf\nunknown: none\nfailed: none\n"; c.out.String() != want {
		t.Errorf("readable = %q, want %q", c.out.String(), want)
	}
	fx.registerHandler = func() (windowsreg.Registration, error) {
		return windowsreg.Registration{Default: []string{".epub", ".pdf"}}, nil
	}
	c, handled, code = runMaintain(t, "default-handler", "set", "--json")
	answered(t, c, handled, code)
	if !strings.HasPrefix(c.out.String(), `{"complete":true,`) {
		t.Errorf("a complete registration: %q", c.out.String())
	}
	// Nothing could be registered: exit 1, the right-click entry is not attempted.
	*log = nil
	fx.registerHandler = func() (windowsreg.Registration, error) {
		*log = append(*log, "handler")
		return windowsreg.Registration{Failed: []string{".epub"}}, errors.New("failed to register any extensions")
	}
	c, handled, code = runMaintain(t, "default-handler", "set")
	failed(t, c, handled, code, ExitFailed, "Could not set as default handler: failed to register any extensions")
	if !reflect.DeepEqual(*log, []string{"handler"}) {
		t.Errorf("effects after a failed handler = %v", *log)
	}
	fx.registerHandler = func() (windowsreg.Registration, error) {
		return windowsreg.Registration{Default: []string{".epub"}}, nil
	}
	fx.registerContextMenu = func() ([]string, error) { return nil, errors.New("key is locked") }
	c, handled, code = runMaintain(t, "default-handler", "set")
	failed(t, c, handled, code, ExitFailed, `Could not add the right-click "Convert to HTML" entry: key is locked`)

	// Release: what was released, then what Windows still opens with the product; nothing to release is exit 0.
	log = stubEffects(t)
	c, handled, code = runMaintain(t, "default-handler", "release", "--json")
	answered(t, c, handled, code)
	if want := `{"released":[".epub"],"stillDefault":[".pdf"]}` + "\n"; c.out.String() != want {
		t.Errorf("stdout = %q, want %q", c.out.String(), want)
	}
	if !reflect.DeepEqual(*log, []string{"unregister", "status"}) {
		t.Errorf("effects = %v, want unregister then status", *log)
	}
	fx.unregister = func() ([]string, error) { return nil, nil }
	fx.handlerStatus = func() windowsreg.Status { return windowsreg.Status{} }
	c, handled, code = runMaintain(t, "default-handler", "release", "--json")
	answered(t, c, handled, code)
	if want := `{"released":[],"stillDefault":[]}` + "\n"; c.out.String() != want {
		t.Errorf("nothing to release: stdout = %q, want %q", c.out.String(), want)
	}
	c, handled, code = runMaintain(t, "default-handler", "release")
	answered(t, c, handled, code)
	if want := "released: none\nstillDefault: none\n"; c.out.String() != want {
		t.Errorf("readable = %q, want %q", c.out.String(), want)
	}
	fx.unregister = func() ([]string, error) { return []string{".epub"}, errors.New("could not release .pdf") }
	c, handled, code = runMaintain(t, "default-handler", "release")
	failed(t, c, handled, code, ExitFailed, "could not release .pdf")
}

// Under a Store package the registry is virtualized and the manifest declares the associations: the three registry verbs
// answer exit 11 with a sentence that says so and call no registry function; the other two are not affected.
func TestRegistryVerbsRefuseUnderAPackage(t *testing.T) {
	log := stubEffects(t)
	fx.packaged = func() bool { return true }
	for _, verb := range [][]string{{"integration", "add"}, {"default-handler", "set"}, {"default-handler", "release"}} {
		c, handled, code := runMaintain(t, append(verb, "--json")...)
		failed(t, c, handled, code, ExitNotFound, "Microsoft Store package")
		if !strings.Contains(c.err.String(), `"`+strings.Join(verb, " ")+`"`) {
			t.Errorf("%v: the sentence does not name the verb: %q", verb, c.err.String())
		}
	}
	if len(*log) != 0 {
		t.Errorf("a refused verb reached the registry: %v", *log)
	}
	for _, verb := range [][]string{{"ocr", "download", "--lang", "rus"}, {"report", "build"}} {
		c, handled, code := runMaintain(t, verb...)
		answered(t, c, handled, code)
	}
	if ReasonPackaged != "packaged" {
		t.Errorf("ReasonPackaged = %q", ReasonPackaged)
	}
	// The refusal is in the interface language: the verb stays in the sentence, in every language of the catalogue.
	for _, lang := range []string{"ru", "de", "zh"} {
		var c capture
		env := c.env()
		env.Maintain, env.Lang = true, lang
		if _, code := Front([]string{"integration", "add"}, env); code != ExitNotFound || !strings.Contains(c.err.String(), `"integration add"`) || strings.Contains(c.err.String(), "Microsoft Store package") {
			t.Errorf("%s: exit=%d stderr=%q, want the translated sentence naming the verb", lang, code, c.err.String())
		}
	}
}

// APP-COMMANDS rule 4: none of the five verbs asks anything of a person. With standard input an open pipe nobody writes
// to, every verb returns; and the file that serves them has no way to read a line, to offer Settings or to pause.
func TestMaintainVerbsNeverPrompt(t *testing.T) {
	stubEffects(t)
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	saved := os.Stdin
	os.Stdin = r
	defer func() {
		os.Stdin = saved
		_ = w.Close()
		_ = r.Close()
	}()
	done := make(chan []int, 1)
	go func() {
		var codes []int
		for _, args := range [][]string{
			{"ocr", "download", "--lang", "rus"}, {"report", "build"}, {"integration", "add"},
			{"default-handler", "set"}, {"default-handler", "release"},
		} {
			c := &capture{}
			env := c.env()
			env.Maintain = true
			_, code := Front(args, env)
			codes = append(codes, code)
		}
		done <- codes
	}()
	select {
	case codes := <-done:
		for i, code := range codes {
			if code != ExitDone {
				t.Errorf("verb %d ended %d, want 0", i, code)
			}
		}
	case <-time.After(3 * time.Second):
		t.Fatal("a verb waited on standard input")
	}

	src, err := os.ReadFile("maintain.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, token := range []string{"os.Stdin", "bufio", "askYes", "OpenDefaultAppsSettings", "fmt.Scan", "ReadString"} {
		if strings.Contains(string(src), token) {
			t.Errorf("maintain.go names %s: a verb must never prompt (APP-COMMANDS rule 4)", token)
		}
	}
}

// The launcher registers itself under its own name, so it does not serve the group: each of the five is exit 10 with the
// sentence that the console program runs it, and no effect is called.
func TestMaintainGroupIsNotServedByTheLauncher(t *testing.T) {
	log := stubEffects(t)
	for _, args := range [][]string{
		{"ocr", "download", "--lang", "rus"}, {"report", "build"}, {"integration", "add"},
		{"default-handler", "set"}, {"default-handler", "release"},
	} {
		var c capture
		env := c.env() // no Maintain: the launcher
		handled, code := Front(args, env)
		failed(t, &c, handled, code, ExitUsage, "is run by doc-html-translate.exe, not by this program")
		if c.opened != 1 {
			t.Errorf("%v: opened %d consoles, want 1", args, c.opened)
		}
	}
	if len(*log) != 0 {
		t.Errorf("the launcher reached the effects: %v", *log)
	}
}

// Refusals of the grammar (rule 3): exit 10, nothing on standard output, no effect called, no file written.
func TestMaintainGroupRefusals(t *testing.T) {
	log := stubEffects(t)
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{"ocr download unknown option", []string{"ocr", "download", "--nope"}, `unknown option "--nope" for "ocr download"`},
		{"ocr download stray word", []string{"ocr", "download", "--lang", "rus", "extra"}, `unexpected value "extra" for "ocr download"`},
		{"ocr download lang without a value", []string{"ocr", "download", "--lang"}, `option "--lang" needs a value`},
		{"ocr download empty lang", []string{"ocr", "download", "--lang", ""}, `unexpected value "" for "--lang"`},
		{"ocr download repeated", []string{"ocr", "download", "--lang", "rus", "--lang", "eng"}, `option "--lang" was given more than once`},
		{"report build unknown option", []string{"report", "build", "--zip"}, `unknown option "--zip" for "report build"`},
		{"report build stray word", []string{"report", "build", "logs.zip"}, `unexpected value "logs.zip" for "report build"`},
		{"report build out without a value", []string{"report", "build", "--out"}, `option "--out" needs a value`},
		{"report build empty out", []string{"report", "build", "--out", ""}, `unexpected value "" for "--out"`},
		{"report build bad bool", []string{"report", "build", "--overwrite=maybe"}, `unexpected value "maybe" for "--overwrite"`},
		{"report build single dash", []string{"report", "build", "-overwrite"}, `unknown option "-overwrite"`},
		{"integration add unknown option", []string{"integration", "add", "--default"}, `unknown option "--default" for "integration add"`},
		{"integration add stray word", []string{"integration", "add", ".epub"}, `unexpected value ".epub" for "integration add"`},
		{"default-handler set unknown option", []string{"default-handler", "set", "--force"}, `unknown option "--force" for "default-handler set"`},
		{"default-handler release stray word", []string{"default-handler", "release", "all"}, `unexpected value "all" for "default-handler release"`},
		{"default-handler alone", []string{"default-handler"}, `unknown command "default-handler"`},
		{"integration alone", []string{"integration"}, `unknown command "integration"`},
		{"integration remove", []string{"integration", "remove"}, `unknown command "integration"`},
		{"report alone", []string{"report"}, `unknown command "report"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			t.Chdir(dir)
			c, handled, code := runMaintain(t, tc.args...)
			failed(t, c, handled, code, ExitUsage, tc.want)
			if entries, _ := os.ReadDir(dir); len(entries) != 0 {
				t.Errorf("a refusal wrote files: %v", entries)
			}
		})
	}
	if len(*log) != 0 {
		t.Errorf("a refusal reached the effects: %v", *log)
	}
}

func TestManifestNamesTheMaintenanceGroup(t *testing.T) {
	m, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []struct {
		name, risk string
		aliases    []string
		timeout    int
		idempotent bool
		exits      []int
		packaged   bool
	}{
		{"ocr download", "leaves-machine", []string{"-ocr-download"}, 600000, true, []int{0, 1, 10}, false},
		{"report build", "writes-user-data", []string{"-report"}, 30000, false, []int{0, 1, 10, 11, 13}, false},
		{"integration add", "writes-user-data", []string{"", "-register-openwith"}, 10000, true, []int{0, 1, 10, 11}, true},
		{"default-handler set", "writes-user-data", []string{"-register"}, 10000, true, []int{0, 1, 10, 11}, true},
		{"default-handler release", "writes-user-data", []string{"-unregister"}, 10000, true, []int{0, 1, 10, 11}, true},
	} {
		c := m.Find(strings.Fields(want.name)...)
		if c == nil {
			t.Errorf("%s is not in the manifest", want.name)
			continue
		}
		if c.Risk != want.risk || c.Forward != "never" || c.DefaultFor != "" || c.Since != "1.0" || c.ColdStart != "oneshot" || c.Wait != "done" ||
			c.TimeoutMs == nil || *c.TimeoutMs != want.timeout || c.Interactive == nil || *c.Interactive ||
			c.Idempotent == nil || *c.Idempotent != want.idempotent {
			t.Errorf("%s: risk %q forward %q since %q coldStart %q timeout %v interactive %v idempotent %v", want.name, c.Risk, c.Forward, c.Since, c.ColdStart, c.TimeoutMs, c.Interactive, c.Idempotent)
		}
		var spellings []string
		for _, a := range c.Aliases {
			spellings = append(spellings, a.Spelling)
		}
		if !reflect.DeepEqual(spellings, want.aliases) {
			t.Errorf("%s: aliases %v, want %v", want.name, spellings, want.aliases)
		}
		var exits []int
		for _, e := range c.ExitCodes {
			exits = append(exits, e.Code)
			if e.Code == ExitNotFound && want.packaged && !strings.Contains(e.Meaning, ReasonPackaged) {
				t.Errorf("%s: exit 11 does not name the reason %s: %q", want.name, ReasonPackaged, e.Meaning)
			}
			if want.name == "report build" && e.Code == ExitNotFound && !strings.Contains(e.Meaning, ReasonExists) {
				t.Errorf("%s: exit 11 does not name the reason %s: %q", want.name, ReasonExists, e.Meaning)
			}
		}
		if !reflect.DeepEqual(exits, want.exits) {
			t.Errorf("%s: exit codes %v, want %v", want.name, exits, want.exits)
		}
	}
}

// C13 is closed: every flag the converter ships is an option or an alias of a verb, so the list of pending flags is empty
// and no line may be added to it.
func TestPendingFlagsIsEmpty(t *testing.T) {
	if p := pendingFlags(t); len(p) != 0 {
		t.Errorf("flags-pending.txt still holds %v", p)
	}
}
