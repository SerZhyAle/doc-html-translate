// ui-dialog.js - the product's own confirmation dialog (BUTTONS 2.0 section 4 rules 4 and 5), used instead
// of the browser's confirm() so the answers read in the interface language, in the shared component, with
// the safe answer first and holding the focus.
//
// The answers read Cancel first, then the acting one; Cancel is neutral and focused when the action is
// irreversible, Enter from the dialog selects it, Escape takes the same no-action path, and focus returns
// to the control that opened the dialog. The acting button of an irreversible confirmation alone takes the
// danger face.

import { t } from "./i18n.js";
import { glyph } from "./glyphs.js";

let open = null;

function answer(value, glyphId, caption, className = "") {
  const btn = document.createElement("button");
  btn.type = "submit";
  btn.value = value;
  if (className) btn.className = className;
  const text = document.createElement("span");
  text.textContent = caption;
  btn.append(glyph(glyphId), text);
  return btn;
}

// confirmDialog resolves true only for the acting answer. A second call while one is open returns the
// open dialog's promise: a double click must not stack two questions about one action.
export function confirmDialog({ title, message, confirmLabel, confirmGlyph, danger = false }) {
  if (open) return open;
  open = new Promise((resolve) => {
    const invoker = document.activeElement;
    const dlg = document.createElement("dialog");
    dlg.className = "dlg";
    dlg.setAttribute("aria-labelledby", "dlgTitle");
    dlg.setAttribute("aria-describedby", "dlgMessage");
    const form = document.createElement("form");
    form.method = "dialog";
    const heading = document.createElement("h2");
    heading.id = "dlgTitle";
    heading.textContent = title || t("dlgConfirmTitle", "Please confirm");
    const body = document.createElement("p");
    body.id = "dlgMessage";
    body.textContent = message;
    const actions = document.createElement("div");
    actions.className = "dlg-actions";
    const cancel = answer("no", "action.cancel", t("dlgCancel", "Cancel"));
    const act = answer("yes", confirmGlyph, confirmLabel, danger ? "danger" : "");
    actions.append(cancel, act);
    form.append(heading, body, actions);
    dlg.append(form);
    dlg.addEventListener("close", () => {
      const yes = dlg.returnValue === "yes";
      dlg.remove();
      open = null;
      if (invoker && invoker.isConnected) invoker.focus();
      resolve(yes);
    }, { once: true });
    document.body.append(dlg);
    dlg.showModal();
    cancel.focus();
  });
  return open;
}
