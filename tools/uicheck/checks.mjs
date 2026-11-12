// The scenarios of the UI-contract checks (ticket 119): BUTTONS 2.0, BOOLEAN-OPTION 2.0, COLLAPSIBLE-GROUP 1.1,
// SETTINGS-NAVIGATION 2.0 and UI-SIZING 1.1, measured on the real build. A screenshot alone proves none of these,
// so every scenario reads geometry, colours, focus and events from the engine and states what it measured.
import path from 'node:path';
import { Report, SCALE, TEXT, TARGET, readerSource, nearestRankP95, writeJSON } from './lib.mjs';

const approx = (a, b, tol = 1) => Math.abs(a - b) <= tol;
const COMMAND = 'button:not(.drop):not(.path-btn):not(.qsel):not(.rail-item):not(.opt-box)';

async function prepare(page) { await page.evaluate(readerSource); }

// every visible command button of the active page, with its measured box and the box of its natural content width
async function commandBoxes(page) {
  return page.evaluate((sel) => {
    const U = window.__uic;
    const out = [];
    for (const b of document.querySelectorAll(sel)) {
      if (!U.visible(b)) continue;
      if (b.closest('[hidden]')) continue;
      const r = U.rect(b), cs = getComputedStyle(b);
      const clone = b.cloneNode(true);
      clone.style.cssText = 'position:absolute;visibility:hidden;width:max-content;max-width:none;min-width:0;flex:none;left:0;top:0';
      clone.removeAttribute('id');
      b.parentNode.appendChild(clone);
      const natural = clone.getBoundingClientRect().width;
      clone.remove();
      out.push({
        id: b.id || b.className || b.textContent.trim().slice(0, 24),
        r, natural,
        radius: parseFloat(cs.borderTopLeftRadius),
        pill: cs.borderTopLeftRadius.includes('%'),
        clipped: b.scrollWidth > b.clientWidth + 1 || b.scrollHeight > b.clientHeight + 1,
        iconOnly: !/\p{L}/u.test(b.textContent),
        stacked: !!b.closest('.state-stack'),
        rowTop: b.parentElement.className,
      });
    }
    return out;
  }, COMMAND);
}

