package htmlgen

import (
	"fmt"
	"hash/fnv"
	"html"
	"os"
	"path" // hrefs are URLs, not OS paths
	"path/filepath"
	"strings"

	"doc-html-translate/internal/epub"
	"doc-html-translate/internal/logging"
	"doc-html-translate/internal/syslocale"
)

// projectURL is the home page opened from the version link in the navbar.
const projectURL = "https://serzhyale.github.io/doc-html-translate/"

// NavInfo describes navigation links for a single chapter page.
type NavInfo struct {
	PrevHref   string // empty if first page
	NextHref   string // empty if last page
	IndexHref  string // relative path to index.html
	Title      string
	SourceName string // original source file name, shown on the left of the bar
	Current    int
	Total      int
	BookKey    string // stable per-book id for localStorage (reading position)
	SelfHref   string // this page's href relative to the book root
}

// navBarCSS is the inline style for the sticky navigation bar. Colours come from the
// shared theme variables (readerCSS) so the bar follows the reading theme, matching the
// browser extension's viewer chrome.
const navBarCSS = `
<style id="dht-nav">
  .dht-navbar {
    position: sticky;
    top: 0;
    z-index: 9999;
    background: var(--dht-bar-bg);
    color: var(--dht-bar-fg);
    border-bottom: 1px solid var(--dht-border);
    display: flex;
    align-items: center;
		gap: 6px;
		padding: 6px 10px;
    font-family: "Segoe UI", system-ui, Arial, sans-serif;
    font-size: 14px;
  }
  .dht-navbar a {
    color: var(--dht-bar-fg);
    text-decoration: none;
    padding: 4px 10px;
    border-radius: 6px;
    transition: background 0.2s;
  }
  .dht-navbar a:hover {
    background: rgba(127,127,127,0.14);
  }
  .dht-navbar a.disabled {
    color: var(--dht-muted);
    pointer-events: none;
    cursor: default;
  }
  .dht-navbar .nav-file {
    font-weight: 600;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    max-width: 40%;
    padding-right: 12px;
  }
  .dht-navbar .nav-title {
    font-weight: 400;
    color: var(--dht-muted);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    max-width: 30%;
    padding-right: 12px;
  }
	.dht-navbar .nav-actions {
    display: flex;
    align-items: center;
		gap: 4px;
		margin-left: auto;
  }
  .dht-navbar .nav-info {
    font-size: 12px;
    color: var(--dht-muted);
		margin-left: 4px;
  }
  .dht-navbar a.nav-version {
    font-size: 11px;
    color: var(--dht-muted);
    padding: 2px 6px;
    margin-left: 4px;
    text-decoration: none;
  }
  .dht-navbar a.nav-version:hover {
    color: var(--dht-bar-fg);
    background: rgba(127,127,127,0.14);
  }
  /* prev/next turn group, pinned to the far corner for easy "next" clicks */
  .dht-navbar .nav-turn { display: flex; align-items: center; gap: 4px; margin-left: 8px; }
  .dht-navbar .nav-turn a.dht-next { font-weight: 600; }
  img {
    max-height: 100vh;
		height: auto;
		width: auto;
    max-width: 100%;
		object-fit: contain;
  }
</style>
`

