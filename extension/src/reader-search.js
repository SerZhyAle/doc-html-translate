// Search stays in the viewer tab. It walks rendered document text, including OCR plates.
// Lowering can change length (Turkic dotted I lowers to two code units under most locales),
// so every lowered code unit records the offset of the character it came from and matches
// report offsets in the ORIGINAL string - a highlight must land on what was matched.
export function occurrences(text, query, lang) {
  const fold = (s) => { try { return s.toLocaleLowerCase(lang); } catch { return s.toLowerCase(); } };
  const needle = fold(query);
  const found = [];
  if (!needle) return found;
  const map = [];
  let lowered = "", i = 0;
  for (const ch of text) {
    const f = fold(ch);
    for (let k = 0; k < f.length; k++) map.push(i);
    lowered += f;
    i += ch.length;
  }
  for (let from = 0, at; (at = lowered.indexOf(needle, from)) !== -1; from = at + Math.max(needle.length, 1)) found.push(map[at]);
  return found;
}

// isSearchChord tells whether a keydown is INPUT-PARITY's `search` binding: Ctrl+F, or Cmd+F on
// macOS, with no Alt and no Shift. The key decides on Latin layouts; on a layout whose F key types
// another letter (Cyrillic, Greek) the physical key does, as the browser's own find does. The
// test is the desktop reader's (internal/htmlgen/search.go) word for word, so both editions take
// the same chords.
export function isSearchChord(event, mac) {
  if (event.altKey || event.shiftKey || (mac ? !event.metaKey || event.ctrlKey : !event.ctrlKey || event.metaKey)) return false;
  return event.key === "f" || event.key === "F" || (event.code === "KeyF" && !/^[a-z]$/i.test(event.key));
}

function isMacPlatform() {
  return /Mac|iP(hone|ad|od)/.test(globalThis.navigator?.platform || "");
}

export function setupReaderSearch({ root, beforeWholeBook, scopeLabel, translate }) {
  const $ = (id) => document.getElementById(id);
  const panel = $("search-panel"), button = $("btn-search"), input = $("search-input");
  const results = $("search-results"), status = $("search-status"), scope = $("search-scope");
  let marks = [], generation = 0;
  const tr = (key, fallback, ...args) => translate(key, fallback, ...args);

  function clear() {
    for (const mark of marks) if (mark.isConnected) mark.replaceWith(document.createTextNode(mark.textContent));
    marks = [];
    root.normalize();
  }
  function currentSection() {
    const sections = [...root.querySelectorAll("section[data-page]")];
    if (!sections.length) return root;
    let current = sections[0];
    for (const section of sections) { if (section.getBoundingClientRect().top <= 120) current = section; else break; }
    return current;
  }
  function nodes(target) {
    const walker = document.createTreeWalker(target, NodeFilter.SHOW_TEXT);
    const out = [];
    for (let node; (node = walker.nextNode());) {
      if (!node.nodeValue.trim() || node.parentElement.closest("script,style,template,noscript,svg,button,select,[hidden],[aria-hidden='true']")) continue;
      out.push(node);
    }
    return out;
  }
  function select(mark) {
    for (const m of marks) m.classList.remove("reader-search-current");
    mark.classList.add("reader-search-current");
    mark.scrollIntoView({ block: "center" });
    mark.tabIndex = -1;
    mark.focus({ preventScroll: true });
  }
  async function search() {
    const id = ++generation, query = input.value.trim();
    clear(); results.replaceChildren();
    if (!query) { status.textContent = tr("vSearchPrompt", "Enter text to search"); return; }
    status.textContent = tr("vSearchWorking", "Searching..");
    if (scope.value === "book") await beforeWholeBook();
    if (id !== generation) return;
    const matches = [], lang = document.documentElement.lang || undefined;
    for (const node of nodes(scope.value === "page" ? currentSection() : root)) {
      const source = node.nodeValue, at = occurrences(source, query, lang), made = [];
      for (const offset of at) matches.push({
        context: source.slice(Math.max(0, offset - 35), Math.min(source.length, offset + query.length + 45)).trim(),
        plate: !!node.parentElement.closest(".ocr-plate"),
        mark: null,
      });
      for (let i = at.length - 1; i >= 0; i--) {
        const range = document.createRange();
        range.setStart(node, at[i]); range.setEnd(node, at[i] + query.length);
        const mark = document.createElement("mark"); mark.className = "reader-search-hit";
        range.surroundContents(mark); made.unshift(mark);
      }
      marks.push(...made);
    }
    matches.forEach((match, i) => { match.mark = marks[i]; });
    const label = scopeLabel(scope.value);
    const onlyPlates = matches.length > 0 && matches.every((m) => m.plate);
    status.textContent = tr("vSearchCount", "Matches: {1} - {2}", matches.length, label)
      + (onlyPlates ? ` (${tr("vSearchPlates", "OCR text plates")})` : "")
      + (!matches.length ? ` (${tr("vSearchNoText", "Scanned pages need OCR text plates")})` : "");
    for (const match of matches.slice(0, 500)) {
      const li = document.createElement("li"), result = document.createElement("button");
      result.type = "button"; result.textContent = match.context;
      result.addEventListener("click", () => select(match.mark));
      li.append(result); results.append(li);
    }
  }
  let timer;
  input.addEventListener("input", () => { clearTimeout(timer); timer = setTimeout(search, 180); });
  scope.addEventListener("change", search);
  function open() {
    const wasHidden = panel.hidden;
    panel.hidden = false; button.setAttribute("aria-expanded", "true");
    input.focus(); input.select();
    if (wasHidden && input.value.trim()) search();
  }
  button.addEventListener("click", () => {
    if (panel.hidden) open();
    else { panel.hidden = true; button.setAttribute("aria-expanded", "false"); clear(); }
  });
  function close() { panel.hidden = true; button.setAttribute("aria-expanded", "false"); clear(); button.focus(); }
  $("search-close").addEventListener("click", close);
  const mac = isMacPlatform();
  document.addEventListener("keydown", (event) => {
    if (event.key === "Escape" && !panel.hidden) { close(); return; }
    // A second Ctrl+F from inside the field is left to the browser, so its own find bar (which
    // also searches the chrome) stays one chord away.
    if (!isSearchChord(event, mac) || event.target === input) return;
    event.preventDefault();
    open();
  });
  return { reset() { ++generation; clearTimeout(timer); clear(); input.value = ""; results.replaceChildren(); status.textContent = ""; panel.hidden = true; button.setAttribute("aria-expanded", "false"); } };
}
