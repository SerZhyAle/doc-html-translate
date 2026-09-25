# Both editions describe the same plate twice, and nothing notices when the copies part

**Status:** BlockNeedUserTest
**Priority:** 44
**Date:** 2026-08-15
**Tactical plan:** [`2026-08-15_plate-styling-single-source/`](2026-08-15_plate-styling-single-source/INDEX.md)

> Cross-edition feature ticket. One feature = one ticket covering every edition.
> Read [`docs/PARITY.md`](../../docs/PARITY.md) before starting; update it when a shared invariant moves.

## What / why

The OCR plate - the opaque box that carries recognized text over the source artwork - is described
twice: once as a CSS literal inside the desktop overlay generator, once as a stylesheet in the
extension. Neither reads the other. The two copies are kept together by hand, by a prose table in
`docs/PARITY.md`, and by a guard test that pins **three** of the plate's declarations.

That is not enough, and the failure has now been observed by a reader rather than by a test.

The extension's plate carried `box-shadow: 0 0 0 1px rgba(0, 0, 0, 0.06)` - a 1 px hairline ring
around every plate. The desktop plate never had it. It arrived with the first OCR-overlay commit
(`fc6bc11`) and was removed on 2026-08-15, so the two editions drew a visibly different plate for the
whole life of the feature. It is worst exactly where the overlay is working best: when the sampled
paper colour matches the source - a speech balloon, a plain scan margin - the plate is meant to
disappear, and the ring is then the only thing announcing it, drawing a box around text the source
drew without one. Reported from a rendered comic page, not from a gate.

Nothing was red while this was true. `TestParityOCRFontFit` compares plate padding, corner radius and
which element carries the paper. `scripts/parity-check.ps1` only warns, only about files inside the
current change set, and never about a difference that already sits in both trees. `docs/PARITY.md`
does not mention the ring at all. A guard that pins three values out of fourteen reports green on the
eleven it does not read.

## What is actually duplicated, measured

Three roles make up the overlay, and each is written twice:

- **the container** - 7 declarations per side, currently identical;
- **the image** - 5 declarations on the desktop side against 3 in the extension's overlay stylesheet
  (see the equivalence case below);
- **the plate** - 14 declarations per side, identical as of 2026-08-15, which is one removed ring away
  from having been 15 against 14 since the feature shipped.

So roughly 26 declarations per edition are held in agreement by hand, and 3 of them are actually
gated.

## The four states a shared declaration can be in

The rule this ticket asks for is only useful if it tells these apart. All four exist in the tree
today, which is why a naive text comparison would be both falsely red and falsely green.

1. **Identical.** The overwhelming majority. Cheap to pin, and pinning it is the whole point.
2. **Divergent by accident.** The hairline ring: present on one side, absent on the other, in neither
   document nor test. This is the class the gate must catch on the day it appears.
3. **Equivalent, expressed differently.** The image guard that stops a page-level image reset from
   shifting the picture under its plates: the desktop side puts `margin` and `max-height` on the image
   role itself, the extension neutralizes its own reader reset in the reader stylesheet instead, so
   the shared overlay stylesheet does not carry them. Same outcome, two places, and the extension's
   standalone OCR page - which loads the overlay stylesheet without the reader one - relies on the
   browser default rather than on a stated rule. A file-to-file comparison calls this drift; it is
   not. The best of the two is the desktop shape - the guard belongs to the role, not to the one
   context that happens to need it - and this ticket should unify it rather than whitelist it.
4. **Intentionally different.** The selector and toggle names (`.ocr-box` / `html.dht-ocr-off` against
   `.ocr-plate` / `html.ocr-layer-off`), already recorded as a divergence in `docs/PARITY.md`. The
   gate must be blind to naming and read only what the declarations say.

## The experience that never travelled

The ring is the small version of a larger complaint, and the larger one is the reason this ticket
exists at all. A great deal of work went into getting the desktop conversion right - the PDF reflow
heuristics, then the OCR pipeline with a measurement lab behind it, constants bracketed from two
directions, corpus runs, recorded bounds. That work produced knowledge. The knowledge stayed where it
was found.

