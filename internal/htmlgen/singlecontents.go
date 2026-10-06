package htmlgen

import (
	"fmt"
	"html"
	"strings"

	"doc-html-translate/internal/epub"
	"doc-html-translate/internal/i18n"

	gohtml "golang.org/x/net/html"
)

// singlePageContents retains the authored hierarchy while resolving its targets
// through the same chapter/id map used for in-book links during the merge.
func singlePageContents(entries []epub.TOCEntry, chapters []*mergeChapter) []epub.TOCEntry {
	index := make(map[string]*mergeChapter, len(chapters))
	for _, ch := range chapters {
		if index[ch.href] == nil {
			index[ch.href] = ch
		}
	}
	var mapEntries func([]epub.TOCEntry) []epub.TOCEntry
	mapEntries = func(src []epub.TOCEntry) []epub.TOCEntry {
		var out []epub.TOCEntry
		for _, e := range src {
			children := mapEntries(e.Children)
			href := ""
			if external, clickable := epub.ExternalHref(e.Href); external {
				if clickable {
					href = strings.TrimSpace(e.Href)
				}
			} else if strings.HasPrefix(e.Href, "#") && len(chapters) > 0 {
				href = "#" + epub.URLPath(chapters[0].targetID(e.Href[1:]))
			} else if target, suffix, ok := resolveRelative(".", e.Href); ok {
				if ch := index[target]; ch != nil {
					_, frag, _ := strings.Cut(suffix, "#")
					href = "#" + epub.URLPath(ch.targetID(frag))
				}
			}
			if href == "" && len(children) == 0 {
				continue
			}
			if strings.TrimSpace(e.Title) == "" && len(children) == 0 {
				continue
			}
			out = append(out, epub.TOCEntry{Title: e.Title, Href: href, Children: children})
		}
		return out
	}
	return mapEntries(entries)
}

func singlePageHeadingContents(chapters []*mergeChapter) []epub.TOCEntry {
	used := make(map[string]bool)
	for _, id := range chromeIDs {
		used[id] = true
	}
	for _, ch := range chapters {
		collectIDs(ch.doc, used)
		used[ch.anchor] = true
	}
	var all []flatHeading
	counter := 0
	for _, ch := range chapters {
		body := findBodyNode(ch.doc)
		if body == nil {
			continue
		}
		var walk func(*gohtml.Node)
		walk = func(n *gohtml.Node) {
			if n.Type == gohtml.ElementNode {
				if isStructuralSkip(n) {
					return
				}
				if level := headingLevel(n.Data); level > 0 {
					if title := strings.TrimSpace(textContent(n)); title != "" {
						id := nodeAttr(n, "id")
						if id == "" {
							id = uniqueID(slugify(title), used, &counter)
							setAttr(n, "id", id)
						}
						all = append(all, flatHeading{level: level, title: title, id: id})
					}
					return
				}
			}
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				walk(c)
			}
		}
		walk(body)
	}
	return nestHeadings(all, "")
}

func hasChapterAnchor(ch *mergeChapter) bool {
	body := findBodyNode(ch.doc)
	if body == nil {
		return false
	}
	for c := body.FirstChild; c != nil; c = c.NextSibling {
		if nodeAttr(c, "id") == ch.anchor {
			return true
		}
	}
	return false
}