const navBarScript = `
<script id="dht-zoom-sync">//<![CDATA[
(function () {
	var key = "dht_zoom";
	var min = 50;
	var max = 300;
	var step = 10;

	function clamp(value) {
		if (value < min) return min;
		if (value > max) return max;
		return value;
	}

	function readZoom() {
		var params = new URLSearchParams(window.location.search);
		var fromQuery = parseInt(params.get("z"), 10);
		if (!Number.isNaN(fromQuery)) {
			return clamp(fromQuery);
		}

		try {
			var fromSession = parseInt(sessionStorage.getItem(key), 10);
			if (!Number.isNaN(fromSession)) {
				return clamp(fromSession);
			}
		} catch (e) {
			// ignore storage errors
		}

		return 100;
	}

	function applyZoom(value) {
		var zoom = clamp(value);
		document.documentElement.style.zoom = (zoom / 100).toString();
		try {
			sessionStorage.setItem(key, String(zoom));
		} catch (e) {
			// ignore storage errors
		}
		return zoom;
	}

	function hrefWithZoom(rawHref, zoom) {
		try {
			var url = new URL(rawHref, window.location.href);
			url.searchParams.set("z", String(zoom));
			return url.href;
		} catch (e) {
			return rawHref;
		}
	}

	// Keep image proportions when any script/style changes image height.
	function preserveImageProportion(img) {
		if (!img) return;

		function apply() {
			var nw = img.naturalWidth || 0;
			var nh = img.naturalHeight || 0;
			if (!nw || !nh) {
				return;
			}

			var hStyle = (img.style && img.style.height) ? img.style.height : "";
			var hAttr = img.getAttribute("height") || "";
			var explicitHeight =
				(hStyle && hStyle !== "auto") ||
				(hAttr && hAttr !== "auto");

			if (!explicitHeight) {
				// Default path: let browser preserve ratio naturally.
				img.style.width = "auto";
				return;
			}

			var renderedH = img.getBoundingClientRect().height;
			if (!renderedH || renderedH <= 0) {
				img.style.width = "auto";
				return;
			}

			var computedW = Math.round((renderedH * nw) / nh);
			if (computedW > 0) {
				img.style.width = String(computedW) + "px";
			}
		}

		if (img.complete) {
			apply();
		} else {
			img.addEventListener("load", apply, { once: true });
		}
	}

	function installImageAspectGuards() {
		var images = document.querySelectorAll("img");
		images.forEach(function (img) {
			preserveImageProportion(img);
			var observer = new MutationObserver(function () {
				preserveImageProportion(img);
			});
			observer.observe(img, {
				attributes: true,
				attributeFilter: ["style", "height", "width"]
			});
		});
	}

	function getIndexHref() {
		var indexLink = document.querySelector('.dht-navbar a[href*="index.html"]');
		if (!indexLink) return "";
		return indexLink.getAttribute("href") || "";
	}

	function isLegacyXHTMLHref(href) {
		return /\.xhtml?(?:[?#]|$)/i.test(href || "");
	}

	var zoom = applyZoom(readZoom());
	installImageAspectGuards();

	// Legacy guard: if a chapter .xhtml is opened directly, redirect to index.html.
	if (isLegacyXHTMLHref(window.location.pathname)) {
		var indexHref = getIndexHref();
		if (indexHref) {
			window.location.replace(hrefWithZoom(indexHref, zoom));
			return;
		}
	}

	document.addEventListener("wheel", function (event) {
		if (!event.ctrlKey) {
			return;
		}
		event.preventDefault();

		if (event.deltaY < 0) {
			zoom = applyZoom(zoom + step);
		} else {
			zoom = applyZoom(zoom - step);
		}
	}, { passive: false });

	var links = document.querySelectorAll(".dht-navbar a[href]");
	links.forEach(function (link) {
		link.addEventListener("click", function (event) {
			var rawHref = link.getAttribute("href");
			if (!rawHref) {
				return;
			}

			// Block direct chapter-to-chapter XHTML navigation in legacy folders.
			if (isLegacyXHTMLHref(rawHref)) {
				event.preventDefault();
				var idx = getIndexHref();
				if (idx) {
					window.location.href = hrefWithZoom(idx, zoom);
				}
				return;
			}

			link.setAttribute("href", hrefWithZoom(rawHref, zoom));
		});
	});

	// --- Edge-scroll auto-navigation ---
	// PageDown at bottom → next page; PageUp at top → prev page.
	// Wheel: 3 consecutive overflow ticks in same direction → navigate.
	(function () {
		var SCROLL_THRESHOLD = 3;
		var overflowCount = 0;
		var overflowDir = 0;

		function isAtBottom() {
			return Math.round(window.scrollY + window.innerHeight) >= document.documentElement.scrollHeight - 4;
		}
		function isAtTop() {
			return window.scrollY <= 4;
		}
		function tryNavigate(dir) {
			var actions = document.querySelector(".nav-actions");
			if (!actions) return;
			var link = actions.querySelector(dir > 0 ? ".dht-next" : ".dht-prev");
			if (link && !link.classList.contains("disabled")) {
				link.click();
			}
		}

		document.addEventListener("wheel", function (e) {
			if (e.ctrlKey) return;
			var dir = e.deltaY > 0 ? 1 : (e.deltaY < 0 ? -1 : 0);
			if (dir === 0) return;
			var atEdge = (dir > 0 && isAtBottom()) || (dir < 0 && isAtTop());
			if (atEdge) {
				if (dir === overflowDir) {
					overflowCount++;
				} else {
					overflowDir = dir;
					overflowCount = 1;
				}
				if (overflowCount >= SCROLL_THRESHOLD) {
					overflowCount = 0;
					tryNavigate(dir);
				}
			} else {
				overflowCount = 0;
				overflowDir = 0;
			}
		}, { passive: true });

		document.addEventListener("keydown", function (e) {
			if (e.key === "PageDown" && isAtBottom()) {
				e.preventDefault();
				tryNavigate(1);
			} else if (e.key === "PageUp" && isAtTop()) {
				e.preventDefault();
				tryNavigate(-1);
			}
		});
	})();
})();
//]]></script>
`

