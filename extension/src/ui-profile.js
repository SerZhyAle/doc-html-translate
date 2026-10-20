// ui-profile.js - applies the saved interface settings to a page: the interface scale, the text-only
// enlargement, the input profile and the chrome theme (UI-SIZING 1.1, SETTINGS-NAVIGATION 3 rule 1).
// ui-sizing.css does the rest from the attributes this sets on the root.
//
// The input profile is chosen when a page opens and only by the setting: "auto" keeps the answer
// ui-boot.js read once from the primary pointer, "pointer" and "touch" are an explicit choice. Nothing
// here listens to a pointer, touch or resize event.

import { UI_SIZES, TEXT_SIZES, INPUT_PROFILES, CHROME_THEMES, pick } from "./ui-settings.js";
import { DEFAULT_OPTIONS } from "./defaults.js";

// autoInput is the surface the page opened on. ui-boot.js records it before the first paint; a page
// without that script (a test document) asks the media queries once and keeps the answer.
function autoInput(root) {
  if (!root.dataset.inputAuto) {
    let touch = false;
    try {
      touch = globalThis.matchMedia("(pointer: coarse)").matches || globalThis.matchMedia("(any-pointer: none)").matches;
    } catch { /* no media queries: a pointer surface */ }
    root.dataset.inputAuto = touch ? "touch" : "pointer";
  }
  return root.dataset.inputAuto;
}

// applyUiProfile sets data-size, data-text, data-input and data-chrome-theme from the stored options.
// An unknown or missing value takes its default, so a corrupt storage still draws a usable page.
export function applyUiProfile(options, root = document.documentElement) {
  const o = options || {};
  root.dataset.size = pick(o.uiSize, UI_SIZES, DEFAULT_OPTIONS.uiSize);
  root.dataset.text = pick(o.textSize, TEXT_SIZES, DEFAULT_OPTIONS.textSize);
  const profile = pick(o.inputProfile, INPUT_PROFILES, DEFAULT_OPTIONS.inputProfile);
  root.dataset.input = profile === "auto" ? autoInput(root) : profile;
  const theme = pick(o.chromeTheme, CHROME_THEMES, DEFAULT_OPTIONS.chromeTheme);
  if (theme === "system") delete root.dataset.chromeTheme;
  else root.dataset.chromeTheme = theme;
}

// revealPage shows the page once its language, theme, size and state are in place: one coherent first
// frame, with no English or default-size flash before the reader's own.
export function revealPage(root = document.documentElement) {
  root.removeAttribute("data-boot");
}
