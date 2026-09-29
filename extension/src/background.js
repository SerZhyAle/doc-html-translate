// background.js - MV3 service worker. Owns PDF/EPUB interception and the on/off
// toggle.
//
// Interception uses declarativeNetRequest *dynamic* rules built at runtime from
// chrome.runtime.getURL(), so the extension id is never hardcoded. Two redirect
// rules send main_frame requests for supported document extensions (http/https and
// file://) to the reflow viewer. DNR matches the *request* URL, so documents served without
// a matching extension in the URL are not intercepted - a documented limitation
// (spec sec 4/7).

import { DEFAULT_OPTIONS } from "./defaults.js";
import { HTTPS_INTERCEPT_REGEX, FILE_INTERCEPT_REGEX, isInterceptableUrl } from "./intercept.js";
import { siteHost } from "./site-host.js";
import * as badge from "./badge.js";
// The whole-page OCR broker attaches its own message and tab listeners on import; this file only
// owns the menu entry that starts it. See DEV/plan/done/2026-09-19_page-ocr-overlay.md.
import { startRun as startPageOcr } from "./page-ocr.js";

const RULE_HTTPS = 1;
const RULE_FILE = 2;

// The intercept patterns live in intercept.js, shared with the popup and the command handler;
// re-exported here for the worker's own tests.
export { HTTPS_INTERCEPT_REGEX, FILE_INTERCEPT_REGEX };

// Extensions offered by the "Convert with doc-html-translate" right-click entry. Broader
// than the DNR interception list (pdf/epub/rtf/fb2/mobi/azw3/cbz/cbt) because the viewer
// can also render txt/md/html on demand, and CBR/CB7 are offered here so the viewer can
// give a clear "desktop app only" notice on explicit request (they cannot be decoded in
// the browser, so they are not auto-intercepted). Match patterns ignore query strings, so
// `*.pdf` still matches `file.pdf?download=1`.
const CONVERT_EXTS = ["pdf", "epub", "rtf", "fb2", "mobi", "azw3", "txt", "md", "html", "htm", "cbz", "cbt", "cbr", "cb7"];
const CONVERT_URL_PATTERNS = CONVERT_EXTS.flatMap((e) => [`*://*/*.${e}`, `file:///*.${e}`]);

async function getOptions() {
  const got = await chrome.storage.local.get("options");
  return { ...DEFAULT_OPTIONS, ...(got.options || {}) };
}

function viewerBase() {
  return chrome.runtime.getURL("src/viewer.html");
}

// ruleDomains keeps the disabled-host entries DNR accepts: lower-case ASCII (punycode) domain
// names. The popup stores location.hostname, which for an IPv6 page is "[::1]" and for an IDN is
// already punycode, but a single entry DNR rejects fails the whole atomic update - interception
// then silently keeps the previous rules. An entry that cannot be a DNR domain is dropped instead.
export function ruleDomains(hosts) {
  const out = [];
  for (const h of Array.isArray(hosts) ? hosts : []) {
    let host;
    try { host = new URL(`http://${String(h).trim()}/`).hostname; } catch { continue; }
    if (!host || !/^[a-z0-9-]+(\.[a-z0-9-]+)*$/.test(host)) continue;
    if (!out.includes(host)) out.push(host);
  }
  return out;
}