func isPagedChapters(chapters []*mergeChapter) bool {
	if len(chapters) == 0 {
		return false
	}
	for _, ch := range chapters {
		body := findBodyNode(ch.doc)
		if body == nil {
			return false
		}
		found := false
		for c := body.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == gohtml.ElementNode && nodeHasClass(c, "dht-page") {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func renderSingleContents(entries []epub.TOCEntry, depth int) string {
	label := html.EscapeString(i18n.S("Table of contents"))
	// nav.close as its word, the search panel's close: a typed cross is not the vocabulary's
	// drawing (ICON-SET rule 1, ICON-RENDER rule 7), and the visible word is the accessible name.
	closeLabel := html.EscapeString(i18n.S("Close"))
	var sb strings.Builder
	fmt.Fprintf(&sb, `<nav id="dht-contents" class="dht-contents" lang="%s"%s aria-label="%s" hidden><div class="dht-contents-head"><strong>%s</strong><button id="dht-contents-close" type="button">%s</button></div>`, i18n.Language(), chromeDirAttr(), label, label, closeLabel)
	renderTOCList(&sb, entries, "", depth, 1, 1)
	sb.WriteString("</nav>\n")
	return sb.String()
}

const singleContentsCSS = `<style id="dht-contents-css">
  /* APP-STYLE section 5: the panel's black shadow and the links' grey hover tint stay outside
     the palette on purpose - neutral overlays that carry no text, so one value reads on all four
     reading themes while the words keep their themed colours. */
  .dht-contents[hidden] { display: none; }
  .dht-contents { position: fixed; z-index: 9998; inset-block-start: 3.5rem; inset-inline-start: 0; width: min(20rem, calc(100vw - 1rem)); max-height: calc(100dvh - 4rem); overflow: auto; box-sizing: border-box; padding: .75rem 1rem; background: var(--dht-bar-bg); color: var(--dht-bar-fg); border: 1px solid var(--dht-border); box-shadow: 0 6px 20px #0003; font: 14px/1.5 "Segoe UI", system-ui, sans-serif; }
  .dht-contents-head { display: flex; align-items: center; justify-content: space-between; gap: 1rem; }
  .dht-contents-head button { color: inherit; background: transparent; border: 1px solid var(--dht-border); border-radius: 4px; font: inherit; padding: .25em .7em; cursor: pointer; }
  .dht-contents ul { list-style: none; padding-inline-start: 1rem; margin: .3rem 0; }
  .dht-contents > ul { padding-inline-start: 0; }
  .dht-contents li { margin: .2rem 0; }
  .dht-contents a { color: inherit; display: inline-block; padding: .2rem .4rem; border-radius: 4px; text-decoration: none; }
  .dht-contents a:hover, .dht-contents a[aria-current="location"] { color: var(--dht-accent); background: rgba(127,127,127,.14); }
  .dht-contents :focus-visible, #dht-contents-button:focus-visible { outline: 2px solid var(--dht-accent); outline-offset: 2px; }
  main.dht-single [id] { scroll-margin-top: 4rem; }
</style>
`

const singleContentsScript = `<script id="dht-contents-script">(function(){
  var button = document.getElementById('dht-contents-button');
  var panel = document.getElementById('dht-contents');
  var close = document.getElementById('dht-contents-close');
  if (!button || !panel) return;
  function show(open, focus) {
    panel.hidden = !open;
    button.setAttribute('aria-expanded', open ? 'true' : 'false');
    if (focus) (open ? close : button).focus();
  }
  button.addEventListener('click', function(){ show(panel.hidden, true); });
  close.addEventListener('click', function(){ show(false, true); });
  document.addEventListener('keydown', function(e){ if (e.key === 'Escape' && !panel.hidden) show(false, true); });
  document.addEventListener('click', function(e){ if (!panel.hidden && !panel.contains(e.target) && !button.contains(e.target)) show(false, false); });
  var links = Array.prototype.slice.call(panel.querySelectorAll('a[href^="#"]'));
  var targets = links.map(function(link){
    var id;
    try { id = decodeURIComponent(link.getAttribute('href').slice(1)); } catch(e) { return null; }
    return document.getElementById(id);
  });
  panel.addEventListener('click', function(e){ if (e.target.closest('a[href^="#"]')) show(false, false); });
  var queued = false;
  function update(){
    queued = false;
    var best = -1;
    for (var i = 0; i < targets.length; i++) {
      if (targets[i] && targets[i].getBoundingClientRect().top <= 90) best = i;
    }
    if (best < 0) { for (var j = 0; j < targets.length; j++) { if (targets[j]) { best = j; break; } } }
    links.forEach(function(link, i){
      if (i === best) { link.setAttribute('aria-current', 'location'); }
      else { link.removeAttribute('aria-current'); }
    });
    if (best >= 0) {
      for (var p = links[best].parentElement; p && p !== panel; p = p.parentElement) {
        if (p.tagName === 'DETAILS') p.open = true;
      }
    }
  }
  window.addEventListener('scroll', function(){ if (!queued) { queued = true; requestAnimationFrame(update); } }, {passive:true});
  window.addEventListener('load', update);
  window.addEventListener('hashchange', update);
  update();
})();</script>
`
