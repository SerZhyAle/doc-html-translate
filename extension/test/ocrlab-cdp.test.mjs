// Tests for the lab's DevTools transport (scripts/_ocrlab-cdp.mjs).
//
// One property is worth a test here, and it is the one that cost a whole baseline run: a call the
// browser never answers must fail, not wait for ever. A wedged renderer used to hang
// Runtime.evaluate, which hung waitFor's predicate before its deadline was ever read, which hung
// the run with no evidence file and no failed scene - the silent, unbounded outcome the strategic
// spec forbids. tools/ocrlab/runner/cdp.go has always bounded its calls; this is the JS side of
// the same contract.

import { test } from "node:test";
import assert from "node:assert/strict";
import { CDP, CallTimeoutMs } from "../scripts/_ocrlab-cdp.mjs";

// A stand-in for the WebSocket: it records what was sent and answers only what a test tells it
// to. Node's global WebSocket cannot be pointed at nothing, and the behaviour under test is
// entirely about replies that do not arrive.
function fakeSocket() {
  const listeners = new Map();
  return {
    sent: [],
    addEventListener(type, fn) { listeners.set(type, [...(listeners.get(type) ?? []), fn]); },
    send(data) { this.sent.push(JSON.parse(data)); },
    close() {},
    emit(type, ev) { for (const fn of listeners.get(type) ?? []) fn(ev); },
  };
}

test("a call the browser never answers rejects instead of hanging", async () => {
  const ws = fakeSocket();
  const cdp = new CDP(ws);
  await assert.rejects(
    cdp.send("Runtime.evaluate", { expression: "1" }, undefined, 40),
    /Runtime\.evaluate: no reply after 40ms/,
  );
  assert.equal(cdp.pending.size, 0, "a timed-out call must not stay pending");
});

test("a reply resolves the call and cancels its timer", async () => {
  const ws = fakeSocket();
  const cdp = new CDP(ws);
  const call = cdp.send("Browser.getVersion");
  ws.emit("message", { data: JSON.stringify({ id: ws.sent[0].id, result: { product: "Chrome/151" } }) });
  assert.deepEqual(await call, { product: "Chrome/151" });
  assert.equal(cdp.pending.size, 0);
});

test("a browser that dies mid-call fails every waiting call", async () => {
  const ws = fakeSocket();
  const cdp = new CDP(ws);
  const first = cdp.send("Page.navigate", { url: "about:blank" });
  const second = cdp.send("Page.captureScreenshot");
  ws.emit("close", {});
  await assert.rejects(first, /closed the DevTools connection/);
  await assert.rejects(second, /closed the DevTools connection/);
  assert.equal(cdp.pending.size, 0);
});

test("the default call bound matches the desktop runner's", () => {
  assert.equal(CallTimeoutMs, 90_000, "tools/ocrlab/runner/cdp.go browserTimeout is 90s");
});
