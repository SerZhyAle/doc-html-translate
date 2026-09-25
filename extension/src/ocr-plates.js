// ocr-plates.js - the plate half of the OCR overlay: block geometry becomes plate geometry,
// a plate spec becomes a DOM node, and a rendered plate is fitted to its box at runtime.
// Split out of ocr-overlay.js so this half can run where the recognition engine must not:
// the page agent draws plates inside a third-party document, and pulling ocr-overlay.js in
// would carry the Tesseract module into that document with it (see
// DEV/plan/done/2026-09-19_page-ocr-overlay.md, ADR-1). ocr-overlay.js re-exports everything here,
// so the viewer and the single-image page import the same names they always did.
//
// There is exactly one implementation of the plate rules, in this file, for every surface -
// tests/parity_test.go pins the constants and the paper carrier against the desktop app's
// overlay.go (see docs/PARITY.md "OCR").

// Shrinks the plate font below the block's raw line height so the recognized text fits
// inside the block box (the opaque box - sized by min-height - is what covers the source,
// independent of the font). Without it a tall title block wraps to more lines than the
// source and the plate grows past its region, colliding with the next plate. Shared with
// the desktop app's overlay.go fontFitFactor (see docs/PARITY.md). 0.92 keeps plate text close to
// the source size while leaving headroom for longer translations and word-wrap slack.
export const FONT_FIT = 0.92;

// Ceiling for the runtime grow branch, as a multiple of the compile-time size. Shared with the
// desktop app's ocrScript (see docs/PARITY.md). A block's box is the union of its lines and so
// includes the leading between them: filling it is not the same as matching the source's type, and
// on a loosely leaded block "fill the box" would print the translation larger than the words it
// covers. 1.15 is a little over 1/FONT_FIT, so a plate may reach the measured ink height of the
// source's own lines and no further.
export const FONT_GROW_CAP = 1.15;

// plateSpecs is the one place a recognized block becomes plate geometry. Positions and sizes are
// in percent of the source image so they survive responsive scaling, and the font size is in
// container-width units (cqw) derived from the block's median line height, scaled by FONT_FIT.
//
// It returns plain data on purpose: the page-OCR broker hands these across a message boundary to
// an agent running in the reader's document, which has no engine and must not compute plate
// geometry of its own. A second implementation of this arithmetic anywhere is the drift that
// docs/PARITY.md exists to prevent.
export function plateSpecs({ blocks, width, height }) {
  const specs = [];
  if (!width || !height) return specs;
  for (const b of blocks || []) {
    if (!b || !b.text) continue;
    const { x0, y0, x1, y1 } = b.bbox;
    specs.push({
      text: b.text,
      left: `${(x0 / width) * 100}%`,
      top: `${(y0 / height) * 100}%`,
      width: `${((x1 - x0) / width) * 100}%`,
      minHeight: `${((y1 - y0) / height) * 100}%`,
      fontSize: `${((b.lineHeight / width) * 100 * FONT_FIT).toFixed(2)}cqw`,
      ink: b.colors ? b.colors.ink : "",
      bg: b.colors ? b.colors.bg : "",
    });
  }
  return specs;
}

// renderPlates appends one opaque plate per spec to a container that is already sized to the
// picture and is an inline-size container query root (.ocr-overlay does both). It does not fit
// the text - call scheduleFit once the container is in the document.
export function renderPlates(container, specs) {
  for (const s of specs) {
    const plate = document.createElement("div");
    plate.className = "ocr-plate";
    plate.style.left = s.left;
    plate.style.top = s.top;
    plate.style.width = s.width;
    plate.style.minHeight = s.minHeight;
    plate.style.fontSize = s.fontSize;
    // Paper and ink both land on the plate box: the box is what covers the source region, so it is
    // what has to be opaque. The paper sat on an inline span hugging the string for one day, which
    // gave it the shape of the rendered words but left a mean 93% of the source lettering showing
    // around short strings against 17% for the box - see ocr-overlay.css and docs/PARITY.md for the
    // measurement, and overlay.go for the desktop mirror.
    if (s.ink || s.bg) { plate.style.color = s.ink; plate.style.background = s.bg; }
    plate.textContent = s.text;
    container.append(plate);
  }
  return container;
}

// A container the size of the image (via aspect-ratio) with the image as a base layer
// and one opaque plate per recognized block, positioned/sized in percent so it survives
// responsive scaling. Plates grow downward (min-height) so longer post-translation text
// wraps instead of clipping.
export function buildOverlay({ imageSrc, imageEl, blocks, width, height }) {
  const container = document.createElement("div");
  container.className = "ocr-overlay";
  if (width && height) container.style.aspectRatio = `${width} / ${height}`;

  const img = imageEl || document.createElement("img");
  if (!imageEl) img.src = imageSrc;
  img.classList.add("ocr-overlay-img");
  container.append(img);

  renderPlates(container, plateSpecs({ blocks, width, height }));
  liveFits.set(container, scheduleFit(container));
  return container;
}

