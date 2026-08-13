# COMMIT LOG

## Project Links
- GitHub Pages: https://serzhyale.github.io/doc-html-translate/
- GitHub Repository: https://github.com/SerZhyAle/doc-html-translate

| Timestamp | Branch | Commit | Message |
|---|---|---|---|
| 2026-03-04 01:35:13 | master | f3fbc43 | chore: initialize master-only workflow with build-commit journaling |

Changed files:

| 2026-03-04 01:35:42 | master | 2cea839 | chore: enforce build-commit flow and live plan progress |

Changed files:
- DEV/CHANGELOG.md
- DEV/COMMIT_LOG.md
- DEV/plan.md
- scripts/commit_after_build.ps1

| 2026-03-04 01:38:14 | master | 1b43f01 | chore: move all build intermediates and logs to temp |

Changed files:
- .gitignore
- DEV/CHANGELOG.md
- DEV/plan.md
- README.md
- assets/epub2html.ico
- build/epub2html.ico
- configs/.typos.toml
- scripts/build.ps1
- scripts/check.ps1
- scripts/generate-icon.ps1
- scripts/lint.ps1
- scripts/test.ps1
- scripts/typo.ps1

| 2026-03-04 01:38:51 | master | 5d7c2be | fix: prevent root build artifacts from being tracked |

Changed files:
- .gitignore
- DEV/CHANGELOG.md
- build/epub2html.ico

| 2026-03-04 01:39:44 | master | 1ddcb28 | chore: move all project documentation into DEV directory |

Changed files:
- DEV/CHANGELOG.md
- DEV/README.md
- DEV/plan.md
- DEV/universal_copilot_instructions.md
- README.md
- universal_copilot_instructions.md

| 2026-03-04 01:40:28 | master | a013e3a | chore: archive outdated docs to temp and formalize docs lifecycle |

Changed files:
- DEV/CHANGELOG.md
- DEV/README.md
- DEV/plan.md
- DEV/task.md

| 2026-03-04 01:41:14 | master | 985ce17 | docs: add user questionnaire text file in DEV |

Changed files:
- DEV/CHANGELOG.md
- DEV/questionnaire_for_user.txt

| 2026-03-04 01:41:47 | master | 2c0a12d | chore: formalize local-only git and rollback policy in plan |

Changed files:
- DEV/CHANGELOG.md
- DEV/plan.md

| 2026-03-04 01:55:11 | master | 088d702 | feat: Phase 1 core - EPUB extraction, HTML generation, browser opening, idempotent re-open |

Changed files:
- .gitignore
- DEV/plan.md
- DEV/questionnaire_for_user.txt
- cmd/epub2html/main.go
- internal/app/app.go
- internal/browser/browser.go
- internal/browser/browser_nonwindows.go
- internal/browser/browser_windows.go
- internal/config/flags.go
- internal/epub/epub.go
- internal/epub/epub_test.go
- internal/htmlgen/htmlgen.go
- internal/htmlgen/htmlgen_test.go
- internal/pipeline/pipeline.go
- test_epub/#1 With a Bullet - a Transgender Romance Novel.epub
- test_epub/A Love Inspired - a Transgender Romance Novel.epub
- test_epub/A Touch of Magic_ 10 Book Bundl - Clover Cox.epub
- test_epub/After Midnight- A Succubus Feminization Romance.epub
- test_epub/Arabian Nights.epub
- test_epub/Bakery Girl.epub
- test_epub/Becoming Callie_ A Steamy Trans - Kate Stormdottir.epub
- test_epub/Becoming Kelly_ A Story of Tran - Kate Stormdottir.epub
- test_epub/Becoming The Prom Queen 2.epub
- test_epub/Birthday Present For A Sissy Hubby - Book 2 A Feminization Of A Husband Tale.epub
- test_epub/Bossed Around Feminization Transgender Transfor.epub

| 2026-03-04 01:58:30 | master | 0aad4a4 | feat: Phase 2 - Google Translate client, HTML text extraction, translation pipeline wired |

Changed files:
- go.mod
- go.sum
- internal/htmlproc/htmlproc.go
- internal/htmlproc/htmlproc_test.go
- internal/pipeline/pipeline.go
- internal/translator/translator.go
- internal/translator/translator_test.go

| 2026-03-04 02:02:05 | master | cfa725f | feat: Phase 3 - UX polish, error handling, version flag, README update, plan progress |

Changed files:
- DEV/README.md
- DEV/plan.md
- cmd/epub2html/main.go
- internal/config/flags.go
- scripts/build.ps1

| 2026-03-04 20:08:20 | master | 3b1d985 | feat: add PDF text extraction and pipeline support (Phase 5) |

Changed files:
- DEV/CHANGELOG.md
- DEV/README.md
- DEV/plan.md
- go.mod
- go.sum
- internal/app/app.go
- internal/config/flags.go
- internal/config/flags_test.go
- internal/htmlgen/navbar.go
- internal/htmlgen/navbar_test.go
- internal/pdf/extract.go
- internal/pdf/extract_test.go
- internal/pipeline/pipeline.go
- internal/translator/translator.go
- test_pdf/(His Executive Gender Swap Book 02) People Pleaser.pdf
- test_pdf/Closet+Trap.pdf
- test_pdf/Fashion Mistress - Alyson Belle.pdf
- test_pdf/Rivalry+Game.pdf

