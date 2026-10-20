// Captures of the real window for the help pages (CONTEXT-HELP rule 14): per locale and theme, demo content
// only, and every pin placed from the element's own bounding box in the page - never by hand.
//
// A setup brings the window to the state a shot shows by the page's own functions (setInputPath,
// addFilesToQueue, openDialog, ..) and by the saved settings of the scratch profile; the only text it
// rewrites is what the machine would otherwise print into a published image (the path in the command
// line), and it says so below.
import os from 'node:os';
import path from 'node:path';
import { demoProfile, REPO } from './app.mjs';

const WIDTH = 1160;
const GUTTER = 40;      // the window's own background, added at the start side of a window capture so the pins sit beside the content, not on it
const MARGIN = 10;
const DIALOG_MARGIN = 20;

export const BOOKS = ['Alice in Wonderland.epub', 'Pride and Prejudice.epub', 'Moby Dick.fb2'];

// The converter's own questions, in the three authored languages, as the window receives them (the strings of
// internal/i18n: "Google Translate - cost", the cost message with its two figures, "FileDO secret file" and
// "Enter the password for %s.").
const CONVERTER_TEXT = {
  en: {
    costTitle: 'Google Translate - cost',
    costMessage: 'Characters to send: 184,200\nEstimated cost: $3.68 USD\n\nSend the text to Google Translate and pay for it? Declining still converts the book, untranslated.',
    secretTitle: 'FileDO secret file',
    secretMessage: 'Enter the password for diary.fd-sec.',
  },
  ru: {
    costTitle: 'Google Translate - стоимость',
    costMessage: 'Символов к отправке: 184,200\nОценка стоимости: $3.68 USD\n\nОтправить текст в Google Translate и оплатить перевод? При отказе книга всё равно будет сконвертирована, без перевода.',
    secretTitle: 'Секретный файл FileDO',
    secretMessage: 'Введите пароль для diary.fd-sec.',
  },
  uk: {
    costTitle: 'Google Translate - вартість',
    costMessage: 'Символів до надсилання: 184,200\nОцінка вартості: $3.68 USD\n\nНадіслати текст до Google Translate і оплатити переклад? У разі відмови книгу все одно буде сконвертовано, без перекладу.',
    secretTitle: 'Секретний файл FileDO',
    secretMessage: 'Введіть пароль для diary.fd-sec.',
  },
};

// The profile of a person who has just installed the app: nothing but the look chosen, the free engine and three
// recent documents.
export function profileSettings({ lang, theme, books, extra }) {
  return Object.assign({ lang, theme, engine: 'chrome', assocPrompted: true, recent: books, srcLang: 'en', dstLang: 'ru' }, extra || {});
}

async function settle(page) {
  // The page asks the server a few things after load (readiness, recent status, the output status, the Google key).
  await page.waitForFunction(() => document.getElementById('aboutVer') && document.getElementById('aboutVer').textContent.trim() !== '');
  await page.waitForTimeout(700);
}

// The demo paths printed into the command line: the machine's own are replaced by these two, so a published image
// carries no path of the developer's disk.
const DEMO_EXE = 'C:\\Tools\\doc-html-translate.exe';
const DEMO_DIR = 'C:\\Books\\';