// ---- BUTTONS 2.0 + UI-SIZING geometry ------------------------------------------------------------------
export async function buttonsGeometry(session, report, { size, input, text = 'ordinary', lang = 'en', page: pageId = 'convert', w = 1366, h = 768 }) {
  const tag = `${pageId}/${size}/${input}/text-${text}/${lang}/${w}`;
  const s = SCALE[size], tgt = TARGET[input] * s;
  const { page, close } = await session.page({ q: { size, input, text, lang, page: pageId }, w, h });
  try {
    await prepare(page);
    const boxes = await commandBoxes(page);
    report.check(`buttons.found/${tag}`, boxes.length > 0, `${boxes.length} command buttons measured`);
    for (const b of boxes) {
      const where = `${tag} [${b.id}]`;
      report.check(`buttons.target-height/${where}`, b.r.h >= tgt - 1, `${b.r.h.toFixed(1)} px, floor ${tgt.toFixed(1)}`);
      if (b.iconOnly) report.check(`buttons.target-width/${where}`, b.r.w >= tgt - 1, `${b.r.w.toFixed(1)} px, floor ${tgt.toFixed(1)}`);
      report.check(`buttons.radius/${where}`, !b.pill && approx(b.radius, 4 * s, 0.6), `radius ${b.radius.toFixed(1)}, baseline ${(4 * s).toFixed(1)}`);
      report.check(`buttons.no-clip/${where}`, !b.clipped, 'caption not clipped');
      if (!b.stacked) report.check(`buttons.content-width/${where}`, b.r.w <= Math.max(b.natural, tgt) + 2, `${b.r.w.toFixed(1)} px vs content ${b.natural.toFixed(1)} px and the ${tgt.toFixed(1)} px target minimum (never stretched)`);
    }
    // rows: heights equalize within a visual line, gaps are the token, lines wrap whole buttons
    const rows = await page.evaluate(() => {
      const U = window.__uic, out = [];
      for (const row of document.querySelectorAll('.btn-row')) {
        if (!U.visible(row) || row.closest('[hidden]')) continue;
        const kids = [...row.children].filter((k) => U.visible(k)).map((k) => ({ r: U.rect(k), id: k.id || k.className }));
        if (kids.length) out.push({ id: row.id || row.parentElement.id || row.className, kids });
      }
      return out;
    });
    for (const row of rows) {
      const lines = [];
      for (const k of row.kids) {
        const line = lines.find((l) => Math.abs(l[0].r.y - k.r.y) < 2);
        if (line) line.push(k); else lines.push([k]);
      }
      for (const line of lines) {
        const hs = line.map((k) => k.r.h);
        report.check(`buttons.row-height/${tag} [${row.id}]`, Math.max(...hs) - Math.min(...hs) < 1.5, `heights ${hs.map((x) => x.toFixed(1)).join(', ')} in one line`);
        const sorted = [...line].sort((a, b) => a.r.x - b.r.x);
        for (let i = 1; i < sorted.length; i++) {
          const gap = sorted[i].r.x - (sorted[i - 1].r.x + sorted[i - 1].r.w);
          report.check(`buttons.gap/${tag} [${row.id}]`, approx(gap, 8 * s, 1.5), `gap ${gap.toFixed(1)}, token ${(8 * s).toFixed(1)}`);
        }
      }
    }
    // no two interactive targets of the page overlap (same option / same stack excepted)
    const overlaps = await page.evaluate(() => {
      const U = window.__uic;
      const els = [...document.querySelectorAll('.page:not([hidden]) :is(button, input, select, a[href], summary)'), ...document.querySelectorAll('.rail-item, .head button')]
        .filter((e) => U.visible(e) && !e.closest('[hidden]') && e.type !== 'hidden');
      const bad = [];
      for (let i = 0; i < els.length; i++) for (let j = i + 1; j < els.length; j++) {
        const a = els[i], b = els[j];
        if (a.contains(b) || b.contains(a)) continue;
        if (a.closest('.opt') && a.closest('.opt') === b.closest('.opt')) continue;
        if (a.closest('.state-stack') && a.closest('.state-stack') === b.closest('.state-stack')) continue;
        const r = U.rect(a), q = U.rect(b);
        const ox = Math.min(r.x + r.w, q.x + q.w) - Math.max(r.x, q.x), oy = Math.min(r.y + r.h, q.y + q.h) - Math.max(r.y, q.y);
        if (ox > 1 && oy > 1) bad.push((a.id || a.className || a.tagName) + ' x ' + (b.id || b.className || b.tagName));
      }
      return bad;
    });
    report.check(`targets.no-overlap/${tag}`, overlaps.length === 0, overlaps.slice(0, 4).join('; ') || 'no overlapping interactive targets');
    report.check(`page.no-horizontal-overflow/${tag}`, await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth + 1), 'no horizontal page scroll');
    report.check(`console.clean/${tag}`, page.consoleErrors.length === 0, page.consoleErrors.join(' | '));
    // UI-SIZING section 5: a size change remeasures the stateful stack at its final width
    if (pageId === 'convert') {
      const stack = await page.evaluate(() => {
        const st = document.getElementById('runStack');
        const kids = [...st.children];
        const w = kids.map((k) => { const keep = kids.map((o) => o.style.display); kids.forEach((o) => { o.style.display = o === k ? '' : 'none'; }); const x = k.getBoundingClientRect().width; kids.forEach((o, i) => { o.style.display = keep[i]; }); return x; });
        return { min: parseFloat(st.style.minWidth), widths: w };
      });
      report.check(`buttons.state-width/${tag}`, stack.widths.every((x) => approx(x, Math.max(...stack.widths), 1.5)), `Convert/Cancel widths ${stack.widths.map((x) => x.toFixed(1)).join(' / ')} (stack min ${stack.min})`);
    }
  } finally { await close(); }
}

