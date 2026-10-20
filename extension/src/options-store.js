// options-store.js - the one place the extension pages read and write the stored `options`
// object and the interface's remembered state. popup.js, options.js and ocr-lang-ui.js each used
// to carry their own copy of the options pair.
//
// `options` and `uiState` are separate storage keys on purpose. Every write of `options` makes
// background.js rebuild the interception rules and refresh every tab's badge; the page and group
// memory changes on each navigation click, and a click on a settings rail must not run an
// operation (SETTINGS-NAVIGATION 4 rule 3).

import { DEFAULT_OPTIONS } from "./defaults.js";
import { normalizeUiState } from "./ui-settings.js";

export async function getOptions() {
  const got = await chrome.storage.local.get("options");
  return { ...DEFAULT_OPTIONS, ...(got.options || {}) };
}

export async function setOptions(opts) {
  await chrome.storage.local.set({ options: opts });
}

export async function getUiState() {
  try {
    return normalizeUiState((await chrome.storage.local.get("uiState")).uiState);
  } catch {
    return normalizeUiState(null);
  }
}

// setUiState merges a patch into the remembered state. `groups` and `scroll` merge key by key,
// so two pages saving different groups never undo each other. Writes run one after another: each
// is a read-modify-write, and unawaited callers would overwrite each other's keys.
let writeTail = Promise.resolve();

export function setUiState(patch) {
  const run = writeTail.then(async () => {
    const state = await getUiState();
    const next = { ...state, ...patch };
    if (patch.groups) next.groups = { ...state.groups, ...patch.groups };
    if (patch.scroll) next.scroll = { ...(state.scroll || {}), ...patch.scroll };
    await chrome.storage.local.set({ uiState: normalizeUiState(next) });
  });
  writeTail = run.catch(() => {});
  return run;
}
