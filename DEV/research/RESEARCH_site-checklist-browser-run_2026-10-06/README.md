# SITE-CHECKLIST run with real browser input

Ticket 102, 2026-10-06. Contracts `SITE-STRUCTURE`, `SITE-EXPERIENCE`, `SITE-REPRESENTATION` (draft 0.1 / 0.2 / 0.1), guide tier.

## Instrument

`tools/sitecheck` (Node, `playwright-core` driving the Chromium of the local Playwright cache; not shipped):

```
cd tools/sitecheck && npm install
node run.mjs --live            # the published site
node run.mjs --tree            # the working tree served under /doc-html-translate/ like Pages (a miss answers 404.html)
node run.mjs --live --only T,K # a subset of the sections
```

Sections: `S0` deployment identity (SHA-256 of 23 served files against `origin/main`), `A` crawl and sitemap, `B` landmarks,
kit, resolver, flash on reload, storage values, origins, `E` language control, `H` held addresses and the not-found page,
`I0` / `L` widths 360, 768, 1280, 1920, 2560, `K` real Tab walk with a pixel check of every focus ring and the skip link,
`M` `prefers-reduced-motion` emulation, `T` contrast (text rendered transparent, the background read from the screenshot
under each text run, worst 10 percent of the box; every scroll step) and 44 px targets under fine and coarse pointers,
`R` accessibility tree, `F` / `G` / `C` / `X` privacy origins, editions, the manual's structure and quoted names
(`C4` the quoted names against the product strings and the glyph rule, `C5` the fixed anatomy of the four task sections,
both added or tightened by ticket 107).
Output goes to `temp/site-run/<stamp>/` (`results.json`, `REPORT.md`, screenshots).

`shots.mjs` (ticket 107) takes the captures the manual's task sections use: it starts a build of `doc-html-ui`, opens the
About block and the secret-file password dialog in `en`, `ru` and `uk`, and writes `tools/store/docs-about-*.png` and
`docs-password-*.png`. Build first: `go build -ldflags "-X main.Version=<release>" -o temp/docs-shots/doc-html-ui.exe ./cmd/doc-html-ui`,
then `node shots.mjs --ui ../../temp/docs-shots/doc-html-ui.exe`.

What the instrument cannot do: speak. The accessibility tree was read (names, landmarks, state), no screen reader was run.

## Runs read

| Run | Subject | Read |
| --- | --- | --- |
| live | published site, deployment of `origin/main` f063557 (23 of 23 files equal the blobs) | `report-live.txt`: 27 pass, 28 fail, 2 n/a, 10 info |
| tree | working tree over HEAD 2b705c4 (the uncommitted work of tickets 97 to 101, 106) | `report-tree.txt`: 36 pass, 19 fail, 2 n/a, 10 info |

The tree is what the next publish ships; the live run is the verdict of record.

## Verdict by box (live / tree)

| Box | Live | Tree | Where it goes |
| --- | --- | --- | --- |
| A tier, reach in 3 steps (all 18 pages in 1), sitemap both ways | pass | pass | - |
| A landing's seven things | fail (no icon, no full name in display type) | pass | published by the next push |
| B kit byte-identical | fail (e544a6ce vs catalog aea958f8) | fail | ticket 104 |
| B resolver text equals `PAGE-STYLE` 7, validates the stored value | fail (2 variants, `sepia` accepted) | fail | ticket 104 |
| B resolver before the kit, no theme flash on reload (6 runs), `sza-lang` / `sza-theme` values | pass | pass | - |
| B one `main`, one `h1`, no skipped level, `lang`, `dir`, no `style` attribute, origins declared | pass | pass | - |
| B labelled `nav` landmark on every page | fail (7 pages) | pass | ticket 100, published by the next push |
| B privacy pages name the three origins (27 checks) | fail | pass | ticket 99, published by the next push |
| D search | n/a (guide tier, 18 pages) | n/a | - |
| E every language target answers, 13 / 3 / 3 locales | pass | pass | - |
| E ru and uk of `install-trust.html` carry a translated title | fail | fail | ticket 107 |
| H 8 distinct held addresses (65 holder pairs), section ids, `?l=` | pass | pass | - |
| H a missing address | fail (GitHub default) | pass (`404.html`, product chrome) | ticket 97, published by the next push |
| I0 full width at 1920 / 2560 | fail (1760 px column) | fail (1836 / 2464, rule needs 1840 / 2480) | ticket 104 |
| L 360 / 768 / 1280, no horizontal overflow | pass | pass | - |
| K skip link, Tab order, visible ring on every stop (6 pixel-checked walks) | fail (no skip link); rings all visible | pass on the landing and privacy; manual: focus under the sticky header | tickets 100, 106 |
| M reduced motion | fail | fail (`summary::after` keeps 0.2 s, back-to-top glides) | ticket 105 |
| T contrast, light | fail (154 of 1610 runs, green accent) | fail (7 of 1725: back-to-top covers the footer contact) | tickets 100, 106 |
| T contrast, dark | fail (8, same cover) | fail (6, same cover) | ticket 106 |
| T 44 px targets | fail (36 x 24 language buttons) | fail (36 x 44 and 30 x 44 buttons, footer links 17 px) | ticket 106 |
| `SITE-REPRESENTATION` 5 to 10 read on the manual's task sections | fail | fail | ticket 107 |
| Screen reader speech | not run | not run | owner pass, see the registry row |

Evidence of the measurement fixes made on the way: a first contrast pass read 154 light-theme failures on the tree; they
were the harness measuring text under the sticky header and mid smooth-scroll (the kit sets `scroll-behavior:smooth`),
and the pass was redone with the offset settled and the header band skipped before any number was recorded.