| 2026-03-04 20:15:11 | master | fc565c0 | rename: epub2html -> doc-html-translate (module, imports, binary, scripts, CSS, registry, docs) |

Changed files:
- DEV/CHANGELOG.md
- DEV/README.md
- DEV/plan.md
- cmd/doc-html-translate/main.go
- cmd/epub2html/main.go
- configs/.typos.toml
- go.mod
- internal/app/app.go
- internal/config/flags.go
- internal/htmlgen/htmlgen.go
- internal/htmlgen/htmlgen_test.go
- internal/htmlgen/navbar.go
- internal/htmlgen/navbar_test.go
- internal/pdf/extract.go
- internal/pipeline/pipeline.go
- internal/windowsreg/register_windows.go
- scripts/build.ps1
- scripts/generate-icon.ps1

| 2026-03-04 20:21:23 | master | c925cf6 | feat: splash screen on first launch + register .epub and .pdf |

Changed files:
- DEV/plan.md
- internal/app/app.go
- internal/windowsreg/register_nonwindows.go
- internal/windowsreg/register_windows.go

| 2026-03-04 21:04:24 | master | 4de44aa | feat: TXT support (Phase 6-O) - extractor, pipeline, registration |

Changed files:
- internal/config/flags.go
- internal/pipeline/pipeline.go
- internal/txt/extract.go
- internal/txt/extract_test.go
- internal/windowsreg/register_nonwindows.go
- internal/windowsreg/register_windows.go
- "test_txt/3_\320\270\320\263\321\200\320\260.txt"
- "test_txt/4_\320\274\320\275\320\276\320\263\320\276 \321\210\321\203\320\274\320\260.txt"
- test_txt/Post Message.txt
- test_txt/matrix_copy_src_SMB_1768776354503.txt
- "test_txt/\320\225\321\201\320\273\320\270 \320\261\321\213 \321\202\320\276\320\273\321\214\320\272\320\276_If only.txt"

| 2026-03-04 21:19:24 | master | d8d013b | feat: re-register on every empty launch + add TXT to splash |

Changed files:
- internal/app/app.go

| 2026-03-04 21:53:05 | master | 0f7cecc | fix: wider content layout for TXT/PDF (95% width, max 1400px) |

Changed files:
- internal/pdf/extract.go
- internal/txt/extract.go
- test_txt/Post Message/index.html
- test_txt/Post Message/page_001.html
- "test_txt/\320\225\321\201\320\273\320\270 \320\261\321\213 \321\202\320\276\320\273\321\214\320\272\320\276_If only/index.html"
- "test_txt/\320\225\321\201\320\273\320\270 \320\261\321\213 \321\202\320\276\320\273\321\214\320\272\320\276_If only/page_001.html"
- "test_txt/\320\225\321\201\320\273\320\270 \320\261\321\213 \321\202\320\276\320\273\321\214\320\272\320\276_If only/page_002.html"
- "test_txt/\320\225\321\201\320\273\320\270 \320\261\321\213 \321\202\320\276\320\273\321\214\320\272\320\276_If only/page_003.html"
- "test_txt/\320\225\321\201\320\273\320\270 \320\261\321\213 \321\202\320\276\320\273\321\214\320\272\320\276_If only/page_004.html"
- "test_txt/\320\225\321\201\320\273\320\270 \320\261\321\213 \321\202\320\276\320\273\321\214\320\272\320\276_If only/page_005.html"
- "test_txt/\320\225\321\201\320\273\320\270 \320\261\321\213 \321\202\320\276\320\273\321\214\320\272\320\276_If only/page_006.html"
- "test_txt/\320\225\321\201\320\273\320\270 \320\261\321\213 \321\202\320\276\320\273\321\214\320\272\320\276_If only/page_007.html"
- "test_txt/\320\225\321\201\320\273\320\270 \320\261\321\213 \321\202\320\276\320\273\321\214\320\272\320\276_If only/page_008.html"
- "test_txt/\320\225\321\201\320\273\320\270 \320\261\321\213 \321\202\320\276\320\273\321\214\320\272\320\276_If only/page_009.html"
- "test_txt/\320\225\321\201\320\273\320\270 \320\261\321\213 \321\202\320\276\320\273\321\214\320\272\320\276_If only/page_010.html"
- "test_txt/\320\225\321\201\320\273\320\270 \320\261\321\213 \321\202\320\276\320\273\321\214\320\272\320\276_If only/page_011.html"
- "test_txt/\320\225\321\201\320\273\320\270 \320\261\321\213 \321\202\320\276\320\273\321\214\320\272\320\276_If only/page_012.html"
- "test_txt/\320\225\321\201\320\273\320\270 \320\261\321\213 \321\202\320\276\320\273\321\214\320\272\320\276_If only/page_013.html"
- "test_txt/\320\225\321\201\320\273\320\270 \320\261\321\213 \321\202\320\276\320\273\321\214\320\272\320\276_If only/page_014.html"
- "test_txt/\320\225\321\201\320\273\320\270 \320\261\321\213 \321\202\320\276\320\273\321\214\320\272\320\276_If only/page_015.html"
- "test_txt/\320\225\321\201\320\273\320\270 \320\261\321\213 \321\202\320\276\320\273\321\214\320\272\320\276_If only/page_016.html"
- "test_txt/\320\225\321\201\320\273\320\270 \320\261\321\213 \321\202\320\276\320\273\321\214\320\272\320\276_If only/page_017.html"
- "test_txt/\320\225\321\201\320\273\320\270 \320\261\321\213 \321\202\320\276\320\273\321\214\320\272\320\276_If only/page_018.html"
- "test_txt/\320\225\321\201\320\273\320\270 \320\261\321\213 \321\202\320\276\320\273\321\214\320\272\320\276_If only/page_019.html"
- "test_txt/\320\225\321\201\320\273\320\270 \320\261\321\213 \321\202\320\276\320\273\321\214\320\272\320\276_If only/page_020.html"
- "test_txt/\320\225\321\201\320\273\320\270 \320\261\321\213 \321\202\320\276\320\273\321\214\320\272\320\276_If only/page_021.html"
- "test_txt/\320\225\321\201\320\273\320\270 \320\261\321\213 \321\202\320\276\320\273\321\214\320\272\320\276_If only/page_022.html"
- "test_txt/\320\225\321\201\320\273\320\270 \320\261\321\213 \321\202\320\276\320\273\321\214\320\272\320\276_If only/page_023.html"