// readerCSS styles the reading themes ([12]), the theme toggle button, the
// thin reading-progress bar and the index-page toolbar ([14]). It is injected
// into <head> on both chapter pages and index.html.
// readerCSS defines the shared reading themes (light/sepia/dark/night), the reader font
// variables (size + family), the theme/font controls, and the progress bar. The colour
// values are the canonical palette shared with the browser extension's viewer.css so both
// front-ends look identical. Injected into <head> on chapter pages and index.html.
const readerCSS = `
<style id="dht-reader-css">
  :root {
    --dht-bg:#faf9f7; --dht-fg:#1b1b1b; --dht-muted:#6b6b6b;
    --dht-bar-bg:#ffffff; --dht-bar-fg:#222222; --dht-border:#e2e0db;
    --dht-accent:#2563eb; --dht-link:#1a4fb4;
    /* keep in step with DEFSZ in the reader script: this styles the page before that
       runs, so a mismatch shows as the text resizing under the reader on every load */
    --dht-reader-size:175%; --dht-reader-font:Georgia,"Times New Roman",serif;
  }
  html[data-dht-theme="sepia"] {
    --dht-bg:#f4ecd8; --dht-fg:#4a3f2f; --dht-muted:#7a6c54;
    --dht-bar-bg:#efe6cf; --dht-bar-fg:#4a3f2f; --dht-border:#ddd0b0;
    --dht-accent:#8a5a2b; --dht-link:#7a4a1b;
  }
  html[data-dht-theme="dark"] {
    --dht-bg:#1a1a1c; --dht-fg:#e6e4df; --dht-muted:#9a9893;
    --dht-bar-bg:#232327; --dht-bar-fg:#e6e4df; --dht-border:#36363b;
    --dht-accent:#5b8dff; --dht-link:#8fb4ff;
  }
  html[data-dht-theme="night"] {
    --dht-bg:#0a0a0b; --dht-fg:#9a9a9a; --dht-muted:#6a6a6a;
    --dht-bar-bg:#131315; --dht-bar-fg:#b8b8b8; --dht-border:#262629;
    --dht-accent:#5599d6; --dht-link:#6aa8e0;
  }
  html, body { background:var(--dht-bg) !important; color:var(--dht-fg) !important; }
  body { font-size:var(--dht-reader-size); font-family:var(--dht-reader-font); }
  body a { color:var(--dht-link); }
  /* A page that is nothing but a scan. The converter has already sized this box to the
     window (see pdf.pageScanBox: min of the image's own width, 96vw and its height in vh),
     so all that is left is to centre it out of the narrow text column it sits in - by
     shifting the box, not by widening the column, which would wreck the measure of a page
     that does have text. The image fills the box; with OCR on, .ocr-fig fills it instead
     and the image fills that, so the plates stay pinned to what they cover. */
  .pdf-page-scan { margin-left:50%; transform:translateX(-50%); }
  .pdf-page-scan img { display:block; width:100%; height:auto; max-height:none; margin:0; }
  .dht-btn, .dht-navbar select, .dht-toolbar select {
    background:transparent; color:var(--dht-bar-fg); border:1px solid var(--dht-border);
    border-radius:6px; padding:3px 8px; font:inherit; font-size:13px; cursor:pointer;
  }
  .dht-btn:hover, .dht-navbar select:hover, .dht-toolbar select:hover { border-color:var(--dht-accent); }
  .dht-progress { position:absolute; left:0; bottom:0; height:3px; width:0; background:var(--dht-accent); transition:width .12s linear; }
  .dht-toolbar { display:flex; gap:8px; align-items:center; flex-wrap:wrap; margin:0 0 1.2em; }
  .dht-toolbar a.dht-continue { display:none; background:var(--dht-accent); color:#fff; text-decoration:none; padding:4px 10px; border-radius:6px; }
  .dht-toolbar a.dht-continue:hover { filter:brightness(1.08); }
</style>
`

