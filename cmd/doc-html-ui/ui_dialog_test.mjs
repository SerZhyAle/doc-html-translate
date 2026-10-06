import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import {test} from 'node:test';
import vm from 'node:vm';

const page = readFileSync(new URL('./ui.html', import.meta.url), 'utf8');
function section(start, end) {
    const a = page.indexOf(start), b = page.indexOf(end, a);
    assert(a >= 0 && b > a, `missing page section ${start}`);
    return page.slice(a, b);
}

function deferred() {
    let resolve;
    const promise = new Promise(r => { resolve = r; });
    return {promise, resolve};
}

function harness() {
    const opened = [];
    const answers = [];
    const streams = new Map();
    const listeners = new Map();
    const dlg = {
        returnValue: '',
        addEventListener(event, fn) { listeners.set(event, fn); },
        showModal() { opened.push(title.textContent); },
        close() { listeners.get('close')(); },
    };
    const title = {textContent: ''};
    const message = {textContent: ''};
    const actions = {innerHTML: '', lastChild: null, appendChild(btn) { this.lastChild = btn; }};
    const elements = {dlg, dlgTitle: title, dlgMsg: message, dlgActions: actions};
    const context = vm.createContext({
        Promise, AbortController, TextDecoder, console,
        el: id => elements[id],
        document: {createElement: () => ({focus() {}})},
        t: key => key,
        api: async (path, opts) => {
            if (path === '/api/answer') { answers.push(JSON.parse(opts.body)); return {}; }
            assert.equal(path, '/api/run');
            const cfg = JSON.parse(opts.body);
            return {ok: true, body: {getReader: () => ({read: () => streams.get(cfg.input).promise})}};
        },
    });
    vm.runInContext(section('let dlgQueue = Promise.resolve();', '// informDialog shows'), context);
    vm.runInContext('const MARK = "\\x1edht:";\n' +
        section('async function streamRun(cfg, hooks)', 'async function runConvert()') +
        section('async function askForTheRun(cfg, q, run)', '// reportRunEnd words'), context);
    const streamRun = vm.runInContext('streamRun', context);
    const openDialog = vm.runInContext('openDialog', context);
    const hooks = {onLog() {}, onStage() {}, onCount() {}, onRefused() {}, onAbort() {}};
    function start(input) {
        streams.set(input, deferred());
        return streamRun({input, output: input + '.html'}, hooks);
    }
    function finish(input, question) {
        const marker = question ? `\x1edht:ask ${JSON.stringify({title: question, message: question})}\n` : '';
        let first = true;
        const stream = streams.get(input);
        stream.resolve({value: new TextEncoder().encode(marker), done: false});
        stream.promise = new Promise(resolve => {
            stream.end = () => resolve({done: true});
        });
    }
    return {start, finish, streams, opened, answers, dlg, openDialog};
}

const tick = () => new Promise(resolve => setImmediate(resolve));

test('ending one parallel run leaves the other cost question unanswered', async () => {
    const h = harness();
    const a = h.start('A'), b = h.start('B');
    await tick();
    h.finish('B', 'B cost');
    await tick();
    assert.deepEqual(h.opened, ['B cost']);
    h.finish('A', 'A cost');
    await tick();
    h.streams.get('A').end();
    await a;
    assert.deepEqual(h.opened, ['B cost']);
    assert.deepEqual(h.answers, []);
    h.dlg.returnValue = 'yes';
    h.dlg.close();
    await tick();
    assert.equal(h.answers.length, 1);
    assert.equal(h.answers[0].input, 'B');
    assert.equal(h.answers[0].yes, true);
    h.streams.get('B').end();
    await b;
});

test('a queued cost question is discarded when its run ends', async () => {
    const h = harness();
    const blocker = h.openDialog({title: 'blocker', buttons: [{id: 'ok', key: 'ok'}], cancel: 'ok'});
    await tick();
    const a = h.start('A');
    await tick();
    h.finish('A', 'A cost');
    await tick();
    h.streams.get('A').end();
    await a;
    h.dlg.returnValue = 'ok';
    h.dlg.close();
    await blocker;
    await tick();
    assert.deepEqual(h.opened, ['blocker']);
    assert.deepEqual(h.answers, []);
});

