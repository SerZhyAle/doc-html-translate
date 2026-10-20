// SETTINGS-NAVIGATION 2.0 on the options page, and the product dialog of BUTTONS 2.0 section 4 (ticket 119, slice X1).
// navigationChecks: the rail, keyboard, nothing happening while navigating, memory of the page, the groups and the
// viewport, the narrow drawer. dialogChecks: the reset and the overwriting import. timingChecks: activation to frame.
import path from 'node:path';
import { REPO, nearestRankP95, writeJSON } from './lib.mjs';
import { pathToFileURL } from 'node:url';
import { approx } from './extension-rules.mjs';

const { DEFAULT_OPTIONS } = await import(pathToFileURL(path.join(REPO, 'extension', 'src', 'defaults.js')).href);
const { OPTIONS_PAGES } = await import(pathToFileURL(path.join(REPO, 'extension', 'src', 'ui-settings.js')).href);
const { makeSettingsFile } = await import(pathToFileURL(path.join(REPO, 'extension', 'src', 'settings-transfer.js')).href);

const f1 = (n) => (Math.round(n * 10) / 10).toString();
const visiblePage = (page) => page.evaluate(() => [...document.querySelectorAll('.page')].filter((p) => !p.hidden).map((p) => p.id));
const BASE = { name: 'options', url: '/src/options.html', size: 'ordinary', input: 'pointer' };

