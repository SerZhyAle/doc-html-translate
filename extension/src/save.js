// save.js - saves the files "Save media file for translation" found (media-save.js leaves their
// addresses in session storage and opens this page). Each file is read with the host access the
// extension already holds, the reader's cookies for that site included, and handed to the browser
// as its own download. Nothing is uploaded and no downloads permission is involved.

import { fileNameFor, JOB_PREFIX } from "./media-save.js";
import { saveBlobFile } from "./save-file.js";
import { t, initI18n, applyI18n, loadMessages, uiLang } from "./i18n.js";
import { applyGlyphs } from "./glyphs.js";
import { getOptions } from "./options-store.js";
import { applyUiProfile, revealPage } from "./ui-profile.js";

// How long the tab of a single saved file stays: long enough for the browser to take the file over.
const CLOSE_AFTER_MS = 8000;

const statusEl = document.getElementById("status");
const listEl = document.getElementById("files");
const hintEl = document.getElementById("hint");

async function closeTab() {
  try {
    const tab = await chrome.tabs.getCurrent();
    await chrome.tabs.remove(tab.id);
  } catch { window.close(); }
}

function line(text) {
  const li = document.createElement("li");
  li.textContent = text;
  listEl.append(li);
}

async function saveOne(url) {
  const res = await fetch(url, { credentials: "include" });
  if (!res.ok) throw new Error(`HTTP ${res.status}`);
  const blob = await res.blob();
  const name = fileNameFor({ url, disposition: res.headers.get("content-disposition"), type: blob.type });
  saveBlobFile(name, blob);
  return name;
}

// takeJob reads the addresses once: a reload of the page saves nothing again.
async function takeJob() {
  const id = location.hash.slice(1);
  if (!id) return [];
  const key = JOB_PREFIX + id;
  const job = (await chrome.storage.session.get(key))[key];
  await chrome.storage.session.remove(key);
  return job && Array.isArray(job.urls) ? job.urls : [];
}

async function run() {
  await initI18n();
  await loadMessages(uiLang());
  applyUiProfile(await getOptions());
  applyI18n(document);
  applyGlyphs(document);
  document.getElementById("close").addEventListener("click", closeTab);
  revealPage();

  const urls = await takeJob();
  if (!urls.length) {
    statusEl.textContent = t("savePageNothing", "Nothing to save. Use the right-click command again.");
    return;
  }
  hintEl.hidden = urls.length < 2;
  let failed = 0;
  for (const [i, url] of urls.entries()) {
    statusEl.textContent = t("savePageSaving", "Saving {1} of {2}..", i + 1, urls.length);
    try {
      line(t("savePageItemSaved", "{1} - saved to the download folder", await saveOne(url)));
    } catch (e) {
      failed++;
      line(t("savePageItemFailed", "{1} - could not save: {2}", fileNameFor({ url }), String((e && e.message) || e)));
    }
  }
  statusEl.textContent = "";
  if (urls.length === 1 && !failed) setTimeout(closeTab, CLOSE_AFTER_MS);
  // A failed single file brings the tab forward, so the reason is seen.
  if (failed) {
    try { await chrome.tabs.update((await chrome.tabs.getCurrent()).id, { active: true }); } catch { /* the tab stays where it is */ }
  }
}

run().catch((e) => { statusEl.textContent = String((e && e.message) || e); revealPage(); });