test('an unanswered visible question is withdrawn with its own run', async () => {
    const h = harness();
    const a = h.start('A');
    await tick();
    h.finish('A', 'A cost');
    await tick();
    h.streams.get('A').end();
    await a;
    await tick();
    assert.deepEqual(h.opened, ['A cost']);
    assert.deepEqual(h.answers, []);
});

// APP-BEHAVIOUR rule 3: while the first run is still preparing - readiness, the key check, the
// rebuild question, all before its stream exists - a second press (or Ctrl+Enter) starts nothing,
// and the button stays disabled until the first run's outcome is in.
test('a second Convert press during the first run starts nothing', async () => {
    const btn = {disabled: false};
    const started = [];
    let finish;
    const context = vm.createContext({
        queue: [], queueBusy: false, runAbort: null,
        el: id => { assert.equal(id, 'btnRun'); return btn; },
        runQueue: () => started.push('queue'),
        convertSingle: () => { started.push('single'); return new Promise(resolve => { finish = resolve; }); },
    });
    vm.runInContext(section('function runPressed()', '// makeLogAppender returns') +
        section('async function runConvert()', '// convertSingle is one'), context);
    const runPressed = vm.runInContext('runPressed', context);
    runPressed();
    assert.equal(btn.disabled, true, 'the button is disabled before the first await');
    runPressed();
    runPressed();
    assert.deepEqual(started, ['single']);
    finish();
    await tick();
    assert.equal(btn.disabled, false);
    runPressed();
    assert.deepEqual(started, ['single', 'single']);
});

// APP-BEHAVIOUR rule 7: formatting never throws; a missing, null or undefined argument renders a
// blank, never "{name}" or "undefined"; a missing translation falls back to English, then the key.
test('formatting renders a missing argument blank', () => {
    const context = vm.createContext({
        currentLang: 'xx',
        I18N: {en: {pair: '{a} and {b}', zero: 'n={n}', plain: 'no {x} here'}, xx: {pair: '{a} & {b}'}},
    });
    vm.runInContext(section('function t(key, params)', '// Dynamic status lines'), context);
    const t = vm.runInContext('t', context);
    assert.equal(t('pair', {a: 'A'}), 'A & ');
    assert.equal(t('pair', {a: 'A', b: undefined}), 'A & ');
    assert.equal(t('pair', {a: null, b: 'B'}), ' & B');
    assert.equal(t('zero', {n: 0}), 'n=0');
    assert.equal(t('plain'), 'no  here');
    assert.equal(t('missing'), 'missing');
    assert.equal(t(''), '');
});

// A FileDO secret file's password: the one modal dialog with a masked field. The field is made for
// the question, Enter confirms with the primary button, the value goes to /api/secret and nowhere
// else, and the field is wiped and removed when the dialog closes.
function secretHarness() {
    const secrets = [], opened = [];
    const listeners = new Map();
    let primaryClicks = 0, lastBox = null, lastInput = null;
    const fakeEl = (tag) => {
        const e = {tag, children: [], handlers: {}, value: '', textContent: '', attrs: {}, removed: false, clicks: 0,
            focus() {}, click() { e.clicks++; }, append(...c) { e.children.push(...c); },
            setAttribute(k, v) { e.attrs[k] = v; }, addEventListener(ev, fn) { e.handlers[ev] = fn; },
            remove() { e.removed = true; }};
        return e;
    };
    const dlg = {returnValue: '', addEventListener(ev, fn) { listeners.set(ev, fn); },
        showModal() { opened.push(true); }, close() { listeners.get('close')(); }};
    const actions = {innerHTML: '', lastChild: null, buttons: [],
        appendChild(b) { this.lastChild = b; this.buttons.push(b); },
        before(box) { lastBox = box; lastInput = box.children[1]; }};
    const elements = {dlg, dlgTitle: {textContent: ''}, dlgMsg: {textContent: ''}, dlgActions: actions};
    const context = vm.createContext({
        Promise, console,
        el: id => elements[id],
        document: {createElement: fakeEl},
        t: key => key,
        api: async (path, opts) => { secrets.push({path, body: JSON.parse(opts.body)}); return {}; },
    });
    vm.runInContext('let queueBusy = false;', context);
    vm.runInContext(section('let dlgQueue = Promise.resolve();', '// informDialog shows') +
        section('let queueSecret = null;', '// APP-BEHAVIOUR rule 6'), context);
    return {
        ask: (q) => vm.runInContext('askSecretForTheRun', context)({input: 'A.fd-sec', output: ''}, q, {active: true}),
        setQueue: (busy) => vm.runInContext(`queueBusy = ${busy}`, context),
        queueSecret: () => vm.runInContext('queueSecret', context),
        secrets, opened, dlg, actions,
        box: () => lastBox, input: () => lastInput,
    };
}

