# Phase 03 - CLI, converted page, and the `-ui-lang` flag

**Strategic spec:** [`../2026-07-28_thirteen-ui-languages.md`](../2026-07-28_thirteen-ui-languages.md)
**Tactical index:** [`INDEX.md`](INDEX.md)
**Status:** ✅ Done
**Depends on:** Phase 02
**Steps done:** 9 / 9

## Objective

Move every Go user-visible string onto `internal/i18n`, ship all 13 languages in the console output and the
converted page chrome, and add the `-ui-lang` flag that selects them.

## Prerequisites

- [ ] Phase 02 ✅ Done.
- [ ] Working tree clean or on a feature branch.

## Files touched

| File | New / Modified | Line budget |
|------|:--------------:|------------:|
| `internal/config/flags.go` | Modified | ≤ 200 |
| `internal/app/app.go` | Modified | ≤ 260 (shrinks: two splash funcs leave) |
| `internal/app/splash.go` | New | ≤ 60 |
| `internal/app/splash/<code>.txt` × 13 | New | ≤ 60 each |
| `internal/i18n/i18n_cli.go` | New | ≤ 200 |
| `internal/i18n/i18n_reader.go` | New | ≤ 120 |
| `internal/htmlgen/navbar.go` | Modified | ≤ 700 (backup first) |
| `internal/htmlgen/htmlgen.go` | Modified | ≤ 420 |
| `internal/syslocale/locale_windows.go` | Modified | ≤ 90 |
| `internal/syslocale/locale_nonwindows.go` | Modified | ≤ 30 |
| `cmd/doc-html-ui/main.go` | Modified | ≤ 1060 |
| `tests/smoke_test.go` | Modified | ≤ 400 |

> `internal/htmlgen/navbar.go` is 660+ lines - back it up before editing.

## Steps

### Step 03.1 - Add the `-ui-lang` flag

**Files:** `internal/config/flags.go`
**Depends on:** - start of phase