// Build the dynamic redirect rules from current options. Returns [] when the
// extension is globally off, which removes interception entirely.
function buildRules(options) {
  if (!options.enabledByDefault) return [];
  const sub = `${viewerBase()}?file=\\1`;
  const httpsRule = {
    id: RULE_HTTPS,
    priority: 1,
    action: { type: "redirect", redirect: { regexSubstitution: sub } },
    condition: {
      // Capture the whole URL so the substitution keeps any query string intact.
      regexFilter: HTTPS_INTERCEPT_REGEX,
      resourceTypes: ["main_frame"],
    },
  };
  // A switched-off site means its documents are left alone wherever they are served from: a PDF
  // on the site itself (the request's domain) and one on a CDN or another host that a page of the
  // site links to (the navigation's initiator). Keying on only one of them made the switch a
  // no-op for every cross-host link. See site-host.js for which host the popup stores.
  const excluded = ruleDomains(options.disabledHosts);
  if (excluded.length) {
    httpsRule.condition.excludedRequestDomains = excluded;
    httpsRule.condition.excludedInitiatorDomains = excluded;
  }
  const fileRule = {
    id: RULE_FILE,
    priority: 1,
    action: { type: "redirect", redirect: { regexSubstitution: sub } },
    condition: {
      regexFilter: FILE_INTERCEPT_REGEX,
      resourceTypes: ["main_frame"],
    },
  };
  return [httpsRule, fileRule];
}

// Syncs run one at a time, each reading the options when its turn comes. Two storage changes in
// quick succession used to race, and the one whose read finished last could apply the older
// options. A failure is logged and reported, never left as an unhandled rejection.
let syncTail = Promise.resolve();

function syncRules() {
  const run = syncTail.then(async () => {
    const options = await getOptions();
    await chrome.declarativeNetRequest.updateDynamicRules({
      removeRuleIds: [RULE_HTTPS, RULE_FILE],
      addRules: buildRules(options),
    });
  });
  syncTail = run.catch(() => {});
  return run.then(() => ({ ok: true }), (e) => {
    console.warn("interception rules not updated", e);
    return { ok: false, error: String((e && e.message) || e) };
  });
}

// Right-click actions. removeAll first so re-running onInstalled / onStartup never throws
// on a duplicate id.
//   - "OCR & translate this image": on any image, opens the OCR overlay page for it.
//   - "OCR every image on this page": recognizes the whole page in place, without leaving it.
//     Offered on the image context too, because on the pages this is for - a webcomic, a scanned
//     archive - a right-click almost always lands on a picture, and a page-only entry would be
//     unreachable exactly where the feature belongs. On the link context too, so the root item
//     below always has at least one visible child.
//   - "Convert with doc-html-translate": on a link to a supported document, or on the page
//     when viewing one directly, opens it in the reflow viewer. Always available even when
//     auto-interception is off (the default), so it is how users convert on demand.
//
// All of them hang off one short root item. Chrome prints its own group header - the full
// extension name, long enough to crowd the menu - only when we register more than one top-level
// entry; with a single parent it prints that parent's title instead, so the root can say what the
// actions do rather than what the product is called.
const ROOT_MENU_ID = "sza-root";
const OCR_MENU_ID = "ocr-image";
const OCR_PAGE_MENU_ID = "ocr-page";

// The whole-page run needs a document the extension may be injected into. A chrome:// page, the
// Web Store and a PDF Chrome renders itself are none of those, and an entry that always fails is
// worse than no entry.
const PAGE_OCR_URL_PATTERNS = ["http://*/*", "https://*/*", "file:///*"];
const CONVERT_LINK_MENU_ID = "convert-doc-link";
const CONVERT_PAGE_MENU_ID = "convert-doc-page";

// In a context-menu title "&" marks the next character as the keyboard mnemonic and is eaten,
// so the English item rendered as "OCR _translate this image". Doubling it prints one literal
// ampersand. Applied to every title, not just the one that happens to carry an "&" today.
function menuTitle(key, fallback) {
  const text = chrome.i18n.getMessage(key) || fallback;
  return text.replace(/&/g, "&&");
}

