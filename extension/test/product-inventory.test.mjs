// The About page is drawn from product-inventory.js (SETTINGS-NAVIGATION 2.0 section 3 rules 3 and 4): one
// inventory, localized labels, and only addresses the repository already publishes - nothing invented.

import test from "node:test";
import assert from "node:assert/strict";
import fs from "node:fs";
import path from "node:path";
import { LINKS, UPDATE_SOURCES, RESOURCES, RELATED } from "../src/product-inventory.js";

const ext = path.join(import.meta.dirname, "..");
const repo = path.join(ext, "..");
const en = JSON.parse(fs.readFileSync(path.join(ext, "_locales", "en", "messages.json"), "utf8"));
const read = (...p) => fs.readFileSync(path.join(repo, ...p), "utf8");

const every = () => [...Object.values(LINKS), ...UPDATE_SOURCES, ...RELATED];

test("every label of the inventory is a message in every locale", () => {
  const keys = [...Object.values(LINKS), ...UPDATE_SOURCES, ...RESOURCES, ...RELATED].map((e) => e.key).filter(Boolean);
  assert.ok(keys.length >= 10);
  for (const dir of fs.readdirSync(path.join(ext, "_locales"))) {
    const messages = JSON.parse(fs.readFileSync(path.join(ext, "_locales", dir, "messages.json"), "utf8"));
    for (const key of keys) assert.ok(messages[key]?.message.trim(), `${dir}: no message for ${key}`);
  }
  for (const key of ["relOpen", "aboutLicence", "optUpdatesNote", "optResIntro"]) assert.ok(key in en, key);
});

test("every outside address is one the repository already publishes", () => {
  const listing = read("extension", "store", "LISTING.md");
  const site = read("index.html") + read("extension.html");
  for (const e of every()) {
    const href = e.href;
    if (e.local) continue;
    if (href.startsWith("https://serzhyale.github.io/doc-html-translate/")) {
      const page = href.slice("https://serzhyale.github.io/doc-html-translate/".length);
      assert.ok(page === "" || fs.existsSync(path.join(repo, page)), `${href} names a site page that does not exist`);
    } else if (href.startsWith("https://github.com/SerZhyAle/doc-html-translate")) {
      assert.ok(site.includes(href) || read("README.md").includes(href), `${href} is not published anywhere in the repository`);
    } else if (href.startsWith("https://chromewebstore.google.com/") || href.startsWith("https://microsoftedge.microsoft.com/")) {
      assert.ok(listing.includes(href), `${href} is not the store listing named in store/LISTING.md`);
    } else if (href.startsWith("mailto:")) {
      assert.equal(href, "mailto:sza@ukr.net");
    } else {
      assert.ok(site.includes(href), `${href} is not an address the site publishes`);
    }
  }
});

test("the files the About page links inside the package exist", () => {
  const src = path.join(ext, "src");
  assert.ok(fs.existsSync(path.join(src, LINKS.notices.href)), "THIRD-PARTY-NOTICES.txt");
  const built = fs.existsSync(path.join(ext, "vendor", "pdf.mjs"));
  for (const r of RESOURCES) {
    if (!r.licence) continue;
    // vendor/ is produced by `npm run vendor` and is not in a fresh checkout; the notices file always is.
    if (r.licence.startsWith("../vendor/") && !built) continue;
    assert.ok(fs.existsSync(path.join(src, r.licence)), `${r.id}: ${r.licence} is missing from the package`);
  }
});

test("ids are unique, names are present, and no entry is a placeholder", () => {
  for (const list of [UPDATE_SOURCES, RESOURCES, RELATED]) {
    const ids = list.map((e) => e.id);
    assert.equal(new Set(ids).size, ids.length);
    for (const e of list) assert.ok((e.name || e.key) && e.id, JSON.stringify(e));
  }
  for (const e of every()) assert.ok(e.href && !/example\.|TODO|#$/.test(e.href), e.href);
});
