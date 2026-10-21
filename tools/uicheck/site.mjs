// The official site against BUTTONS 2.0, COLLAPSIBLE-GROUP 1.1 and UI-SIZING 1.1 (ticket 119, slice W1), measured on
// the working tree as GitHub Pages serves it (tools/sitecheck's static server), in a headless Chromium with a fine
// pointer and a coarse one. The kit (assets/sza-kit.css) is catalog-owned and never edited; this checks what the page
// layer (assets/site.css) and the pages make of it.
import { Report, readerSource } from './lib.mjs';
import { pathToFileURL } from 'node:url';
import path from 'node:path';
import fs from 'node:fs';
import { REPO } from './lib.mjs';

const sitecheck = await import(pathToFileURL(path.join(REPO, 'tools', 'sitecheck', 'lib.mjs')).href);

const PAGES = ['index.html', 'docs.html', 'docs.ru.html', 'docs.uk.html', 'extension.html', 'privacy.html', 'support.html', '404.html'];
const BUTTONISH = '.btn, .seg a, .seg button, .theme-btn, .copybox .copy, .to-top';

export async function siteChecks(report) {
  const { server, base } = await sitecheck.serveTree();
  const browser = await sitecheck.launch();
  try {
    for (const mode of ['fine', 'coarse']) {
      const tgt = mode === 'coarse' ? 48 : 44;
      const ctx = await browser.newContext(mode === 'coarse'
        ? { viewport: { width: 390, height: 844 }, hasTouch: true, isMobile: true, deviceScaleFactor: 2 }
        : { viewport: { width: 1280, height: 900 } });
      for (const f of PAGES) {
        const page = await ctx.newPage();
        // the 404 page names its stylesheets by absolute address (it must work from any path); the check serves
        // the working tree's own files for those, so what is measured is this tree, not the published site
        await page.route(sitecheck.LIVE_BASE + '**', async (route) => {
          const rel = new URL(route.request().url()).pathname.replace('/doc-html-translate/', '');
          const file = path.join(REPO, rel);
          if (fs.existsSync(file) && fs.statSync(file).isFile()) await route.fulfill({ path: file }); else await route.abort();
        });
        await page.goto(base + f, { waitUntil: 'load' });
        await page.evaluate(readerSource);
        const tag = `${f}/${mode}`;
        const m = await page.evaluate((sel) => {
          const U = window.__uic, out = [];
          for (const e of document.querySelectorAll(sel)) {
            if (!U.visible(e)) continue;
            const r = U.rect(e), cs = getComputedStyle(e);
            const after = getComputedStyle(e, '::after');
            out.push({ k: (e.className || e.tagName) + ' ' + (e.textContent || '').trim().slice(0, 18), w: r.w, h: r.h, radius: parseFloat(cs.borderTopLeftRadius), pill: cs.borderTopLeftRadius.includes('%'), shadow: cs.boxShadow, blank: e.target === '_blank' && e.classList.contains('btn'), marker: after.content, markerW: parseFloat(after.width) });
          }
          return { items: out, coarse: matchMedia('(pointer: coarse)').matches, overflow: document.documentElement.scrollWidth > window.innerWidth + 1 };
        }, BUTTONISH);
        report.check(`site.profile/${tag}`, m.coarse === (mode === 'coarse'), `(pointer: coarse) is ${m.coarse}`);
        for (const x of m.items) {
          report.check(`site.target/${tag} [${x.k}]`, x.h >= tgt - 1 && x.w >= Math.min(tgt, x.w) - 1 && (x.w >= tgt - 1 || !/theme-btn|to-top|seg/.test(x.k)), `${x.w.toFixed(0)} x ${x.h.toFixed(0)} px, floor ${tgt}`);
          report.check(`site.radius/${tag} [${x.k}]`, !x.pill && x.radius <= 4.5, `radius ${x.radius}`);
          report.check(`site.no-glow/${tag} [${x.k}]`, !/rgba?\(.*\) \d+px \d+px \d+px/.test(x.shadow) || x.shadow === 'none', `shadow ${x.shadow.slice(0, 40)}`);
          if (x.blank) report.check(`site.external-marker/${tag} [${x.k}]`, x.marker !== 'none' && x.markerW > 8, `marker content ${x.marker}, ${x.markerW}px`);
        }
        report.check(`site.no-horizontal-overflow/${tag}`, !m.overflow, 'no horizontal page scroll');
        // command rows flow from the reading start with the 8 px gap
        const rows = await page.evaluate(() => [...document.querySelectorAll('.button-group, .channels, .sec-tools, .actions')].filter((r) => window.__uic.visible(r)).map((r) => { const cs = getComputedStyle(r); return { id: r.className, jc: cs.justifyContent, gap: parseFloat(cs.columnGap) }; }));
        for (const r of rows) report.check(`site.row-flow/${tag} [${r.id}]`, /flex-start|normal|start/.test(r.jc) && r.gap === 8, `justify ${r.jc}, gap ${r.gap}`);
        // disclosure groups
        const groups = await page.evaluate(() => [...document.querySelectorAll('details.sec')].filter((d) => window.__uic.visible(d)).slice(0, 3).map((d) => {
          const U = window.__uic, sm = d.querySelector('summary');
          return { id: d.id || sm.textContent.trim().slice(0, 20), head: U.rect(sm), beforeMask: getComputedStyle(sm, '::before').maskImage, afterContent: getComputedStyle(sm, '::after').content, face: U.bgOf(sm).join(','), bodyFace: U.bgOf(d.parentElement).join(','), headContrast: U.textContrast(sm) };
        }));
        for (const g of groups) {
          report.check(`group.marker-vector-leading/${tag} [${g.id}]`, g.beforeMask && g.beforeMask !== 'none' && (g.afterContent === 'none' || g.afterContent === 'normal'), `mask ${String(g.beforeMask).slice(0, 30)}, ::after ${g.afterContent}`);
          report.check(`group.header-target/${tag} [${g.id}]`, g.head.h >= tgt - 1, `${g.head.h.toFixed(0)} px`);
          report.check(`group.header-contrast/${tag} [${g.id}]`, g.headContrast >= 4.5, `${g.headContrast.toFixed(2)}:1`);
        }
        if (groups.length) {
          const t = await page.evaluate(() => {
            const d = [...document.querySelectorAll('details.sec')].find((x) => window.__uic.visible(x)); d.id = d.id || 'uic-probe'; const sm = d.querySelector('summary');
            const r = sm.getBoundingClientRect(); const before = d.open;
            sm.scrollIntoView({ block: 'center', behavior: 'instant' });
            const r2 = sm.getBoundingClientRect();
            return { id: d.id, before, x: r2.x + r2.width - 12, y: r2.y + r2.height / 2, w: r2.width };
          });
          await page.mouse.click(t.x, t.y);
          const after = await page.evaluate((id) => document.getElementById(id).open, t.id);
          report.check(`group.whitespace-toggles/${tag}`, after !== t.before, 'a click on the empty header space toggled the group');
          if (f === 'extension.html') {
            const stacked = await page.evaluate(() => { const s = document.querySelector('.sections'); return getComputedStyle(s).display; });
            report.check(`group.no-equal-tiles/${tag}`, stacked !== 'grid', `.sections is ${stacked}`);
          }
        }
        // a language switch is link semantics where it navigates (docs pages)
        if (/^docs/.test(f)) {
          const links = await page.evaluate(() => [...document.querySelectorAll('.seg [data-set-lang]')].map((e) => ({ tag: e.tagName, href: e.getAttribute('href'), cur: e.getAttribute('aria-current') })));
          report.check(`nav.language-links/${tag}`, links.length === 3 && links.every((l) => l.tag === 'A' && l.href) && links.filter((l) => l.cur === 'true').length === 1, JSON.stringify(links));
        }
        await page.close();
      }
      await ctx.close();
    }
  } finally { await browser.close(); server.close(); }
}
