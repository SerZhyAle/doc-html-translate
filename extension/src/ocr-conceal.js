// ocr-conceal.js - how a plate conceals the source lettering under it: the concealment-mode
// decision and the plate background each mode paints. Pure arithmetic over a pixel sampler, so it
// runs (and is tested) without a canvas. Mirrors internal/ocr/conceal.go - the mode names, the
// constants and the rule are a shared contract (docs/PARITY.md "OCR", TestParityOCRConcealment).
//
// Every mode is CSS over the untouched <img>: the source picture is never modified, and hiding the
// overlay shows it exactly as it was.

// The data-ocr-mode values, and the lab's evidence.Mode strings.
export const MODE_FILL = "fill";               // the sampled paper over the block rectangle - the plate as it always was
export const MODE_RECONSTRUCT = "reconstruct"; // a gradient between two opposite sides of the ring
export const MODE_MASK = "mask";               // paper over the line boxes only; the rest stays transparent

// Concealment-mode decision, shared verbatim with conceal.go and held equal by
// TestParityOCRConcealment. Bracketed over every block of the 47-scene corpus plus the synthetic
// scenes (DEV/research/RESEARCH_ocr-concealment-modes_2026-09-26.md); the reasons are on the Go side.
// OCR-OVERLAY rule 13: derived - RESEARCH_ocr-concealment-modes_2026-09-26 (OCR-PIPELINE amendment
// 1.3 D).
const MODE_BUSY_MAX = 0.06;      // share of ring pixels off their side's median above which the ring is busy
// OCR-OVERLAY rule 13: derived - RESEARCH_ocr-concealment-modes_2026-09-26 (OCR-PIPELINE amendment
// 1.3 D).
const MODE_FLAT_SPREAD = 40;     // side-to-side median distance above which two sides differ
// OCR-OVERLAY rule 13: policy - covers an antialiased edge and a descender, leaves most of the
// leading open; not measured.
const MODE_MASK_PAD_DIVISOR = 6; // a masked line box grows by its line height over this on each side

const dist = (a, b) => Math.abs(a[0] - b[0]) + Math.abs(a[1] - b[1]) + Math.abs(a[2] - b[2]);

function medianOf(v) {
  if (!v.length) return 0;
  const s = v.slice().sort((a, b) => a - b);
  return s[Math.floor(s.length / 2)];
}

function sideOf(sample, x0, y0, x1, y1, deviation) {
  const side = { med: [0, 0, 0], busy: 0, n: 0, hasMed: false };
  if (x1 - x0 < 1 || y1 - y0 < 1) return side;
  const { rs, gs, bs } = sample(x0, y0, x1 - x0, y1 - y0);
  if (!rs.length) return side;
  side.med = [medianOf(rs), medianOf(gs), medianOf(bs)];
  side.hasMed = true;
  side.n = rs.length;
  for (let i = 0; i < rs.length; i++) {
    if (Math.abs(rs[i] - side.med[0]) + Math.abs(gs[i] - side.med[1]) + Math.abs(bs[i] - side.med[2]) > deviation) side.busy++;
  }
  return side;
}

// measureRing samples the four sides of the band just outside the block. The geometry is
// ringNearerInk's (ocr-overlay.js): lh / padDivisor wide, at least minPad, clamped to the picture,
// with the corners belonging to the top and bottom sides. `ring` carries the shared colour-sampling
// constants from ocr-overlay.js, so they have one declaration in this edition.
// Mirrors conceal.go measureRing.
export function measureRing(sample, bbox, lineHeight, W, H, ring) {
  const clamp = (v, hi) => Math.max(0, Math.min(hi, v));
  const x0 = clamp(Math.floor(bbox.x0), W), y0 = clamp(Math.floor(bbox.y0), H);
  const x1 = clamp(Math.ceil(bbox.x1), W), y1 = clamp(Math.ceil(bbox.y1), H);
  const lh = Math.round(lineHeight) >= 1 ? Math.round(lineHeight) : y1 - y0;
  const pad = Math.max(ring.minPad, Math.floor(lh / ring.padDivisor));
  const ox0 = clamp(x0 - pad, W), oy0 = clamp(y0 - pad, H);
  const ox1 = clamp(x1 + pad, W), oy1 = clamp(y1 + pad, H);
  const r = {
    top: sideOf(sample, ox0, oy0, ox1, y0, ring.deviation),
    bottom: sideOf(sample, ox0, y1, ox1, oy1, ring.deviation),
    left: sideOf(sample, ox0, y0, x0, y1, ring.deviation),
    right: sideOf(sample, x1, y0, ox1, y1, ring.deviation),
    busy: 0, n: 0, minSamples: ring.minSamples, minPad: ring.minPad,
  };
  for (const s of [r.top, r.bottom, r.left, r.right]) { r.busy += s.busy; r.n += s.n; }
  return r;
}