| 2026-03-04 23:48:24 | master | cba5e65 | feat: add Markdown, FB2, RTF, HTML/HTM format support |

Changed files:
- go.mod
- go.sum
- internal/app/app.go
- internal/config/flags.go
- internal/fb2/extract.go
- internal/fb2/extract_test.go
- internal/htmlconv/extract.go
- internal/htmlconv/extract_test.go
- internal/md/extract.go
- internal/md/extract_test.go
- internal/pipeline/pipeline.go
- internal/rtf/extract.go
- internal/rtf/extract_test.go
- internal/windowsreg/register_nonwindows.go
- internal/windowsreg/register_windows.go
- test_txt/Post Message/index.html
- test_txt/Post Message/page_001.html
- "test_txt/\320\225\321\201\320\273\320\270 \320\261\321\213 \321\202\320\276\320\273\321\214\320\272\320\276_If only/index.html"
- "test_txt/\320\225\321\201\320\273\320\270 \320\261\321\213 \321\202\320\276\320\273\321\214\320\272\320\276_If only/page_001.html"
- "test_txt/\320\225\321\201\320\273\320\270 \320\261\321\213 \321\202\320\276\320\273\321\214\320\272\320\276_If only/page_002.html"
- "test_txt/\320\225\321\201\320\273\320\270 \320\261\321\213 \321\202\320\276\320\273\321\214\320\272\320\276_If only/page_003.html"
- "test_txt/\320\225\321\201\320\273\320\270 \320\261\321\213 \321\202\320\276\320\273\321\214\320\272\320\276_If only/page_004.html"
- "test_txt/\320\225\321\201\320\273\320\270 \320\261\321\213 \321\202\320\276\320\273\321\214\320\272\320\276_If only/page_005.html"
- "test_txt/\320\225\321\201\320\273\320\270 \320\261\321\213 \321\202\320\276\320\273\321\214\320\272\320\276_If only/page_006.html"
- "test_txt/\320\225\321\201\320\273\320\270 \320\261\321\213 \321\202\320\276\320\273\321\214\320\272\320\276_If only/page_007.html"
- "test_txt/\320\225\321\201\320\273\320\270 \320\261\321\213 \321\202\320\276\320\273\321\214\320\272\320\276_If only/page_008.html"
- "test_txt/\320\225\321\201\320\273\320\270 \320\261\321\213 \321\202\320\276\320\273\321\214\320\272\320\276_If only/page_009.html"
- "test_txt/\320\225\321\201\320\273\320\270 \320\261\321\213 \321\202\320\276\320\273\321\214\320\272\320\276_If only/page_010.html"
- "test_txt/\320\225\321\201\320\273\320\270 \320\261\321\213 \321\202\320\276\320\273\321\214\320\272\320\276_If only/page_011.html"
- "test_txt/\320\225\321\201\320\273\320\270 \320\261\321\213 \321\202\320\276\320\273\321\214\320\272\320\276_If only/page_012.html"
- "test_txt/\320\225\321\201\320\273\320\270 \320\261\321\213 \321\202\320\276\320\273\321\214\320\272\320\276_If only/page_013.html"
- "test_txt/\320\225\321\201\320\273\320\270 \320\261\321\213 \321\202\320\276\320\273\321\214\320\272\320\276_If only/page_014.html"
- "test_txt/\320\225\321\201\320\273\320\270 \320\261\321\213 \321\202\320\276\320\273\321\214\320\272\320\276_If only/page_015.html"
- "test_txt/\320\225\321\201\320\273\320\270 \320\261\321\213 \321\202\320\276\320\273\321\214\320\272\320\276_If only/page_016.html"
- "test_txt/\320\225\321\201\320\273\320\270 \320\261\321\213 \321\202\320\276\320\273\321\214\320\272\320\276_If only/page_017.html"
- "test_txt/\320\225\321\201\320\273\320\270 \320\261\321\213 \321\202\320\276\320\273\321\214\320\272\320\276_If only/page_018.html"
- "test_txt/\320\225\321\201\320\273\320\270 \320\261\321\213 \321\202\320\276\320\273\321\214\320\272\320\276_If only/page_019.html"
- "test_txt/\320\225\321\201\320\273\320\270 \320\261\321\213 \321\202\320\276\320\273\321\214\320\272\320\276_If only/page_020.html"
- "test_txt/\320\225\321\201\320\273\320\270 \320\261\321\213 \321\202\320\276\320\273\321\214\320\272\320\276_If only/page_021.html"
- "test_txt/\320\225\321\201\320\273\320\270 \320\261\321\213 \321\202\320\276\320\273\321\214\320\272\320\276_If only/page_022.html"
- "test_txt/\320\225\321\201\320\273\320\270 \320\261\321\213 \321\202\320\276\320\273\321\214\320\272\320\276_If only/page_023.html"

