import { test } from "node:test";
import assert from "node:assert/strict";
import fs from "node:fs";
import vm from "node:vm";

// Drive the actual navigation functions with both live system preferences. No vendor engine
// is needed: only the browser's scroll policy is under test (ticket 90).
const source = fs.readFileSync(new URL("../src/viewer.js", import.meta.url), "utf8");
const navigation = source.slice(source.indexOf("async function scrollToPage("), source.indexOf("// ---- Page rendering"));
for (const reduced of [false, true]) {
  test(`reader navigation honours reduced motion = ${reduced}`, async () => {
    const calls = [];
    const target = { scrollIntoView: (opts) => calls.push(opts) };
    const context = vm.createContext({
      window: { matchMedia: (query) => {
        assert.equal(query, "(prefers-reduced-motion: reduce)");
        return { matches: reduced };
      } },
      document: { querySelector: () => target, getElementById: () => target },
      ensurePageRendered: async () => {},
    });
    vm.runInContext(navigation, context);
    await context.scrollToPage(2);
    context.scrollToAnchor("chapter");
    assert.equal(calls.length, 2);
    for (const call of calls) {
      assert.equal(call.behavior, reduced ? "auto" : "smooth");
      assert.equal(call.block, "start");
    }
  });
}
