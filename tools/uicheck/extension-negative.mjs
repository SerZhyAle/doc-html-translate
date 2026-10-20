// Negative controls (ticket 119, slice X1): each one breaks a real page of the extension on purpose - a stretched button, a
// pill corner, a washed-out explanation, a checkbox with no hit area, a sliding switch that came back - and runs the same
// rules that PASS on the unbroken page. The rule list is only evidence if it can fail: a control the rules do not catch is a
// failed check here, not a quiet pass. Nothing is written to the extension; the break lives in the page under test.
import { Report } from './lib.mjs';
import { barCase } from './extension-bar.mjs';

const OPTIONS = { name: 'options', size: 'ordinary', input: 'pointer', w: 1280, h: 900 };
const POPUP = { name: 'popup', size: 'ordinary', input: 'pointer', enabled: true };
const BAR = { size: 'ordinary', input: 'pointer', w: 1100, h: 700 };

// runner: which real scenario runs; expect: the rule id that must fail
const CONTROLS = [
  { name: 'a stretched command button', runner: 'options', c: { ...OPTIONS, breakCss: '#export-settings { flex: 1 1 100% !important; width: 100% !important; }' }, expect: 'buttons.content-width' },
  { name: 'a pill corner', runner: 'options', c: { ...OPTIONS, breakCss: 'button { border-radius: 16px !important; }' }, expect: 'buttons.radius' },
  { name: 'a target below the floor', runner: 'options', c: { ...OPTIONS, breakCss: '#export-settings { min-height: 18px !important; height: 18px !important; padding: 0 12px !important; }' }, expect: 'buttons.target-height' },
  { name: 'unequal heights in one visual row', runner: 'options', c: { ...OPTIONS, breakCss: '#page-data .btn-row { align-items: flex-start !important; } #import-settings { height: 80px !important; }' }, expect: 'buttons.row-height' },
  { name: 'a wrong gap between buttons', runner: 'options', c: { ...OPTIONS, breakCss: '#page-data .btn-row { gap: 30px !important; }' }, expect: 'buttons.gap' },
  { name: 'a clipped caption', runner: 'options', c: { ...OPTIONS, breakCss: '#reset-settings { width: 40px !important; min-width: 0 !important; overflow: hidden !important; white-space: nowrap !important; overflow-wrap: normal !important; }' }, expect: 'buttons.no-clip' },
  { name: 'overlapping targets', runner: 'options', c: { ...OPTIONS, breakCss: '#import-settings { margin-inline-start: -60px !important; }' }, expect: 'targets.no-overlap' },
  { name: 'horizontal overflow of the pane', runner: 'options', c: { ...OPTIONS, breakCss: '#page-data { width: 4000px !important; flex: none !important; }' }, expect: 'page.no-horizontal-overflow' },
  { name: 'a rail row below the target', runner: 'options', c: { ...OPTIONS, breakCss: '.rail-item { min-height: 12px !important; padding: 0 12px !important; }' }, expect: 'rail.rows-target' },
  { name: 'a washed-out explanation', runner: 'options-deep', c: { ...OPTIONS, deep: true, breakCss: '.hint, .opt-expl { color: #cfcfcf !important; }' }, expect: 'contrast.text' },
  { name: 'a glyph that vanishes into its face', runner: 'options-deep', c: { ...OPTIONS, deep: true, breakCss: 'button .glyph { color: #fdfdfd !important; }' }, expect: 'contrast.glyph' },
  { name: 'a faint checkbox outline', runner: 'options-deep', c: { ...OPTIONS, deep: true, breakCss: '.opt-box::before { border-color: #e4e4e4 !important; }' }, expect: 'contrast.checkbox-outline' },
  { name: 'a checkbox with no hit area', runner: 'option-row', c: { ...OPTIONS, breakCss: '.opt-box { padding: 0 !important; margin: 0 !important; }' }, expect: 'option.hit-target' },
  { name: 'a checkbox drawn too small', runner: 'option-row', c: { ...OPTIONS, breakCss: '.opt-box::before { inset: 10px !important; }' }, expect: 'option.box-size' },
  { name: 'a checkbox off its first line', runner: 'option-row', c: { ...OPTIONS, breakCss: '.opt-box { margin-top: 30px !important; }' }, expect: 'option.checkbox-first-line' },
  { name: 'a group header that does not stand out', runner: 'groups', c: { ...OPTIONS, breakCss: '.group > summary { background: transparent !important; font-weight: 400 !important; }' }, expect: 'group.header-distinct' },
  { name: 'a group with no state triangle', runner: 'groups', c: { ...OPTIONS, breakCss: '.group > summary::before { display: none !important; }' }, expect: 'group.marker-is-vector' },
  { name: 'a sliding switch that came back', runner: 'popup', c: { ...POPUP, breakJs: () => { const s = document.createElement('span'); s.className = 'slider'; document.body.append(s); } }, expect: 'popup.no-sliding-switch' },
  { name: 'a full-width call to action', runner: 'popup', c: { ...POPUP, breakCss: '#open-pdf { width: 100% !important; display: flex !important; } .btn-row { display: block !important; }' }, expect: 'buttons.content-width' },
  { name: 'an outside link without its marker', runner: 'popup', c: { ...POPUP, breakJs: () => { document.querySelector('#help [data-glyph]').replaceChildren(); } }, expect: 'popup.external-links-marked' },
  { name: 'Settings as a dead link again', runner: 'popup', c: { ...POPUP, breakJs: () => { const a = document.createElement('a'); a.href = '#'; a.id = 'opts'; a.textContent = 'Settings'; document.getElementById('opts').replaceWith(a); } }, expect: 'popup.settings-is-a-button' },
  { name: 'a stateful button with no width reserve', runner: 'options', c: { ...OPTIONS, breakCss: '.cap[data-reserve]::after { content: none !important; }' }, expect: 'buttons.state-width' },
  { name: 'a hovered face that loses its contrast', runner: 'options-deep', c: { ...OPTIONS, deep: true, breakCss: 'button:hover { background: #e9e9e9 !important; color: #f4f4f4 !important; }' }, expect: 'contrast.hover' },
  { name: 'a bar button that jumps when its caption changes', runner: 'bar', c: { ...BAR, states: true, breakCss: '#dht-ocr-bar .dht-ocr-cap[data-reserve]::after { content: none !important; }' }, expect: 'bar.state-width-reserved' },  { name: 'a bar button with a pill corner', runner: 'bar', c: { ...BAR, breakCss: '#dht-ocr-bar .dht-ocr-btn { border-radius: 20px !important; }' }, expect: 'bar.radius' },
  { name: 'a stretched bar button', runner: 'bar', c: { ...BAR, breakCss: '#dht-ocr-bar .dht-ocr-btn { flex: 1 1 100% !important; }' }, expect: 'bar.content-width' },
  { name: 'a bar button below the target', runner: 'bar', c: { ...BAR, breakCss: '#dht-ocr-bar .dht-ocr-btn { min-height: 14px !important; padding: 0 12px !important; }' }, expect: 'bar.target-height' },
];

