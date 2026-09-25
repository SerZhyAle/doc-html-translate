/* doc-html-translate site - the shared body scripts of PAGE-STYLE section 7, loaded at the end of
   <body> on every page. The pre-paint resolver stays inline in each <head>.

   A page declares its strings before loading this file:
     window.SITE = {strings: {ru: {...}, en: {...}, ua: {...}}}   - trilingual page (RU EN UA switch)
     window.SITE = {strings: {page: {...}}}                        - single-language page (locale landings)
     A per-language page (docs.ru.html) fixes data-lang in its markup, declares only its own
     language's strings, and gives each switch button a data-href to the sibling page.
   Keys: title, desc, copied, toLight, toDark, toTop, expand, collapse - each optional.

   The language value space is ru|en|ua. Every SZA page shares one origin and therefore one
   localStorage 'sza-lang'; a stored 'uk' (written by older pages) is read as 'ua'. */
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
      each('[data-set-lang]', function (b) { b.setAttribute('aria-pressed', String(b.getAttribute('data-set-lang') === l)); });
    }
    var t = str('title'), desc = str('desc');
    if (t) { document.title = t; meta('meta[property="og:title"]', t); meta('meta[name="twitter:title"]', t); }
    if (desc) { meta('meta[name="description"]', desc); meta('meta[property="og:description"]', desc); meta('meta[name="twitter:description"]', desc); }
    var top = $('toTop'); if (top && str('toTop')) top.setAttribute('aria-label', str('toTop'));
    each('[data-sec="open"]', function (b) { if (str('expand')) b.textContent = str('expand'); });
    each('[data-sec="close"]', function (b) { if (str('collapse')) b.textContent = str('collapse'); });
    paintTheme();
  }

  each('[data-set-lang]', function (b) {
    b.addEventListener('click', function () {
      var l = b.getAttribute('data-set-lang');
      try { localStorage.setItem('sza-lang', l); } catch (e) {}
      // Per-language pages (the docs) switch by navigating to the sibling page.
      var href = b.getAttribute('data-href');
      if (href) { location.href = href; return; }
      applyLang(l);
    });
  });

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
    addEventListener('scroll', function () { top.classList.toggle('show', scrollY > 600); }, { passive: true });
    top.addEventListener('click', function () { scrollTo({ top: 0, behavior: 'smooth' }); });
  }

  var tag = $('releaseTag');
  if (tag && window.fetch) {
    fetch('https://api.github.com/repos/SerZhyAle/doc-html-translate/releases/latest', { headers: { Accept: 'application/vnd.github+json' } })
      .then(function (r) { return r.ok ? r.json() : null; })
      .then(function (j) { if (j && j.tag_name) tag.textContent = ' ' + j.tag_name; })
      .catch(function () {});
  }

  applyLang(d.getAttribute('data-lang'));
})();
