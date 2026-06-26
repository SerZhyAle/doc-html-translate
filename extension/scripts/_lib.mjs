// _lib.mjs - shared helpers for the publish scripts. Dependency-free (Node builtins
// only), matching build.mjs. Loads a local .env, reads required env vars, and offers
// a small fetch wrapper with readable errors.

import { existsSync, readFileSync } from "node:fs";
import { resolve } from "node:path";

// loadEnv reads a .env file from the current working directory (extension/ when run
// via the npm scripts) and populates process.env for any key not already set, so a
// real CI environment's secrets always win over a local .env. Format: KEY=VALUE per
// line, # comments and blanks ignored, optional surrounding quotes stripped. Values
// may not span lines except a service-account JSON pasted on one line.
export function loadEnv(file = ".env") {
  const path = resolve(process.cwd(), file);
  if (!existsSync(path)) return;
  for (const raw of readFileSync(path, "utf8").split(/\r?\n/)) {
    const line = raw.trim();
    if (!line || line.startsWith("#")) continue;
    const eq = line.indexOf("=");
    if (eq < 0) continue;
    const key = line.slice(0, eq).trim();
    let val = line.slice(eq + 1).trim();
    if ((val.startsWith('"') && val.endsWith('"')) || (val.startsWith("'") && val.endsWith("'"))) {
      val = val.slice(1, -1);
    }
    if (!(key in process.env)) process.env[key] = val;
  }
}

export function die(msg) {
  console.error(`ERROR: ${msg}`);
  process.exit(1);
}

export function requireEnv(name, hint) {
  const v = process.env[name];
  if (!v) die(`missing ${name}${hint ? ` (${hint})` : ""}`);
  return v;
}

// parseFlags turns ["--source", "x.zip", "--upload-only"] into
// { source: "x.zip", "upload-only": true }. Values that look like another flag, or
// the end of argv, make the preceding flag a boolean.
export function parseFlags(argv) {
  const out = {};
  for (let i = 0; i < argv.length; i++) {
    if (!argv[i].startsWith("--")) continue;
    const key = argv[i].slice(2);
    const next = argv[i + 1];
    if (next && !next.startsWith("--")) { out[key] = next; i++; }
    else out[key] = true;
  }
  return out;
}

// httpText does a fetch and returns { status, headers, text }. It never throws on a
// non-2xx; callers decide. body may be a Buffer/Uint8Array/string/undefined.
export async function httpText(url, { method = "GET", headers = {}, body } = {}) {
  const res = await fetch(url, { method, headers, body });
  const text = await res.text();
  return { status: res.status, ok: res.ok, headers: res.headers, text };
}

export function parseJson(text) {
  try { return JSON.parse(text); } catch { return null; }
}

export const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