| 2026-03-04 23:54:37 | master | daa31bd | feat: skip TOC and navigation for single-page output |

Changed files:
- DEV/CHANGELOG.md
- internal/htmlgen/htmlgen.go
- internal/pipeline/pipeline.go

| 2026-03-04 23:58:47 | master | daef769 | feat: add first-sentence snippets to TOC from translated content |

Changed files:
- internal/htmlgen/htmlgen.go
- internal/htmlgen/htmlgen_test.go
- internal/pipeline/pipeline.go

| 2026-03-05 00:12:56 | master | 4190483 | feat: fall back to TXT extractor for unknown file extensions |

Changed files:
- internal/config/flags.go
- internal/pipeline/pipeline.go

| 2026-03-05 00:15:19 | master | 9029991 | fix: handle Linux LF, Windows CRLF and old Mac CR line endings in TXT parser |

Changed files:
- internal/txt/extract.go
- internal/txt/extract_test.go

| 2026-03-05 00:42:34 | master | c3947fe | fix: wrap navbar JS in CDATA for XHTML/EPUB compatibility |

Changed files:
- internal/htmlgen/navbar.go

| 2026-03-05 00:54:56 | master | a10fee4 | feat: TOC full-width, snippet replaces filename label, 2 sentences |

Changed files:
- internal/htmlgen/htmlgen.go
- internal/htmlgen/htmlgen_test.go


| 2026-06-27 16:26:12 | main | ec7d9b2 | feat: GUI default-handler + cost/TOC controls, local build/release split, cheaper CI |

Changed files:
- .claude/commands/build.md
- .claude/commands/release.md
- .gitattributes
- .github/workflows/publish-extension.yml
- .github/workflows/release.yml
- AGENTS.md
- CLAUDE.md
- DEV/CHANGELOG.md
- DEV/RELEASE.md
- README.md
- build/doc-html-translate.exe
- build/doc-html-ui.exe
- cmd/doc-html-translate/main.go
- cmd/doc-html-ui/main.go
- cmd/doc-html-ui/main_test.go
- cmd/doc-html-ui/ui.html
- configs/.golangci.yml
- configs/.typos.toml
- docs.html
- docs.ru.html
- docs.uk.html
- extension.html
- extension/README.md
- extension/build.mjs
- extension/src/options.html
- extension/src/viewer.js
- extension/store/LISTING.md
- index.html
- internal/app/app.go
- internal/config/flags.go
- internal/dialog/dialog_windows.go
- internal/epub/epub.go
- internal/fb2/extract.go
- internal/htmlconv/extract.go
- internal/md/extract.go
- internal/pipeline/pipeline.go
- internal/translator/ollama.go
- internal/translator/translator_test.go
- scripts/build-local.ps1
- scripts/commit_after_build.ps1
- scripts/release.ps1

| 2026-06-27 18:37:07 | main | b72e522 | feat: GUI drag-and-drop, compact layout, language pickers, persisted settings

Also fixes: Cyrillic dialog paths (base64 over the PowerShell boundary),
the file dialog opening behind the window (own to the foreground window),
and the server being killed mid-conversion by the heartbeat watchdog.

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com> |

Changed files:
- DEV/CHANGELOG.md
- build/doc-html-translate.exe
- build/doc-html-ui.exe
- cmd/doc-html-ui/main.go
- cmd/doc-html-ui/main_test.go
- cmd/doc-html-ui/ui.html

| 2026-07-01 10:06:40 | main | bc72494 | feat: OCR image overlay in browser extension and desktop app (Tesseract) |

