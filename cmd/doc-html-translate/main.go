package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"doc-html-translate/internal/app"
	"doc-html-translate/internal/commands"
	"doc-html-translate/internal/config"
	"doc-html-translate/internal/logging"
	"doc-html-translate/internal/report"
)

// Version is set at build time via -ldflags.
var Version = "dev"

func main() {
	// The command front reads the line first (APP-COMMANDS rules 1, 3 and 17): `commands`, `version`, a refusal. It
	// touches no window, setting, worker or log. Anything it does not own - a document and its flags, -h, -version,
	// the no-argument first run - goes to the conversion parser below exactly as before.
	if handled, code := commands.Front(os.Args[1:], commands.Env{
		Version: Version,
		Usage:   config.UsageText,
		// `convert` runs the very conversion the bare call runs; it never pauses for Enter after a failure, because a
		// command never waits on a person it did not ask for (APP-COMMANDS rule 4).
		Convert: func(cfg config.Config) int { return run(cfg, false) },
		// The maintenance verbs register this very executable and write its own files: only the console converter
		// serves them (ticket E0147); doc-html-ui.exe answers with the sentence that this program does.
		Maintain: true,
	}); handled {
		os.Exit(code)
	}

	cfg, err := config.ParseArgs(os.Args[1:])
	if err != nil {
		// -h and -version are requests, not failures: they print what was asked for and exit 0,
		// so `-h | more` and `-version` are usable from a script. Everything else is an error.
		if errors.Is(err, config.ErrHelp) {
			os.Exit(0)
		}
		if errors.Is(err, config.ErrVersion) {
			fmt.Print(commands.VersionLine(Version))
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		waitOnError()
		os.Exit(1)
	}

	// os.Exit skips deferred calls, so the run's log is closed inside run, not around it.
	os.Exit(run(cfg, true))
}

// run converts and returns the exit code. The run log's tail is written by its deferred Finish
// (DIAGNOSTIC-REPORT 4: the middle of an oversized log is dropped, the end is kept), so every way
// out of this function - a return, a panic of this goroutine - leaves the log complete.
//
// pause is whether a failure at a console waits for Enter: the bare call does, so a double-clicked
// window can be read; the `convert` command never does (APP-COMMANDS rule 4).
func run(cfg config.Config, pause bool) int {
	logging.AppVersion = Version

	// Keep this run's log on disk so a failure can still be reported an hour later. Every
	// error here is swallowed on purpose: a store that cannot be written must leave the run
	// exactly as it was, with nothing said about it.
	_, _ = report.Trim()
	finishLog := func() {}
	if err := os.MkdirAll(report.LogsDir(), 0o755); err == nil {
		if f, err := os.OpenFile(report.RunLogPath(time.Now(), os.Getpid()), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600); err == nil {
			capped := report.CapRunLog(f)
			logging.SetRunLogFilter(report.Redact)
			logging.StartRunLog(capped)
			finishLog = capped.Finish
			defer func() {
				logging.StopRunLog()
				capped.Finish()
				_ = f.Close()
			}()
		}
	}

	if cfg.InputFile != "" {
		report.HideDocument(cfg.InputFile)
	}
	logging.Printf("doc-html-translate %s\n", Version)
	// APP-COMMANDS rule 16: an accepted command is logged - verb, origin, process id, outcome - with no path
	// and no argument. The bare call with a document is the same command, so it writes the same two lines;
	// -register, -report and the other document-less flags are not conversions and write neither.
	converting := cfg.InputFile != ""
	if converting {
		logging.RunLogf("Command: convert (origin cli, pid %d)\n", os.Getpid())
	}
	application := app.New(cfg)
	exitCode, err := application.Run()
	if converting {
		logging.RunLogf("Command convert ended: exit code %d\n", exitCode)
	}
	if err != nil {
		reportFailure(os.Stderr, exitCode, err)
		// Whoever pressed Ctrl+C is at the console and wants it back, not a pause. The log is
		// finished first: a console window closed at the prompt ends the process on the spot.
		if exitCode != app.ExitInterrupted {
			finishLog()
			if pause {
				waitOnError()
			}
		}
	}
	return exitCode
}

// reportFailure prints the error that ended the run and records it in the run log, which the
// console text never reaches: without it a -report bundle shows a run that simply stops.
func reportFailure(stderr io.Writer, exitCode int, err error) {
	fmt.Fprintf(stderr, "Error: %v\n", err)
	logging.RunLogf("Run failed (exit code %d): %v\n", exitCode, err)
}

// waitOnError keeps the console window open so the user can read the error message (R9).
// A double-clicked exe needs it; a piped or redirected run has no window to hold open and
// nobody to press a key, so pausing there hangs the caller until it is killed.
func waitOnError() {
	if !logging.StdoutIsTerminal() {
		return
	}
	fmt.Fprintln(os.Stderr, "\nPress Enter to close..")
	_, _ = fmt.Scanln()
}
