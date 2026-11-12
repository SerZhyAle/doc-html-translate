/* doc-html-translate site - the shared body scripts of PAGE-STYLE section 7, loaded at the end of
   <body> on every page. The pre-paint resolver stays inline in each <head>.

   A page declares its strings before loading this file:
     window.SITE = {strings: {ru: {...}, en: {...}, ua: {...}}}   - trilingual page (RU EN UA switch)
     window.SITE = {strings: {page: {...}}}                        - single-language page (locale landings)
     A per-language page (docs.ru.html) fixes data-lang in its markup, declares only its own
     language's strings, and gives each switch button a data-href to the sibling page.
   Keys: title, desc, copied, toLight, toDark, toTop, nav, expand, collapse - each optional.
   nav labels every <nav data-nav> (the header link row); a page whose nav carries a fixed aria-label omits it.

   The language value space is ru|en|ua. Every SZA page shares one origin and therefore one
   localStorage 'sza-lang'; a stored 'uk' (written by older pages) is read as 'ua'. A page opened by ?lang= or ?l= takes
   that language for the visit without storing it (the head reader of each page), and carryLang below passes it on. */
(function () {
  var d = document.documentElement;
  var S = (window.SITE && window.SITE.strings) || {};
  var LIGHT = '#eef3ea', DARK = '#0a0f0a';

  function norm(l) { return l === 'uk' ? 'ua' : (l === 'ru' || l === 'en' || l === 'ua') ? l : null; }
  function cur() { return S.page ? 'page' : (norm(d.getAttribute('data-lang')) || 'en'); }
  function str(k) { var m = S[cur()] || S.en || {}; return m[k]; }
  function $(id) { return document.getElementById(id); }
  function meta(sel, v) { var e = document.querySelector(sel); if (e && v) e.setAttribute('content', v); }
  function each(sel, fn) { Array.prototype.forEach.call(document.querySelectorAll(sel), fn); }

  // SITE-STRUCTURE 8.B: the language a visitor arrived in is kept while the visitor moves. When storage does not hold
  // it - the page was opened by ?lang= (the head reader marks that with data-lang-src) or is a page of one language
  // (the docs trio: language links, no buttons) - the next page would fall back to the browser's language, so the links
  // to the pages that carry their languages in one file get ?lang=<code> here, at run time. The markup keeps its canonical
  // hrefs; a page whose language came from storage or from the browser changes nothing.
  var CARRY = /^(?:\.\/|index\.html|extension\.html|extension-privacy\.html|install-trust\.html|privacy\.html|release-notes\.html|support\.html)(?=[?#]|$)/;
  var LANG_QUERY = /[?&](?:lang|l)=/;
  function carryLang(l) {
    if (!d.hasAttribute('data-lang-src') && !document.querySelector('.seg a[data-set-lang]')) return;
    each('a[href]', function (a) {
      if (a.siteHref === undefined) a.siteHref = a.getAttribute('href');
      var h = a.siteHref;
      if (!CARRY.test(h) || LANG_QUERY.test(h)) return;
      var i = h.indexOf('#'), base = i < 0 ? h : h.slice(0, i), frag = i < 0 ? '' : h.slice(i);
      a.setAttribute('href', base + (base.indexOf('?') < 0 ? '?' : '&') + 'lang=' + (l === 'ua' ? 'uk' : l) + frag);
    });
  }

  function paintTheme() {
    var dark = d.getAttribute('data-theme') !== 'light';
    meta('meta[name="theme-color"]', dark ? DARK : LIGHT);
    var b = $('themeBtn'), label = str(dark ? 'toLight' : 'toDark');
    if (b && label) { b.setAttribute('aria-label', label); b.setAttribute('title', label); }
  }

  function applyLang(l) {
    if (!S.page) {
      l = norm(l) || 'en';
      d.setAttribute('data-lang', l);
      d.setAttribute('lang', l === 'ua' ? 'uk' : l);
      each('[data-set-lang]', function (b) {
        var on = b.getAttribute('data-set-lang') === l;
        // a language LINK (the docs pages, one page per language) says which one is current; a button says it is pressed
        if (b.tagName === 'A') { if (on) b.setAttribute('aria-current', 'true'); else b.removeAttribute('aria-current'); }
        else b.setAttribute('aria-pressed', String(on));
      });
      carryLang(l);
    }
    // the share card (og:, twitter:) is never rewritten: a scraper reads the static tags, so they are the only card
    var t = str('title'), desc = str('desc');
    if (t) document.title = t;
    if (desc) meta('meta[name="description"]', desc);
    var top = $('toTop'); if (top && str('toTop')) top.setAttribute('aria-label', str('toTop'));
    each('[data-nav]', function (n) { if (str('nav')) n.setAttribute('aria-label', str('nav')); });
    each('[data-sec="open"]', function (b) { if (str('expand')) b.textContent = str('expand'); });
    each('[data-sec="close"]', function (b) { if (str('collapse')) b.textContent = str('collapse'); });
    paintTheme();
  }

  each('[data-set-lang]', function (b) {
    b.addEventListener('click', function () {
      var l = b.getAttribute('data-set-lang');
      try { localStorage.setItem('sza-lang', l); } catch (e) {}
      // A language link (the docs pages, one page per language) navigates by itself: BUTTONS 2.5 - navigation to a
      // web address is a link, so the choice is stored and the browser follows the href.
      if (b.tagName === 'A') return;
      var href = b.getAttribute('data-href');
      if (href) { location.href = href; return; }
      applyLang(l);
      writeLangQuery(l);
    });
  });

  // PAGE-STYLE 3.5: the chosen language goes into the address, so a shared link opens in it; lang= replaces a legacy l=
  function writeLangQuery(l) {
    if (!window.history || !history.replaceState || !window.URL) return;
    try {
      var u = new URL(location.href);
      u.searchParams.delete('l');
      u.searchParams.set('lang', norm(l) === 'ua' ? 'uk' : (norm(l) || 'en'));
      history.replaceState(history.state, '', u.pathname + u.search + u.hash);
    } catch (e) {}
  }

  var themeBtn = $('themeBtn');
  if (themeBtn) themeBtn.addEventListener('click', function () {
    var t = d.getAttribute('data-theme') === 'light' ? 'dark' : 'light';
    d.setAttribute('data-theme', t);
    try { localStorage.setItem('sza-theme', t); } catch (e) {}
    paintTheme();
  });

  function fallbackCopy(text) {
    var t = document.createElement('textarea');
    t.value = text; t.setAttribute('readonly', ''); t.style.position = 'fixed'; t.style.opacity = '0';
    document.body.appendChild(t); t.select();
    try { document.execCommand('copy'); } catch (e) {}
    t.remove();
  }
  each('.copybox .copy', function (b) {
    var label = b.innerHTML;
    b.addEventListener('click', function () {
      var box = b.closest('.copybox'), text = box.getAttribute('data-copy') || box.textContent.trim();
      var done = function () {
        b.classList.add('done'); b.textContent = str('copied') || 'Copied';
        setTimeout(function () { b.classList.remove('done'); b.innerHTML = label; }, 1600);
      };
      if (navigator.clipboard && window.isSecureContext) navigator.clipboard.writeText(text).then(done, function () { fallbackCopy(text); done(); });
      else { fallbackCopy(text); done(); }
    });
  });

  function openFromHash() {
    var id = decodeURIComponent(location.hash.slice(1)), e = id && $(id);
    if (!e) return;
    for (var p = e; p; p = p.parentElement) if (p.tagName === 'DETAILS') p.open = true;
    e.scrollIntoView();
  }
  addEventListener('hashchange', openFromHash);
  if (location.hash) openFromHash();

  each('[data-sec]', function (b) {
    b.addEventListener('click', function () {
      var open = b.getAttribute('data-sec') === 'open', scope = b.closest('section') || document;
      Array.prototype.forEach.call(scope.querySelectorAll('details.sec'), function (x) { x.open = open; });
    });
  });

  var top = $('toTop');
  if (top) {
    // shown after 600 px, hidden again while the footer is on screen: the fixed button would sit over the contact
    // links (and the copyright line on right-to-left pages), so it stays clear of the footer instead
    var foot = document.querySelector('.site-footer');
    var paintTop = function () {
      var clear = !foot || foot.getBoundingClientRect().top >= innerHeight;
      top.classList.toggle('show', scrollY > 600 && clear);
    };
    addEventListener('scroll', paintTop, { passive: true });
    addEventListener('resize', paintTop);
    paintTop();
    top.addEventListener('click', function () {
      var still = window.matchMedia && matchMedia('(prefers-reduced-motion: reduce)').matches;
      scrollTo({ top: 0, behavior: still ? 'auto' : 'smooth' });
    });
  }

  applyLang(d.getAttribute('data-lang'));
})();
