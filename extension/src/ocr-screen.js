// ocr-screen.js - the hand-port of the desktop app's internal/ocr/screen.go: find the halftone
// screen a picture is printed with, so the rescue ladder's last rung can low-pass it away with a
// kernel derived from the screen's own period. It lives in its own module, like ocr-cluster.js, so
// the arithmetic can be unit-tested without loading the Tesseract worker bundle.
//
// A halftone screen - the dot lattice a press lays down to print a tone - is the one thing the grey
// rescue ladder cannot see through. When the screen's own tone falls between the ink and the paper,
// a global thresholder either swallows the lettering with the dots or turns the whole picture into
// texture, and the mask that reaches recognition holds no text.
//
// Keep this file and screen.go in sync (docs/PARITY.md).

// OCR_SCREEN_SIGMA_DIVISOR turns the measured screen pitch into a Gaussian sigma. Chosen because it
// is at or next to the optimum on every screened image measured, at two different pitches and on
// real material as well as synthetic (DEV/research/ocr_halftone_2026-08-12.md). Shared invariant -
// see screen.go ocrScreenSigmaDivisor.
export const OCR_SCREEN_SIGMA_DIVISOR = 4;

// The detector. A dot lattice repeats at a fixed period, so after a high-pass the residual
// correlates with itself at that lag - no frequency transform needed, which is what makes the test
// cheap enough to run inside a rescue. Shared invariants, all of them: see screen.go.
export const OCR_SCREEN_TILE = 64; // side of the square the autocorrelation is taken over
export const OCR_SCREEN_MIN_PITCH = 3; // below this a "period" is JPEG noise or the sensor
export const OCR_SCREEN_MAX_PITCH = 24; // above this the lattice is coarser than any lettering
export const OCR_SCREEN_MAX_TILES = 96; // cap the work so a big page costs the same as a panel
export const OCR_SCREEN_MIN_ENERGY = 3; // a tile flatter than this is paper or solid ink
export const OCR_SCREEN_PEAK_FLOOR = 0.3; // autocorrelation at the winning lag, relative to lag 0
export const OCR_SCREEN_TILE_FRAC = 0.25; // share of textured tiles that must agree on one pitch

// OCR_SCREEN_TILE_COVER_MAX is how much of a tile an existing plate may cover before the tile stops
// counting as evidence of screened area the reader is not served on. Half rather than "touches at
// all": on a dense page a plate clipping a tile's corner would otherwise blind the detector to a
// screened caption standing right beside a balloon, which is the case the additive sweep exists for.
// Shared invariant - see screen.go ocrScreenTileCoverMax.
export const OCR_SCREEN_TILE_COVER_MAX = 0.5;

// tileStep spaces the sampled tiles so no more than OCR_SCREEN_MAX_TILES of them fit, whatever the
// image size. Never below the tile itself, so tiles never overlap.
export function tileStep(width, height) {
  const tiles = Math.floor(width / OCR_SCREEN_TILE) * Math.floor(height / OCR_SCREEN_TILE);
  if (tiles <= OCR_SCREEN_MAX_TILES) return OCR_SCREEN_TILE;
  return OCR_SCREEN_TILE * Math.ceil(Math.sqrt(tiles / OCR_SCREEN_MAX_TILES));
}

// tileServed reports whether one of the covered rectangles already takes more than
// OCR_SCREEN_TILE_COVER_MAX of this tile. Each rectangle is tested on its own rather than as a
// union: two plates that between them cover a tile are two separate regions with a gap down the
// middle, and that gap is exactly where an unserved caption would sit.
function tileServed(tx, ty, covered) {
  const limit = Math.trunc(OCR_SCREEN_TILE_COVER_MAX * OCR_SCREEN_TILE * OCR_SCREEN_TILE);
  for (const c of covered) {
    const w = Math.min(c.x1, tx + OCR_SCREEN_TILE) - Math.max(c.x0, tx);
    const h = Math.min(c.y1, ty + OCR_SCREEN_TILE) - Math.max(c.y0, ty);
    if (w > 0 && h > 0 && w * h > limit) return true;
  }
  return false;
}

