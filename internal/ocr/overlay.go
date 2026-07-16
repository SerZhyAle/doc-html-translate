package ocr

import (
	"fmt"
	"image"
	_ "image/gif"  // register decoders for colour sampling
	_ "image/jpeg" //
	_ "image/png"  //
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "golang.org/x/image/tiff" // extracted PDF images may be TIFF
	_ "golang.org/x/image/webp" // EPUB images may be WebP
	gohtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// ocrCSS styles the overlay: a positioned container sized to the image (inline-size
// container query so font-size can scale with it), and opaque plates covering the source
// text. Injected once per page into <head>.
//
// The container MUST be display:block with an explicit width (not inline-block): a
// shrink-to-fit box with container-type:inline-size collapses to zero inline size (size
// containment removes the content's contribution), which hides the image and every plate.
// The image is width:100% - plus margin:0 and max-height:none so a page-level `img` reset can't
// offset or shrink it below the container (which would drift the plates vertically) - so the
// percent-positioned plates line up with it. Plates top-align their text (align-items:flex-start)
// so it starts at the source line and longer translations grow downward. This mirrors the
// extension's .ocr-overlay (see docs/PARITY.md and ocr-overlay.css).
const ocrCSS = `.ocr-fig{position:relative;display:block;width:100%;max-width:100%;margin:0 auto;container-type:inline-size;line-height:1.1}
.ocr-fig>img{display:block;width:100%;height:auto;margin:0;max-height:none}
.ocr-box{position:absolute;box-sizing:border-box;overflow:hidden;background:#fff;color:#111;padding:0.05em 0.15em;border-radius:2px;display:flex;align-items:flex-start;justify-content:center;text-align:center;white-space:pre-wrap;overflow-wrap:anywhere;word-break:break-word;font-family:"Segoe UI",system-ui,Arial,sans-serif}`

// OverlayFile OCRs every local <img> in the HTML file and rewrites each into a positioned
// container with opaque, translatable text plates over the image. Returns the number of
// images overlaid. Best-effort: an image that fails OCR is left untouched, and the file is
// only rewritten when at least one overlay was added.
//
// onProgress, when non-nil, is called before each image with the number already handled
// and the total found. Recognizing one image shells out to Tesseract and takes about a
// second, so a single-page book of scans keeps this loop busy for a long while; the
// callback lets the caller show that without this package having to know about logging
// (the same shape as translator.ProgressReporter).
func OverlayFile(bin, htmlPath, lang, dataDir string, onProgress func(done, total int)) (int, error) {
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
	imgs := collectImgs(doc)
	for i, img := range imgs {
		if onProgress != nil {
			onProgress(i, len(imgs))
		}
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
		wrapImage(img, res, decodeImage(imgFile))
		count++
	}
	if onProgress != nil {
		onProgress(len(imgs), len(imgs))
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
// recognized block, positioned in percent of the image's natural size. When srcImg is
// non-nil each plate also borrows the block's paper/ink colours from the image so the
// overlay blends in (best-effort; nil srcImg keeps the default white plate).
func wrapImage(img *gohtml.Node, res Result, srcImg image.Image) {
	parent := img.Parent
	if parent == nil {
		return
	}
	wrap := &gohtml.Node{
		Type: gohtml.ElementNode, Data: "span", DataAtom: atom.Span,
		Attr: []gohtml.Attribute{
			{Key: "class", Val: "ocr-fig"},
			// aspect-ratio keeps the container sized to the image even before a
			// lazy-loaded <img> arrives, so the percent-positioned plates line up
			// immediately (mirrors the extension's buildOverlay). Width/Height are
			// validated > 0 by the caller before wrapImage runs.
			{Key: "style", Val: fmt.Sprintf("aspect-ratio:%d / %d", res.Width, res.Height)},
		},
	}
	parent.InsertBefore(wrap, img)
	parent.RemoveChild(img)
	wrap.AppendChild(img)

	for _, b := range res.Blocks {
		style := percentStyle(b, res.Width, res.Height)
		if srcImg != nil {
			if bg, ink, ok := blockColors(srcImg, b); ok {
				style += ";background:" + bg + ";color:" + ink
			}
		}
		box := &gohtml.Node{
			Type: gohtml.ElementNode, Data: "span", DataAtom: atom.Span,
			Attr: []gohtml.Attribute{
				{Key: "class", Val: "ocr-box"},
				{Key: "style", Val: style},
			},
		}
		box.AppendChild(&gohtml.Node{Type: gohtml.TextNode, Data: b.Text})
		wrap.AppendChild(box)
	}
}

// fontFitFactor shrinks the plate font below the block's raw line height so the recognized
// text reliably fits inside the block box (which is what actually covers the source - the
// opaque box is sized by min-height, independent of the font). Without it a tall title
// block wraps to more lines than the source and the plate grows past its region, colliding
// with the next plate. 0.92 keeps plate text close to the source size while still absorbing
// font-metric and word-wrap slack. Shared with the extension's ocr-overlay.js FONT_FIT (see
// docs/PARITY.md).
const fontFitFactor = 0.92

// percentStyle positions a plate as percentages of the image dimensions and sizes its font
// from the block's line height (in cqw, i.e. percent of the container width), scaled by
// fontFitFactor so the text fits the block instead of overflowing it.
func percentStyle(b Block, w, h int) string {
	if w <= 0 || h <= 0 {
		return ""
	}
	return fmt.Sprintf(
		"left:%.2f%%;top:%.2f%%;width:%.2f%%;min-height:%.2f%%;font-size:%.2fcqw",
		pct(b.X0, w), pct(b.Y0, h), pct(b.X1-b.X0, w), pct(b.Y1-b.Y0, h), pct(b.LineH, w)*fontFitFactor,
	)
}

func pct(v, total int) float64 {
	if total <= 0 {
		return 0
	}
	return float64(v) / float64(total) * 100
}

// ---- Adaptive plate colours ------------------------------------------------
// decodeImage decodes an image file for colour sampling; nil on any failure (plates then
// keep the default white/dark CSS).
func decodeImage(path string) image.Image {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	im, _, err := image.Decode(f)
	if err != nil {
		return nil
	}
	return im
}

// blockColors samples the source image so a plate can borrow the block's background
// ("paper") and text ("ink") colours - the overlay then blends into the document instead
// of being a white patch. bg is the median colour over the whole block (text is the
// minority); ink is the mean of the pixels that stand out from bg within the FIRST line
// (real text lives there, not figures lower in a merged block - "the colour of the
// original's first letter"), with a near-black/near-white fallback that guarantees
// contrast. ok=false leaves the CSS default. Mirrors the extension's ocr-overlay.js
// blockColors (see docs/PARITY.md - keep the two in sync).
func blockColors(img image.Image, b Block) (bg, ink string, ok bool) {
	bnds := img.Bounds()
	x0 := clampInt(b.X0, bnds.Min.X, bnds.Max.X)
	y0 := clampInt(b.Y0, bnds.Min.Y, bnds.Max.Y)
	x1 := clampInt(b.X1, bnds.Min.X, bnds.Max.X)
	y1 := clampInt(b.Y1, bnds.Min.Y, bnds.Max.Y)
	if x1-x0 < 2 || y1-y0 < 2 {
		return "", "", false
	}
	ar, ag, ab := samplePixels(img, x0, y0, x1, y1)
	if len(ar) == 0 {
		return "", "", false
	}
	bgR, bgG, bgB := medianOf(ar), medianOf(ag), medianOf(ab)

	lh := b.LineH
	if lh < 1 {
		lh = y1 - y0
	}
	yFirst := y0 + int(float64(lh)*1.3)
	if yFirst > y1 {
		yFirst = y1
	}
	fr, fg, fb := samplePixels(img, x0, y0, x1, yFirst)
	var sr, sg, sb, c int
	for i := range fr {
		if absInt(fr[i]-bgR)+absInt(fg[i]-bgG)+absInt(fb[i]-bgB) > 90 {
			sr += fr[i]
			sg += fg[i]
			sb += fb[i]
			c++
		}
	}
	minInk := len(fr) * 15 / 1000
	if minInk < 6 {
		minInk = 6
	}
	var inkR, inkG, inkB int
	if c >= minInk {
		inkR, inkG, inkB = sr/c, sg/c, sb/c
	} else {
		inkR, inkG, inkB = fallbackInk(bgR, bgG, bgB)
	}
	if absInt(luma(inkR, inkG, inkB)-luma(bgR, bgG, bgB)) < 55 {
		inkR, inkG, inkB = fallbackInk(bgR, bgG, bgB)
	}
	return fmt.Sprintf("rgb(%d,%d,%d)", bgR, bgG, bgB),
		fmt.Sprintf("rgb(%d,%d,%d)", inkR, inkG, inkB), true
}

func fallbackInk(r, g, b int) (int, int, int) {
	if luma(r, g, b) > 140 {
		return 17, 17, 17
	}
	return 240, 240, 240
}

// samplePixels returns sub-sampled 8-bit R,G,B channels for the rectangle, capped at
// ~6000 samples so large blocks stay cheap. Fully transparent pixels are skipped.
func samplePixels(img image.Image, x0, y0, x1, y1 int) (rs, gs, bs []int) {
	n := (x1 - x0) * (y1 - y0)
	if n <= 0 {
		return nil, nil, nil
	}
	step := n / 6000
	if step < 1 {
		step = 1
	}
	i := 0
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			if i%step == 0 {
				cr, cg, cb, ca := img.At(x, y).RGBA()
				if ca>>8 >= 128 {
					rs = append(rs, int(cr>>8))
					gs = append(gs, int(cg>>8))
					bs = append(bs, int(cb>>8))
				}
			}
			i++
		}
	}
	return rs, gs, bs
}

func medianOf(v []int) int {
	if len(v) == 0 {
		return 0
	}
	s := append([]int(nil), v...)
	sort.Ints(s)
	return s[len(s)/2]
}

func luma(r, g, b int) int { return (299*r + 587*g + 114*b) / 1000 }

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
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
