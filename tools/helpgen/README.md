# tools/helpgen - the in-app help generator

Ticket 118, `CONTEXT-HELP` (the shared contract for F1 and the `?` button). The GUI launcher's help is a set of
pages, one per surface, each with a screenshot of the real window whose numbered pins come from the window's own
element bounds. This tool writes the whole set; nobody edits it by hand (canon invariant 16).

```
node tools/helpgen/generate.mjs                    build the exe, capture, render, write both copies
node tools/helpgen/generate.mjs --locales en,ru    a subset (stale files of the other locales stay)
node tools/helpgen/generate.mjs --only converter   one page
node tools/helpgen/generate.mjs --exe <path>       use an exe instead of building one
node tools/helpgen/generate.mjs --check            regenerate into temp/helpgen/check-out and report what differs
node tools/helpgen/repage.mjs                      the fast form: re-render the pages from the committed captures
node tools/helpgen/repage.mjs --check              the same in memory; exit 1 when the committed pages differ
```

`repage.mjs` is for a change of wording only (the copy deck's prose, `help.css`, the renderer). It builds nothing and
opens no browser, so a call takes well under a second (0.07 s on 2026-10-29). It reuses the screenshots and the pin
places in `captures.json`, so it refuses (exit 2, nothing written) when an input of the window changed after the last
capture - `ui.html` markup or style, `i18n.js`, `generate.mjs`, `shots.mjs` - because the new pages and the images
beside them would then disagree; that case is answered by `generate.mjs` alone. It also refreshes `freshness.json`,
so the gate stops naming the wording inputs as newer than the set. It cannot see an edit of a selector, a pin mode, a
setup or a capture kind in the copy deck (they decide where a pin sits); `generate.mjs --check` at the release
compares its output with the full generator's byte for byte.

Which one runs when (ticket E0125, owner decision 2026-10-30): a change of wording reaches both copies of the set in
the same edit through `repage.mjs`. A change of what the window shows (`ui.html`, `i18n.js`, the capture code) leaves
the screenshots and pin places to the release, because `generate.mjs` builds the GUI and takes about two minutes and
nothing above 5 s runs between releases; CONTEXT-HELP rule 18 carries a dated exception for that half. Until then the
screenshots show the previous window, `repage.mjs` refuses, `help-check` names the moved input, and the release
pre-flight runs `generate.mjs` and then `generate.mjs --check`.

Needs the Go toolchain, Node, and the Playwright Chromium of `tools/sitecheck` (`npm ci` there once). Run it from
the repository root in PowerShell. It runs the real `doc-html-ui.exe` under a scratch profile
(`temp/helpgen/profile`, `LOCALAPPDATA` redirected, `DOCHT_GUI_NO_WINDOW` so no browser window opens) with demo
documents; it refuses to write a capture that shows a path, a user name or a key of the machine it ran on.

## What it reads

| Input | Holds |
| --- | --- |
| `cmd/doc-html-ui/help/help-map.json` | the surfaces and their pages: the one map the window's resolver (`help.js`) and the gate read |
| `cmd/doc-html-ui/i18n.js` | the 13-language GUI dictionary: names and one-line summaries come from it (`{k:key}`, `nameKey`, `oneKey`), so a renamed button is renamed in its help by the same edit |
| `tools/helpgen/copy/*.json` | the copy deck: the authored text of each page in `en`, `ru`, `uk`, the shots, the elements with their selectors and pin modes |
| `tools/helpgen/vendor/help.css` | byte copy of the catalog's `CONTEXT-HELP` reference `help.css` |
| `assets/sza-kit.css` | the site kit, byte-identical to the catalog's |

## What it writes

Into `cmd/doc-html-ui/help/` (embedded in the exe with `go:embed`, served under `/help/`) and, the same bytes, into
`help/` at the repository root (the site's copy):

- `<locale>/<page>.html`, `<locale>/index.html` - the pages, complete without script (`page.js` only follows the
  window's theme and opens the entry a link points at);
- `<locale>/<page>-<shot>-<theme>.png` - the screenshots, in the page's own language, light and dark;
- `sza-kit.css`, `help.css`, `page.js` - the shared parts;
- `captures.json` - size and pin percentages of every capture; `freshness.json` - the digests of the inputs, which
  `internal/helpcheck` recomputes to say the set is older than an input.

## The gate

`scripts/help-check.ps1` (`go run ./tools/helpcheck`) holds the surfaces of `ui.html` against the map, the pages
against the locale set, the links, rule 12 (a page makes no request), the house text style, the two copies against
each other and the freshness: an advisory between releases, a failure under `-Release`
(`scripts/help-check.ps1 -Release`, `go run ./tools/helpcheck -strict-freshness`), which the full gate and the release
pre-flight use. The locales the GUI ships and the help does not have yet are the set baseline
`configs/help-baseline.txt` (it may fall and never rise).

## Adding a locale

Write its strings in `copy/chrome.json` and every `copy/*.json` page, add the code to `locales` in `help-map.json`,
remove `locale:<code>` from `configs/help-baseline.txt`, run the generator.

## Adding a surface

Mark the region `data-help="<surface>.<region>"` in `ui.html` (or give the dialog a `surface:`), add its row to
`help-map.json` (a reason when it is `none` or `inherit`), write its entry in the copy deck, run the generator. The
gate fails until all four agree.
