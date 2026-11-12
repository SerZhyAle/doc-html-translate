// The display modes and the files of the extension's own pages (ticket 119, slice X1): the interface theme following the
// browser live (SETTINGS-NAVIGATION 3 rule 1), forced colours (what a colour carried must survive in system colours), more
// contrast, and the two files the options page hands the reader - the settings file and the diagnostics summary - which
// must be local downloads that contain no document text and request nothing.
const f1 = (n) => (Math.round(n * 10) / 10).toString();
const OPTIONS = { name: 'options', url: '/src/options.html', size: 'ordinary', input: 'pointer', w: 1280, h: 900 };

const bgOfBody = (page) => page.evaluate(() => getComputedStyle(document.body).backgroundColor);

export async function modeChecks(browser, base, report, { openPage }) {
  // ---- the interface theme: System follows the browser live; Light and Dark pin it ----
  const t = await openPage(browser, base, { ...OPTIONS, chrome: 'system', scheme: 'light' });
  try {
    const { page } = t;
    const light = await bgOfBody(page);
    await page.emulateMedia({ colorScheme: 'dark' });
    await page.waitForTimeout(80);
    const dark = await bgOfBody(page);
    report.check('theme.system-follows-the-browser-live', light !== dark && dark === 'rgb(26, 26, 28)', `System: light ${light}, after the browser switched to dark ${dark}, no reload`);
    await page.selectOption('#chrome-theme', 'light');
    await page.waitForTimeout(80);
    const pinnedLight = await bgOfBody(page);
    report.check('theme.light-wins-over-a-dark-browser', pinnedLight === 'rgb(250, 249, 247)' && (await page.evaluate(() => getComputedStyle(document.documentElement).colorScheme)) === 'light', `Light under a dark browser: ${pinnedLight}, applied at once`);
    await page.emulateMedia({ colorScheme: 'light' });
    await page.selectOption('#chrome-theme', 'dark');
    await page.waitForTimeout(80);
    const pinnedDark = await bgOfBody(page);
    report.check('theme.dark-wins-over-a-light-browser', pinnedDark === 'rgb(26, 26, 28)' && (await page.evaluate(() => getComputedStyle(document.documentElement).colorScheme)) === 'dark', `Dark under a light browser: ${pinnedDark}`);
    report.check('theme.choice-is-stored-once', (await page.evaluate(() => window.__stored.options.chromeTheme)) === 'dark', 'the choice is a stored option, so the popup takes it too');
    await page.selectOption('#chrome-theme', 'system');
    await page.waitForTimeout(80);
    report.check('theme.system-again-follows-the-browser', (await page.evaluate(() => document.documentElement.hasAttribute('data-chrome-theme'))) === false, 'choosing System removes the pin');
  } finally { await t.close(); }
  const p = await openPage(browser, base, { name: 'popup', url: '/src/popup.html', w: 800, h: 900, chrome: 'dark', scheme: 'light' });
  try { report.check('theme.popup-takes-the-chosen-theme', (await bgOfBody(p.page)) === 'rgb(26, 26, 28)', `popup under a light browser with Dark chosen: ${await bgOfBody(p.page)}`); } finally { await p.close(); }

  // ---- forced colours: the mask-drawn marks, the selection, the rank and the focus ring survive in system colours ----
  const fc = await openPage(browser, base, { ...OPTIONS, forced: true, options: { ocrImages: true }, uiState: { page: 'ocr', groups: {} } });
  try {
    const { page } = fc;
    const m = await page.evaluate(() => {
      const U = window.__uic, box = document.getElementById('ocr-images');
      const fill = getComputedStyle(box, '::before').backgroundColor, mark = getComputedStyle(box, '::after').backgroundColor, outline = getComputedStyle(box, '::before').borderTopColor;
      const canvas = getComputedStyle(document.body).backgroundColor;
      const sel = getComputedStyle(document.querySelector('.rail-item[aria-selected="true"]'));
      document.getElementById('tab-about').click();
      const sm = document.querySelector('#aboutSupport > summary');
      const markerColor = getComputedStyle(sm, '::before').backgroundColor;
      return { fill, mark, outline, canvas, selBg: sel.backgroundColor, selInk: sel.color, markerColor, markerMask: getComputedStyle(sm, '::before').maskImage !== 'none' };
    });
    report.check('forced.checkbox-mark-visible', m.mark !== 'rgba(0, 0, 0, 0)' && m.mark !== m.fill && m.outline !== m.canvas, `checked mark ${m.mark} on a ${m.fill} square, outline ${m.outline} on ${m.canvas}`);
    report.check('forced.rail-selection-is-a-system-pair', m.selBg !== m.canvas && m.selBg !== 'rgba(0, 0, 0, 0)' && m.selInk !== m.selBg, `selected row ${m.selBg} with ink ${m.selInk}, page ${m.canvas}`);
    report.check('forced.group-marker-visible', m.markerMask && m.markerColor !== m.canvas && m.markerColor !== 'rgba(0, 0, 0, 0)', `state triangle ${m.markerColor} on ${m.canvas}`);
    await page.click('#tab-data');
    await page.keyboard.press('Tab');
    const ring = await page.evaluate(() => { const cs = getComputedStyle(document.activeElement); return { style: cs.outlineStyle, width: parseFloat(cs.outlineWidth), id: document.activeElement.id || document.activeElement.tagName }; });
    report.check('forced.focus-ring-stays', ring.style !== 'none' && ring.width >= 1, `keyboard focus on ${ring.id} draws a ${ring.width}px ${ring.style} ring`);
  } finally { await fc.close(); }
  const fp = await openPage(browser, base, { name: 'popup', url: '/src/popup.html', w: 800, h: 900, forced: true });
  try {
    const r = await fp.page.evaluate(() => { const p = getComputedStyle(document.getElementById('open-pdf')), n = getComputedStyle(document.getElementById('opts')); return { primary: p.backgroundColor, neutral: n.backgroundColor, ink: p.color }; });
    report.check('forced.primary-rank-survives', r.primary !== r.neutral && r.primary !== r.ink, `primary ${r.primary} against a neutral ${r.neutral} button`);
  } finally { await fp.close(); }

  // ---- prefers-contrast: more draws every border in the ink ----
  const cm = await openPage(browser, base, { ...OPTIONS, contrastMore: true, uiState: { page: 'data', groups: {} } });
  try {
    const b = await cm.page.evaluate(() => [...document.querySelectorAll('#page-data button, select')].filter((e) => window.__uic.visible(e)).map((e) => ({ k: e.id, border: getComputedStyle(e).borderTopColor, ink: getComputedStyle(e).color })));
    report.check('contrast-more.borders-are-the-ink', b.length > 0 && b.every((x) => x.border === x.ink), b.map((x) => `${x.k} ${x.border}/${x.ink}`).join('; '));
  } finally { await cm.close(); }
}

