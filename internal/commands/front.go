package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"doc-html-translate/internal/config"
	"doc-html-translate/internal/fsutil"
	"doc-html-translate/internal/i18n"
	"doc-html-translate/internal/syslocale"
)

var (
	launcherGroup = map[string]bool{"status": true, "show": true, "quit": true, "open": true}
	launcherOnly  = map[string]bool{"show": true, "open": true}
)

// The exit codes of APP-COMMANDS rule 11 that the front returns.
const (
	ExitDone   = 0
	ExitFailed = 1
	ExitUsage  = 10
)

// Env is what the front needs from the executable that calls it.
type Env struct {
	// Version is the build's version (main.Version).
	Version string
	// Usage returns the flag usage of the conversion parser, which the readable `commands` output begins with. Nil
	// leaves it out (the launcher has no flag usage).
	Usage func() string
	// Open is called once, just before the front writes its first byte, and returns where standard output and standard
	// error are. The window-subsystem launcher attaches to its parent's console here (rule 9), and only when the front
	// answers; nil means os.Stdout and os.Stderr.
	Open func() (stdout, stderr io.Writer)
	// Lang is the interface language of a refusal; empty means the Windows UI language (no saved setting is read).
	Lang string
	// Convert runs one conversion described by cfg and returns its exit code (OCR-INVOCATION section D). The console
	// converter supplies it; nil means this executable does not serve `convert` and `output status` (the launcher),
	// which then say that doc-html-translate.exe does (ticket E0145, assumption Y7). The inspection group (`doctor`,
	// `input check`, `ocr langs`, ticket E0146) needs no hook: both executables serve it.
	Convert func(cfg config.Config) int
	// Maintain says this executable serves the maintenance group (`ocr download`, `report build`, `integration add`,
	// `default-handler set`, `default-handler release`, ticket E0147). Only the console converter sets it: the registry
	// functions write the running executable as the handler. Any other process refuses the five with the sentence that
	// doc-html-translate.exe runs them (assumption W2).
	Maintain bool
	// Launcher connects doc-html-ui.exe to the single-instance lock, command pipe and window lifecycle (ticket E0144).
	Launcher *LauncherHooks
	// PipeName and MutexName override the session defaults in tests.
	PipeName  string
	MutexName string
}

// VersionLine is the one line `-version` and `version` print: OCR-INVOCATION section F, nothing else on standard
// output. Both paths use this function, so they cannot differ.
func VersionLine(version string) string { return Product + " " + version + "\n" }

// VersionJSON is `version --json`: the product, its version and the version of the command set, as one JSON object.
func VersionJSON(version, cliVersion string) []byte {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(struct {
		Product    string `json:"product"`
		Version    string `json:"version"`
		CLIVersion string `json:"cliVersion"`
	}{Product, version, cliVersion})
	return b.Bytes()
}

// Front reads a command line before the program does anything else. It answers true, with the exit code, when the
// line was a command of the manifest or a refusal; false means the line is not the front's and goes on to the
// conversion parser (or, with no arguments, to the first-run flow) exactly as before.
//
// It touches no window, setting, worker or log store (APP-COMMANDS rules 6 and 17): everything it needs is the
// executable's own copy of the manifest. The aliases `-h`, `-help`, `--help`, `-version` and `--version` are frozen
// spellings served by the conversion parser, so they come back as not handled (assumption X5).
func Front(args []string, env Env) (handled bool, exit int) {
	fr := &front{env: env}
	if len(args) == 0 {
		if env.Launcher == nil || env.Launcher.BypassSingleInstance {
			return false, 0
		}
		return true, fr.launcherBareShow()
	}
	if err := CheckBounds(args); err != nil {
		return true, fr.refuse(err)
	}
	m, err := Load()
	if err != nil {
		return false, 0 // an unreadable copy must not stop a conversion; the tests keep it readable
	}
	cmd, used := m.MatchVerb(args)
	if cmd == nil {
		if m.hasVerbPrefix(args[0]) {
			return true, fr.refuse(usage(ReasonUnknownVerb, msgUnknownCommand, args[0]))
		}
		if env.Launcher != nil {
			return fr.launcherDefaultOpen(args)
		}
		return false, 0
	}
	if convertGroup[cmd.Name] && env.Convert == nil {
		return true, fr.refuse(usage(ReasonUnknownVerb, msgNotServed, cmd.Name))
	}
	if maintainGroup[cmd.Name] && !env.Maintain {
		return true, fr.refuse(usage(ReasonUnknownVerb, msgNotServed, cmd.Name))
	}
	if launcherOnly[cmd.Name] && env.Launcher == nil {
		return true, fr.refuse(usage(ReasonUnknownVerb, msgNotServedByLauncher, cmd.Name))
	}
	inv, uerr := Parse(cmd, args[used:])
	if uerr != nil {
		return true, fr.refuse(uerr)
	}
	switch cmd.Name {
	case "commands":
		return true, fr.commands(m, inv)
	case "version":
		return true, fr.version(m, inv)
	case "convert":
		return true, fr.convert(inv)
	case "output status":
		return true, fr.outputStatus(inv)
	case "doctor":
		return true, fr.doctor(inv)
	case "input check":
		return true, fr.inputCheck(inv)
	case "ocr langs":
		return true, fr.ocrLangs(inv)
	case "ocr download":
		return true, fr.ocrDownload(inv)
	case "report build":
		return true, fr.reportBuild(inv)
	case "integration add":
		return true, fr.integrationAdd(inv)
	case "default-handler set":
		return true, fr.defaultHandlerSet(inv)
	case "default-handler release":
		return true, fr.defaultHandlerRelease(inv)
	case "status":
		return true, fr.launcherStatus(cmd, inv)
	case "show":
		return true, fr.launcherShow(cmd, inv)
	case "quit":
		return true, fr.launcherQuit(cmd, inv, args[used:])
	case "open":
		return true, fr.launcherOpen(cmd, inv)
	}
	// A verb of the manifest that this front does not serve is a manifest ahead of the code: refuse, never guess.
	return true, fr.refuse(usage(ReasonUnknownVerb, msgUnknownCommand, cmd.Name))
}

