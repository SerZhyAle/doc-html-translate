// reading-position.js - resume reading in the viewer (ticket 53).
//
// The viewer remembers where the reader stopped in each document and offers a
// "Continue reading" action when the same document reopens. Everything here is local
// to the browser profile: the map of positions lives under one chrome.storage.local
// key, nothing about a document or its reading history ever leaves the device, and a
// stored record carries only numbers and an element id - never a URL or document text.
//
// Document identity mirrors the desktop reader's key (internal/htmlgen.ReaderKey,
// docs/PARITY.md "Reading position"): an FNV-1a 64 hash over the source file's name
// and byte size plus the title and page count as extracted, before any translation can
// rewrite the title. Two documents that merely share a displayed title do not collide
// (their name or size differs); a genuine reopen hashes to the same key. When an input
// is missing the key is "" and resume is omitted entirely - a document that cannot be
// identified reliably is never handed another book's place.
//
// The saved shape prefers a stable anchor over a raw scroll fraction: the 1-based
// page index the sections carry in data-page, the id of the deepest element at or
// above the reading line inside that section, and pixel offsets from that element and
// from the section's top. Images that load late or a changed text size move content
// below the anchor, so restoring from the anchor lands where the reader stopped; if
// the element is gone the section offset is the fallback.

const POSITIONS_KEY = "readingPositions"; // the single chrome.storage.local key
const POSITIONS_MAX = 100;                // documents remembered; the oldest falls off
const SAVE_THROTTLE_MS = 2000;            // scrolling coalesces into one write per tick
const FRAG_MAX = 128;                     // ids are renderer-namespaced; cap defensively

// fnv1a64 hashes bytes to 16 hex digits - the same FNV-1a the desktop's ReaderKey
// derives its key with, over the same input shape.
export function fnv1a64(bytes) {
  let h = 0xcbf29ce484222325n;
  for (const b of bytes) {
    h ^= BigInt(b);
    h = (h * 0x100000001b3n) & 0xffffffffffffffffn;
  }
  return h.toString(16).padStart(16, "0");
}

// readerKey derives a document's position key, or "" when it cannot be identified
// reliably. The key never stores the inputs themselves - only their hash.
export function readerKey(name, size, title, pages) {
  if (!name || !(size > 0) || !title || !(pages > 0)) return "";
  return fnv1a64(new TextEncoder().encode(`${name}\x00${size}\x00${title}\x00${pages}`));
}

// readingLine is where the reader is looking: a third of the viewport down, the same
// line the page selector's scroll sync judges sections by.
function readingLine(win) {
  const viewport = Number(win.innerHeight) > 0 ? win.innerHeight : 0;
  return (win.scrollY || 0) + viewport / 3;
}

// currentPosition reads the live document. Null when nothing is rendered (a notice
// screen) - nothing is saved then.
export function currentPosition(doc, win) {
  const sections = doc.querySelectorAll("#content section[data-page]");
  if (!sections.length) return null;
  const mid = readingLine(win);
  let sec = sections[0];
  for (const s of sections) {
    if (s.offsetTop > mid) break;
    sec = s;
  }
  const page = Number(sec.dataset.page) || 1;
  let frag = "";
  let off = Math.max(0, Math.round(mid - sec.offsetTop));
  const secoff = off;
  for (const n of sec.querySelectorAll("[id]")) {
    if (n === sec) continue;
    if (n.offsetTop > mid) break;
    frag = String(n.id || "");
    off = Math.max(0, Math.round(mid - n.offsetTop));
  }
  if (frag.length > FRAG_MAX) frag = "";
  return { page, frag, off, secoff };
}

// resolvePositionTarget maps a saved position onto the live document: the saved
// anchor while it still exists inside its section, else the section itself with the
// section-level offset. Null when that section is not in the DOM (a chunked PDF the
// caller has not rendered forward to yet).
export function resolvePositionTarget(pos, doc) {
  if (!pos || !(pos.page > 0)) return null;
  let sec = null;
  for (const s of doc.querySelectorAll("#content section[data-page]")) {
    if (Number(s.dataset.page) === pos.page) { sec = s; break; }
  }
  if (!sec) return null;
  let el = sec;
  let off = Math.max(0, Math.round(pos.secoff || 0));
  if (pos.frag) {
    const f = doc.getElementById(pos.frag);
    if (f && sec.contains(f)) {
      el = f;
      off = Math.max(0, Math.round(pos.off || 0));
    }
  }
  return { el, off };
}

// mergePosition sets one document's position and prunes the map to the newest
// POSITIONS_MAX entries by their save time, so storage cannot grow without limit.
// Pure: returns a new map.
export function mergePosition(store, key, pos, now) {
  if (!key || !(pos && pos.page > 0)) return prunePositions(store);
  const next = keepOtherEntries(store, key);
  next[key] = {
    page: Math.round(pos.page),
    frag: typeof pos.frag === "string" ? pos.frag.slice(0, FRAG_MAX) : "",
    off: Math.max(0, Math.round(pos.off || 0)),
    secoff: Math.max(0, Math.round(pos.secoff || 0)),
    at: now,
  };
  return prunePositions(next);
}

// removePosition drops one document's position ("Start over"). Pure.
export function removePosition(store, key) {
  return keepOtherEntries(store, key);
}

function keepOtherEntries(store, key) {
  const next = {};
  for (const [k, v] of Object.entries(store || {})) {
    if (k !== key && v && typeof v === "object") next[k] = v;
  }
  return next;
}

function prunePositions(store) {
  const entries = Object.entries(store || {}).filter(([, v]) => v && typeof v === "object");
  if (entries.length <= POSITIONS_MAX) return Object.fromEntries(entries);
  entries.sort((a, b) => (b[1].at || 0) - (a[1].at || 0));
  return Object.fromEntries(entries.slice(0, POSITIONS_MAX));
}

export { POSITIONS_KEY, POSITIONS_MAX, SAVE_THROTTLE_MS };
