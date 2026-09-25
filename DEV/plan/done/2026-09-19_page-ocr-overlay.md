# The reader can only OCR a live page one picture at a time, in a tab that is no longer the page

**Ticket:** 2026-09-19_page-ocr-overlay
**Status:** Implemented - moved to done/ by the owner on 2026-09-25. Still open outside this ticket: a hands-on pass on a real webcomic, a real scanned-archive page and a page whose pictures are cross-origin, plus one site with a restrictive content security policy and one browser below the offscreen-API version, before the store-listing permission text is written (§3.3 owner sign-off).
**Priority:** 50
**Date:** 2026-09-19
**Tier:** Strategic
**Tactical plan:** none - implemented directly from this strategic spec once research item 1 was
answered in code (capability detection with a documented fallback). The shipped shape is recorded
under [Intentional divergences](../../../docs/PARITY.md#intentional-divergences-do-not-fix).

> **Scope:** STRATEGIC. Goals, constraints, open questions. No class names, paths, line budgets,
> schema versions, or framework module details.

> Extension-only feature ticket. There is no desktop counterpart and there must not be one - the Go
> app does not live inside somebody else's page. Read [`docs/PARITY.md`](../../../docs/PARITY.md) before
> starting and record this under "Intentional divergences", not as a gap to close.

---

## 1. Problem

A reader on an ordinary web page - a webcomic, a scanned archive, a gallery of screenshots - has
pictures in front of them whose text the browser cannot see. Today the extension answers only for one
picture at a time: the image context menu recognizes the picture the reader right-clicked and opens
the result in **its own tab**, which is no longer the page. The reader loses the page's layout, its
links and its reading order, and has to repeat the trip for every panel. What they want is the page
they are already on, unchanged and still clickable, with the recognized words sitting on top of the
pictures as real selectable text - so the words can be copied, and so the browser's own
"Translate page" translates them together with everything else.

## 2. Goals

1. A reader can ask, from the page context menu, for **every picture on the page at once** to be
   recognized, without leaving the page.
2. The page keeps working exactly as before: its links, its scripts, its navigation, its reading
   order. Nothing is re-rendered or rebuilt.
3. Recognized text lands **over the picture it came from**, as real text in the page, so the reader
   can select it, copy it and search it with the browser's find.
4. Chrome's built-in page translation translates that text along with the rest of the page, in place,
   with no key and no account - the product's signature "free" flow, now on a page the product never
   converted.
5. The reader can turn the layer off again and get the untouched page back.
6. A picture whose text the reader wants to click through stays clickable.

**Non-goals:**

- Recognizing anything that is not an `<img>` in the first iteration: CSS background images,
  `<canvas>`, WebGL, video frames.
- Reaching into cross-origin frames the extension is not already allowed into.
- Translating the recognized text ourselves. The browser's page translation does that, as it does in
  the viewer today.
- Any desktop-app counterpart.
- Persisting the layer across a reload.

## 3. Wishes and constraints

### 3.1 Owner wishes

- The recognized layer should look like the one the viewer already renders, so a reader who has seen
  one has seen both.
- Recognition should start with what the reader is looking at rather than with the top of the
  document, so a long page is useful before it is finished.

### 3.2 Hard constraints

- **Platform / versions:** the extension's declared minimum Chrome version is a **frozen reach
  commitment** - raising it to buy an API is a release that shrinks reach and is forbidden. Any
  capability newer than the current floor must degrade, not gate.
- **Performance:** recognition is seconds per picture. A page of forty panels must never block the
  page, never freeze the tab, and must be interruptible; the tab-crash class this product already hit
  once is the thing to design against.
- **Permissions:** every new permission must be one the runtime actually uses and must carry a
  one-sentence user-visible justification, in the manifest and in both store listings.
- **Data compatibility:** none - nothing is persisted beyond the existing options.
- **Localization:** every new user-visible string ships in all authored locales in the same edit.
- **Accessibility:** the layer is real text, so it must be reachable by the same means as the page's
  own text, and the off switch must be operable without a pointer.

### 3.3 Owner inputs (Approval gate)

- **Related tickets:** none blocking. Shares the recognition and plate-composition work tracked by
  [`14_2026-08-15_plate-styling-single-source`](../14_2026-08-15_plate-styling-single-source.md) and the
  closed composition tickets in [`done/`](./); this ticket must not fork that logic.
- **Copy/tone policy:** the new menu item and the layer's controls follow the wording of the existing
  image menu item, in all authored locales.
- **Performance budget:** the page stays interactive throughout; recognition is queued, one picture at
  a time, and the reader can stop it.
- **Platform constraints:** the minimum browser version does not move. See §6, item 1.
- **Localization:** all authored locales in one edit, per the canon.
- **Validation level:** automated where the arithmetic allows, plus a hands-on pass on a real
  webcomic, a real scanned archive page and a page whose pictures are cross-origin.
- **Owner sign-off:** required before the store-listing permission text is written.

## 4. Current architecture context

The extension is built around **its own tab**. Everything it does today happens in a page it serves
itself - the viewer, and the single-image recognition page - which is why it has never needed to run
inside somebody else's document and declares no content scripts at all. The recognition unit itself
is already the right shape: it recognizes a picture, groups the words into block-level plates and
renders those plates as real text over the source image precisely so that the browser's page
translation can reach them. It just has no way to be pointed at a document it does not own.

So the problem is not recognition and not composition. It is **reach and placement**: nothing in the
extension can observe a third-party page's pictures, and nothing can keep a rendered plate attached to
a picture whose size the page decides and keeps changing. Two properties of the browser make this more
than plumbing. First, a picture served from another origin cannot be read back out of the page by the
page's own code, so the bytes have to be fetched by the part of the extension that is allowed to fetch
them. Second, the recognizer needs to compile and run a WebAssembly module, and a module compiled from
inside a third-party document is subject to **that document's** content security policy, which a great
many sites set restrictively - so the recognizer cannot simply be carried into the page.

## 5. Proposed approach

Split the work across three roles that already have the right privileges, and let the page hold only
the cheapest one.

- A **page agent**, injected on demand, which is the only part that touches the reader's document. It
  finds the pictures, reports where each one sits, draws the plates it is handed, keeps them attached
  as the page reflows, and removes them on request. It does no recognition and holds no engine.
- A **broker**, the extension's existing background role, which owns the reader's intent, fetches
  picture bytes under the extension's own origin privileges, orders the work and can stop it.
- A **recognizer host**, an extension-owned document that is not the reader's page, where the existing
  recognition unit runs under the extension's own security policy, unchanged.

Only geometry and text cross between them: the reader's page never receives the engine, and the
recognizer never receives the reader's DOM.

### 5.1 Pillars / modules

1. **Intent and control.** A page-level context-menu entry, the state of the layer per tab, and a
   visible way to stop a run and to remove the layer. Requirement: an interrupted run leaves no
   half-drawn layer and no running work.
2. **Picture discovery and prioritization.** Enumerate the document's pictures, skip what cannot
   carry text (decorative sizes, sprites, data-less placeholders), and order the queue by what the
   reader is looking at. Requirement: pictures that appear later - lazy loading, infinite scroll - are
   either picked up or explicitly out of scope, stated rather than silently missed.
3. **Byte acquisition.** Obtain each picture's pixels regardless of origin, with the existing
   orientation handling preserved. Requirement: a picture whose bytes cannot be obtained fails
   alone and is reported, never taking the run down.
4. **Recognition.** Reuse the shipped recognition and plate-composition unit as-is. Requirement: no
   second copy of the clustering, rescue or composition rules, in any edition.
5. **Placement and survival.** Render the plates over the picture in the live document and keep them
   correct when the picture is resized, re-sourced or moved. Requirement: the page's own geometry is
   never modified to make placement easier.
6. **Coexistence with the page.** The layer must not steal interaction the page needs, and must not be
   mistaken by the page's own scripts for its content. Requirement: a picture that is a link stays a
   link.
7. **Surfaces.** Permission justification, store listings, README and site, all locales, one edit.

### 5.2 Data and event flows

- Reader -> context menu -> broker: "recognize this page".
- Broker -> page agent (injected): "report your pictures". Page agent -> broker: identity and
  geometry per picture.
- Broker: fetch bytes -> recognizer host: pixels plus the reader's language settings. Recognizer host
  -> broker: plates as text plus boxes in the picture's own coordinates, and progress.
- Broker -> page agent: plates for one picture. Page agent: draw, then keep them attached to that
  picture for as long as the layer lives.
- Reader -> control -> broker: "stop" or "remove", which propagates to both the queue and the page
  agent.

### 5.3 Extension points

- The set of recognizable picture sources: the first iteration answers for `<img>`, and the
  discovery step is the one place a later source is added.
- The recognizer host is one seam, so a different engine or a different host mechanism replaces one
  role and not the feature.
- The layer's styling stays on the single styling source the plate-styling ticket is consolidating -
  this feature must not open a second one.

## 6. Open questions / research items

1. **Where the recognizer runs without moving the minimum browser version.**
   - **Question:** the obvious host for the engine is newer than the extension's declared minimum
     Chrome version, and raising that minimum shrinks reach, which is forbidden. Can the host be used
     when present and fall back when absent, and what is the fallback worth?
   - **Options:** capability detection with a documented fallback host; an extension-owned frame
     inserted into the page; declining the feature below the floor with an honest message.
   - **To find out:** the exact version each candidate host needs; what share of the installed base
     sits below it; whether the fallback survives a restrictive page policy.
   - **Status:** Resolved - **capability detection with a documented fallback**. The offscreen
     document is used wherever the browser offers it and an extension-origin frame parked in the page
     is used where it does not, so the declared minimum browser version does not move and a reader
     below the newer API gets the feature rather than a refusal. The minimum is pinned by a test, so
     a later edit cannot quietly buy the API with reach. What the fallback is worth on a restrictive
     site is item 2, and it is now a measurement on shipped code rather than a design question: the
     run reports the refusal to the reader instead of producing nothing.

2. **Whether a third-party page's security policy reaches the fallback.**
   - **Question:** if the fallback host lives inside the reader's document, does that document's
     policy block the engine, and on what kind of site?
   - **To find out:** measure on a set of real sites with restrictive policies, including at least one
     webcomic and one archive, and record which ones refuse. The code to measure with is shipped, and
     it names the refusal, so this is now a hands-on pass and not a design question.
   - **Status:** Open - measurement pending, part of the `BlockNeedUserTest` gate. It does not block
     the offscreen path, which is what nearly every current reader will take.

3. **What "every picture on the page" costs on a real page.**
   - **Question:** how many pictures a typical target page has, how long a full run takes, and where
     the reader stops tolerating it.
   - **To find out:** measure on a webcomic page, a scanned-archive page and a gallery; record picture
     count, bytes and wall-clock per picture.
   - **Status:** Open.

4. **How the layer survives the page.**
   - **Question:** which page behaviours break placement - responsive re-sourcing, lazy loading,
     sticky and transformed containers, virtualized lists - and which of them the first iteration
     refuses out loud instead of handling.
   - **Answered so far by the shipped design:** placement re-reads each picture's live geometry on an
     animation frame, for the pictures in or near the viewport only, so re-sourcing, lazy growth,
     sticky and transformed containers and a moving list all follow by construction. Pictures that
     appear *after* the run are deliberately not swept up behind the reader's back - the bar offers a
     rescan instead. The one refusal stated out loud: a transform on `<html>` or `<body>` makes the
     layer's containing block the transformed box, and the layer would sit off the art there.
   - **Status:** Open - the refusal list is a measurement, part of the `BlockNeedUserTest` gate.

5. **What the stores will accept.**
   - **Question:** what justification the new permissions need in both store reviews, and whether
     "acts on every image on any site" changes the data-safety answers.
   - **To find out:** read the current review guidance for both stores; draft the justification before
     any code ships.
   - **Status:** Open.

## 7. Risks

| Risk | Likelihood | Impact | Mitigation |
|------|:----------:|--------|-----------|
| The engine cannot run on a restrictive site | High | The feature silently does nothing on real targets | Research item 2 before implementation; host the engine outside the page; report a refusal rather than a blank result |
| A full-page run freezes or crashes the tab | Med | Worst class of failure; this product already shipped one | One picture at a time, bounded queue, visible stop, run only what the reader can see |
| Plates drift off the pictures as the page reflows | High | The layer covers the wrong art, which is worse than no layer | Placement follows the picture's live geometry; refuse the page behaviours that cannot be followed, and say so |
| Plates swallow clicks the page needs | Med | A comic whose panels are links stops navigating | Interaction stays with the page by default; the layer's own interaction is deliberate and reversible |
| A new broad permission is refused or delays a store review | Med | The release stalls | Draft the justification first, sign it off, keep the permission list to what the runtime uses |
| The composition logic gets forked for the page case | Med | Two sets of plate rules, drifting | Reuse the shipped unit; a parity guard fails on a second implementation |
| Reach is shrunk to buy an API | Low | Forbidden by the canon; orphans installed users | Research item 1 is a hard gate on the approach |

## 8. User impact (docs)

Yes - this is a capability the user will read as new. One sentence for the feature docs: the extension
can recognize the text in every picture on an ordinary web page and lay it over the pictures as real
text, so it can be copied and translated by the browser along with the rest of the page. It reaches the
README, the site, both store listings and the permission justification, in every authored locale, in
one edit.

## 9. Architecture decisions (ADR)

**ADR-1: the reader's page never hosts the recognition engine.** *Decision:* recognition happens in an
extension-owned document; the page receives only text and geometry. *Alternatives:* carrying the engine
into the page, which is fewer moving parts. *Why:* a third-party document's security policy governs
what code compiles inside it, so the simpler design fails on exactly the sites this feature exists for,
and fails invisibly.

**ADR-2: picture bytes are fetched by the extension, not read out of the page.** *Decision:* the byte
acquisition role uses the extension's own privileges. *Alternatives:* reading the rendered picture from
inside the page. *Why:* a picture from another origin cannot be read back inside the page at all, and
most pictures worth recognizing are from another origin.

**ADR-3: the feature is extension-only, on purpose.** *Decision:* record it in the parity document as
an intentional divergence with no desktop counterpart. *Alternatives:* inventing a desktop equivalent.
*Why:* the desktop app converts documents it is given; it has no live page to sit inside. This is the
mirror of the divergences already recorded in the other direction.

## 10. Links to other specs

- [`14_2026-08-15_plate-styling-single-source`](../14_2026-08-15_plate-styling-single-source.md) - the layer's
  styling must come from the source that ticket is consolidating, not from a new one.
- [`done/2026-07-01_app-ocr-image-overlay`](2026-07-01_app-ocr-image-overlay.md) - the overlay
  this feature reuses.
- [`done/2026-08-12_extension-crashes-the-tab-on-a-detailed-scan`](2026-08-12_extension-crashes-the-tab-on-a-detailed-scan.md)
  - the failure mode the queue design exists to avoid.

## 11. Done criteria (strategic)

1. On a real webcomic page, the reader chooses one menu item and, without leaving the page, sees
   recognized text over the panels.
2. The page's links still navigate and its own scripts still run, verified on a page whose pictures
   are themselves links.
3. The recognized text can be selected and copied out of the live page, and the browser's find locates
   a word that exists only in a picture.
4. The browser's "Translate page" renders that text in the reader's language in place, with no key
   and no account.
5. The reader can stop a run in progress and remove the layer, and the page is then indistinguishable
   from before.
6. A page of at least forty pictures stays scrollable and interactive for the whole run, and the tab
   does not crash.
7. A picture that cannot be obtained or cannot be recognized is reported and does not stop the rest.
8. The plate rules are the shipped ones - a guard fails if a second implementation appears.
9. The minimum supported browser version is unchanged from the previous release.
10. Every new string exists in every authored locale, and the permission justification is live on both
    store listings before the feature ships.

## 12. Next step

Implemented. What is left is not code:

1. **Hands-on pass** on a real webcomic, a real scanned-archive page, a page whose pictures are
   cross-origin, a site with a restrictive content security policy, and a browser below the
   offscreen-API version. This is what closes research items 2, 3 and 4 and lifts
   `BlockNeedUserTest`.
2. **Owner sign-off, then the store-listing permission text** for the two new permissions, on both
   listings, before the feature ships (§3.3, research item 5). The manifest justification is written;
   the listings are not, on purpose.
3. **Screenshots and the store "what's new" line** at release time, per the release checklist.