// spread is 0 when either side is missing: a block against the picture's edge has no band there.
function spread(a, b) {
  return a.hasMed && b.hasMed ? dist(a.med, b.med) : 0;
}

// gradientAxis: a ramp along the axis with the larger spread, whose two cross sides sit in the
// middle third of it - which is what tells a gradient from an edge beside the block (a panel rule
// next to a white caption differs from the paper just as much). Mirrors conceal.go gradientAxis.
export function gradientAxis(r) {
  let from = r.top, to = r.bottom, dir = "to bottom", a = r.left, b = r.right;
  if (spread(r.left, r.right) > spread(r.top, r.bottom)) {
    from = r.left; to = r.right; dir = "to right"; a = r.top; b = r.bottom;
  }
  if (spread(from, to) <= MODE_FLAT_SPREAD || !a.hasMed || !b.hasMed) return { from, to, dir, ok: false };
  const mid = { hasMed: true, med: [0, 1, 2].map((c) => Math.floor((from.med[c] + to.med[c]) / 2)) };
  const s = spread(from, to);
  return { from, to, dir, ok: 6 * spread(a, mid) <= s && 6 * spread(b, mid) <= s };
}

const clamp01 = (v) => Math.min(1, Math.max(0, v));

// decideMode: a ring too thin to judge is a mask at confidence 0; a busy ring is a mask; a
// consistent ramp is a reconstruction; anything else keeps the fill. Mirrors conceal.go decideMode.
export function decideMode(r) {
  if (r.n < r.minSamples) return { mode: MODE_MASK, conf: 0 };
  const busy = r.busy / r.n;
  if (busy > MODE_BUSY_MAX) return { mode: MODE_MASK, conf: clamp01((busy - MODE_BUSY_MAX) / MODE_BUSY_MAX) };
  const conf = clamp01(1 - busy / MODE_BUSY_MAX);
  return { mode: gradientAxis(r).ok ? MODE_RECONSTRUCT : MODE_FILL, conf };
}

const rgb = (c) => `rgb(${c[0]},${c[1]},${c[2]})`;

// plateBackground is the CSS background for a plate in the given mode, or "" to keep the fill.
// The mask's stripes are in cqw from the plate's own corner - the container is the picture, so
// they stay on the source lines at every viewport and do not stretch when the fit grows the plate.
// Mirrors conceal.go plateBackground.
export function plateBackground(mode, r, block, paper, W) {
  if (mode === MODE_RECONSTRUCT) {
    const { from, to, dir } = gradientAxis(r);
    return `linear-gradient(${dir},${rgb(from.med)},${rgb(to.med)})`;
  }
  if (mode === MODE_MASK) {
    const lines = block.lines || [];
    if (!paper || !(W > 0) || !lines.length) return "";
    const { x0: bx0, y0: by0, y1: by1 } = block.bbox;
    const lh = Math.round(block.lineHeight) >= 1 ? Math.round(block.lineHeight) : by1 - by0;
    const pad = Math.max(r.minPad, Math.floor(lh / MODE_MASK_PAD_DIVISOR));
    const cq = (v) => ((v / W) * 100).toFixed(3);
    return lines.map((l) => `linear-gradient(${paper},${paper}) ${cq(l.x0 - pad - bx0)}cqw ${cq(l.y0 - pad - by0)}cqw/${cq(l.x1 - l.x0 + 2 * pad)}cqw ${cq(l.y1 - l.y0 + 2 * pad)}cqw no-repeat`).join(",");
  }
  return "";
}

// conceal decides a block's mode and its background in one call - what the overlay stores on the
// block for plateSpecs to carry across. `paper` is blockColors' sampled bg ("" without one).
export function conceal(sample, block, W, H, ring, paper) {
  const r = measureRing(sample, block.bbox, block.lineHeight, W, H, ring);
  const { mode, conf } = decideMode(r);
  return { mode, conf, background: plateBackground(mode, r, block, paper, W) };
}