// Every overlay buildOverlay made and has not released yet, with its fit's stop function. The
// viewer swaps documents inside one long-lived page, so an overlay's window listeners and
// observers outlive its image unless someone stops them; this is that someone.
const liveFits = new Map();

// releaseOverlays stops the fit of every overlay under root (root itself included). The viewer
// calls it before replacing a document. A fit also stops itself once it finds its container
// detached, which covers an overlay dropped any other way.
export function releaseOverlays(root) {
  for (const [container, stop] of liveFits) {
    if (root === container || (root && root.contains && root.contains(container))) {
      liveFits.delete(container);
      try { stop(); } catch { /* ignore */ }
    }
  }
}

// fitPlate fits one plate's text to its box: it shrinks the cqw font down to a floor, and if the
// text still overflows there it lets the box grow so nothing is ever clipped. The source region
// height (the inline min-height) is the target the font is fitted to.
//
// It also grows, because the compile-time size is deliberately conservative - the font is the
// median *ink* height times FONT_FIT, and an ink box is shorter than the type that drew it - so a
// plate whose text is no longer than the source's leaves the string floating in white space, which
// reads as an oversized patch rather than as the original lettering. Growth is capped at
// FONT_GROW_CAP x the base and stops one step before the content overflows, so a plate never prints
// larger than the region it covers. Mirrors the desktop app's ocrScript fit() (see docs/PARITY.md
// and overlay.go).
export function fitPlate(b) {
  if (!b.dataset.ocrCqw) {
    const m = /([0-9.]+)cqw/.exec(b.style.fontSize || "");
    b.dataset.ocrCqw = m ? m[1] : "0";
  }
  const base = parseFloat(b.dataset.ocrCqw);
  b.style.height = "";
  const target = parseFloat(getComputedStyle(b).minHeight) || 0;
  if (target > 0) b.style.height = target + "px";
  if (base > 0) {
    let s = base; const floor = base * 0.5; let g = 0;
    b.style.fontSize = s + "cqw";
    while (b.scrollHeight > b.clientHeight + 1 && s > floor && g < 40) {
      s -= Math.max(0.3, s * 0.08); g++;
      b.style.fontSize = s + "cqw";
    }
    if (b.scrollHeight <= b.clientHeight + 1) {
      const cap = base * FONT_GROW_CAP;
      let prev = s, n = s, gg = 0;
      while (n < cap && gg < 20) {
        n = Math.min(cap, n + Math.max(0.3, n * 0.04)); gg++;
        b.style.fontSize = n + "cqw";
        if (b.scrollHeight > b.clientHeight + 1) { b.style.fontSize = prev + "cqw"; break; }
        prev = n;
      }
    }
  }
  if (b.scrollHeight > b.clientHeight + 1) b.style.height = "auto";
}

// scheduleFit fits every plate in a container once it is laid out in the DOM (the caller appends it
// synchronously, so a setTimeout(0) sees it placed), and re-fits when the page translator swaps a
// plate's text or the container resizes - the compile-time font size is computed from the source
// geometry and cannot know the translated length. Mirrors the desktop app's ocrScript scheduling.
//
// Returns a stop function. The page agent needs it: its layers are removed while the document
// lives on, and observers left running on a detached container are a leak the viewer never had to
// care about because its page went away with the overlay.
export function scheduleFit(container) {
  let placed = false;
  let stopped = false;
  const fitAll = () => {
    if (stopped) return;
    if (!container.isConnected) {
      // Detached after having been placed: nothing will ever need fitting again.
      if (placed) stop();
      return;
    }
    placed = true;
    container.querySelectorAll(".ocr-plate").forEach(fitPlate);
  };
  let t;
  const go = () => { clearTimeout(t); t = setTimeout(fitAll, 0); };
  const stops = [];
  const stop = () => {
    if (stopped) return;
    stopped = true;
    clearTimeout(t);
    for (const s of stops) s();
    liveFits.delete(container);
  };
  go();
  if (typeof window !== "undefined") {
    window.addEventListener("load", go);
    window.addEventListener("resize", go);
    stops.push(() => { window.removeEventListener("load", go); window.removeEventListener("resize", go); });
  }
  if (typeof MutationObserver !== "undefined") {
    const mo = new MutationObserver((muts) => {
      for (const mu of muts) {
        let n = mu.target;
        while (n && n !== container) {
          if (n.classList && n.classList.contains("ocr-plate")) { go(); return; }
          n = n.parentNode;
        }
      }
    });
    mo.observe(container, { childList: true, characterData: true, subtree: true });
    stops.push(() => mo.disconnect());
  }
  if (typeof ResizeObserver !== "undefined") {
    try {
      const ro = new ResizeObserver(go);
      ro.observe(container);
      stops.push(() => ro.disconnect());
    } catch { /* ignore */ }
  }
  return stop;
}

// A progress badge (styled by .ocr-badge) callers overlay on a pending image.
export function makeBadge(text) {
  const badge = document.createElement("div");
  badge.className = "ocr-badge";
  badge.textContent = text;
  return badge;
}
