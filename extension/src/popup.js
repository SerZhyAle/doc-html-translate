// popup.js - global on/off and per-site disable. Writes the shared `options`
// object to storage; background.js rebuilds the DNR rules on the storage change.

const DEFAULT_OPTIONS = { enabledByDefault: true, disabledHosts: [], sourceLang: "auto", theme: "light" };

const globalEl = document.getElementById("global");
const siteEl = document.getElementById("site");
const hostEl = document.getElementById("host");

async function getOptions() {
  const got = await chrome.storage.local.get("options");
  return { ...DEFAULT_OPTIONS, ...(got.options || {}) };
}
async function setOptions(opts) {
  await chrome.storage.local.set({ options: opts });
}

async function activeHost() {
  try {
    const [tab] = await chrome.tabs.query({ active: true, currentWindow: true });
    if (tab && tab.url) return new URL(tab.url).hostname;
  } catch { /* no host access */ }
  return "";
}

async function init() {
  const opts = await getOptions();
  const host = await activeHost();

  globalEl.checked = opts.enabledByDefault;

  if (host) {
    hostEl.textContent = host;
    siteEl.disabled = !opts.enabledByDefault;
    siteEl.checked = opts.enabledByDefault && !opts.disabledHosts.includes(host);
  } else {
    hostEl.textContent = "(not a website)";
    siteEl.disabled = true;
  }

  globalEl.addEventListener("change", async () => {
    const o = await getOptions();
    o.enabledByDefault = globalEl.checked;
    await setOptions(o);
    siteEl.disabled = !o.enabledByDefault || !host;
    siteEl.checked = o.enabledByDefault && host && !o.disabledHosts.includes(host);
  });

  siteEl.addEventListener("change", async () => {
    if (!host) return;
    const o = await getOptions();
    const set = new Set(o.disabledHosts);
    if (siteEl.checked) set.delete(host);
    else set.add(host);
    o.disabledHosts = [...set];
    await setOptions(o);
  });

  document.getElementById("open-pdf").addEventListener("click", () => {
    // Open the viewer's empty state in a new tab; the user clicks "Open a PDF file"
    // there (the OS file dialog needs a user gesture on the viewer page itself).
    chrome.tabs.create({ url: chrome.runtime.getURL("src/viewer.html") });
    window.close();
  });

  document.getElementById("opts").addEventListener("click", (e) => {
    e.preventDefault();
    chrome.runtime.openOptionsPage();
  });
}

init();
