package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"

	"doc-html-translate/internal/config"
	"doc-html-translate/internal/i18n"
	"doc-html-translate/internal/logging"
	"doc-html-translate/internal/ocr"
	"doc-html-translate/internal/pipeline"
	"doc-html-translate/internal/report"
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
		reg, err := windowsreg.RegisterHandler()
		if err != nil {
			return 1, err
		}
		_, menuErr := windowsreg.RegisterContextMenu()
		printSplash()
		printIntegrationErrors(nil, menuErr)
		printDefaultHandlerResult(reg)
		offerDefaultAppsSettings(reg)
		printPressEnterAndPause()
		return 0, nil
	}

	// First run (no args): register the non-destructive right-click "Convert to HTML"
	// entry + "Open with" for every supported type, then OFFER to become the default
	// handler. It never grabs defaults silently.
	if a.cfg.FirstRun {
		_, openWithErr := windowsreg.RegisterOpenWith()
		ctx, menuErr := windowsreg.RegisterContextMenu()
		printSplash()
		printFirstRunRegistered(ctx)
		printIntegrationErrors(openWithErr, menuErr)
		if promptSetDefault() {
			reg, err := windowsreg.RegisterHandler()
			if err != nil {
				fmt.Fprintln(os.Stderr, i18n.S("  Could not set as default handler: %v", err))
			}
			printDefaultHandlerResult(reg)
			offerDefaultAppsSettings(reg)
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
		_, menuErr := windowsreg.RegisterContextMenu()
		printIntegrationErrors(nil, menuErr)
		fmt.Println(`Added to the Windows "Open with" list and right-click menu for:`)
		for _, ext := range advertised {
			fmt.Printf("  * %s\n", ext)
		}
		return 0, nil
	}

	// Release the default-handler association (leaves the right-click entry + "Open with").
	if a.cfg.Unregister {
		released, err := windowsreg.Unregister()
		printUnregistered(released)
		printStillDefault(windowsreg.HandlerStatus().Default)
		if err != nil {
			return 1, err
		}
		return 0, nil
	}

	// OCR language management commands: run and exit without a document.
	if a.cfg.OCRList {
		printOCRLangs()
		return 0, nil
	}
	// Pack the recent run logs for the author and exit. The CLI prints where the archive is and
	// stops there - opening a mail program is the GUI's job, not a console tool's.
	if a.cfg.Report {
		path, dropped, err := report.Build(report.BuildOptions{
			AppVersion: logging.AppVersion,
			Packaged:   false,
			At:         time.Now(),
		})
		if err != nil {
			return 1, err
		}
		fmt.Println(i18n.S("Report written to:"))
		// The path itself is printed raw so it can be copied out of the console verbatim.
		fmt.Println(path)
		if dropped > 0 {
			fmt.Println(i18n.S("%d older logs did not fit and were left out.", dropped))
		}
		return 0, nil
	}

	if a.cfg.OCRDownload != "" {
		if err := ocr.CheckLang(a.cfg.OCRDownload); err != nil {
			return 1, err
		}
		fmt.Printf("Downloading OCR language %q (%s)..\n", a.cfg.OCRDownload, ocr.LangName(a.cfg.OCRDownload))
		if err := ocr.Download(a.cfg.OCRDownload); err != nil {
			return 1, err
		}
		fmt.Printf("Installed into %s\n", ocr.UserDataDir())
		return 0, nil
	}

	for _, n := range a.cfg.Notices {
		logging.Println(n)
	}

	// Ctrl+C cancels the run cooperatively: the pipeline stops between pages and returns
	// ExitInterrupted instead of the process exiting in the middle of a page write. Once the
	// first interrupt is seen the default handling is back, so a second Ctrl+C still kills a run
	// that is stuck inside one long step.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	context.AfterFunc(ctx, stop)
	return pipeline.NewRunner(a.cfg).RunContext(ctx)
}

// ExitInterrupted is the exit code of a run stopped by Ctrl+C.
const ExitInterrupted = pipeline.ExitInterrupted

// printOCRLangs lists which OCR languages are installed and which can be downloaded.
func printOCRLangs() {
	installed := map[string]bool{}
	for _, c := range ocr.Installed() {
		installed[c] = true
	}
	fmt.Println("OCR languages (tessdata:", strings.Join(ocr.DataDirs(), "; ")+")")
	for _, l := range ocr.Available {
		mark := "  available - download with: -ocr-download " + l.Code
		if installed[l.Code] {
			mark = "  installed"
		}
		fmt.Printf("  %-9s %-22s%s\n", l.Code, l.Name, mark)
	}
}