Nothing in the process pushes a proven result outward. The port is done by hand, the record of what
must match is prose in `docs/PARITY.md`, and the only mechanism that can actually fail is a guard
test - written per invariant, by whoever remembered to write one. So an invariant is protected in
proportion to how recently someone thought about it, not in proportion to what it cost to establish.
The result, read off `docs/PARITY.md` against the test files:

- **Guarded, and drift would be caught:** theme palette, PDF reflow constants, comic page order and
  entry filter, report field labels, the OCR lab evidence schema, and eight OCR contracts.
- **Written down, but nothing checks it:** input format detection by byte signature, the plain-text
  encoding decode order, reader font families, PDF page-image selection, the EPUB TOC rules, settings
  defaults, the product URL and feedback address, the interface language set.

The second list is not a list of suspected bugs - the two spot-checks done while writing this ticket
found the feedback address consistent across all thirteen desktop splash files, the GUI and the three
extension surfaces, and found the font families differing only in whitespace and quoting, which
renders identically. It is a list of places where **being right is currently luck**, and where a
future change has nothing to stop it from parting the editions the way the ring did.

So the deliverable is not only "unify the plate". It is a rule with a stated reach, plus an inventory
of what that reach currently leaves out, so the next person can see the gap instead of rediscovering
it from a screenshot.

## The shape the fix has to have

- **One description of the shared appearance, and both editions derive theirs from it.** The
  project's own process note already prefers a single source in code over "copy plus a values-match
  comment"; the plate is the case that proves why. In scope: the OCR overlay unit (plate, container,
  image) **and the reader theme palette**. Selector, toggle and custom-property names stay
  per-edition - the desktop side prefixes its properties, the extension does not, and neither naming
  is part of the appearance.
- **The gate blocks, and it reads the whole role, not a chosen constant.** A declaration present on
  one side and absent on the other must fail the normal test gate, including a declaration nobody
  thought to pin - that is precisely what the ring was.
- **A divergence is legal only when it is named.** Anything the two sides may legitimately differ on
  is listed with its reason, so the list is reviewable and an unlisted difference is a failure by
  default rather than a judgement call.
- **The equivalence class is resolved, not tolerated.** The image guard moves to the role on both
  sides, so the standalone OCR page stops depending on a browser default and the comparison has one
  less exception to carry.
- **No visual change to either edition.** This ticket unifies a description and adds a rule. If
  unifying moves a rendered pixel anywhere, that is a separate decision with its own evidence - the
  plate's measured constants (padding, radius, paper carrier) were each bracketed by a lab run and
  are not to be re-tuned here.

## Edition parity checklist

| Edition | Status | Notes / rationale |
|---|---|---|
| CLI (`doc-html-translate`) | `[x]` | derives both at run time from `internal/appearance` (`overlay.go` `ocrCSS`, `navbar.go` `readerCSS`); output still self-contained |
| GUI (`doc-html-ui`) | `[x]` | inherits the pipeline; no new flag, no surface change |
| MSIX Store app | `[x]` | inherits the GUI; `appearance.json` is embedded with `go:embed`, so nothing new ships beside the binaries |
| Browser extension | `[x]` | marked regions of `ocr-overlay.css` / `viewer.css` generated by `scripts/gen-appearance.mjs`; the image guard is on the role; `zip` refuses a stale region |
| Website / docs | `[~]` | `docs/PARITY.md` gains the shared-appearance row, the named-divergence list and the guarded / prose-only mark on every invariant; the catalog's `OCR-PIPELINE.md` points at the single source instead of at two files. `docs/PARITY.md` done; the catalog edit is Step 05.3, blocked on the catalog not being reachable from the implementing session |

## Shared invariants touched

- The plate's full appearance, which is today only partly an invariant: padding, corner radius and
  paper carrier are pinned; the other eleven declarations are not.
- The image role's reset guard, which is currently a desktop-only rule.
- The reader theme palette - 4 themes x 8 colour tokens, guarded today by a comparison written for it
  alone, which this ticket replaces with the general rule. Its values do not move.
- The named-divergence list itself becomes an invariant: selector, toggle and custom-property names
  are on it.

## Done criteria

1. A declaration added to one edition's plate and not the other fails the standard test gate, and the
   failure names the declaration and the side that is missing it. Demonstrated by re-introducing the
   hairline ring on one side and showing the gate red, then removing it and showing it green.
