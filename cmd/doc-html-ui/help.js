// The window's one help resolver (CONTEXT-HELP rules 1-3 and 17, ticket 118). F1 and every help
// button call open() and nothing else: a surface id and, optionally, an element id go in, a page
// of the embedded help set comes out. The help map (help/help-map.json, served at
// /help/help-map.json) is the one table; the gate and this file read the same file.
//
// Nothing here is registered system-wide: F1 is a keydown handler of this page, so a browser's own
// help is shadowed only while this window has the focus. A surface the map marks "none" (a
// confirmation, a notice) answers F1 with nothing, and the browser's own help does not open there either.
(function (root) {
    'use strict';

    const WINDOW_NAME = 'dht-help';

    // resolve maps (surface, element, language) to {surface, path, anchor, locale}, or null when the
    // surface has no help. An element id such as "main.drop" is a row of its surface's `pages`; it
    // gives the anchor, and F1 pressed in that region goes straight to its entry (rule 3). A row that
    // inherits resolves through its parent, up to a few levels.
    function resolve(map, surfaceId, elementId, lang) {
        if (!map || !Array.isArray(map.surfaces)) return null;
        let row = null, anchor = '';
        if (elementId) {
            for (const s of map.surfaces) {
                const p = (s.pages || []).find(x => x.id === elementId);
                if (p) { row = s; anchor = p.anchor || ''; break; }
            }
        }
        if (!row) row = map.surfaces.find(s => s.id === surfaceId) || null;
        for (let depth = 0; row && row.form === 'inherit' && depth < 4; depth++) {
            const parent = row.parent;
            if (!anchor && row.anchor) anchor = row.anchor;
            row = map.surfaces.find(s => s.id === parent) || null;
        }
        if (!row || row.form === 'none' || !row.help) return null;
        const source = map.sourceLocale || 'en';
        const have = Array.isArray(map.locales) && map.locales.length ? map.locales : [source];
        const base = String(lang || source).toLowerCase().split('-')[0];
        const locale = have.includes(base) ? base : source;
        const path = '/' + (map.helpRoot || 'help') + '/' + (map.pageTemplate || '{locale}/{help}.html')
            .replace('{locale}', locale).replace('{help}', row.help);
        return {surface: row.id, path, anchor, locale};
    }

    // resolveIndex is the help index of the language (rule 8 item 7: a page never dead-ends).
    function resolveIndex(map, lang) {
        if (!map) return null;
        const source = map.sourceLocale || 'en';
        const have = Array.isArray(map.locales) && map.locales.length ? map.locales : [source];
        const base = String(lang || source).toLowerCase().split('-')[0];
        const locale = have.includes(base) ? base : source;
        const path = '/' + (map.helpRoot || 'help') + '/' + (map.pageTemplate || '{locale}/{help}.html')
            .replace('{locale}', locale).replace('{help}', map.index || 'index');
        return {surface: 'index', path, anchor: '', locale};
    }

    // targetUrl turns a resolution into the address to open. The page follows the window's theme when
    // the user pinned one; with the theme on "system" it follows the system on its own.
    function targetUrl(r, theme) {
        if (!r) return null;
        const q = theme === 'light' || theme === 'dark' ? '?theme=' + theme : '';
        return r.path + q + (r.anchor ? '#' + r.anchor : '');
    }

    // contextOf says which surface F1 asks about: an open dialog, else the main window; and, in the
    // window, the region the focus is in (the nearest [data-help] ancestor).
    function contextOf(doc) {
        const dlg = doc.querySelector('dialog[open]');
        if (dlg) return {surface: dlg.dataset.surface || '', element: ''};
        const active = doc.activeElement;
        const host = active && active.closest ? active.closest('[data-help]') : null;
        return {surface: 'main', element: host ? host.dataset.help : ''};
    }

    // isHelpKey is bare F1: a chord with a modifier belongs to the browser or to nobody.
    function isHelpKey(ev) {
        return ev.key === 'F1' && !ev.ctrlKey && !ev.altKey && !ev.shiftKey && !ev.metaKey;
    }

    const api = {resolve, resolveIndex, targetUrl, contextOf, isHelpKey, WINDOW_NAME};

    if (typeof document !== 'undefined' && typeof window !== 'undefined') {
        let map = null;
        const ready = fetch('/help/help-map.json', {cache: 'no-store'})
            .then(r => r.ok ? r.json() : null)
            .catch(() => null)
            .then(m => { map = m; document.dispatchEvent(new CustomEvent('dht-help-ready')); return m; });

        const lang = () => document.documentElement.lang || 'en';
        const theme = () => document.documentElement.dataset.theme || '';

        // has says whether a surface has a help target; a dialog shows its button only if so.
        api.has = (surfaceId, elementId) => !!resolve(map, surfaceId, elementId, lang());
        api.ready = ready;
        // open shows the help of a surface in the one help window of this session (the name makes
        // a second press reuse it). It opens nothing for a surface without help and returns whether
        // a window was asked for.
        api.open = (surfaceId, elementId) => {
            const url = targetUrl(resolve(map, surfaceId, elementId, lang()), theme());
            if (!url) return false;
            const w = window.open(url, WINDOW_NAME);
            if (w && w.focus) w.focus();
            return !!w;
        };
        api.openIndex = () => {
            const url = targetUrl(resolveIndex(map, lang()), theme());
            if (!url) return false;
            const w = window.open(url, WINDOW_NAME);
            if (w && w.focus) w.focus();
            return !!w;
        };

        // The key. Capture phase, so a focused control or an open modal cannot swallow it; it asks about
        // the surface in front of the user and about the region the focus is in.
        document.addEventListener('keydown', ev => {
            if (!isHelpKey(ev)) return;
            ev.preventDefault();
            const c = contextOf(document);
            if (map) api.open(c.surface, c.element); else ready.then(() => api.open(c.surface, c.element));
        }, true);

        // The button: any control carrying data-help-open="<surface id>" asks for that surface. The
        // dialog's own button asks for whichever dialog is open.
        document.addEventListener('click', ev => {
            const b = ev.target && ev.target.closest ? ev.target.closest('[data-help-open]') : null;
            if (!b) return;
            ev.preventDefault();
            const want = b.dataset.helpOpen;
            const go = () => {
                if (want === 'index') api.openIndex();
                else if (want === 'dialog') api.open(contextOf(document).surface, '');
                else api.open(want, '');
            };
            // a press in the first moments of the window may beat the map's arrival
            if (map) go(); else ready.then(go);
        });
    }

    if (typeof module !== 'undefined' && module.exports) module.exports = api;
    else root.DHTHelp = api;
})(typeof window !== 'undefined' ? window : globalThis);
