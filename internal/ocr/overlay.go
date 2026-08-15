package ocr

import (
	"fmt"
	"image"
	_ "image/gif"  // register decoders for colour sampling
	_ "image/jpeg" //
	_ "image/png"  //
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"

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
// percent-positioned plates line up with it. Plates centre their text (align-items:center) so
// short text sits in the middle of its region rather than pinned to the top with empty space
// below; the runtime re-fit (ocrScript) then shrinks the font so longer text - a reflow, or the
// page translator swapping in a longer string - fits instead of clipping. Mirrors the extension's
// .ocr-overlay (see docs/PARITY.md and ocr-overlay.css).
//
// The opaque paper is on the plate box. It was on an inline span hugging the string between
// 2026-08-13 and this change, so that the paper took the shape of the rendered words rather than
// of the block rectangle, and the corpus was asked which of the two a reader is better served by:
// over 46 scenes the box carrier left a mean 17% of the source lettering still showing under a
// plate and the string carrier left 93% - a plate that concealed almost nothing it covered, and a
// regression against the lab's recorded 0.28 bound. The cost of coming back is real and is the
// reason the string carrier was tried: a block box is wider than centred copy on its last line, so
// the plate paints paper beside that line - measured at 91 px either side of a 984 px caption.
// That is a patch beside one line against the whole document showing through every plate, and it
// is bounded by the coverage rule in tesseract.go, which stops a plate from being a page.
//
// The padding is `0.08em 0.28em` because that is what the paper had when the lab's concealment
// bound was measured, and it is load-bearing rather than cosmetic. The lab reads residual ink by
// asking whether the *rendered* page still has ink where the source had it, so a plate's own
// lettering counts against it wherever the two coincide - and the padding is what decides how large
// the runtime fit grows that lettering. Measured on samson-and-delilah-03-scroll, whose plate rects
// are byte-identical either way: `0.05em 0.15em` grew one plate's font from 56.4 px to 60.0 px and
// took the scene from 0.2705 to 0.2841, over the recorded 0.28 bound. Changing it needs a lab run,
// not an opinion.
//
// `print-color-adjust:exact` on the plate is the one declaration that makes a printed page hold
// together. A browser omits background colours when it prints, and the plate is nothing but an
// opaque background carrying text - so without it the translated string prints over the still
// legible source lettering and the sheet is unreadable. Print is also the one output where the
// reader cannot toggle the overlay off (html.dht-ocr-off is a live control, not a paper one), so
// there is no fallback. It is scoped to `.ocr-box` rather than the page, which leaves the rest of
// the document on the browser's own print economy. The -webkit- prefix is what Chromium still
// reads. Mirrors the extension's .ocr-plate (docs/PARITY.md).
const ocrCSS = `.ocr-fig{position:relative;display:block;width:100%;max-width:100%;margin:0 auto;container-type:inline-size;line-height:1.1}
.ocr-fig>img{display:block;width:100%;height:auto;margin:0;max-height:none}
.ocr-box{position:absolute;box-sizing:border-box;overflow:hidden;background:#fff;border-radius:0.35em;padding:0.08em 0.28em;color:#111;display:flex;align-items:center;justify-content:center;text-align:center;white-space:pre-wrap;overflow-wrap:anywhere;word-break:break-word;font-family:"Segoe UI",system-ui,Arial,sans-serif;-webkit-print-color-adjust:exact;print-color-adjust:exact}
/* The navbar's OCR toggle sets this on <html> to reveal the untouched artwork. display:none,
   not visibility/opacity: a hidden plate must not keep taking pointer events or be read out,
   and the page translator must not find text to swap in a layer the reader turned off. */
html.dht-ocr-off .ocr-box{display:none}`

// ocrScript fits each plate's text to its box after layout, and again whenever the page
// translator swaps the text for a longer string - the case a compile-time font size cannot handle,
// because the plate box and font are computed from the *source* geometry while the text poured in
// (reflowed, then translated) is a different length. It shrinks the cqw font down to a floor, and
// if the text still overflows there it lets the box grow so nothing is ever clipped.
//
// The fit runs in both directions. Growing exists because the compile-time size is deliberately
// conservative - the font is the median *ink* height times ocrFontFitFactor, and an ink box is
// shorter than the type that drew it - so a plate whose text is no longer than the source's ends up
// with the string floating in white space, which reads as an oversized patch rather than as the
// original lettering. Measured on a screenshot caption: source capitals 28 px against the plate's
// 24 px, with the slack showing as margin on all four sides.
//
// Growth stops one step before the content overflows *and* at 1.15x the base, and the second bound
// is the important one. A block's box is the union of its lines, so it includes the leading between
// them; filling that box is not the same as matching the source's type, and on a loosely leaded
// block "fill the box" would print the translation larger than the words it covers. 1.15 is a
// little over 1/ocrFontFitFactor, so the plate may reach the measured ink height of the source's
// own lines and no further. The step is 4% against the shrink's 8%, because overshooting here is
// visible while undershooting is not. A translated string is normally longer than its source, so on
// a translated page this branch does not fire at all.
//
// Injected once per overlaid page; a no-op degrade to the CSS (overflow:hidden) if the script does
// not run. Mirrors the extension's fitPlate + observer (see docs/PARITY.md and ocr-overlay.js).
const ocrScript = `(function(){
function fit(b){
  if(!b.dataset.ocrCqw){var m=/([0-9.]+)cqw/.exec(b.style.fontSize||"");b.dataset.ocrCqw=m?m[1]:"0";}
  var base=parseFloat(b.dataset.ocrCqw);
  b.style.height="";
  var target=parseFloat(getComputedStyle(b).minHeight)||0;
  if(target>0)b.style.height=target+"px";
  if(base>0){var s=base,floor=base*0.5,g=0;b.style.fontSize=s+"cqw";
    while(b.scrollHeight>b.clientHeight+1&&s>floor&&g<40){s-=Math.max(0.3,s*0.08);g++;b.style.fontSize=s+"cqw";}
    if(b.scrollHeight<=b.clientHeight+1){var cap=base*1.15,p=s,n=s,gg=0;
      while(n<cap&&gg<20){n=Math.min(cap,n+Math.max(0.3,n*0.04));b.style.fontSize=n+"cqw";gg++;
        if(b.scrollHeight>b.clientHeight+1){b.style.fontSize=p+"cqw";break;}p=n;}}}
  if(b.scrollHeight>b.clientHeight+1)b.style.height="auto";
}
function fitAll(){var l=document.querySelectorAll(".ocr-box");for(var i=0;i<l.length;i++)fit(l[i]);}
var t;function go(){clearTimeout(t);t=setTimeout(fitAll,0);}
if(document.readyState!=="loading")fitAll();else document.addEventListener("DOMContentLoaded",fitAll);
window.addEventListener("load",fitAll);window.addEventListener("resize",go);
function watch(){new MutationObserver(function(m){for(var i=0;i<m.length;i++){var n=m[i].target;
  while(n&&n!==document.body){if(n.classList&&n.classList.contains("ocr-box")){go();return;}n=n.parentNode;}}})
  .observe(document.body,{childList:true,characterData:true,subtree:true});}
if(document.body)watch();else document.addEventListener("DOMContentLoaded",watch);
})();`

// OverlayBook OCRs every local <img> across all of a book's content files and rewrites each
// into a positioned container with opaque, translatable text plates over the image. Returns a
// per-image breakdown (see OverlayResult) rather than a bare count, so the caller can report
// an image that failed apart from one that simply had no text. Best-effort throughout: a
// missing tesseract, an unreadable page, or a single failed image never aborts the book.
//
// The batching boundary is the whole book, not one file. Recognition - the expensive part,
// one Tesseract shell-out of about a second per image - runs across ocrWorkers() processes
// over *every* image at once, so the pool stays at full width even in -multipage mode, where
// each content file holds a single page with a single image and a per-file pool would degrade
// to serial (the defect this replaced). Recognition needs only the image's path, so it is
// decoupled from the DOM in three phases:
//
//  1. parse each file, collect the recognizable image paths (deduped) - docs are released;
//  2. recognize every unique image once, across one book-wide pool;
//  3. re-parse each file one at a time and wrap its images from the precomputed results.
//
// Only the small results map (text + geometry) is held across the book - never every parsed
// page or decoded image at once - so memory stays close to the old per-file cost. decodeImage
// (for plate colours) and the HTML rewrite both stay in phase 3, one file at a time, because
// the HTML tree is not safe for concurrent mutation.
//
// onProgress, when non-nil, is called as images finish with the number done and the total to
// do across the book; the counter is per-image (not per-file), so single-page mode - where the
// whole book is one file - still shows real motion instead of sitting at 0/1.
func OverlayBook(bin string, htmlPaths []string, lang, dataDir string, langFixed bool, onProgress func(done, total int)) OverlayResult {
	var stats OverlayResult
	stats.Lang = lang

	// Phase 1: every recognizable image across the book, deduped by absolute path.
	order := collectBookImages(htmlPaths)
	if len(order) == 0 {
		return stats
	}

	// Phase 1b: let the document's own script correct a language nobody chose (see script.go).
	// Detection runs once, on the book's first image, not once per page: a book is one document in
	// one language, and the pass costs a whole extra Tesseract process - half a second here against
	// four minutes on a 480-page comic.
	script, conf, detected := "", 0.0, false
	if !langFixed {
		script, conf, detected = DetectScript(bin, order[0], dataDir)
	}
	use, note, stop := resolveScript(lang, langFixed, script, conf, detected, installedForScript(dataDir, script))
	stats.Lang, stats.ScriptNote = use, note
	if stop {
		return stats
	}
	lang = use

	// Phase 2: recognize every image once, across one pool spanning the whole book.
	results := recognizePaths(bin, lang, dataDir, order, onProgress)

	// Phase 3: re-parse each file (one at a time) and wrap its images from the results.
	for _, htmlPath := range htmlPaths {
		doc, baseDir, err := parseHTMLFile(htmlPath)
		if err != nil {
			continue
		}
		fileStats, changed := applyOverlays(doc, baseDir, results)
		stats.Add(fileStats)
		if !changed {
			continue // nothing added, so the file on disk is already what we would write
		}
		ensureStyle(doc)
		ensureScript(doc)
		if err := renderHTMLFile(htmlPath, doc); err != nil {
			// A write failure on one page must not abort the book; it just stays un-overlaid.
			stats.Failed = append(stats.Failed, OverlayFailure{File: htmlPath, Err: err})
		}
	}
	return stats
}

// OverlayFile OCRs a single content file. It is OverlayBook over a one-file book, kept for the
// single-file callers and tests; the pool it runs is that file's images only, so prefer
// OverlayBook when a whole book's worth of pages is available.
func OverlayFile(bin, htmlPath, lang, dataDir string, onProgress func(done, total int)) (OverlayResult, error) {
	return OverlayBook(bin, []string{htmlPath}, lang, dataDir, true, onProgress), nil
}

// parseHTMLFile opens and parses one content file, returning its DOM and the directory its
// relative image srcs resolve against.
func parseHTMLFile(htmlPath string) (*gohtml.Node, string, error) {
	f, err := os.Open(htmlPath)
	if err != nil {
		return nil, "", err
	}
	doc, perr := gohtml.Parse(f)
	f.Close()
	if perr != nil {
		return nil, "", perr
	}
	return doc, filepath.Dir(htmlPath), nil
}

// renderHTMLFile writes the (possibly rewritten) DOM back to disk.
func renderHTMLFile(htmlPath string, doc *gohtml.Node) error {
	out, err := os.Create(htmlPath)
	if err != nil {
		return err
	}
	if err := gohtml.Render(out, doc); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// applyOverlays wraps every recognizable image in doc from the precomputed recognition
// results, returning the per-file breakdown and whether any overlay was added (i.e. whether
// the file needs rewriting). An image with no result (it was not recognized) is skipped.
func applyOverlays(doc *gohtml.Node, baseDir string, results map[string]recognition) (OverlayResult, bool) {
	var stats OverlayResult
	for _, job := range collectOverlayJobs(collectImgs(doc), baseDir) {
		r, found := results[job.file]
		switch {
		case !found:
			continue
		case r.err != nil:
			stats.Failed = append(stats.Failed, OverlayFailure{File: job.file, Err: r.err})
		case !r.ok:
			stats.NoText++
		default:
			// The boxes already arrived in display space (stageForOCR turned the picture before
			// recognition), so the colours have to be sampled from the same view - otherwise a
			// rotated photo takes each plate's paper and ink from somewhere else in the image.
			srcImg := orientImage(decodeImage(job.file), exifOrientation(job.file))
			wrapImage(job.node, r.res, srcImg)
			// Off unless DOCHT_OCR_DIAG is set; see diag.go. Never touches the DOM.
			recordDiagnostics(job.file, r.res, srcImg)
			stats.Overlaid++
		}
	}
	return stats, stats.Overlaid > 0
}

// overlayJob is one image to recognize and then wrap: the <img> node and its file on disk.
type overlayJob struct {
	node *gohtml.Node
	file string
}

// recognition is the outcome of OCRing one image path. ok and err are not opposites, and
// keeping them apart is the point: ok=false with err=nil means the image was read fine and
// simply holds no text (an art panel, a rule, a photo) - the normal case. A non-nil err means
// recognition itself failed, which never is.
type recognition struct {
	res Result
	ok  bool
	err error
}

// OverlayResult reports what became of each image, so a caller can tell the ordinary case
// from a broken one. A bare success count cannot: a graphic novel whose art pages carry no
// dialogue and a book whose OCR is silently failing both read as "N of M overlaid", and
// telling them apart then costs an investigation per book.
type OverlayResult struct {
	Overlaid int              // images that got text plates
	NoText   int              // read fine, no text inside - expected, not a problem
	Failed   []OverlayFailure // recognition errored - always worth naming
	// Lang is the language the book was actually recognized with, which is not always the one the
	// caller asked for: an unchosen default can be corrected by the document's own script
	// (script.go). ScriptNote is the sentence explaining that, empty when nothing happened.
	Lang       string
	ScriptNote string
}

// OverlayFailure is one image that could not be recognized, and why.
type OverlayFailure struct {
	File string
	Err  error
}

// Add folds another file's result into this one.
func (r *OverlayResult) Add(o OverlayResult) {
	r.Overlaid += o.Overlaid
	r.NoText += o.NoText
	r.Failed = append(r.Failed, o.Failed...)
}

// collectBookImages returns every recognizable image across the book's content files, deduped
// by absolute path and in first-seen order (stable recognition + progress ordering). This is
// the batching boundary: the pool in phase 2 runs over this whole list, not one file's images.
// An unreadable page contributes nothing rather than aborting the run; an image referenced by
// more than one file is recognized once.
func collectBookImages(htmlPaths []string) []string {
	var order []string
	seen := make(map[string]bool)
	for _, htmlPath := range htmlPaths {
		doc, baseDir, err := parseHTMLFile(htmlPath)
		if err != nil {
			continue
		}
		for _, job := range collectOverlayJobs(collectImgs(doc), baseDir) {
			if !seen[job.file] {
				seen[job.file] = true
				order = append(order, job.file)
			}
		}
	}
	return order
}

// collectOverlayJobs keeps the images that are actually recognizable: a local src that
// resolves to a file on disk. Filtering up front means the progress total is the real
// amount of work, not a count padded with images that get skipped instantly.
func collectOverlayJobs(imgs []*gohtml.Node, baseDir string) []overlayJob {
	jobs := make([]overlayJob, 0, len(imgs))
	for _, img := range imgs {
		src := attrVal(img, "src")
		if src == "" || isExternal(src) {
			continue
		}
		file := filepath.Join(baseDir, filepath.FromSlash(src))
		if _, err := os.Stat(file); err != nil {
			continue
		}
		jobs = append(jobs, overlayJob{node: img, file: file})
	}
	return jobs
}

// ocrWorkers is how many Tesseract processes to keep in flight. Each one is essentially
// single-threaded, so the pool is what uses the machine. NumCPU-2 leaves the desktop (and
// the GUI streaming this log) a core to breathe rather than stalling the whole machine to
// finish a background conversion slightly sooner; the ceiling stops a very wide server
// from launching a process per core, where the per-process startup and the disk reads
// stop paying for themselves.
func ocrWorkers() int {
	const maxWorkers = 16
	n := runtime.NumCPU() - 2
	if n > maxWorkers {
		n = maxWorkers
	}
	if n < 1 {
		n = 1
	}
	return n
}

// classifyRecognition turns one Recognize outcome into the two things the caller must be able
// to tell apart: whether there are plates to draw, and - when there are not - whether that is
// a problem. An image with no text in it is ordinary and must not be reported as a failure;
// anything that stopped recognition from happening must not be reported as an absence of text.
// Pure, so the distinction is tested without a tesseract on PATH.
func classifyRecognition(res Result, err error) (ok bool, reason error) {
	switch {
	case err != nil:
		return false, err
	case res.Width <= 0 || res.Height <= 0:
		// Tesseract returned without an error but reported no page geometry, so there is
		// nothing to position plates against. That is a failure, not an empty page.
		return false, fmt.Errorf("recognized no page dimensions")
	case len(res.Blocks) == 0:
		// Read fine, holds no text: the art panel, the photo, the rule. Not a failure.
		return false, nil
	default:
		return true, nil
	}
}

// recognizePaths OCRs every image path across ocrWorkers() Tesseract processes and returns a
// map from path to its outcome. The pool spans the whole slice it is given, so a book's worth
// of pages is recognized at full width rather than one file's images at a time. One process
// pins about one core, so this is what actually uses a multi-core machine; a scanned book that
// recognized serially left ~90% of the cores idle while the reader waited. Completions are
// reported to onProgress in finishing order (not path order).
func recognizePaths(bin, lang, dataDir string, paths []string, onProgress func(done, total int)) map[string]recognition {
	if onProgress != nil {
		onProgress(0, len(paths))
	}
	out := make([]recognition, len(paths))
	queue := make(chan int)
	var wg sync.WaitGroup
	var mu sync.Mutex
	done := 0

	for w := 0; w < ocrWorkers(); w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range queue {
				// Recognize is self-contained (its own temp file, its own process), so it
				// is safe to run concurrently; each worker writes only its own slot.
				res, err := Recognize(bin, paths[i], lang, dataDir)
				ok, reason := classifyRecognition(res, err)
				r := recognition{ok: ok, err: reason}
				switch {
				case ok:
					r.res = res
				case reason == nil:
					// Read fine, no plates. The result is kept anyway so the diagnostics can say
					// *why*: a page whose every recognized line was thrown away by the confidence
					// floor is the case this exists for, and without the record it is
					// indistinguishable from a page that genuinely holds no text. Nothing else
					// consumes it - applyOverlays still takes the !ok branch and the DOM is
					// untouched.
					r.res = Result{Width: res.Width, Height: res.Height, Dropped: res.Dropped}
				}
				out[i] = r
				mu.Lock()
				done++
				if onProgress != nil {
					onProgress(done, len(paths))
				}
				mu.Unlock()
			}
		}()
	}
	for i := range paths {
		queue <- i
	}
	close(queue)
	wg.Wait()

	results := make(map[string]recognition, len(paths))
	for i, p := range paths {
		results[p] = out[i]
	}
	return results
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
			if paper, ink, ok := blockColors(srcImg, b); ok {
				// Paper and ink both land on the plate box: the box is what covers the source
				// region, so it is what has to be opaque (see ocrCSS for the measurement that
				// decided this against carrying the paper on the string).
				style += ";background:" + paper + ";color:" + ink
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
	var ir, ig, ib []int
	for i := range fr {
		if absInt(fr[i]-bgR)+absInt(fg[i]-bgG)+absInt(fb[i]-bgB) > 90 {
			ir = append(ir, fr[i])
			ig = append(ig, fg[i])
			ib = append(ib, fb[i])
		}
	}
	c := len(ir)
	minInk := len(fr) * 15 / 1000
	if minInk < 6 {
		minInk = 6
	}
	var inkR, inkG, inkB int
	if c >= minInk {
		// Median, not mean, and for the same reason the background is a median: a glyph's edge is
		// a ramp of antialiased pixels running from the ink to the paper, and the deviation test
		// admits most of that ramp. Averaging it drags the answer toward the paper - measured on a
		// screenshot caption whose lettering is rgb(17,17,17) on rgb(253,253,253), the mean
		// returned rgb(61,61,61) while the median of the same pixels returns rgb(7,7,7). The
		// difference is visible: the plate's text reads as grey next to black source lettering.
		inkR, inkG, inkB = medianOf(ir), medianOf(ig), medianOf(ib)
		// Which of the two colours is the paper is decided by what surrounds the block, not by
		// which of them covers more of it. The median above assumes the text is the minority of
		// its own box - true of body text in a balloon, false of heavy display capitals, whose
		// strokes cover more of a tight box than the paper between them does. Measured on the
		// reported poster (DEV/research/ocr_display_lettering_2026-08-12.md): the plate over
		// МОЖЕМ came out as cream lettering on a near-black ground, the exact inverse of the
		// poster. What is outside a text box is paper in both cases - a balloon's interior
		// around its lines, a poster's ground around its word - so the ring just outside the
		// block is what says which way round the pair goes.
		if ringNearerInk(img, x0, y0, x1, y1, lh, bgR, bgG, bgB, inkR, inkG, inkB) {
			bgR, bgG, bgB, inkR, inkG, inkB = inkR, inkG, inkB, bgR, bgG, bgB
		}
	} else {
		inkR, inkG, inkB = fallbackInk(bgR, bgG, bgB)
	}
	if absInt(luma(inkR, inkG, inkB)-luma(bgR, bgG, bgB)) < 55 {
		inkR, inkG, inkB = fallbackInk(bgR, bgG, bgB)
	}
	return fmt.Sprintf("rgb(%d,%d,%d)", bgR, bgG, bgB),
		fmt.Sprintf("rgb(%d,%d,%d)", inkR, inkG, inkB), true
}

// ringNearerInk reports whether the band just outside the block sits nearer the ink colour than
// the paper colour - which means the two were assigned the wrong way round.
//
// The band is a third of a line on each side, so it is the text's own surroundings rather than the
// next thing on the page, and it is read outside the box rather than inside it: a box drawn tightly
// around display capitals has their strokes on its own edges, so an inside ring would answer with
// the ink it is supposed to be judging. `lh` is the raw line height, not the 1.3-line strip the ink
// is sampled in. A band that falls entirely off the image (a block against the edge) leaves too few
// samples to decide and the caller keeps what it had.
func ringNearerInk(img image.Image, x0, y0, x1, y1, lh, bgR, bgG, bgB, inkR, inkG, inkB int) bool {
	pad := lh / 3
	if pad < 2 {
		pad = 2
	}
	bnds := img.Bounds()
	ox0 := clampInt(x0-pad, bnds.Min.X, bnds.Max.X)
	oy0 := clampInt(y0-pad, bnds.Min.Y, bnds.Max.Y)
	ox1 := clampInt(x1+pad, bnds.Min.X, bnds.Max.X)
	oy1 := clampInt(y1+pad, bnds.Min.Y, bnds.Max.Y)

	nearInk, nearBg := 0, 0
	count := func(sx0, sy0, sx1, sy1 int) {
		if sx1-sx0 < 1 || sy1-sy0 < 1 {
			return
		}
		rs, gs, bs := samplePixels(img, sx0, sy0, sx1, sy1)
		for i := range rs {
			dBg := absInt(rs[i]-bgR) + absInt(gs[i]-bgG) + absInt(bs[i]-bgB)
			dInk := absInt(rs[i]-inkR) + absInt(gs[i]-inkG) + absInt(bs[i]-inkB)
			if dInk < dBg {
				nearInk++
			} else if dBg < dInk {
				nearBg++
			}
		}
	}
	count(ox0, oy0, ox1, y0) // above
	count(ox0, y1, ox1, oy1) // below
	count(ox0, y0, x0, y1)   // left
	count(x1, y0, ox1, y1)   // right
	if nearInk+nearBg < ringMinSamples {
		return false
	}
	return nearInk > nearBg
}

// ringMinSamples is how many pixels the surrounding band must contribute before it is allowed to
// swap the pair. Small enough that a block against one edge of the image still decides on the
// three bands it has, large enough that a sliver is not a vote.
const ringMinSamples = 40

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

// ensureScript appends the plate re-fit script (ocrScript) to <body> - or <html>/document if there
// is no body - once per document. script/style are raw-text elements in html.Render, so the JS is
// written verbatim (not entity-escaped), the same as ensureStyle's CSS.
func ensureScript(doc *gohtml.Node) {
	var body, htmlEl *gohtml.Node
	var walk func(*gohtml.Node)
	walk = func(n *gohtml.Node) {
		if n.Type == gohtml.ElementNode {
			switch n.DataAtom {
			case atom.Body:
				if body == nil {
					body = n
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

	target := body
	if target == nil {
		target = htmlEl
	}
	if target == nil {
		target = doc
	}
	script := &gohtml.Node{Type: gohtml.ElementNode, Data: "script", DataAtom: atom.Script}
	script.AppendChild(&gohtml.Node{Type: gohtml.TextNode, Data: ocrScript})
	target.AppendChild(script)
}
