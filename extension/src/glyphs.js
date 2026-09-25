// glyphs.js - the vocabulary glyphs the extension draws (ICON-SET section 2).
//
// Each value is the `d` of the catalog's 24 x 24 drawing, copied verbatim from
// assets/glyphs/<id>.svg. The desktop edition keeps its own copy (internal/htmlgen/glyphs.go);
// tests/iconography_test.go holds both tables to the vendored files, so one meaning is one picture
// in both editions (docs/PARITY.md "Vocabulary glyphs"). The drawings derive from Material Icons
// (Apache-2.0, THIRD-PARTY-NOTICES.txt). A new control picks a meaning that is already here or in
// the catalog - never a private picture (ICON-SET rule 5); docs/GLYPH-MAP.md maps every glyph.

const GLYPHS = {
  "nav.contents": "M3 7h2v2H3v-2zm0 4h2v2H3v-2zm0 4h2v2H3v-2zm4-8h14v2H7V7zm0 4h14v2H7v-2zm0 4h14v2H7v-2z",
  "nav.expand": "M16.59 8.59L12 13.17 7.41 8.59 6 10l6 6 6-6z",
  "nav.collapse": "M12 8l-6 6 1.41 1.41L12 10.83l4.59 4.58L18 14z",
  "nav.open-external": "M19,19H5V5h7V3H5c-1.11,0 -2,0.9 -2,2v14c0,1.1 0.89,2 2,2h14c1.1,0 2,-0.9 2,-2v-7h-2v7zM14,3v2h3.59l-9.83,9.83 1.41,1.41L19,6.41V10h2V3h-7z",
  "action.save": "M17,3H5c-1.11,0 -2,0.9 -2,2v14c0,1.1 0.89,2 2,2h14c1.1,0 2,-0.9 2,-2V7l-4,-4zM12,19c-1.66,0 -3,-1.34 -3,-3s1.34,-3 3,-3 3,1.34 3,3 -1.34,3 -3,3zM15,9H5V5h10v4z",
  "action.export": "M9,16h6v-6h4l-7,-7 -7,7h4zM5,18h14v2H5z",
  "action.text-smaller": "M.99 19h2.42l1.27-3.58h5.65L11.59 19h2.42L8.75 5h-2.5L.99 19zm4.42-5.61L7.44 7.6h.12l2.03 5.79H5.41zM23 11v2h-8v-2h8z",
  "action.text-larger": "M.99 19h2.42l1.27-3.58h5.65L11.59 19h2.42L8.75 5h-2.5L.99 19zm4.42-5.61L7.44 7.6h.12l2.03 5.79H5.41zM20 11h3v2h-3v3h-2v-3h-3v-2h3V8h2v3z",
  "view.text-layer": "M20 4H4c-1.1 0-2 .9-2 2v12c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V6c0-1.1-.9-2-2-2zM4 12h4v2H4v-2zm10 6H4v-2h10v2zm6 0h-4v-2h4v2zm0-4H10v-2h10v2z",
  "app.theme": "M12 22c5.52 0 10-4.48 10-10S17.52 2 12 2 2 6.48 2 12s4.48 10 10 10zm1-17.93c3.94.49 7 3.85 7 7.93s-3.05 7.44-7 7.93V4.07z",
  "nav.go-to-page": "M7 15H5.5v-4.5H4V9h3v6zm6.5-1.5h-3v-1h2c.55 0 1-.45 1-1V10c0-.55-.45-1-1-1H9v1.5h3v1h-2c-.55 0-1 .45-1 1V15h4.5v-1.5zm6 .5v-4c0-.55-.45-1-1-1H15v1.5h3v1h-2v1h2v1h-3V15h3.5c.55 0 1-.45 1-1z",
};

// Meanings drawn mirrored in a right-to-left layout (the record's `rtl: mirror`, ICON-RENDER
// rule 7). Everything else is fixed.
const MIRRORED = new Set(["nav.open-external"]);

const SVG_NS = "http://www.w3.org/2000/svg";

// glyph returns the drawing of a meaning as an inline, decorative SVG element: the control around
// it carries the meaning's name as text or aria-label (ICON-RENDER rule 8), and the colour is the
// surrounding text colour (rule 2).
function glyph(id) {
  const d = GLYPHS[id];
  if (!d) throw new Error(`glyphs: no vocabulary glyph ${id}`);
  const svg = document.createElementNS(SVG_NS, "svg");
  svg.setAttribute("class", MIRRORED.has(id) ? "glyph glyph-mirror" : "glyph");
  svg.setAttribute("viewBox", "0 0 24 24");
  svg.setAttribute("aria-hidden", "true");
  svg.setAttribute("focusable", "false");
  const path = document.createElementNS(SVG_NS, "path");
  path.setAttribute("fill", "currentColor");
  path.setAttribute("d", d);
  svg.append(path);
  return svg;
}

// applyGlyphs fills every `[data-glyph]` placeholder under root with its drawing. Static markup
// names the meaning instead of carrying a second copy of the path, so this table stays the
// edition's only one.
function applyGlyphs(root) {
  (root || document).querySelectorAll("[data-glyph]").forEach((el) => {
    el.replaceChildren(glyph(el.dataset.glyph));
  });
}

export { GLYPHS, MIRRORED, glyph, applyGlyphs };
