package htmlgen

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"doc-html-translate/internal/epub"
	"doc-html-translate/internal/fsutil"
	"doc-html-translate/internal/i18n"
	gohtml "golang.org/x/net/html"
)

// WriteSearchIndex writes the final translated/OCR text as a local script. file:// pages
// cannot reliably fetch sibling files, but they can load a sibling script.
const SearchIndexName = "dht-search-index.js"

func WriteSearchIndex(book *epub.Book, outputDir string) error {
	type page struct {
		Href  string   `json:"href"`
		Text  []string `json:"text"`
		Plate []bool   `json:"plate"`
	}
	var pages []page
	for _, href := range book.SpineHrefs() {
		full := href
		if book.BasePath != "" && book.BasePath != "." {
			full = book.BasePath + "/" + href
		}
		data, err := os.ReadFile(filepath.Join(outputDir, filepath.FromSlash(full)))
		if err != nil {
			return err
		}
		doc, err := gohtml.Parse(bytes.NewReader(data))
		if err != nil {
			return err
		}
		texts, plates := searchableNodes(doc)
		pages = append(pages, page{Href: epub.URLPath(full), Text: texts, Plate: plates})
	}
	encoded, err := json.Marshal(struct {
		Key   string `json:"key"`
		Pages []page `json:"pages"`
	}{readerKey(book), pages})
	if err != nil {
		return err
	}
	return fsutil.WriteFile(filepath.Join(outputDir, SearchIndexName), []byte(fmt.Sprintf("window.dhtSearchIndex=%s;\n", encoded)), 0o644)
}

// searchableNodes mirrors the browser's text-node walk. Reader chrome and hidden
// markup are excluded; OCR plates remain part of the document text.
func searchableNodes(root *gohtml.Node) ([]string, []bool) {
	var out []string
	var plates []bool
	var walk func(*gohtml.Node, bool, bool)
	walk = func(n *gohtml.Node, skip, plate bool) {
		if n.Type == gohtml.ElementNode {
			switch n.Data {
			case "script", "style", "template", "noscript", "svg", "head", "nav", "button", "select", "option":
				skip = true
			}
			for _, a := range n.Attr {
				if a.Key == "class" && (strings.Contains(" "+a.Val+" ", " dht-navbar ") || strings.Contains(" "+a.Val+" ", " dht-toolbar ")) {
					skip = true
				}
				if a.Key == "class" && (strings.Contains(" "+a.Val+" ", " ocr-plate ") || strings.Contains(" "+a.Val+" ", " ocr-overlay ")) {
					plate = true
				}
				if a.Key == "hidden" || a.Key == "aria-hidden" && a.Val == "true" {
					skip = true
				}
			}
		}
		if n.Type == gohtml.TextNode && !skip && strings.TrimSpace(n.Data) != "" {
			out = append(out, n.Data)
			plates = append(plates, plate)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c, skip, plate)
		}
	}
	walk(root, false, false)
	return out, plates
}

const searchCSS = `<style id="dht-search-css">
.dht-search-panel{position:fixed;z-index:10001;inset-block-start:3.5em;inset-inline-end:1em;width:min(30em,calc(100vw - 2em));max-height:75vh;overflow:auto;padding:1em;background:var(--dht-bar-bg);color:var(--dht-bar-fg);border:1px solid var(--dht-border);box-shadow:0 8px 24px #0005;font:14px/1.4 system-ui,sans-serif}
.dht-search-panel[hidden]{display:none}.dht-search-panel input{width:100%;box-sizing:border-box;padding:.5em;background:var(--dht-bg);color:var(--dht-fg);border:1px solid var(--dht-border)}
.dht-search-panel ol{padding-inline-start:1.5em}.dht-search-panel li{margin:.4em 0}.dht-search-panel button{cursor:pointer}.dht-search-panel .dht-result{background:none;border:0;color:var(--dht-link);text-align:start;font:inherit}
/* The close button is a bare word at the panel's corner; padding puts it over the 24px
   target-size floor (WCAG 2.2, 2.5.8). */
#dht-search-close{padding:.25em .7em}
mark.dht-search-hit{background:#ffdc62!important;color:#191919!important;outline:1px solid #7b4a00}mark.dht-search-current{background:#ff8b46!important;color:#191919!important;outline:3px solid #3156c9}
</style>`