// ---- the two files: made locally, free of document text, requesting nothing ----
export async function fileChecks(browser, base, report, { openPage }) {
  const o = await openPage(browser, base, { ...OPTIONS, permissions: ['clipboard-read', 'clipboard-write'], uiState: { page: 'about', groups: {} }, options: { disabledHosts: ['secret-intranet.example'], theme: 'dark' } });
  try {
    const { page } = o;
    const before = page.requests.length;
    const [saved] = await Promise.all([page.waitForEvent('download', { timeout: 8000 }), page.click('#save-diag')]);
    const text = await (async () => { const chunks = []; for await (const c of await saved.createReadStream()) chunks.push(c); return Buffer.concat(chunks).toString('utf8'); })();
    // DIAGNOSTIC-REPORT 8 A: the saved file and the clipboard carry one JSON envelope with the readable lines in `report`.
    const parse = (s) => { try { return JSON.parse(s); } catch { return null; } };
    const env = parse(text);
    report.check('about.save-diagnostics-is-a-local-text-file', saved.suggestedFilename() === 'doc-html-translate-diagnostics.json' && !!env && env.schemaVersion === 1 && env.app && env.app.edition === 'browser extension' && Array.isArray(env.report) && env.report.includes('edition: browser extension') && env.report.some((l) => /^version: /.test(l)), `${saved.suggestedFilename()}, schemaVersion ${env && env.schemaVersion}, ${env && env.report ? env.report.length : 0} report lines`);
    report.check('about.diagnostics-carry-no-document-text-or-host', !/secret-intranet|https?:/i.test(text) && !!env && env.options.disabledHosts === 1 && env.report.includes('disabled hosts: 1'), 'a count of disabled hosts, no host name, no address');
    report.check('about.saving-requests-nothing', page.remote.length === 0 && page.requests.slice(before).every((u) => u.startsWith(base) || /^blob:|^data:/.test(u)), 'no request left the extension origin');
    await page.click('#copy-diag');
    await page.waitForTimeout(250);
    const copied = await page.evaluate(async () => ({ status: document.getElementById('diag-status').textContent, clip: await navigator.clipboard.readText().catch(() => null) }));
    // The Windows clipboard turns line feeds into CR LF on its own; the text is the same.
    const clip = (copied.clip || '').replace(/\r\n/g, '\n');
    // Save and Copy each build their envelope when clicked, so the two differ by `timestamp` alone; the summary is the rest.
    const strip = (e) => (e ? JSON.stringify({ ...e, timestamp: undefined }) : null);
    const same = strip(parse(clip)) !== null && strip(parse(clip)) === strip(env);
    report.check('about.copy-diagnostics-puts-the-same-summary-on-the-clipboard', same && copied.status.length > 0, `status "${copied.status}", the clipboard holds the same summary as the saved file (timestamp aside): ${same}`);
    await page.click('#tab-data');
    const [exported] = await Promise.all([page.waitForEvent('download', { timeout: 8000 }), page.click('#export-settings')]);
    const json = await (async () => { const chunks = []; for await (const c of await exported.createReadStream()) chunks.push(c); return JSON.parse(Buffer.concat(chunks).toString('utf8')); })();
    report.check('data.export-is-a-local-settings-file', exported.suggestedFilename() === 'doc-html-translate-settings.json' && json.version === 2 && json.options.theme === 'dark' && json.options.uiSize === 'ordinary' && !!json.ui, `${exported.suggestedFilename()}, format version ${json.version}, carries the interface settings and the page memory (${f1(JSON.stringify(json).length)} bytes)`);
    report.check('data.export-has-no-document-or-reading-data', !/readingPositions|lastRun|scroll/.test(JSON.stringify(json)), 'no reading positions, no last run, no viewports');
  } finally { await o.close(); }
}