function setupContextMenu() {
  try {
    chrome.contextMenus.removeAll(() => {
      // The root carries the page-OCR patterns so it can never render as an empty submenu: where
      // it is offered at all, the page-OCR child matches in every context the root declares.
      chrome.contextMenus.create({
        id: ROOT_MENU_ID,
        title: menuTitle("rootMenu", "OCR & translate"),
        contexts: ["page", "image", "link"],
        documentUrlPatterns: PAGE_OCR_URL_PATTERNS,
      });
      chrome.contextMenus.create({
        id: OCR_MENU_ID,
        parentId: ROOT_MENU_ID,
        title: menuTitle("ocrImageMenu", "OCR & translate this image"),
        contexts: ["image"],
      });
      chrome.contextMenus.create({
        id: OCR_PAGE_MENU_ID,
        parentId: ROOT_MENU_ID,
        title: menuTitle("ocrPageMenu", "OCR every image on this page"),
        contexts: ["page", "image", "link"],
        documentUrlPatterns: PAGE_OCR_URL_PATTERNS,
      });
      const convertTitle = menuTitle("convertDocMenu", "Convert with doc-html-translate");
      chrome.contextMenus.create({
        id: CONVERT_LINK_MENU_ID,
        parentId: ROOT_MENU_ID,
        title: convertTitle,
        contexts: ["link"],
        targetUrlPatterns: CONVERT_URL_PATTERNS,
      });
      chrome.contextMenus.create({
        id: CONVERT_PAGE_MENU_ID,
        parentId: ROOT_MENU_ID,
        title: convertTitle,
        contexts: ["page"],
        documentUrlPatterns: CONVERT_URL_PATTERNS,
      });
    });
  } catch (e) {
    console.warn("context menu setup failed", e);
  }
}

// Open a document URL in the reflow viewer. Encoding matches viewer.js's manual
// `viewer.html?file=<encoded>` entry point (parseFileParam decodes it).
function openInViewer(url) {
  if (!url) return;
  chrome.tabs.create({ url: `${viewerBase()}?file=${encodeURIComponent(url)}` });
}

chrome.contextMenus.onClicked.addListener((info, tab) => {
  if (info.menuItemId === OCR_PAGE_MENU_ID) {
    if (tab && tab.id != null) startPageOcr(tab.id);
    return;
  }
  if (info.menuItemId === OCR_MENU_ID) {
    if (!info.srcUrl) return;
    chrome.tabs.create({ url: `${chrome.runtime.getURL("src/ocr.html")}?src=${encodeURIComponent(info.srcUrl)}` });
    return;
  }
  if (info.menuItemId === CONVERT_LINK_MENU_ID) openInViewer(info.linkUrl);
  else if (info.menuItemId === CONVERT_PAGE_MENU_ID) openInViewer(info.pageUrl);
});

chrome.runtime.onInstalled.addListener(() => { syncRules(); setupContextMenu(); });
chrome.runtime.onStartup.addListener(() => { syncRules(); setupContextMenu(); });

// Rebuild rules whenever the options change (popup/options page write storage).
chrome.storage.onChanged.addListener((changes, area) => {
  if (area === "local" && changes.options) syncRules();
});

// Per-tab bypass rule id allocator. Tab ids grow unbounded over a session, so a
// `1000 + tabId % N` scheme would eventually collide and let one tab's bypass
// clobber another's. Instead we hand out monotonically increasing ids from a
// reserved band and remember the exact id used for each tab.
const BYPASS_ID_BASE = 100000;
let bypassIdSeq = BYPASS_ID_BASE;
const tabBypassRuleId = new Map();

// "Open original PDF": temporarily allow the original request for just this tab
// (an allow rule out-prioritizes the redirect), then navigate the tab to it.
async function openOriginal(url, tabId) {
  if (tabId == null || !url || !/^(https?|file):/i.test(url)) return;
  // Reuse this tab's existing bypass id if it has one, else allocate a fresh one.
  let allowId = tabBypassRuleId.get(tabId);
  if (allowId == null) {
    allowId = ++bypassIdSeq;
    tabBypassRuleId.set(tabId, allowId);
  }
  try {
    await chrome.declarativeNetRequest.updateSessionRules({
      removeRuleIds: [allowId],
      addRules: [{
        id: allowId,
        priority: 100,
        action: { type: "allow" },
        condition: { resourceTypes: ["main_frame"], tabIds: [tabId] },
      }],
    });
    await chrome.tabs.update(tabId, { url });
  } catch (e) {
    console.warn("openOriginal failed", e);
  }
  // Drop the bypass once the navigation has had time to commit, so future PDF
  // opens in this tab are intercepted again. Key cleanup on the exact id added.
  setTimeout(() => {
    chrome.declarativeNetRequest.updateSessionRules({ removeRuleIds: [allowId] }).catch(() => {});
    if (tabBypassRuleId.get(tabId) === allowId) tabBypassRuleId.delete(tabId);
  }, 5000);
}

