package main

import (
	"fmt"
	"os"

	"doc-html-translate/internal/app"
	"doc-html-translate/internal/config"
	"doc-html-translate/internal/logging"
)

// Version is set at build time via -ldflags.
var Version = "dev"

func main() {
	cfg, err := config.ParseArgs(os.Args[1:])
	if err != nil {
		if err.Error() == "version" {
			fmt.Println("doc-html-translate " + Version)
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		waitOnError()
		os.Exit(1)
	}

	logging.AppVersion = Version
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
func waitOnError() {
	fmt.Fprintln(os.Stderr, "\nPress Enter to close...")
	fmt.Scanln()
}
