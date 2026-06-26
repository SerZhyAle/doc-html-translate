// publish-cws.mjs - upload + publish a new version to the Chrome Web Store via the
// V2 Publish API (chromewebstore.googleapis.com), dependency-free.
//
//   node scripts/publish-cws.mjs                 upload dist zip, then publish to all
//   node scripts/publish-cws.mjs --upload-only   upload only (review the draft by hand)
//   node scripts/publish-cws.mjs --publish-only   publish the existing draft, no upload
//   node scripts/publish-cws.mjs --percentage 10 staged rollout (needs >~10k users)
//   node scripts/publish-cws.mjs --source path/to.zip
//
// Auth (one is required), preferring the service account (recommended for CI - it has
// no 7-day / 6-month token expiry, unlike an OAuth refresh token):
//   CWS_SERVICE_ACCOUNT_KEY        the service-account JSON key, inline (CI secret), or
//   CWS_SERVICE_ACCOUNT_KEY_FILE   path to that JSON key file, or
//   CWS_CLIENT_ID + CWS_CLIENT_SECRET + CWS_REFRESH_TOKEN   the OAuth fallback.
// Always required: CWS_PUBLISHER_ID, CWS_EXTENSION_ID.
//
// The FIRST publish (creating the item + filling the Store listing / Privacy tabs)
// cannot be automated - the V2 API has no create-item or listing-metadata endpoint.
// See PUBLISHING.md. The V1 API is sunset 2026-10-15; this targets V2 only.

import { readFile } from "node:fs/promises";
import { existsSync } from "node:fs";
import { createSign } from "node:crypto";
import { loadEnv, die, requireEnv, parseFlags, httpText, parseJson, sleep } from "./_lib.mjs";

loadEnv();

const SCOPE = "https://www.googleapis.com/auth/chromewebstore";
const TOKEN_URL = "https://oauth2.googleapis.com/token";
const HOST = "https://chromewebstore.googleapis.com";

const flags = parseFlags(process.argv.slice(2));
const PUBLISHER_ID = requireEnv("CWS_PUBLISHER_ID", "your developer/publisher GUID");
const EXTENSION_ID = requireEnv("CWS_EXTENSION_ID", "the 32-char item ID from the dashboard");
const SOURCE = flags.source || process.env.CWS_ZIP || "dist/doc-html-translate-extension.zip";

const b64url = (buf) =>
  Buffer.from(buf).toString("base64").replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");

// ---- access token ----------------------------------------------------------
async function getAccessToken() {
  let keyJson = process.env.CWS_SERVICE_ACCOUNT_KEY;
  if (!keyJson && process.env.CWS_SERVICE_ACCOUNT_KEY_FILE) {
    keyJson = await readFile(process.env.CWS_SERVICE_ACCOUNT_KEY_FILE, "utf8");
  }
  if (keyJson) return tokenFromServiceAccount(JSON.parse(keyJson));

  if (process.env.CWS_CLIENT_ID && process.env.CWS_CLIENT_SECRET && process.env.CWS_REFRESH_TOKEN) {
    return tokenFromRefreshToken();
  }
  die("no Chrome credentials - set CWS_SERVICE_ACCOUNT_KEY[_FILE] or CWS_CLIENT_ID/SECRET/REFRESH_TOKEN");
}

async function tokenFromServiceAccount(key) {
  const now = Math.floor(Date.now() / 1000);
  const header = b64url(JSON.stringify({ alg: "RS256", typ: "JWT" }));
  const claim = b64url(JSON.stringify({
    iss: key.client_email,
    scope: SCOPE,
    aud: key.token_uri || TOKEN_URL,
    iat: now,
    exp: now + 3600,
  }));
  const signature = b64url(createSign("RSA-SHA256").update(`${header}.${claim}`).sign(key.private_key));
  const assertion = `${header}.${claim}.${signature}`;
  const body = new URLSearchParams({
    grant_type: "urn:ietf:params:oauth:grant-type:jwt-bearer",
    assertion,
  });
  const { ok, text } = await httpText(key.token_uri || TOKEN_URL, {
    method: "POST",
    headers: { "Content-Type": "application/x-www-form-urlencoded" },
    body: body.toString(),
  });
  const json = parseJson(text);
  if (!ok || !json?.access_token) die(`service-account token exchange failed: ${text}`);
  return json.access_token;
}

async function tokenFromRefreshToken() {
  const body = new URLSearchParams({
    client_id: process.env.CWS_CLIENT_ID,
    client_secret: process.env.CWS_CLIENT_SECRET,
    refresh_token: process.env.CWS_REFRESH_TOKEN,
    grant_type: "refresh_token",
  });
  const { ok, text } = await httpText(TOKEN_URL, {
    method: "POST",
    headers: { "Content-Type": "application/x-www-form-urlencoded" },
    body: body.toString(),
  });
  const json = parseJson(text);
  if (!ok || !json?.access_token) die(`refresh-token exchange failed (a 'Testing' consent screen expires the token after 7 days): ${text}`);
  return json.access_token;
}

// ---- API calls -------------------------------------------------------------
async function upload(token) {
  if (!existsSync(SOURCE)) die(`package not found: ${SOURCE} - run \`npm run build\` first`);
  const bytes = await readFile(SOURCE);
  console.log(`Uploading ${SOURCE} (${(bytes.length / 1024 / 1024).toFixed(1)} MB)..`);
  const { ok, text } = await httpText(
    `${HOST}/upload/v2/publishers/${PUBLISHER_ID}/items/${EXTENSION_ID}:upload`,
    { method: "POST", headers: { Authorization: `Bearer ${token}`, "Content-Type": "application/zip" }, body: bytes },
  );
  const json = parseJson(text) || {};
  const state = json.uploadState || (ok ? "SUCCESS" : "FAILURE");
  if (!ok || state === "FAILURE") {
    const errs = (json.itemError || []).map((e) => e.error_detail || e.errorDetail || JSON.stringify(e)).join("; ");
    die(`upload failed (${state}): ${errs || text}\nHint: did you bump manifest.json version? Re-uploading the same version is rejected.`);
  }
  console.log(`Upload state: ${state}`);
}

async function publish(token) {
  let body;
  if (flags.percentage != null && flags.percentage !== true) {
    body = JSON.stringify({ publishType: "STAGED_PUBLISH", deployInfos: [{ deployPercentage: Number(flags.percentage) }] });
    console.log(`Publishing (staged ${flags.percentage}%)..`);
  } else {
    console.log("Publishing to the item's current visibility..");
  }
  const headers = { Authorization: `Bearer ${token}` };
  if (body) headers["Content-Type"] = "application/json";
  const { ok, text } = await httpText(
    `${HOST}/v2/publishers/${PUBLISHER_ID}/items/${EXTENSION_ID}:publish`,
    { method: "POST", headers, body },
  );
  if (!ok) die(`publish failed: ${text}\nHint: the API publishes the existing visibility; a new item must be published once in the dashboard first.`);
  console.log(`Publish response: ${text}`);
}

async function fetchStatus(token) {
  const { text } = await httpText(
    `${HOST}/v2/publishers/${PUBLISHER_ID}/items/${EXTENSION_ID}:fetchStatus`,
    { headers: { Authorization: `Bearer ${token}` } },
  );
  console.log(`Status: ${text}`);
}

// ---- main ------------------------------------------------------------------
const token = await getAccessToken();
if (!flags["publish-only"]) await upload(token);
if (!flags["upload-only"]) {
  await sleep(1500); // give the upload a moment to register before publishing
  await publish(token);
}
await fetchStatus(token);
console.log("Done. Review/approval still happens out-of-band in the dashboard.");