// readerScript emits the theme + reading-position controller injected into
// every page. On chapter pages (self != "") it restores and tracks scroll
// position and drives the progress bar; on index.html (self == "") it wires up
// the "Continue reading" link. The theme choice is global and lives in
// localStorage so it survives across sessions (unlike the zoom sessionStorage).
func readerScript(bookKey, self string, idx, total int) string {
	return fmt.Sprintf(`
<script id="dht-reader">//<![CDATA[
(function(){
	var BOOK = %q;
	var SELF = %q;
	var IDX = %d;
	var TOTAL = %d;

	var THEME_KEY = "dht_theme", SIZE_KEY = "dht_fontsize", FAM_KEY = "dht_family";
	var FAMILIES = {
		serif: 'Georgia,"Times New Roman",serif',
		sans: '"Segoe UI",system-ui,Arial,sans-serif',
		mono: '"Cascadia Code",Consolas,monospace'
	};
	// DEFSZ is a percentage of the browser's own body size (16px unless the reader changed
	// it), so 175 lands on ~28px - matching the extension viewer's own default. It sits far
	// above 100 on purpose: this is a reading view, not a web page. MAXSZ has to stay well
	// clear of DEFSZ, or A+ would have nowhere to go.
	var MINSZ = 70, MAXSZ = 300, STEPSZ = 10, DEFSZ = 175;

	// Theme (dropdown, 4 themes; global, localStorage).
	function getTheme(){ try { return localStorage.getItem(THEME_KEY) || "light"; } catch(e){ return "light"; } }
	function applyTheme(t){
		document.documentElement.setAttribute("data-dht-theme", t);
		var s = document.getElementById("dht-theme-sel");
		if (s && s.value !== t) s.value = t;
	}
	function setTheme(t){ try { localStorage.setItem(THEME_KEY, t); } catch(e){} applyTheme(t); }
	applyTheme(getTheme());
	var tsel = document.getElementById("dht-theme-sel");
	if (tsel) tsel.addEventListener("change", function(){ setTheme(tsel.value); });

	// Text size (A-/A+), scales the em-based content without touching images.
	function getSize(){ var n; try { n = parseInt(localStorage.getItem(SIZE_KEY), 10); } catch(e){} return (!n || Number.isNaN(n)) ? DEFSZ : n; }
	// persist only when the reader actually asked for a size. Writing it on every load
	// would stamp the current default into storage the first time any book is opened, and
	// a later change to DEFSZ could then never reach anyone who had ever read anything.
	function applySize(n, persist){
		if (n < MINSZ) n = MINSZ; if (n > MAXSZ) n = MAXSZ;
		document.documentElement.style.setProperty("--dht-reader-size", n + "%%");
		if (persist) { try { localStorage.setItem(SIZE_KEY, String(n)); } catch(e){} }
		return n;
	}
	var sizeNow = applySize(getSize(), false);
	var dec = document.getElementById("dht-font-dec");
	if (dec) dec.addEventListener("click", function(){ sizeNow = applySize(sizeNow - STEPSZ, true); });
	var inc = document.getElementById("dht-font-inc");
	if (inc) inc.addEventListener("click", function(){ sizeNow = applySize(sizeNow + STEPSZ, true); });

	// Font family (dropdown).
	function getFam(){ try { return localStorage.getItem(FAM_KEY) || "serif"; } catch(e){ return "serif"; } }
	function applyFam(f){
		document.documentElement.style.setProperty("--dht-reader-font", FAMILIES[f] || FAMILIES.serif);
		var s = document.getElementById("dht-family-sel");
		if (s && s.value !== f) s.value = f;
	}
	function setFam(f){ try { localStorage.setItem(FAM_KEY, f); } catch(e){} applyFam(f); }
	applyFam(getFam());
	var fsel = document.getElementById("dht-family-sel");
	if (fsel) fsel.addEventListener("change", function(){ setFam(fsel.value); });

	var POS_KEY = "dht_pos:" + BOOK;
	function readPos(){ try { return JSON.parse(localStorage.getItem(POS_KEY) || "null"); } catch(e){ return null; } }
	function writePos(p){ try { localStorage.setItem(POS_KEY, JSON.stringify(p)); } catch(e){} }
	function frac(){
		var h = document.documentElement.scrollHeight - window.innerHeight;
		if (h <= 0) return 0;
		var f = window.scrollY / h;
		return f < 0 ? 0 : (f > 1 ? 1 : f);
	}
	function setBar(f){
		var bar = document.getElementById("dht-progress");
		if (!bar) return;
		var overall = TOTAL > 0 ? ((IDX - 1 + f) / TOTAL) : f;
		bar.style.width = (overall * 100).toFixed(2) + "%%";
	}

	if (SELF) {
		var saved = readPos();
		if (saved && saved.href === SELF && typeof saved.frac === "number") {
			window.addEventListener("load", function(){
				var h = document.documentElement.scrollHeight - window.innerHeight;
				if (h > 0) window.scrollTo(0, saved.frac * h);
			});
		}
		var ticking = false;
		window.addEventListener("scroll", function(){
			if (ticking) return;
			ticking = true;
			window.requestAnimationFrame(function(){
				var f = frac();
				setBar(f);
				writePos({href: SELF, idx: IDX, frac: f});
				ticking = false;
			});
		}, {passive:true});
		setBar(frac());
	} else {
		var p = readPos();
		var cont = document.getElementById("dht-continue");
		if (cont && p && p.href) {
			cont.setAttribute("href", p.href);
			cont.style.display = "inline-block";
		}
	}
})();
//]]></script>
`, bookKey, self, idx, total)
}