export const SETUPS = {
  async 'converter-ready'(page, ctx) {
    await page.evaluate((p) => { setInputPath(p); }, ctx.books[0]);
    await page.waitForTimeout(500);
    // A finished result of this file exists: the page's own state, drawn by its own function.
    await page.evaluate(() => {
      outputStatus = {exists: true, paramsChanged: false, outputDir: 'C:\\Books\\Alice in Wonderland'};
      renderPrevResultHint();
    });
    await page.waitForTimeout(300);
  },
  async 'converter-states'(page, ctx) {
    await page.evaluate((books) => {
      document.querySelector('input[name="engine"][value="google"]').checked = true;
      syncEngineUI();
      addFilesToQueue(books);
      queue[0].state = 'done';
      queue[1].state = 'failed';
      renderQueue();
      el('assocPrompt').style.display = '';
    }, ctx.books);
    await page.waitForTimeout(900);
  },
  // The settings live on pages (ticket 119): a shot brings the page it explains to the pane, by the window's own
  // function; the About page also opens its groups, so the captured page shows everything it holds.
  async 'page-advanced'(page) { await showPage_(page, 'advanced'); },
  async 'page-ollama'(page) { await showPage_(page, 'ollama'); },
  async 'page-ocr'(page) { await showPage_(page, 'ocr'); },
  async 'page-windows'(page) { await showPage_(page, 'windows'); },
  async 'page-interface'(page) { await showPage_(page, 'interface'); },
  async 'page-about'(page) {
    await showPage_(page, 'about');
    await page.evaluate(() => { for (const d of document.querySelectorAll('#page-about details')) d.open = true; });
    await page.waitForTimeout(500);
    // The resource index shows what this machine has; a published capture shows fixed demo answers instead, so the
    // set does not depend on the machine that generated it (one helper missing, to show that state too).
    await page.evaluate(() => { resData = { tesseract: true, calibre: true, sevenzip: false, filedo: true }; renderResources(); });
  },
  async 'dialog-cost'(page, ctx) {
    const t = CONVERTER_TEXT[ctx.lang];
    await page.evaluate((t) => {
      openDialog({ title: t.costTitle, message: t.costMessage, surface: 'dlg-cost',
        buttons: [{ id: 'no', key: 'costNo', glyph: 'action.cancel', focus: true },
                  { id: 'yes', key: 'costYes', kind: 'danger', glyph: 'action.translate' }], cancel: 'no' });
    }, t);
    await page.waitForSelector('#dlg[open]');
  },
  async 'dialog-secret'(page, ctx) {
    const t = CONVERTER_TEXT[ctx.lang];
    await page.evaluate((t) => {
      openDialog({ title: t.secretTitle, message: t.secretMessage, secret: true, surface: 'dlg-secret',
        buttons: [{ id: 'cancel', key: 'btnCancel', glyph: 'action.cancel' },
                  { id: 'open', key: 'btnOpenSecret', kind: 'ok', ans: 'ans-ok', glyph: 'action.confirm' }], cancel: 'cancel' });
    }, t);
    await page.waitForSelector('#dlg[open]');
    // A demo password, typed so the field shows its dots; it opens nothing.
    await page.fill('#dlgSecretInput', 'demo-password');
  },
  async 'dialog-run-failed'(page) {
    await page.evaluate(() => {
      const msg = t('runCauseParse');
      openDialog({ title: t('runFailedTitle'), message: msg + '\n\n' + t('runFailedHelp'), surface: 'dlg-run-failed',
        buttons: [{ id: 'close', key: 'dlgClose', glyph: 'nav.close', focus: true },
                  { id: 'send', key: 'btnSendLogs', glyph: 'action.email' },
                  { id: 'retry', key: 'btnRetry', kind: 'ok', ans: 'ans-ok', glyph: 'action.retry' }], cancel: 'close' });
    });
    await page.waitForSelector('#dlg[open]');
  },
};

async function showPage_(page, id) {
  await page.evaluate((i) => { showPage(i, { force: true }); }, id);
  await page.waitForTimeout(400);
}

// The text nodes of the command line carry the machine's paths; the demo ones replace them.
async function maskPaths(page, ctx) {
  await page.evaluate(({ books, exe, dir, profile, demoProfileDir }) => {
    const walker = document.createTreeWalker(document.body, NodeFilter.SHOW_TEXT);
    for (let n = walker.nextNode(); n; n = walker.nextNode()) {
      let s = n.nodeValue;
      if (!/[A-Za-z]:\\/.test(s)) continue;
      for (const b of books) s = s.split(b).join(dir + b.split(/[\\/]/).pop());
      s = s.split(profile).join(demoProfileDir);
      n.nodeValue = s.replace(/&\s+\S*doc-html-translate\.exe/i, '& ' + exe);
    }
  }, { books: ctx.books, exe: DEMO_EXE, dir: DEMO_DIR, profile: path.join(ctx.root, 'profile'), demoProfileDir: 'C:\\Users\\demo\\AppData\\Local' });
}

const union = (rects) => {
  const x1 = Math.min(...rects.map((r) => r.x)), y1 = Math.min(...rects.map((r) => r.y));
  const x2 = Math.max(...rects.map((r) => r.x + r.width)), y2 = Math.max(...rects.map((r) => r.y + r.height));
  return { x: x1, y: y1, width: x2 - x1, height: y2 - y1 };
};

// The pin point of an element: the box's own corner, or a point just outside it, by the mode the copy asks for.
function pinPoint(b, mode) {
  const cx = b.x + b.width / 2, cy = b.y + b.height / 2, off = 18;
  switch (mode || 'tl') {
    case 'tl': return { x: b.x, y: b.y };
    case 'tr': return { x: b.x + b.width, y: b.y };
    case 'bl': return { x: b.x, y: b.y + b.height };
    case 'br': return { x: b.x + b.width, y: b.y + b.height };
    case 'l': return { x: b.x - off, y: cy };
    case 'r': return { x: b.x + b.width + off, y: cy };
    case 't': return { x: cx, y: b.y - off };
    case 'b': return { x: cx, y: b.y + b.height + off };
    case 'gt': return { x: b.x - 24, y: b.y + 14 };      // in the gutter beside the first line
    case 'in': return { x: b.x + 16, y: b.y + 16 };      // inside the corner of a large control
    default: return { x: cx, y: cy };
  }
}

async function rectsOf(page, selectors) {
  return page.evaluate((sels) => sels.map((s) => {
    const el = document.querySelector(s);
    if (!el) return { missing: s };
    const r = el.getBoundingClientRect();
    if (r.width === 0 || r.height === 0) return { missing: s + ' (not visible)' };
    return { x: r.x, y: r.y, width: r.width, height: r.height };
  }), selectors);
}