// ---- dialogs: order, neutral faces, safe default ----------------------------------------------------------
export async function dialogs(session, report) {
  const { page, close } = await session.page({ q: { lang: 'en' } });
  try {
    await prepare(page);
    // a destructive confirmation: Cancel first, the destructive action last, safe answer focused, neutral Cancel
    await page.evaluate(() => { window.__ans = null; confirmDialog('Delete it?', 'btnRebuild', 'danger', true).then((v) => { window.__ans = v; }); });
    await page.waitForSelector('dialog[open]');
    const d = await page.evaluate(() => {
      const U = window.__uic, bs = [...document.querySelectorAll('#dlgActions button')];
      return {
        order: bs.map((b) => b.value), focus: document.activeElement.value,
        classes: bs.map((b) => b.className),
        faces: bs.map((b) => { const bg = U.bgOf(b).map(Math.round).join(','); return bg; }),
        contrast: bs.map((b) => U.textContrast(b)),
        dangerFace: getComputedStyle(bs[bs.length - 1]).backgroundColor,
      };
    });
    report.check('dialog.order/destructive', d.order.join() === 'no,yes', `answers read ${d.order.join(' then ')}`);
    report.check('dialog.safe-default-focus', d.focus === 'no', `focus on ${d.focus}`);
    report.check('dialog.cancel-neutral', !/danger|ans-/.test(d.classes[0]) && d.faces[0] !== d.faces[1], `Cancel face ${d.faces[0]}, class "${d.classes[0]}"`);
    report.check('dialog.destructive-rank', /danger/.test(d.classes[1]), `acting answer class "${d.classes[1]}"`);
    report.check('dialog.caption-contrast', d.contrast.every((c) => c >= 4.5), `captions ${d.contrast.map((c) => c.toFixed(2)).join(' / ')}:1`);
    // Enter on the dialog background selects the safe answer; Escape is the same no-action path
    await page.keyboard.press('Enter');
    await page.waitForFunction(() => window.__ans !== null);
    report.check('dialog.enter-selects-safe', (await page.evaluate(() => window.__ans)) === false, 'Enter from the dialog took the safe answer');
    await page.evaluate(() => { window.__ans = null; confirmDialog('Convert anyway?', 'btnConvertAnyway', 'warn', true).then((v) => { window.__ans = v; }); });
    await page.waitForSelector('dialog[open]');
    const w = await page.evaluate(() => [...document.querySelectorAll('#dlgActions button')].map((b) => b.value));
    report.check('dialog.order/warn', w.join() === 'no,yes', `answers read ${w.join(' then ')}`);
    await page.keyboard.press('Escape');
    await page.waitForFunction(() => window.__ans !== null);
    report.check('dialog.escape-cancels', (await page.evaluate(() => window.__ans)) === false, 'Escape took the no-action path');
    // focus returns to the invoker (the page button the dialog was opened from)
    await page.focus('#btnSwap');
    await page.evaluate(() => { confirmDialog('x', 'btnRebuild', 'danger', true); });
    await page.waitForSelector('dialog[open]');
    await page.keyboard.press('Escape');
    await page.waitForFunction(() => !document.querySelector('dialog[open]'));
    report.check('dialog.focus-returns', (await page.evaluate(() => document.activeElement.id)) === 'btnSwap', 'focus returned to the invoker after closing');
  } finally { await close(); }
}

