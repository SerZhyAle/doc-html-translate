// SITE-CHECKLIST run with real browser input (ticket 102).
//   node run.mjs [--live | --tree] [--only A,B,E,H,I0,K,M,T,R,X] [--out <dir>]
// --live (default) reads the published site; --tree serves the working tree like Pages does. Every check writes
// verdict records {box, group, status, note, data}; status is pass | fail | na | info | manual | error.
// The shared state is built in env.mjs; the checks live in sections/*.mjs, one exported function per section.
import fs from 'node:fs';
import path from 'node:path';
import { REPO, git, ts } from './lib.mjs';
import { createEnv } from './env.mjs';
import { deployment, siteCrawl, landing, heldAddresses, versionStrings } from './sections/site.mjs';
import { renderedStructure, themeResolver, accessibilityTree } from './sections/structure.mjs';
import { languageControl, privacyOrigins, editionsAndChrome, manualStructure } from './sections/content.mjs';
import { layout, keyboard, reducedMotion, contrast } from './sections/rendering.mjs';

const args = process.argv.slice(2);
const flag = (n) => args.includes(n);
const opt = (n, d) => { const i = args.indexOf(n); return i >= 0 ? args[i + 1] : d; };
const MODE = flag('--tree') ? 'tree' : 'live';
const ONLY = (opt('--only', '') || '').split(',').filter(Boolean);
const want = (k) => !ONLY.length || ONLY.includes(k);
const OUT = path.resolve(opt('--out', path.join(REPO, 'temp', 'site-run', `${ts()}-${MODE}`)));
fs.mkdirSync(path.join(OUT, 'shots'), { recursive: true });

const env = await createEnv({ MODE, OUT, want });
const { R, BASE, browser, server } = env;

// The order is the order of the verdicts in the console and in results.json; do not reorder.
if (want('S0')) await deployment(env);
if (want('A') || want('E') || want('H')) await siteCrawl(env);
if (want('A')) await landing(env);
if (want('B')) await renderedStructure(env);
if (want('B')) await themeResolver(env);
if (want('E')) await languageControl(env);
if (want('H')) await heldAddresses(env);
if (want('I0') || want('L')) await layout(env);
if (want('K')) await keyboard(env);
if (want('M')) await reducedMotion(env);
if (want('T')) await contrast(env);
if (want('R')) await accessibilityTree(env);
if (want('F')) await privacyOrigins(env);
if (want('G')) await editionsAndChrome(env);
if (want('C')) await manualStructure(env);
if (want('X')) await versionStrings(env);

fs.writeFileSync(path.join(OUT, 'results.json'), JSON.stringify({ mode: MODE, base: BASE, at: new Date().toISOString(), ref: git('rev-parse', '--short', 'origin/main'), head: git('rev-parse', '--short', 'HEAD'), results: R }, null, 1));
fs.writeFileSync(path.join(OUT, 'REPORT.md'), `# Site checklist run - ${MODE}\n\nBase ${BASE}; origin/main ${git('rev-parse', '--short', 'origin/main')}; HEAD ${git('rev-parse', '--short', 'HEAD')}; ${new Date().toISOString()}\n\n| Box | Group | Status | Note |\n| --- | --- | --- | --- |\n` + R.map((r) => `| ${r.box} | ${r.group} | ${r.status} | ${r.note.replace(/\|/g, '/').replace(/\n/g, ' ')} |`).join('\n') + '\n');
const cnt = R.reduce((a, r) => ((a[r.status] = (a[r.status] || 0) + 1), a), {});
console.log('\nDONE', JSON.stringify(cnt), '->', OUT);
await browser.close();
if (server) server.close();