type front struct {
	env      Env
	out, err io.Writer
}

// streams opens the output channels once, on first use.
func (f *front) streams() (io.Writer, io.Writer) {
	if f.out == nil {
		f.out, f.err = os.Stdout, os.Stderr
		if f.env.Open != nil {
			f.out, f.err = f.env.Open()
		}
	}
	return f.out, f.err
}

func (f *front) lang() string {
	if f.env.Lang != "" {
		return f.env.Lang
	}
	return i18n.Resolve("", "", syslocale.Lang())
}

// refuse prints one sentence on standard error and returns exit 10 (rule 3). Nothing was executed.
func (f *front) refuse(e *UsageError) int {
	_, stderr := f.streams()
	fmt.Fprintf(stderr, "Error: %s\n", e.Message(f.lang()))
	return ExitUsage
}

func (f *front) fail(key string) int {
	_, stderr := f.streams()
	fmt.Fprintf(stderr, "Error: %s\n", i18n.T(f.lang(), key))
	return ExitFailed
}

// answer writes the response data to --result when it was given (whole or not at all, rule 9). The caller prints
// the answer after it, so a result file that cannot be written leaves standard output empty.
func (f *front) answer(inv *Invocation, data []byte) int {
	path, uerr := inv.String("result")
	if uerr != nil {
		return f.refuse(uerr)
	}
	if path != "" {
		if err := fsutil.WriteFile(path, data, 0o644); err != nil {
			return f.fail(msgResultFailed)
		}
	}
	return ExitDone
}

func (f *front) commands(m *Manifest, inv *Invocation) int {
	asJSON, uerr := inv.Bool("json")
	if uerr != nil {
		return f.refuse(uerr)
	}
	if code := f.answer(inv, Embedded()); code != ExitDone {
		return code
	}
	stdout, _ := f.streams()
	if asJSON {
		_, _ = stdout.Write(Embedded())
		return ExitDone
	}
	usageText := ""
	if f.env.Usage != nil {
		usageText = f.env.Usage()
	}
	_, _ = io.WriteString(stdout, ReadableList(m, usageText))
	return ExitDone
}

func (f *front) version(m *Manifest, inv *Invocation) int {
	asJSON, uerr := inv.Bool("json")
	if uerr != nil {
		return f.refuse(uerr)
	}
	data := VersionJSON(f.env.Version, m.CLIVersion)
	if code := f.answer(inv, data); code != ExitDone {
		return code
	}
	stdout, _ := f.streams()
	if asJSON {
		_, _ = stdout.Write(data)
		return ExitDone
	}
	_, _ = io.WriteString(stdout, VersionLine(f.env.Version))
	return ExitDone
}

// ReadableList is the output of `commands` without --json: the flag usage of the conversion parser when there is one
// (exactly the bytes `-h` prints, assumption X1), then one block per verb with its summary and its aliases.
func ReadableList(m *Manifest, usageText string) string {
	var b strings.Builder
	if usageText != "" {
		b.WriteString(usageText)
		if !strings.HasSuffix(usageText, "\n") {
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
	fmt.Fprintf(&b, "Commands of %s (command set %s):\n", m.Product, m.CLIVersion)
	width := 0
	for _, c := range m.Commands {
		if len(c.Name) > width {
			width = len(c.Name)
		}
	}
	for _, c := range m.Commands {
		fmt.Fprintf(&b, "  %-*s  %s\n", width, c.Name, c.Summary)
		var al []string
		for _, a := range c.Aliases {
			if a.Spelling != "" {
				al = append(al, a.Spelling)
			}
		}
		if len(al) > 0 {
			fmt.Fprintf(&b, "  %-*s  aliases: %s\n", width, "", strings.Join(al, " "))
		}
		if c.DefaultFor != "" {
			fmt.Fprintf(&b, "  %-*s  the call with no verb word (a path and flags) is this command\n", width, "")
		}
	}
	b.WriteString("\n\"commands --json\" prints the manifest as data.\n")
	return b.String()
}