// bookStorageKey derives a stable per-book id used to namespace reading-position
// localStorage. It must be identical on chapter pages and index.html, so it is
// computed from the same (raw title, page count) on both paths.
func bookStorageKey(title string, total int) string {
	h := fnv.New32a()
	fmt.Fprintf(h, "%s|%d", title, total)
	return fmt.Sprintf("%08x", h.Sum32())
}

// buildNavBarHTML generates the HTML for the navigation bar.
func buildNavBarHTML(nav NavInfo) string {
	labelPrev, labelNext, labelTOC := "Back", "Forward", "Contents"
	if syslocale.IsRussian() {
		labelPrev, labelNext, labelTOC = "Назад", "Вперёд", "Оглавление"
	}

	prevLink := fmt.Sprintf(`<a class="disabled">&#9664; %s</a>`, labelPrev)
	if nav.PrevHref != "" {
		prevLink = fmt.Sprintf(`<a class="dht-nav-link dht-prev" href="%s">&#9664; %s</a>`, html.EscapeString(nav.PrevHref), labelPrev)
	}

	nextLink := fmt.Sprintf(`<a class="disabled">%s &#9654;</a>`, labelNext)
	if nav.NextHref != "" {
		nextLink = fmt.Sprintf(`<a class="dht-nav-link dht-next" href="%s">%s &#9654;</a>`, html.EscapeString(nav.NextHref), labelNext)
	}

	indexLink := fmt.Sprintf(`<a class="dht-nav-link" href="%s">&#9776; %s</a>`, html.EscapeString(nav.IndexHref), labelTOC)
	info := fmt.Sprintf(`<span class="nav-info">%d / %d</span>`, nav.Current, nav.Total)

	var fileEl string
	if strings.TrimSpace(nav.SourceName) != "" {
		fileEl = fmt.Sprintf(`<span class="nav-file" title="%s">%s</span>`,
			html.EscapeString(nav.SourceName), html.EscapeString(nav.SourceName))
	}

	var titleEl string
	if strings.TrimSpace(nav.Title) != "" {
		titleEl = fmt.Sprintf(`<span class="nav-title" title="%s">%s</span>`,
			html.EscapeString(nav.Title), html.EscapeString(nav.Title))
	}

	versionLink := fmt.Sprintf(
		`<a class="nav-version" href="%s" target="_blank" rel="noopener" title="%s">%s</a>`,
		projectURL, html.EscapeString(projectURL), html.EscapeString(versionLabel()))

	// prev/next live in a turn group pinned to the far corner (next flush to the edge)
	// so "next" is the easiest button to hit while reading.
	turn := fmt.Sprintf(`<span class="nav-turn">%s%s</span>`, prevLink, nextLink)

	return fmt.Sprintf(`<div class="dht-navbar">%s%s<div class="nav-actions">%s%s%s%s%s</div><div id="dht-progress" class="dht-progress"></div></div>%s%s`,
		fileEl, titleEl, indexLink, readerControlsHTML(), versionLink, info, turn, navBarScript,
		readerScript(nav.BookKey, nav.SelfHref, nav.Current, nav.Total))
}

