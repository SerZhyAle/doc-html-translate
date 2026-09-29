// export-html.js - the "save as HTML" document shell, its image encoding, and the completeness
// decision behind the export. Kept apart from viewer.js so it can be tested without the viewer's page.
//
// The saved file leaves the extension, and with it the extension's content policy - the only
// thing that kept a stray script URL inert in the viewer (ADR-1 of
// DEV/plan/done/19_2026-09-24_bugfix-extension-content-security.md). So the file carries its own
// policy: no script of any kind, no plugins, no forms, no base rewrite. Images and media may be
// remote because the reader may have allowed remote content in the viewer, and a parked
// (unallowed) URL sits in a data- attribute that no policy needs to cover.

import { formatBytes } from "./limits.js";
export const EXPORT_CSP = [
  "default-src 'none'",
  "img-src data: http: https:",
  "media-src data: http: https:",
  "style-src 'unsafe-inline'",
  "font-src data:",
  "script-src 'none'",
  "object-src 'none'",
  "base-uri 'none'",
  "form-action 'none'",
].join("; ");

// Rows read per getImageData call while looking for transparency: a whole comic page at once is
// tens of megabytes of pixel copy just to learn one bit.
const ALPHA_SCAN_ROWS = 256;

// exportImageEncoding picks how an image is re-encoded into the saved file. JPEG has no alpha
// channel, so a transparent picture (a diagram, a formula, line art) turned into a black box;
// any pixel short of opaque keeps a lossless PNG. An opaque image - a scan, a comic panel, a
// photo - stays JPEG, which is several times smaller. ctx holds the image drawn at 0,0 on an
// otherwise untouched canvas.
export function exportImageEncoding(ctx, width, height) {
  for (let y = 0; y < height; y += ALPHA_SCAN_ROWS) {
    const rows = Math.min(ALPHA_SCAN_ROWS, height - y);
    const data = ctx.getImageData(0, y, width, rows).data;
    for (let i = 3; i < data.length; i += 4) {
      if (data[i] < 255) return { type: "image/png" };
    }
  }
  return { type: "image/jpeg", quality: 0.85 };
}

export function escapeHtml(s) {
  return String(s).replace(/[&<>"]/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;" }[c]));
}

// The policy meta sits right after the charset: a meta policy covers only what the parser meets
// after it.
export function buildExportHtml({ title, theme, lang, styleVars, css, body }) {
  const attrs = `lang="${escapeHtml(lang)}" data-theme="${escapeHtml(theme)}"` +
    (styleVars ? ` style="${escapeHtml(styleVars)}"` : "");
  return `<!DOCTYPE html>
<html ${attrs}>
<head>
<meta charset="UTF-8">
<meta http-equiv="Content-Security-Policy" content="${EXPORT_CSP}">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<meta name="referrer" content="no-referrer">
<title>${escapeHtml(title)}</title>
<style>
${css}
</style>
</head>
<body>
${body}
</body>
</html>
`;
}

// ---- Completeness planning -------------------------------------------------
// A paged document - a chunk-rendered PDF or a scroll-inflated comic - may hold fewer pages in
// the live #content than the source declares, and the export serializes exactly what is on
// screen. exportPlan turns the viewer's counters into one of three decisions:
//
//   direct        - every page is materialized; save at once (the short path).
//   offer-prepare - the export would be partial, but preparing the rest here in the viewer
//                   stays within the budget below, so the reader is offered it.
//   partial-only  - the export would be partial and a complete one is refused for this
//                   document; the reason travels so the dialog can explain before any file
//                   is written.
//
// The budget is deliberately conservative. Nothing here loads a whole document at once - the
// viewer's own chunked rendering and per-page inflation stay in charge - and a preparation is
// additionally stopped the moment its accumulated image bytes pass the file budget (the
// viewer counts them as pages hand them over). Pure (no DOM) - unit-tested under node.

// Most pages a preparation may still have to render. Past this, finishing the book in the tab
// is exactly the unbounded render the chunking exists to avoid (PAGE_CHUNK in viewer.js), so
// only the partial export is offered.
export const EXPORT_PREPARE_MAX_PAGES = 1000;

// Largest estimated complete file a preparation may aim at. Every image rides in the saved
// file as a data: URI, and the export holds about three copies of that in the tab (the map of
// encoded images, the serialized clone, the final document string), so past this size the
// complete export is what would fall over, not the reading.
export const EXPORT_PREPARE_MAX_FILE_BYTES = 150 * 1024 * 1024;

// base64 grows bytes by a factor of 4/3; the rest is slack for the HTML around the images.
const DATA_URI_GROWTH = 1.37;

export function estimateExportBytes(imageBytes) {
  return Math.ceil(imageBytes * DATA_URI_GROWTH);
}

// partialOnlyReason shapes the refusal the dialog shows the reader. Key/fallback/args mirror
// InputLimitError (limits.js) so the viewer renders it with the same t() call.
function partialOnlyReason(key, fallback, args) {
  return { key, fallback, args };
}

export function exportPlan({ total = 0, rendered = 0, imageBytes = 0 } = {}) {
  if (total > 0 && rendered < total) {
    const remaining = total - rendered;
    if (remaining > EXPORT_PREPARE_MAX_PAGES) {
      return {
        action: "partial-only",
        reason: partialOnlyReason(
          "vExportReasonPages",
          "preparing the remaining {1} pages is above the limit of {2}",
          [remaining, EXPORT_PREPARE_MAX_PAGES],
        ),
      };
    }
    // Comic pages declare their inflated size in the archive listing, so the complete file's
    // size is known before anything is prepared. A 0 means "unknown here" (a PDF's rasters are
    // counted during preparation instead) and never refuses on its own.
    const estimated = estimateExportBytes(imageBytes);
    if (imageBytes > 0 && estimated > EXPORT_PREPARE_MAX_FILE_BYTES) {
      return {
        action: "partial-only",
        reason: partialOnlyReason(
          "vExportReasonBytes",
          "the complete file would be about {1}, more than a browser tab holds reliably",
          [formatBytes(estimated)],
        ),
      };
    }
    return { action: "offer-prepare", remaining };
  }
  return { action: "direct" };
}