test('the password dialog sends the typed value to /api/secret and wipes its field', async () => {
    const h = secretHarness();
    const done = h.ask({title: 'FileDO secret file', message: 'Enter it', retry: false});
    await tick();
    assert.equal(h.opened.length, 1);
    const input = h.input();
    assert.equal(input.type, 'password', 'the field is masked');
    assert.equal(input.autocomplete, 'off');
    assert.equal(h.box().children[0].htmlFor, 'dlgSecretInput', 'the field has a label');
    input.value = 'typed "pw"';
    h.dlg.returnValue = 'open';
    h.dlg.close();
    await done;
    assert.equal(h.secrets.length, 1);
    assert.equal(h.secrets[0].path, '/api/secret');
    assert.deepEqual(h.secrets[0].body, {input: 'A.fd-sec', output: '', value: 'typed "pw"'});
    assert.equal(input.value, '', 'the field is wiped');
    assert.equal(h.box().removed, true, 'and removed');
    assert.equal(h.queueSecret(), null, 'a single conversion never keeps the password');
});

test('Enter in the password field presses the primary button', async () => {
    const h = secretHarness();
    const done = h.ask({title: 't', message: 'm'});
    await tick();
    let prevented = false;
    h.input().handlers.keydown({key: 'a', preventDefault() { prevented = true; }});
    assert.equal(h.actions.buttons[0].clicks, 0);
    h.input().handlers.keydown({key: 'Enter', preventDefault() { prevented = true; }});
    assert.equal(h.actions.buttons[0].clicks, 1);
    assert.equal(prevented, true);
    h.dlg.returnValue = 'cancel';
    h.dlg.close();
    await done;
});

test('cancelling the password dialog sends a cancel, never a value', async () => {
    const h = secretHarness();
    const done = h.ask({title: 't', message: 'm'});
    await tick();
    h.input().value = 'typed but cancelled';
    h.dlg.returnValue = 'cancel';
    h.dlg.close();
    await done;
    assert.deepEqual(h.secrets[0].body, {input: 'A.fd-sec', output: '', cancel: true});
});

test('a queue types the password once and asks again only after a wrong one', async () => {
    const h = secretHarness();
    h.setQueue(true);
    const first = h.ask({title: 't', message: 'm', retry: false});
    await tick();
    h.input().value = 'queue-pw';
    h.dlg.returnValue = 'open';
    h.dlg.close();
    await first;
    assert.equal(h.queueSecret(), 'queue-pw');

    // The next container gets the held password with no dialog.
    await h.ask({title: 't', message: 'm', retry: false});
    assert.equal(h.opened.length, 1, 'no second dialog');
    assert.equal(h.secrets[1].body.value, 'queue-pw');

    // A container it did not open comes back with retry set and asks again.
    const retry = h.ask({title: 't', message: 'm', retry: true});
    await tick();
    assert.equal(h.opened.length, 2);
    h.input().value = 'other-pw';
    h.dlg.returnValue = 'open';
    h.dlg.close();
    await retry;
    assert.equal(h.queueSecret(), 'other-pw');
    assert.equal(h.secrets[2].body.value, 'other-pw');
});