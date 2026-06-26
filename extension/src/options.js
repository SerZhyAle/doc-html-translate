// options.js - default on/off, reading theme, source-language hint, and a view of
// the per-site disable list. Persists the shared `options` object.

const DEFAULT_OPTIONS = { enabledByDefault: true, disabledHosts: [], sourceLang: "auto", theme: "light" };

const enabledEl = document.getElementById("enabled");
const themeEl = document.getElementById("theme");
const langEl = document.getElementById("lang");
const hostsEl = document.getElementById("hosts");

async function getOptions() {
  const got = await chrome.storage.local.get("options");
  return { ...DEFAULT_OPTIONS, ...(got.options || {}) };
}
async function setOptions(opts) {
  await chrome.storage.local.set({ options: opts });
}

function flash(id) {
  const e = document.getElementById(id);
  if (!e) return;
  e.classList.add("show");
  setTimeout(() => e.classList.remove("show"), 900);
}

function renderHosts(hosts) {
  hostsEl.replaceChildren();
  if (!hosts.length) {
    const li = document.createElement("li");
    li.className = "hint";
    li.textContent = "None";
    hostsEl.append(li);
    return;
  }
  for (const h of hosts) {
    const li = document.createElement("li");
    const btn = document.createElement("button");
    btn.textContent = "remove";
    btn.style.marginLeft = "0.6rem";
    btn.addEventListener("click", async () => {
      const o = await getOptions();
      o.disabledHosts = o.disabledHosts.filter((x) => x !== h);
      await setOptions(o);
      renderHosts(o.disabledHosts);
    });
    li.textContent = h;
    li.append(btn);
    hostsEl.append(li);
  }
}

async function init() {
  const o = await getOptions();
  enabledEl.checked = o.enabledByDefault;
  themeEl.value = o.theme;
  langEl.value = o.sourceLang;
  renderHosts(o.disabledHosts);

  enabledEl.addEventListener("change", async () => {
    const opts = await getOptions();
    opts.enabledByDefault = enabledEl.checked;
    await setOptions(opts);
  });
  themeEl.addEventListener("change", async () => {
    const opts = await getOptions();
    opts.theme = themeEl.value;
    await setOptions(opts);
    flash("saved-theme");
  });
  langEl.addEventListener("change", async () => {
    const opts = await getOptions();
    opts.sourceLang = langEl.value;
    await setOptions(opts);
  });

  chrome.storage.onChanged.addListener((changes, area) => {
    if (area === "local" && changes.options) renderHosts((changes.options.newValue || {}).disabledHosts || []);
  });
}

init();