**Prompt for developer:**
> Add `UILang string` to `Config` and the flag `-ui-lang` (default `""`, help text: "interface language for
> the console output and the converted page (default: Windows UI language); one of: en ru uk de it es fr pt
> ar hi bn ur zh"). Reject an unknown non-empty value with an error naming the accepted codes; build the
> list from `i18n.Codes` so it can never drift from the layer.

**Verification:**
- `fs.String("ui-lang"` matches exactly once.
- `UILang` is a field of `Config` and is assigned in the returned struct.
- `i18n.Codes` is referenced in the validation path.
- `go test ./internal/config -count=1` exits 0.

**Status:** `[x] done`

---

### Step 03.2 - Resolve the language once at start-up

**Files:** `internal/app/app.go`
**Depends on:** Step 03.1

**Prompt for developer:**
> In `New(cfg config.Config)`, call `i18n.SetLanguage(i18n.Resolve(cfg.UILang, "", syslocale.Lang()))`
> before anything prints. Everything downstream - console output and `internal/htmlgen` - reads
> `i18n.Language()`, so this is the only resolution site in the process.

**Verification:**
- `i18n.SetLanguage(i18n.Resolve(` matches exactly once in `internal/app/app.go`.
- `go build ./...` exits 0.

**Status:** `[x] done`

---

### Step 03.3 - Move the splash to embedded resources (en, ru)

**Files:** `internal/app/splash.go`, `internal/app/splash/en.txt`, `internal/app/splash/ru.txt`, `internal/app/app.go`
**Depends on:** Step 03.2

**Prompt for developer:**
> Create `internal/app/splash/en.txt` and `ru.txt` holding the exact text `printSplashEN` / `printSplashRU`
> print today (rule lines included as a `{{rule}}` placeholder or as literal `=` runs - pick one and
> document it in `splash.go`). Add `splash.go` with `//go:embed splash/*.txt` and
> `func printSplash()` that picks `<i18n.Language()>.txt`, falls back to `en.txt`, and writes it to stdout.
> Delete `printSplashEN` and `printSplashRU`.

**Verification:**
- `//go:embed splash/*.txt` matches exactly once.
- `printSplashEN` and `printSplashRU` return zero grep hits in the repo.
- Running `doc-html-translate.exe` with no arguments still prints the same first three lines as before
  (compare against a capture taken before the edit).
- `go test ./internal/app -count=1` exits 0.

**Status:** `[x] done`

---

### Step 03.4 - Add the remaining 11 splash files

**Files:** `internal/app/splash/{uk,de,it,es,fr,pt,ar,hi,bn,ur,zh}.txt`
**Depends on:** Step 03.3

**Prompt for developer:**
> Translate the splash text into the remaining 11 languages, one file per code. Keep the ASCII layout
> (rule lines, two-space indents, the literal command examples in `Usage:` untranslated). For `ar` and `ur`
> do not attempt ASCII-art mirroring - the console is not RTL-aware; keep the same left-aligned shape.

**Verification:**
- `ls internal/app/splash/*.txt` returns 13 files whose base names equal `i18n.Codes`.
- No file is empty; each contains the literal `doc-html-translate.exe`.
- `go test ./tests -run TestTypography` exits 0 (the Phase 01 scan now covers these files).

**Status:** `[x] done`

---

### Step 03.5 - Register the CLI strings and delete the `IsRussian` branches

**Files:** `internal/i18n/i18n_cli.go`, `internal/app/app.go`
**Depends on:** Step 03.4

**Prompt for developer:**
> Register in `i18n_cli.go` every string currently chosen by an `if syslocale.IsRussian()` in `app.go`:
> the registration result block, the first-run right-click notice, the unregister results, the
> default-handler prompt, and the "Press Enter to close" line. Replace each branch with `i18n.S(..)`.
> Keep the answer parser accepting `y`/`yes`/`д`/`да` and extend it with the affirmative first letter of
> each added language where it is unambiguous; document the ones deliberately left out.

**Verification:**
- `syslocale.IsRussian` returns zero grep hits in `internal/app/app.go`.
- `i18n.S(` appears at least 9 times in `internal/app/app.go`.
- `go test ./internal/app ./internal/i18n -count=1` exits 0.

**Status:** `[x] done`

---

### Step 03.6 - Localize the converted-page chrome

**Files:** `internal/i18n/i18n_reader.go`, `internal/htmlgen/navbar.go`, `internal/htmlgen/htmlgen.go`
**Depends on:** Step 03.5

**Prompt for developer:**
> Back up `navbar.go` first. Register in `i18n_reader.go` and route through `i18n.S`: `Back`, `Forward`,
> `Contents`, `Smaller text`, `Larger text`, `Font`, `Theme`, `Light`, `Sepia`, `Dark`, `Night`,
> `Continue reading`, plus the two that are English-only today - `Chapters: %d` and the font-family option
> labels `Serif` / `Sans` / `Mono`. Delete both `if syslocale.IsRussian()` blocks.

**Verification:**
- `syslocale.IsRussian` returns zero grep hits under `internal/htmlgen/`.
- `i18n.S(` appears at least 15 times under `internal/htmlgen/`.
- `go test ./internal/htmlgen -count=1` exits 0.

**Status:** `[x] done`

---

### Step 03.7 - Give the chrome its own `lang` and `dir`

**Files:** `internal/htmlgen/navbar.go`, `internal/htmlgen/htmlgen.go`
**Depends on:** Step 03.6

**Prompt for developer:**
> Emit `lang="<i18n.Language()>"` and, when `i18n.IsRTL`, `dir="rtl"` on the navbar container and on the
> reader-controls toolbar - and only on those. `<html lang>` keeps carrying the **document's** language and
> must not be touched. Add a comment at both sites stating that mixing UI words into the body's language
> is what breaks Chrome's "Translate page" offer, which is the product's core free flow.

**Verification:**
- `dht-nav` container markup contains `lang="` exactly once per emitted bar.
- `<html lang=` assignment in `singlepage.go` is unchanged (git diff shows no edit to `htmlLang`).
- `go test ./internal/htmlgen -count=1` exits 0.

**Status:** `[x] done`

---

### Step 03.8 - Retire `syslocale.IsRussian` and forward the flag from the GUI

**Files:** `internal/syslocale/locale_windows.go`, `internal/syslocale/locale_nonwindows.go`, `cmd/doc-html-ui/main.go`
**Depends on:** Step 03.7

**Prompt for developer:**
> Delete `IsRussian` from both platform files. In the GUI's `assembleArgs`, forward the chosen interface
> language as `-ui-lang <code>` so the page chrome matches the window the user converted from; the GUI's
> own switcher is the control, so the parity allow-list entry (if you add one) must say exactly that.

**Verification:**
- `IsRussian` returns zero grep hits repo-wide.
- `"-ui-lang"` appears in `cmd/doc-html-ui/main.go`.
- `go test ./tests -run TestParityGUIExposesEveryCLIFlag -count=1` exits 0.

**Status:** `[x] done`

---

### Step 03.9 - Prove a converted page in a non-Latin and an RTL language

**Files:** `tests/smoke_test.go`
**Depends on:** Step 03.8

**Prompt for developer:**
> Extend the smoke test: convert the existing fixture three times with `-ui-lang en`, `-ui-lang zh` and
> `-ui-lang ar`, and assert per run that the navbar contains that language's `Contents` translation, that
> the chrome container carries the matching `lang=`, that `dir="rtl"` appears only for `ar`, and that
> `<html lang=` still reports the document's language in all three.

**Verification:**
- `func TestConvertedChromeLanguage` matches exactly once.
- `go test ./tests -run TestConvertedChromeLanguage -count=1` exits 0.
- `./scripts/verify-html.ps1` on the `ar` output reports no errors.

**Status:** `[x] done`

## Phase done criteria

- [x] Every `Step 03.*` is `[x] done`.
- [x] `./scripts/test.ps1` exits 0.
- [x] Grep for `TODO(phase-03)` returns zero hits.
- [x] Changelog entry added for every file in "Files touched".

## Step log

- 2026-07-28 - 03.1 `-ui-lang` added, validated against `i18n.Codes`; `go test ./internal/config` exit 0.
- 2026-07-28 - 03.2 single resolution site in `app.New`; build exit 0.
- 2026-07-28 - 03.3 splash moved to `splash/{en,ru}.txt` + `splash.go` (`{{rule}}` placeholder); `printSplashEN/RU` gone; opening lines pinned by `TestSplashKeepsItsOpeningLines`.
- 2026-07-28 - 03.4 11 more splash files; 13 total, typography guard green over them.
- 2026-07-28 - 03.5 9 CLI strings registered in `i18n_cli.go` (plus the per-language affirmative for the y/N prompt); `IsRussian` gone from `app.go`; 10 `i18n.S` call sites.
- 2026-07-28 - 03.6 16 `i18n.S` occurrences under `internal/htmlgen`; `Chapters: %d` and Serif/Sans/Mono localized for the first time.
- 2026-07-28 - 03.7 `lang`/`dir` on the navbar (both paths) and the index toolbar via `chromeDirAttr`; `htmlLang` untouched.
- 2026-07-28 - 03.8 `IsRussian` deleted from both platform files; GUI forwards `-ui-lang`; parity test green again.
- 2026-07-28 - 03.9 `TestConvertedChromeLanguage` en/zh/ar PASS. Real Arabic conversion: `<html lang="en">` with `<div class="dht-navbar" lang="ar" dir="rtl">`; `verify-html.ps1` 2/2 pages OK.
- 2026-07-28 - phase check `./scripts/test.ps1` -> "Tests passed" (corpus run 209 s).

## Handoff notes

`-ui-lang` is now the switch every downstream consumer uses: the GUI forwards it, and Phase 07 drives it to
take 13 sets of screenshots. The chrome-carries-its-own-`lang` rule established in 03.7 is a PARITY.md
invariant recorded in Phase 09 - the extension must match it in Phase 05.

## Rollback plan

Revert the phase commit(s). `internal/i18n` survives unused; nothing else depends on it yet.
