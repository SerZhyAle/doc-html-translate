// ui-settings.js - the vocabulary of the extension's own interface settings (UI-SIZING 1.1,
// SETTINGS-NAVIGATION 2.0). Constants only, no DOM: defaults.js, settings-transfer.js and the
// pages all read one list, so a stored value that is not in it falls back instead of producing a
// zero-size or unreachable interface.
//
// The values and their meaning are the desktop launcher's (cmd/doc-html-ui/ui.html, the
// "interface size and controls profile" block); docs/PARITY.md "Extension interface (ticket 119,
// slice X1)" records what is shared and what differs.

// Interface scale: compact 0.875, ordinary 1, large 1.25, extra large 1.5, huge 2 (ui-sizing.css).
export const UI_SIZES = ["compact", "ordinary", "large", "xlarge", "huge"];
// Text-only enlargement: it grows text and layout first and the glyphs only moderately.
export const TEXT_SIZES = ["ordinary", "large", "xlarge", "huge"];
// "auto" reads the primary pointer once when a page opens; the other two are an explicit choice.
export const INPUT_PROFILES = ["auto", "pointer", "touch"];
// "system" follows the browser's light or dark setting live.
export const CHROME_THEMES = ["system", "light", "dark"];

// The options page's destinations, in rail order. Ids are stable: a remembered page is looked up
// by id, and a retired or unknown one falls back to the first.
export const OPTIONS_PAGES = ["general", "ocr", "sites", "data", "interface", "about"];

// Collapsible groups that remember their state, by stable id (COLLAPSIBLE-GROUP 4 rule 7).
export const GROUP_IDS = ["popupOcrLangs", "aboutSupport", "aboutResources", "aboutRelated"];

// What the interface remembers between openings. Not an option: writing it must not rebuild the
// interception rules or refresh every tab's badge, which any write of `options` does.
export const DEFAULT_UI_STATE = { page: OPTIONS_PAGES[0], groups: {} };

// pick returns value when it is one of allowed, else fallback.
export function pick(value, allowed, fallback) {
  return allowed.includes(value) ? value : fallback;
}

// normalizeUiState turns whatever storage holds into a valid state: an unknown page becomes the
// first, group ids outside the list are dropped, non-boolean states are dropped. Page viewports
// (`scroll`) are kept as {id, off} pairs so a stored anchor survives a restart.
export function normalizeUiState(raw) {
  const state = { page: DEFAULT_UI_STATE.page, groups: {} };
  if (!raw || typeof raw !== "object") return state;
  state.page = pick(raw.page, OPTIONS_PAGES, DEFAULT_UI_STATE.page);
  if (raw.groups && typeof raw.groups === "object" && !Array.isArray(raw.groups)) {
    for (const id of GROUP_IDS) if (typeof raw.groups[id] === "boolean") state.groups[id] = raw.groups[id];
  }
  if (raw.scroll && typeof raw.scroll === "object" && !Array.isArray(raw.scroll)) {
    state.scroll = {};
    for (const id of OPTIONS_PAGES) {
      const a = raw.scroll[id];
      if (a && typeof a.id === "string" && a.id.length <= 80 && Number.isFinite(a.off)) {
        state.scroll[id] = { id: a.id, off: Math.round(a.off) };
      }
    }
  }
  return state;
}
