package app

import (
	"fmt"
	"strings"

	"doc-html-translate/internal/config"
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
	if a.cfg.Register {
		registered, err := windowsreg.RegisterHandler()
		if err != nil {
			return 1, err
		}
		printSplash(registered)
		_, _ = fmt.Scanln() // pause — keep console open until user presses Enter
		return 0, nil
	}

	// Non-destructive: add the app to the "Open with" list without becoming the
	// default handler. Scriptable; prints its result and exits without pausing.
	if a.cfg.RegisterOpenWith {
		advertised, err := windowsreg.RegisterOpenWith()
		if err != nil {
			return 1, err
		}
		fmt.Println(`Added to the Windows "Open with" list for:`)
		for _, ext := range advertised {
			fmt.Printf("  * %s\n", ext)
		}
		return 0, nil
	}

	// OCR language management commands: run and exit without a document.
	if a.cfg.OCRList {
		printOCRLangs()
		return 0, nil
	}
	if a.cfg.OCRDownload != "" {
		fmt.Printf("Downloading OCR language %q (%s)...\n", a.cfg.OCRDownload, ocr.LangName(a.cfg.OCRDownload))
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

// printSplash prints the welcome screen shown on first launch (no-args mode).
func printSplash(registeredExts []string) {
	line := strings.Repeat("=", 62)
	if syslocale.IsRussian() {
		printSplashRU(line, registeredExts)
	} else {
		printSplashEN(line, registeredExts)
	}
}

func printSplashEN(line string, registeredExts []string) {
	fmt.Println(line)
	fmt.Println("  DOC-HTML-TRANSLATE")
	fmt.Println("  Document converter to HTML, with translation for the rest of us")
	fmt.Println(line)
	fmt.Println()
	fmt.Println("  Converts documents to HTML and opens the result")
	fmt.Println("  in your default browser. No ceremony required.")
	fmt.Println()
	fmt.Println("  Features:")
	fmt.Println("    - Convert EPUB, PDF, TXT, Markdown, FB2, RTF, HTML, MOBI, AZW3 to readable HTML")
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

	if len(registeredExts) > 0 {
		fmt.Println("  Windows registration: DONE")
		fmt.Println("  Program set as default handler for:")
		for _, ext := range registeredExts {
			fmt.Printf("    * %s\n", ext)
		}
		fmt.Println()
		fmt.Println("  Double-clicking a file will now open it with this program.")
		fmt.Println()
		fmt.Println("  Note: if Windows does not apply the setting,")
		fmt.Println(`  select the file → "Open with" → "Always".`)
	}

	fmt.Println(line)
	fmt.Println()
	fmt.Println("  Press Enter to close.. (we both know you'll close the window anyway)")
}

func printSplashRU(line string, registeredExts []string) {
	fmt.Println(line)
	fmt.Println("  DOC-HTML-TRANSLATE")
	fmt.Println("  Конвертер документов в HTML с переводом - для тех, кто не полиглот")
	fmt.Println(line)
	fmt.Println()
	fmt.Println("  Программа преобразует документы в HTML и открывает")
	fmt.Println("  результат в браузере по умолчанию. Без лишних церемоний.")
	fmt.Println()
	fmt.Println("  Возможности:")
	fmt.Println("    - Конвертация EPUB, PDF, TXT, Markdown, FB2, RTF, HTML, MOBI, AZW3 в читаемый HTML")
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

	if len(registeredExts) > 0 {
		fmt.Println("  Регистрация в Windows: ВЫПОЛНЕНА")
		fmt.Println("  Программа назначена обработчиком по умолчанию для:")
		for _, ext := range registeredExts {
			fmt.Printf("    * %s\n", ext)
		}
		fmt.Println()
		fmt.Println("  Теперь двойной клик на файле открывает эту программу.")
		fmt.Println()
		fmt.Println("  Примечание: если Windows не применяет настройку,")
		fmt.Println(`  выберите файл → "Открыть с помощью" → "Всегда".`)
	}

	fmt.Println(line)
	fmt.Println()
	fmt.Println("  Нажмите Enter для закрытия.. (хотя вы всё равно закроете окно крестиком)")
}