// printDefaultHandlerResult prints what "set as default handler" actually achieved. Writing
// the class key is not enough where the user has chosen an app in Windows itself, so each
// outcome gets its own list instead of one blanket "DONE".
func printDefaultHandlerResult(reg windowsreg.Registration) {
	if reg.Complete() {
		fmt.Println(i18n.S("  Windows registration: DONE"))
	} else {
		fmt.Println(i18n.S("  Windows registration: INCOMPLETE"))
	}
	printExtList(i18n.S("  Program set as default handler for:"), reg.Default)
	printExtList(i18n.S("  Registered, but Windows keeps the app you chose earlier for:"), reg.Blocked)
	printExtList(i18n.S("  Registered, but the handler Windows uses could not be read for:"), reg.Unknown)
	printExtList(i18n.S("  Could not register:"), reg.Failed)
	fmt.Println()
	if reg.Complete() {
		fmt.Println(i18n.S("  Double-clicking a file will now open it with this program."))
		return
	}
	if len(reg.Blocked)+len(reg.Unknown) > 0 {
		printDefaultAppsHowTo()
	}
}

func printExtList(title string, exts []string) {
	if len(exts) == 0 {
		return
	}
	fmt.Println(title)
	for _, ext := range exts {
		fmt.Printf("    * %s\n", ext)
	}
}

// printDefaultAppsHowTo is the manual step Windows leaves to the user: no program may
// overwrite the user's own choice.
func printDefaultAppsHowTo() {
	fmt.Println(i18n.S("  Only you can change this in Windows: open Settings > Apps > Default apps,"))
	fmt.Println(i18n.S("  search for the file type (for example .epub) and pick DOC-HTML-TRANSLATE."))
	fmt.Println()
}

// offerDefaultAppsSettings opens Default apps on an explicit yes. It asks only at a console:
// the GUI runs -register with no one to answer and offers its own button instead.
func offerDefaultAppsSettings(reg windowsreg.Registration) {
	if len(reg.Blocked)+len(reg.Unknown) == 0 || !logging.StdoutIsTerminal() {
		return
	}
	if !askYes(i18n.S("  Open Default apps in Settings now? [y/N]: ")) {
		return
	}
	if err := windowsreg.OpenDefaultAppsSettings(); err != nil {
		fmt.Fprintln(os.Stderr, i18n.S("  Could not open Settings: %v", err))
	}
}

// printIntegrationErrors reports the non-destructive integrations that could not be written,
// which used to vanish and leave the user looking for a menu entry that was never added.
func printIntegrationErrors(openWithErr, menuErr error) {
	if openWithErr != nil {
		fmt.Fprintln(os.Stderr, i18n.S(`  Could not add the "Open with" entry: %v`, openWithErr))
	}
	if menuErr != nil {
		fmt.Fprintln(os.Stderr, i18n.S(`  Could not add the right-click "Convert to HTML" entry: %v`, menuErr))
	}
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
	fmt.Println(i18n.S("Default-handler association removed, and the previous handler restored where one was saved, for:"))
	for _, ext := range released {
		fmt.Printf("  * %s\n", ext)
	}
}

// printStillDefault names the types Windows keeps opening with the app after unregistering:
// the user chose it in Windows itself, and that choice is theirs to undo.
func printStillDefault(exts []string) {
	if len(exts) == 0 {
		return
	}
	fmt.Println(i18n.S("Windows still opens these with this program, because it was chosen in Windows itself:"))
	for _, ext := range exts {
		fmt.Printf("  * %s\n", ext)
	}
	fmt.Println(i18n.S("Change them in Settings > Apps > Default apps."))
}

// promptSetDefault asks whether to become the default handler and returns the answer.
func promptSetDefault() bool {
	return askYes(i18n.S("  Make DOC-HTML-TRANSLATE the default handler for these file types? [y/N]: "))
}

// askYes prints prompt and reads one answer. Anything but an explicit yes is treated as no, so
// pressing Enter declines. Accepted are "y"/"yes" plus the affirmative of the interface
// language, because someone reading the prompt in Bengali will answer in Bengali.
func askYes(prompt string) bool {
	fmt.Print(prompt)
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
