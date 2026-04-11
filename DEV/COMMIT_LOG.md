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

| 2026-03-04 21:04:24 | master | 4de44aa | feat: TXT support (Phase 6-O) — extractor, pipeline, registration |

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


