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
  "nav.close": "M19,6.41L17.59,5 12,10.59 6.41,5 5,6.41 10.59,12 5,17.59 6.41,19 12,13.41 17.59,19 19,17.59 13.41,12z",
  "nav.expand": "M16.59 8.59L12 13.17 7.41 8.59 6 10l6 6 6-6z",
  "nav.collapse": "M12 8l-6 6 1.41 1.41L12 10.83l4.59 4.58L18 14z",
  "nav.open-external": "M19,19H5V5h7V3H5c-1.11,0 -2,0.9 -2,2v14c0,1.1 0.89,2 2,2h14c1.1,0 2,-0.9 2,-2v-7h-2v7zM14,3v2h3.59l-9.83,9.83 1.41,1.41L19,6.41V10h2V3h-7z",
  "action.search": "M15.5 14h-.79l-.28-.27A6.471 6.471 0 0 0 16 9.5 6.5 6.5 0 1 0 9.5 16c1.61 0 3.09-.59 4.23-1.57l.27.28v.79l5 4.99L20.49 19l-4.99-5zm-6 0C7.01 14 5 11.99 5 9.5S7.01 5 9.5 5 14 7.01 14 9.5 11.99 14 9.5 14z",
  "action.save": "M17,3H5c-1.11,0 -2,0.9 -2,2v14c0,1.1 0.89,2 2,2h14c1.1,0 2,-0.9 2,-2V7l-4,-4zM12,19c-1.66,0 -3,-1.34 -3,-3s1.34,-3 3,-3 3,1.34 3,3 -1.34,3 -3,3zM15,9H5V5h10v4z",
  "action.export": "M9,16h6v-6h4l-7,-7 -7,7h4zM5,18h14v2H5z",
  "action.text-smaller": "M.99 19h2.42l1.27-3.58h5.65L11.59 19h2.42L8.75 5h-2.5L.99 19zm4.42-5.61L7.44 7.6h.12l2.03 5.79H5.41zM23 11v2h-8v-2h8z",
  "action.text-larger": "M.99 19h2.42l1.27-3.58h5.65L11.59 19h2.42L8.75 5h-2.5L.99 19zm4.42-5.61L7.44 7.6h.12l2.03 5.79H5.41zM20 11h3v2h-3v3h-2v-3h-3v-2h3V8h2v3z",
  "view.text-layer": "M20 4H4c-1.1 0-2 .9-2 2v12c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V6c0-1.1-.9-2-2-2zM4 12h4v2H4v-2zm10 6H4v-2h10v2zm6 0h-4v-2h4v2zm0-4H10v-2h10v2z",
  "app.theme": "M12 22c5.52 0 10-4.48 10-10S17.52 2 12 2 2 6.48 2 12s4.48 10 10 10zm1-17.93c3.94.49 7 3.85 7 7.93s-3.05 7.44-7 7.93V4.07z",
  "app.night-mode": "M12,3c-4.97,0 -9,4.03 -9,9s4.03,9 9,9s9,-4.03 9,-9c0,-0.46 -0.04,-0.92 -0.1,-1.36c-0.98,1.37 -2.58,2.26 -4.4,2.26c-2.98,0 -5.4,-2.42 -5.4,-5.4c0,-1.81 0.89,-3.42 2.26,-4.4C13.92,3.04 12.46,3 12,3L12,3z",
  "action.reset": "M14,12c0,-1.1 -0.9,-2 -2,-2s-2,0.9 -2,2 0.9,2 2,2 2,-0.9 2,-2zM12,3c-4.97,0 -9,4.03 -9,9L0,12l4,4 4,-4L5,12c0,-3.87 3.13,-7 7,-7s7,3.13 7,7 -3.13,7 -7,7c-1.51,0 -2.91,-0.49 -4.06,-1.3l-1.42,1.44C8.04,20.3 9.94,21 12,21c4.97,0 9,-4.03 9,-9s-4.03,-9 -9,-9z",
  "nav.go-to-page": "M7 15H5.5v-4.5H4V9h3v6zm6.5-1.5h-3v-1h2c.55 0 1-.45 1-1V10c0-.55-.45-1-1-1H9v1.5h3v1h-2c-.55 0-1 .45-1 1V15h4.5v-1.5zm6 .5v-4c0-.55-.45-1-1-1H15v1.5h3v1h-2v1h2v1h-3V15h3.5c.55 0 1-.45 1-1z",
  "feature.continue-reading": "M21,5c-1.11,-0.35 -2.33,-0.5 -3.5,-0.5 -1.95,0 -4.05,0.4 -5.5,1.5 -1.45,-1.1 -3.55,-1.5 -5.5,-1.5S2.45,4.9 1,6v14.65c0,0.25 0.25,0.5 0.5,0.5 0.1,0 0.15,-0.05 0.25,-0.05C3.1,20.45 5.05,20 6.5,20c1.95,0 4.05,0.4 5.5,1.5 1.35,-0.85 3.8,-1.5 5.5,-1.5 1.65,0 3.35,0.3 4.75,1.05 0.1,0.05 0.15,0.05 0.25,0.05 0.25,0 0.5,-0.25 0.5,-0.5V6c-0.6,-0.45 -1.25,-0.75 -2,-1zM10,17.5l-3,-2v-7l3,2v7zM21,17.5l-3,2v-7l3,-2v7z",
  "app.settings": "M19.14,12.94c0.04,-0.3 0.06,-0.61 0.06,-0.94c0,-0.32 -0.02,-0.64 -0.07,-0.94l2.03,-1.58c0.18,-0.14 0.23,-0.41 0.12,-0.61l-1.92,-3.32c-0.12,-0.22 -0.37,-0.29 -0.59,-0.22l-2.39,0.96c-0.5,-0.38 -1.03,-0.7 -1.62,-0.94L14.4,2.81c-0.04,-0.24 -0.24,-0.41 -0.48,-0.41h-3.84c-0.24,0 -0.43,0.17 -0.47,0.41L9.25,5.35C8.66,5.59 8.12,5.92 7.63,6.29L5.24,5.33c-0.22,-0.08 -0.47,0 -0.59,0.22L2.74,8.87C2.62,9.08 2.66,9.34 2.86,9.48l2.03,1.58C4.84,11.36 4.8,11.69 4.8,12s0.02,0.64 0.07,0.94l-2.03,1.58c-0.18,0.14 -0.23,0.41 -0.12,0.61l1.92,3.32c0.12,0.22 0.37,0.29 0.59,0.22l2.39,-0.96c0.5,0.38 1.03,0.7 1.62,0.94l0.36,2.54c0.05,0.24 0.24,0.41 0.48,0.41h3.84c0.24,0 0.44,-0.17 0.47,-0.41l0.36,-2.54c0.59,-0.24 1.13,-0.56 1.62,-0.94l2.39,0.96c0.22,0.08 0.47,0 0.59,-0.22l1.92,-3.32c0.12,-0.22 0.07,-0.47 -0.12,-0.61L19.14,12.94zM12,15.6c-1.98,0 -3.6,-1.62 -3.6,-3.6s1.62,-3.6 3.6,-3.6s3.6,1.62 3.6,3.6S13.98,15.6 12,15.6z",
  "system.internet": "M12,2a10,10 0,1 0,0 20a10,10 0,0 0,0 -20zM4.1,11a8.1,8.1 0,0 1,1.7,-4.1h2.8A14.7,14.7 0,0 0,8,11H4.1zM4.1,13H8a14.7,14.7 0,0 0,0.6 4.1H5.8A8.1,8.1 0,0 1,4.1 13zM12,19.9a12.7,12.7 0,0 1,-1.4,-4.9h2.8A12.7,12.7 0,0 1,12 19.9zM13.4,13h-2.8a12.7,12.7 0,0 1,0,-2h2.8a12.7,12.7 0,0 1,0 2zM12,4.1a12.7,12.7 0,0 1,1.4,4.9h-2.8A12.7,12.7 0,0 1,12 4.1zM15.4,17.1a14.7,14.7 0,0 0,0.6,-4.1h3.9a8.1,8.1 0,0 1,-1.7 4.1h-2.8zM16,11a14.7,14.7 0,0 0,-0.6,-4.1h2.8a8.1,8.1 0,0 1,1.7 4.1H16z",
  "action.import": "M19,9h-4V3H9v6H5l7,7 7,-7zM5,18v2h14v-2H5z",
  "action.extract-text": "M4.5,9h3c0.6,0 1,0.5 1,1v4c0,0.5 -0.4,1 -1,1h-3c-0.6,0 -1,-0.5 -1,-1v-4c0,-0.5 0.4,-1 1,-1zM5,10.5v3h2v-3zM13.5,9h-3c-0.6,0 -1,0.5 -1,1v4c0,0.5 0.4,1 1,1h3c0.6,0 1,-0.5 1,-1v-0.5h-3.5v-3h3.5V10c0,-0.5 -0.4,-1 -1,-1zM15.5,9v6h1.5v-2h1.1l1.4,2h1.7l-1.5,-2.1c0.5,-0.2 0.8,-0.7 0.8,-1.2V10c0,-0.6 -0.5,-1 -1,-1zM17,10.5h2v1h-2z",
  "action.text-settings": "M9 4v3h5v12h3V7h5V4H9zm-6 8h3v7h3v-7h3V9H3v3z",
  "app.info": "M11 7h2v2h-2zm0 4h2v6h-2zm1-9C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm0 18c-4.41 0-8-3.59-8-8s3.59-8 8-8 8 3.59 8 8-3.59 8-8 8z",
  "action.copy": "M16,1L4,1C2.9,1 2,1.9 2,3L2,17L4,17L4,3L16,3L16,1ZM19,5L8,5C6.9,5 6,5.9 6,7L6,21C6,22.1 6.9,23 8,23L19,23C20.1,23 21,22.1 21,21L21,7C21,5.9 20.1,5 19,5ZM19,21L8,21L8,7L19,7L19,21Z",
  "action.remove": "M7,11v2h10v-2H7zM12,2C6.48,2 2,6.48 2,12s4.48,10 10,10 10,-4.48 10,-10S17.52,2 12,2zM12,20c-4.41,0 -8,-3.59 -8,-8s3.59,-8 8,-8 8,3.59 8,8 -3.59,8 -8,8z",
  "action.download": "M19.35,10.04C18.67,6.59 15.64,4 12,4 9.11,4 6.6,5.64 5.35,8.04 2.34,8.36 0,10.91 0,14c0,3.31 2.69,6 6,6h13c2.76,0 5,-2.24 5,-5 0,-2.64 -2.05,-4.78 -4.65,-4.96zM17,13l-5,5 -5,-5h3V9h4v4h3z",
  "action.retry": "M12 5V1L7 6l5 5V7c3.31 0 6 2.69 6 6s-2.69 6-6 6-6-2.69-6-6H4c0 4.42 3.58 8 8 8s8-3.58 8-8-3.58-8-8-8z",
  "action.cancel": "M12,2C6.47,2 2,6.47 2,12s4.47,10 10,10 10,-4.47 10,-10S17.53,2 12,2zM17,15.59L15.59,17 12,13.41 8.41,17 7,15.59 10.59,12 7,8.41 8.41,7 12,10.59 15.59,7 17,8.41 13.41,12 17,15.59z",
  "app.help": "M11,18h2v-2h-2v2zM12,2C6.48,2 2,6.48 2,12s4.48,10 10,10 10,-4.48 10,-10S17.52,2 12,2zM12,20c-4.41,0 -8,-3.59 -8,-8s3.59,-8 8,-8 8,3.59 8,8 -3.59,8 -8,8zM12,6c-2.21,0 -4,1.79 -4,4h2c0,-1.1 0.9,-2 2,-2s2,0.9 2,2c0,2 -3,1.75 -3,5h2c0,-2.25 3,-2.5 3,-5 0,-2.21 -1.79,-4 -4,-4z",
  "content.document": "M14,2H6C4.9,2 4,2.9 4,4v16c0,1.1 0.9,2 2,2h12c1.1,0 2,-0.9 2,-2V8L14,2zM16,18H8v-2h8v2zM16,14H8v-2h8v2zM13,9V3.5L18.5,9H13z",
};

// Meanings drawn mirrored in a right-to-left layout (the record's `rtl: mirror`, ICON-RENDER
// rule 7). Everything else is fixed.
const MIRRORED = new Set(["nav.open-external"]);

// Drawings whose catalog file carries fill-rule="evenodd": without it the counters of the letters
// of the recognized-text meaning fill in.
const EVENODD = new Set(["action.extract-text"]);

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
  if (EVENODD.has(id)) path.setAttribute("fill-rule", "evenodd");
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

export { GLYPHS, MIRRORED, EVENODD, glyph, applyGlyphs };