// ---- BOOLEAN-OPTION 2.0 -------------------------------------------------------------------------------------
export async function options(session, report, { size = 'ordinary', input = 'pointer' } = {}) {
  const tag = `${size}/${input}`;
  const s = SCALE[size], tgt = TARGET[input] * s, box = (input === 'touch' ? 24 : 20) * s;
  const { page, close } = await session.page({ q: { size, input, page: 'ocr' } });
  try {
    await prepare(page);
    const state = () => page.evaluate(() => document.getElementById('chkOCR').checked);
    const geo = await page.evaluate(() => {
      const U = window.__uic, c = document.getElementById('chkOCR'), name = document.querySelector('label[for="chkOCR"]'), ex = document.getElementById('chkOCRD'), cs = getComputedStyle(c, '::before');
      return {
        hit: U.rect(c), name: U.rect(name), ex: U.rect(ex), boxW: parseFloat(cs.width) + 2 * parseFloat(cs.borderTopWidth), boxH: parseFloat(cs.height) + 2 * parseFloat(cs.borderTopWidth),
        outline: U.contrast(U.over(U.rgba(cs.borderTopColor), U.bgOf(c)), U.bgOf(c)),
        captionContrast: U.textContrast(name), explContrast: U.textContrast(ex),
        explSize: parseFloat(getComputedStyle(ex).fontSize), captionSize: parseFloat(getComputedStyle(name).fontSize),
        glyph: U.rect(name.querySelector('.glyph')),
        helpBtn: U.rect(document.querySelector('#page-ocr .opt-help')),
        described: c.getAttribute('aria-describedby'),
      };
    });
    report.check(`option.hit-target/${tag}`, geo.hit.w >= tgt - 1 && geo.hit.h >= tgt - 1, `checkbox hit area ${geo.hit.w.toFixed(1)} x ${geo.hit.h.toFixed(1)}, floor ${tgt.toFixed(1)}`);
    report.check(`option.box-size/${tag}`, approx(geo.boxW, box, 1) && approx(geo.boxH, box, 1), `visible square ${geo.boxW.toFixed(1)} x ${geo.boxH.toFixed(1)}, token ${box.toFixed(1)}`);
    report.check(`option.checkbox-first-line/${tag}`, Math.abs((geo.hit.y + geo.hit.h / 2) - (geo.name.y + geo.name.h / 2)) < 2, 'the checkbox is centred on the first heading line');
    report.check(`option.explanation-below/${tag}`, geo.ex.y >= geo.name.y + geo.name.h - 6, 'the explanation sits under the heading');
    report.check(`option.help-after-name/${tag}`, geo.helpBtn.x >= geo.name.x + geo.name.w - 1 && geo.helpBtn.x - (geo.name.x + geo.name.w) < 24 * s, `help button ${(geo.helpBtn.x - (geo.name.x + geo.name.w)).toFixed(1)} px after the name, not pushed to an edge`);
    report.check(`option.contrast/${tag}`, geo.outline >= 3 && geo.captionContrast >= 4.5 && geo.explContrast >= 4.5, `outline ${geo.outline.toFixed(2)}:1, caption ${geo.captionContrast.toFixed(2)}:1, explanation ${geo.explContrast.toFixed(2)}:1`);
    report.check(`option.explanation-size/${tag}`, geo.explSize >= 12 * s - 0.5 && geo.explSize < geo.captionSize, `explanation ${geo.explSize.toFixed(1)} px, caption ${geo.captionSize.toFixed(1)} px`);
    report.check(`option.described-by/${tag}`, geo.described === 'chkOCRD', `aria-describedby=${geo.described}`);
    if (size !== 'ordinary' || input !== 'pointer') return;

    // activation regions toggle exactly once; the explanation, the gap and the help button never do
    const before = await state();
    await page.click('label[for="chkOCR"] span');
    report.check('option.caption-toggles-once', (await state()) === !before, 'a click on the caption toggled once');
    await page.click('label[for="chkOCR"] .glyph');
    report.check('option.glyph-toggles-once', (await state()) === before, 'a click on the glyph toggled once');
    await page.click('#chkOCR', { position: { x: tgt / 2, y: tgt / 2 } });
    report.check('option.checkbox-toggles-once', (await state()) === !before, 'a click on the checkbox toggled once');
    await page.click('#chkOCRD');
    report.check('option.explanation-does-not-toggle', (await state()) === !before, 'a click on the explanation changed nothing');
    const hb = await page.$('#page-ocr .opt-help');
    await hb.click();
    report.check('option.help-does-not-toggle', (await state()) === !before, 'a click on help changed nothing');
    const full = await page.evaluate(() => { const p = document.querySelector('#page-ocr .opt-pop'); return { hidden: p.hidden, text: p.textContent, expanded: document.querySelector('#page-ocr .opt-help').getAttribute('aria-expanded') }; });
    report.check('option.help-full-on-click', !full.hidden && full.expanded === 'true' && full.text.includes('Off:'), 'activating help shows the full explanation (both states)');
    await page.keyboard.press('Escape');
    report.check('option.help-escape-dismisses', await page.evaluate(() => document.querySelector('#page-ocr .opt-pop').hidden), 'Escape closed the help without moving the pointer');
    report.check('option.help-escape-returns-focus', await page.evaluate(() => document.activeElement.className.includes('opt-help')), 'focus returned to the help button');
    await page.mouse.move(0, 0);
    await page.hover('#page-ocr .opt-help');
    const short = await page.evaluate(() => { const p = document.querySelector('#page-ocr .opt-pop'); return { hidden: p.hidden, text: p.textContent }; });
    report.check('option.help-short-on-hover', !short.hidden && short.text.length > 0 && !short.text.includes('Off:'), 'hover shows the short summary only');
    // the pointer can travel into the bubble without closing it
    const pb = await page.evaluate(() => window.__uic.rect(document.querySelector('#page-ocr .opt-pop')));
    await page.mouse.move(pb.x + pb.w / 2, pb.y + pb.h / 2, { steps: 4 });
    report.check('option.help-stays-open-while-read', !(await page.evaluate(() => document.querySelector('#page-ocr .opt-pop').hidden)), 'the summary stays open while the pointer is inside it');
    await page.mouse.move(0, 0);
    // the short and the full text come from one authored string
    const same = await page.evaluate(() => { const raw = t('helpOptOcr'); const i = raw.indexOf('\n\n'); return { s: raw.slice(0, i), f: raw.slice(0, i) + '\n' + raw.slice(i + 2) }; });
    report.check('option.help-one-source', same.f.startsWith(same.s) && same.f.length > same.s.length, 'the full text begins with the summary (one dictionary string)');
    // keyboard: Space toggles, merely focusing does not
    await page.focus('#chkOCR');
    const afterFocus = await state();
    await page.keyboard.press('Space');
    report.check('option.space-toggles', (await state()) === !afterFocus, 'Space toggled the focused checkbox');
    report.check('option.focus-does-not-toggle', afterFocus === !before, 'focusing alone changed nothing');
    await page.keyboard.press('Space');
    // keyboard order: the checkbox, then its help button; the heading and explanation add no tab stops
    await page.focus('#chkOCR');
    await page.keyboard.press('Tab');
    report.check('option.tab-order', await page.evaluate(() => document.activeElement.className.includes('opt-help')), 'Tab from the checkbox reaches its help button');
    // a disabled option cannot toggle from any path
    await page.evaluate(() => { document.getElementById('chkOCR').disabled = true; });
    const dis = await state();
    await page.click('label[for="chkOCR"] span', { force: true });
    await page.focus('#page-ocr .opt-help');
    report.check('option.disabled-no-toggle', (await state()) === dis, 'a disabled option does not toggle from the caption');
    await page.evaluate(() => { document.getElementById('chkOCR').disabled = false; });
  } finally { await close(); }
}

