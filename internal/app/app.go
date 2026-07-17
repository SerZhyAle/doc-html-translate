package app

import (
	"fmt"
	"os"
	"strings"

	"doc-html-translate/internal/config"
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

// printSplash prints the informational welcome screen (help, features, usage, links).
// The registration result and the "Press Enter" pause are printed by the caller so the
// first-run flow can slot its default-handler opt-in prompt in between.
func printSplash() {
	line := strings.Repeat("=", 62)
	if syslocale.IsRussian() {
		printSplashRU(line)
	} else {
		printSplashEN(line)
	}
}

// printDefaultHandlerResult prints the "set as default handler" confirmation block.
func printDefaultHandlerResult(exts []string) {
	if syslocale.IsRussian() {
		fmt.Println("  Регистрация в Windows: ВЫПОЛНЕНА")
		fmt.Println("  Программа назначена обработчиком по умолчанию для:")
	} else {
		fmt.Println("  Windows registration: DONE")
		fmt.Println("  Program set as default handler for:")
	}
	for _, ext := range exts {
		fmt.Printf("    * %s\n", ext)
	}
	fmt.Println()
	if syslocale.IsRussian() {
		fmt.Println("  Теперь двойной клик на файле открывает эту программу.")
	} else {
		fmt.Println("  Double-clicking a file will now open it with this program.")
	}
}

// printFirstRunRegistered notes the non-destructive right-click entries added on first
// run, and makes clear the default handler was left untouched.
func printFirstRunRegistered(exts []string) {
	if len(exts) == 0 {
		return
	}
	if syslocale.IsRussian() {
		fmt.Println("  Добавлен пункт правого клика «Convert to HTML» и «Открыть с помощью» для:")
	} else {
		fmt.Println(`  Added a right-click "Convert to HTML" entry and "Open with" for:`)
	}
	for _, ext := range exts {
		fmt.Printf("    * %s\n", ext)
	}
	fmt.Println()
	if syslocale.IsRussian() {
		fmt.Println("  Ассоциация по умолчанию НЕ изменена - это по желанию.")
	} else {
		fmt.Println("  Your default handlers were NOT changed - association is optional.")
	}
	fmt.Println()
}

// printUnregistered reports the result of releasing the default-handler association.
func printUnregistered(released []string) {
	if len(released) == 0 {
		if syslocale.IsRussian() {
			fmt.Println("Нечего снимать: программа не была обработчиком по умолчанию.")
		} else {
			fmt.Println("Nothing to remove: the program was not the default handler.")
		}
		return
	}
	if syslocale.IsRussian() {
		fmt.Println("Ассоциация обработчика по умолчанию снята для:")
	} else {
		fmt.Println("Default-handler association removed for:")
	}
	for _, ext := range released {
		fmt.Printf("  * %s\n", ext)
	}
}

// promptSetDefault asks whether to become the default handler and returns the answer.
// Anything but an explicit yes (y/yes/д/да) is treated as no, so pressing Enter declines.
func promptSetDefault() bool {
	if syslocale.IsRussian() {
		fmt.Print("  Сделать DOC-HTML-TRANSLATE обработчиком по умолчанию для этих типов? [y/N]: ")
	} else {
		fmt.Print("  Make DOC-HTML-TRANSLATE the default handler for these file types? [y/N]: ")
	}
	var answer string
	_, _ = fmt.Scanln(&answer)
	switch strings.ToLower(strings.TrimSpace(answer)) {
	case "y", "yes", "д", "да":
		return true
	}
	return false
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
	if syslocale.IsRussian() {
		fmt.Println("  Нажмите Enter для закрытия.. (хотя вы всё равно закроете окно крестиком)")
	} else {
		fmt.Println("  Press Enter to close.. (we both know you'll close the window anyway)")
	}
	_, _ = fmt.Scanln() // pause - keep console open until user presses Enter
}

func printSplashEN(line string) {
	fmt.Println(line)
	fmt.Println("  DOC-HTML-TRANSLATE")
	fmt.Println("  Document converter to HTML, with translation for the rest of us")
	fmt.Println(line)
	fmt.Println()
	fmt.Println("  Converts documents to HTML and opens the result")
	fmt.Println("  in your default browser. No ceremony required.")
	fmt.Println()
	fmt.Println("  Features:")
	fmt.Println("    - Convert EPUB, PDF, TXT, Markdown, FB2, RTF, HTML, MOBI, AZW3, CBZ/CBR/CB7/CBT comics to readable HTML")
	fmt.Println("    - Navigation between pages/chapters")
	fmt.Println("    - Ctrl+scroll zoom with persistence")
	fmt.Println("    - Text translation via Google Translate API")
	fmt.Println("    - Re-running opens the already-generated HTML")
	fmt.Println()
	fmt.Println("  Usage:")
	fmt.Println(`    doc-html-translate.exe "book.epub"`)
	fmt.Println(`    doc-html-translate.exe "report.pdf"`)
	fmt.Println(`    doc-html-translate.exe "notes.txt"`)
	fmt.Println(`    doc-html-translate.exe "readme.md"`)
	fmt.Println(`    doc-html-translate.exe "book.fb2"`)
	fmt.Println(`    doc-html-translate.exe "document.rtf"`)
	fmt.Println(`    doc-html-translate.exe "page.html"`)
	fmt.Println(`    doc-html-translate.exe "book.mobi"           # requires Calibre`)
	fmt.Println(`    doc-html-translate.exe "comic.cbz"           # comic pages, text recognized via OCR`)
	fmt.Println(`    doc-html-translate.exe "comic.cbr"           # CBR/CB7 require 7-Zip`)
	fmt.Println(`    doc-html-translate.exe "book.epub"        # default: convert + open, no translation engine`)
	fmt.Println(`    doc-html-translate.exe -notranslate "book.epub"  # explicit equivalent`)
	fmt.Println(`    doc-html-translate.exe -google "book.epub"`)
	fmt.Println(`    doc-html-translate.exe -ollama "book.epub"`)
	fmt.Println(`    doc-html-translate.exe -src en -dst de "book.epub"`)
	fmt.Println(`    doc-html-translate.exe -force "book.epub"`)
	fmt.Println()
	fmt.Println("  Flags:")
	fmt.Println("    -notranslate    Convert only, no translation")
	fmt.Println("    -google         Translate via Google Translate API (paid)")
	fmt.Println("    -ollama         Translate via local Ollama (free)")
	fmt.Println("    -ollama-model   Ollama model (default: gemma3:12b)")
	fmt.Println("    -toc-depth N    TOC nesting depth on index.html (0 = unlimited)")
	fmt.Println("    -force          Force regeneration")
	fmt.Println("    -src LANG       Source language (default: en)")
	fmt.Println("    -dst LANG       Target language (default: ru)")
	fmt.Println()
	fmt.Println("  Links:")
	fmt.Println("    Product site: https://serzhyale.github.io/doc-html-translate/")
	fmt.Println("    Feedback:     sza@ukr.net")
	fmt.Println()
	fmt.Println(line)
}

func printSplashRU(line string) {
	fmt.Println(line)
	fmt.Println("  DOC-HTML-TRANSLATE")
	fmt.Println("  Конвертер документов в HTML с переводом - для тех, кто не полиглот")
	fmt.Println(line)
	fmt.Println()
	fmt.Println("  Программа преобразует документы в HTML и открывает")
	fmt.Println("  результат в браузере по умолчанию. Без лишних церемоний.")
	fmt.Println()
	fmt.Println("  Возможности:")
	fmt.Println("    - Конвертация EPUB, PDF, TXT, Markdown, FB2, RTF, HTML, MOBI, AZW3, комиксов CBZ/CBR/CB7/CBT в читаемый HTML")
	fmt.Println("    - Навигация между страницами/главами")
	fmt.Println("    - Масштабирование Ctrl+колёсико с сохранением")
	fmt.Println("    - Перевод текста через Google Translate API")
	fmt.Println("    - Повторный запуск открывает уже готовый HTML")
	fmt.Println()
	fmt.Println("  Использование:")
	fmt.Println(`    doc-html-translate.exe "book.epub"`)
	fmt.Println(`    doc-html-translate.exe "report.pdf"`)
	fmt.Println(`    doc-html-translate.exe "notes.txt"`)
	fmt.Println(`    doc-html-translate.exe "readme.md"`)
	fmt.Println(`    doc-html-translate.exe "book.fb2"`)
	fmt.Println(`    doc-html-translate.exe "document.rtf"`)
	fmt.Println(`    doc-html-translate.exe "page.html"`)
	fmt.Println(`    doc-html-translate.exe "book.mobi"           # требуется Calibre`)
	fmt.Println(`    doc-html-translate.exe "comic.cbz"           # страницы комикса, текст распознаётся через OCR`)
	fmt.Println(`    doc-html-translate.exe "comic.cbr"           # CBR/CB7 требуют 7-Zip`)
	fmt.Println(`    doc-html-translate.exe "book.epub"        # default: convert + open, no translation engine`)
	fmt.Println(`    doc-html-translate.exe -notranslate "book.epub"  # explicit equivalent`)
	fmt.Println(`    doc-html-translate.exe -google "book.epub"`)
	fmt.Println(`    doc-html-translate.exe -ollama "book.epub"`)
	fmt.Println(`    doc-html-translate.exe -src en -dst de "book.epub"`)
	fmt.Println(`    doc-html-translate.exe -force "book.epub"`)
	fmt.Println()
	fmt.Println("  Флаги:")
	fmt.Println("    -notranslate    Только конвертация, без перевода")
	fmt.Println("    -google         Перевести через Google Translate API (платно)")
	fmt.Println("    -ollama         Перевести через локальный Ollama (бесплатно)")
	fmt.Println("    -ollama-model   Модель Ollama (по умолчанию: gemma3:12b)")
	fmt.Println("    -toc-depth N    Глубина оглавления на index.html (0 = без ограничений)")
	fmt.Println("    -force          Принудительная перегенерация")
	fmt.Println("    -src LANG       Исходный язык (по умолчанию: en)")
	fmt.Println("    -dst LANG       Целевой язык  (по умолчанию: ru)")
	fmt.Println()
	fmt.Println("  Ссылки:")
	fmt.Println("    Сайт продукта:  https://serzhyale.github.io/doc-html-translate/")
	fmt.Println("    Обратная связь: sza@ukr.net")
	fmt.Println()
	fmt.Println(line)
}
