// Behaviour tests for background.js - the MV3 service worker. It has no exports: everything it
// does hangs off chrome.* event listeners registered on import. So the fake `chrome` below records
// every listener and every API call, and each test fires the event a browser would fire and checks
// what the worker asked the browser to do in return.

import { test } from "node:test";
import assert from "node:assert/strict";

const VIEWER = "chrome-extension://test/src/viewer.html";
const on = {};                   // event name -> listeners
const calls = {
  dynamicRules: [], sessionRules: [], menus: [], tabsCreated: [], tabsUpdated: [], insertCSS: [],
};
let storedOptions;               // what chrome.storage.local holds under "options"
let failSessionRules = false;

const event = (name) => ({ addListener: (fn) => { (on[name] ||= []).push(fn); } });
const fire = (name, ...args) => (on[name] || []).map((fn) => fn(...args));

globalThis.chrome = {
  runtime: {
    getURL: (p) => `chrome-extension://test/${p}`,
    sendMessage: async () => {},
    onInstalled: event("installed"),
    onStartup: event("startup"),
    onMessage: event("message"),
  },
  storage: {
    local: { get: async () => (storedOptions ? { options: storedOptions } : {}) },
    onChanged: event("storageChanged"),
  },
  declarativeNetRequest: {
    updateDynamicRules: async (arg) => { calls.dynamicRules.push(arg); },
    updateSessionRules: async (arg) => {
      if (failSessionRules) throw new Error("quota");
      calls.sessionRules.push(arg);
    },
  },
  contextMenus: {
    removeAll: (cb) => { calls.menus.length = 0; cb(); },
    create: (item) => { calls.menus.push(item); },
    onClicked: event("menuClicked"),
  },
  tabs: {
    create: (arg) => { calls.tabsCreated.push(arg); },
    update: async (tabId, arg) => { calls.tabsUpdated.push({ tabId, ...arg }); },
    sendMessage: async () => null,
    onRemoved: event("tabRemoved"),
    onUpdated: event("tabUpdated"),
  },
  scripting: {
    insertCSS: async (arg) => { calls.insertCSS.push(arg); },
    executeScript: async () => {},
  },
  i18n: { getMessage: () => "" },
};

await import("../src/background.js");

// The rule sync is fire-and-forget from the listeners, so let its awaits drain before asserting.
const settle = () => new Promise((r) => setImmediate(r));

// sendMessage runs the worker's message listener the way the runtime does and resolves with the
// async response, if the listener promised one.
function sendMessage(msg, sender = {}) {
  return new Promise((resolve) => {
    const kept = fire("message", msg, sender, resolve);
    if (!kept.includes(true)) resolve(undefined);
  });
}

test("install with default options clears interception and builds the context menu", async () => {
  storedOptions = undefined;
  fire("installed");
  await settle();

  // Interception is opt-in, so the defaults must remove both redirect rules and add none.
  assert.deepEqual(calls.dynamicRules.at(-1), { removeRuleIds: [1, 2], addRules: [] });

  // One root and every action hangs off it - two top-level entries would make Chrome print its
  // own long group header.
  const roots = calls.menus.filter((m) => !m.parentId);
  assert.deepEqual(roots.map((m) => m.id), ["sza-root"]);
  assert.deepEqual(calls.menus.filter((m) => m.parentId === "sza-root").map((m) => m.id).sort(),
    ["convert-doc-link", "convert-doc-page", "ocr-image", "ocr-page"]);
  // A lone "&" is a mnemonic marker in a menu title; the worker has to double it to print one.
  assert.equal(calls.menus.find((m) => m.id === "ocr-image").title, "OCR && translate this image");
});

test("enabling interception installs redirect rules that honour disabled hosts", async () => {
  storedOptions = { enabledByDefault: true, disabledHosts: ["intranet.example"] };
  fire("storageChanged", { options: { newValue: storedOptions } }, "local");
  await settle();

  const { addRules } = calls.dynamicRules.at(-1);
  assert.equal(addRules.length, 2);
  const [https, file] = addRules;
  assert.deepEqual(https.condition.excludedRequestDomains, ["intranet.example"]);
  assert.equal(https.action.redirect.regexSubstitution, `${VIEWER}?file=\\1`);

  // DNR evaluates these as RE2; the subset used here reads the same in JS, so the match behaviour
  // the comments promise can be checked directly.
  const httpsRe = new RegExp(https.condition.regexFilter);
  assert.equal(httpsRe.exec("https://x.test/a/book.pdf?dl=1")[1], "https://x.test/a/book.pdf?dl=1");
  assert.ok(httpsRe.test("http://x.test/book.epub"));
  assert.ok(!httpsRe.test("https://x.test/page.html"), "html is converted on demand, never intercepted");
  const fileRe = new RegExp(file.condition.regexFilter);
  assert.ok(fileRe.test("file:///C:/books/a.fb2"));
  assert.ok(!fileRe.test("file://server/share/a.pdf"), "UNC paths are left to the browser");
});

