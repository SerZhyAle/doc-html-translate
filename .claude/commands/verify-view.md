# Verify-view — headless-check converted HTML

> **GLOBAL DIRECTIVES:**
> 1. Observe real rendered behaviour - do not claim "Chrome-checked" without running this.
> 2. Local + free. Read-only: it renders output, it never edits or publishes.
> 3. Outputs go in the repo's gitignored `temp/`, never the scratchpad (its long path breaks MAX_PATH / OCR;
>    see memory `run-artifacts-in-project-temp`).

Render a converted book (or a single page) in headless Chrome/Edge and assert the common checks, producing
the exact evidence line the changelog wants. Backs the `a v` alias and `scripts/verify-html.ps1`.

## Usage

```text
/verify-view <path to a converted output folder or .html file>
```

## Process

**Step 1 — Convert to `temp/` if needed.** If there is no output folder yet, run the CLI into the repo's
`temp/` (not the scratchpad): `./build/doc-html-translate.exe -notranslate -noopen -force "<input>"`.

**Step 2 — Run the verifier:**

```powershell
./scripts/verify-html.ps1 -Path "temp/<book folder>"
# single page + a required marker:
./scripts/verify-html.ps1 -Path temp/out/page_002.html -Expect "<embed application/pdf"
```

It prints one line per page in the changelog's own vocabulary:
`page_002.html   total=3 render=3 broken=0  [embed pdf]`, and exits non-zero if any page has a broken
image, a missing expected marker, or a blank render.

**Step 3 — Read the result.** `broken=0` and no FAIL lines = the render is clean. A `broken=N` means N `<img>`
point at files that are not on disk (missing/misnamed extraction output) - investigate before shipping.

**Step 4 — Feed the evidence to `/changelog`.** Paste the summary line(s) into the changelog Description
(`Chrome-checked: total=1 render=1 broken=0`, `<embed application/pdf> present`) so the record is grounded
in an actual render, not an assumption.

## When NOT to use

- Pure code/docs change with no converted-HTML surface to render → nothing to verify here.
- You need runtime console-error / naturalWidth checks beyond on-disk image resolution → that needs a full
  CDP harness; note the gap rather than claiming it was checked.
