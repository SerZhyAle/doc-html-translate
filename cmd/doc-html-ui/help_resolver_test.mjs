import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import {createRequire} from 'node:module';
import {test} from 'node:test';

// help.js is the window's one resolver (CONTEXT-HELP rule 17); under node it exports its pure functions.
const help = createRequire(import.meta.url)('./help.js');
const map = JSON.parse(readFileSync(new URL('./help/help-map.json', import.meta.url), 'utf8'));

test('the main window resolves to the converter page, a region to its anchor', () => {
    assert.deepEqual(help.resolve(map, 'main', '', 'en'), {surface: 'main', path: '/help/en/converter.html', anchor: '', locale: 'en'});
    const drop = help.resolve(map, 'main', 'main.drop', 'en');
    assert.equal(drop.path, '/help/en/converter.html');
    assert.equal(drop.anchor, 'drop');
    // settings groups live in the same window but answer with the settings page
    const ocr = help.resolve(map, 'main', 'settings.ocr', 'ru');
    assert.equal(ocr.path, '/help/ru/settings.html');
    assert.equal(ocr.anchor, 'ocr');
});

test('a dialog resolves to its own page, and a confirmation to nothing', () => {
    assert.equal(help.resolve(map, 'dlg-cost', '', 'uk').path, '/help/uk/cost-question.html');
    assert.equal(help.resolve(map, 'dlg-secret', '', 'en').path, '/help/en/secret-file-password.html');
    assert.equal(help.resolve(map, 'dlg-run-failed', '', 'ru').path, '/help/ru/run-failed.html');
    assert.equal(help.resolve(map, 'dlg-confirm', '', 'en'), null);
    assert.equal(help.resolve(map, 'dlg-notice', '', 'en'), null);
    assert.equal(help.resolve(map, 'os-file-picker', '', 'en'), null);
    assert.equal(help.resolve(map, 'no-such-surface', '', 'en'), null);
});

test('a language without pages falls back to the source language, never to a missing page', () => {
    for (const lang of ['de', 'ar', 'zh', 'pt-BR', '', undefined]) {
        const r = help.resolve(map, 'main', '', lang);
        assert.equal(r.locale, 'en', `${lang} must fall back to en`);
        assert.equal(r.path, '/help/en/converter.html');
    }
    assert.equal(help.resolve(map, 'main', '', 'ru-RU').locale, 'ru');
    assert.equal(help.resolveIndex(map, 'uk').path, '/help/uk/index.html');
    assert.equal(help.resolveIndex(map, 'ja').path, '/help/en/index.html');
});

test('an inheriting row answers with its parent, keeping its own anchor', () => {
    const m = {helpRoot: 'help', pageTemplate: '{locale}/{help}.html', index: 'index', sourceLocale: 'en', locales: ['en'], surfaces: [
        {id: 'main', form: 'page', help: 'main'},
        {id: 'rename', form: 'inherit', parent: 'main', anchor: 'rename', reason: 'a one-field dialog'},
        {id: 'loop', form: 'inherit', parent: 'loop', reason: 'a cycle must not hang'},
    ]};
    const r = help.resolve(m, 'rename', '', 'en');
    assert.equal(r.path, '/help/en/main.html');
    assert.equal(r.anchor, 'rename');
    assert.equal(help.resolve(m, 'loop', '', 'en'), null);
});

test('the address carries the pinned theme and the anchor, and nothing else', () => {
    const r = help.resolve(map, 'main', 'main.output', 'en');
    assert.equal(help.targetUrl(r, 'dark'), '/help/en/converter.html?theme=dark#output');
    assert.equal(help.targetUrl(r, ''), '/help/en/converter.html#output');
    assert.equal(help.targetUrl(r, 'system'), '/help/en/converter.html#output');
    assert.equal(help.targetUrl(null, 'dark'), null);
});

test('F1 is bare F1: a chord belongs to the browser', () => {
    assert.equal(help.isHelpKey({key: 'F1'}), true);
    for (const mod of ['ctrlKey', 'altKey', 'shiftKey', 'metaKey']) assert.equal(help.isHelpKey({key: 'F1', [mod]: true}), false, mod);
    assert.equal(help.isHelpKey({key: 'F2'}), false);
});

test('the key asks about the dialog in front, else about the region the focus is in', () => {
    const region = {dataset: {help: 'main.languages'}};
    const doc = (dlg, active) => ({
        querySelector: sel => (sel === 'dialog[open]' ? dlg : null),
        activeElement: active,
    });
    assert.deepEqual(help.contextOf(doc({dataset: {surface: 'dlg-cost'}}, region)), {surface: 'dlg-cost', element: ''});
    assert.deepEqual(help.contextOf(doc(null, {closest: sel => (sel === '[data-help]' ? region : null)})), {surface: 'main', element: 'main.languages'});
    assert.deepEqual(help.contextOf(doc(null, {closest: () => null})), {surface: 'main', element: ''});
    assert.deepEqual(help.contextOf(doc(null, null)), {surface: 'main', element: ''});
});

// The key and the button resolve through the same table: every region the window marks, and every surface it
// can show a button for, has a target - nothing the button opens is something the key does not.
test('every region of the map resolves in every language the set holds', () => {
    for (const s of map.surfaces) {
        for (const lang of map.locales) {
            if (s.form === 'none') { assert.equal(help.resolve(map, s.id, '', lang), null, s.id); continue; }
            assert.ok(help.resolve(map, s.id, '', lang), `${s.id} ${lang}`);
            for (const p of s.pages || []) {
                const r = help.resolve(map, s.id, p.id, lang);
                assert.equal(r.anchor, p.anchor, p.id);
                assert.equal(r.surface, s.id, p.id);
            }
        }
    }
});
