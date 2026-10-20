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
```

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
each other and the freshness. The locales the GUI ships and the help does not have yet are the set baseline
`configs/help-baseline.txt` (it may fall and never rise).

## Adding a locale

Write its strings in `copy/chrome.json` and every `copy/*.json` page, add the code to `locales` in `help-map.json`,
remove `locale:<code>` from `configs/help-baseline.txt`, run the generator.

## Adding a surface

Mark the region `data-help="<surface>.<region>"` in `ui.html` (or give the dialog a `surface:`), add its row to
`help-map.json` (a reason when it is `none` or `inherit`), write its entry in the copy deck, run the generator. The
gate fails until all four agree.