// readerControlsHTML returns the shared reader controls (text size, font family, theme
// dropdown) used identically in the chapter navbar and the index toolbar, so both the
// generated HTML and the browser extension expose the same operations.
func readerControlsHTML() string {
	titleSmaller, titleLarger, titleFont, titleTheme := "Smaller text", "Larger text", "Font", "Theme"
	tLight, tSepia, tDark, tNight := "Light", "Sepia", "Dark", "Night"
	if syslocale.IsRussian() {
		titleSmaller, titleLarger, titleFont, titleTheme = "Мельче", "Крупнее", "Шрифт", "Тема"
		tLight, tSepia, tDark, tNight = "Светлая", "Сепия", "Тёмная", "Ночь"
	}
	return fmt.Sprintf(
		`<button id="dht-font-dec" class="dht-btn" type="button" title="%s">A&minus;</button>`+
			`<button id="dht-font-inc" class="dht-btn" type="button" title="%s">A+</button>`+
			`<select id="dht-family-sel" title="%s"><option value="serif">Serif</option><option value="sans">Sans</option><option value="mono">Mono</option></select>`+
			`<select id="dht-theme-sel" title="%s"><option value="light">&#9728; %s</option><option value="sepia">&#9681; %s</option><option value="dark">&#9790; %s</option><option value="night">&#9679; %s</option></select>`,
		titleSmaller, titleLarger, titleFont, titleTheme, tLight, tSepia, tDark, tNight)
}

// versionLabel formats the running app version for display in the navbar.
// Numeric calendar versions are prefixed with "v"; "dev" and the like are shown as-is.
func versionLabel() string {
	v := strings.TrimSpace(logging.AppVersion)
	if v == "" {
		v = "dev"
	}
	if v != "dev" && !strings.HasPrefix(v, "v") {
		v = "v" + v
	}
	return v
}

