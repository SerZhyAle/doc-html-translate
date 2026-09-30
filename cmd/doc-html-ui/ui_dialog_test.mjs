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
