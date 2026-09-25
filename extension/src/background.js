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
// The whole-page OCR broker attaches its own message and tab listeners on import; this file only
// owns the menu entry that starts it. See DEV/plan/29_2026-09-19_page-ocr-overlay.md.
import { startRun as startPageOcr } from "./page-ocr.js";

const RULE_HTTPS = 1;
const RULE_FILE = 2;

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
      regexFilter: "^(https?://.*\\.(?:pdf|epub|rtf|fb2|mobi|azw3|cbz|cbt)(?:[?#].*)?)$",
      resourceTypes: ["main_frame"],
    },
  };
  if (options.disabledHosts && options.disabledHosts.length) {
    httpsRule.condition.excludedRequestDomains = options.disabledHosts;
  }
  const fileRule = {
    id: RULE_FILE,
    priority: 1,
    action: { type: "redirect", redirect: { regexSubstitution: sub } },
    condition: {
      // Three slashes: only empty-host file URLs. UNC paths (file://server/share)
      // can't be granted to extensions by any match pattern, so the viewer could
      // never fetch them - leave those to Chrome's built-in viewer.
      regexFilter: "^(file:///.*\\.(?:pdf|epub|rtf|fb2|mobi|azw3|cbz|cbt)(?:[?#].*)?)$",
      resourceTypes: ["main_frame"],
    },
  };
  return [httpsRule, fileRule];
}

async function syncRules() {
  const options = await getOptions();
  const addRules = buildRules(options);
  await chrome.declarativeNetRequest.updateDynamicRules({
    removeRuleIds: [RULE_HTTPS, RULE_FILE],
    addRules,
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

chrome.runtime.onMessage.addListener((msg, sender, sendResponse) => {
  if (!msg || typeof msg !== "object") return;
  if (msg.type === "open-original") {
    const tabId = msg.tabId != null ? msg.tabId : sender.tab && sender.tab.id;
    openOriginal(msg.url, tabId).then(() => sendResponse({ ok: true }));
    return true; // async response
  }
  if (msg.type === "sync-rules") {
    syncRules().then(() => sendResponse({ ok: true }));
    return true;
  }
});