// InjectNavBars adds a sticky navigation bar to all spine HTML files.
// It inserts CSS into <head> and the nav bar right after <body>.
// sourceName is the original input file name, shown on the left of the bar.
func InjectNavBars(book *epub.Book, outputDir, sourceName string) error {
	spineHrefs := book.SpineHrefs()
	total := len(spineHrefs)
	if total == 0 {
		return nil
	}
	WriteFavicon(outputDir)
	bookKey := bookStorageKey(book.Title, total)

	// Build full href paths (with BasePath prefix)
	fullHrefs := make([]string, total)
	for i, href := range spineHrefs {
		if book.BasePath != "" && book.BasePath != "." {
			fullHrefs[i] = book.BasePath + "/" + href
		} else {
			fullHrefs[i] = href
		}
	}

	for i, href := range fullHrefs {
		filePath := filepath.Join(outputDir, filepath.FromSlash(href))

		// Calculate relative paths from this file's directory to siblings and index
		thisDir := filepath.Dir(href)

		var prevRel, nextRel string
		if i > 0 {
			prevRel = relativePath(thisDir, fullHrefs[i-1])
		}
		if i < total-1 {
			nextRel = relativePath(thisDir, fullHrefs[i+1])
		}
		indexRel := relativePath(thisDir, "index.html")

		nav := NavInfo{
			PrevHref:   prevRel,
			NextHref:   nextRel,
			IndexHref:  indexRel,
			Title:      book.Title,
			SourceName: sourceName,
			Current:    i + 1,
			Total:      total,
			BookKey:    bookKey,
			SelfHref:   href,
		}

		if err := injectNavIntoFile(filePath, nav); err != nil {
			// Best-effort: warn and continue
			fmt.Fprintf(os.Stderr, "WARNING: navbar inject skip %s: %v\n", href, err)
		}
	}

	return nil
}

// injectNavIntoFile reads an HTML file, injects the navbar CSS and HTML.
func injectNavIntoFile(filePath string, nav NavInfo) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	content := string(data)
	navHTML := buildNavBarHTML(nav)

	// Inject the tab icon + CSS before </head>. The icon lives at the output root, so the
	// link is relative to this page's own depth.
	if idx := strings.Index(strings.ToLower(content), "</head>"); idx >= 0 {
		content = content[:idx] + faviconLink(path.Dir(nav.SelfHref)) + navBarCSS + readerCSS + content[idx:]
	}

	// Inject navbar after <body> (or <body ...>)
	bodyIdx := findBodyTagEnd(content)
	if bodyIdx >= 0 {
		content = content[:bodyIdx] + navHTML + content[bodyIdx:]
	}

	return os.WriteFile(filePath, []byte(content), 0o644)
}

// findBodyTagEnd finds the position right after the <body...> tag.
func findBodyTagEnd(content string) int {
	lower := strings.ToLower(content)
	bodyStart := strings.Index(lower, "<body")
	if bodyStart < 0 {
		return -1
	}
	// Find the closing > of the <body> tag
	closeIdx := strings.Index(content[bodyStart:], ">")
	if closeIdx < 0 {
		return -1
	}
	return bodyStart + closeIdx + 1
}

// relativePath computes a relative URL path from fromDir to target.
// Both use forward slashes (URL convention).
func relativePath(fromDir, target string) string {
	// Normalize to forward slashes
	fromDir = strings.ReplaceAll(fromDir, "\\", "/")
	target = strings.ReplaceAll(target, "\\", "/")

	if fromDir == "." || fromDir == "" {
		return target
	}

	fromParts := strings.Split(fromDir, "/")
	targetDir := filepath.Dir(target)
	targetDir = strings.ReplaceAll(targetDir, "\\", "/")
	targetBase := filepath.Base(target)

	targetParts := strings.Split(targetDir, "/")
	if targetDir == "." || targetDir == "" {
		targetParts = nil
	}

	// Find common prefix length
	common := 0
	for common < len(fromParts) && common < len(targetParts) && fromParts[common] == targetParts[common] {
		common++
	}

	// Build relative path: go up from fromDir, then down to target
	var parts []string
	for i := common; i < len(fromParts); i++ {
		parts = append(parts, "..")
	}
	for i := common; i < len(targetParts); i++ {
		parts = append(parts, targetParts[i])
	}
	parts = append(parts, targetBase)

	return strings.Join(parts, "/")
}
