// _ocrlab-cdp.mjs - the dependency-free DevTools client the lab runner drives the browser with.
//
// Node ships a global WebSocket, which is the only piece the Chrome DevTools Protocol actually
// needs, so the extension keeps depending on Node builtins alone rather than pulling in Puppeteer
// and its own Chrome download. Same pattern as scripts/make-screenshots.mjs, which needs a browser
// with the extension loaded for the same reason: the viewer only exists under a
// chrome-extension:// origin.

import { existsSync } from "node:fs";
import { die } from "./_lib.mjs";

export const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

// CallTimeoutMs bounds one protocol call, mirroring browserTimeout in tools/ocrlab/runner/cdp.go.
// Without it a wedged renderer never answers Runtime.evaluate and the whole run stops with no
// output at all - which is what happened on a 2.7 MB scan and is exactly the silent, unbounded
// outcome the strategic spec forbids. A hung call is a scene that failed, and the runner already
// records a failed scene and carries on.
export const CallTimeoutMs = 90_000;

export class CDP {
  constructor(ws) {
    this.ws = ws;
    this.nextId = 1;
    this.pending = new Map();
    ws.addEventListener("message", (ev) => {
      const msg = JSON.parse(ev.data);
      if (!msg.id || !this.pending.has(msg.id)) return;
      const { resolve: ok, reject } = this.pending.get(msg.id);
      this.pending.delete(msg.id);
      if (msg.error) reject(new Error(`${msg.error.message} (${JSON.stringify(msg.error.data ?? "")})`));
      else ok(msg.result);
    });
    // A browser that dies mid-call must fail every waiting call rather than leave them pending
    // for ever - the same reason the timeout exists. It stays dead afterwards: once the transport
    // is gone every later call is going to fail, and waiting the full timeout on each one turns a
    // crash on scene 7 into an hour of grinding through 38 scenes that never had a browser.
    const fail = (why) => () => {
      this.dead = why;
      for (const [id, { reject }] of this.pending) {
        this.pending.delete(id);
        reject(new Error(why));
      }
    };
    ws.addEventListener("close", fail("the browser closed the DevTools connection"), { once: true });
    ws.addEventListener("error", fail("the DevTools connection errored"), { once: true });
  }

  static async connect(url) {
    const ws = new WebSocket(url);
    await new Promise((ok, reject) => {
      ws.addEventListener("open", ok, { once: true });
      ws.addEventListener("error", () => reject(new Error(`cannot connect to ${url}`)), { once: true });
    });
    return new CDP(ws);
  }

  send(method, params = {}, sessionId, timeoutMs = CallTimeoutMs) {
    if (this.dead) return Promise.reject(new Error(`${method}: ${this.dead}`));
    const id = this.nextId++;
    const payload = { id, method, params };
    if (sessionId) payload.sessionId = sessionId;
    return new Promise((ok, reject) => {
      const timer = setTimeout(() => {
        this.pending.delete(id);
        reject(new Error(`${method}: no reply after ${timeoutMs}ms`));
      }, timeoutMs);
      const settle = (fn) => (v) => { clearTimeout(timer); fn(v); };
      this.pending.set(id, { resolve: settle(ok), reject: settle(reject) });
      this.ws.send(JSON.stringify(payload));
    });
  }

  close() { try { this.ws.close(); } catch { /* already gone */ } }
}

// evaluate runs an expression in the page and returns its value. Anything the page throws
// surfaces here rather than being swallowed into a blank measurement.
export async function evaluate(cdp, session, expression, awaitPromise = false) {
  const res = await cdp.send("Runtime.evaluate", { expression, awaitPromise, returnByValue: true }, session);
  if (res.exceptionDetails) {
    throw new Error(`page threw: ${res.exceptionDetails.text} ${res.exceptionDetails.exception?.description ?? ""}`);
  }
  return res.result?.value;
}

export async function waitFor(label, predicate, timeoutMs, everyMs = 250) {
  const deadline = Date.now() + timeoutMs;
  for (;;) {
    const value = await predicate();
    if (value) return value;
    if (Date.now() > deadline) throw new Error(`timed out waiting for ${label}`);
    await sleep(everyMs);
  }
}

// findChrome locates the browser to drive. A missing one is a hard error: the strategic spec is
// explicit that a missing dependency fails the run rather than silently reducing its coverage.
export function findChrome() {
  if (process.env.CHROME) return process.env.CHROME;
  const candidates = [
    `${process.env.ProgramFiles}\\Google\\Chrome\\Application\\chrome.exe`,
    `${process.env["ProgramFiles(x86)"]}\\Google\\Chrome\\Application\\chrome.exe`,
    `${process.env.LOCALAPPDATA}\\Google\\Chrome\\Application\\chrome.exe`,
    `${process.env.ProgramFiles}\\Microsoft\\Edge\\Application\\msedge.exe`,
    `${process.env["ProgramFiles(x86)"]}\\Microsoft\\Edge\\Application\\msedge.exe`,
  ];
  const found = candidates.find((p) => p && existsSync(p));
  if (!found) die("no browser found - set CHROME=<path to chrome.exe> (Edge works too)");
  return found;
}