Changed files:
- AGENTS.md
- DEV/CHANGELOG.md
- DEV/plan/2026-07-01_app-ocr-image-overlay.md
- DEV/plan/2026-07-01_ocr-image-overlay.md
- DEV/plan/2026-07-01_ocr-image-overlay/INDEX.md
- DEV/plan/2026-07-01_ocr-image-overlay/PHASE_01__ocr-engine-vendor.md
- DEV/plan/2026-07-01_ocr-image-overlay/PHASE_02__language-manager.md
- DEV/plan/2026-07-01_ocr-image-overlay/PHASE_03__overlay-core.md
- DEV/plan/2026-07-01_ocr-image-overlay/PHASE_04__context-menu-page.md
- DEV/plan/2026-07-01_ocr-image-overlay/PHASE_05__controls-ui.md
- DEV/plan/2026-07-01_ocr-image-overlay/PHASE_06__epub-integration.md
- DEV/plan/2026-07-01_ocr-image-overlay/PHASE_07__pdf-integration.md
- DEV/plan/2026-07-01_ocr-image-overlay/PHASE_08__docs-cleanup.md
- DEV/plan/2026-07-01_ocr-image-overlay/research/01__ocr-stack-decisions.md
- README.md
- build.ps1
- build/doc-html-translate.exe
- build/doc-html-ui.exe
- build/tessdata/eng.traineddata
- cmd/doc-html-ui/main.go
- cmd/doc-html-ui/ui.html
- docs.html
- docs.ru.html
- docs.uk.html
- extension.html
- extension/README.md
- extension/_locales/en/messages.json
- extension/_locales/ru/messages.json
- extension/_locales/uk/messages.json
- extension/build.mjs
- extension/manifest.json
- extension/package-lock.json
- extension/package.json
- extension/src/background.js
- extension/src/ocr-lang.js
- extension/src/ocr-overlay.css
- extension/src/ocr-overlay.js
- extension/src/ocr.html
- extension/src/ocr.js
- extension/src/options.html
- extension/src/options.js
- extension/src/pdf-images.js
- extension/src/popup.html
- extension/src/popup.js
- extension/src/viewer.css
- extension/src/viewer.js
- extension/store/LISTING.md
- extension/store/PRIVACY.md
- index.html
- internal/app/app.go
- internal/config/flags.go
- internal/ocr/overlay.go
- internal/ocr/tessdata.go
- internal/ocr/tesseract.go
- internal/ocr/tsv_test.go
- internal/pipeline/pipeline.go
- scripts/build-ui.ps1
- scripts/build.ps1

| 2026-07-01 16:38:50 | main | 34da39e | feat: cross-edition parity - extension formats + OCR overlay quality (rendering fix, adaptive colours, noise filter) |

Changed files:
- .gitignore
- AGENTS.md
- CLAUDE.md
- DEV/CHANGELOG.md
- DEV/README.md
- DEV/plan/2026-07-01_app-ocr-image-overlay.md
- DEV/plan/2026-07-01_cross-edition-parity.md
- DEV/plan/2026-07-01_extension-format-parity.md
- DEV/plan/2026-07-01_extension-format-parity/INDEX.md
- DEV/plan/2026-07-01_extension-format-parity/PHASE_01__foundation.md
- DEV/plan/2026-07-01_extension-format-parity/PHASE_02__text-rtf-html.md
- DEV/plan/2026-07-01_extension-format-parity/PHASE_03__markdown-fb2.md
- DEV/plan/2026-07-01_extension-format-parity/PHASE_04__ebook-mobi-azw3.md
- DEV/plan/2026-07-01_extension-format-parity/PHASE_05__docs-cleanup.md
- DEV/plan/_TEMPLATE_cross-edition.md
- DEV/research/extension_formats_feasibility_ru.md
- README.md
- assets/doc-html-translate.ico
- build/doc-html-translate.exe
- build/doc-html-ui.exe
- cmd/doc-html-ui/favicon.ico
- cmd/doc-html-ui/main.go
- cmd/doc-html-ui/ui.html
- docs.html
- docs.ru.html
- docs.uk.html
- docs/PARITY.md
- extension-privacy.html
- extension.html
- extension/README.md
- extension/_locales/en/messages.json
- extension/_locales/ru/messages.json
- extension/_locales/uk/messages.json
- extension/build.mjs
- extension/manifest.json
- extension/package-lock.json
- extension/package.json
- extension/src/background.js
- extension/src/defaults.js
- extension/src/ebook.js
- extension/src/epub.js
- extension/src/fb2.js
- extension/src/html.js
- extension/src/md.js
- extension/src/ocr-lang.js
- extension/src/ocr-overlay.js
- extension/src/ocr-text.js
- extension/src/options.html
- extension/src/options.js
- extension/src/popup.html
- extension/src/popup.js
- extension/src/rtf.js
- extension/src/sanitize.js
- extension/src/txt.js
- extension/src/viewer.css
- extension/src/viewer.html
- extension/src/viewer.js
- extension/store/LISTING.md
- extension/store/PRIVACY.md
- extension/test/ebook.test.mjs
- extension/test/ocr-text.test.mjs
- extension/test/rtf.test.mjs
- extension/test/txt.test.mjs
- index.html
- internal/app/app.go
- internal/config/flags.go
- internal/epub/toc.go
- internal/htmlgen/htmlgen.go
- internal/htmlgen/htmlgen_test.go
- internal/htmlgen/navbar.go
- internal/htmlgen/navbar_test.go
- internal/htmlgen/singlepage.go
- internal/ocr/overlay.go
- internal/ocr/tessdata.go
- internal/ocr/tesseract.go
- internal/ocr/text.go
- internal/ocr/tsv_test.go
- internal/pdf/extract.go
- internal/pipeline/pipeline.go
- internal/syslocale/locale_nonwindows.go
- internal/syslocale/locale_windows.go
- scripts/build-ui.ps1
- scripts/generate-icon.ps1
- tests/parity_test.go
- tests/testdoc_test.go
- tests/ui_cli_parity_test.go

| 2026-07-02 13:39:55 | main | 521a4fa | ext: OCR overlay quality + docs |

