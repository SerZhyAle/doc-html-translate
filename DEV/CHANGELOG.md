# DEV CHANGELOG

## Project Links
- GitHub Pages: https://serzhyale.github.io/doc-html-translate/
- GitHub Repository: https://github.com/SerZhyAle/doc-html-translate

| Timestamp | Path | Target | Description |
|---|---|---|---|
| 2026-03-04 01:30:27 | DEV/research/epub2html_research_ru.md | research | Добавлен подробный research по задаче EPUB->HTML+Translate: варианты решения, риски, задачи разработчика |
| 2026-03-04 01:32:14 | DEV/plan.md | planning | Создан стратегический план разработки: 4 фазы, 11 блоков задач, Q1-Q7 вопросы пользователю |
| 2026-03-04 01:33:49 | scripts/generate-icon.ps1 | build-assets | Добавлен генератор примитивной иконки EPUB2HTML в формате .ico |
| 2026-03-04 01:33:49 | scripts/build.ps1 | build | Сборка дополнена генерацией и копированием иконки epub2html.ico рядом с exe |
| 2026-03-04 01:33:49 | internal/windowsreg/register_windows.go | windows-registry | Регистрация .epub расширена: ProgID, open command и DefaultIcon с использованием epub2html.ico |
| 2026-03-04 01:35:37 | scripts/commit_after_build.ps1 | git-workflow | Исправлен commit workflow: COMMIT_LOG добавляется в тот же коммит через amend |
| 2026-03-04 01:35:37 | DEV/plan.md | planning | Обновлен live-прогресс: baseline commit выполнен (f3fbc43) |
| 2026-03-04 01:37:57 | scripts/build.ps1 | build | Сборка переведена на temp/: output temp/build, icon temp/assets, build.log в temp/logs |
| 2026-03-04 01:37:57 | scripts/test.ps1 | test | Тестовый скрипт пишет лог в temp/logs/test.log |
| 2026-03-04 01:37:57 | scripts/lint.ps1 | lint | Линтер пишет лог в temp/logs/lint.log |
| 2026-03-04 01:37:57 | scripts/typo.ps1 | typo | Проверка typos пишет лог в temp/logs/typo.log |
| 2026-03-04 01:37:57 | scripts/check.ps1 | check | Агрегирующий скрипт сохраняет логи шагов в temp/logs |
| 2026-03-04 01:37:58 | scripts/generate-icon.ps1 | build-assets | Default output иконки изменен на temp/assets/epub2html.ico |
| 2026-03-04 01:37:58 | configs/.typos.toml | config | Исключения typos обновлены: temp вместо build |
| 2026-03-04 01:37:58 | .gitignore | git | Добавлено правило temp/ для исключения промежуточных артефактов |
| 2026-03-04 01:37:58 | README.md | docs | Документация обновлена: все артефакты/логи в temp |
| 2026-03-04 01:37:58 | assets/epub2html.ico | cleanup | Удален артефакт иконки из корня проекта |
| 2026-03-04 01:38:10 | DEV/plan.md | planning | Live-прогресс обновлен: зафиксирована temp-only политика для артефактов и логов |
| 2026-03-04 01:38:47 | .gitignore | git | Добавлено исключение build/ для предотвращения попадания root-артефактов в git |
| 2026-03-04 01:38:47 | build/epub2html.ico | cleanup | Удален случайно отслеживаемый build-артефакт, политика temp-only восстановлена |
| 2026-03-04 01:39:40 | DEV/README.md | docs | README перенесен из корня в DEV согласно политике хранения документации |
| 2026-03-04 01:39:40 | DEV/universal_copilot_instructions.md | docs | Файл universal_copilot_instructions перенесен из корня в DEV |
| 2026-03-04 01:39:40 | DEV/plan.md | planning | План обновлен: документация хранится в DEV, путь K3 обновлен на DEV/README.md |
| 2026-03-04 01:40:22 | temp/docs-archive/task_2026-03-04.md | docs-archive | Устаревший DEV/task.md перенесен в temp/docs-archive |
| 2026-03-04 01:40:22 | DEV/README.md | docs | Добавлен раздел жизненного цикла документации и правила архивирования |
| 2026-03-04 01:40:22 | DEV/plan.md | planning | В план добавлено правило переноса устаревшей документации в temp/docs-archive |
| 2026-03-04 01:41:13 | DEV/questionnaire_for_user.txt | docs | Создан текстовый опросник для пользователя с 15 вопросами по уточнению требований |
| 2026-03-04 01:41:46 | DEV/plan.md | planning | Зафиксированы решения: локальный непубличный git, приоритет истории откатов, допустимость хранения чувствительных данных в v1 |
| 2026-03-04 02:06:59 | internal/htmlgen/navbar.go | htmlgen | Added sticky navigation bar (prev/TOC/next) injected into all content HTML pages |
| 2026-03-04 02:07:04 | internal/htmlgen/navbar_test.go | htmlgen | Tests for navbar: relativePath, buildNavBarHTML, InjectNavBars (4 tests) |
| 2026-03-04 02:07:08 | internal/pipeline/pipeline.go | pipeline | Wire InjectNavBars call between index generation and translation steps |
| 2026-03-04 02:08:37 | internal/htmlgen/navbar.go | htmlgen | Adjusted navbar layout: compact adjacent buttons aligned to top-right |
| 2026-03-04 02:16:56 | internal/translator/translator.go | translator | Configured hardcoded Google Translate API key in HARDCODED_API_KEY |
| 2026-03-04 02:22:44 | internal/app/app.go | app | Clarified -register output: HKCU registration done but Windows UserChoice may still override default app |
| 2026-03-04 02:24:48 | internal/config/flags.go | config | No-args launch now enters implicit register mode for first-click UX |
| 2026-03-04 02:24:48 | internal/config/flags_test.go | config | Updated tests for implicit register on empty args and missing-file validation |
| 2026-03-04 02:24:49 | DEV/README.md | docs | Documented first-run no-arg registration behavior |
| 2026-03-04 02:25:15 | internal/pipeline/pipeline.go | pipeline | Implemented -force mode: rebuild output instead of idempotent reopen |
| 2026-03-04 02:25:16 | DEV/README.md | docs | Documented -force flag and behavior in usage notes |
| 2026-03-04 02:36:57 | internal/htmlgen/navbar.go | htmlgen | Added zoom persistence across chapter navigation via injected JS (Ctrl+Wheel + z query/session sync) |
| 2026-03-04 20:07:16 | internal/pdf/extract.go | pdf | Create PDF text extraction package using ledongthuc/pdf. Per-page extraction with GetTextByRow, HTML page generation, epub.Book adapter. |
| 2026-03-04 20:07:23 | internal/pdf/extract_test.go | pdf | Add 8 unit tests for PDF extraction: helpers, valid PDF, empty PDF, invalid file, non-existent file. |
| 2026-03-04 20:07:31 | internal/pipeline/pipeline.go | pipeline | Add PDF format detection (.epub/.pdf by extension). New extractPDF branch. Rename outputDirForEpub->outputDirForFile. Add ExitParse alias. |
| 2026-03-04 20:07:39 | DEV/plan.md | docs | Add Phase 5: PDF support with blocks L, M, N. Architecture decisions R19-R24. |
| 2026-03-04 20:14:53 | go.mod | rename | Module renamed: epub2html -> doc-html-translate |
| 2026-03-04 20:15:01 | **/*.go,scripts/*,DEV/* | rename | Full project rename epub2html -> doc-html-translate: module, imports, binary, scripts, CSS prefix (dht-), ProgID, docs, icon |
| 2026-03-04 23:48:42 | internal/md/extract.go | Markdown extractor | Goldmark-based MD->HTML converter with heading-based pagination |
| 2026-03-04 23:48:44 | internal/fb2/extract.go | FB2 extractor | XML parser for FictionBook2 format with paragraph pagination |
| 2026-03-04 23:48:46 | internal/rtf/extract.go | RTF extractor | Lightweight RTF stripper with CP1251 hex escape decoding |
| 2026-03-04 23:48:48 | internal/htmlconv/extract.go | HTML/HTM extractor | HTML passthrough with body extraction and CSS wrapping |
| 2026-03-04 23:48:50 | internal/pipeline/pipeline.go | Pipeline | Added MD/FB2/RTF/HTML/HTM cases to format router |
| 2026-03-04 23:48:52 | internal/windowsreg/register_windows.go | Registration | Extended SupportedExtensions with .md .fb2 .rtf .html .htm |
| 2026-03-05 03:14:24 | internal/htmlgen/htmlgen.go | TOC | Fix double numbering: ol→ul (auto-numbers were redundant with explicit %d. in toc-label span) |
| 2026-03-05 03:14:24 | internal/pipeline/pipeline.go | Title translation | Translate book.Title via Google API at end of translateContent before GenerateIndex is called |
| 2026-03-05 23:31:07 | internal/htmlgen/navbar.go | navBarCSS | Added img { max-height: 100vh; width: auto; max-width: 100% } to limit image height to one screen |
| 2026-03-07 02:43:26 | internal/translator/ollama.go | ollama.go | Add translateSingle() for echo-back retries; reduce ollamaBatchSize 40->20; add ollamaMaxRetries=2 |
| 2026-03-07 02:43:26 | internal/htmlproc/htmlproc.go | htmlproc.go | ReplaceTexts: skip empty translations (translated=empty -> keep original); fix leading whitespace operator precedence bug |
| 2026-03-07 02:56:37 | internal/logging/log.go | logging | New package: timestamp-prefixed console output (Printf/Println/Errorf/Progress) |
| 2026-03-07 03:06:24 | internal/translator/ollama.go | OllamaClient | Add num_ctx=8192 option to all Ollama requests; concurrent batch goroutines with semaphore; thread-safe firstRequest via sync.Mutex; SetParallelism/SetNumCtx methods |
| 2026-03-07 03:06:24 | internal/config/flags.go | Config | Add OllamaParallel int and OllamaNumCtx int fields; -ollama-parallel and -ollama-ctx flags |
| 2026-03-07 03:06:24 | internal/pipeline/pipeline.go | UseOllama case | Wire SetParallelism + SetNumCtx from config into ollamaWorker |
| 2026-03-07 03:18:07 | internal/htmlsplit/split.go | htmlsplit | New package: SplitIfNeeded splits HTML spine items at paragraph boundaries to stay within maxChars |
| 2026-03-07 03:18:07 | internal/config/flags.go | Config.SplitSize | Add SplitSize int field; -split flag with auto-inject 5000 when used without value |
| 2026-03-07 03:18:07 | internal/pipeline/pipeline.go | Run() | Add split step after extraction using htmlsplit.SplitIfNeeded when SplitSize > 0 |
| 2026-03-07 03:22:49 | internal/config/flags.go | Config.OutputFolder | Add OutputFolder string field; -folder flag for custom output directory |
| 2026-03-07 03:22:49 | internal/pipeline/pipeline.go | outputDirFor() | Replace outputDirForFile with outputDirFor(path, folder); uses -folder when set |
| 2026-03-07 03:44:55 | cmd/doc-html-ui/main.go | DOC-HTML-UI | Created standalone GUI launcher (embedded web UI + Go HTTP server, Edge/Chrome app-mode, native file dialogs via PowerShell, all CLI flags as checkboxes/inputs, streaming log output, heartbeat auto-shutdown) |
| 2026-03-07 03:45:00 | cmd/doc-html-ui/ui.html | DOC-HTML-UI | Dark-themed responsive HTML/CSS/JS interface with all doc-html-translate flags |
| 2026-03-07 03:45:05 | scripts/build-ui.ps1 | DOC-HTML-UI | Build script for doc-html-ui.exe (pure Go, no CGO, -H windowsgui, icon embedding) |
| 2026-03-07 03:50:31 | internal/pdf/extract.go | PDF | Added panic recovery (defer/recover) for malformed PDFs — function-level and per-page level |
| 2026-03-07 03:54:13 | internal/pdf/extract.go | PDF | Added auto-repair fallback via pdfcpu (OptimizeFile + retry extraction), improved no-text diagnostic for scanned PDFs |
| 2026-03-07 03:54:44 | internal/pdf/extract.go | PDF | Mapped repaired-PDF no-text error back to original input path for clearer UX |
| 2026-03-07 03:55:17 | go.mod | Dependencies | Added pdfcpu dependency for PDF repair fallback; removed temporary parser experiment deps |
| 2026-03-07 03:55:17 | go.sum | Dependencies | Updated checksums after introducing pdfcpu fallback and tidy |
| 2026-03-07 03:58:02 | internal/pdf/extract.go | PDF | Implemented no-OCR fallback for image-only PDFs: create single HTML wrapper page with embedded original.pdf instead of failing extraction |
| 2026-03-07 04:05:54 | cmd/doc-html-ui/ui.html | DOC-HTML-UI | Added in-window drag-drop handling, fixed-size window enforcement, and compact horizontally-scrollable log styling |
| 2026-03-07 04:11:49 | internal/epub/epub.go | EPUB | Added XHTML->HTML normalization for content files and intra-book links to improve Chrome Translate compatibility on local files |
| 2026-03-07 17:35:38 | internal/browser/browser.go | Browser | Added normalizeTarget() to redirect XHTML open targets to nearest ancestor index.html |
| 2026-03-07 17:35:38 | internal/browser/browser_windows.go | Browser | Applied normalizeTarget() before Windows browser launch |
| 2026-03-07 17:35:38 | internal/htmlgen/navbar.go | Navigation | Added legacy XHTML guard: direct .xhtml open redirects to index.html; navbar blocks chapter-to-chapter .xhtml navigation |
| 2026-03-07 17:37:00 | internal/htmlgen/navbar.go | Navigation | Added image aspect-ratio guard: when image height changes, width is auto/computed from natural proportions |
| 2026-03-09 23:54:57 | internal/epub/epub.go | epub.Extract | Fix: resolveReservedNames renames EPUB content files named index.html to _content_index.html to avoid overwrite by generated nav TOC |
| 2026-03-10 01:03:46 | internal/config/flags.go | SplitSize | Default SplitSize changed from 0 to 5000; removed injectDefaultSplitValue hack; disable via -split 0 |
| 2026-03-10 01:16:56 | scripts/build.ps1 | goversioninfo | Fix: added -64 flag to goversioninfo to generate amd64-compatible .syso (was: relocation type 7 error) |
| 2026-03-10 01:32:31 | internal/htmlgen/navbar.go | navBarScript | feat: edge-scroll auto-navigation — PageDown/wheel-down at bottom → next page; PageUp/wheel-up at top → prev page (wheel threshold: 3 events) |