// screenPitch returns the period in pixels of the halftone screen covering the image, or 0 when
// there is no lattice to find. `grey` is one luminance byte per pixel, row-major.
//
// `covered` restricts the answer to the parts of the picture no rectangle reaches, which turns the
// question into the one the additive sweep asks: not "does this picture carry a screen" but "is
// there screened area the reader has no plate over". A screen under lettering that is already plated
// has nothing left to give, and the sweep it would trigger costs a whole recognition. The vote share
// OCR_SCREEN_TILE_FRAC then applies to the tiles that survive the skip - a quarter of the *unserved*
// textured tiles must agree. Mirrors screen.go screenPitchOutside (docs/PARITY.md).
//
// Requiring agreement across tiles is what separates a press screen from an accident: a screen
// covers an area with one period, whereas JPEG blocking, film grain and engraved hatching either
// vary tile to tile or fall outside the pitch bounds.
export function screenPitch(grey, width, height, covered = []) {
  if (width < OCR_SCREEN_TILE || height < OCR_SCREEN_TILE) return 0;
  const step = tileStep(width, height);
  const buf = new Float64Array(OCR_SCREEN_TILE * OCR_SCREEN_TILE);
  const votes = new Map();
  let textured = 0;

  for (let ty = 0; ty + OCR_SCREEN_TILE <= height; ty += step) {
    for (let tx = 0; tx + OCR_SCREEN_TILE <= width; tx += step) {
      if (tileServed(tx, ty, covered)) continue;
      if (!tileResidual(grey, width, height, tx, ty, buf)) continue;
      textured++;
      const [lag, peak] = tilePeriod(buf);
      if (lag > 0 && peak >= OCR_SCREEN_PEAK_FLOOR) votes.set(lag, (votes.get(lag) || 0) + 1);
    }
  }
  if (!textured) return 0;

  let best = 0;
  let agreed = 0;
  for (const [lag, n] of votes) {
    if (n > agreed || (n === agreed && lag < best)) { best = lag; agreed = n; }
  }
  if (best < OCR_SCREEN_MIN_PITCH || best > OCR_SCREEN_MAX_PITCH) return 0;
  if (agreed < OCR_SCREEN_TILE_FRAC * textured) return 0;
  return best;
}

// OCR_SCREEN_MERGE_MAX_OVERLAP is how much of a screen-pass plate may already be covered by the
// plates the ordinary pass produced before it is dropped as a duplicate. A fraction of the *new*
// plate rather than of the existing one, because the question is "is this lettering already plated"
// and a small candidate sitting wholly inside a large existing plate is a duplicate however little
// of that plate it occupies. Small because the two outcomes are not symmetric: an untranslated
// caption is a miss, a second plate over lettering that already has one is visible damage. Shared
// invariant - see screen.go ocrScreenMergeMaxOverlap.
export const OCR_SCREEN_MERGE_MAX_OVERLAP = 0.2;

// mergeScreenBlocks returns kept, unchanged and in order, followed by those of found that the plates
// already accepted leave mostly uncovered - including the screen plates taken earlier in the same
// call, so two of them cannot stack on each other either. Every plate the ordinary pass produced
// survives: the sweep is additive, and a pass that could move or drop an existing plate would put a
// page that reads fine today at risk to help one that does not. Mirrors screen.go mergeScreenBlocks.
export function mergeScreenBlocks(kept, found) {
  const out = kept.slice();
  const taken = kept.map((b) => b.bbox);
  for (const b of found) {
    if (coveredFraction(b.bbox, taken) > OCR_SCREEN_MERGE_MAX_OVERLAP) continue;
    out.push(b);
    taken.push(b.bbox);
  }
  return out;
}

