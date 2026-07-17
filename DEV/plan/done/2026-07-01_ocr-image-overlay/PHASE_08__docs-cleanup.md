# Phase 08 — Docs cleanup

**Strategic spec:** [`../2026-07-01_ocr-image-overlay.md`](../2026-07-01_ocr-image-overlay.md)
**Tactical index:** [`INDEX.md`](INDEX.md)
**Status:** ✅ Done
**Depends on:** all prior phases
**Steps done:** 4 / 4

## Objective
Update user-facing docs, privacy policy, and store copy to describe image OCR + optional language
downloads, and record the changelog.

## Prerequisites
- [ ] Phases 01-07 ✅ Done.

## Files touched
| File | New / Modified | Line budget |
|------|:--------------:|------------:|
| `extension/store/PRIVACY.md` | Modified | ≤ 80 |
| `extension/store/LISTING.md` | Modified | ≤ 300 |
| `extension/README.md` | Modified | ≤ 300 |
| `extension/_locales/en/messages.json` (+ ru, uk `storeDescription`) | Modified | ≤ 200 |

## Steps

### Step 8.1 — Privacy policy: disclose optional language downloads
**Files:** `extension/store/PRIVACY.md`
**Depends on:** - start of phase

**Prompt for developer:**
> Update `extension/store/PRIVACY.md`: add that image OCR runs on-device (Tesseract, bundled English),
> and that additional OCR languages are downloaded — as data, only on the user's explicit action — from
> a public CDN and cached locally; nothing is sent to servers we control. Bump the "Last updated" date.
> Keep the hosted-URL note in sync.

**Verification:**
- Grep in `PRIVACY.md` shows mention of on-device OCR and the optional language download from a CDN.
- The "Last updated" date is changed.

**Status:** `[x]` done

---

### Step 8.2 — Store listing + description strings
**Files:** `extension/store/LISTING.md`, `extension/_locales/en/messages.json`, `extension/_locales/ru/messages.json`, `extension/_locales/uk/messages.json`
**Depends on:** - start of phase

**Prompt for developer:**
> Add an "Image OCR / translate text in pictures" feature bullet to `store/LISTING.md` and to the
> `storeDescription` message in all three locale catalogs (right-click any image; OCR images inside
> PDFs/EPUBs; English built-in, more languages downloadable; 100% local for the default). Follow repo
> typography rules.

**Verification:**
- Grep in `store/LISTING.md` shows an OCR/image-translation bullet.
- Each `messages.json` `storeDescription` mentions image OCR.

**Status:** `[x]` done

---

### Step 8.3 — README usage docs
**Files:** `extension/README.md`
**Depends on:** - start of phase

**Prompt for developer:**
> Add a short "Translate text in images (OCR)" section to `extension/README.md`: the three entry points
> (right-click a web image; the "Use OCR for images" toggle for PDFs/EPUBs), how to add languages, and
> the privacy note. Follow repo typography rules.

**Verification:**
- Grep in `extension/README.md` shows an OCR section naming the context-menu and the toggle.

**Status:** `[x]` done

---

### Step 8.4 — Changelog + INDEX completion
**Files:** `extension/README.md` (or the repo changelog used by the build flow)
**Depends on:** Step 8.1, Step 8.2, Step 8.3

**Prompt for developer:**
> Add a changelog entry summarizing the OCR feature and listing the files added/changed across phases
> 01-07, in the location the project's build/changelog flow uses. Then confirm the INDEX completion
> gate.

**Verification:**
- A changelog entry mentioning OCR image overlay exists in the project's changelog location.
- INDEX "Completion gate" boxes are all check-ready (verified in `/spec-check`).

**Status:** `[x]` done

## Phase done criteria
- [ ] Every `Step 8.*` is `[x] done`.
- [ ] `npm run zip` in `extension/` still succeeds (docs are excluded but packaging must stay green).
- [ ] Grep for `TODO(phase-08)` returns zero hits.
- [ ] Changelog entry present.

## Handoff notes
See INDEX.md Completion gate. Feature is release-eligible only after owner sign-off (privacy + store
copy changed) per strategic §3.3.

## Rollback plan
Revert phase commit(s); docs-only, low risk.
