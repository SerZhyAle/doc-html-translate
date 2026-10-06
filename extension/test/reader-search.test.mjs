import test from 'node:test';
import assert from 'node:assert/strict';
import { parseHTML } from 'linkedom';
import { occurrences, isSearchChord, setupReaderSearch } from '../src/reader-search.js';

test('search matches non-Latin and RTL text without changing its content', () => {
  assert.deepEqual(occurrences('مرحبًا بالعالم مرحبًا', 'مرحبًا', 'ar'), [0, 15]);
  assert.deepEqual(occurrences('東京と東京', '東京', 'ja'), [0, 3]);
});

test('search uses the document locale for case', () => {
  assert.deepEqual(occurrences('I ı İ i', 'ı', 'tr'), [0, 2]);
  assert.deepEqual(occurrences('I ı İ i', 'i', 'tr'), [4, 6]);
});

test('match offsets stay in the original string when lowering changes length', () => {
  // İ lowers to two code units under most locales, so the lowered haystack is longer
  // than its source; every reported offset must still point into the ORIGINAL string,
  // or the highlight would land one character short of the match.
  assert.deepEqual(occurrences('İstanbul izmir', 'izmir', 'en'), [9]);
});

// INPUT-PARITY `search`: Ctrl+F, or Cmd+F on macOS, with no Alt and no Shift.
test('the search chord is Ctrl+F, or Cmd+F on macOS, and nothing near it', () => {
  const key = (props) => ({ key: 'f', code: 'KeyF', ctrlKey: false, metaKey: false, altKey: false, shiftKey: false, ...props });
  assert.equal(isSearchChord(key({ ctrlKey: true }), false), true);
  assert.equal(isSearchChord(key({ ctrlKey: true, key: 'F' }), false), true, 'Caps Lock does not change the chord');
  assert.equal(isSearchChord(key({ ctrlKey: true, shiftKey: true }), false), false);
  assert.equal(isSearchChord(key({ ctrlKey: true, altKey: true }), false), false, 'AltGr arrives as Ctrl+Alt');
  assert.equal(isSearchChord(key({ metaKey: true }), false), false, 'the Windows key belongs to the system');
  assert.equal(isSearchChord(key({ metaKey: true }), true), true);
  assert.equal(isSearchChord(key({ ctrlKey: true }), true), false, 'on macOS the chord is Cmd+F');
  assert.equal(isSearchChord(key({ ctrlKey: true, key: 'а' }), false), true, 'a Cyrillic layout is matched by the physical key');
  assert.equal(isSearchChord(key({ ctrlKey: true, key: 'u' }), false), false, 'a Latin layout is matched by the letter it types');
  assert.equal(isSearchChord(key({ ctrlKey: true, key: 'g', code: 'KeyG' }), false), false);
});

// mountSearch renders the viewer's search markup and wires it. linkedom has no focus model, so
// focus() and select() are recorded by hand and document.activeElement follows focus().
function mountSearch() {
  const { document, window } = parseHTML(`<!doctype html><html><body>
    <button id="btn-search" type="button" aria-expanded="false">Search</button>
    <section id="search-panel" hidden>
      <input id="search-input" type="search">
      <select id="search-scope"><option value="page">This page</option></select>
      <button id="search-close" type="button">Close</button>
      <p id="search-status"></p><ol id="search-results"></ol>
    </section>
    <button id="elsewhere" type="button">Elsewhere</button>
    <main id="content"><p>Plain prose.</p></main></body></html>`);
  globalThis.document = document;
  let active = document.body;
  Object.defineProperty(document, 'activeElement', { configurable: true, get: () => active });
  const input = document.getElementById('search-input');
  const state = { selected: 0 };
  for (const el of [input, document.getElementById('btn-search'), document.getElementById('elsewhere')]) {
    el.focus = () => { active = el; };
  }
  input.select = () => { state.selected++; };
  setupReaderSearch({
    root: document.getElementById('content'),
    beforeWholeBook: async () => {},
    scopeLabel: () => 'this page',
    translate: (k, fallback) => fallback,
  });
  // Node reports the host platform as the browser would ("MacIntel" on macOS), so the chord
  // under test is the one the module expects here.
  const mac = /Mac|iP(hone|ad|od)/.test(globalThis.navigator?.platform || '');
  const press = (target, props) => {
    const event = new window.Event('keydown', { bubbles: true, cancelable: true });
    Object.assign(event, { code: '', ctrlKey: false, metaKey: false, altKey: false, shiftKey: false }, props);
    target.dispatchEvent(event);
    return event;
  };
  const chord = mac ? { key: 'f', code: 'KeyF', metaKey: true } : { key: 'f', code: 'KeyF', ctrlKey: true };
  return { document, input, state, press, chord, focus: (el) => el.focus() };
}

test('Ctrl+F opens the reader search with the field focused and its text selected', () => {
  const { document, input, state, press, chord, focus } = mountSearch();
  const panel = document.getElementById('search-panel');
  focus(document.getElementById('elsewhere'));
  const event = press(document.getElementById('elsewhere'), chord);
  assert.equal(event.defaultPrevented, true, 'the chord is taken from the browser');
  assert.equal(panel.hidden, false);
  assert.equal(document.getElementById('btn-search').getAttribute('aria-expanded'), 'true');
  assert.equal(document.activeElement, input);
  assert.equal(state.selected, 1);

  // With the panel already open but focus elsewhere, the chord brings the focus back.
  focus(document.getElementById('elsewhere'));
  assert.equal(press(document.getElementById('elsewhere'), chord).defaultPrevented, true);
  assert.equal(document.activeElement, input);
  assert.equal(state.selected, 2);

  // Esc still closes the panel.
  press(input, { key: 'Escape' });
  assert.equal(panel.hidden, true);
});

test('Ctrl+F from inside the search field is left to the browser', () => {
  const { document, input, state, press, chord, focus } = mountSearch();
  focus(document.getElementById('elsewhere'));
  press(document.getElementById('elsewhere'), chord);
  assert.equal(document.activeElement, input);
  const selectedBefore = state.selected;
  const again = press(input, chord);
  assert.equal(again.defaultPrevented, false, "the browser's own find bar opens");
  assert.equal(state.selected, selectedBefore);
  assert.equal(document.getElementById('search-panel').hidden, false);
});
