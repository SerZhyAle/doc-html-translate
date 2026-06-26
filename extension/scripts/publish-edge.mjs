// publish-edge.mjs - upload + publish a new version to Microsoft Edge Add-ons via the
// Partner Center Update REST API v1.1 (ApiKey auth), dependency-free.
//
//   node scripts/publish-edge.mjs                upload dist zip, then publish
//   node scripts/publish-edge.mjs --upload-only  upload the draft only, publish by hand
//   node scripts/publish-edge.mjs --source path/to.zip
//
// Required env: EDGE_PRODUCT_ID, EDGE_CLIENT_ID, EDGE_API_KEY. Optional: EDGE_NOTES.
//
// The v1.1 API uses headers `Authorization: ApiKey <key>` + `X-ClientID: <id>` (the old
// v1 login.microsoftonline.com OAuth flow was deprecated 2025-01-10 - do not use it).
// As with Chrome, the FIRST submission (creating the product + listing/screenshots/
// privacy) must be done by hand in Partner Center to obtain EDGE_PRODUCT_ID; only
// version updates are automatable. NOTE: the v1.1 API key expires ~every 72 days -
// rotate it in Partner Center and update EDGE_API_KEY, or this starts returning 401.

import { readFile } from "node:fs/promises";
import { existsSync } from "node:fs";
import { loadEnv, die, requireEnv, parseFlags, httpText, parseJson, sleep } from "./_lib.mjs";

loadEnv();

const flags = parseFlags(process.argv.slice(2));
const PRODUCT_ID = requireEnv("EDGE_PRODUCT_ID", "the product GUID from Partner Center");
const CLIENT_ID = requireEnv("EDGE_CLIENT_ID");
const API_KEY = requireEnv("EDGE_API_KEY");
const SOURCE = flags.source || process.env.EDGE_ZIP || "dist/doc-html-translate-extension.zip";
const NOTES = process.env.EDGE_NOTES || "Automated submission via the Edge Add-ons API.";

const BASE = `https://api.addons.microsoftedge.microsoft.com/v1/products/${PRODUCT_ID}`;
const AUTH = { Authorization: `ApiKey ${API_KEY}`, "X-ClientID": CLIENT_ID };

// operationId from the Location header (".../operations/<id>") or a JSON {id} body.
function operationId(res, text) {
  const loc = res.headers.get("location") || res.headers.get("Location");
  if (loc) return loc.split("/").filter(Boolean).pop();
  return parseJson(text)?.id || "";
}

// poll a Partner Center operation URL until it stops being InProgress.
async function poll(url, what) {
  for (let i = 0; i < 60; i++) {
    const { ok, text } = await httpText(url, { headers: AUTH });
    const json = parseJson(text) || {};
    const status = json.status || (ok ? "Unknown" : "Failed");
    if (status === "Succeeded") { console.log(`${what}: Succeeded`); return json; }
    if (status === "Failed") {
      die(`${what} failed: ${json.message || ""} ${json.errorCode || ""} ${JSON.stringify(json.errors || "")}`.trim());
    }
    process.stdout.write(`  ${what}: ${status}..\r`);
    await sleep(5000);
  }
  die(`${what} timed out`);
}

async function upload() {
  if (!existsSync(SOURCE)) die(`package not found: ${SOURCE} - run \`npm run build\` first`);
  const bytes = await readFile(SOURCE);
  console.log(`Uploading ${SOURCE} (${(bytes.length / 1024 / 1024).toFixed(1)} MB) to Edge..`);
  const res = await fetch(`${BASE}/submissions/draft/package`, {
    method: "POST",
    headers: { ...AUTH, "Content-Type": "application/zip" },
    body: bytes,
  });
  const text = await res.text();
  if (res.status !== 202) die(`upload rejected (HTTP ${res.status}): ${text}`);
  await poll(`${BASE}/submissions/draft/package/operations/${operationId(res, text)}`, "Upload");
}

async function publish() {
  console.log("Publishing the Edge draft submission..");
  const res = await fetch(`${BASE}/submissions`, {
    method: "POST",
    headers: { ...AUTH, "Content-Type": "application/json" },
    body: JSON.stringify({ notes: NOTES }),
  });
  const text = await res.text();
  if (res.status !== 202) die(`publish rejected (HTTP ${res.status}): ${text}`);
  await poll(`${BASE}/submissions/operations/${operationId(res, text)}`, "Publish");
}

// ---- main ------------------------------------------------------------------
await upload();
if (!flags["upload-only"]) await publish();
console.log("Done. Microsoft certification still runs out-of-band.");