2. Renaming a selector or the toggle class on one side alone does not fail that gate - naming is
   outside what it compares.
3. Every legitimate difference between the two sides is listed with a reason, and the gate reads that
   list rather than an assumption baked into its own code.
4. The extension's standalone OCR page states the image guard rather than inheriting a browser
   default, and the desktop page is unchanged in that respect.
5. A rendered comic page from the corpus shows no ring and no other visible change against the
   2026-08-15 output, in both editions.
6. A theme colour changed on one edition and not the other fails the gate, and renaming a colour
   token on one side alone does not - the palette is held by the same rule as the plate, not by a
   second bespoke comparison.
7. `docs/PARITY.md` describes the shared appearance as derived from one source, and no longer as two
   copies to be checked by eye.
8. Every invariant listed in `docs/PARITY.md` carries an explicit mark for whether a gate enforces it
   or only prose does, so the "being right is currently luck" list is visible in the document rather
   than only in this ticket. Marking is the deliverable; closing the unguarded entries is not, and
   each one that stays open is named for a later ticket.
9. `./scripts/test.ps1`, `./scripts/lint.ps1`, `./scripts/check.ps1` and `npm test` green; a
   `DEV/CHANGELOG.md` entry.

## Open questions

1. **How far does "one source" reach?** **Resolved 2026-08-15 by the owner: the overlay unit and the
   reader theme palette, together.** The palette is duplicated on different terms - 4 themes x 8
   colour tokens per side, under different property names, with a working comparison test and no
   observed drift - so it is the case that shows the rule generalizing beyond the place a defect was
   found, which is the point of the ticket. Everything on the "written down, nothing checks it" list
   above stays out of this ticket's implementation and is carried into its inventory instead.
   **Status:** Resolved.
2. **Does the desktop side keep emitting its stylesheet inline?** Self-contained offline output is a
   product property, so whatever the single source is, the generated page must keep carrying its
   styles with it. This constrains the mechanism, not the decision.
   **Status:** Open, constraint stated.
3. **What happens to a difference that is real but unrecorded when the gate first runs?** The tree is
   believed clean for the plate as of 2026-08-15, but the container and image roles have not been
   compared declaration by declaration under a rule. The first run may surface more equivalence-class
   cases like the image guard.
   **Status:** Open - resolved by running the comparison once during implementation and triaging what
   it finds into "unify" or "name it".

## Related

- [`2026-08-13_ocr-sweep-plate-composition.md`](2026-08-13_ocr-sweep-plate-composition.md) - set the
  plate's measured constants (padding, radius, paper carrier) and the three-value guard this ticket
  widens. Those measurements are inputs here, not open questions.
- [`docs/PARITY.md`](../../docs/PARITY.md) - the process note preferring a single source in code over
  a copy plus a comment is the rule this ticket applies.

## Last Audit

**Date:** 2026-09-25
**Mode:** full
**Outcome:** BlockNeedUserTest
**Counts:** PASS 90 · WARN 0 · FAIL 0 · MANUAL 4 · EXEMPT 2

Static audit of the tree after the budget fix-up. Every step predicate of phases 01-04, 05.1-05.2
and 06.1 holds; every file is within its (re-set) budget - appearance.go 238/240,
appearance_parity_test.go 580/600, overlay.go 73/75 changed, parity_test.go 121/125, PARITY.md
126/130. Count mismatches on `.ocr-badge`, `#content img` and `--reader-size` are a compound selector,
a comment and a `var()` use - one declaration each. Criteria 1-4 and 6-8 hold statically. EXEMPT:
user docs (§8 mandates none) and the verification-tag invariant - no manual item below points at
a source location, so there is nothing to tag.

### Manual / on-target
- [ ] Step 05.3: repoint the catalog's `ocr-pipeline.md` plate rows and §3.1 at
      `internal/appearance`. The catalog is not reachable from a cloud session.
- [ ] Criterion 5: render one corpus comic page with OCR on, in both editions, and compare it with the
      2026-08-15 output. The headless-Chromium CSS comparison (0 differing pixels) does not replace it.
- [ ] Criterion 9: `./scripts/test.ps1`, `lint.ps1`, `typo.ps1` and `check.ps1` exit 0 on Windows.
- [ ] Step 06.2: record the exit codes and the corpus page in the INDEX change log.