// ---- COLLAPSIBLE-GROUP 1.1 ---------------------------------------------------------------------------------
export async function groups(session, report) {
  for (const dir of ['ltr', 'rtl']) {
    const { page, close } = await session.page({ q: { page: 'about', lang: dir === 'rtl' ? 'ar' : 'en' }, h: 640 });
    try {
      await prepare(page);
      const h = (id) => page.evaluate((i) => { const g = document.getElementById(i), sm = g.querySelector('summary'), U = window.__uic; const r = U.rect(sm); return { open: g.open, head: r, group: U.rect(g), body: g.querySelector('.body') ? U.rect(g.querySelector('.body')) : null, marker: getComputedStyle(sm, '::before').maskImage, transform: getComputedStyle(sm, '::before').transform, bg: U.bgOf(sm).join(','), weight: getComputedStyle(sm).fontWeight, headContrast: U.textContrast(sm) }; }, id);
      const closed = await h('grpResources');
      const sm = '#grpResources > summary';
      report.check(`group.closed-body-no-space/${dir}`, approx(closed.group.h, closed.head.h + 2, 3), `collapsed group ${closed.group.h.toFixed(1)} px = its ${closed.head.h.toFixed(1)} px header`);
      report.check(`group.header-distinct/${dir}`, closed.bg !== await page.evaluate(() => window.__uic.bgOf(document.querySelector('#page-about')).join(',')) && Number(closed.weight) >= 600, `header face rgb(${closed.bg}), weight ${closed.weight}`);
      report.check(`group.header-contrast/${dir}`, closed.headContrast >= 4.5, `${closed.headContrast.toFixed(2)}:1`);
      report.check(`group.header-target/${dir}`, closed.head.h >= 31, `header ${closed.head.h.toFixed(1)} px high`);
      // activation: triangle, caption and whitespace of the header each toggle exactly once; the body does not
      const hr = closed.head;
      const click = async (x, y) => { await page.mouse.click(x, y); };
      await click(hr.x + 12, hr.y + hr.h / 2);
      report.check(`group.triangle-toggles/${dir}`, (await h('grpResources')).open === true, 'a click on the triangle opened the group');
      const opened = await h('grpResources');
      await click(hr.x + hr.w - 40, hr.y + hr.h / 2);
      report.check(`group.whitespace-toggles-once/${dir}`, (await h('grpResources')).open === false, 'a click on the empty header space closed it (once)');
      await click(hr.x + 40, hr.y + hr.h / 2);
      report.check(`group.caption-toggles/${dir}`, (await h('grpResources')).open === true, 'a click on the caption opened it');
      const body = (await h('grpResources')).body;
      await page.mouse.click(body.x + body.w / 2, body.y + body.h - 4);
      report.check(`group.body-does-not-toggle/${dir}`, (await h('grpResources')).open === true, 'a click inside the body did not collapse the group');
      report.check(`group.marker-state-changes/${dir}`, opened.marker !== closed.marker, 'the state triangle changes with the state');
      const exp = await h('grpResources');
      report.check(`group.full-width-body/${dir}`, approx(exp.body.w, exp.group.w - 2, 2), `body ${exp.body.w.toFixed(1)} of group ${exp.group.w.toFixed(1)} px`);
      if (dir === 'rtl') report.check('group.rtl-collapsed-points-to-reading-end', closed.transform !== 'none', `collapsed marker transform ${closed.transform} (mirrored, pointing left)`);
      // keyboard: Enter and Space toggle, focus stays on the header; a focused child hands focus to the header
      await page.focus(sm);
      await page.keyboard.press('Enter');
      report.check(`group.enter-toggles-focus-stays/${dir}`, (await h('grpResources')).open === false && await page.evaluate(() => document.activeElement.tagName === 'SUMMARY'), 'Enter closed it, focus stayed on the header');
      await page.keyboard.press('Space');
      report.check(`group.space-toggles/${dir}`, (await h('grpResources')).open === true, 'Space opened it');
      await page.evaluate(() => { document.querySelector('#grpResources a').focus(); });
      await page.evaluate(() => { document.getElementById('grpResources').open = false; });
      await page.waitForTimeout(60);
      report.check(`group.focus-moves-to-header/${dir}`, await page.evaluate(() => document.activeElement === document.querySelector('#grpResources > summary')), 'closing a group with a focused child moved the focus to its header');
      await page.focus('#grpResources > summary');
      await page.keyboard.press('Tab');
      report.check(`group.closed-children-unreachable/${dir}`, await page.evaluate(() => !document.activeElement.closest('#grpResources > .body') && [...document.querySelectorAll('#grpResources .body a')].every((a) => !a.checkVisibility())), 'Tab leaves a collapsed group without entering its body, and its links are not rendered');
      // viewport anchor: opening a group keeps its header where it was
      await page.evaluate(() => { document.getElementById('grpRelated').open = false; document.getElementById('grpSupport').open = false; document.getElementById('grpResources').open = false; document.getElementById('grpUpdates').open = false; });
      await page.setViewportSize({ width: 1366, height: 300 });
      await page.evaluate(() => { document.getElementById('grpSupport').open = true; document.getElementById('grpResources').open = true; });
      await page.evaluate(() => { const sm = document.querySelector('#grpRelated > summary'); sm.scrollIntoView({ block: 'center' }); });
      const topBefore = await page.evaluate(() => document.querySelector('#grpRelated > summary').getBoundingClientRect().top);
      await page.evaluate(() => document.querySelector('#grpRelated > summary').click());
      const topAfter = await page.evaluate(() => document.querySelector('#grpRelated > summary').getBoundingClientRect().top);
      report.check(`group.viewport-anchor/${dir}`, approx(topBefore, topAfter, 3), `header top ${topBefore.toFixed(1)} -> ${topAfter.toFixed(1)} px when the body opened`);
    } finally { await close(); }
  }
}

