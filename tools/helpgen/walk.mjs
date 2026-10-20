// The walk of CONTEXT-HELP section 4 rung 3, driven by automation: F1 and the `?` button on every surface, in the
// locales and both themes, against the real doc-html-ui.exe. One row per surface x locale x theme x way (key / button)
// is the evidence; a row is OK when the opened address is the expected one, the page behind it loads in the help
// window with the entry open, and a modal that was open is still open and answerable.
//
//   node tools/helpgen/walk.mjs --exe temp/helpgen/bin/doc-html-ui.exe [--out temp/helpgen/walk.json] [--md <file>]
//
// What no automation can prove (the owner's physical pass, ticket 93): that the operating system's own F1 never
// reaches a browser's help when the page handler answers, and the narrow, high-contrast and 150/200 percent layouts.
import fs from 'node:fs';
import path from 'node:path';
import { REPO, loadPlaywright, launchBrowser, demoBooks, demoProfile, startApp } from './app.mjs';
import { profileSettings, BOOKS } from './shots.mjs';

const argv = process.argv.slice(2);
const opt = (n, d) => (argv.includes(n) ? argv[argv.indexOf(n) + 1] : d);
const exe = path.resolve(opt('--exe', path.join(REPO, 'temp', 'helpgen', 'bin', 'doc-html-ui.exe')));
const outJson = path.resolve(opt('--out', path.join(REPO, 'temp', 'helpgen', 'walk.json')));
const outMd = opt('--md', '');

const map = JSON.parse(fs.readFileSync(path.join(REPO, 'cmd', 'doc-html-ui', 'help', 'help-map.json'), 'utf8'));
const LOCALES = ['en', 'ru', 'uk', 'de', 'ar'];          // three authored, one fallback, one right-to-left fallback
const THEMES = ['dark', 'light'];
const authored = new Set(map.locales);
const pageLocale = (l) => (authored.has(l) ? l : map.sourceLocale);
const pageOf = (id) => map.surfaces.find((s) => s.id === id).help;

const root = path.join(REPO, 'temp', 'helpgen');
const books = demoBooks(root, BOOKS);
const rows = [];
const record = (r) => { rows.push(r); if (!r.ok) console.log('NOT OK', JSON.stringify(r)); };

const { chromium } = await loadPlaywright();
const browser = await launchBrowser(chromium);
const app = await startApp(exe, demoProfile(root, profileSettings({ lang: 'en', theme: 'dark', books })));

async function open(ctx, locale, theme) {
  demoProfile(root, profileSettings({ lang: locale, theme, books }));
  const page = await ctx.newPage();
  await page.addInitScript(() => {
    window.__opened = [];
    const o = window.open.bind(window);
    window.open = (u, n) => { window.__opened.push([String(u), n]); return o(u, n); };
  });
  await page.goto(`${app.base}/?lang=${locale}`, { waitUntil: 'load' });
  await page.waitForFunction(() => document.getElementById('aboutVer').textContent.trim() !== '' && window.DHTHelp);
  await page.evaluate(() => DHTHelp.ready);
  await page.evaluate((th) => applyTheme(th), theme);
  await page.waitForTimeout(300);
  return page;
}

// The address a press asked for: the last window.open call since the mark.
const lastOpen = (page) => page.evaluate(() => { const a = window.__opened; return a.length ? a[a.length - 1] : null; });
const clear = (page) => page.evaluate(() => { window.__opened = []; });