func searchControlsHTML() string {
	labels := []string{"Search", "Search text", "Scope", "This page", "Whole book", "Close"}
	for i, label := range labels {
		labels[i] = htmlEscape(i18n.S(label))
	}
	return fmt.Sprintf(`<button id="dht-search-button" class="dht-btn" type="button" aria-controls="dht-search-panel" aria-expanded="false" title="%[1]s">%[1]s</button><section id="dht-search-panel" class="dht-search-panel" lang="%[7]s"%[8]s hidden><label for="dht-search-input">%[2]s</label><input id="dht-search-input" type="search" autocomplete="off"><label for="dht-search-scope">%[3]s</label><select id="dht-search-scope"><option value="page">%[4]s</option><option value="book">%[5]s</option></select><button id="dht-search-close" type="button" aria-label="%[6]s">%[6]s</button><p id="dht-search-status" role="status" aria-live="polite"></p><ol id="dht-search-results"></ol></section>`, labels[0], labels[1], labels[2], labels[3], labels[4], labels[5], i18n.Language(), chromeDirAttr())
}

func htmlEscape(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;").Replace(s)
}

func searchScript(bookKey, self, indexHref string) string {
	s := strings.ReplaceAll(searchRuntime, "__BOOK__", jsString(bookKey))
	s = strings.ReplaceAll(s, "__SELF__", jsString(self))
	s = strings.ReplaceAll(s, "__INDEX__", jsString(indexHref))
	s = strings.ReplaceAll(s, "__INDEX_FILE__", jsString(SearchIndexName))
	labels := []string{"This page", "Whole book", "Matches: {1} - {2}", "OCR text plates", "Scanned pages need OCR text plates", "Enter text to search", "Searching whole book", "Search index unavailable", "showing first 500"}
	for i, label := range labels {
		labels[i] = i18n.S(label)
	}
	encoded, _ := json.Marshal(labels) // fixed literal strings; encoding cannot fail
	return strings.ReplaceAll(s, "__WORDS__", string(encoded))
}

