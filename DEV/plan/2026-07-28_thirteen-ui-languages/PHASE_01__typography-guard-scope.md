# Phase 01 - Scope the typography guard to the house-style languages

**Strategic spec:** [`../2026-07-28_thirteen-ui-languages.md`](../2026-07-28_thirteen-ui-languages.md)
**Tactical index:** [`INDEX.md`](INDEX.md)
**Status:** ✅ Done
**Depends on:** none - foundation phase
**Steps done:** 4 / 4

## Objective

Make `tests/typography_test.go` enforce the house style (`..` not `...`, short hyphen) only on the
author-written languages (en, ru, uk) and accept script-correct punctuation in the other ten, so the
translation phases can land without a red build.

## Prerequisites

- [ ] Working tree clean or on a feature branch.

## Files touched

| File | New / Modified | Line budget |
|------|:--------------:|------------:|
| `tests/typography_test.go` | Modified | ≤ 320 |

## Steps

### Step 01.1 - Make `badTypography` language-aware

**Files:** `tests/typography_test.go`
**Depends on:** - start of phase

**Prompt for developer:**
> Change `badTypography(v string) string` to `badTypography(lang, v string) string`. For `lang` in
> `en`, `ru`, `uk` keep the current three rules unchanged (ellipsis character, three-dot ellipsis, em dash
> with the `<title>` exception). For any other language return `""` for those three characters, and instead
> reject only an ASCII three-dot `...` outside CJK - documented in a comment as "house style is an
> author-language rule; script punctuation is not drift". Update every existing call site to pass `"en"`
> (Go source literals are English by canon).

**Verification:**
- `func badTypography(lang, v string) string` matches exactly once.
- `badTypography("en"` appears in every call site inside `TestTypographyGoOutput`.
- `go test ./tests -run TestTypography` exits 0.

**Status:** `[x] done`

---

### Step 01.2 - Key the `_locales` scan on its directory name

**Files:** `tests/typography_test.go`
**Depends on:** Step 01.1

**Prompt for developer:**
> In the test that walks `extension/_locales/*/messages.json`, derive the language from the containing
> directory name (`en`, `ru`, `uk`, later `de`, `pt`, `zh_CN`, ..), normalise it to its base code by cutting
> at `_`, and pass it to `badTypography`. Unknown directories must default to the non-house-style branch,
> never to `en`, so a new locale cannot fail the build for using its own punctuation.

**Verification:**
- The `_locales` walk calls `badTypography(` with a variable, not a literal.
- A `strings.Cut(dir, "_")` (or equivalent) normalisation is present.
- `go test ./tests -run TestTypography` exits 0.

**Status:** `[x] done`

---

### Step 01.3 - Scan embedded localization text resources

**Files:** `tests/typography_test.go`
**Depends on:** Step 01.2

**Prompt for developer:**
> Add a scan over `internal/app/splash/*.txt` (the embedded splash resources Phase 03 introduces): read
> each file, take the language from the file's base name, and run `badTypography` on its contents. The glob
> matching nothing must pass, not fail - the files do not exist yet.

**Verification:**
- The pattern string `internal/app/splash` appears in the test file.
- The test tolerates zero matches (no `t.Fatal` on an empty glob).
- `go test ./tests -run TestTypography` exits 0 with the directory absent.

**Status:** `[x] done`

---

### Step 01.4 - Table-test the rule itself

**Files:** `tests/typography_test.go`
**Depends on:** Step 01.3

**Prompt for developer:**
> Add `TestBadTypographyLanguageScope` with at least these cases: `("en", "wait...")` rejected;
> `("en", "a - b")` accepted; `("en", "<title>Book — Page 1</title>")` accepted; `("ru", "ждём…")`
> rejected; `("zh", "等待——完成")` accepted; `("de", "Zeit – Ort")` accepted; `("ar", "انتظر؟")` accepted.

**Verification:**
- `func TestBadTypographyLanguageScope` matches exactly once.
- `go test ./tests -run TestBadTypographyLanguageScope -v` reports 7 or more subtests, all passing.

**Status:** `[x] done`

## Phase done criteria

- [x] Every `Step 01.*` is `[x] done`.
- [x] `go test ./tests -run "Typography|BadTypography"` exits 0.
- [x] Grep for `TODO(phase-01)` returns zero hits.
- [x] Changelog entry added for `tests/typography_test.go`.

## Step log

- 2026-07-28 - 01.1 `badTypography(lang, v)` with `houseStyleLangs{en,ru,uk}`; both call sites pass `"en"`. `go test ./tests -run TestTypography` PASS.
- 2026-07-28 - 01.2 `_locales` scan keys on the directory name, `strings.Cut` on `_`. PASS.
- 2026-07-28 - 01.3 `TestTypographySplashResources` added; glob currently matches 0 files and passes. PASS.
- 2026-07-28 - 01.4 `TestBadTypographyLanguageScope` with 10 subtests, all PASS.
- 2026-07-28 - phase check `go test ./tests -run "Typography|BadTypography"` exit 0; TODO(phase-01) hits 0.

## Handoff notes

The guard now distinguishes author languages from translated ones. Later phases may add any script's
punctuation to a non-en/ru/uk value; adding a long dash to an English or Russian string still fails.

## Rollback plan

Revert the phase commit; the guard returns to its global rule.
