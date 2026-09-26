// export-html.js - the "save as HTML" document shell and its image encoding. Kept apart from viewer.js so it can be
// tested without the viewer's page.
//
// The saved file leaves the extension, and with it the extension's content policy - the only
// thing that kept a stray script URL inert in the viewer (ADR-1 of
// DEV/plan/done/19_2026-09-24_bugfix-extension-content-security.md). So the file carries its own
// policy: no script of any kind, no plugins, no forms, no base rewrite. Images and media may be
// remote because the reader may have allowed remote content in the viewer, and a parked
// (unallowed) URL sits in a data- attribute that no policy needs to cover.
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
