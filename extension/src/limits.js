// limits.js - the published input limits, the same numbers the desktop app enforces in
// internal/limits (docs/PARITY.md, "Input limits"). A browser tab has far less memory than
// the desktop process, so an archive is held against these from its listing, before anything
// is inflated, and every inflation counts its bytes as they arrive.
//
// Pure (no DOM) - unit-tested under node.

export const ARCHIVE_MAX_ENTRIES = 20000;
export const ARCHIVE_MAX_TOTAL_BYTES = 4 * 1024 * 1024 * 1024;
export const EPUB_MAX_ENTRY_BYTES = 100 * 1024 * 1024;
export const COMIC_MAX_PAGE_BYTES = 200 * 1024 * 1024;

// InputLimitError is a refusal on size. It carries its message key and arguments so the
// viewer can show it in the reader's language; message is the English text for logs and
// tests.
export class InputLimitError extends Error {
  constructor(key, fallback, ...args) {
    super(fill(fallback, args));
    this.key = key;
    this.fallback = fallback;
    this.args = args;
  }
}

function fill(text, args) {
  return text.replace(/\{(\d)\}/g, (m, i) => (args[i - 1] === undefined ? m : String(args[i - 1])));
}

// formatBytes renders a size in binary units, the way the limits are published. Matches
// internal/limits FormatBytes.
export function formatBytes(n) {
  const gb = 1024 ** 3;
  const mb = 1024 ** 2;
  if (n >= gb) return n % gb === 0 ? `${n / gb} GB` : `${(n / gb).toFixed(1)} GB`;
  if (n >= mb) return `${Math.floor(n / mb)} MB`;
  return `${Math.ceil(n / 1024)} KB`;
}

// checkArchive refuses a listing whose entry count or unpacked total (over the entries that
// will actually be inflated) is over the budget.
export function checkArchive(entries, total) {
  if (entries > ARCHIVE_MAX_ENTRIES) {
    throw new InputLimitError("vLimitEntries", "The archive has {1} entries, above the limit of {2}", entries, ARCHIVE_MAX_ENTRIES);
  }
  if (total > ARCHIVE_MAX_TOTAL_BYTES) {
    throw new InputLimitError("vLimitTotal", "The archive unpacks to {1}, above the limit of {2}",
      formatBytes(total), formatBytes(ARCHIVE_MAX_TOTAL_BYTES));
  }
}

export function entryTooLarge(name, limit) {
  return new InputLimitError("vLimitEntry", "{1} unpacks to more than {2}, the limit for one file", name, formatBytes(limit));
}

// inflateRawCapped decompresses a raw DEFLATE stream (ZIP method 8) with the platform's
// DecompressionStream, counting bytes as they arrive, and throws entryTooLarge the moment the
// output passes limit. The header's size is not trusted: a bomb can declare anything, and
// inflating it whole first is exactly what this replaces.
export async function inflateRawCapped(bytes, name, limit) {
  const reader = new Blob([bytes]).stream().pipeThrough(new DecompressionStream("deflate-raw")).getReader();
  const chunks = [];
  let total = 0;
  for (;;) {
    const { done, value } = await reader.read();
    if (done) break;
    total += value.length;
    if (total > limit) {
      await reader.cancel();
      throw entryTooLarge(name, limit);
    }
    chunks.push(value);
  }
  const out = new Uint8Array(total);
  let off = 0;
  for (const c of chunks) {
    out.set(c, off);
    off += c.length;
  }
  return out;
}