Changed files:
- DEV/CHANGELOG.md
- DEV/plan/2026-07-01_ocr-overlay-line-clustering.md
- DEV/plan/2026-07-01_ocr-pre-upscale.md
- DEV/plan/2026-07-02_ocr-desktop-tsv-config.md
- DEV/plan/2026-07-02_ocr-psm-parity.md
- build.ps1
- build/doc-html-translate.exe
- build/doc-html-ui.exe
- cmd/doc-html-ui/main.go
- cmd/doc-html-ui/ui.html
- docs.html
- docs.ru.html
- docs.uk.html
- docs/PARITY.md
- extension.html
- extension/README.md
- extension/_locales/en/messages.json
- extension/_locales/ru/messages.json
- extension/_locales/uk/messages.json
- extension/build.mjs
- extension/manifest.json
- extension/package.json
- extension/src/background.js
- extension/src/ocr-overlay.css
- extension/src/ocr-overlay.js
- extension/src/options.html
- extension/src/options.js
- extension/src/popup.html
- extension/src/popup.js
- extension/src/viewer.css
- extension/src/viewer.js
- extension/store/LISTING.md
- internal/ocr/overlay.go
- internal/ocr/tesseract.go
- internal/ocr/tsv_test.go
- internal/ocr/upscale_test.go
- internal/outputpath/outputpath.go
- internal/outputpath/outputpath_test.go
- internal/pipeline/pipeline.go
- internal/pipeline/pipeline_test.go
- scripts/build-extension.ps1
- tests/parity_test.go

| 2026-07-09 02:18:09 | main | 8db7934 | feat: show app in Windows "Open with" list without setting it as default |

Changed files:
- DEV/CHANGELOG.md
- README.md
- build/doc-html-translate.exe
- build/doc-html-ui.exe
- cmd/doc-html-ui/main.go
- cmd/doc-html-ui/ui.html
- docs.html
- docs.ru.html
- docs.uk.html
- internal/app/app.go
- internal/config/flags.go
- internal/windowsreg/register_nonwindows.go
- internal/windowsreg/register_windows.go
- tests/ui_cli_parity_test.go

| 2026-07-09 22:29:22 | main | 6a78fe8 | fix(pdf): detect Cyrillic ALL-CAPS headings, warn on malformed -src/-dst |

Changed files:
- DEV/CHANGELOG.md
- DEV/plan/2026-07-01_cross-edition-parity.md
- build/doc-html-translate.exe
- build/doc-html-ui.exe
- docs/PARITY.md
- internal/config/lang.go
- internal/config/lang_test.go
- internal/pdf/extract.go
- internal/pdf/extract_test.go
- internal/pipeline/pipeline.go

| 2026-07-14 17:04:38 | main | 8c02055 | docs: changelog for 26.0714.1658 (image input) + 26.0712 (ext PDF mirror fix) |

Changed files:
- DEV/CHANGELOG.md
- build/doc-html-translate.exe
- build/doc-html-ui.exe

| 2026-07-15 21:58:20 | main | f774ef8 | feat: opt-in file association + right-click Convert to HTML; universal setup.exe installer

App (CLI + GUI): file-type association is now opt-in, off by default. Every launch / first-run
registers only the non-destructive "Convert to HTML" right-click verb + "Open with" for all
supported types; becoming the default handler is a separate opt-in (-register, GUI toggle,
one-time first-run prompt), and new -unregister releases it.

Extension: auto-interception off by default (enabledByDefault false); new
"Convert with doc-html-translate" context menu opens supported doc links / pages in the viewer.

Distribution: new universal setup.exe (Inno Setup, x86 + x64, per-user, no admin) built by
scripts/build-installer.ps1; plus dev-workflow scripts (a.ps1 launcher, commit-push.ps1,
reinstall.ps1). Docs updated across README, index.html (en/ru/uk), docs pages, PARITY,
DEV/RELEASE, changelog.

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com> |

Changed files:
- AGENTS.md
- DEV/CHANGELOG.md
- DEV/README.md
- DEV/RELEASE.md
- DEV/plan/2026-07-15_optional-file-association-context-menu.md
- README.md
- a.ps1
- assets/doc-html-translate.ico
- build/doc-html-translate.exe
- build/doc-html-ui.exe
- cmd/doc-html-ui/favicon.ico
- cmd/doc-html-ui/main.go
- cmd/doc-html-ui/ui.html
- configs/.typos.toml
- docs.html
- docs.ru.html
- docs.uk.html
- docs/PARITY.md
- extension.html
- extension/_locales/en/messages.json
- extension/_locales/ru/messages.json
- extension/_locales/uk/messages.json
- extension/src/background.js
- extension/src/defaults.js
- extension/src/options.js
- extension/src/popup.html
- extension/src/popup.js
- index.html
- installer/doc-html-translate.iss
- internal/app/app.go
- internal/config/flags.go
- internal/config/flags_test.go
- internal/windowsreg/register_nonwindows.go
- internal/windowsreg/register_windows.go
- reinstall.ps1
- scripts/build-installer.ps1
- scripts/commit-push.ps1
- scripts/release.ps1
- tests/ui_cli_parity_test.go

| 2026-07-17 00:51:45 | main | b400f59 | fix: large PDFs stop freezing - one-pass PDF image extraction (2h20m -> 6.2s) + chunked extension render |

Changed files:
- DEV/CHANGELOG.md
- assets/doc-html-translate.ico
- build/doc-html-translate.exe
- build/doc-html-ui.exe
- cmd/doc-html-ui/favicon.ico
- docs/PARITY.md
- extension/README.md
- extension/src/viewer.js
- internal/pdf/extract.go

| 2026-07-18 02:12:43 | main | 474072d | chore(gate): pin amd64 test arch, skip oversized corpus samples, sweep run-2 notes |

