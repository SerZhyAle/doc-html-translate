package main

import (
	"errors"
	"fmt"
	"os"
	"time"

	"doc-html-translate/internal/app"
	"doc-html-translate/internal/config"
	"doc-html-translate/internal/logging"
	"doc-html-translate/internal/report"
)

// Version is set at build time via -ldflags.
var Version = "dev"

func main() {
	cfg, err := config.ParseArgs(os.Args[1:])
	if err != nil {
		// -h and -version are requests, not failures: they print what was asked for and exit 0,
		// so `-h | more` and `-version` are usable from a script. Everything else is an error.
		if errors.Is(err, config.ErrHelp) {
			os.Exit(0)
		}
		if err.Error() == "version" {
			fmt.Println("doc-html-translate " + Version)
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		waitOnError()
		os.Exit(1)
	}

	logging.AppVersion = Version

	// Keep this run's log on disk so a failure can still be reported an hour later. Every
	// error here is swallowed on purpose: a store that cannot be written must leave the run
	// exactly as it was, with nothing said about it.
	_, _ = report.Trim()
	if err := os.MkdirAll(report.LogsDir(), 0o755); err == nil {
		if f, err := os.OpenFile(report.RunLogPath(time.Now()), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600); err == nil {
			logging.StartRunLog(f)
			defer func() {
				logging.StopRunLog()
				_ = f.Close()
			}()
		}
	}

	logging.Printf("doc-html-translate %s\n", Version)
	application := app.New(cfg)
	exitCode, err := application.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		waitOnError()
		os.Exit(exitCode)
	}

	os.Exit(exitCode)
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