export async function navigationChecks(browser, base, report, { openPage }) {
  // ---- the wide window: rail, selection, keyboard, nothing happens on navigation ----
  const wide = await openPage(browser, base, { ...BASE, w: 1280, h: 900 });
  const { page } = wide;
  try {
    const rail = await page.evaluate(() => [...document.querySelectorAll('.rail-item')].map((b) => ({ id: b.dataset.page, glyph: !!b.querySelector('svg'), name: b.textContent.trim() })));
    report.check('rail.order', rail.map((x) => x.id).join() === OPTIONS_PAGES.join(), `destinations ${rail.map((x) => x.id).join(', ')} (General first, About last, ${rail.length} of at most 7)`);
    report.check('rail.glyph-and-name', rail.every((x) => x.glyph && x.name.length > 0), 'every destination has a glyph and a short name');
    const sel = await page.evaluate(() => { const b = document.getElementById('tab-general'); const cs = getComputedStyle(b); return { aria: b.getAttribute('aria-selected'), weight: cs.fontWeight, shadow: cs.boxShadow, stops: [...document.querySelectorAll('.rail-item')].filter((x) => x.tabIndex === 0).length, orientation: document.getElementById('railList').getAttribute('aria-orientation') }; });
    report.check('rail.selected-cue', sel.aria === 'true' && Number(sel.weight) >= 600 && sel.shadow !== 'none', `aria-selected, weight ${sel.weight}, inline-start bar ${sel.shadow !== 'none'}`);
    report.check('rail.one-tab-stop', sel.stops === 1, `${sel.stops} rail tab stop`);
    report.check('rail.vertical-list-semantics', sel.orientation === 'vertical', `tablist orientation ${sel.orientation}`);
    await page.focus('#tab-general');
    await page.keyboard.press('ArrowDown');
    report.check('rail.arrow-selects', JSON.stringify(await visiblePage(page)) === '["page-ocr"]', 'ArrowDown moved to and showed the next page');
    await page.keyboard.press('End');
    report.check('rail.end-selects-about', JSON.stringify(await visiblePage(page)) === '["page-about"]', 'End jumped to About');
    await page.keyboard.press('Home');
    report.check('rail.home-selects-first', JSON.stringify(await visiblePage(page)) === '["page-general"]', 'Home jumped to the first page');
    report.check('rail.arrow-keeps-focus-on-rail', await page.evaluate(() => document.activeElement.id === 'tab-general'), 'focus stayed on the rail');

    // an edit survives; navigation commits nothing, sends nothing and opens nothing
    await page.selectOption('#theme', 'sepia');
    await page.waitForTimeout(80);
    const win0 = await page.evaluate(() => [window.outerWidth, window.outerHeight, window.__uic.rect(document.getElementById('rail')).x]);
    await page.evaluate(() => { window.__log.sets.length = 0; window.__log.sends.length = 0; window.__log.created.length = 0; });
    for (const id of ['ocr', 'interface', 'about', 'data', 'sites', 'general']) await page.click('#tab-' + id);
    await page.waitForTimeout(700);
    const after = await page.evaluate(() => ({ theme: document.getElementById('theme').value, stored: window.__stored.options.theme, log: window.__log, win: [window.outerWidth, window.outerHeight, window.__uic.rect(document.getElementById('rail')).x], focus: document.activeElement.id }));
    report.check('nav.outer-window-and-rail-unchanged', win0.join() === after.win.join(), `outer window and rail position unchanged (${win0.join(' x ')})`);
    report.check('nav.edit-preserved', after.theme === 'sepia' && after.stored === 'sepia', `the edited reading theme is ${after.theme} on the page and ${after.stored} in storage after six page changes`);
    const keys = [...new Set(after.log.sets.flat())];
    report.check('nav.no-commit-and-no-operation', keys.every((k) => k === 'uiState') && after.log.sends.length === 0 && after.log.created.length === 0 && after.log.opened === 0, `after six page changes: stored keys [${keys.join(', ')}] (only the remembered page may be written), ${after.log.sends.length} messages, ${after.log.created.length} tabs, ${after.log.opened} option pages`);
    report.check('nav.writes-are-batched', after.log.sets.length <= 2, `${after.log.sets.length} storage write(s) for six clicks (the memory is saved a moment after the last one)`);
    report.check('nav.focus-stays-on-rail', after.focus === 'tab-general', 'focus stayed on the rail (no jump to an input)');
    report.check('nav.no-remote-request', page.remote.length === 0, page.remote.join(', ') || 'no request left the extension origin');

    // rapid switching ends on the last choice with exactly one page visible
    for (const id of ['about', 'ocr', 'data', 'sites', 'interface', 'about', 'general', 'ocr']) await page.evaluate((i) => document.getElementById('tab-' + i).click(), id);
    report.check('nav.rapid-switching-ends-on-last', JSON.stringify(await visiblePage(page)) === '["page-ocr"]', 'eight clicks without a pause left one page, the last one chosen');

    // the page, the groups and the viewport are remembered by stable ids
    await page.click('#tab-about');
    await page.evaluate(() => { document.getElementById('aboutSupport').open = false; document.getElementById('aboutRelated').open = true; });
    await page.click('#tab-interface');
    await page.waitForTimeout(700);
    const mem = await page.evaluate(() => window.__stored.uiState);
    report.check('nav.page-remembered', mem && mem.page === 'interface', `stored page ${mem && mem.page}`);
    report.check('nav.group-state-remembered', mem && mem.groups && mem.groups.aboutSupport === false && mem.groups.aboutRelated === true, `groups ${JSON.stringify(mem && mem.groups)}`);
    const carried = await page.evaluate(() => JSON.parse(JSON.stringify(window.__stored)));
    const again = await openPage(browser, base, { ...BASE, stored: carried, w: 1280, h: 900 });
    try {
      report.check('nav.page-restored-before-first-frame', JSON.stringify(await visiblePage(again.page)) === '["page-interface"]', 'a new opening shows the remembered page');
      const g = await again.page.evaluate(() => ({ support: document.getElementById('aboutSupport').open, related: document.getElementById('aboutRelated').open, resources: document.getElementById('aboutResources').open }));
      report.check('nav.group-state-restored', g.support === false && g.related === true && g.resources === false, `support ${g.support}, related ${g.related}, resources ${g.resources}`);
      report.check('nav.restoring-writes-nothing', (await again.page.evaluate(() => window.__log.sets.length)) === 0, 'restoring the page and the groups was not treated as an edit');
    } finally { await again.close(); }
  } finally { await wide.close(); }

  // retired or unknown destinations fall back to the first
  const retired = await openPage(browser, base, { ...BASE, stored: { options: { ...DEFAULT_OPTIONS }, uiState: { page: 'retired-page', groups: { gone: true } } } });
  try { report.check('nav.unknown-page-falls-back', JSON.stringify(await visiblePage(retired.page)) === '["page-general"]', 'a retired destination opens General'); } finally { await retired.close(); }

  // the viewport of a long page is restored after layout
  const tall = await openPage(browser, base, { ...BASE, w: 1100, h: 380, uiState: { page: 'about', groups: { aboutResources: true, aboutRelated: true, aboutSupport: true } } });
  try {
    const before = await tall.page.evaluate(() => { const p = document.getElementById('pane'); p.scrollTop = Math.min(300, p.scrollHeight - p.clientHeight); return { top: p.scrollTop, room: p.scrollHeight - p.clientHeight }; });
    await tall.page.waitForTimeout(600);
    await tall.page.click('#tab-general');
    await tall.page.click('#tab-about');
    const back = await tall.page.evaluate(() => document.getElementById('pane').scrollTop);
    report.check('nav.viewport-restored-on-return', before.room > 50 && approx(back, before.top, 10), `pane scrolled ${f1(before.top)} px of ${f1(before.room)}; after leaving and returning ${f1(back)} px`);
    await tall.page.waitForTimeout(700);
    const carried2 = await tall.page.evaluate(() => JSON.parse(JSON.stringify(window.__stored)));
    const reopened = await openPage(browser, base, { ...BASE, stored: carried2, w: 1100, h: 380 });
    try {
      const top = await reopened.page.evaluate(() => document.getElementById('pane').scrollTop);
      report.check('nav.viewport-restored-on-reopen', approx(top, before.top, 12), `a new opening scrolled the pane to ${f1(top)} px (was ${f1(before.top)})`);
    } finally { await reopened.close(); }
  } finally { await tall.close(); }

  // ---- the narrow window: the same menu as a start-side drawer ----
  const narrow = await openPage(browser, base, { ...BASE, w: 700, h: 800 });
  try {
    const np = narrow.page;
    const d = await np.evaluate(() => ({ btn: getComputedStyle(document.getElementById('btnPages')).display, rail: getComputedStyle(document.getElementById('rail')).visibility, scroll: document.documentElement.scrollWidth <= window.innerWidth + 1 }));
    report.check('drawer.pages-button-shows-and-rail-hides', d.btn !== 'none' && d.rail === 'hidden' && d.scroll, `Pages button ${d.btn}, rail ${d.rail}, no horizontal scroll ${d.scroll}`);
    await np.click('#btnPages');
    const o = await np.evaluate(() => ({ open: document.getElementById('rail').classList.contains('open'), exp: document.getElementById('btnPages').getAttribute('aria-expanded'), focus: document.activeElement.id, rail: getComputedStyle(document.getElementById('rail')).visibility }));
    report.check('drawer.opens-with-focus-on-selected-page', o.open && o.exp === 'true' && o.focus === 'tab-general' && o.rail === 'visible', `open ${o.open}, aria-expanded ${o.exp}, focus ${o.focus}`);
    await np.keyboard.press('Escape');
    const e = await np.evaluate(() => ({ open: document.getElementById('rail').classList.contains('open'), focus: document.activeElement.id }));
    report.check('drawer.escape-closes-and-returns-focus', !e.open && e.focus === 'btnPages', `open ${e.open}, focus ${e.focus}`);
    await np.click('#btnPages');
    await np.click('#tab-ocr');
    const c = await np.evaluate(() => ({ open: document.getElementById('rail').classList.contains('open'), page: [...document.querySelectorAll('.page')].filter((p) => !p.hidden).map((p) => p.id).join() }));
    report.check('drawer.choosing-a-page-closes-it', !c.open && c.page === 'page-ocr', `drawer open ${c.open}, page ${c.page}`);
    await np.click('#btnPages');
    await np.mouse.click(400, 400);
    report.check('drawer.scrim-closes', !(await np.evaluate(() => document.getElementById('rail').classList.contains('open'))), 'a click outside the drawer closed it');
    // a page edit survives opening the drawer
    report.check('drawer.page-unchanged-by-opening', JSON.stringify(await visiblePage(np)) === '["page-ocr"]', 'opening and closing the drawer left the page as it was');
  } finally { await narrow.close(); }
}