Changed files:
- DEV/CHANGELOG.md
- DEV/RELEASE_STATE.md
- DEV/research/format_verification_sweep.md
- build/doc-html-translate.exe
- build/doc-html-ui.exe
- configs/.typos.toml
- internal/comic/comic_test.go
- internal/comic/extract.go
- internal/ocr/batch_test.go
- internal/ocr/overlay.go
- scripts/test.ps1
- tests/testdoc_test.go

| 2026-07-18 02:18:29 | main | 9e03eba | docs: metadata + store-copy polish for release 26.0718 (comics in MSIX/description, typography) |

Changed files:
- DEV/CHANGELOG.md
- build/doc-html-translate.exe
- build/doc-html-ui.exe
- extension.html
- extension/store/LISTING.md
- extension/store/PRIVACY.md
- msix/AppxManifest.xml

| 2026-07-28 20:48:28 | main | 6b835ad | feat(i18n): thirteen interface languages across CLI, GUI, extension, site and listings |

Changed files:
- AGENTS.md
- DEV/CHANGELOG.md
- DEV/DOCS_SURFACES.md
- DEV/plan/2026-07-28_thirteen-ui-languages.md
- DEV/plan/2026-07-28_thirteen-ui-languages/INDEX.md
- DEV/plan/2026-07-28_thirteen-ui-languages/PHASE_01__typography-guard-scope.md
- DEV/plan/2026-07-28_thirteen-ui-languages/PHASE_02__go-i18n-layer.md
- DEV/plan/2026-07-28_thirteen-ui-languages/PHASE_03__cli-page-and-flag.md
- DEV/plan/2026-07-28_thirteen-ui-languages/PHASE_04__gui-thirteen-languages.md
- DEV/plan/2026-07-28_thirteen-ui-languages/PHASE_05__extension-thirteen-languages.md
- DEV/plan/2026-07-28_thirteen-ui-languages/PHASE_06__site-and-readme.md
- DEV/plan/2026-07-28_thirteen-ui-languages/PHASE_07__screenshots.md
- DEV/plan/2026-07-28_thirteen-ui-languages/PHASE_08__packaging-and-listings.md
- DEV/plan/2026-07-28_thirteen-ui-languages/PHASE_09__docs-cleanup.md
- DEV/plan/ROADMAP.md
- README.md
- README_RU.md
- README_UK.md
- ar/index.html
- bn/index.html
- build/doc-html-translate.exe
- build/doc-html-ui.exe
- cmd/doc-html-ui/i18n.js
- cmd/doc-html-ui/i18n_test.go
- cmd/doc-html-ui/main.go
- cmd/doc-html-ui/ui.html
- configs/.typos.toml
- de/index.html
- docs.html
- docs.ru.html
- docs.uk.html
- docs/PARITY.md
- es/index.html
- extension-privacy.html
- extension.html
- extension/README.md
- extension/_locales/ar/messages.json
- extension/_locales/bn/messages.json
- extension/_locales/de/messages.json
- extension/_locales/en/messages.json
- extension/_locales/es/messages.json
- extension/_locales/fr/messages.json
- extension/_locales/hi/messages.json
- extension/_locales/it/messages.json
- extension/_locales/pt/messages.json
- extension/_locales/ru/messages.json
- extension/_locales/uk/messages.json
- extension/_locales/ur/messages.json
- extension/_locales/zh_CN/messages.json
- extension/src/i18n.js
- extension/src/options.html
- extension/src/options.js
- extension/src/popup.html
- extension/src/popup.js
- extension/src/viewer.html
- extension/src/viewer.js
- extension/store/LISTING.md
- extension/test/i18n.test.mjs
- fr/index.html
- hi/index.html
- index.html
- installer/doc-html-translate.iss
- internal/app/app.go
- internal/app/splash.go
- internal/app/splash/ar.txt
- internal/app/splash/bn.txt
- internal/app/splash/de.txt
- internal/app/splash/en.txt
- internal/app/splash/es.txt
- internal/app/splash/fr.txt
- internal/app/splash/hi.txt
- internal/app/splash/it.txt
- internal/app/splash/pt.txt
- internal/app/splash/ru.txt
- internal/app/splash/uk.txt
- internal/app/splash/ur.txt
- internal/app/splash/zh.txt
- internal/app/splash_test.go
- internal/config/flags.go
- internal/htmlgen/htmlgen.go
- internal/htmlgen/navbar.go
- internal/htmlgen/singlepage.go
- internal/i18n/i18n.go
- internal/i18n/i18n_cli.go
- internal/i18n/i18n_reader.go
- internal/i18n/i18n_test.go
- internal/syslocale/locale_nonwindows.go
- internal/syslocale/locale_windows.go
- it/index.html
- msix/AppxManifest.xml
- privacy.html
- pt/index.html
- robots.txt
- sitemap.xml
- tests/smoke_test.go
- tests/typography_test.go
- tools/store/build-store-listing-csv.ps1
- tools/store/gui-ar.png
- tools/store/gui-bn.png
- tools/store/gui-de.png
- tools/store/gui-en-us.png
- tools/store/gui-es.png
- tools/store/gui-fr.png
- tools/store/gui-hi.png
- tools/store/gui-it.png
- tools/store/gui-pt-br.png
- tools/store/gui-ru.png
- tools/store/gui-uk.png
- tools/store/gui-ur.png
- tools/store/gui-zh-hans.png
- tools/store/listing/ar.txt
- tools/store/listing/bn.txt
- tools/store/listing/de.txt
- tools/store/listing/en.txt
- tools/store/listing/es.txt
- tools/store/listing/fr.txt
- tools/store/listing/hi.txt
- tools/store/listing/it.txt
- tools/store/listing/pt.txt
- tools/store/listing/ru.txt
- tools/store/listing/uk.txt
- tools/store/listing/ur.txt
- tools/store/listing/zh.txt
- tools/store/listingData.csv
- tools/store/make-gui-screenshot.ps1
- tools/store/make-screenshot.ps1
- tools/store/reading-view-ar.png
- tools/store/reading-view-bn.png
- tools/store/reading-view-de.png
- tools/store/reading-view-en-us.png
- tools/store/reading-view-es.png
- tools/store/reading-view-fr.png
- tools/store/reading-view-hi.png
- tools/store/reading-view-it.png
- tools/store/reading-view-pt-br.png
- tools/store/reading-view-ru.png
- tools/store/reading-view-uk.png
- tools/store/reading-view-ur.png
- tools/store/reading-view-zh-hans.png
- tools/store/reading-view.png
- tools/store/table-of-contents-ar.png
- tools/store/table-of-contents-bn.png
- tools/store/table-of-contents-de.png
- tools/store/table-of-contents-en-us.png
- tools/store/table-of-contents-es.png
- tools/store/table-of-contents-fr.png
- tools/store/table-of-contents-hi.png
- tools/store/table-of-contents-it.png
- tools/store/table-of-contents-pt-br.png
- tools/store/table-of-contents-ru.png
- tools/store/table-of-contents-uk.png
- tools/store/table-of-contents-ur.png
- tools/store/table-of-contents-zh-hans.png
- tools/store/table-of-contents.png
- ur/index.html
- winget/SerZhyAle.DocHtmlTranslate.locale.ar-SA.yaml
- winget/SerZhyAle.DocHtmlTranslate.locale.bn-BD.yaml
- winget/SerZhyAle.DocHtmlTranslate.locale.de-DE.yaml
- winget/SerZhyAle.DocHtmlTranslate.locale.en-US.yaml
- winget/SerZhyAle.DocHtmlTranslate.locale.es-ES.yaml
- winget/SerZhyAle.DocHtmlTranslate.locale.fr-FR.yaml
- winget/SerZhyAle.DocHtmlTranslate.locale.hi-IN.yaml
- winget/SerZhyAle.DocHtmlTranslate.locale.it-IT.yaml
- winget/SerZhyAle.DocHtmlTranslate.locale.pt-BR.yaml
- winget/SerZhyAle.DocHtmlTranslate.locale.ru-RU.yaml
- winget/SerZhyAle.DocHtmlTranslate.locale.uk-UA.yaml
- winget/SerZhyAle.DocHtmlTranslate.locale.ur-PK.yaml
- winget/SerZhyAle.DocHtmlTranslate.locale.zh-CN.yaml
- zh/index.html

