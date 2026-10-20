// The rule evaluators of the extension's own pages (ticket 119, slice X1): BUTTONS 2.0, UI-SIZING 1.1,
// BOOLEAN-OPTION 2.0 and COLLAPSIBLE-GROUP 1.1 measured on what the engine laid out, with the same numbers and the
// same tests as the launcher's checks.mjs - so one rule list holds both products. Every function takes a Playwright
// page that already ran readerSource (window.__uic) and a Report, and states what it measured in the detail.
export const approx = (a, b, tol = 1) => Math.abs(a - b) <= tol;
const f1 = (n) => (Math.round(n * 10) / 10).toString();

// the command buttons of the visible page: navigation rows and the checkbox are components of their own
export const COMMAND = 'button:not(.rail-item):not(.opt-box)';

export async function commandBoxes(page, scope = 'body') {
  return page.evaluate(([sel, scopeSel]) => {
    const U = window.__uic;
    const out = [];
    for (const b of document.querySelectorAll(scopeSel + ' ' + sel)) {
      if (!U.visible(b) || b.closest('[hidden]')) continue;
      const r = U.rect(b), cs = getComputedStyle(b);
      const clone = b.cloneNode(true);
      clone.style.cssText = 'position:absolute;visibility:hidden;width:max-content;max-width:none;min-width:0;flex:none;left:0;top:0';
      clone.removeAttribute('id');
      b.parentNode.appendChild(clone);
      const natural = clone.getBoundingClientRect().width;
      clone.remove();
      out.push({
        id: b.id || (typeof b.className === 'string' && b.className) || b.textContent.trim().slice(0, 24),
        r, natural, radius: parseFloat(cs.borderTopLeftRadius), pill: cs.borderTopLeftRadius.includes('%'),
        clipped: b.scrollWidth > b.clientWidth + 1 || b.scrollHeight > b.clientHeight + 1,
        iconOnly: !/\p{L}/u.test(b.textContent), glyphFirst: !!b.firstElementChild && (b.firstElementChild.matches('svg') || !!b.firstElementChild.querySelector('svg')),
        hasGlyph: !!b.querySelector('svg'), fs: parseFloat(cs.fontSize), disabled: b.disabled,
      });
    }
    return out;
  }, [COMMAND, scope]);
}