// ---- the product dialog: reset and the overwriting import ----------------------------------------------------------
export async function dialogChecks(browser, base, report, { openPage }) {
  const c = { ...BASE, w: 1280, h: 900, uiState: { page: 'data', groups: {} }, options: { theme: 'sepia', uiSize: 'large' } };
  const { page, close } = await openPage(browser, base, c);
  try {
    const dlg = () => page.evaluate(() => {
      const U = window.__uic, d = document.querySelector('dialog[open]');
      if (!d) return null;
      const bs = [...d.querySelectorAll('.dlg-actions button')];
      return {
        order: bs.map((b) => b.value), classes: bs.map((b) => b.className), focus: document.activeElement.value, glyphs: bs.map((b) => !!b.querySelector('svg')),
        contrast: bs.map((b) => U.textContrast(b)), faces: bs.map((b) => U.bgOf(b).map(Math.round).join(',')), message: d.querySelector('p').textContent,
        labelled: !!document.getElementById(d.getAttribute('aria-labelledby')), described: !!document.getElementById(d.getAttribute('aria-describedby')), modal: d.matches(':modal'),
      };
    });
    const stored = () => page.evaluate(() => JSON.parse(JSON.stringify(window.__stored)));
    // ---- reset ----
    await page.click('#reset-settings');
    await page.waitForSelector('dialog[open]');
    const d = await dlg();
    report.check('dialog.reset-order-cancel-first', d.order.join() === 'no,yes', `answers read ${d.order.join(' then ')}`);
    report.check('dialog.reset-acting-answer-is-danger', /danger/.test(d.classes[1]) && !/danger/.test(d.classes[0]), `classes "${d.classes[0]}" / "${d.classes[1]}"`);
    report.check('dialog.reset-safe-default-focus', d.focus === 'no', `focus on ${d.focus}`);
    report.check('dialog.reset-cancel-neutral', d.faces[0] !== d.faces[1], `Cancel face ${d.faces[0]}, acting face ${d.faces[1]}`);
    report.check('dialog.reset-glyph-on-both-answers', d.glyphs.every(Boolean), 'each answer has its glyph before the caption');
    report.check('dialog.reset-caption-contrast', d.contrast.every((x) => x >= 4.5), `captions ${d.contrast.map(f1).join(' / ')}:1`);
    report.check('dialog.reset-modal-and-named', d.modal && d.labelled && d.described, `modal ${d.modal}, labelled ${d.labelled}, described ${d.described}`);
    await page.keyboard.press('Escape');
    const gone = await page.evaluate(() => ({ open: !!document.querySelector('dialog[open]'), focus: document.activeElement.id }));
    report.check('dialog.reset-escape-cancels-and-returns-focus', !gone.open && gone.focus === 'reset-settings' && (await stored()).options.theme === 'sepia', `dialog open ${gone.open}, focus ${gone.focus}, theme still ${(await stored()).options.theme}`);
    await page.click('#reset-settings');
    await page.waitForSelector('dialog[open]');
    await page.keyboard.press('Enter');
    report.check('dialog.reset-enter-takes-the-safe-answer', !(await page.evaluate(() => !!document.querySelector('dialog[open]'))) && (await stored()).options.theme === 'sepia', 'Enter from the dialog selected Cancel; nothing changed');
    await page.click('#reset-settings');
    await page.waitForSelector('dialog[open]');
    await page.evaluate(() => { window.__log.sets.length = 0; });
    await page.click('dialog[open] button[value="yes"]');
    await page.waitForFunction(() => !document.querySelector('dialog[open]'));
    await page.waitForTimeout(200);
    const s = await stored();
    const same = Object.keys(DEFAULT_OPTIONS).every((k) => JSON.stringify(s.options[k]) === JSON.stringify(DEFAULT_OPTIONS[k]));
    report.check('dialog.reset-acting-answer-resets', same && s.uiLang === '', `every option is its default: ${same}`);
    report.check('dialog.reset-applies-without-restart', (await page.evaluate(() => document.documentElement.dataset.size)) === 'ordinary' && (await page.evaluate(() => document.getElementById('theme').value)) === 'light', 'the size and the reading theme went back at once');
    // empty operation: nothing to reset, nothing to confirm
    await page.click('#reset-settings');
    await page.waitForTimeout(150);
    report.check('dialog.empty-reset-asks-nothing', !(await page.evaluate(() => !!document.querySelector('dialog[open]'))), 'resetting settings that already are the defaults opened no dialog');

    // ---- import ----
    const file = makeSettingsFile({ ...DEFAULT_OPTIONS, theme: 'night', uiSize: 'large', chromeTheme: 'dark' }, 'de', { page: 'about', groups: { aboutSupport: false } });
    const send = (obj, name = 'settings.json') => page.setInputFiles('#import-file', { name, mimeType: 'application/json', buffer: Buffer.from(JSON.stringify(obj)) });
    await send(file);
    await page.waitForSelector('dialog[open]');
    const i = await dlg();
    report.check('dialog.import-asks-before-overwriting', i && i.order.join() === 'no,yes' && /danger/.test(i.classes[1]) && i.focus === 'no', `answers ${i && i.order.join(' then ')}, focus ${i && i.focus}`);
    report.check('dialog.import-names-file-and-changes', i && i.message.includes('settings.json') && /theme/.test(i.message), `"${i && i.message}"`);
    await page.keyboard.press('Escape');
    await page.waitForTimeout(100);
    report.check('dialog.import-cancel-keeps-settings', (await stored()).options.theme === 'light', `theme ${(await stored()).options.theme} after Cancel`);
    await send(file);
    await page.waitForSelector('dialog[open]');
    await page.click('dialog[open] button[value="yes"]');
    await page.waitForFunction(() => !document.querySelector('dialog[open]'));
    await page.waitForTimeout(250);
    const imp = await page.evaluate(() => ({ stored: window.__stored, size: document.documentElement.dataset.size, chrome: document.documentElement.dataset.chromeTheme, lang: document.documentElement.lang, groups: document.getElementById('aboutSupport').open }));
    report.check('dialog.import-acting-answer-applies', imp.stored.options.theme === 'night' && imp.stored.options.uiSize === 'large' && imp.stored.uiLang === 'de', `theme ${imp.stored.options.theme}, size ${imp.stored.options.uiSize}, language ${imp.stored.uiLang}`);
    report.check('dialog.import-applies-live', imp.size === 'large' && imp.chrome === 'dark' && imp.lang === 'de' && imp.groups === false, `size ${imp.size}, theme ${imp.chrome}, language ${imp.lang}, Support open ${imp.groups}`);
    report.check('dialog.import-round-trips-page-and-groups', imp.stored.uiState.page === 'about' && imp.stored.uiState.groups.aboutSupport === false, `stored page ${imp.stored.uiState.page}, groups ${JSON.stringify(imp.stored.uiState.groups)}`);
    // an unchanged file and a bad file ask nothing
    await send(makeSettingsFile(imp.stored.options, imp.stored.uiLang, imp.stored.uiState));
    await page.waitForTimeout(200);
    report.check('dialog.import-of-same-settings-asks-nothing', !(await page.evaluate(() => !!document.querySelector('dialog[open]'))), 'a file that changes nothing opened no dialog');
    await send({ format: 'another-extension', version: 2 });
    await page.waitForTimeout(150);
    const bad = await page.evaluate(() => ({ dialog: !!document.querySelector('dialog[open]'), status: document.getElementById('settings-status').textContent, error: document.getElementById('settings-status').classList.contains('error') }));
    report.check('dialog.bad-file-is-refused-inline', !bad.dialog && bad.error && bad.status.length > 0, `status "${bad.status}"`);
    report.check('console.clean/dialogs', page.errors.length === 0, page.errors.slice(0, 3).join(' | ') || 'no script or console error');
  } finally { await close(); }
}

