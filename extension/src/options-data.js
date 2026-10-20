// options-data.js - the options page's "Data" destination: export, import and reset of the settings file.
//
// The two actions that overwrite what the reader has - an import that changes something, a reset - ask
// first in the product's own dialog (BUTTONS 2.0 section 4 rules 4 and 5): Cancel first and focused, the
// acting answer on the danger face, naming the consequence. An operation that would change nothing asks
// nothing. The settings file never carries documents, reading positions or history (settings-transfer.js).

import { DEFAULT_OPTIONS } from "./defaults.js";
import { makeSettingsFile, validateSettingsFile, changedSettings } from "./settings-transfer.js";
import { DEFAULT_UI_STATE } from "./ui-settings.js";
import { getUiState } from "./options-store.js";
import { confirmDialog } from "./ui-dialog.js";
import { saveTextFile } from "./save-file.js";
import { t } from "./i18n.js";

const MAX_FILE_BYTES = 1024 * 1024;
const msg = (key, fallback, ...args) => t(key, fallback, ...args);

function showStatus(message, error = false) {
  const status = document.getElementById("settings-status");
  status.textContent = message;
  status.classList.toggle("error", error);
}

// The remembered page and group states are not a setting the reader would name; a difference in them alone
// is not a change worth asking about.
const meaningful = (changed) => changed.filter((key) => key !== "ui");

async function currentSettings() {
  const stored = await chrome.storage.local.get(["options", "uiLang"]);
  return makeSettingsFile(stored.options, stored.uiLang || "", await getUiState());
}

// initSettingsTransfer binds the three buttons. `onApplied()` runs after storage holds new settings, so the
// page can redraw in the new language, size and theme; `groups` is the group controller of ui-groups.js.
export function initSettingsTransfer({ onApplied, groups }) {
  const input = document.getElementById("import-file");

  // Writes the settings of `file` (a validated settings file) over the stored ones. The remembered viewport
  // of each page is the device's and stays; the page and groups come from the file.
  async function store(file) {
    const state = await getUiState();
    await chrome.storage.local.set({
      options: file.options,
      uiLang: file.uiLang,
      uiState: { ...state, page: file.ui.page, groups: file.ui.groups },
    });
    groups.apply(file.ui.groups);
    await onApplied();
  }

  document.getElementById("export-settings").addEventListener("click", async () => {
    try {
      const file = await currentSettings();
      saveTextFile("doc-html-translate-settings.json", JSON.stringify(file, null, 2) + "\n", "application/json");
      showStatus(msg("optSettingsExported", "Settings file downloaded."));
    } catch (error) { showStatus(error.message, true); }
  });

  document.getElementById("import-settings").addEventListener("click", () => input.click());
  input.addEventListener("change", async () => {
    const picked = input.files?.[0];
    input.value = "";
    if (!picked) return;
    try {
      if (picked.size > MAX_FILE_BYTES) throw new Error("Settings file is too large");
      let parsed;
      try { parsed = JSON.parse(await picked.text()); }
      catch { throw new Error("Invalid JSON in settings file"); }
      const next = validateSettingsFile(parsed);
      const changed = meaningful(changedSettings(await currentSettings(), next));
      if (!changed.length) {
        // Nothing the reader would notice differs, so there is nothing to confirm; the remembered page and
        // group states still follow the file, which keeps an export and an import a true round trip.
        await store(next);
        showStatus(msg("optSettingsImported", "Imported. Changed: {1}", msg("optSettingsNone", "none")));
        return;
      }
      const list = changed.join(", ");
      const yes = await confirmDialog({
        message: msg("optSettingsImportConfirm", "Replace your current settings with the ones in {1}? This changes: {2}.", picked.name, list),
        confirmLabel: msg("optSettingsImport", "Import settings"),
        confirmGlyph: "action.import",
        danger: true,
      });
      if (!yes) return;
      await store(next);
      showStatus(msg("optSettingsImported", "Imported. Changed: {1}", list));
    } catch (error) { showStatus(error.message, true); }
  });

  document.getElementById("reset-settings").addEventListener("click", async () => {
    try {
      const reset = makeSettingsFile(DEFAULT_OPTIONS, "", DEFAULT_UI_STATE);
      // Resetting settings that already are the defaults is an empty operation: no confirmation for it.
      if (!meaningful(changedSettings(await currentSettings(), reset)).length) {
        showStatus(msg("optSettingsResetDone", "Settings reset to defaults."));
        return;
      }
      const yes = await confirmDialog({
        message: msg("optSettingsResetConfirm", "Reset all extension settings to their defaults?"),
        confirmLabel: msg("optSettingsReset", "Reset settings"),
        confirmGlyph: "action.reset",
        danger: true,
      });
      if (!yes) return;
      await store(reset);
      showStatus(msg("optSettingsResetDone", "Settings reset to defaults."));
    } catch (error) { showStatus(error.message, true); }
  });
}