const searchRuntime = `<script id="dht-search-script">(function(){
var key=__BOOK__,self=__SELF__,indexHref=__INDEX__,words=__WORDS__;
var button=document.getElementById('dht-search-button'),panel=document.getElementById('dht-search-panel');
if(!button||!panel)return;
var input=document.getElementById('dht-search-input'),scope=document.getElementById('dht-search-scope');
var status=document.getElementById('dht-search-status'),list=document.getElementById('dht-search-results');
var marks=[],hits=[],serial=0,lang=document.documentElement.lang||undefined;
var pageOpt=scope.querySelector('[value="page"]');if(!indexHref||!self){if(pageOpt)pageOpt.remove();scope.value='book';}
function textNodes(){
 var root=document.body,walker=document.createTreeWalker(root,NodeFilter.SHOW_TEXT);
 var nodes=[],n;
 while((n=walker.nextNode())){
  if(!n.nodeValue.trim())continue;
  if(n.parentElement.closest('script,style,template,noscript,svg,nav,button,select,.dht-navbar,.dht-toolbar,.dht-search-panel,[hidden],[aria-hidden="true"]'))continue;
  nodes.push(n);
 }
 return nodes;
}
function fold(s){try{return s.toLocaleLowerCase(lang)}catch(e){return s.toLowerCase()}}
// Lowering can change length (Turkic dotted I lowers to two code units under most locales),
// so every lowered code unit records the offset of the character it came from and matches
// report offsets in the ORIGINAL string - a highlight must land on what was matched.
function occurrences(s,q){var needle=fold(q);if(!needle)return[];var text='',map=[],i=0;
 for(const ch of s){var f=fold(ch);for(var k=0;k<f.length;k++)map.push(i);text+=f;i+=ch.length}
 var out=[],from=text.indexOf(needle);
 while(from>=0){out.push(map[from]);from=text.indexOf(needle,from+Math.max(needle.length,1))}
 return out;
}
function clearMarks(){marks.forEach(function(m){if(m.isConnected)m.replaceWith(document.createTextNode(m.textContent))});marks=[];document.body.normalize()}
function markLocal(q){clearMarks();var nodes=textNodes(),found=[],ordinal=0;
 nodes.forEach(function(node){var positions=occurrences(node.nodeValue,q),original=node.nodeValue;
  positions.forEach(function(at){found.push({ordinal:ordinal++,context:original.slice(Math.max(0,at-35),Math.min(original.length,at+q.length+45)).trim(),plate:!!node.parentElement.closest('.ocr-plate,.ocr-overlay')})});
  var made=[];for(var i=positions.length-1;i>=0;i--){var range=document.createRange();range.setStart(node,positions[i]);range.setEnd(node,positions[i]+q.length);var mark=document.createElement('mark');mark.className='dht-search-hit';range.surroundContents(mark);made.unshift(mark)}marks.push.apply(marks,made);
 });return found;
}
function choose(n){marks.forEach(function(m){m.classList.remove('dht-search-current')});if(marks[n]){marks[n].classList.add('dht-search-current');marks[n].scrollIntoView({block:'center'});marks[n].setAttribute('tabindex','-1');marks[n].focus({preventScroll:true})}}
var cur=-1;
function open(i){if(!hits.length)return;cur=(i%hits.length+hits.length)%hits.length;var hit=hits[cur];
 if(!hit.href||hit.href===self){choose(hit.ordinal);return}
 var url=new URL(hit.href,new URL(indexHref,location.href));url.searchParams.set('dhtq',input.value.trim());url.searchParams.set('dhtn',String(hit.ordinal));location.href=url.href;
}
function render(q,pages,local){list.replaceChildren();hits=[];cur=-1;var allPlate=true,hasPlates=local?!!document.querySelector('.ocr-plate,.ocr-overlay'):false;
 if(local){hits=local.map(function(x){return {href:self,ordinal:x.ordinal,context:x.context,plate:x.plate}});allPlate=hits.every(function(x){return x.plate})}
 else {
 hasPlates=pages.some(function(p){return p.plate&&p.plate.indexOf(true)>=0});
 pages.forEach(function(page){var ordinal=0;page.text.forEach(function(value,nodeIndex){
  occurrences(value,q).forEach(function(at){var context=value.slice(Math.max(0,at-35),Math.min(value.length,at+q.length+45)).trim();
   var plate=page.plate&&page.plate[nodeIndex];if(!plate)allPlate=false;
   hits.push({href:page.href,ordinal:ordinal++,context:context,plate:plate});
  });
 })})}
 status.textContent=words[2].replace('{1}',hits.length).replace('{2}',scope.value==='page'?words[0]:words[1])
  +(hits.length&&allPlate?' ('+words[3]+')':'')+(!hits.length&&hasPlates?' ('+words[4]+')':'');
 hits.slice(0,500).forEach(function(hit,i){var li=document.createElement('li'),b=document.createElement('button');b.type='button';b.className='dht-result';b.textContent=(hit.href&&hit.href!==self?hit.href+' - ':'')+hit.context;b.addEventListener('click',function(){open(i)});li.append(b);list.append(li)});
 if(hits.length>500)status.textContent+='; '+words[8];
}
function run(){var q=input.value.trim(),id=++serial;clearMarks();list.replaceChildren();if(!q){status.textContent=words[5];return}
 var local=markLocal(q);
 if(scope.value==='page'||!indexHref){render(q,[],local);return}
 status.textContent=words[6]+'..';
 function loaded(){if(id!==serial)return;var data=window.dhtSearchIndex;if(!data||data.key!==key){status.textContent=words[7];return}render(q,data.pages)}
 if(window.dhtSearchIndex){loaded();return}
 var script=document.createElement('script');script.src=new URL(__INDEX_FILE__,new URL(indexHref,location.href)).href;script.onload=loaded;script.onerror=function(){if(id===serial)status.textContent=words[7]};document.head.append(script);
}
button.addEventListener('click',function(){panel.hidden=!panel.hidden;button.setAttribute('aria-expanded',String(!panel.hidden));if(!panel.hidden){input.focus();if(input.value.trim())run()}else clearMarks()});
document.getElementById('dht-search-close').addEventListener('click',function(){panel.hidden=true;button.setAttribute('aria-expanded','false');clearMarks();button.focus()});
input.addEventListener('input',function(){clearTimeout(input._timer);input._timer=setTimeout(run,180)});scope.addEventListener('change',run);
input.addEventListener('keydown',function(e){if(e.key!=='Enter')return;e.preventDefault();if(!hits.length){run();if(hits.length)open(0);return}open(cur+(e.shiftKey?-1:1))});
document.addEventListener('keydown',function(e){if(e.key==='Escape'&&!panel.hidden){panel.hidden=true;button.setAttribute('aria-expanded','false');clearMarks();button.focus()}});
var params=new URLSearchParams(location.search),from=params.get('dhtq');if(from){input.value=from;panel.hidden=false;button.setAttribute('aria-expanded','true');scope.value='page';var restore=function(){run();var n=parseInt(params.get('dhtn'),10);if(Number.isFinite(n))setTimeout(function(){choose(n)},0)};if(document.readyState==='loading')document.addEventListener('DOMContentLoaded',restore,{once:true});else restore()}
})();</script>`
