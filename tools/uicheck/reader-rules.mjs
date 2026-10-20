// The rules of BUTTONS 2.0 and UI-SIZING 1.1 that a reader chrome's measured geometry must satisfy (ticket 119,
// slice R1). Pure functions over what reader-measure.mjs read from the engine: they take the measurement and the
// profile (coarse or fine pointer, interface scale) and return one row per rule with the measured evidence, so a
// failure names the control and the number. The same rules judge the generated reader and the extension viewer.

const f1 = (n) => (Math.round(n * 10) / 10).toString();
const label = (c) => c.key + (c.caption ? ' "' + c.caption.slice(0, 16) + '"' : '');

// The tokens of the profile: ordinary 32 / 48 targets, radius 4, border 1, gap 8, glyph tier 16 / 20, type 14 / 12,
// all multiplied by the clamped interface scale (UI-SIZING 2 rule 2).
export function tokens(coarse, scale) {
  const s = Math.min(2, Math.max(0.875, Number.isFinite(scale) ? scale : 1));
  return { s, tgt: (coarse ? 48 : 32) * s, radius: 4 * s, gap: 8 * s, glyph: (coarse ? 20 : 16) * s, fs: 14 * s, fsNote: 12 * s, pad: (coarse ? 16 : 12) * s };
}

// bands groups the controls into the visual rows the engine made: a control joins a band when its centre lies inside
// the band's vertical extent, so a row whose members are centred at different heights (not equalized) is one band
// with unequal heights - which the height rule then reports - instead of two clean-looking bands.
export function bands(m) {
  const out = [];
  for (const c of [...m.controls].sort((a, b) => a.y - b.y || a.x - b.x)) {
    const cy = c.y + c.h / 2;
    const b = out.find((x) => cy > x.top + 1 && cy < x.bottom - 1);
    if (b) { b.controls.push(c); b.top = Math.min(b.top, c.y); b.bottom = Math.max(b.bottom, c.y + c.h); } else out.push({ top: c.y, bottom: c.y + c.h, controls: [c], items: [] });
  }
  for (const it of m.items) {
    const cy = it.y + it.h / 2;
    const b = out.find((x) => cy > x.top - 1 && cy < x.bottom + 1);
    if (b) b.items.push(it);
  }
  return out.sort((a, b) => a.top - b.top);
}