// ---- activation to frame (SETTINGS-NAVIGATION 5 rule 4) ---------------------------------------------------------------
export async function timingChecks(browser, base, report, { openPage, outDir }) {
  const meas = () => {
    window.__switch = (id) => new Promise((resolve) => {
      const t0 = performance.now();
      document.getElementById('tab-' + id).click();
      requestAnimationFrame(() => requestAnimationFrame(() => resolve(performance.now() - t0)));
    });
  };
  const raw = { warm: {}, cold: {} };
  const { page, close } = await openPage(browser, base, { ...BASE, w: 1280, h: 900 });
  try {
    await page.evaluate(meas);
    for (const id of OPTIONS_PAGES) await page.evaluate((i) => window.__switch(i), id);
    for (const id of OPTIONS_PAGES) {
      const samples = [];
      for (let k = 0; k < 20; k++) {
        await page.evaluate((i) => window.__switch(i), OPTIONS_PAGES[(OPTIONS_PAGES.indexOf(id) + 1) % OPTIONS_PAGES.length]);
        samples.push(await page.evaluate((i) => window.__switch(i), id));
      }
      raw.warm[id] = samples;
      const p95 = nearestRankP95(samples), max = Math.max(...samples);
      report.check(`timing.warm/${id}`, p95 < 150 && max < 150, `p95 ${f1(p95)} ms, max ${f1(max)} ms over ${samples.length} warm transitions (budget 150)`);
    }
  } finally { await close(); }
  for (const id of ['about', 'ocr', 'interface']) {
    const samples = [];
    for (let k = 0; k < 20; k++) {
      const s = await openPage(browser, base, { ...BASE, w: 1280, h: 900 });
      try { await s.page.evaluate(meas); samples.push(await s.page.evaluate((i) => window.__switch(i), id)); } finally { await s.close(); }
    }
    raw.cold[id] = samples;
    const p95 = nearestRankP95(samples), max = Math.max(...samples);
    report.check(`timing.first-visit/${id}`, p95 < 500, `p95 ${f1(p95)} ms, max ${f1(max)} ms over ${samples.length} independent first visits (budget 500)`);
  }
  if (outDir) writeJSON(path.join(outDir, 'extension-timings.raw.json'), { method: 'activation (click) to the second animation frame after it, performance.now(); headless Chromium over loopback', raw });
}