// ---- SETTINGS-NAVIGATION 2.0 -------------------------------------------------------------------------------
export async function navigation(session, report) {
  const { page, close } = await session.page({ q: { lang: 'en' } });
  try {
    await prepare(page);
    const ids = ['convert', 'ollama', 'ocr', 'windows', 'interface', 'advanced', 'about'];
    const rail = await page.evaluate(() => { const U = window.__uic; return [...document.querySelectorAll('.rail-item')].map((b) => ({ id: b.dataset.page, r: U.rect(b), name: b.textContent.trim(), glyph: !!b.querySelector('svg.glyph'), clipped: b.scrollWidth > b.clientWidth + 1 })); });
    report.check('rail.order', rail.map((x) => x.id).join() === ids.join(), `destinations ${rail.map((x) => x.id).join(', ')} (main first, About last)`);
    report.check('rail.glyph-and-name', rail.every((x) => x.glyph && x.name.length > 0), 'every destination has a glyph and a short name');
    report.check('rail.names-whole', rail.every((x) => !x.clipped), 'no caption is clipped');
    report.check('rail.row-target', rail.every((x) => x.r.h >= 31), `rows ${rail.map((x) => x.r.h.toFixed(0)).join('/')} px high`);
    // selection: explicit non-colour cue, aria state, one tab stop, arrows move and select
    const sel = await page.evaluate(() => { const b = document.getElementById('tab-convert'); const cs = getComputedStyle(b); return { aria: b.getAttribute('aria-selected'), weight: cs.fontWeight, shadow: cs.boxShadow, tab: [...document.querySelectorAll('.rail-item')].filter((x) => x.tabIndex === 0).length }; });
    report.check('rail.selected-cue', sel.aria === 'true' && Number(sel.weight) >= 600 && sel.shadow !== 'none', `aria-selected, weight ${sel.weight}, bar ${sel.shadow !== 'none'}`);
    report.check('rail.one-tab-stop', sel.tab === 1, `${sel.tab} rail tab stop`);
    await page.focus('#tab-convert');
    await page.keyboard.press('ArrowDown');
    report.check('rail.arrow-selects', (await page.evaluate(() => document.querySelector('.page:not([hidden])').id)) === 'page-ollama', 'ArrowDown moved to and showed the next page');
    await page.keyboard.press('End');
    report.check('rail.end-selects-about', (await page.evaluate(() => document.querySelector('.page:not([hidden])').id)) === 'page-about', 'End jumped to About');
    // navigation never resizes the window, moves the rail, discards edits or invokes an operation
    await page.click('#tab-convert');
    await page.fill('#outputFolder', 'D:\\typed-by-the-test');
    // the edit itself refreshes the command-line preview after a short debounce, on input and again when the field
    // loses focus (its change event); commit the edit and let both settle before navigation is recorded
    await page.evaluate(() => document.getElementById('outputFolder').blur());
    await page.waitForTimeout(900);
    const win0 = await page.evaluate(() => [window.outerWidth, window.outerHeight, window.__uic.rect(document.getElementById('rail')).x]);
    page.requests.length = 0;
    for (const id of ['ocr', 'interface', 'about', 'advanced', 'convert']) await page.click('#tab-' + id);
    const win1 = await page.evaluate(() => [window.outerWidth, window.outerHeight, window.__uic.rect(document.getElementById('rail')).x]);
    report.check('nav.outer-window-unchanged', win0.join() === win1.join(), `outer window and rail position unchanged (${win0.join(' x ')})`);
    report.check('nav.edit-preserved', (await page.inputValue('#outputFolder')) === 'D:\\typed-by-the-test', 'an unsaved edit survived five page changes');
    const ops = page.requests.filter((r) => !/api\/(settings|resources)/.test(r));
    report.check('nav.no-operation', ops.length === 0, ops.join(', ') || 'only the settings save and the local resource lookup were requested');
    report.check('nav.focus-stays-on-rail', await page.evaluate(() => document.activeElement.id === 'tab-convert'), 'focus stayed on the rail (no jump to an input)');
    // About: opening it contacts nothing remote
    const remote = [];
    page.on('request', (r) => { if (!r.url().startsWith(session.server.base)) remote.push(r.url()); });
    await page.click('#tab-about');
    await page.waitForTimeout(400);
    report.check('about.no-remote-on-open', remote.length === 0, remote.join(', ') || 'no request left this machine');
    // the page and its viewport are remembered
    await page.evaluate(() => { document.getElementById('grpSupport').open = false; });
    await page.click('#tab-interface');
    await page.waitForTimeout(700);
    const saved = JSON.parse(await (await page.request.get(session.server.base + '/api/settings', { headers: { 'X-DHT-Token': await page.evaluate(() => API_TOKEN) } })).text());
    report.check('nav.page-remembered', saved.page === 'interface', `saved page ${saved.page}`);
    report.check('nav.group-state-remembered', saved.groups && saved.groups.grpSupport === false && saved.groups.grpResources === false, `groups ${JSON.stringify(saved.groups)}`);
  } finally { await close(); }
  // stale asynchronous results never replace the current destination: delay the resource lookup, leave About
  const s2 = await session.page({ q: { lang: 'en', page: 'convert' } });
  try {
    await s2.page.route('**/api/resources', async (route) => { await new Promise((r) => setTimeout(r, 500)); await route.continue(); });
    await prepare(s2.page);
    await s2.page.click('#tab-about');
    await s2.page.click('#tab-ocr');
    await s2.page.waitForTimeout(900);
    report.check('nav.stale-result-ignored', (await s2.page.evaluate(() => document.querySelector('.page:not([hidden])').id)) === 'page-ocr', 'a late About lookup did not change the page the reader is on');
  } finally { await s2.close(); }
}