export function geometryRules(m, { coarse, scale = 1, versionKey = '.nav-version', rtl = false }) {
  const T = tokens(coarse, scale), rows = [];
  const add = (id, ok, detail) => rows.push({ id, ok, detail });
  const C = m.controls;
  add('profile', m.coarse === coarse, `(pointer: coarse) is ${m.coarse}, expected ${coarse}`);
  add('has-controls', C.length >= 8, `${C.length} controls measured`);

  const small = C.filter((c) => c.h < T.tgt - 0.5 || c.w < T.tgt - 0.5);
  add('target', small.length === 0, small.length ? small.map((c) => `${label(c)} ${f1(c.w)}x${f1(c.h)}`).join('; ') + ` < ${T.tgt}` : `every control >= ${f1(T.tgt)} x ${f1(T.tgt)} (smallest ${f1(Math.min(...C.map((c) => c.h)))} high, ${f1(Math.min(...C.map((c) => c.w)))} wide)`);

  const bad = C.filter((c) => c.radiusPct || c.radius > T.radius + 0.5 || c.radius >= c.h / 2 - 1);
  add('radius', bad.length === 0, bad.length ? bad.map((c) => `${label(c)} radius ${c.radius}${c.radiusPct ? '%' : ''}`).join('; ') : `radius <= ${f1(T.radius)}, no pill (largest ${Math.max(...C.map((c) => c.radius))})`);

  const nob = C.filter((c) => !/nav-version|proj-link/.test(c.key) && c.bw < 1);
  add('border-1px', nob.length === 0, nob.length ? nob.map(label).join('; ') + ' have no border' : 'every button, link-button and select has its 1 px border');

  const stretched = C.filter((c) => c.w > Math.max(c.natural, T.tgt) + 1);
  add('content-width', stretched.length === 0, stretched.length ? stretched.map((c) => `${label(c)} ${f1(c.w)} > content ${f1(c.natural)}`).join('; ') : 'no control wider than its content (or the target minimum)');

  const clipped = C.filter((c) => c.clipX || c.clipY);
  add('caption-not-clipped', clipped.length === 0, clipped.length ? clipped.map(label).join('; ') + ' clip' : 'scrollWidth/Height within the box for every button and link');

  const B = bands(m);
  const uneven = B.map((b) => ({ b, d: Math.max(...b.controls.map((c) => c.h)) - Math.min(...b.controls.map((c) => c.h)) })).filter((x) => x.d > 1.5);
  add('row-height-equal', uneven.length === 0, uneven.length ? uneven.map((x) => `row at y=${f1(x.b.top)}: ${f1(x.d)} px spread`).join('; ') : `${B.length} visual row(s), each one height`);

  const gapBad = [];
  for (const b of B) {
    const mem = [...b.controls, ...b.items].sort((p, q) => (rtl ? q.x + q.w - (p.x + p.w) : p.x - q.x));
    for (let i = 1; i < mem.length; i++) {
      const p = mem[i - 1], q = mem[i];
      const g = rtl ? p.x - (q.x + q.w) : q.x - (p.x + p.w);
      if (Math.abs(g - T.gap) > 0.75) gapBad.push(`${label(p)} -> ${label(q)} ${f1(g)}`);
    }
  }
  add('gap-8', gapBad.length === 0, gapBad.length ? gapBad.slice(0, 6).join('; ') + ` (token ${f1(T.gap)})` : `every neighbour in a row is ${f1(T.gap)} apart`);

  const rowGapBad = [];
  for (let i = 1; i < B.length; i++) { const g = B[i].top - B[i - 1].bottom; if (Math.abs(g - T.gap) > 0.75) rowGapBad.push(`rows ${i - 1}/${i} ${f1(g)}`); }
  add('row-gap-8', rowGapBad.length === 0, rowGapBad.length ? rowGapBad.join('; ') + ` (token ${f1(T.gap)})` : `${Math.max(0, B.length - 1)} wrapped-row gap(s) of ${f1(T.gap)}`);

  // Rows flow from the reading start and the last line aligns to the group start: the first member of every row sits on
  // the bar's inline-start content edge, so nothing is pinned to the far end and no row is centred or justified.
  const root = m.roots[0];
  if (root) {
    const startBad = [];
    for (const b of B) {
      const mem = [...b.controls, ...b.items];
      const edge = rtl ? Math.max(...mem.map((x) => x.x + x.w)) : Math.min(...mem.map((x) => x.x));
      const want = rtl ? root.x + root.w - root.br - root.pr : root.x + root.bl + root.pl;
      if (Math.abs(edge - want) > 1.5) startBad.push(`row y=${f1(b.top)} starts ${f1(edge)}, bar content edge ${f1(want)}`);
    }
    add('flow-from-start', startBad.length === 0, startBad.length ? startBad.join('; ') : `every row starts on the bar's ${rtl ? 'right' : 'left'} content edge`);
    add('bar-no-inner-overflow', root.scrollW <= root.clientW + 1, `bar scrollWidth ${root.scrollW}, clientWidth ${root.clientW}`);
    const inside = C.filter((c) => c.x < root.x - 1 || c.x + c.w > root.x + root.w + 1);
    add('controls-inside-bar', inside.length === 0, inside.length ? inside.map(label).join('; ') + ' outside the bar' : 'every control inside the bar box');
  }

  const overlap = [];
  for (let i = 0; i < C.length; i++) for (let j = i + 1; j < C.length; j++) {
    const a = C[i], b = C[j];
    const w = Math.min(a.x + a.w, b.x + b.w) - Math.max(a.x, b.x), h = Math.min(a.y + a.h, b.y + b.h) - Math.max(a.y, b.y);
    if (w > 1 && h > 1) overlap.push(`${label(a)} x ${label(b)}`);
  }
  add('no-overlap', overlap.length === 0, overlap.length ? overlap.join('; ') : 'no two interactive targets overlap');

  const glyphBad = C.filter((c) => c.glyph && (Math.abs(c.glyph.w - T.glyph) > 0.5 || Math.abs(c.glyph.h - T.glyph) > 0.5));
  add('glyph-tier', glyphBad.length === 0, glyphBad.length ? glyphBad.map((c) => `${label(c)} ${f1(c.glyph.w)}`).join('; ') + ` != ${f1(T.glyph)}` : `every glyph ${f1(T.glyph)} px`);
  const iconNoGlyph = C.filter((c) => c.icon && !c.glyph);
  add('icon-only-has-glyph', iconNoGlyph.length === 0, iconNoGlyph.length ? iconNoGlyph.map(label).join('; ') : 'every caption-less button draws its glyph');

  const fsBad = C.filter((c) => Math.abs(c.fs - T.fs) > 0.5 && Math.abs(c.fs - T.fsNote) > 0.5);
  add('type-scale', fsBad.length === 0, fsBad.length ? fsBad.map((c) => `${label(c)} ${c.fs}px`).join('; ') : `controls at ${f1(T.fs)} px (the version link at ${f1(T.fsNote)})`);

  const unnamed = C.filter((c) => (c.icon || c.kind === 'select' || c.kind === 'input') && !c.ariaLabel);
  add('glyph-only-named', unnamed.length === 0, unnamed.length ? unnamed.map(label).join('; ') + ' have no accessible name' : 'every glyph-only control and select has an accessible name');
  const noTip = C.filter((c) => c.icon && !c.title);
  add('glyph-only-tooltip', noTip.length === 0, noTip.length ? noTip.map(label).join('; ') + ' have no tooltip' : 'every glyph-only button has a tooltip');

  const ext = C.filter((c) => c.target === '_blank');
  add('external-link-semantics', ext.every((c) => c.kind === 'link' && c.href), ext.map(label).join('; ') || 'no external link in the bar');
  return rows;
}