// The tab a keyboard command acts on: the command fires with no event argument, and the tab
// that matters is the one in the window the user is looking at.
async function activeTab() {
  try {
    const [tab] = await chrome.tabs.query({ active: true, lastFocusedWindow: true });
    return tab || null;
  } catch { return null; }
}

// Keyboard commands (manifest "commands", rebindable in the browser's settings).
//   - open-viewer: what the popup's primary button does - a document tab opens in the reflow
//     viewer, anything else opens the viewer's empty state with its file picker. Never a
//     silent conversion: the same viewer flow, only faster to reach.
//   - toggle-interception: the popup's "On this site" switch without the popup. A no-op where
//     there is no site (browser pages) and while interception is globally off (there is
//     nothing to toggle per site; the popup disables the switch there too).
chrome.commands.onCommand.addListener(async (command) => {
  const tab = await activeTab();
  if (!tab) return;
  if (command === "open-viewer") {
    const url = isInterceptableUrl(tab.url) ? tab.url : "";
    if (url) openInViewer(url);
    else chrome.tabs.create({ url: viewerBase() });
    return;
  }
  if (command === "toggle-interception") {
    const host = siteHost(tab.url || "", viewerBase());
    if (!host) return;
    const options = await getOptions();
    if (!options.enabledByDefault) return;
    const set = new Set(options.disabledHosts || []);
    if (set.has(host)) set.delete(host);
    else set.add(host);
    options.disabledHosts = [...set];
    await chrome.storage.local.set({ options });
    // The rules rebuild from the storage change; the badge has no listener of its own.
    badge.tabBase(tab.id, tab.url, options);
  }
});

// The passive half of the badge: a tab whose site documents are left alone wears a grey
// "off". Recomputed from the tab's own URL on navigation, so it never needs document content.
async function refreshTabBase(tabId, url) {
  try {
    badge.tabBase(tabId, url, await getOptions());
  } catch { /* options unavailable - the badge stays as it is */ }
}

chrome.tabs.onUpdated.addListener((tabId, info) => {
  // A navigation drops the previous page's job, error and flash state with it.
  if (info.status === "loading") {
    badge.tabNavigated(tabId);
    chrome.tabs.get(tabId).then((tab) => refreshTabBase(tabId, tab && tab.url)).catch(() => {});
  } else if (typeof info.url === "string") {
    refreshTabBase(tabId, info.url);
  }
});

chrome.tabs.onRemoved.addListener((tabId) => badge.tabGone(tabId));

chrome.runtime.onMessage.addListener((msg, sender, sendResponse) => {
  if (!msg || typeof msg !== "object") return;
  // The viewer's job reports. The tab is the sender: a page cannot move another tab's badge.
  if (msg.dht === "badge" && sender.tab && sender.tab.id != null) {
    const tabId = sender.tab.id;
    if (msg.t === "job" && typeof msg.kind === "string") {
      badge.tabJob(tabId, msg.kind, msg.phase, Number(msg.done) || 0, Number(msg.total) || 0);
    } else if (msg.t === "error") badge.tabError(tabId);
    else if (msg.t === "reset") badge.tabReset(tabId);
    return;
  }
  if (msg.type === "open-original") {
    const tabId = msg.tabId != null ? msg.tabId : sender.tab && sender.tab.id;
    openOriginal(msg.url, tabId).then(() => sendResponse({ ok: true }));
    return true; // async response
  }
  if (msg.type === "sync-rules") {
    syncRules().then(sendResponse);
    return true;
  }
});
