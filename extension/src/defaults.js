// defaults.js - the single source of truth for the extension's stored `options`
// object. Imported by background.js, popup.js, options.js and viewer.js so the four
// can never drift. They used to each hardcode their own copy and background.js
// silently lacked the OCR fields (ocrImages/ocrLang). See ../../docs/PARITY.md
// ("Settings defaults"); keep these aligned with the desktop app's flag defaults.
export const DEFAULT_OPTIONS = {
  // Off by default: the extension no longer auto-intercepts PDF/EPUB/.. links. Users
  // opt in via the popup toggle, or convert on demand through the "Convert with
  // doc-html-translate" right-click menu (background.js). Mirrors the desktop app, where
  // the default-handler association is likewise an explicit opt-in. See docs/PARITY.md.
  enabledByDefault: false,
  disabledHosts: [],
  sourceLang: "auto",
  theme: "light",
  ocrImages: false,
  ocrLang: "eng",
};
