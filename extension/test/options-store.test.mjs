// options-store.js keeps the stored options and the interface's remembered state in two separate keys. The split is the
// point: every write of `options` makes background.js rebuild the interception rules and refresh every tab's badge, and a
// click on the settings rail must not run an operation (SETTINGS-NAVIGATION 2.0 section 4 rule 3).

import test from "node:test";
import assert from "node:assert/strict";
import { DEFAULT_OPTIONS } from "../src/defaults.js";

function stubStorage(initial = {}) {
  const store = { ...initial };
  const writes = [];
  globalThis.chrome = {
    storage: {
      local: {
        get: async (keys) => {
          const out = {};
          for (const k of [].concat(keys)) if (k in store) out[k] = structuredClone(store[k]);
          return out;
        },
        set: async (obj) => { writes.push(Object.keys(obj)); Object.assign(store, structuredClone(obj)); },
      },
    },
  };
  return { store, writes };
}

const { getOptions, setOptions, getUiState, setUiState } = await import("../src/options-store.js");

test("the options read merges the defaults, so a new key is never undefined for an old profile", async () => {
  stubStorage({ options: { enabledByDefault: true } });
  assert.deepEqual(await getOptions(), { ...DEFAULT_OPTIONS, enabledByDefault: true });
});

test("remembering a page or a group writes uiState and never options", async () => {
  const { store, writes } = stubStorage({ options: { theme: "dark" } });
  await setUiState({ page: "about" });
  await setUiState({ groups: { aboutSupport: false } });
  assert.deepEqual(writes, [["uiState"], ["uiState"]], "only the remembered state was written");
  assert.deepEqual(store.options, { theme: "dark" }, "options are untouched");
  assert.deepEqual(await getUiState(), { page: "about", groups: { aboutSupport: false } });
});

test("groups and viewports merge key by key, so two pages saving different groups never undo each other", async () => {
  stubStorage();
  await setUiState({ groups: { aboutSupport: false } });
  await setUiState({ groups: { popupOcrLangs: true } });
  await setUiState({ scroll: { about: { id: "grpX", off: 40 } } });
  await setUiState({ scroll: { ocr: { id: "ocr-images", off: 4 } } });
  const state = await getUiState();
  assert.deepEqual(state.groups, { aboutSupport: false, popupOcrLangs: true });
  assert.deepEqual(state.scroll, { about: { id: "grpX", off: 40 }, ocr: { id: "ocr-images", off: 4 } });
  await setUiState({ scroll: { about: null } });
  assert.deepEqual((await getUiState()).scroll, { ocr: { id: "ocr-images", off: 4 } }, "a null anchor forgets the page's viewport");
});

test("writes that are not awaited land in call order", async () => {
  stubStorage();
  setUiState({ page: "ocr" });
  setUiState({ groups: { aboutRelated: true } });
  await setUiState({ page: "sites" });
  const state = await getUiState();
  assert.equal(state.page, "sites");
  assert.deepEqual(state.groups, { aboutRelated: true });
});

test("a corrupt stored state falls back to a valid one, and a failing storage read does too", async () => {
  stubStorage({ uiState: { page: 42, groups: "x", scroll: [] } });
  assert.deepEqual(await getUiState(), { page: "general", groups: {} });
  globalThis.chrome = { storage: { local: { get: async () => { throw new Error("storage unavailable"); } } } };
  assert.deepEqual(await getUiState(), { page: "general", groups: {} });
});

test("setOptions writes the options key alone", async () => {
  const { writes } = stubStorage();
  await setOptions({ ...DEFAULT_OPTIONS, uiSize: "huge" });
  assert.deepEqual(writes, [["options"]]);
});
