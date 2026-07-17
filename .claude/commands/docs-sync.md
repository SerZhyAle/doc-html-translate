# Docs-sync — update every documentation surface together

> **GLOBAL DIRECTIVES:**
> 1. Dry, factual prose - no marketing filler.
> 2. **en/ru/uk is one atomic edit.** Never ship a surface in one language and leave the others "for later".
> 3. Typography in all output: short hyphens (no long dashes), Russian **ё**, ".." not "..." (CLAUDE.md).
> 4. Local + free. Editing docs never pushes or publishes - the commit/build is a separate step (`/build`).

Update the user-facing documentation for a change across **all** the surfaces that move together, so none
is forgotten and every language stays in parity. The surface list is the manifest
[DEV/DOCS_SURFACES.md](../../DEV/DOCS_SURFACES.md) - read it first; it is the source of truth.

## Usage

```text
/docs-sync <what changed - the feature/fix to document>
```

## Process

**Step 1 — Read the manifest.** Open [DEV/DOCS_SURFACES.md](../../DEV/DOCS_SURFACES.md). That table is the
checklist for this pass. If the change adds a brand-new surface, add a row to the manifest too.

**Step 2 — Decide which surfaces this change touches.** Not every change hits every surface (a privacy
change touches `PRIVACY.md`; a new feature touches almost everything and **leads the landing hero**). State
the subset you will edit, in one line, before editing.

**Step 3 — Edit surface by surface, trios in lockstep.** For each surface in your subset:
- Single-language surfaces (`README.md`, `extension.html`, `extension/README.md`, `LISTING.md`): edit directly.
- In-page multilingual (`index.html`, `extension.html`): edit the en **and** ru **and** uk copies inside the file.
- **The trios** - `docs.html` / `docs.ru.html` / `docs.uk.html`, and `_locales/{en,ru,uk}/messages.json`:
  edit all three in the same pass. For `messages.json`, add/rename a key in all three or none (keys must match).
- A new user-facing feature **leads** the landing hero (not a collapsed section) and is mirrored everywhere.

**Step 4 — Verify parity.** Confirm the trio files carry the same information (only translated), and that
`messages.json` key sets are identical across en/ru/uk. Call out any surface you deliberately skipped and why.

**Step 5 — Changelog + hand off.** Record the docs change with `/changelog` (`docs: ...` subject), then stop.
The commit/build is `/build`; publishing is `/release`. List what you edited and what you skipped.

## When NOT to use

- Code-only change with no user-facing text → skip; just `/changelog` + `/build`.
- You are publishing → that's `/release` (which includes a docs step that can call this).