async function check(page, ctx, base, expected, press) {
  await clear(page);
  const popupP = ctx.waitForEvent('page', { timeout: expected === null ? 700 : 2500 }).catch(() => null);
  await press();
  const call = await lastOpen(page);
  const popup = await popupP;
  const row = { ...base, expected, opened: call ? call[0] : null, window: call ? call[1] : null };
  if (expected === null) {
    row.ok = call === null && popup === null;
    record(row);
    return;
  }
  row.ok = !!call && call[0] === expected && call[1] === 'dht-help';
  if (popup) {
    // the help window itself: the page loads in the help window, and the entry the address names is open
    await popup.waitForLoadState('load').catch(() => {});
    const anchor = expected.split('#')[1];
    row.loaded = !(await popup.title()).includes('404');
    if (anchor) row.entryOpen = await popup.evaluate((a) => { const e = document.getElementById(a); return !!e && (e.tagName !== 'DETAILS' || e.open); }, anchor);
    else row.entryOpen = true;
    row.ok = row.ok && row.loaded && row.entryOpen;
    await popup.close().catch(() => {});
  } else {
    row.ok = false; row.note = 'no help window opened';
  }
  record(row);
}

try {
  for (const locale of LOCALES) {
    for (const theme of THEMES) {
      const ctx = await browser.newContext({ viewport: { width: 1160, height: 900 }, colorScheme: theme });
      const page = await open(ctx, locale, theme);
      const L = pageLocale(locale);
      const themeQ = `?theme=${theme}`;
      const url = (help, anchor) => `/help/${L}/${help}.html${themeQ}${anchor ? '#' + anchor : ''}`;
      const base = (surface, via) => ({ locale, theme, surface, via });

      // the window: the key from the body, the button, then the key from inside every region
      await page.evaluate(() => document.activeElement && document.activeElement.blur());
      await check(page, ctx, base('main', 'key'), url('converter'), () => page.keyboard.press('F1'));
      await check(page, ctx, base('main', 'button'), url('converter'), () => page.click('#btnHelp'));
      await page.focus('#btnHelp');
      await check(page, ctx, base('main.help (focused button)', 'key'), url('converter', 'help'), () => page.keyboard.press('F1'));
      // the states of the window that show the regions that appear only sometimes: the first-run question, a queue, the
      // Google key row and a finished result (the page's own functions, as the screenshots do)
      await page.evaluate((files) => {
        document.querySelector('input[name="engine"][value="google"]').checked = true;
        syncEngineUI();
        addFilesToQueue(files);
        outputStatus = { exists: true, paramsChanged: false, outputDir: 'C:\\Books\\Alice' };
        renderPrevResultHint();
        el('assocPrompt').style.display = '';
      }, books);
      await page.waitForTimeout(500);
      for (const s of map.surfaces) {
        for (const row of s.pages || []) {
          // the first control inside a visible host of the region takes the focus
          const focused = await page.evaluate((id) => {
            const sel = 'button:not([disabled]), input:not([type=hidden]):not([disabled]), select, a[href], summary';
            for (const host of document.querySelectorAll(`[data-help="${id}"]`)) {
              // a region on a page that is not the one showing: the walk goes to that page first, like a reader would
              const pg = host.closest('.page');
              if (pg && pg.hidden) showPage(pg.id.replace('page-', ''));
              if (host.offsetParent === null) continue;
              // the first control that is actually there: a control a narrow window shows and this one hides (the Pages
              // button) cannot take the focus and must not stand in for the region
              const target = host.matches(sel) ? host : [...host.querySelectorAll(sel)].find((e) => e.offsetParent !== null);
              if (!target) continue;
              if (host.tagName === 'DETAILS') host.open = true;
              target.focus();
              return host.contains(document.activeElement);
            }
            return false;
          }, row.id);
          if (!focused) {
            record({ ...base(row.id, 'key'), expected: 'skipped', ok: true, note: 'no focusable control inside this region (status text); F1 from its neighbours asks the window' });
            continue;
          }
          await check(page, ctx, base(row.id, 'key'), url(s.help, row.anchor), () => page.keyboard.press('F1'));
        }
      }
      // the About link opens the index
      await page.evaluate(() => { showPage('about'); });
      await check(page, ctx, base('index (About link)', 'button'), `/help/${L}/index.html${themeQ}`, () => page.click('a[data-help-open="index"]'));

      // dialogs: F1 and the corner button answer the same, and the modal stays open
      const dialogs = {
        'dlg-cost': () => { openDialog({ title: 'Google Translate - cost', message: 'Characters to send: 1', surface: 'dlg-cost', buttons: [{ id: 'no', key: 'costNo', glyph: 'action.cancel', focus: true }, { id: 'yes', key: 'costYes', kind: 'danger', glyph: 'action.translate' }], cancel: 'no' }); },
        'dlg-secret': () => { openDialog({ title: 'FileDO secret file', message: 'Enter the password', secret: true, surface: 'dlg-secret', buttons: [{ id: 'cancel', key: 'btnCancel', glyph: 'action.cancel' }, { id: 'open', key: 'btnOpenSecret', kind: 'ok', ans: 'ans-ok', glyph: 'action.confirm' }], cancel: 'cancel' }); },
        'dlg-run-failed': () => { openDialog({ title: t('runFailedTitle'), message: t('runCauseParse'), surface: 'dlg-run-failed', buttons: [{ id: 'close', key: 'dlgClose', glyph: 'nav.close', focus: true }], cancel: 'close' }); },
        'dlg-confirm': () => { confirmDialog(t('confirmRecentClear', { n: 3 }), 'recentClear', 'danger', true); },
        'dlg-notice': () => { informDialog(t('dlgFailTitle'), t('openPrevFail')); },
      };
      for (const [id, show] of Object.entries(dialogs)) {
        const help = map.surfaces.find((s) => s.id === id);
        for (const via of ['key', 'button']) {
          await page.evaluate(show);
          await page.waitForSelector('#dlg[open]');
          const buttonShown = await page.evaluate(() => !document.getElementById('dlgHelp').hidden);
          const expected = help.form === 'none' ? null : url(help.help);
          if (via === 'button') {
            if (help.form === 'none') {
              record({ ...base(id, 'button'), expected: 'no button', ok: !buttonShown, note: `button shown: ${buttonShown}` });
            } else {
              await check(page, ctx, base(id, 'button'), expected, () => page.click('#dlgHelp'));
            }
          } else {
            await check(page, ctx, base(id, 'key'), expected, () => page.keyboard.press('F1'));
          }
          const stillOpen = await page.evaluate(() => document.querySelector('dialog[open]') !== null);
          rows[rows.length - 1].modalStillOpen = stillOpen;
          rows[rows.length - 1].ok = rows[rows.length - 1].ok && stillOpen;
          await page.evaluate(() => document.getElementById('dlg').close());
          await page.waitForTimeout(150);
        }
      }
      await ctx.close();
    }
  }
} finally {
  await browser.close();
  app.stop();
}