// captureShot brings the page to the shot's state and returns the image and the pin positions in percent of it.
export async function captureShot({ page, base, locale, theme, page_, shot, elements, ctx, profileDir, settingsFor }) {
  // The profile is rewritten before every load: the window saves what a setup changes, and a shot never inherits
  // the state of the one before it.
  demoProfile(ctx.root, settingsFor(theme));
  const wide = shot.capture === 'dialog' ? WIDTH : WIDTH + GUTTER;
  await page.setViewportSize({ width: wide, height: 4000 });
  await page.goto(`${base}/?lang=${locale}`, { waitUntil: 'load' });
  await settle(page);
  await page.evaluate((th) => { applyTheme(th); }, theme);
  await SETUPS[shot.setup](page, { ...ctx, lang: locale });
  if (shot.capture !== 'dialog') await page.evaluate((g) => { document.querySelector('.win').style.paddingLeft = (16 + g) + 'px'; }, GUTTER);
  await maskPaths(page, ctx);
  // Rule 14: a published image shows demo content, never a path, a name or a key of the machine it was made on.
  const word = (s) => '\\b' + s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&') + '\\b';
  const needles = [REPO, ctx.root, os.homedir()].filter(Boolean).map((s) => s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'))
    .concat([os.userInfo().username, process.env.COMPUTERNAME].filter((s) => s && s.length > 2).map(word));
  const leak = await page.evaluate((ns) => {
    const text = document.body.innerText + ' ' + [...document.querySelectorAll('input')].map((i) => (i.type === 'password' || i.type === 'hidden' ? '' : i.value)).join(' ');
    for (const n of ns) {
      const m = new RegExp(n, 'i').exec(text);
      if (m) return m[0] + '" in "' + text.slice(Math.max(0, m.index - 40), m.index + m[0].length + 40).replace(/\s+/g, ' ');
    }
    return null;
  }, needles);
  if (leak) throw new Error(`shot ${shot.id}: the window shows "${leak}" - a published capture must carry demo content only`);

  let clip;
  if (shot.capture === 'window' || shot.capture === 'region') {
    // The window is taller than a real one so nothing is cut off: the viewport grows to the bottom of the page's
    // content (on the Convert page the controls column, not the log beside it), which is where the page would end if
    // it did not scroll inside itself.
    const h = await page.evaluate(() => {
      const pg = document.querySelector('.page:not([hidden])');
      const body = pg.querySelector('.convert-main') || pg;
      return Math.ceil(body.getBoundingClientRect().bottom + 24);
    });
    const height = Math.max(760, Math.min(h, 4000));
    await page.setViewportSize({ width: WIDTH + GUTTER, height });
    await page.waitForTimeout(400);
    clip = { x: 0, y: 0, width: WIDTH + GUTTER, height };
  }
  if (shot.capture === 'region') {
    const rects = await rectsOf(page, shot.region);
    const miss = rects.find((r) => r.missing);
    if (miss) throw new Error(`shot ${shot.id}: ${miss.missing} is not in the window`);
    const u = union(rects);
    const left = GUTTER + MARGIN;
    clip = { x: Math.max(0, u.x - left), y: Math.max(0, u.y - MARGIN), width: u.width + left + MARGIN, height: u.height + 2 * MARGIN };
  } else if (shot.capture === 'window') {
    // the clip above is the whole viewport
  } else if (shot.capture === 'dialog') {
    // The dialog alone on the page background: the window behind it is hidden and the backdrop is clear, so the
    // image holds nothing but the surface it explains.
    await page.addStyleTag({ content: 'dialog.dlg::backdrop{background:transparent!important}.win{visibility:hidden!important}' });
    const rects = await rectsOf(page, ['#dlg']);
    if (rects[0].missing) throw new Error(`shot ${shot.id}: the dialog is not open`);
    const u = rects[0];
    clip = { x: Math.max(0, u.x - DIALOG_MARGIN), y: Math.max(0, u.y - DIALOG_MARGIN), width: u.width + 2 * DIALOG_MARGIN, height: u.height + 2 * DIALOG_MARGIN };
  } else {
    throw new Error(`unknown capture kind ${shot.capture}`);
  }
  clip = { x: Math.floor(clip.x), y: Math.floor(clip.y), width: Math.ceil(clip.width), height: Math.ceil(clip.height) };

  const pins = {};
  for (const e of elements) {
    const sels = Array.isArray(e.sel) ? e.sel : [e.sel];
    const rects = await rectsOf(page, sels);
    const miss = rects.find((r) => r.missing);
    if (miss) throw new Error(`page ${page_.id}, element ${e.id}: ${miss.missing} - the selector found nothing to point at`);
    const pt = pinPoint(union(rects), e.pin);
    pins[e.id] = {
      x: +((pt.x - clip.x) / clip.width * 100).toFixed(2),
      y: +((pt.y - clip.y) / clip.height * 100).toFixed(2),
    };
  }
  const png = await page.screenshot({ clip, type: 'png' });
  return { png, w: clip.width, h: clip.height, pins };
}