test("interception matches the document extension in the URL path only", async () => {
  const { HTTPS_INTERCEPT_REGEX, FILE_INTERCEPT_REGEX } = await import("../src/background.js");
  const https = new RegExp(HTTPS_INTERCEPT_REGEX);
  const file = new RegExp(FILE_INTERCEPT_REGEX);
  const table = [
    // [url, intercepted]
    ["https://site/a.pdf", true],
    ["https://site/dir/book.epub?token=1&x=2", true],
    ["https://site/a.cbz#p=3", true],
    ["https://site/viewer?file=a.pdf", false],
    ["https://site/viewer?next=/x/book.epub#top", false],
    ["https://site/page#a.pdf", false],
    ["https://site/a.pdf.html", false],
    ["https://site/a.pdfx", false],
    ["https://site/pdf", false],
  ];
  for (const [url, want] of table) assert.equal(https.test(url), want, url);
  assert.equal(file.test("file:///C:/b/a.azw3?x"), true);
  assert.equal(file.test("file:///C:/b/viewer.html?f=a.pdf"), false);
});

test("a storage change outside the options leaves the rules alone", async () => {
  const before = calls.dynamicRules.length;
  fire("storageChanged", { uiLang: { newValue: "de" } }, "local");
  fire("storageChanged", { options: { newValue: {} } }, "sync");
  await settle();
  assert.equal(calls.dynamicRules.length, before);
});

test("context menu clicks open the viewer, the OCR page or the page run", async () => {
  fire("menuClicked", { menuItemId: "convert-doc-link", linkUrl: "https://x.test/a b.pdf?x=1&y=2" }, { id: 3 });
  assert.equal(calls.tabsCreated.at(-1).url,
    `${VIEWER}?file=${encodeURIComponent("https://x.test/a b.pdf?x=1&y=2")}`);

  fire("menuClicked", { menuItemId: "ocr-image", srcUrl: "https://x.test/p.png" }, { id: 3 });
  assert.equal(calls.tabsCreated.at(-1).url,
    `chrome-extension://test/src/ocr.html?src=${encodeURIComponent("https://x.test/p.png")}`);

  fire("menuClicked", { menuItemId: "ocr-page" }, { id: 42 });
  await settle();
  assert.equal(calls.insertCSS.at(-1).target.tabId, 42, "the page run injects into the clicked tab");
});

test("context menu clicks without a target open nothing", () => {
  const before = calls.tabsCreated.length;
  fire("menuClicked", { menuItemId: "ocr-image" }, { id: 3 });
  fire("menuClicked", { menuItemId: "convert-doc-link" }, { id: 3 });
  fire("menuClicked", { menuItemId: "ocr-page" }, undefined);
  assert.equal(calls.tabsCreated.length, before);
});

test("open-original adds a tab-scoped allow rule, navigates, then drops the rule", async (t) => {
  t.mock.timers.enable({ apis: ["setTimeout"] });
  const res = await sendMessage({ type: "open-original", url: "https://x.test/a.pdf" }, { tab: { id: 7 } });
  assert.deepEqual(res, { ok: true });

  const added = calls.sessionRules.at(-1);
  const rule = added.addRules[0];
  assert.deepEqual(added.removeRuleIds, [rule.id]);
  assert.deepEqual(rule.action, { type: "allow" });
  assert.deepEqual(rule.condition.tabIds, [7]);
  assert.ok(rule.priority > 1, "the allow rule must outrank the redirect rules");
  assert.deepEqual(calls.tabsUpdated.at(-1), { tabId: 7, url: "https://x.test/a.pdf" });

  t.mock.timers.tick(5000);
  assert.deepEqual(calls.sessionRules.at(-1), { removeRuleIds: [rule.id] });
});

test("open-original refuses a non-document URL and survives a rule failure", async (t) => {
  t.mock.timers.enable({ apis: ["setTimeout"] });
  const rulesBefore = calls.sessionRules.length;
  const navBefore = calls.tabsUpdated.length;

  // The viewer is web-accessible, so the URL is opener-controlled and must never be navigated to
  // unless it is a document scheme.
  assert.deepEqual(await sendMessage({ type: "open-original", url: "javascript:alert(1)", tabId: 7 }), { ok: true });
  assert.equal(calls.sessionRules.length, rulesBefore);
  assert.equal(calls.tabsUpdated.length, navBefore);

  failSessionRules = true;
  const warn = console.warn;
  console.warn = () => {};
  try {
    assert.deepEqual(await sendMessage({ type: "open-original", url: "https://x.test/a.pdf", tabId: 8 }), { ok: true });
  } finally {
    console.warn = warn;
    failSessionRules = false;
  }
  assert.equal(calls.tabsUpdated.length, navBefore, "no navigation when the bypass could not be set");
  t.mock.timers.tick(5000);
});

test("sync-rules answers after resyncing, and unrelated messages get no response", async () => {
  const before = calls.dynamicRules.length;
  assert.deepEqual(await sendMessage({ type: "sync-rules" }), { ok: true });
  assert.equal(calls.dynamicRules.length, before + 1);

  assert.equal(await sendMessage({ type: "nope" }), undefined);
  assert.equal(await sendMessage(null), undefined);
});
