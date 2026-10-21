package htmlgen

// A document picture that cannot be shown (ICON-EXTERNAL section 13 F): the images inside a book are
// the reader's own local content, and one that is missing or broken shows the vocabulary's
// content.image with the picture's alt text where the browser would draw its broken-image icon. The
// alt text is the book's own words, so the fallback adds no string of this product. The browser
// extension's viewer does the same (extension/src/missing-image.js); docs/PARITY.md "Missing
// document pictures".

// missingImageCSS styles the stand-in. It inherits the reading theme's ink and uses no palette
// value of its own; the size follows the surrounding text, since the chrome's size tokens are
// scoped to the bars and not to the book.
const missingImageCSS = `
  .dht-missing-img {
    display: inline-flex; align-items: center; gap: .5em; box-sizing: border-box; max-width: 100%;
    padding: .35em .7em; border: 1px dashed var(--dht-border); border-radius: .25em;
    color: var(--dht-fg); font-size: .8em; line-height: 1.35; text-indent: 0; vertical-align: middle; overflow-wrap: anywhere;
  }
  .dht-missing-img .dht-glyph { width: 1.5em; height: 1.5em; flex: none; }
`

// missingImageScript swaps a picture that fails to load for its stand-in. An error event does not
// bubble, so the listener is a capture-phase one on the document; it sits in the page head, ahead of
// the first <img>, because the single page puts its reader script after the whole book and a picture
// can fail before that script runs. The failed picture's src is never touched, so nothing is
// requested again.
//
// A picture inside an OCR plate container keeps the browser's rendering: the plates are pinned to the
// picture's box, which the stand-in would not keep.
var missingImageScript = `
<script id="dht-missing-image">//<![CDATA[
(function(){
	var GLYPH = ` + jsString(glyphPath("content.image")) + `;
	var SVGNS = "http://www.w3.org/2000/svg";
	function standIn(alt){
		var box = document.createElement("span");
		box.className = "dht-missing-img";
		var svg = document.createElementNS(SVGNS, "svg");
		svg.setAttribute("class", "dht-glyph");
		svg.setAttribute("viewBox", "0 0 24 24");
		svg.setAttribute("aria-hidden", "true");
		svg.setAttribute("focusable", "false");
		var path = document.createElementNS(SVGNS, "path");
		path.setAttribute("fill", "currentColor");
		path.setAttribute("d", GLYPH);
		svg.appendChild(path);
		box.appendChild(svg);
		if (alt) {
			var text = document.createElement("span");
			text.textContent = alt;
			box.appendChild(text);
		}
		return box;
	}
	document.addEventListener("error", function(e){
		var img = e.target;
		if (!img || img.tagName !== "IMG" || !img.parentNode || img.closest(".ocr-fig")) return;
		img.parentNode.replaceChild(standIn((img.getAttribute("alt") || "").trim()), img);
	}, true);
})();
//]]></script>
`