// BUTTONS 2.0 sections 3 and 4 and UI-SIZING 1.1 section 2: target, corner, content width, clipping, rows, gaps, overlap
export async function buttonsGeometry(page, report, tag, { tgt, s, expectButtons = true }) {
  const boxes = await commandBoxes(page);
  // The Interface page is three labelled editors: no command button is expected there.
  if (expectButtons) report.check(`buttons.found/${tag}`, boxes.length > 0, `${boxes.length} command buttons measured`);
  for (const b of boxes) {
    const where = `${tag} [${b.id}]`;
    report.check(`buttons.target-height/${where}`, b.r.h >= tgt - 1, `${f1(b.r.h)} px, floor ${f1(tgt)}`);
    if (b.iconOnly) report.check(`buttons.target-width/${where}`, b.r.w >= tgt - 1, `${f1(b.r.w)} px, floor ${f1(tgt)}`);
    report.check(`buttons.radius/${where}`, !b.pill && approx(b.radius, 4 * s, 0.6), `radius ${f1(b.radius)}, baseline ${f1(4 * s)}`);
    report.check(`buttons.no-clip/${where}`, !b.clipped, 'caption not clipped');
    report.check(`buttons.content-width/${where}`, b.r.w <= Math.max(b.natural, tgt) + 2, `${f1(b.r.w)} px vs content ${f1(b.natural)} px and the ${f1(tgt)} px target minimum (never stretched)`);
  }
  const rows = await page.evaluate(() => {
    const U = window.__uic, out = [];
    for (const row of document.querySelectorAll('.btn-row, .dlg-actions, .ocr-error')) {
      if (!U.visible(row) || row.closest('[hidden]')) continue;
      const kids = [...row.children].filter((k) => U.visible(k) && k.matches('button')).map((k) => ({ r: U.rect(k), id: k.id || k.className }));
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
      report.check(`buttons.row-height/${tag} [${row.id}]`, Math.max(...hs) - Math.min(...hs) < 1.5, `heights ${hs.map(f1).join(', ')} in one line`);
      const sorted = [...line].sort((a, b) => a.r.x - b.r.x);
      for (let i = 1; i < sorted.length; i++) {
        const gap = sorted[i].r.x - (sorted[i - 1].r.x + sorted[i - 1].r.w);
        report.check(`buttons.gap/${tag} [${row.id}]`, approx(gap, 8 * s, 1.5), `gap ${f1(gap)}, token ${f1(8 * s)}`);
      }
    }
  }
  const overlaps = await page.evaluate(() => {
    const U = window.__uic;
    const els = [...document.querySelectorAll('button, input, select, a[href], summary')].filter((e) => U.visible(e) && !e.closest('[hidden]') && e.type !== 'hidden');
    const bad = [];
    for (let i = 0; i < els.length; i++) for (let j = i + 1; j < els.length; j++) {
      const a = els[i], b = els[j];
      if (a.contains(b) || b.contains(a)) continue;
      // a checkbox's hit area and its own heading label are one activation region of one option
      if (a.closest('.opt') && a.closest('.opt') === b.closest('.opt') && (a.matches('.opt-box') || b.matches('.opt-box')) && !(a.matches('.opt-help') || b.matches('.opt-help'))) continue;
      const r = U.rect(a), q = U.rect(b);
      const ox = Math.min(r.x + r.w, q.x + q.w) - Math.max(r.x, q.x), oy = Math.min(r.y + r.h, q.y + q.h) - Math.max(r.y, q.y);
      if (ox > 1 && oy > 1) bad.push((a.id || a.className || a.tagName) + ' x ' + (b.id || b.className || b.tagName));
    }
    return bad;
  });
  report.check(`targets.no-overlap/${tag}`, overlaps.length === 0, overlaps.slice(0, 4).join('; ') || 'no overlapping interactive targets');
  report.check(`page.no-horizontal-overflow/${tag}`, await page.evaluate(() => {
    const pane = document.getElementById('pane');
    return document.documentElement.scrollWidth <= window.innerWidth + 1 && (!pane || pane.scrollWidth <= pane.clientWidth + 1);
  }), 'no horizontal scroll of the page or the pane');
  return boxes;
}

// BUTTONS 3 rule 6 and BOOLEAN-OPTION 3 rule 5: contrast of the actual pairs, idle
export async function contrastRules(page, report, tag) {
  const m = await page.evaluate(() => {
    const U = window.__uic, rows = [];
    const sel = 'button:not(:disabled), .opt-name, .opt-expl, .hint, summary, .rail-item, label, a[href], h1, h2, .about-row, .res-name, .res-what, select';
    for (const e of document.querySelectorAll(sel)) {
      if (!U.visible(e) || e.closest('[hidden]') || !e.textContent.trim()) continue;
      const cs = getComputedStyle(e);
      let text = null;
      try { text = U.textContrast(e); } catch { /* an element with no own face */ }
      const g = e.querySelector('svg');
      let glyph = null;
      if (g) { const face = U.bgOf(e); const ink = U.over(U.fgOf(g, 'color'), face); glyph = U.contrast(ink, face); }
      const large = parseFloat(cs.fontSize) >= 24 || (parseFloat(cs.fontSize) >= 18.66 && Number(cs.fontWeight) >= 700);
      rows.push({ key: (e.id ? '#' + e.id : (typeof e.className === 'string' && e.className.split(' ')[0]) || e.tagName.toLowerCase()) + ' "' + e.textContent.trim().slice(0, 16) + '"', text, glyph, large });
    }
    return rows;
  });
  const lowText = m.filter((x) => x.text !== null && x.text < (x.large ? 3 : 4.5));
  const lowGlyph = m.filter((x) => x.glyph !== null && x.glyph < 3);
  const minT = Math.min(...m.filter((x) => x.text !== null).map((x) => x.text));
  const minG = Math.min(...m.filter((x) => x.glyph !== null).map((x) => x.glyph));
  report.check(`contrast.text/${tag}`, m.length > 0 && lowText.length === 0, lowText.length ? lowText.map((x) => `${x.key} ${f1(x.text)}`).join('; ') : `${m.length} text pairs, lowest ${f1(minT)}:1`);
  report.check(`contrast.glyph/${tag}`, lowGlyph.length === 0, lowGlyph.length ? lowGlyph.map((x) => `${x.key} ${f1(x.glyph)}`).join('; ') : `lowest glyph pair ${f1(minG)}:1`);
}

// the three outlines of BUTTONS 3 rule 6 / BOOLEAN-OPTION 3 rule 5 that are not text: a button's border and a checkbox
// square against the surface they sit on, and the check mark against the filled square
export async function outlineContrast(page, report, tag) {
  const o = await page.evaluate(() => {
    const U = window.__uic, out = { buttons: [], boxes: [], marks: [] };
    for (const b of document.querySelectorAll('button:not(:disabled):not(.rail-item):not(.opt-help):not(.opt-box):not(.secondary)')) {
      if (!U.visible(b) || b.closest('[hidden]')) continue;
      const bd = U.rgba(getComputedStyle(b).borderTopColor), side = U.bgOf(b.parentElement);
      if (bd) out.buttons.push({ key: b.id || b.textContent.trim().slice(0, 14), c: U.contrast(U.over(bd, side), side) });
    }
    for (const c of document.querySelectorAll('input.opt-box')) {
      if (!U.visible(c) || c.closest('[hidden]')) continue;
      const cs = getComputedStyle(c, '::before'), surface = U.bgOf(c.parentElement);
      const line = U.rgba(cs.borderTopColor);
      out.boxes.push({ key: c.id, checked: c.checked, c: c.checked ? null : U.contrast(U.over(line, surface), surface) });
      if (c.checked) {
        const fill = U.rgba(cs.backgroundColor), mark = U.rgba(getComputedStyle(c, '::after').backgroundColor);
        if (fill && mark) out.marks.push({ key: c.id, c: U.contrast(U.over(mark, fill), fill), fillVsSurface: U.contrast(U.over(fill, surface), surface) });
      }
    }
    return out;
  });
  const lowB = o.buttons.filter((x) => x.c < 3), lowC = o.boxes.filter((x) => x.c !== null && x.c < 3), lowM = o.marks.filter((x) => x.c < 3 || x.fillVsSurface < 3);
  report.check(`contrast.button-border/${tag}`, lowB.length === 0, lowB.length ? lowB.map((x) => `${x.key} ${f1(x.c)}`).join('; ') : `${o.buttons.length} borders, lowest ${f1(Math.min(...o.buttons.map((x) => x.c), 99))}:1 (needs 3:1)`);
  report.check(`contrast.checkbox-outline/${tag}`, lowC.length === 0, lowC.length ? lowC.map((x) => `${x.key} ${f1(x.c)}`).join('; ') : `${o.boxes.filter((x) => !x.checked).length} outlines, lowest ${f1(Math.min(...o.boxes.filter((x) => !x.checked).map((x) => x.c), 99))}:1`);
  report.check(`contrast.checkbox-mark/${tag}`, lowM.length === 0, lowM.length ? lowM.map((x) => `${x.key} mark ${f1(x.c)} fill ${f1(x.fillVsSurface)}`).join('; ') : `${o.marks.length} checked boxes: mark and fill both clear 3:1`);
}

// BOOLEAN-OPTION 2.0 sections 2 and 3: geometry of one option row
export async function optionGeometry(page, report, tag, { id, tgt, box, s, glyph }) {
  const g = await page.evaluate(([i]) => {
    const U = window.__uic, c = document.getElementById(i), name = document.querySelector(`label[for="${i}"]`), ex = document.getElementById(c.getAttribute('aria-describedby') || ''), cs = getComputedStyle(c, '::before');
    const help = c.closest('.opt').querySelector('.opt-help');
    return {
      hit: U.rect(c), name: U.rect(name), ex: ex ? U.rect(ex) : null, boxW: parseFloat(cs.width) + 2 * parseFloat(cs.borderTopWidth), boxH: parseFloat(cs.height) + 2 * parseFloat(cs.borderTopWidth),
      explSize: ex ? parseFloat(getComputedStyle(ex).fontSize) : null, captionSize: parseFloat(getComputedStyle(name).fontSize),
      help: help ? U.rect(help) : null, glyph: name.querySelector('svg') ? U.rect(name.querySelector('svg')) : null, described: c.getAttribute('aria-describedby'),
    };
  }, [id]);
  report.check(`option.hit-target/${tag}`, g.hit.w >= tgt - 1 && g.hit.h >= tgt - 1, `checkbox hit area ${f1(g.hit.w)} x ${f1(g.hit.h)}, floor ${f1(tgt)}`);
  report.check(`option.box-size/${tag}`, approx(g.boxW, box, 1) && approx(g.boxH, box, 1), `visible square ${f1(g.boxW)} x ${f1(g.boxH)}, token ${f1(box)}`);
  report.check(`option.checkbox-first-line/${tag}`, Math.abs((g.hit.y + g.hit.h / 2) - (g.name.y + g.name.h / 2)) < 2, 'the checkbox is centred on the first heading line');
  if (g.ex) {
    report.check(`option.explanation-below/${tag}`, g.ex.y >= g.name.y + g.name.h - 6, 'the explanation sits under the heading');
    report.check(`option.explanation-size/${tag}`, g.explSize >= 12 * s - 0.5 && g.explSize < g.captionSize, `explanation ${f1(g.explSize)} px, caption ${f1(g.captionSize)} px`);
    report.check(`option.described-by/${tag}`, !!g.described, `aria-describedby=${g.described}`);
  }
  if (g.help) {
    const gap = g.help.x - (g.name.x + g.name.w);
    const rtl = g.help.x < g.name.x;
    const near = rtl ? (g.name.x - (g.help.x + g.help.w)) : gap;
    report.check(`option.help-after-name/${tag}`, near >= -1 && near < 24 * s, `help button ${f1(near)} px after the name, not pushed to an edge`);
    const overlap = Math.min(g.help.x + g.help.w, g.hit.x + g.hit.w) - Math.max(g.help.x, g.hit.x) > 1 && Math.min(g.help.y + g.help.h, g.hit.y + g.hit.h) - Math.max(g.help.y, g.hit.y) > 1;
    report.check(`option.help-no-overlap/${tag}`, !overlap, 'the help target does not overlap the checkbox target');
  }
  if (g.glyph && glyph) report.check(`option.heading-glyph-tier/${tag}`, approx(g.glyph.w, glyph, 1), `heading glyph ${f1(g.glyph.w)} px, tier ${f1(glyph)}`);
}

// BOOLEAN-OPTION 3 and 4: what a click, a key and a help interaction do and do not toggle
export async function optionInteraction(page, report, tag, { id, tgt, named }) {
  const state = () => page.evaluate((i) => document.getElementById(i).checked, id);
  const before = await state();
  await page.click(`label[for="${id}"] span:last-child`);
  report.check(`option.caption-toggles-once/${tag}`, (await state()) === !before, 'a click on the caption toggled once');
  if (await page.$(`label[for="${id}"] svg`)) {
    await page.click(`label[for="${id}"] svg`);
    report.check(`option.glyph-toggles-once/${tag}`, (await state()) === before, 'a click on the glyph toggled once');
    await page.click(`label[for="${id}"] span:last-child`);
  }
  const now = await state();
  await page.click(`#${id}`, { position: { x: tgt / 2, y: tgt / 2 } });
  report.check(`option.checkbox-toggles-once/${tag}`, (await state()) === !now, 'a click on the checkbox toggled once');
  const settled = await state();
  const expl = await page.getAttribute(`#${id}`, 'aria-describedby');
  if (expl) {
    await page.click(`#${expl}`);
    report.check(`option.explanation-does-not-toggle/${tag}`, (await state()) === settled, 'a click on the explanation changed nothing');
  }
  const help = await page.$(`#${id}`).then((h) => h.evaluateHandle((e) => e.closest('.opt').querySelector('.opt-help')));
  const helpEl = help.asElement();
  if (helpEl) {
    await helpEl.click();
    report.check(`option.help-does-not-toggle/${tag}`, (await state()) === settled, 'a click on help changed nothing');
    const full = await page.evaluate((i) => { const w = document.getElementById(i).closest('.opt').querySelector('.opt-help-wrap'); const p = w.querySelector('.opt-pop'), b = w.querySelector('.opt-help'); return { hidden: p.hidden, text: p.textContent, expanded: b.getAttribute('aria-expanded'), name: b.getAttribute('aria-label'), closeBtn: !!p.querySelector('button') }; }, id);
    report.check(`option.help-full-on-click/${tag}`, !full.hidden && full.expanded === 'true' && full.text.length > 40 && full.closeBtn, `activating help shows the full explanation with a Close button (${full.text.length} characters)`);
    report.check(`option.help-named-for-its-option/${tag}`, !!full.name && full.name.includes(named), `accessible name "${full.name}"`);
    await page.keyboard.press('Escape');
    const closed = await page.evaluate((i) => { const w = document.getElementById(i).closest('.opt').querySelector('.opt-help-wrap'); return { hidden: w.querySelector('.opt-pop').hidden, focus: document.activeElement === w.querySelector('.opt-help') }; }, id);
    report.check(`option.help-escape-dismisses-returns-focus/${tag}`, closed.hidden && closed.focus, 'Escape closed the help without moving the pointer and focus returned to the help button');
    await page.mouse.move(0, 0);
    await helpEl.hover();
    const short = await page.evaluate((i) => { const p = document.getElementById(i).closest('.opt').querySelector('.opt-pop'); return { hidden: p.hidden, text: p.textContent }; }, id);
    report.check(`option.help-short-on-hover/${tag}`, !short.hidden && short.text.length > 0 && short.text.length < full.text.length, 'hover shows the short summary only, and it is shorter than the full text');
    report.check(`option.help-one-source/${tag}`, full.text.startsWith(short.text), 'the full text begins with the summary (one authored string)');
    const pb = await page.evaluate((i) => window.__uic.rect(document.getElementById(i).closest('.opt').querySelector('.opt-pop')), id);
    await page.mouse.move(pb.x + pb.w / 2, pb.y + pb.h / 2, { steps: 5 });
    await page.waitForTimeout(420);
    report.check(`option.help-stays-open-while-read/${tag}`, !(await page.evaluate((i) => document.getElementById(i).closest('.opt').querySelector('.opt-pop').hidden, id)), 'the summary stays open while the pointer is inside it');
    await page.mouse.move(0, 0);
    await page.waitForTimeout(420);
  }
  await page.focus(`#${id}`);
  const afterFocus = await state();
  await page.keyboard.press('Space');
  report.check(`option.space-toggles/${tag}`, (await state()) === !afterFocus, 'Space toggled the focused checkbox');
  report.check(`option.focus-does-not-toggle/${tag}`, afterFocus === settled, 'focusing alone changed nothing');
  await page.keyboard.press('Space');
  if (helpEl) {
    await page.focus(`#${id}`);
    await page.keyboard.press('Tab');
    report.check(`option.tab-order/${tag}`, await page.evaluate(() => document.activeElement.classList.contains('opt-help')), 'Tab from the checkbox reaches its help button');
    await page.keyboard.press('Escape');
  }
  await page.evaluate((i) => { document.getElementById(i).disabled = true; }, id);
  const dis = await state();
  await page.click(`label[for="${id}"] span:last-child`, { force: true });
  report.check(`option.disabled-no-toggle/${tag}`, (await state()) === dis, 'a disabled option does not toggle from the caption');
  await page.evaluate((i) => { document.getElementById(i).disabled = false; }, id);
}

// COLLAPSIBLE-GROUP 1.1 sections 3 and 4 on one group
export async function groupRules(page, report, tag, { id, rtl, tgt }) {
  const sm = `#${id} > summary`;
  const h = () => page.evaluate((i) => {
    const U = window.__uic, g = document.getElementById(i), s = g.querySelector('summary');
    const body = g.querySelector('.body');
    return { open: g.open, head: U.rect(s), group: U.rect(g), body: body ? U.rect(body) : null, marker: getComputedStyle(s, '::before').maskImage, transform: getComputedStyle(s, '::before').transform, bg: U.bgOf(s).join(','), weight: getComputedStyle(s).fontWeight, headContrast: U.textContrast(s), markerSize: parseFloat(getComputedStyle(s, '::before').width), markerDisplay: getComputedStyle(s, '::before').display };
  }, id);
  await page.evaluate((i) => { document.getElementById(i).open = false; }, id);
  await page.waitForTimeout(60);
  const closed = await h();
  report.check(`group.closed-body-no-space/${tag}`, approx(closed.group.h, closed.head.h + 2, 3), `collapsed group ${f1(closed.group.h)} px = its ${f1(closed.head.h)} px header`);
  const pageBg = await page.evaluate((i) => window.__uic.bgOf(document.getElementById(i).parentElement).join(','), id);
  report.check(`group.header-distinct/${tag}`, closed.bg !== pageBg && Number(closed.weight) >= 600, `header face rgb(${closed.bg}) on a rgb(${pageBg}) page, weight ${closed.weight}`);
  report.check(`group.header-contrast/${tag}`, closed.headContrast >= 4.5, `${f1(closed.headContrast)}:1`);
  report.check(`group.header-target/${tag}`, closed.head.h >= tgt - 1, `header ${f1(closed.head.h)} px high, floor ${f1(tgt)}`);
  report.check(`group.marker-is-vector/${tag}`, closed.marker && closed.marker !== 'none' && closed.markerDisplay !== 'none' && closed.markerSize >= 8, `the state triangle is a mask-drawn vector, ${f1(closed.markerSize)} px`);
  const hr = closed.head;
  await page.mouse.click(hr.x + 12, hr.y + hr.h / 2);
  report.check(`group.triangle-toggles/${tag}`, (await h()).open === true, 'a click on the triangle opened the group');
  const opened = await h();
  await page.mouse.click(hr.x + hr.w - 40, hr.y + hr.h / 2);
  report.check(`group.whitespace-toggles-once/${tag}`, (await h()).open === false, 'a click on the empty header space closed it (once)');
  await page.mouse.click(hr.x + 40, hr.y + hr.h / 2);
  report.check(`group.caption-toggles/${tag}`, (await h()).open === true, 'a click on the caption opened it');
  const body = (await h()).body;
  await page.mouse.click(body.x + body.w / 2, body.y + body.h - 4);
  report.check(`group.body-does-not-toggle/${tag}`, (await h()).open === true, 'a click inside the body did not collapse the group');
  report.check(`group.marker-state-changes/${tag}`, opened.marker !== closed.marker, 'the state triangle changes with the state');
  const exp = await h();
  report.check(`group.full-width-body/${tag}`, approx(exp.body.w, exp.group.w - 2, 2), `body ${f1(exp.body.w)} of group ${f1(exp.group.w)} px`);
  if (rtl) report.check(`group.rtl-collapsed-points-to-reading-end/${tag}`, closed.transform !== 'none', `collapsed marker transform ${closed.transform} (mirrored, pointing left)`);
  await page.focus(sm);
  await page.keyboard.press('Enter');
  report.check(`group.enter-toggles-focus-stays/${tag}`, (await h()).open === false && await page.evaluate(() => document.activeElement.tagName === 'SUMMARY'), 'Enter closed it, focus stayed on the header');
  await page.keyboard.press('Space');
  report.check(`group.space-toggles/${tag}`, (await h()).open === true, 'Space opened it');
  const child = await page.evaluate((i) => { const c = document.querySelector(`#${i} .body a, #${i} .body button`); if (c) c.focus(); return !!c; }, id);
  if (child) {
    await page.evaluate((i) => { document.getElementById(i).open = false; }, id);
    await page.waitForTimeout(80);
    report.check(`group.focus-moves-to-header/${tag}`, await page.evaluate((s) => document.activeElement === document.querySelector(s), sm), 'closing a group with a focused child moved the focus to its header');
    await page.focus(sm);
    await page.keyboard.press('Tab');
    report.check(`group.closed-children-unreachable/${tag}`, await page.evaluate((i) => !document.activeElement.closest(`#${i} > .body`), id), 'Tab leaves a collapsed group without entering its body');
    await page.evaluate((i) => { document.getElementById(i).open = true; }, id);
  }
}

// BUTTONS 3 rule 6: contrast on the hovered face too, for every enabled command button of the visible page
export async function hoverContrast(page, report, tag) {
  const centers = await page.evaluate((sel) => {
    const U = window.__uic;
    return [...document.querySelectorAll(sel)].filter((b) => U.visible(b) && !b.closest('[hidden]') && !b.disabled && !b.classList.contains('opt-help'))
      .map((b) => { const r = U.rect(b); return { x: r.x + r.w / 2, y: r.y + r.h / 2 }; });
  }, COMMAND);
  const rows = [];
  for (const c of centers) {
    await page.mouse.move(c.x, c.y);
    rows.push(await page.evaluate(([x, y]) => {
      const U = window.__uic, b = document.elementFromPoint(x, y).closest('button');
      if (!b) return null;
      const face = U.bgOf(b), ink = U.over(U.fgOf(b), face), g = b.querySelector('svg');
      return { key: b.id || b.textContent.trim().slice(0, 14), hovered: b.matches(':hover'), text: U.contrast(ink, face), glyph: g ? U.contrast(U.over(U.fgOf(g, 'color'), face), face) : null };
    }, [c.x, c.y]));
  }
  await page.mouse.move(0, 0);
  const seen = rows.filter(Boolean);
  const low = seen.filter((x) => x.text < 4.5 || (x.glyph !== null && x.glyph < 3));
  report.check(`contrast.hover/${tag}`, seen.length > 0 && seen.every((x) => x.hovered) && low.length === 0, low.length ? low.map((x) => `${x.key} ${f1(x.text)}/${f1(x.glyph ?? 0)}`).join('; ') : `${seen.length} hovered buttons, lowest caption ${f1(Math.min(...seen.map((x) => x.text)))}:1`);
}

// UI-SIZING 1.1 section 5 and BUTTONS 3 rule 3: a stateful button keeps the width of its widest state, so the neighbours
// do not move while a language downloads
export async function stateWidth(page, report, tag) {
  const r = await page.evaluate(() => {
    const U = window.__uic, btn = [...document.querySelectorAll('.ocr-lang button')].find((b) => U.visible(b));
    if (!btn) return null;
    const cap = btn.querySelector('.cap'), name = btn.parentElement.querySelector('.name');
    const keep = cap.textContent;
    const before = { w: U.rect(btn).w, x: U.rect(btn).x, name: U.rect(name).x };
    // the progress text of the page's own language at a shorter percentage: the reserve is its longest form
    cap.textContent = (cap.dataset.reserve || 'Downloading 100%').replace('100%', '57%');
    const after = { w: U.rect(btn).w, x: U.rect(btn).x, name: U.rect(name).x, h: U.rect(btn).h };
    cap.textContent = keep;
    return { before, after };
  });
  if (!r) return;
  report.check(`buttons.state-width/${tag}`, approx(r.before.w, r.after.w, 0.6) && approx(r.before.x, r.after.x, 0.6) && approx(r.before.name, r.after.name, 0.6), `Download ${f1(r.before.w)} px, Downloading 57% ${f1(r.after.w)} px; the button and its row neighbour did not move`);
}