// The GUI dictionary as data. cmd/doc-html-ui/i18n.js is the one string source of the window and of its help
// (CONTEXT-HELP rule 13): a renamed button is renamed in its help by the same edit, because the help reads
// the dictionary instead of copying words from it.
import fs from 'node:fs';
import path from 'node:path';
import vm from 'node:vm';
import { REPO } from './app.mjs';

export function loadDictionary() {
  const src = fs.readFileSync(path.join(REPO, 'cmd', 'doc-html-ui', 'i18n.js'), 'utf8') + '\n;globalThis.__dict = I18N;';
  const ctx = {};
  vm.createContext(ctx);
  vm.runInContext(src, ctx);
  return ctx.__dict;
}

// word looks a key up in a locale and falls back to English, the way the window does.
export function word(dict, locale, key) {
  const v = (dict[locale] && dict[locale][key] != null) ? dict[locale][key] : dict.en[key];
  if (v == null) throw new Error(`dictionary key ${key} is not in i18n.js`);
  return v;
}