// ---- boot: no flash of the wrong language ----------------------------------------------------------------
export async function boot(session, report) {
  const frames = [];
  const { page, close } = await session.page({ q: { lang: 'ru' }, init: `(() => { window.__frames = []; const tick = () => { const b = document.querySelector('#tab-convert span'); window.__frames.push({ boot: document.documentElement.hasAttribute('data-boot'), text: b ? b.textContent : null, theme: document.documentElement.dataset.theme || '', vis: document.body ? getComputedStyle(document.body).visibility : 'none' }); if (window.__frames.length < 120) requestAnimationFrame(tick); }; requestAnimationFrame(tick); })();` });
  try {
    const f = await page.evaluate(() => window.__frames);
    const shown = f.filter((x) => x.vis !== 'hidden' && x.text !== null);
    report.check('boot.no-untranslated-frame', shown.length > 0 && shown.every((x) => x.text === 'Конвертация'), `${shown.length} visible frames, all in the saved language`);
    report.check('boot.hidden-until-ready', f.some((x) => x.boot) || shown.length === f.length, 'the first frame was held back or already complete');
  } finally { await close(); }
}

// ---- timing: activation to frame, raw samples ------------------------------------------------------------
export async function timings(session, report, outDir, { warm = 20, cold = 20 } = {}) {
  const ids = ['convert', 'ollama', 'ocr', 'windows', 'interface', 'advanced', 'about'];
  const meas = `
    window.__switch = (id) => new Promise((resolve) => {
      const t0 = performance.now();
      document.getElementById('tab-' + id).click();
      requestAnimationFrame(() => requestAnimationFrame(() => resolve(performance.now() - t0)));
    });`;
  const raw = { warm: {}, cold: {} };
  const { page, close } = await session.page({ q: { lang: 'en' } });
  try {
    await page.evaluate(meas);
    for (const id of ids) await page.evaluate((i) => window.__switch(i), id); // first visit of each
    for (const id of ids) {
      const samples = [];
      for (let i = 0; i < warm; i++) {
        await page.evaluate((i) => window.__switch(i), ids[(ids.indexOf(id) + 1) % ids.length]);
        samples.push(await page.evaluate((i) => window.__switch(i), id));
      }
      raw.warm[id] = samples;
      const p95 = nearestRankP95(samples), max = Math.max(...samples);
      report.check(`timing.warm/${id}`, p95 < 150, `p95 ${p95.toFixed(1)} ms, max ${max.toFixed(1)} ms over ${samples.length} warm transitions (budget 150)`);
      report.check(`timing.warm-no-outlier/${id}`, max < 150, `max ${max.toFixed(1)} ms`);
    }
  } finally { await close(); }
  for (const id of ['ollama', 'about', 'advanced']) {
    const samples = [];
    for (let i = 0; i < cold; i++) {
      const s = await session.page({ q: { lang: 'en' } });
      try { await s.page.evaluate(meas); samples.push(await s.page.evaluate((x) => window.__switch(x), id)); } finally { await s.close(); }
    }
    raw.cold[id] = samples;
    const p95 = nearestRankP95(samples), max = Math.max(...samples);
    report.check(`timing.first-visit/${id}`, p95 < 500, `p95 ${p95.toFixed(1)} ms, max ${max.toFixed(1)} ms over ${samples.length} independent first visits (budget 500)`);
  }
  writeJSON(path.join(outDir, 'timings.raw.json'), { method: 'activation (click) to the second animation frame after it, performance.now()', raw });
}