export async function negativeControls(browser, base, report, { openPage, storedFor, runners }) {
  const run = {
    options: (r, c) => runners.optionsCase(browser, base, r, c, null),
    'options-deep': (r, c) => runners.optionsCase(browser, base, r, c, null),
    'option-row': (r, c) => runners.optionRowCases(browser, base, r, c),
    groups: (r, c) => runners.groupCases(browser, base, r, c),
    popup: (r, c) => runners.popupCase(browser, base, r, c, null),
    bar: (r, c) => barCase(browser, base, r, c, { openPage, storedFor }),
  };
  for (const ctl of CONTROLS) {
    const scratch = new Report('negative');
    const c = { ...ctl.c };
    try { await run[ctl.runner](scratch, c); } catch (e) { scratch.check('negative.harness-error', false, String(e.message).split('\n')[0]); }
    const hit = scratch.failed.find((r) => r.id.startsWith(ctl.expect));
    report.check(`negative.caught/${ctl.name}`, !!hit, hit ? `the break failed ${hit.id.split('/')[0]}: ${hit.detail.slice(0, 110)}` : `the break went UNDETECTED (${scratch.failed.length} other failures: ${scratch.failed.slice(0, 2).map((r) => r.id.split('/')[0]).join(', ')})`);
  }
}