| 2026-07-29 01:40:40 | main | f87bf18 | docs(release): what's new for 26.0729.0134 + winget manifests stamped |

Changed files:
- DEV/CHANGELOG.md
- DEV/RELEASE_STATE.md
- build/doc-html-translate.exe
- build/doc-html-ui.exe
- tools/store/build-store-listing-csv.ps1
- tools/store/listing/en.txt
- tools/store/listing/ru.txt
- tools/store/listing/uk.txt
- tools/store/listingData.csv
- winget/SerZhyAle.DocHtmlTranslate.installer.yaml
- winget/SerZhyAle.DocHtmlTranslate.locale.ar-SA.yaml
- winget/SerZhyAle.DocHtmlTranslate.locale.bn-BD.yaml
- winget/SerZhyAle.DocHtmlTranslate.locale.de-DE.yaml
- winget/SerZhyAle.DocHtmlTranslate.locale.en-US.yaml
- winget/SerZhyAle.DocHtmlTranslate.locale.es-ES.yaml
- winget/SerZhyAle.DocHtmlTranslate.locale.fr-FR.yaml
- winget/SerZhyAle.DocHtmlTranslate.locale.hi-IN.yaml
- winget/SerZhyAle.DocHtmlTranslate.locale.it-IT.yaml
- winget/SerZhyAle.DocHtmlTranslate.locale.pt-BR.yaml
- winget/SerZhyAle.DocHtmlTranslate.locale.ru-RU.yaml
- winget/SerZhyAle.DocHtmlTranslate.locale.uk-UA.yaml
- winget/SerZhyAle.DocHtmlTranslate.locale.ur-PK.yaml
- winget/SerZhyAle.DocHtmlTranslate.locale.zh-CN.yaml
- winget/SerZhyAle.DocHtmlTranslate.yaml

| 2026-08-13 18:58:11 | main | 3a8699f | docs: what's new for the OCR-composition release, across the surfaces it touches |

Changed files:
- README.md
- README_RU.md
- README_UK.md
- build/doc-html-translate.exe
- build/doc-html-ui.exe
- docs.html
- docs.ru.html
- docs.uk.html
- extension/store/LISTING.md
- index.html
- tools/store/listing/en.txt
- tools/store/listing/ru.txt
- tools/store/listing/uk.txt

