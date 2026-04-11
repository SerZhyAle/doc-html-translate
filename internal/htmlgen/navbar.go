package htmlgen

import (
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"

	"doc-html-translate/internal/epub"
)

// NavInfo describes navigation links for a single chapter page.
type NavInfo struct {
	PrevHref  string // empty if first page
	NextHref  string // empty if last page
	IndexHref string // relative path to index.html
	Title     string
	Current   int
	Total     int
}

// navBarCSS is the inline style for the sticky navigation bar.
const navBarCSS = `
<style id="dht-nav">
  .dht-navbar {
    position: sticky;
    top: 0;
    z-index: 9999;
    background: #2c3e50;
    color: #ecf0f1;
    display: flex;
    align-items: center;
		justify-content: flex-end;
		gap: 6px;
		padding: 6px 10px;
    font-family: Arial, Helvetica, sans-serif;
    font-size: 14px;
    box-shadow: 0 2px 4px rgba(0,0,0,0.3);
  }
  .dht-navbar a {
    color: #ecf0f1;
    text-decoration: none;
    padding: 4px 12px;
    border-radius: 3px;
    transition: background 0.2s;
  }
  .dht-navbar a:hover {
    background: #34495e;
  }
  .dht-navbar a.disabled {
    color: #7f8c8d;
    pointer-events: none;
    cursor: default;
  }
	.dht-navbar .nav-actions {
    display: flex;
    align-items: center;
		gap: 4px;
  }
  .dht-navbar .nav-info {
    font-size: 12px;
    color: #bdc3c7;
		margin-left: 4px;
  }
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
			// nav-actions children: [0]=prev, [1]=index, [2]=next
			var actions = document.querySelector(".nav-actions");
			if (!actions) return;
			var children = actions.children;
			var link = dir > 0 ? children[children.length - 1] : children[0];
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

// buildNavBarHTML generates the HTML for the navigation bar.
func buildNavBarHTML(nav NavInfo) string {
	prevLink := `<a class="disabled">&#9664; Назад</a>`
	if nav.PrevHref != "" {
		prevLink = fmt.Sprintf(`<a class="dht-nav-link" href="%s">&#9664; Назад</a>`, html.EscapeString(nav.PrevHref))
	}

	nextLink := `<a class="disabled">Вперёд &#9654;</a>`
	if nav.NextHref != "" {
		nextLink = fmt.Sprintf(`<a class="dht-nav-link" href="%s">Вперёд &#9654;</a>`, html.EscapeString(nav.NextHref))
	}

	indexLink := fmt.Sprintf(`<a class="dht-nav-link" href="%s">&#9776; Оглавление</a>`, html.EscapeString(nav.IndexHref))
	info := fmt.Sprintf(`<span class="nav-info">%d / %d</span>`, nav.Current, nav.Total)

	return fmt.Sprintf(`<div class="dht-navbar"><div class="nav-actions">%s%s%s</div>%s</div>%s`,
		prevLink, indexLink, nextLink, info, navBarScript)
}

// InjectNavBars adds a sticky navigation bar to all spine HTML files.
// It inserts CSS into <head> and the nav bar right after <body>.
func InjectNavBars(book *epub.Book, outputDir string) error {
	spineHrefs := book.SpineHrefs()
	total := len(spineHrefs)
	if total == 0 {
		return nil
	}

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
			PrevHref:  prevRel,
			NextHref:  nextRel,
			IndexHref: indexRel,
			Title:     book.Title,
			Current:   i + 1,
			Total:     total,
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

	// Inject CSS before </head>
	if idx := strings.Index(strings.ToLower(content), "</head>"); idx >= 0 {
		content = content[:idx] + navBarCSS + content[idx:]
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
