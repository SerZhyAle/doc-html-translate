// options-nav.js - the options page's rail and pages (SETTINGS-NAVIGATION 2.0).
//
// One list of pages, each with a stable id; choosing one swaps what the pane shows and nothing else - it
// never resizes the window, moves the rail, commits an edit or runs an operation, and it leaves the focus
// on the rail. Every page is already in the document, so a switch is one task that toggles `hidden`: the
// page is complete before it is painted, there is no partial layout to see, and a late answer from an
// asynchronous source only ever updates its own elements, never the destination.
//
// The page and each page's viewport are remembered by stable ids (section 4 rule 3). A narrow window keeps
// the same menu as a start-side drawer opened from the header's Pages button (section 2 rule 6): choosing a
// page, the scrim or Escape closes it, and focus returns to the button.

import { OPTIONS_PAGES } from "./ui-settings.js";
import { getUiState, setUiState } from "./options-store.js";

const el = (id) => document.getElementById(id);
const SAVE_DELAY_MS = 250;

let current = OPTIONS_PAGES[0];
const anchors = {};
let saveTimer = 0;
let saveScroll = false;
let restoring = true;
let onShow = () => {};

// A page's viewport is remembered as a stable anchor - the last element with an id whose top stands above
// the viewport - plus the pixels past it, so a language, size or layout change that moves the pixel geometry
// still lands the reader where they left off. A missing anchor restores nothing: the top is valid.
function pageAnchor(id) {
  const pane = el("pane");
  const page = el(`page-${id}`);
  if (!pane || !page || page.hidden || pane.scrollTop < 4) return null;
  const top = pane.getBoundingClientRect().top;
  let anchor = null;
  for (const kid of page.querySelectorAll("[id]")) {
    if (kid.offsetParent === null) continue;
    const r = kid.getBoundingClientRect();
    if (r.top <= top + 8) anchor = { id: kid.id, off: Math.round(top - r.top) };
    else break;
  }
  return anchor;
}

function restorePageScroll(id) {
  const pane = el("pane");
  const a = anchors[id];
  pane.scrollTop = 0;
  const kid = a ? document.getElementById(a.id) : null;
  if (!kid || kid.offsetParent === null) return;
  pane.scrollTop = Math.max(0, pane.scrollTop + (kid.getBoundingClientRect().top - pane.getBoundingClientRect().top) + a.off);
}

// The memory is written a moment after the last click, so a quick run through the rail is one write.
function remember(scroll) {
  saveScroll = saveScroll || scroll;
  clearTimeout(saveTimer);
  saveTimer = setTimeout(() => {
    const patch = { page: current };
    // Every page's anchor, not only the visible one's: leaving a page updates its anchor in memory a moment
    // before this write, and a page the reader left a second ago must still reopen where it was.
    if (saveScroll) patch.scroll = Object.fromEntries(OPTIONS_PAGES.map((p) => [p, anchors[p] || null]));
    saveScroll = false;
    setUiState(patch).catch(() => { /* memory only: navigation still works */ });
  }, SAVE_DELAY_MS);
}

export function currentPage() { return current; }

export function showPage(id, opts = {}) {
  if (!OPTIONS_PAGES.includes(id)) id = OPTIONS_PAGES[0];
  if (id === current && !el(`page-${id}`).hidden && !opts.force) {
    if (opts.focus) el(`tab-${id}`).focus();
    return;
  }
  const leaving = current;
  // While restoring, the visible page is only the markup's default: there is no viewport of the reader's to keep.
  if (!restoring && !el(`page-${leaving}`).hidden) anchors[leaving] = pageAnchor(leaving);
  for (const p of OPTIONS_PAGES) {
    const on = p === id;
    el(`page-${p}`).hidden = !on;
    const tab = el(`tab-${p}`);
    tab.setAttribute("aria-selected", on ? "true" : "false");
    tab.tabIndex = on ? 0 : -1;
  }
  current = id;
  restorePageScroll(id);
  closeDrawer(false);
  if (opts.focus) el(`tab-${id}`).focus();
  onShow(id);
  if (!restoring) remember(leaving !== id);
}

function drawerMode() { return getComputedStyle(el("btnPages")).display !== "none"; }

function openDrawer() {
  if (!drawerMode()) return;
  el("rail").classList.add("open");
  el("btnPages").setAttribute("aria-expanded", "true");
  el(`tab-${current}`).focus();
}

function closeDrawer(refocus) {
  if (!el("rail").classList.contains("open")) return;
  el("rail").classList.remove("open");
  el("btnPages").setAttribute("aria-expanded", "false");
  if (refocus !== false) el("btnPages").focus();
}

// initNav wires the rail and restores the remembered page. `show` is called with the page id after every
// switch (the About page uses it to stay out of the way until it is first opened).
export async function initNav(show) {
  onShow = show || onShow;
  const state = await getUiState();
  state.scroll && Object.assign(anchors, state.scroll);
  el("railList").addEventListener("click", (ev) => {
    const b = ev.target.closest("[data-page]");
    if (b) showPage(b.dataset.page);
  });
  // A vertical tab list: Up and Down move and select, Home and End jump; Tab leaves the rail for the page
  // (one tab stop).
  el("railList").addEventListener("keydown", (ev) => {
    const tabs = OPTIONS_PAGES.map((p) => el(`tab-${p}`));
    const i = tabs.indexOf(document.activeElement);
    if (i < 0) return;
    let j = -1;
    if (ev.key === "ArrowDown") j = (i + 1) % tabs.length;
    else if (ev.key === "ArrowUp") j = (i - 1 + tabs.length) % tabs.length;
    else if (ev.key === "Home") j = 0;
    else if (ev.key === "End") j = tabs.length - 1;
    if (j < 0) return;
    ev.preventDefault();
    showPage(tabs[j].dataset.page, { focus: true });
  });
  el("btnPages").addEventListener("click", () => (el("rail").classList.contains("open") ? closeDrawer() : openDrawer()));
  el("railScrim").addEventListener("click", () => closeDrawer());
  document.addEventListener("keydown", (ev) => {
    if (ev.key === "Escape" && el("rail").classList.contains("open")) closeDrawer();
  });
  let scrollTimer = 0;
  el("pane").addEventListener("scroll", () => {
    if (restoring) return;
    clearTimeout(scrollTimer);
    scrollTimer = setTimeout(() => {
      anchors[current] = pageAnchor(current);
      remember(true);
    }, 400);
  });
  showPage(state.page, { force: true });
  restoring = false;
}
