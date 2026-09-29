// Behaviour tests for badge.js - the toolbar icon's per-tab state machine. The module keeps its
// tab states in module-level maps, so each test imports a fresh copy (`?case=`), points a recording
// fake at chrome.action, and drives the same entry points the worker's events and message listener
// call: tabJob / tabError / tabReset / tabBase / tabNavigated.

import { test } from "node:test";
import assert from "node:assert/strict";

const calls = { texts: [], colors: [] };

globalThis.chrome = {
  runtime: { getURL: (p) => `chrome-extension://test/${p}` },
  action: {
    setBadgeText: (a) => { calls.texts.push(a); },
    setBadgeBackgroundColor: (a) => { calls.colors.push(a); },
  },
};

// lastText reads the text a tab's cell wears now (paints accumulate; the last one wins).
function lastText(tabId) {
  for (let i = calls.texts.length - 1; i >= 0; i--) {
    if (calls.texts[i].tabId === tabId) return calls.texts[i].text;
  }
  return undefined;
}
function lastColor(tabId) {
  for (let i = calls.colors.length - 1; i >= 0; i--) {
    if (calls.colors[i].tabId === tabId) return calls.colors[i].color;
  }
  return undefined;
}

let caseSeq = 0;

async function fresh() {
  calls.texts.length = 0;
  calls.colors.length = 0;
  return import(`../src/badge.js?case=${++caseSeq}`);
}

test("a job with nothing to count yet paints the working ellipsis, then its counts, then a check mark", async () => {
  const badge = await fresh();
  badge.tabJob(1, "convert", "begin");
  assert.equal(lastText(1), "..");
  assert.equal(lastColor(1), "#2563eb");
  badge.tabJob(1, "convert", "progress", 3, 7);
  assert.equal(lastText(1), "3/7");
  badge.tabJob(1, "convert", "end");
  assert.equal(lastText(1), "✓");
  assert.equal(lastColor(1), "#188038");
});

test("a job's check mark is transient: it clears itself after a few seconds", async (t) => {
  t.mock.timers.enable({ apis: ["setTimeout"] });
  const badge = await fresh();
  badge.tabJob(2, "ocr", "begin", 0, 4);
  badge.tabJob(2, "ocr", "progress", 4, 4);
  badge.tabJob(2, "ocr", "end");
  assert.equal(lastText(2), "✓");
  t.mock.timers.tick(3999);
  assert.equal(lastText(2), "✓");
  t.mock.timers.tick(1);
  assert.equal(lastText(2), "");
});

test("a failure takes the cell over, and a navigation clears it", async () => {
  const badge = await fresh();
  badge.tabJob(3, "convert", "begin");
  badge.tabError(3);
  assert.equal(lastText(3), "!");
  assert.equal(lastColor(3), "#d93025");
  // A fresh document on the same tab starts over: the error is gone, the job is live again.
  badge.tabReset(3);
  assert.equal(lastText(3), "");
  badge.tabJob(3, "convert", "begin");
  assert.equal(lastText(3), "..");
  badge.tabNavigated(3);
  assert.equal(lastText(3), "");
});

test("a site switched off in the options wears the grey off badge; a site that is not, wears nothing", async () => {
  const badge = await fresh();
  const options = { enabledByDefault: true, disabledHosts: ["www.quiet.test"] };
  badge.tabBase(4, "https://www.quiet.test/shelf/", options);
  assert.equal(lastText(4), "off");
  assert.equal(lastColor(4), "#5f6368");
  badge.tabBase(5, "https://www.loud.test/shelf/", options);
  assert.equal(lastText(5), "");
  // Off globally, or no site at all: nothing to announce.
  badge.tabBase(6, "https://www.quiet.test/", { enabledByDefault: false, disabledHosts: ["www.quiet.test"] });
  assert.equal(lastText(6), "");
  badge.tabBase(7, "chrome://newtab/", options);
  assert.equal(lastText(7), "");
});

test("a new job drops the previous job's flash, and re-enabling a site drops its off badge", async () => {
  const badge = await fresh();
  const options = { enabledByDefault: true, disabledHosts: ["www.quiet.test"] };
  badge.tabBase(8, "https://www.quiet.test/a.pdf", options);
  assert.equal(lastText(8), "off");
  badge.tabJob(8, "convert", "begin");
  assert.equal(lastText(8), "..", "a running job outranks the passive off state");
  badge.tabBase(8, "https://www.quiet.test/a.pdf", { enabledByDefault: true, disabledHosts: [] });
  assert.equal(lastText(8), "..", "repainting the passive state never interrupts a live job");
  badge.tabJob(8, "convert", "end");
  assert.equal(lastText(8), "✓");
  badge.tabJob(8, "ocr", "begin", 0, 2);
  assert.equal(lastText(8), "..", "the new job's begin drops the finished job's flash");
});

test("progress for a job the worker never began is ignored, not painted as a job", async () => {
  const badge = await fresh();
  // A restarted worker has no state; a stray progress report must not invent a running job.
  badge.tabJob(9, "convert", "progress", 3, 7);
  assert.equal(lastText(9), "");
  // And an unknown job kind lands in the convert slot, the only kind the viewer reports today.
  badge.tabJob(9, "typeset", "begin");
  assert.equal(lastText(9), "..");
});
