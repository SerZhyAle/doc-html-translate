// ui-help.js - option-local help (BOOLEAN-OPTION 2.0 section 4).
//
// The app.help button follows the option's name. Hover or keyboard focus shows a short non-modal summary
// that stays open while the pointer moves into it; a click, tap or Enter opens the full explanation, which
// closes with its Close button, Escape or a click outside. Both texts come from ONE message - the summary,
// a blank line, then the rest - so they cannot drift apart, and neither ever toggles the option (the button
// sits outside the option's label and its events stop here).
//
// Markup: <span class="opt-help-wrap" data-opt-help="<message key>" data-opt-name="<caption key>">
//           <button type="button" class="opt-help">..glyph..</button><div class="opt-pop" role="note" hidden></div></span>

import { t } from "./i18n.js";
import { glyph } from "./glyphs.js";

const helps = [];

function helpText(key) {
  const raw = t(key, "");
  const i = raw.indexOf("\n\n");
  return i < 0 ? { short: raw, full: raw } : { short: raw.slice(0, i), full: raw.slice(0, i) + "\n" + raw.slice(i + 2) };
}

function closeButton(onClose) {
  const row = document.createElement("div");
  row.className = "btn-row";
  const close = document.createElement("button");
  close.type = "button";
  const caption = document.createElement("span");
  caption.textContent = t("dlgClose", "Close");
  close.append(glyph("nav.close"), caption);
  close.addEventListener("click", onClose);
  row.append(close);
  return row;
}

function bindHelp(wrap) {
  const btn = wrap.querySelector(".opt-help");
  const pop = wrap.querySelector(".opt-pop");
  const h = { wrap, btn, pop, mode: null, timer: 0 };
  pop.id = `optpop-${wrap.dataset.optHelp}`;
  btn.setAttribute("aria-controls", pop.id);
  h.show = (mode) => {
    h.mode = mode;
    const text = helpText(wrap.dataset.optHelp);
    pop.textContent = mode === "full" ? text.full : text.short;
    if (mode === "full") pop.append(closeButton(() => { h.hide(); btn.focus(); }));
    pop.hidden = false;
    btn.setAttribute("aria-expanded", mode === "full" ? "true" : "false");
  };
  h.hide = () => {
    h.mode = null;
    pop.hidden = true;
    btn.setAttribute("aria-expanded", "false");
  };
  // The accessible name names its option ("Help: Reflow documents automatically"), and a language
  // change rewrites it along with an open bubble's text.
  h.label = () => {
    const name = t(wrap.dataset.optName, "");
    const label = t("helpOptLabel", "Help: {1}", name);
    btn.setAttribute("aria-label", label);
    btn.title = label;
    if (h.mode) h.show(h.mode);
  };
  // Hover intent: leaving the button towards the bubble crosses a few pixels outside both, so closing waits
  // a moment, and entering the bubble (a descendant of the wrapper) cancels it - the summary stays open
  // while it is read.
  wrap.addEventListener("mouseenter", () => { clearTimeout(h.timer); if (!h.mode) h.show("short"); });
  wrap.addEventListener("mouseleave", () => {
    if (h.mode !== "short") return;
    clearTimeout(h.timer);
    h.timer = setTimeout(() => {
      if (h.mode === "short" && !wrap.matches(":hover") && document.activeElement !== btn) h.hide();
    }, 300);
  });
  btn.addEventListener("focus", () => { if (!h.mode) h.show("short"); });
  btn.addEventListener("blur", () => { if (h.mode === "short" && !wrap.matches(":hover")) h.hide(); });
  btn.addEventListener("click", (ev) => {
    ev.preventDefault();
    ev.stopPropagation();
    if (h.mode === "full") h.hide(); else h.show("full");
  });
  helps.push(h);
  h.label();
}

let documentBound = false;

// initOptionHelp wires every help wrapper under root; call it again after a language change through
// refreshOptionHelp. The document-level listeners are bound once.
export function initOptionHelp(root = document) {
  root.querySelectorAll(".opt-help-wrap").forEach((wrap) => {
    if (!helps.some((h) => h.wrap === wrap)) bindHelp(wrap);
  });
  if (documentBound) return;
  documentBound = true;
  document.addEventListener("pointerdown", (ev) => {
    for (const h of helps) if (h.mode === "full" && !h.wrap.contains(ev.target)) h.hide();
  });
  // Escape dismisses a help bubble first, without moving the pointer, and keeps the same keypress from
  // also closing whatever else listens for it (the narrow page menu).
  document.addEventListener("keydown", (ev) => {
    if (ev.key !== "Escape") return;
    const open = helps.find((h) => h.mode);
    if (!open) return;
    ev.stopImmediatePropagation();
    const was = open.mode;
    open.hide();
    if (was === "full") open.btn.focus();
  });
}

export function refreshOptionHelp() {
  for (const h of helps) h.label();
}
