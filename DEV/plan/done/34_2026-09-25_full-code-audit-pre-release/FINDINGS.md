# Findings register - pre-release audit (ticket 34)

**Scope:** class A as sliced in [`slices.json`](slices.json) - shipped Go (`cmd/`, `internal/`), the GUI
page, `extension/src`, and the release path (`scripts/`, `msix/`, `installer/`, the winget manifest
source, the extension build and publish scripts, the tag workflows).
**Base:** changes are measured against `41fbc1b`, the commit the previous register cites. Each line
number below cites the commit its slice was read at (named in the slice's phase file).
**Previous register:** [`DEV/research/audit_2026-09-24/README.md`](../../../research/audit_2026-09-24/README.md).

Format, the same as the previous register so the two read as one history:

- Re-check: `id - still fixed | regressed | one edition only | superseded - slice - evidence`.
- New finding: `id - severity - confidence - finding - evidence - ticket`, where severity is
  crit / high / med / low, confidence `conf` (read at the cited line or run) or `plaus` (depends on
  the OS or a third party), and ticket is `#NN` (filed, or an existing ticket linked after dedupe),
  `inline: <proof>` (fixed inside the slice, the test run cited), or `-` (register line only).
- Ids continue per area letter from the previous register: P CLI / pipeline / support, G GUI,
  E EPUB / HTML, X extractors, T translation, O OCR, B extension, Q static checks, and the new
  R for the release path.

`./scripts/audit-slices.ps1 -Summary` reads this file: it counts the re-check lines against the ids
the manifest placed, and the findings by severity.

## Re-check of the 2026-09-24 register

One line per slice that holds an id's code, so an id checked from two slices has two lines. Verdicts:
`still fixed`, `regressed`, `one edition only` (fixed in one edition, its twin still open), `superseded`.
No id regressed. Three are fixed in the desktop edition only - E6, E12, E14, whose extension halves
done/06 left under "Not done"; they are filed as B46. G19, Q5 and Q6 cite only class-B files (tests, the
lab) and are outside this campaign.
- G5 - still fixed - S02 - main.go:875-886
- G10 - still fixed - S02 - ui.html:743-752, liveness.go:29,41-74
- G15 - still fixed - S02 - ui.html:797-806
- G17 - still fixed - S02 - ui.html:1105
- G18 - still fixed - S02 - ui.html:1593-1597
- G1 - still fixed - S01 - run.go:351-375
- G2 - still fixed - S01 - main.go:512, browser_windows.go:16-25, report.go:63-77
- G3 - still fixed - S01 - guard.go:61-86, main.go:85-118
- G4 - still fixed - S01 - main.go:936-939
- G5 - still fixed - S01 - main.go:875-886
- G6 - still fixed - S01 - run.go:227-234,146-156, proc_windows.go:23-43
- G7 - still fixed - S01 - run.go:381-416
- G8 - still fixed - S01 - proc_windows.go:13, hide_windows.go:15-17
- G9 - still fixed - S01 - main.go:226-258
- G10 - still fixed - S01 - liveness.go:28-65
- G11 - still fixed - S01 - main.go:444-456,569-578
- G12 - still fixed - S01 - main.go:302, store.go:104-122
- G13 - still fixed - S01 - main.go:966-1018
- G14 - still fixed - S01 - main.go:593-594,172-190
- G16 - still fixed - S01 - main.go:887-893
- P21 - still fixed - S06 - log.go:54-64
- E22 - still fixed - S06 - (md half) md/extract.go:155,42-49
- X9 - still fixed - S06 - (mobi half) mobi/extract.go:64-69
- Q4 - still fixed - S06 - mobi/extract.go:48
- O5 - still fixed - S10 - ocr/run.go:19-31, script.go:97
- O6 - still fixed - S10 - overlay.go:569-587, budget.go:82-115
- O7 - still fixed - S10 - run.go:41-51 (+pipeline/guard.go:41-49)
- O8 - still fixed - S10 - script.go:122-129
- O9 - still fixed - S10 - run.go:15, run_test.go:41-47
- O10 - still fixed - S10 - overlay.go:205-207
- O11 - still fixed - S10 - overlay.go:333-350
- O12 - still fixed - S10 - overlay.go:317,824-831,873-879,913-920
- P3 - still fixed - S03 - browser_windows.go:25-36
- P12 - still fixed - S03 - app.go:180-198, register_windows.go:63
- P14 - still fixed - S03 - app.go:42-72,234-241
- P15 - still fixed - S03 - bundledtools/cache.go:62-116
- P24 - superseded - S03 - browser_windows.go:32 (ShellExecute, no child)
- X9 - still fixed - S03 - (comic) readers_sevenzip.go:70-77,209-216
- X15 - still fixed - S03 - readers_sevenzip.go:88-107,188-224
- X16 - still fixed - S03 - readers_sevenzip.go:229-246
- X17 - still fixed - S03 - comic/extract.go:131-149,188-198
- X24 - still fixed - S03 - container.go:32-66 (JS residual B32)
- B22 - still fixed - S03 - comic.js:267-317 (residual B31)
- B23 - still fixed - S03 - comic.js:140-150,216-218
- Q1 - still fixed - S03 - browser_windows.go:40
- Q2 - still fixed - S03 - readers_sevenzip.go:251-265
- E14 - one edition only - S09 - htmlconv/extract.go:97-110 decodes; extension/src/html.js:11 still TextDecoder("utf-8") (PARITY.md:419 open gap; filed as B46)
- E15 - still fixed - S09 - htmlconv/extract.go:143, assets/html.go:9-35,107-138
- E21 - still fixed - S09 - assets/copier.go:157-185,102
- E22 - still fixed - S09 - htmlconv/extract.go:154-208, md/extract.go:42,103,155
- X4 - still fixed - S09 - fb2/content.go:59-86, fb2.js:21-40
- X18 - still fixed - S09 - content.go:128-173, fb2.js:134-154 (residual B34)
- X19 - still fixed - S09 - fb2/extract.go:183-210
- X20 - still fixed - S09 - content.go:211-257
- P1 - still fixed - S05 - ownership.go:163, pipeline/outputdir.go:51-77
- P6 - still fixed - S05 - ownership.go:155-194,246-251
- P16 - still fixed - S05 - outputpath.go:28-75
- P23 - still fixed - S05 - textutil/lines.go:24-26
- X6 - still fixed - S05 - pdf/images.go:88-103
- X7 - still fixed - S05 - images.go:125-132,321-330, toc.go:19-23, extract.go:469-474,611-620
- X8 - still fixed - S05 - extract.go:132-140
- X9 - still fixed - S05 - (pdf) extract.go:102-108, images.go:392-396
- X10 - still fixed - S05 - pdftotext_windows.go:79-92
- X11 - still fixed - S05 - images.go:424,435
- X14 - still fixed - S05 - images.go:443-488
- B9 - still fixed - S05 - pdf-images.js:205-221
- O1 - still fixed - S11 - download.go:94-107,124
- O2 - still fixed - S11 - download.go:135-138,174
- O3 - still fixed - S11 - download.go:32-46,170,190-199,251-265
- O4 - still fixed - S11 - tessdata.go:59-65,94-119
- O5 - still fixed - S11 - run.go:19-31
- O6 - still fixed - S11 - overlay.go:576, budget.go:91-115
- O8 - still fixed - S11 - script.go:91,122-129
- B2 - still fixed - S12 - ocr-plates.js:96,103-115,182-188
- B4 - still fixed - S12 - ocr-overlay.js:65-70
- B7 - still fixed - S12 - ocr-overlay.js:139-141
- B16 - still fixed - S12 - viewer.js:342
- B26 - still fixed - S12 - `extension/src/viewer.js:894` sends credentials; the second site the register cited, `ocr-overlay.js` `fetchToBlob`, was outside ticket 19's scope and is B36
- P1 - still fixed - S08 - pipeline.go:70-73
- P2 - still fixed - S08 - outputdir.go:26-77, pipeline.go:88-95
- P4 - still fixed - S08 - pipeline.go:332-341,101-109
- P5 - still fixed - S08 - pipeline.go:105
- P7 - still fixed - S08 - outputdir.go:40-60, pipeline.go:102-104
- P8 - still fixed - S08 - app/app.go:152-155, pipeline.go:248,306,312
- P9 - still fixed - S08 - pipeline.go:354-356, translate.go:70-79
- P10 - still fixed - S08 - cost.go:27-36
- P11 - still fixed - S08 - cost.go:27-32,52-66
- P17 - still fixed - S08 - pipeline.go:64-83
- P22 - still fixed - S08 - report/store.go:27-90, archive.go:40,103-116
- E2 - still fixed - S08 - pipeline.go:267-274, htmlgen/reader_key.go:27-30
- X1 - still fixed - S08 - rtf/parse.go:146-168,250-254,266-294
- X2 - still fixed - S08 - parse.go:130-138,191-197, codepage.go:64-112
- X3 - still fixed - S08 - parse.go:239-240,322-366
- X22 - still fixed - S08 - parse.go:101-103
- T1 - still fixed - S08 - translator/google.go:159-161,236-242
- T5 - still fixed - S08 - translate.go:138-184,213-240, ollama.go:217 (Google half: see T11)
- T8 - still fixed - S08 - cost.go:24-66
- B21 - still fixed - S08 - rtf.js:209,263,276
- B1 - still fixed - S14 - viewer.js:141-142
- B8 - still fixed - S14 - viewer.js:599-605,639-661
- B10 - still fixed - S14 - viewer.js:164-172,895-910,981,1566 (residual B39)
- B11 - still fixed - S14 - viewer.js:300-328
- B14 - still fixed - S14 - export-html.js EXPORT_CSP + buildExportHtml escaping
- B26 - still fixed - S14 - viewer.js:894 (ocr-overlay half: B36)
- B27 - still fixed - S14 - diagnostics.js recordRun/writeTail (residual B44)
- B28 - still fixed - S14 - viewer.js:502-515,1174,1217
- B15 - still fixed - S07 - manifest.json:46-67, content-security.test.mjs:28-34, page-ocr.js:50,321,327
- B29 - still fixed - S07 - build.mjs:28-33,124-143 (desktop scripts unpinned: R20)
- P12 - still fixed - S18 - register_windows.go:62-63,538-560, app.go:180-198
- P13 - still fixed - S18 - register_windows.go:462-495,614-633
- P19 - still fixed - S18 - flags.go:149-160, ollama.go:95-109
- P23 - still fixed - S18 - textutil/lines.go:24-26
- X5 - still fixed - S18 - txt/decode.go:65-77, txt.js:187-192
- X21 - still fixed - S18 - decode.go:25-54, txt.js:119-133
- X22 - still fixed - S18 - (TXT half) txt/extract.go:84-95, txt.js:220
- T1 - still fixed - S18 - google.go:161,228-243
- T2 - still fixed - S18 - google.go:101-105,125-127
- T3 - still fixed - S18 - google.go:25,175-176,247-272
- T4 - still fixed - S18 - google.go:114-123, cache.go:57-59
- T5 - still fixed - S18 - ollama.go:200-217 (gaps T15, T11)
- T6 - still fixed - S18 - retry.go:53-67
- T7 - still fixed - S18 - ollama.go:121-139 (residual T14)
- T9 - still fixed - S18 - ollama.go:369-391
- E1 - still fixed - S15 - epub/resolve.go:31-76, epub.go:362-370
- E5 - still fixed - S15 - resolve.go:36, epub.go:186-199
- E6 - one edition only - S15 - Go normalize.go:317-376 fixed; epub.js:422-424 still replaces an svg with text beside its image
- E10 - still fixed - S15 - normalize.go:95,114
- E11 - still fixed - S15 - links.go:55-153
- E12 - one edition only - S15 - Go normalize.go:178-221; epub.js:456 parses XHTML as text/html
- E16 - still fixed - S15 - epub.go:255-263,284-286,303
- E18 - still fixed - S15 - toc.go:281-283,307-308
- E20 - still fixed - S15 - epub.go:170-180
- B16 - still fixed - S15 - epub.js:399,478
- B17 - still fixed - S15 - url-policy.js:89-95
- B19 - still fixed - S15 - epub.js:513-517
- B20 - still fixed - S15 - epub.js:135-157
- B23 - still fixed - S15 - (epub) epub.js:77-84,102
- E1 - still fixed - S13 - epub/resolve.go:32-81, singlepage.go:162-169
- E2 - still fixed - S13 - reader_key.go:17-31, pipeline.go:274
- E3 - still fixed - S13 - merge.go:160-216
- E4 - still fixed - S13 - merge.go:85-129,175-178,219-234
- E13 - still fixed - S13 - navbar.go:543
- E14 - still fixed - S13 - (htmlgen side) epub/normalize.go:127, charset.go:24-55
- E17 - still fixed - S13 - singlepage.go:114,116
- E18 - still fixed - S13 - htmlgen.go:246-253
- E19 - still fixed - S13 - navbar.go:708-724, htmlgen.go:49,185,269, toc_scan.go:45,141, encode.go:10
- E21 - still fixed - S13 - favicon.go:21, assets/copier.go:183
- E22 - still fixed - S13 - htmlgen.go:72,135-149 (fallback E30)
- E23 - still fixed - S13 - navbar.go:746, singlepage.go:261
- E24 - still fixed - S13 - reader_key.go:17-21
- P10 - still fixed - S19 - pipeline/cost.go:27-34, dialog/host.go:42-46
- P18 - still fixed - S19 - flags.go:24,118, main.go:26
- P19 - still fixed - S19 - flags.go:139-160, app.go:144
- P20 - still fixed - S19 - flags.go:123-125
- B24 - still fixed - S19 - background.js:22-27
- B25 - still fixed - S19 - background.js:51-60,94-109
- P18 - still fixed - S20 - flags.go:24,118; main.go:26; flags_test.go:230
- P8 - still fixed - S17 - (htmlproc half) htmlproc.go:99-104
- E1 - still fixed - S17 - htmlsplit/split.go:63,89
- E7 - still fixed - S17 - htmlsplit/links.go:71-126 (gap E40 fixed inline)
- E8 - still fixed - S17 - chunk.go:77-97
- E9 - still fixed - S17 - chunk.go:112,126-144
- X11 - still fixed - S17 - (img half) img/extract.go:241-249
- X12 - still fixed - S17 - img/extract.go:195,203
- X13 - still fixed - S17 - img/extract.go:223-249
- X23 - still fixed - S17 - img/extract.go:310
- T2 - still fixed - S17 - htmlproc.go:56, google.go:101-127
- Q3 - still fixed - S17 - i18n_cli.go:383
- B3 - still fixed - S16 - page-ocr.js:149-151,163-169
- B5 - still fixed - S16 - page-agent.js:227-264
- B6 - still fixed - S16 - page-agent.js:216-225
- B12 - still fixed - S16 - ocr-host.js:35-84, page-ocr.js:297
- B13 - still fixed - S16 - page-ocr.js:355-368, page-agent.js:37,382
- B14 - still fixed - S16 - url-policy.js:44-47,98-104, export-html.js:10-20,35
- B15 - still fixed - S16 - page-ocr.js:44-55, ocr-host.js:73-78, ocr.js:117
- B17 - still fixed - S16 - url-policy.js:89-94
- B18 - still fixed - S16 - url-policy.js:101-102
- B27 - still fixed - S16 - diagnostics.js:28-49
- Q7 - still fixed - S05 - it cited no file ("observed in the test log"); re-checked by hand: on non-Windows `internal/pdf/pdftotext_nonwindows.go:119-122` only advises, nothing is exec'd from the Windows cache and winget is no longer suggested (X10)

## New findings

### G - GUI launcher (`cmd/doc-html-ui`)

- G20 - low - conf - golangci errcheck was red on `defer windows.CloseHandle(h)`, so `scripts/lint.ps1` failed for the package (S01) - `cmd/doc-html-ui/proc_windows.go:37`, `proc_windows_test.go:12` - inline: `defer func() { _ = windows.CloseHandle(h) }()` in both; `golangci-lint run ./cmd/doc-html-ui/ ./internal/outputpath/` exit 0, `go test -count=1 ./cmd/doc-html-ui/` exit 0
- G21 - low - conf - `outcome` checked `cancelled` before a clean exit, so a Cancel that arrived after the converter had already exited 0 reported a complete output as cancelled (S01) - `cmd/doc-html-ui/run.go:197-209` - inline: clean exit wins; new `TestOutcomeCleanExitWinsOverLateCancel`, `go test -count=1 ./cmd/doc-html-ui/` exit 0
- G22 - low - conf - the log relay buffers up to 256 lines of up to 8 MiB each: about 2 GiB worst case on the 386 build if the page reads slowly and the child prints multi-MiB lines (S01) - `cmd/doc-html-ui/run.go:37,315,399` - -
- G23 - low - conf - `openAppWindow` returns after the first Edge or Chrome it finds even when starting it failed, without logging; the user gets no window and the server exits through the watchdog (S01) - `cmd/doc-html-ui/main.go:1186-1189,1197-1199` - -
- G24 - low - plaus - `handleDeleteOutput` checks only the CLI's lock, not the GUI's own run registry, so a delete during a just-claimed run's start-up window can remove the folder the child is about to reuse (S01) - `cmd/doc-html-ui/main.go:569-578`, `run.go:229-234` - -
- G25 - low - conf - Convert has no re-entry guard before its awaits: a second click during the key or status fetch gets 409, and its `finally` re-enables Convert and drops `runAbort`, so the running job can no longer be cancelled (S02; residual of ticket 23 APP-BEHAVIOUR rule 3) - `cmd/doc-html-ui/ui.html:1505-1536,1587,1593-1615` - -
- G26 - low - conf - `saveSettings` ignores `r.ok` and swallows errors; a 500 (store busy, rename refused) silently loses the settings (S02) - `ui.html:1127-1133`, `store.go:46`, `main.go:302-305` - -
- G27 - low - conf - `readSettings` drops `withFileLock` errors and a failed `setAsideCorrupt`, so a corrupt settings file is served as `{}` without `X-Settings-Corrupt` and overwritten by the next save (S02; residual of G12) - `cmd/doc-html-ui/store.go:114-118` - -
- G28 - low - plaus - the saved OCR language is lost when `/api/ocr-langs` answers before `/api/settings` (`dataset.want` is never applied) (S02) - `ui.html:1117,1157,1169,1178-1191` - -
- G29 - low - conf - a queued cost question outlives its run and posts `/api/answer` for a run that has ended (S02) - `ui.html:659-687,1612,1636` - -
- G30 - low - conf - `fileUriToPath` throws on a malformed `%` in dropped `text/plain`, and the drop silently does nothing (S02) - `ui.html:774,788,822-828` - -
- G31 - low - conf - the `store.go` header described an output history and a read-modify-write cycle that no longer exist (S02) - `cmd/doc-html-ui/store.go:14-19` - inline: comment rewritten; `go test ./cmd/doc-html-ui/` exit 0

### R - release path (scripts, workflows, installer, MSIX, winget source, extension publish)

- R1 - high - conf - `check.ps1 -Plan <subset>` writes release-grade gate evidence (code 0 and the real tree hash, not the plan), and `release.ps1` checks only the code and the tree, so running one child turns the tag line green (S04) - `scripts/check.ps1:50,121-131`, `scripts/release.ps1:63-71` - #36
- R2 - med - conf - the documented build-local flow can never produce matching evidence: check hashes tree T, then `build.ps1` rewrites the tracked `build/doc-html-translate.exe` and the commit is amended with `DEV/COMMIT_LOG.md`, so `HEAD^{tree}` is never T and the BLOCKED hint loops (S04) - `scripts/release.ps1:56,66,77`, `scripts/build.ps1:69`, `scripts/build-local.ps1:41-100` - #36
- R3 - med - conf - `check.ps1` hashes the tree after its children finish, so an edit made during the run is recorded as tested (S04) - `scripts/check.ps1:59-72,125` - #36
- R4 - med - plaus - `commit-push.ps1` says it stays "local + free", but `git add -A` plus a push to `origin main` with no gate publishes the Pages site (S04) - `scripts/commit-push.ps1:13-15,47,65` - #36
- R5 - med - conf - the typo gate FAILs at HEAD (85 of 89 hits in non-English i18n, RTF and translator files `configs/.typos.toml` does not exclude), so `check.ps1` exits 1 and the release gate cannot go green (S04) - `configs/.typos.toml` - #36
- R6 - low - conf - `build.ps1` throws before its cleanup on a goversioninfo or `go build` failure, leaving `resource.syso` and `versioninfo.generated.json`, which are not gitignored (S04) - `scripts/build.ps1:55-56,70-76` - -
- R7 - low - conf - `build.ps1` leaves `$env:GOARCH` / `$env:GOOS` set in the caller's session (S04) - `scripts/build.ps1:67-68` - -
- R8 - low - conf - the `eng.traineddata` fallback in `build.ps1` is downloaded unverified and a partial file sticks (S04; the "Not done" of done/13) - `scripts/build.ps1:108,116` - #38
- R9 - low - plaus - `DEV/private/google_api.key` is copied into the deploy folder `C:\GD\tc\SZA\_APP`, which is likely cloud-synced (S04) - `scripts/build.ps1:90-93` - -
- R10 - low - conf - the verify-html "blank render" check tests only that the screenshot is at least 200x200 px, never its pixels (S04) - `scripts/verify-html.ps1:193-201` - -
- R11 - low - conf - render and sitemap drift is compared with case-insensitive `-ne` (S04) - `scripts/doc-registry.ps1:241`, `scripts/security-posture.ps1:358` - -
- R12 - low - conf - `contract-gate.ps1` reports PASS after checking zero contracts, a vacuous pass (S04) - `scripts/contract-gate.ps1:153-156,211-214` - #36
- R13 - low - plaus - registry product rows are matched by prefix (`-notlike "$Product*"`) (S04) - `scripts/contract-gate.ps1:112,124` - -
- R14 - low - conf - `release.ps1 -Version` is not validated against `YY.MMDD.HHmm`, and the default is recomputed on every run (S04) - `scripts/release.ps1:27-50` - #37
- R15 - low - conf - the parity-check working-tree fallback ignores untracked files (S04) - `scripts/parity-check.ps1:81-84` - -
- R16 - low - conf - a pipe in `-Note` / `-Ref` breaks the release-state table (S04) - `scripts/release-state.ps1:81,119` - -
- R17 - low - conf - verify-exe-version treats a crashing `-version` as "(not run)" rather than a problem (S04) - `scripts/verify-exe-version.ps1:69-72,82` - -
- R18 - high - plaus - `release.yml` takes `workflow_dispatch` with a free-text tag; checkout has no `ref:`, so the dispatched branch is built and released under that tag (the release action creates the tag if missing), nothing checks HEAD against the tag, and `inputs.tag` is pasted raw into the script - the path skips the gate evidence entirely (S07) - `.github/workflows/release.yml:6-10,33-35,44,234-236` - #37
- R19 - med - conf - the installer and MSIX builds stamp the current time and build the working tree without a clean-tree or HEAD-equals-tag check; `release.ps1:157-158` runs the installer build without `-Stamp` and then uploads `setup-<tag version>.exe`, a file that does not exist (S07) - `scripts/build-installer.ps1:57-66`, `msix/build-msix.ps1:63-71` - #37
- R20 - med - conf - when the vendored `eng.traineddata` is missing, the installer and UI builds download tessdata_fast `raw/main` with no version pin, digest or timeout, and a failed download still builds an installer without English OCR data (S07) - `scripts/build-installer.ps1:120-134`, `scripts/build-ui.ps1:101-112` - #38
- R21 - med - plaus - neither the MSIX staging nor the CI zip carries `tessdata/eng.traineddata`, although AGENTS.md and `tessdata.go` say English data ships in `<exe>/tessdata` (S07) - `msix/build-msix.ps1:112-121`, `.github/workflows/release.yml:153-167` - #38
- R22 - med - conf - `build-msix.ps1 -IdentityName` defaults to the winget id `SerZhyAle.DocHtmlTranslate`, not the frozen MSIX identity `SZA.Doc-HTML-Translate`, and `release.ps1:177` / `RELEASE.md:96` pass a placeholder (S07) - `msix/build-msix.ps1:33` - #37
- R23 - low - conf - the release-notes range uses `git describe --tags "$tag^"`, which matches `ext-cws-` / `ext-edge-` tags (checked: `v26.0912.2026` resolves to `ext-cws-v26.0815`), so notes leave out commits (S07) - `.github/workflows/release.yml:200` - #37
- R24 - low - conf - `softprops/action-gh-release@v2` runs with `contents: write` pinned only by a movable tag (S07) - `.github/workflows/release.yml:12-13,234` - #37
- R25 - low - conf - the extension publish workflows accept `workflow_dispatch` from any branch with no tag or confirmation, and have no `permissions:` block while `npm ci` runs install scripts (S07) - `.github/workflows/publish-cws.yml:16`, `publish-edge.yml:16` - #37
- R26 - low - conf - the installer's "openwith" task is checked by default, so a `/SILENT` install writes Open-with and right-click entries the README says wait for the user's yes (S07) - `installer/doc-html-translate.iss:98` - #23
- R27 - low - conf - `build-local.ps1` logs the commit hash and then amends, so the logged hash never exists (S07) - `scripts/build-local.ps1:82,91,100-103` - -
- R28 - low - conf - `build-ui.ps1` cleanup is not in a `finally`: an amd64 `resource.syso` left in `cmd/doc-html-ui` breaks 386 builds, and GOARCH/GOOS stay set (S07) - `scripts/build-ui.ps1:55-79` - -
- R29 - low - conf - `bump-version.mjs` does semver bumps against the date-stamp policy, and `release.ps1:113` still suggests it (S07) - `extension/scripts/bump-version.mjs:15-21` - -
- R30 - med - conf - the slicer put `extension/scripts/_lib.mjs` in class B and `extension/package.json` in no class, so the publish scripts' HTTP layer and the npm build chain were in no slice (S07) - `scripts/audit-slices.ps1:131-132,262` - inline: both are class A; `go test -count=1 -run TestAuditSlices ./tests/` exit 0; tail slice S22 cut with `-Tail` and read
- R31 - low - conf - `-Summary` took `#N` from the whole finding line, so a colour such as `#808080` would count as a ticket (S07) - `scripts/audit-slices.ps1:709` - inline: only the ticket field is scanned; `TestAuditSlices` exit 0
- R32 - low - conf - `-Summary` printed only the count of missing re-checks, not the ids (S07) - `scripts/audit-slices.ps1:717` - inline: the ids are listed; `TestAuditSlices` exit 0
- R33 - low - plaus - the installer build calls `generate-icon.ps1`, which also redraws committed favicons and ICOs (S07) - `scripts/build-installer.ps1:77` - -
- R34 - med - conf - all 13 winget locales say "OCR - English bundled", but the winget zip ships neither Tesseract nor `eng.traineddata` (S21) - `winget/SerZhyAle.DocHtmlTranslate.locale.en-US.yaml:21` and the 12 other locales - #38
- R35 - low - conf - the winget listing leaves out comic archives and image input in every locale (S21) - `winget/*.locale.*.yaml:15-25` - #38
- R36 - low - plaus - the tracked `winget/manifests/` archive inside `winget/` breaks the documented `winget install --manifest winget` gate ("Subdirectory not supported", met in the v26.0912.2026 release), and RELEASE.md does not say so (S21) - `DEV/RELEASE.md:84,88` - #37
- R37 - low - conf - `httpText` has no timeout, so a hung store call waits for the 15-minute CI job timeout (S22) - `extension/scripts/_lib.mjs:58-59` - -
- R38 - low - conf - `npm run release:cws|edge` is build plus publish with no `npm test` (S22) - `extension/package.json:19-20` - -

### P - CLI, pipeline, app wiring, support packages

- P25 - low - conf - `askYes` reads with `fmt.Scanln`, so leftover words answer the next prompt: "n yy" registers the default handler unconfirmed (S03) - `internal/app/app.go:296-306` - #47
- P26 - low - plaus - `pruneOldSets` deletes other hash folders a concurrently running build may use, and `PDFToTextPath` checks only the exe, not its DLLs (S03) - `internal/bundledtools/cache.go:80,126-138`, `pdftotext_windows.go:35-38` - -
- P27 - low - conf - unchecked `defer windows.CloseHandle` in `processAlive` turned lint red (S05) - `internal/outputpath/proc_windows.go:18` - inline: `defer func() { _ = windows.CloseHandle(h) }()`; golangci exit 0, `go test -count=1 ./internal/outputpath` exit 0
- P28 - low - conf - the stale-lock takeover removes by path, so two runs that saw the same stale lock can both hold it; a live run over 24 h loses its lock because age is tested before liveness (S05) - `internal/outputpath/lock.go:268-271,295-313` - -
- P29 - low - conf - `RunLogf` bypassed `runLogFilter`, so unredacted panic values and paths reached the on-disk run log (S06) - `internal/logging/log.go:90-100` - inline: the filter applies; new `runlogf_filter_test.go`, `go test -count=1 ./internal/logging` exit 0
- P30 - low - plaus - helper processes (Calibre, 7-Zip, pdftotext, the JPX converter, Tesseract) run under `context.Background()`, so a run-context cancel does not stop them before their deadline, and an extraction killed by Ctrl+C reports ExitParse instead of interrupted (S06, S08, S11; the deferral recorded in done/11) - `internal/mobi/extract.go:64`, `comic/readers_sevenzip.go:70,209`, `pdf/extract.go:102`, `pdf/images.go:392`, `ocr/run.go:24`, `pipeline/pipeline.go:150-249` - #47
- P31 - low - conf - `CopyCapped` dropped the probe-read error, so an entry of exactly `limit` bytes failing at EOF (zip `ErrChecksum`) passed as clean (S06) - `internal/limits/limits.go:90-101` - inline: a non-EOF probe error is returned; new `copycapped_probe_test.go` (a real bad-CRC zip), `go test -count=1 ./internal/limits` exit 0; the JS twin has no such swallow
- P32 - low - conf - `TestRunMissingBinaryKeepsThePathError` demanded `*os.PathError`, but Windows returns `*exec.Error`, so procrun's tests were red on the product platform (S08; noted in 32 and done/31) - `internal/procrun/procrun_test.go:47-58` - inline: either start error is accepted; `go test ./internal/procrun/` exit 0 on 386 and amd64
- P33 - low - plaus - the helper joins its job object only after `cmd.Start()`, so a grandchild spawned in that window escapes the kill (S08) - `internal/procrun/tree_windows.go:24-49`, `procrun.go:125-131` - -
- P34 - med - conf - on a console run the GUI does not host, `ShowWarning` is a modal MessageBox, so the JPX and blocked-pdftotext warnings halt `-noopen` and batch runs until someone clicks (S19) - `internal/dialog/dialog_windows.go:71-78`, `pdf/images.go:294`, `pdf/pdftotext_windows.go:44` - #47
- P35 - low - conf - three refusals are hard-coded English (the `-max-cost` refusal, "input file is required", "unexpected extra argument") while their siblings go through `i18n.T` (S19) - `internal/config/flags.go:124,203,211` - -
- P36 - med - conf - a failed CLI run's final error goes only to stderr, and most pipeline return paths do not log it, so the run log and the `-report` bundle do not say why the run failed (S20) - `cmd/doc-html-translate/main.go:30,56` - #47
- P37 - low - conf - `iconart` `parsePath` looped forever on numbers after a closepath (`Z 5 5`), an OOM in about a second (dev-tool input only) (S17) - `internal/iconart/path.go:50-58` - inline: rejected; new `robust_test.go`, outputs byte-identical, `go test ./internal/iconart/` exit 0
- P38 - low - conf - `DecodeICO` sliced an unchecked directory and added `off+size` in uint32, panicking on a short ICO (tests and the tool only) (S17) - `internal/iconart/outputs.go:214-224` - inline: length check and a 64-bit add; `go test ./internal/iconart/` exit 0

### E - EPUB, HTML processing, generation, split, htmlconv, md, assets

- E25 - low - conf - `Copier.place`'s error path removed the last recorded copy instead of the failed stylesheet's, so a later reference could return a copy that was never written (S03) - `internal/assets/copier.go:122,133` - inline: removed by its recorded index; new `TestFailedSheetIsForgottenNotItsLastImport` (exit 1 before), `go test -count=1 ./internal/assets ./internal/htmlconv ./internal/md` exit 0
- E26 - low - conf - Go `splitBySections` splits at nested `<h1>`/`<h2>` (in a blockquote or list), leaving unbalanced pages; `md.js` splits only at top level (S06) - `internal/md/extract.go:115-147`, `extension/src/md.js:23-31` - #45
- E27 - low - plaus - Markdown is read whole and rendered with no input size budget, so a huge file ends the 386 build with a fatal OOM (S06) - `internal/md/extract.go:27,37` - #47
- E28 - low - conf - HTML input language: Go takes `xml:lang` and `<body lang>` as found, the extension only `<html lang>`, normalized (S09) - `internal/htmlconv/extract.go:184-208`, `extension/src/html.js:15` - #45
- E29 - low - conf - `docs/PARITY.md:1066` names `copyLocalImages`, now `internal/assets` `Copier` (S09) - `docs/PARITY.md:1066` - #45
- E30 - med - plaus - the merged page and the TOC index fall back to `<html lang="en">` when the first page declares none (Markdown deliberately declares none; EPUB `dc:language` is never read), which can suppress Chrome's Translate offer (S13) - `internal/htmlgen/singlepage.go:51`, `htmlgen.go:136` - #40
- E31 - high - conf - the default single-page merge keeps only each page's `<body>` children, dropping every `<head><style>` and body attribute: PDF `.pdf-flip-y` goes (flipped images render mirrored) with the float layout, FB2 loses its stanza and subtitle styles, EPUB its head CSS (S13; proved with a throwaway merge test: class kept, rule dropped) - `internal/htmlgen/singlepage.go:72-79,98-103`, rule source `internal/pdf/extract.go:426,778` - #39
- E32 - low - conf - the index's "Chapters: N" is an interface string with no `lang`/`dir` of its own inside a page in the book's language (S13) - `internal/htmlgen/htmlgen.go:111` - -
- E33 - low - conf - the index reads `dir` only from `<html>` while the merge falls back to `<body dir>`, so a body-only RTL book gets an LTR index (S13) - `internal/htmlgen/htmlgen.go:170`, `singlepage.go:275-288` - #40
- E34 - med - conf - the XHTML-to-HTML rename has no collision check: `a.xhtml` beside `a.html` overwrites the other chapter and both manifest items land on one href (S15; proved with a throwaway test) - `internal/epub/normalize.go:92-101,160` - #47
- E35 - low - conf - a chapter whose normalization fails keeps `.xhtml` while other chapters' links already point at the never-written `.html` (S15) - `internal/epub/normalize.go:57-59,99-101` - -
- E36 - low - conf - Go TOC hrefs skip `resolveBookPath`: root-relative, backslash and `?query` targets are dropped in Go and resolved in JS, unrecorded in PARITY (S15) - `internal/epub/toc.go:311-319`, `extension/src/epub.js:347-349` - #45
- E37 - low - conf - the Go link rewrite skips root-relative `/..` links (they point at the drive root under `file://`), JS resolves them to the book root (S15) - `internal/epub/links.go:125`, `extension/src/epub.js:446` - #45
- E38 - low - plaus - extraction containment is a prefix check only; zip names with DOS device segments or NTFS stream colons are not refused (`filepath.IsLocal` would) (S15) - `internal/epub/epub.go:270-276` - -
- E39 - low - conf - PARITY "Input limits" says symlink entries are skipped; JS keeps them as files, Go counts them and then fails each (S15) - `extension/src/epub.js:65-83`, `internal/epub/epub.go:255-262,281` - #45
- E40 - low - conf - `rewriteTOC` looked fragments up as written, so a percent-encoded (Cyrillic) fragment kept pointing at part 1 after a split (S17) - `internal/htmlsplit/links.go:78-84` - inline: decoded for the lookup; new `toc_fragment_test.go` (failed before), `go test ./internal/htmlsplit/` exit 0
- E41 - low - conf - a split part is written to `<base>_sN.html` without checking the manifest, so a book's own `ch_s2.html` is overwritten before it is read (S17; extends done/01) - `internal/htmlsplit/split.go:84-90` - -

### X - format extractors

- X25 - high - conf - `isLigaturesArtifact` (4 or more words averaging under 3 letters) drops real short text on the pdftotext path ("Text of page 2", "Is it so? I do."), and `TestExtract_Volume3ImagesOnTheirPages` fails wherever the vendored pdftotext exists, so `scripts/test.ps1` is red on the owner's machine; the Go pdflib path has no filter, JS filters per row (S05) - `internal/pdf/extract.go:306,386-396`, `extension/src/reflow.js:48-53,133` - #43
- X26 - med - conf - Flate DeviceCMYK / Indexed-CMYK rasters are written as `.tif` and linked as `<img>`, which Chrome cannot display; they are also flipped on disk and again by CSS `.pdf-flip-y`, and OCR reads the flipped raster (orientation part plaus) (S05) - `internal/pdf/images.go:351,401-437`, `extract.go:426,778,867-873` - #43
- X27 - med - plaus - Go `betterPageRaster` prefers the raster without a stencil `/Mask` (the MRC fix), JS `dedupeSameShape` keeps the largest, and PARITY is silent and places `selectPageImages` in the wrong file (S05) - `internal/pdf/images.go:249-254`, `extension/src/pdf-images.js:141-155`, `docs/PARITY.md:342` - #43
- X28 - low - conf - the JPEG2000 warning is hard-coded English and can show twice per PDF, because `extractImages` runs again on the pdflib fallback (S05) - `internal/pdf/images.go:293-303` - -
- X29 - low - conf - on 386, RTF `readParam` builds 10 digits into a 32-bit int: `\bin2147483647` overflows `r.pos+param` and panics, and the user gets "internal error", exit 3 (S08; executed) - `internal/rtf/parse.go:29,134-135,155-158` - #47
- X30 - med - conf - Go FB2 pages hardcode `<html lang="en">` and ignore `<title-info><lang>`, which `fb2.js` reads; the same literal is in txt, rtf, pdf, img and comic (S09) - `internal/fb2/extract.go:129` - #40
- X31 - low - conf - FB2 binary ids are not checked against generated names: an id `page_001.html` is overwritten by the page, `index.html` / `favicon.ico` likewise, and `.exe` / `.lnk` ids land in the output (S09) - `internal/fb2/extract.go:62,183-210` - -
- X32 - low - conf - an unpadded or `\f` / `\v`-bearing FB2 base64 binary fails Go's strict decode (placeholder), while the browser decodes it (S09) - `internal/fb2/content.go:229,235,246`, `extension/src/fb2.js:85` - -
- X33 - low - plaus - image-input pages hardcode `lang="en"` whatever the scan's language (S17) - `internal/img/extract.go:298` - #40
- X34 - med - conf - TXT paragraphs differ between editions: Go maps `\f`, `\v`, U+0085, U+2028, U+2029 to newlines and `txt.js` does not ("l1\nl2\n\f\nl3\nl4": Go 2 paragraphs, JS 4) (S18; run) - `internal/textutil/lines.go:28-38`, `internal/txt/extract.go:163`, `extension/src/txt.js:11` - #45

### T - translation engines

- T10 - low - conf - `-google` without a key converts and exits 0, while an unreachable Ollama exits 4: two exit codes for the same "requested engine unavailable" (S08) - `internal/pipeline/translate.go:99-107`, `pipeline.go:354-358` - #41
- T11 - low - conf - `GoogleClient.Translate` returns `nil, err` when a later batch fails, discarding batches already paid for; the cache stores nothing and a rebuild bills them again (the Google half of T5) (S08, S18) - `internal/translator/google.go:83-89` - #41
- T12 - low - conf - `ReplaceTexts` restored only ASCII edge whitespace while `ExtractTexts` trims Unicode spaces, so "Chapter&nbsp;<b>1</b>" came back glued (S17) - `internal/htmlproc/htmlproc.go:84-93` - inline: `unicode.IsSpace` trims; new `whitespace_test.go` (failed before), `go test ./internal/htmlproc/ ./internal/pipeline/` exit 0
- T13 - high - conf - translation runs after navbar injection and htmlproc skips only script/style/code/pre, so every page's reader chrome (file name, title, Previous/Next/TOC, theme and font option labels, version, "N / M") is sent to the paid engine and comes back translated inside a bar marked with the interface language (S17) - `internal/htmlproc/htmlproc.go:19-24,51-68`, `internal/htmlgen/navbar.go:580-621`, `internal/pipeline/pipeline.go:295,311` - #41
- T14 - low - conf - the Ollama reply regex's `\s*` after "N." crosses a newline, so an empty "2." takes line 3's text into slot 2 (S18; run) - `internal/translator/ollama.go:409` - inline: `[ \t]*`; new `ollama_parse_test.go`, `go test ./internal/translator/` exit 0
- T15 - med - conf - an Ollama slot still empty after the echo retries counts as success (the retry error is dropped): the run says "Translation complete", the cache stores "" and the segment stays in the source language (S18) - `internal/translator/ollama.go:242-261`, `cache.go:70-78`, `internal/pipeline/translate.go:224-240` - #41

### O - OCR (desktop)

- O13 - low - conf - `recognizePaths` called `poolWorkers(paths)` in the worker-loop condition, re-reading every image header once per started worker (about 8 000 opens on a 480-page comic) (S10) - `internal/ocr/overlay.go:407-410` - inline: hoisted; `go test -count=1 ./internal/ocr/` exit 0
- O14 - low - plaus - `localImageFile` joins an `<img src>` onto the page folder with no containment, so `../../..` in book content makes OCR read any local image at a known path and write its text into the output (S10) - `internal/ocr/overlay.go:333-350` - -
- O15 - med - plaus - the "ASCII-safe" Tesseract paths (the temp staging and, since O4, `--tessdata-dir` under `%LOCALAPPDATA%`) are under the user profile and never checked, so a profile path outside the ANSI code page loses OCR on upscale, rotate, rescue and screen passes, and on every run after a pack download (S11) - `internal/ocr/tesseract.go:297-298,686,713`, `tessdata.go:59-65`, `script.go:128` - #46
- O16 - med - conf - the GUI always sends `-ocr-lang` (the select falls back to `eng`), so the ticket-30 script check never runs in the GUI, the MSIX / Store entry point (S11) - `cmd/doc-html-ui/main.go:916-917`, `ui.html:1157-1168`, `internal/pipeline/ocrstep.go:54`, `internal/ocr/script.go:139` - #46
- O17 - low - conf - `TessLang` passes unmapped `-src` codes through (cs, sv, hi, zh-CN, pt-BR) and maps nl/tr/ar to packs not in the catalog; the advice "-ocr-download <code>" is then refused by `CheckLang` (S11) - `internal/ocr/tessdata.go:194-211`, `download.go:124` - #46
- O18 - low - conf - `DataDir` returns the per-user folder even when copying a bundled pack failed; `--tessdata-dir` is then dropped and every image fails with an engine error that does not name the cause (S11) - `internal/ocr/tessdata.go:114-118`, `tesseract.go:175-177,297` - #46

### B - browser extension

- B30 - low - conf - the extension's ZIP lister does not skip symlink entries; Go does, and PARITY says both do (S03) - `extension/src/comic.js:191-222`, `internal/comic/readers_zip.go:20` - #45
- B31 - low - conf - TAR name precedence drift: in Go the GNU `L` long name wins over PAX `path=`, in JS the reverse (S03) - `extension/src/comic.js:299-302` - #45
- B32 - low - conf - with no signature match, Go falls back to the extension's container (`.cbz` means ZIP) and JS always to TAR, so a prefixed ZIP `.cbz` fails only in the extension (S03) - `extension/src/comic.js:111-116`, `internal/comic/container.go:57-61` - #45
- B33 - low - conf - `ebook.js` strips a document-supplied `data-dht-target` only when the section has book links; otherwise it survives the sanitizer and becomes `href="#.."` (S06) - `extension/src/ebook.js:51,61,64-69` - #45
- B34 - med - conf - the extension drops an FB2 stanza's own `<title>` / `<subtitle>`, which Go keeps (S09; run on one fixture) - `extension/src/fb2.js:144-147`, `internal/fb2/content.go:128-173` - #45
- B35 - low - conf - the extension never shows the FB2 cover and drops an `<image>` with no binary, where Go shows the cover and a placeholder (S09) - `extension/src/fb2.js:100-101,183` - #45
- B36 - low - plaus - `fetchToBlob` sends no credentials, so context-menu OCR of a login-gated image fails 401/403 (the half of B26 ticket 19 did not cover) (S12) - `extension/src/ocr-overlay.js:117` - -
- B37 - low - conf - ImageBitmaps are not closed on error paths (S12) - `extension/src/ocr-overlay.js:301-308,466-473,644-651,667-669` - -
- B38 - low - conf - `isTranslatable`'s CJK class: Go uses Unicode script tables, JS BMP-only ranges without the `u` flag; halfwidth katakana, compatibility jamo and astral Han are kept by Go and rejected by JS (S12; probed) - `extension/src/ocr-text.js:8`, `internal/ocr/text.go:17-20` - #45
- B39 - med - conf - `renderDocument` checks the load token once, so a superseded PDF load still runs language detection, `applyLang`, `warnIfNoText` and the Export unhide against the new document, overwriting its `<html lang>` (S14; a gap in ticket 18's B10) - `extension/src/viewer.js:1319-1320,1326,1337-1342,1692` - #44
- B40 - med - conf - the HTML export re-encodes every `blob:` image as JPEG, so a transparent PNG (diagrams, formulas, line art) exports as a black box (S14) - `extension/src/viewer.js:651` - #44
- B41 - med - conf - toolbar state carries over between documents: a TOC button hidden for a TOC-less document is never shown again, and the OCR group is never re-hidden (S14) - `extension/src/viewer.js:218,718-720` - #44
- B42 - low - conf - PARITY says `PAGE_CHUNK` 50 / `CHUNK_LEAD` 2 (the code has 100 / 5) and cites stale viewer lines for the font families (S14) - `docs/PARITY.md:286,1084-1085` - #45
- B43 - low - conf - text-size range and step differ between editions (viewer 12-40 px in 1 px steps, desktop 70-300 % in 10 % steps), unrecorded in PARITY (S14) - `extension/src/viewer.js:1715,1719`, `internal/htmlgen/navbar.go:428` - -
- B44 - low - conf - `askPassword` records "Password required" as the run's error and a successful unlock never clears it (S14; the B27 class) - `extension/src/viewer.js:468,1274` - -
- B45 - low - conf - `nav#toc`'s `aria-label` stays English in every interface language (S14) - `extension/src/viewer.html:59` - -
- B46 - med - conf - the extension halves of E6, E12 and E14 have no open ticket: XHTML is parsed as text/html and swallows chapter text, the decoder is UTF-8 only (windows-1251 chapters garble), an SVG cover with text loses the text (S15; done/06 "Not done", PARITY open gaps) - `extension/src/epub.js:183-185,422-424,456`, `extension/src/html.js:11` - #45
- B47 - low - plaus - `convertSvgImage` puts an HTML `<img>` inside `<svg>` for multi-image SVGs, which is not drawn (S15) - `extension/src/epub.js:424` - -
- B48 - low - conf - `parseContainer` falls back to the first rootfile of any type, where Go fails the book (S15) - `extension/src/epub.js:208`, `internal/epub/epub.go:324-334` - #45
- B49 - high - plaus - page OCR collects every `<img>` src the page lists, including `file:`, intranet http and never-loaded images, fetches it with the extension's `<all_urls>` access, and writes the recognized text into the page's DOM where the page's scripts read it; the page can also trigger scans by clicking the hidden rescan button (S16) - `extension/src/page-agent.js:93-96,101-108,349`, `page-ocr.js:182`, `ocr-overlay.js:117` - #42
- B50 - med - conf - the page agent's `teardown()` never removes its `runtime.onMessage` listener, so Remove plus Start leaves two agents (repro: two listeners, two bars) and the stale one answers first (S16) - `extension/src/page-agent.js:452-474` - #42
- B51 - med - conf - ids are prefixed but `url(#id)` references (fill, stroke, clip-path, mask, filter, marker), `usemap` (the map's name is removed), `label for` and `aria-*` are not rewritten, so inline-SVG gradients and image maps break in the viewer (S16) - `extension/src/url-policy.js:89-94,127` - #44
- B52 - low - plaus - remote-content parking misses SVG presentation `url()` attributes and `<template>` content (a declarative shadow root revives a remote image in the export) (S16) - `extension/src/url-policy.js:18,26-28`, `export-html.js:12` - -
- B53 - low - conf - `detectLang` returns `zh` for kanji-dense Japanese (S16) - `extension/src/lang.js:24-26,36-38,79-82` - -
- B54 - low - conf - the OCR page, the page agent and the broker's failure text read only `chrome.i18n`, ignoring the chosen interface language (S16) - `extension/src/ocr.js:16-21`, `page-agent.js:46-51`, `page-ocr.js:227-230` - -
- B55 - low - conf - a host frame that never announced itself stays parked in the page until Remove (S16) - `extension/src/page-ocr.js:141,279-283` - -
- B56 - low - conf - the OCR pair in `configs/parity-map.json` omits `ocr-screen.js`, the twin of `screen.go` (S16) - `configs/parity-map.json:14` - #45
- B57 - low - plaus - `urlKind` treats `/\host` and `\/host` as relative, though under `file:` they are network-path links (S16) - `extension/src/url-policy.js:39` - -
- B58 - med - conf - the popup's "On this site" label carries `data-i18n` on the element that wraps `<span id="host">`, so `applyI18n` detaches the span and the host name is never shown (S19) - `extension/src/popup.html:59`, `popup.js:11,110,131` - #44
- B59 - med - conf - the per-site switch saves the active tab's host while DNR excludes by the request's domain: on the viewer tab the "host" is the extension id, and a PDF on host B linked from site A is still intercepted after A is switched off (S19) - `extension/src/popup.js:100-106,147-155`, `background.js:77-78` - #44
- B60 - low - plaus - `openOriginal` reuses the tab's bypass rule id, so a second press within 5 s has its allow rule removed by the first timer (S19) - `extension/src/background.js:233-257` - -
- B61 - low - plaus - the context-menu `*.pdf` pattern does not match links with a query string (S19) - `extension/src/background.js:33-36` - -
- B62 - low - conf - the options page lists interception formats without cbz/cbt, and "None" / "remove" are untranslated (S19) - `extension/src/options.html:35`, `options.js:117,124` - -
- B63 - low - conf - `open-original` always answers `{ok:true}`, so the viewer's fallback never runs when the rule update fails (S19) - `extension/src/background.js:249-251,264` - -
- B64 - med - conf - the extension's `orderColumns` always uses the ordinary confidence floor (50) where Go uses the pass floor (80 on rescue), so column chaining on rescue passes happens in the extension only (S12) - `extension/src/ocr-overlay.js:377`, `internal/ocr/tesseract.go:1347` - #45
