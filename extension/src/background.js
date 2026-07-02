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

const RULE_HTTPS = 1;
const RULE_FILE = 2;

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
      regexFilter: "^(https?://.*\\.(?:pdf|epub|rtf|fb2|mobi|azw3)(?:[?#].*)?)$",
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
      regexFilter: "^(file:///.*\\.(?:pdf|epub|rtf|fb2|mobi|azw3)(?:[?#].*)?)$",
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

// "OCR & translate this image": a right-click action on any image that opens the OCR
// overlay page in a new tab for that image. removeAll first so re-running onInstalled /
// onStartup never throws on a duplicate id.
const OCR_MENU_ID = "ocr-image";
function setupContextMenu() {
  try {
    chrome.contextMenus.removeAll(() => {
      chrome.contextMenus.create({
        id: OCR_MENU_ID,
        title: chrome.i18n.getMessage("ocrImageMenu") || "OCR & translate this image",
        contexts: ["image"],
      });
    });
  } catch (e) {
    console.warn("context menu setup failed", e);
  }
}

chrome.contextMenus.onClicked.addListener((info) => {
  if (info.menuItemId !== OCR_MENU_ID || !info.srcUrl) return;
  const url = `${chrome.runtime.getURL("src/ocr.html")}?src=${encodeURIComponent(info.srcUrl)}`;
  chrome.tabs.create({ url });
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