// coveredFraction returns how much of r lies inside the union of rects, as a fraction of r's area.
//
// The union rather than the sum of the overlaps, and that is the whole point: a candidate whose two
// halves lie under two different existing plates is entirely covered, but a rule that took each
// existing plate on its own would see two halves under the bound and let it through. Summing instead
// double-counts wherever the existing plates overlap each other and can report more than the whole.
// Computed exactly, by compressing the rectangles' own edge coordinates into a grid - every cell is
// either wholly inside a given overlap or wholly outside it.
export function coveredFraction(r, rects) {
  const area = (r.x1 - r.x0) * (r.y1 - r.y0);
  if (area <= 0) return 0;
  const parts = [];
  const xs = [r.x0, r.x1];
  const ys = [r.y0, r.y1];
  for (const o of rects) {
    const x0 = Math.max(o.x0, r.x0);
    const y0 = Math.max(o.y0, r.y0);
    const x1 = Math.min(o.x1, r.x1);
    const y1 = Math.min(o.y1, r.y1);
    if (x1 <= x0 || y1 <= y0) continue;
    parts.push({ x0, y0, x1, y1 });
    xs.push(x0, x1);
    ys.push(y0, y1);
  }
  if (!parts.length) return 0;
  xs.sort((a, b) => a - b);
  ys.sort((a, b) => a - b);

  let covered = 0;
  for (let i = 0; i + 1 < xs.length; i++) {
    for (let j = 0; j + 1 < ys.length; j++) {
      const cw = xs[i + 1] - xs[i];
      const ch = ys[j + 1] - ys[j];
      if (cw <= 0 || ch <= 0) continue;
      for (const p of parts) {
        if (xs[i] >= p.x0 && xs[i + 1] <= p.x1 && ys[j] >= p.y0 && ys[j + 1] <= p.y1) {
          covered += cw * ch;
          break;
        }
      }
    }
  }
  return covered / area;
}

// tileResidual fills buf with the tile's high-pass residual (the pixel minus its 3x3 mean, which is
// where a screen lives) and reports whether the tile carries enough of it to be worth testing.
function tileResidual(grey, width, height, tx, ty, buf) {
  const at = (x, y) => grey[Math.min(Math.max(y, 0), height - 1) * width + Math.min(Math.max(x, 0), width - 1)];
  let energy = 0;
  for (let j = 0; j < OCR_SCREEN_TILE; j++) {
    for (let i = 0; i < OCR_SCREEN_TILE; i++) {
      const x = tx + i;
      const y = ty + j;
      let mean = 0;
      for (let dy = -1; dy <= 1; dy++) {
        for (let dx = -1; dx <= 1; dx++) mean += at(x + dx, y + dy);
      }
      const v = at(x, y) - mean / 9;
      buf[j * OCR_SCREEN_TILE + i] = v;
      energy += v * v;
    }
  }
  return energy / (OCR_SCREEN_TILE * OCR_SCREEN_TILE) >= OCR_SCREEN_MIN_ENERGY;
}

// tilePeriod returns [lag, peak] for the lag whose normalized autocorrelation is highest in the
// tile. Both axes are summed into one sequence because a press screen is normally rotated - it
// still repeats along x and along y, but neither axis alone carries the whole signal.
//
// Only a local maximum counts. A smooth gradient decorrelates monotonically, so without that rule
// every gradient would report the smallest lag as its "period".
function tilePeriod(buf) {
  const acf = new Float64Array(OCR_SCREEN_MAX_PITCH + 1);
  for (let lag = 0; lag <= OCR_SCREEN_MAX_PITCH; lag++) {
    let sum = 0;
    for (let y = 0; y < OCR_SCREEN_TILE; y++) {
      const row = y * OCR_SCREEN_TILE;
      for (let x = 0; x + lag < OCR_SCREEN_TILE; x++) sum += buf[row + x] * buf[row + x + lag];
    }
    for (let x = 0; x < OCR_SCREEN_TILE; x++) {
      for (let y = 0; y + lag < OCR_SCREEN_TILE; y++) {
        sum += buf[y * OCR_SCREEN_TILE + x] * buf[(y + lag) * OCR_SCREEN_TILE + x];
      }
    }
    acf[lag] = sum;
  }
  if (acf[0] <= 0) return [0, 0];

  let bestLag = 0;
  let bestVal = 0;
  for (let lag = 2; lag <= OCR_SCREEN_MAX_PITCH; lag++) {
    const v = acf[lag] / acf[0];
    if (v <= bestVal || v < acf[lag - 1] / acf[0]) continue;
    if (lag < OCR_SCREEN_MAX_PITCH && v < acf[lag + 1] / acf[0]) continue;
    bestLag = lag;
    bestVal = v;
  }
  return [bestLag, bestVal];
}