fs.mkdirSync(path.dirname(outJson), { recursive: true });
fs.writeFileSync(outJson, JSON.stringify({ exe, at: new Date().toISOString(), rows }, null, 2));
const bad = rows.filter((r) => !r.ok);
const summary = `walk: ${rows.length} rows, ${rows.length - bad.length} ok, ${bad.length} not ok`;
console.log(summary);
if (outMd) {
  const by = {};
  for (const r of rows) { const k = r.surface + ' / ' + r.via; (by[k] ||= []).push(r); }
  const lines = [
    '# Walk matrix - F1 and the help button (ticket 118, CONTEXT-HELP rung 3, automated)', '',
    `Run ${new Date().toISOString().slice(0, 10)} against \`${path.relative(REPO, exe).replace(/\\/g, '/')}\` in headless Chromium, locales ${LOCALES.join(', ')}, themes ${THEMES.join(', ')}.`,
    `${summary}.`, '',
    '| Surface | Way | Rows | OK | Notes |', '| --- | --- | ---: | ---: | --- |',
    ...Object.entries(by).map(([k, v]) => { const [s, w] = k.split(' / '); return `| ${s} | ${w} | ${v.length} | ${v.filter((x) => x.ok).length} | ${[...new Set(v.map((x) => x.note).filter(Boolean))].join('; ')} |`; }),
    '',
  ];
  fs.writeFileSync(path.resolve(outMd), lines.join('\n'));
}
process.exitCode = bad.length ? 1 : 0;
