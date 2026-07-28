# Phase 02 - The Go localization layer

**Strategic spec:** [`../2026-07-28_thirteen-ui-languages.md`](../2026-07-28_thirteen-ui-languages.md)
**Tactical index:** [`INDEX.md`](INDEX.md)
**Status:** ✅ Done
**Depends on:** Phase 01
**Steps done:** 5 / 5

## Objective

Produce `internal/i18n` - the single place a Go user-visible string is chosen - plus a `syslocale` that
can name all 13 languages. No caller moves onto it in this phase.

## Prerequisites

- [ ] Phase 01 ✅ Done.
- [ ] Working tree clean or on a feature branch.

## Files touched

| File | New / Modified | Line budget |
|------|:--------------:|------------:|
| `internal/i18n/i18n.go` | New | ≤ 220 |
| `internal/i18n/i18n_test.go` | New | ≤ 200 |
| `internal/syslocale/locale_windows.go` | Modified | ≤ 90 |
| `internal/syslocale/locale_nonwindows.go` | Modified | ≤ 30 |

## Steps

### Step 02.1 - Create the package skeleton

**Files:** `internal/i18n/i18n.go`
**Depends on:** - start of phase

**Prompt for developer:**
> Create `package i18n` with `Codes = []string{"en","ru","uk","de","it","es","fr","pt","ar","hi","bn","ur","zh"}`
> and a parallel `Names` of endonyms (`English`, `Русский`, `Українська`, `Deutsch`, `Italiano`, `Español`,
> `Français`, `Português`, `العربية`, `हिन्दी`, `বাংলা`, `اردو`, `中文`). Add `IndexOf(code string) int`
> returning 0 (English) for anything unknown. Index 0 is the key language.

**Verification:**
- File `internal/i18n/i18n.go` exists and declares `package i18n`.
- `var Codes = []string{` matches exactly once and the slice has 13 elements.
- `len(Names) == len(Codes)` asserted at init or in the phase test.
- `go build ./internal/i18n` exits 0.

**Status:** `[x] done`

---

### Step 02.2 - Add the translation map and `T`

**Files:** `internal/i18n/i18n.go`
**Depends on:** Step 02.1

**Prompt for developer:**
> Add an unexported `map[string][]string` keyed by the **English** source string, holding the 12
> non-English translations in `Codes` order. Add `Add(en string, tr ...string)` used by the registration
> files, panicking at init if it does not receive exactly 12 translations. Add
> `T(lang, en string, args ...any) string`: returns `en` for `en` or an unknown key, falls back to English
> for an empty translation, and applies `fmt.Sprintf` when `args` are given.

**Verification:**
- `func T(lang, en string, args ...any) string` matches exactly once.
- `func Add(en string, tr ...string)` matches exactly once and contains a `len(tr) != 12` guard.
- `go build ./internal/i18n` exits 0.

**Status:** `[x] done`

---

### Step 02.3 - Add direction, font and process language

**Files:** `internal/i18n/i18n.go`
**Depends on:** Step 02.2

**Prompt for developer:**
> Add `IsRTL(lang string) bool` (true for `ar`, `ur`), `Dir(lang string) string` (`"rtl"`/`"ltr"`),
> `FontFamily(lang string) string` (`Nirmala UI` for `hi`/`bn`, `Microsoft YaHei UI` for `zh`, `""`
> otherwise), and the process-wide pair `SetLanguage(code string)` / `Language() string` defaulting to
> `"en"`. Add `S(en string, args ...any) string` as the shorthand for `T(Language(), en, args...)` - that
> is what the converted-page code will call. Add `Resolve(explicit, saved, system string) string`
> returning the first entry present in `Codes`, else `"en"`.

**Verification:**
- `func IsRTL(`, `func Dir(`, `func FontFamily(`, `func SetLanguage(`, `func Language(`, `func S(`,
  `func Resolve(` each match exactly once.
- `go vet ./internal/i18n` exits 0.

**Status:** `[x] done`

---

### Step 02.4 - Teach `syslocale` all 13 languages

**Files:** `internal/syslocale/locale_windows.go`, `internal/syslocale/locale_nonwindows.go`
**Depends on:** Step 02.3

**Prompt for developer:**
> Extend `Lang()` on Windows to map every shipped language's primary LANGID: `0x19 ru`, `0x22 uk`,
> `0x07 de`, `0x10 it`, `0x0A es`, `0x0C fr`, `0x16 pt`, `0x01 ar`, `0x39 hi`, `0x45 bn`, `0x20 ur`,
> `0x04 zh`, default `en`. Update the doc comment to say the list mirrors `i18n.Codes`. Leave
> `IsRussian()` in place - Phase 03 removes it with its last caller. The non-Windows twin keeps returning
> `"en"`.

**Verification:**
- `0x45` and `0x39` both appear in `locale_windows.go`.
- The switch has 12 non-default cases.
- `go build ./internal/syslocale` exits 0.

**Status:** `[x] done`

---

### Step 02.5 - Test the layer

**Files:** `internal/i18n/i18n_test.go`
**Depends on:** Step 02.4

**Prompt for developer:**
> Add tests: `Codes` and `Names` are the same length with no duplicates; `T` returns the key for an
> unregistered string; a registered string resolves per language; an empty translation falls back to
> English; `Dir`/`IsRTL` are correct for `ar`/`ur` and for one LTR language; `FontFamily` matches the three
> script languages; `Resolve` honours precedence and rejects an unknown code; `SetLanguage`/`Language`
> round-trip. Also assert every registered key carries 12 non-empty translations - this is the test that
> will police the later phases, so write it to walk the whole map.

**Verification:**
- `func TestEveryKeyHasTwelveNonEmptyTranslations` matches exactly once.
- `go test ./internal/i18n ./internal/syslocale -count=1` exits 0.

**Status:** `[x] done`

## Phase done criteria

- [x] Every `Step 02.*` is `[x] done`.
- [x] `go test ./internal/i18n ./internal/syslocale -count=1` exits 0.
- [x] Grep for `TODO(phase-02)` returns zero hits.
- [x] Changelog entry added for every file in "Files touched".

## Step log

- 2026-07-28 - 02.1/02.2/02.3 landed as one write of `internal/i18n/i18n.go`; each step's predicates were checked separately (13 codes, parallel `Names`, `T`/`Add` with the 12-column panic, the seven accessors, `go vet` exit 0).
- 2026-07-28 - 02.4 `syslocale.Lang()` maps 12 non-default LANGIDs including 0x39 hi and 0x45 bn; `IsRussian` left in place for phase 03. Build exit 0.
- 2026-07-28 - 02.5 `i18n_test.go`: 9 tests including `TestEveryKeyHasTwelveNonEmptyTranslations`. `go test ./internal/i18n ./internal/syslocale -count=1` exit 0.
- 2026-07-28 - phase check exit 0, `TODO(phase-02)` 0 hits, typography guard still green over the new package.

## Handoff notes

`i18n.S` is the call the converted-page and CLI code use; `i18n.T` is for callers that know their language
explicitly (tests, the GUI bridge). The 12-non-empty-translations test is the guard every later phase
inherits - never relax it to "key exists".

## Rollback plan

Revert the phase commit. Nothing consumes the package yet, so the revert is isolated.
