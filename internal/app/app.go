package app

import (
	"fmt"
	"os"
	"strings"

	"doc-html-translate/internal/config"
	"doc-html-translate/internal/i18n"
	"doc-html-translate/internal/logging"
	"doc-html-translate/internal/ocr"
	"doc-html-translate/internal/pipeline"
	"doc-html-translate/internal/syslocale"
	"doc-html-translate/internal/windowsreg"
)

type App struct {
	cfg config.Config
}

func New(cfg config.Config) App {
	// The only place the interface language is resolved. Everything downstream - this package's
	// console output and the page chrome in internal/htmlgen - reads i18n.Language(), so a run
	// speaks one language from its first line to the generated navigation bar.
	i18n.SetLanguage(i18n.Resolve(cfg.UILang, "", syslocale.Lang()))
	return App{cfg: cfg}
}

// Run executes the application logic. Returns exit code and error.
func (a App) Run() (int, error) {
	// Explicit opt-in: become the default handler for all supported types, and make
	// sure the non-destructive right-click entry exists too.
	if a.cfg.Register {
		registered, err := windowsreg.RegisterHandler()
		if err != nil {
			return 1, err
		}
		_, _ = windowsreg.RegisterContextMenu()
		printSplash()
		printDefaultHandlerResult(registered)
		printPressEnterAndPause()
		return 0, nil
	}

	// First run (no args): register the non-destructive right-click "Convert to HTML"
	// entry + "Open with" for every supported type, then OFFER to become the default
	// handler. It never grabs defaults silently.
	if a.cfg.FirstRun {
		_, _ = windowsreg.RegisterOpenWith()
		ctx, _ := windowsreg.RegisterContextMenu()
		printSplash()
		printFirstRunRegistered(ctx)
		if promptSetDefault() {
			registered, err := windowsreg.RegisterHandler()
			if err != nil {
				fmt.Fprintln(os.Stderr, "Could not set as default handler:", err)
			} else {
				printDefaultHandlerResult(registered)
			}
		}
		printPressEnterAndPause()
		return 0, nil
	}

	// Non-destructive: add the app to the "Open with" list and the right-click menu
	// without becoming the default handler. Scriptable; prints its result and exits.
	if a.cfg.RegisterOpenWith {
		advertised, err := windowsreg.RegisterOpenWith()
		if err != nil {
			return 1, err
		}
		_, _ = windowsreg.RegisterContextMenu()
		fmt.Println(`Added to the Windows "Open with" list and right-click menu for:`)
		for _, ext := range advertised {
			fmt.Printf("  * %s\n", ext)
		}
		return 0, nil
	}

	// Release the default-handler association (leaves the right-click entry + "Open with").
	if a.cfg.Unregister {
		released, err := windowsreg.Unregister()
		if err != nil {
			return 1, err
		}
		printUnregistered(released)
		return 0, nil
	}

	// OCR language management commands: run and exit without a document.
	if a.cfg.OCRList {
		printOCRLangs()
		return 0, nil
	}
	if a.cfg.OCRDownload != "" {
		fmt.Printf("Downloading OCR language %q (%s)..\n", a.cfg.OCRDownload, ocr.LangName(a.cfg.OCRDownload))
		if err := ocr.Download(a.cfg.OCRDownload); err != nil {
			return 1, err
		}
		fmt.Printf("Installed into %s\n", ocr.DataDir())
		return 0, nil
	}

	runner := pipeline.NewRunner(a.cfg)
	return runner.Run()
}

// printOCRLangs lists which OCR languages are installed and which can be downloaded.
func printOCRLangs() {
	installed := map[string]bool{}
	for _, c := range ocr.Installed() {
		installed[c] = true
	}
	fmt.Println("OCR languages (tessdata:", ocr.DataDir()+")")
	for _, l := range ocr.Available {
		mark := "  available - download with: -ocr-download " + l.Code
		if installed[l.Code] {
			mark = "  installed"
		}
		fmt.Printf("  %-9s %-22s%s\n", l.Code, l.Name, mark)
	}
}

// printDefaultHandlerResult prints the "set as default handler" confirmation block.
func printDefaultHandlerResult(exts []string) {
	fmt.Println(i18n.S("  Windows registration: DONE"))
	fmt.Println(i18n.S("  Program set as default handler for:"))
	for _, ext := range exts {
		fmt.Printf("    * %s\n", ext)
	}
	fmt.Println()
	fmt.Println(i18n.S("  Double-clicking a file will now open it with this program."))
}

// printFirstRunRegistered notes the non-destructive right-click entries added on first
// run, and makes clear the default handler was left untouched.
func printFirstRunRegistered(exts []string) {
	if len(exts) == 0 {
		return
	}
	fmt.Println(i18n.S(`  Added a right-click "Convert to HTML" entry and "Open with" for:`))
	for _, ext := range exts {
		fmt.Printf("    * %s\n", ext)
	}
	fmt.Println()
	fmt.Println(i18n.S("  Your default handlers were NOT changed - association is optional."))
	fmt.Println()
}

// printUnregistered reports the result of releasing the default-handler association.
func printUnregistered(released []string) {
	if len(released) == 0 {
		fmt.Println(i18n.S("Nothing to remove: the program was not the default handler."))
		return
	}
	fmt.Println(i18n.S("Default-handler association removed for:"))
	for _, ext := range released {
		fmt.Printf("  * %s\n", ext)
	}
}

// promptSetDefault asks whether to become the default handler and returns the answer.
// Anything but an explicit yes is treated as no, so pressing Enter declines. Accepted are
// "y"/"yes" plus the affirmative of the interface language, because someone reading the prompt
// in Bengali will answer in Bengali.
func promptSetDefault() bool {
	fmt.Print(i18n.S("  Make DOC-HTML-TRANSLATE the default handler for these file types? [y/N]: "))
	var answer string
	_, _ = fmt.Scanln(&answer)
	answer = strings.ToLower(strings.TrimSpace(answer))
	switch answer {
	case "y", "yes":
		return true
	}
	return answer == i18n.S("y") || answer == i18n.S("yes")
}

// printPressEnterAndPause prints the closing rule and keeps the console open until Enter.
// Piped or redirected there is no window to hold open, so the invitation would be a lie and
// the pause a hang: print the rule and return.
func printPressEnterAndPause() {
	line := strings.Repeat("=", 62)
	fmt.Println(line)
	fmt.Println()
	if !logging.StdoutIsTerminal() {
		return
	}
	fmt.Println(i18n.S("  Press Enter to close.. (we both know you'll close the window anyway)"))
	_, _ = fmt.Scanln() // pause - keep console open until user presses Enter
}
