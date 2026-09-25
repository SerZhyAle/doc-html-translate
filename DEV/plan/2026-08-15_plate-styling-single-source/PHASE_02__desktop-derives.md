# Phase 02 — The desktop editions derive their CSS

**Strategic spec:** [`../2026-08-15_plate-styling-single-source.md`](../2026-08-15_plate-styling-single-source.md)
**Tactical index:** [`INDEX.md`](INDEX.md)
**Status:** ✅ Done
**Depends on:** Phase 01
**Steps done:** 4 / 4

## Objective

The overlay CSS and the reader palette emitted by the Go app are built from the canonical source at
run time, with selector and property names supplied by the caller, and the emitted page keeps
carrying its styles inline.

## Prerequisites

- [x] Phase 01 is ✅ Done.
- [x] Working tree clean or on a feature branch.

## Files touched

| File | New / Modified | Line budget |
|------|:--------------:|------------:|
| `internal/appearance/appearance.go` | New | ≤ 220 |
| `internal/appearance/appearance_test.go` | New | ≤ 160 |
| `internal/ocr/overlay.go` | Modified | ≤ 75 changed |
| `internal/htmlgen/navbar.go` | Modified | ≤ 60 changed |

## Steps

### Step 02.1 — Embed the source and expose two builders

**Files:** `internal/appearance/appearance.go`
**Depends on:** - start of phase

**Prompt for developer:**
> Create package `appearance`. Embed `appearance.json` with `go:embed` and decode it once into
> package-level data. Expose `OverlayCSS(names OverlayNames) string`, where `OverlayNames` carries
> the container, image, plate and hidden-plate selectors, and `PaletteCSS(opts PaletteNames) string`,
> where `PaletteNames` carries the custom-property prefix and the theme attribute name. Both emit
> declarations in source order, minified the way `ocrCSS` is today. Emit each role's `note` fields as
> CSS comments so the measurements stay in the shipped stylesheet.

**Verification:**
- File `internal/appearance/appearance.go` exists.
- `func OverlayCSS(` and `func PaletteCSS(` each match exactly once as declarations.
- `//go:embed appearance.json` matches exactly once.
- `go build ./internal/appearance/` exits 0.

**Status:** `[x]` done

---

### Step 02.2 — Point the overlay generator at the builder

**Files:** `internal/ocr/overlay.go`
**Depends on:** Step 02.1

**Prompt for developer:**
> Replace the `ocrCSS` literal with a value built by `appearance.OverlayCSS`, passing this edition's
> names: container `.ocr-fig`, image `.ocr-fig>img`, plate `.ocr-box`, hidden plate
> `html.dht-ocr-off .ocr-box`. Keep the surrounding doc comment - move any measurement prose it holds
> that is now carried as a `note` in the source down to a pointer at `internal/appearance`. The
> injection site that writes the stylesheet into the page is unchanged.

**Verification:**
- `const ocrCSS = ` no longer matches in `internal/ocr/overlay.go`.
- `appearance.OverlayCSS(` matches exactly once in `internal/ocr/overlay.go`.
- `go build ./internal/ocr/` exits 0.

**Status:** `[x]` done

---

### Step 02.3 — Point the reader palette at the builder

**Files:** `internal/htmlgen/navbar.go`
**Depends on:** Step 02.1

**Prompt for developer:**
> In `readerCSS`, replace the four hand-written theme blocks (`:root` colour tokens plus the three
> `html[data-dht-theme=..]` blocks) with the output of `appearance.PaletteCSS`, prefix `dht-`,
> attribute `data-dht-theme`. Leave every non-palette declaration in `readerCSS` exactly as it is -
> the reader font and size variables, the control styling, the progress bar and the scan-page rules
> are not part of this ticket. `readerCSS` becomes a built value rather than a `const`.

**Verification:**
- `--dht-bg:#faf9f7` no longer matches in `internal/htmlgen/navbar.go`.
- `appearance.PaletteCSS(` matches exactly once in `internal/htmlgen/navbar.go`.
- `--dht-reader-size:175%` still matches exactly once.
- `go build ./internal/htmlgen/` exits 0.

**Status:** `[x]` done

---

### Step 02.4 — Pin the emitted CSS against what shipped

**Files:** `internal/appearance/appearance_test.go`
**Depends on:** Step 02.2, Step 02.3

**Prompt for developer:**
> Add a test asserting that the built overlay CSS and palette CSS contain exactly the declaration
> sets they contained before this phase: parse both outputs into `selector -> {property: value}`
> maps and compare against literals written into the test from the 2026-08-15 shipped values. This
> is the "no visual change" proof for the desktop side; declaration order may differ, content may
> not.

**Verification:**
- `func TestOverlayCSSMatchesShipped(` and `func TestPaletteCSSMatchesShipped(` each match exactly
  once.
- `go test ./internal/appearance/` exits 0.

**Status:** `[x]` done

## Phase done criteria

- [x] Every `Step 02.*` is `[x] done`.
- [x] `go test ./internal/appearance/ ./internal/ocr/ ./internal/htmlgen/` exits 0.
- [x] Grep for `TODO(phase-02)` returns zero hits.
- [x] Changelog entry added for every file in "Files touched".

## Handoff notes

Establishes: `appearance.OverlayCSS` / `appearance.PaletteCSS` as the only desktop path to these
declarations, and a shipped-value baseline the gate in Phase 04 can rely on.

## Rollback plan

Revert phase commit(s); the literals return with them.
