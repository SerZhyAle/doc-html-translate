package ocr

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	gohtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// ocrCSS styles the overlay: a positioned container sized to the image (inline-size
// container query so font-size can scale with it), and opaque plates covering the source
// text. Injected once per page into <head>.
const ocrCSS = `.ocr-fig{position:relative;display:inline-block;max-width:100%;container-type:inline-size;line-height:1.1}
.ocr-fig>img{display:block;max-width:100%;height:auto}
.ocr-box{position:absolute;box-sizing:border-box;overflow:hidden;background:#fff;color:#111;padding:0.05em 0.15em;border-radius:2px;display:flex;align-items:center;justify-content:center;text-align:center;white-space:pre-wrap;overflow-wrap:anywhere;word-break:break-word;font-family:"Segoe UI",system-ui,Arial,sans-serif}`

// OverlayFile OCRs every local <img> in the HTML file and rewrites each into a positioned
// container with opaque, translatable text plates over the image. Returns the number of
// images overlaid. Best-effort: an image that fails OCR is left untouched, and the file is
// only rewritten when at least one overlay was added.
func OverlayFile(bin, htmlPath, lang, dataDir string) (int, error) {
	f, err := os.Open(htmlPath)
	if err != nil {
		return 0, err
	}
	doc, perr := gohtml.Parse(f)
	f.Close()
	if perr != nil {
		return 0, perr
	}

	baseDir := filepath.Dir(htmlPath)
	count := 0
	for _, img := range collectImgs(doc) {
		src := attrVal(img, "src")
		if src == "" || isExternal(src) {
			continue
		}
		imgFile := filepath.Join(baseDir, filepath.FromSlash(src))
		if _, err := os.Stat(imgFile); err != nil {
			continue
		}
		res, err := Recognize(bin, imgFile, lang, dataDir)
		if err != nil || res.Width <= 0 || res.Height <= 0 || len(res.Blocks) == 0 {
			continue
		}
		wrapImage(img, res)
		count++
	}
	if count == 0 {
		return 0, nil
	}
	ensureStyle(doc)

	out, err := os.Create(htmlPath)
	if err != nil {
		return 0, err
	}
	defer out.Close()
	if err := gohtml.Render(out, doc); err != nil {
		return 0, err
	}
	return count, nil
}

func collectImgs(n *gohtml.Node) []*gohtml.Node {
	var imgs []*gohtml.Node
	var walk func(*gohtml.Node)
	walk = func(node *gohtml.Node) {
		if node.Type == gohtml.ElementNode && node.DataAtom == atom.Img {
			imgs = append(imgs, node)
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return imgs
}

func attrVal(n *gohtml.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

func isExternal(src string) bool {
	s := strings.ToLower(src)
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") ||
		strings.HasPrefix(s, "//") || strings.HasPrefix(s, "data:")
}

// wrapImage moves img into a new .ocr-fig container and appends one .ocr-box plate per
// recognized block, positioned in percent of the image's natural size.
func wrapImage(img *gohtml.Node, res Result) {
	parent := img.Parent
	if parent == nil {
		return
	}
	wrap := &gohtml.Node{
		Type: gohtml.ElementNode, Data: "span", DataAtom: atom.Span,
		Attr: []gohtml.Attribute{{Key: "class", Val: "ocr-fig"}},
	}
	parent.InsertBefore(wrap, img)
	parent.RemoveChild(img)
	wrap.AppendChild(img)

	for _, b := range res.Blocks {
		box := &gohtml.Node{
			Type: gohtml.ElementNode, Data: "span", DataAtom: atom.Span,
			Attr: []gohtml.Attribute{
				{Key: "class", Val: "ocr-box"},
				{Key: "style", Val: percentStyle(b, res.Width, res.Height)},
			},
		}
		box.AppendChild(&gohtml.Node{Type: gohtml.TextNode, Data: b.Text})
		wrap.AppendChild(box)
	}
}

// percentStyle positions a plate as percentages of the image dimensions and sizes its
// font from the block's line height (in cqw, i.e. percent of the container width).
func percentStyle(b Block, w, h int) string {
	if w <= 0 || h <= 0 {
		return ""
	}
	return fmt.Sprintf(
		"left:%.2f%%;top:%.2f%%;width:%.2f%%;min-height:%.2f%%;font-size:%.2fcqw",
		pct(b.X0, w), pct(b.Y0, h), pct(b.X1-b.X0, w), pct(b.Y1-b.Y0, h), pct(b.LineH, w),
	)
}

func pct(v, total int) float64 {
	if total <= 0 {
		return 0
	}
	return float64(v) / float64(total) * 100
}

// ensureStyle appends the overlay stylesheet to <head> (or <html>/document if there is no
// head), once per document.
func ensureStyle(doc *gohtml.Node) {
	var head, htmlEl *gohtml.Node
	var walk func(*gohtml.Node)
	walk = func(n *gohtml.Node) {
		if n.Type == gohtml.ElementNode {
			switch n.DataAtom {
			case atom.Head:
				if head == nil {
					head = n
				}
			case atom.Html:
				if htmlEl == nil {
					htmlEl = n
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	target := head
	if target == nil {
		target = htmlEl
	}
	if target == nil {
		target = doc
	}
	style := &gohtml.Node{Type: gohtml.ElementNode, Data: "style", DataAtom: atom.Style}
	style.AppendChild(&gohtml.Node{Type: gohtml.TextNode, Data: ocrCSS})
	target.AppendChild(style)
}
