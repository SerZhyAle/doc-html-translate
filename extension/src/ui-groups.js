// ui-groups.js - remembers which collapsible groups are open (COLLAPSIBLE-GROUP 1.1 section 4).
//
// A group is a <details class="group" id="..."> with a stable id; its open state is stored under that id
// in the interface's remembered state, restored when a page opens and saved when the reader changes it.
// A restored state is not an edit, and closing a group never touches the controls inside it. The markup's
// own `open` attribute is the group's default, which the remembered state overrides.

import { getUiState, setUiState } from "./options-store.js";

// bindGroups restores and then tracks the groups under root. It returns `apply(states)`, which sets every
// group to the state in `states` (or to its default where the map is silent) without counting that as a
// choice - an import or a reset keeps the open page coherent with what storage now says.
export async function bindGroups(root = document) {
  const groups = [...root.querySelectorAll("details.group[id]")];
  const defaults = new Map(groups.map((d) => [d.id, d.open]));
  // What each group shows right now. The toggle events a restore queues find their state already known
  // and save nothing, so opening a page never writes storage.
  const known = new Map(defaults);
  const apply = (states) => {
    for (const d of groups) {
      const want = typeof states?.[d.id] === "boolean" ? states[d.id] : defaults.get(d.id);
      known.set(d.id, want);
      d.open = want;
    }
  };
  apply((await getUiState()).groups);
  for (const d of groups) {
    d.addEventListener("toggle", () => {
      // A group closed while one of its controls holds the focus hands the focus to its header before
      // the controls leave the traversal, so the keyboard is never stranded in a hidden body.
      if (!d.open) {
        const focused = document.activeElement;
        if (focused && d.contains(focused)) d.querySelector("summary")?.focus();
      }
      if (known.get(d.id) === d.open) return;
      known.set(d.id, d.open);
      setUiState({ groups: { [d.id]: d.open } }).catch(() => { /* memory only: the group still works */ });
    });
  }
  return { apply };
}
