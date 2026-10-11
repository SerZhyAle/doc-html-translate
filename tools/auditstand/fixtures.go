// fixtures.go holds the synthetic corpus of the audit (FIXTURES.md of the tactical folder).
// Everything is generated locally and deterministically: the same inputs produce byte-identical
// fixtures, so a run record's fixture SHA-256 values are comparable across days. Nothing real,
// nothing downloaded, nothing committed.

package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

// fixtureTime is the fixed modification time every generated entry carries; determinism needs
// the clock out of the way.
var fixtureTime = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

// fixture is one catalog row of FIXTURES.md.
type fixture struct {
	id       string
	file     string // name inside the fixture root
	big      bool   // built only by -Fixtures -Big
	generate func(dir string) error
}

// fixtureCatalog is the contract between this file, the self-test and the scene steps.
var fixtureCatalog = []fixture{
	{"epub-small", "epub-small.epub", false, genEpubSmall},
	{"epub-nested", "epub-nested.epub", false, genEpubNested},
	{"pdf-text", "pdf-text.pdf", false, genPdfText},
	{"pdf-scan", "pdf-scan.pdf", false, genPdfScan},
	{"img-photo", "img-photo.png", false, genImgPhoto},
	{"comic-small-cbz", "comic-small.cbz", false, genComicSmallCbz},
	{"comic-cbt", "comic.cbt", false, genComicCbt},
	{"comic-corrupt", "comic-corrupt.cbz", false, genComicCorrupt},
	{"mobi-probe", "mobi-probe.mobi", false, genMobiProbe},
	{"comic-probe", "comic-probe.cbr", false, genComicProbeCbr},
	{"rtf-nested", "rtf-nested.rtf", false, genRtfNested},
	{"tiff-multiframe", "tiff-multiframe.tif", false, genMultiFrameTIFF},
	{"tiff-over-cap", "tiff-over-cap.tif", false, genOverCapTIFF},
	{"comic-500", "comic-500.cbz", true, genComic500},
	{"comic-big-2gb", "comic-big-2gb.cbz", true, genComicBig2gb},
	// The fd-sec container must be created by FileDO itself (FIXTURES.md).
}

// fixturePath is where one fixture lives inside the fixture root.
func fixturePath(root string, f fixture) string { return filepath.Join(root, f.file) }

// generateFixtures builds every requested fixture deterministically and writes manifest.json
// (id -> sha256) so the self-test can tell "missing" from "present but corrupt".
func generateFixtures(root string, big bool) ([]string, error) {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, err
	}
	var built []string
	manifest := map[string]string{}
	for _, f := range fixtureCatalog {
		if f.big && !big {
			continue
		}
		p := fixturePath(root, f)
		if err := f.generate(root); err != nil {
			return built, fmt.Errorf("fixture %s: %w", f.id, err)
		}
		sum, err := sha256File(p)
		if err != nil {
			return built, fmt.Errorf("fixture %s: %w", f.id, err)
		}
		manifest[f.id] = sum
		built = append(built, f.id)
	}
	mb, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return built, err
	}
	if err := os.WriteFile(filepath.Join(root, "manifest.json"), mb, 0o644); err != nil {
		return built, err
	}
	return built, nil
}

// checkFixtures verifies presence and integrity. Returns: missing ids, corrupt ids.
func checkFixtures(root string) (missing, corrupt []string, err error) {
	manifest := map[string]string{}
	if mb, rerr := os.ReadFile(filepath.Join(root, "manifest.json")); rerr == nil {
		if jerr := json.Unmarshal(mb, &manifest); jerr != nil {
			return nil, nil, fmt.Errorf("fixture manifest: %w", jerr)
		}
	} else if !os.IsNotExist(rerr) {
		return nil, nil, rerr
	}
	for _, f := range fixtureCatalog {
		if f.big {
			continue // big fixtures are advisory until their scene's phase starts
		}
		p := fixturePath(root, f)
		if _, serr := os.Stat(p); serr != nil {
			missing = append(missing, f.id)
			continue
		}
		sum, err := sha256File(p)
		if err != nil {
			corrupt = append(corrupt, f.id)
			continue
		}
		if want, ok := manifest[f.id]; !ok || want != sum {
			corrupt = append(corrupt, f.id)
		}
	}
	return missing, corrupt, nil
}

// --- EPUB ---------------------------------------------------------------------------

// epubChapter is one generated chapter.
type epubChapter struct {
	title string
	paras []string
}

// writeEpub packs a minimal EPUB2: mimetype, container, OPF, NCX (nested for epub-nested) and
// one XHTML file per chapter. Deterministic by construction.
func writeEpub(path string, title string, chapters []epubChapter, nested bool) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := zip.NewWriter(f)
	defer w.Close()

	must := func(name string, content []byte, store bool) error {
		h := &zip.FileHeader{Name: name, Method: zip.Deflate, Modified: fixtureTime}
		if store {
			h.Method = zip.Store
		}
		fw, err := w.CreateHeader(h)
		if err != nil {
			return err
		}
		_, err = fw.Write(content)
		return err
	}
	if err := must("mimetype", []byte("application/epub+zip"), true); err != nil {
		return err
	}
	if err := must("META-INF/container.xml", []byte(`<?xml version="1.0"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles><rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/></rootfiles>
</container>`), false); err != nil {
		return err
	}

	var manifest, spine, ncx strings.Builder
	for i, ch := range chapters {
		id := fmt.Sprintf("ch%03d", i+1)
		href := id + ".xhtml"
		manifest.WriteString(fmt.Sprintf(`<item id="%s" href="%s" media-type="application/xhtml+xml"/>`+"\n", id, href))
		spine.WriteString(fmt.Sprintf(`<itemref idref="%s"/>`+"\n", id))
		var body strings.Builder
		body.WriteString("<h1>" + ch.title + "</h1>\n")
		for _, p := range ch.paras {
			body.WriteString("<p>" + p + "</p>\n")
		}
		if err := must("OEBPS/"+href, []byte(xhtmlDoc(ch.title, body.String())), false); err != nil {
			return err
		}
	}
	// NCX navMap: nested when asked, so the multi-level TOC of S04 has something real to read.
	if nested {
		ncx.WriteString(nestedNavMap(chapters))
	} else {
		for i, ch := range chapters {
			ncx.WriteString(fmt.Sprintf(`<navPoint id="np%03d" playOrder="%d"><navLabel><text>%s</text></navLabel><content src="ch%03d.xhtml"/></navPoint>`+"\n",
				i+1, i+1, ch.title, i+1))
		}
	}
	opf := fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="2.0" unique-identifier="bookid">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>%s</dc:title>
    <dc:language>en</dc:language>
    <dc:identifier id="bookid">urn:uuid:11111111-2222-3333-4444-555555555555</dc:identifier>
  </metadata>
  <manifest>
<item id="ncx" href="toc.ncx" media-type="application/x-dtbncx+xml"/>
%s  </manifest>
  <spine toc="ncx">
%s  </spine>
</package>`, title, manifest.String(), spine.String())
	if err := must("OEBPS/content.opf", []byte(opf), false); err != nil {
		return err
	}
	ncxDoc := fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?>
<ncx xmlns="http://www.daisy.org/z3986/2005/ncx/" version="2005-1">
  <head><meta name="dtb:uid" content="urn:uuid:11111111-2222-3333-4444-555555555555"/></head>
  <docTitle><text>%s</text></docTitle>
  <navMap>
%s  </navMap>
</ncx>`, title, ncx.String())
	return must("OEBPS/toc.ncx", []byte(ncxDoc), false)
}

// nestedNavMap groups chapters three levels deep: 10 sections of 3 chapters each.
func nestedNavMap(chapters []epubChapter) string {
	var b strings.Builder
	play := 1
	for s := 0; s*3 < len(chapters); s++ {
		b.WriteString(fmt.Sprintf(`<navPoint id="sec%02d" playOrder="%d"><navLabel><text>Part %d</text></navLabel><content src="ch%03d.xhtml"/>`+"\n",
			s+1, play, s+1, s*3+1))
		play++
		for c := 0; c < 3 && s*3+c < len(chapters); c++ {
			i := s*3 + c
			b.WriteString(fmt.Sprintf(`<navPoint id="np%03d" playOrder="%d"><navLabel><text>%s</text></navLabel><content src="ch%03d.xhtml"/></navPoint>`+"\n",
				i+1, play, chapters[i].title, i+1))
			play++
		}
		b.WriteString("</navPoint>\n")
	}
	return b.String()
}

func xhtmlDoc(title, body string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?>
<!DOCTYPE html>
<html xmlns="http://www.w3.org/1999/xhtml"><head><title>%s</title></head>
<body>
%s</body></html>`, title, body)
}

func genEpubSmall(dir string) error {
	var chapters []epubChapter
	for i := 1; i <= 5; i++ {
		chapters = append(chapters, epubChapter{
			title: fmt.Sprintf("Chapter %d", i),
			paras: paragraphs(i, 4),
		})
	}
	return writeEpub(filepath.Join(dir, "epub-small.epub"), "Audit Small Book", chapters, false)
}

func genEpubNested(dir string) error {
	var chapters []epubChapter
	for i := 1; i <= 30; i++ {
		chapters = append(chapters, epubChapter{
			title: fmt.Sprintf("Part %d Chapter %d", (i-1)/10+1, i),
			paras: paragraphs(i, 6),
		})
	}
	return writeEpub(filepath.Join(dir, "epub-nested.epub"), "Audit Nested Book", chapters, true)
}

// paragraphs returns n deterministic sentences, numbered by seed so a diff points at its page.
func paragraphs(seed, n int) []string {
	var out []string
	for i := 0; i < n; i++ {
		out = append(out, fmt.Sprintf("Sentence %d.%d of the synthetic audit corpus. The reader is a machine; the words only need to exist, wrap and measure.", seed, i+1))
	}
	return out
}

// --- PDF ----------------------------------------------------------------------------

func genPdfText(dir string) error {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetCreationDate(fixtureTime)
	pdf.SetModificationDate(fixtureTime)
	pdf.SetCatalogSort(true)
	pdf.SetMargins(20, 20, 20)
	pdf.SetAutoPageBreak(true, 20)
	for p := 1; p <= 8; p++ {
		pdf.AddPage()
		pdf.SetFont("Helvetica", "", 16)
		pdf.Cell(0, 10, fmt.Sprintf("Synthetic text page %d", p))
		pdf.Ln(12)
		pdf.SetFont("Helvetica", "", 11)
		for i := 0; i < 18; i++ {
			pdf.MultiCell(0, 6, fmt.Sprintf("Line %d.%d: the synthetic audit corpus fills a text-layer PDF deterministically, so every run measures the same extraction work.", p, i+1), "", "L", false)
		}
	}
	return pdf.OutputFileAndClose(filepath.Join(dir, "pdf-text.pdf"))
}

func genPdfScan(dir string) error {
	img := scanPageImage(1)
	tmp := filepath.Join(dir, ".scan-page-1.png")
	if err := writePNG(tmp, img); err != nil {
		return err
	}
	defer os.Remove(tmp)
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetCreationDate(fixtureTime)
	pdf.SetModificationDate(fixtureTime)
	pdf.SetCatalogSort(true)
	pdf.SetAutoPageBreak(false, 0)
	pdf.AddPage()
	opts := fpdf.ImageOptions{ImageType: "png"}
	pdf.ImageOptions(tmp, 0, 0, 210, 297, false, opts, 0, "")
	if pdf.Err() {
		return pdf.Error()
	}

	return pdf.OutputFileAndClose(filepath.Join(dir, "pdf-scan.pdf"))
}

// scanPageImage renders one grey page image with deterministic lettering - the OCR scenes'
// synthetic scan (basicfont draws the only text, so the annotation is known by construction).
func scanPageImage(seed int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 620, 840))
	draw.Draw(img, img.Bounds(), &image.Uniform{color.RGBA{230, 230, 226, 255}}, image.Point{}, draw.Src)
	drawText(img, 40, 60, fmt.Sprintf("SYNTHETIC SCAN %d", seed))
	for i := 0; i < 16; i++ {
		drawText(img, 40, 120+i*32, fmt.Sprintf("Line %d.%d of the synthetic scan corpus.", seed, i+1))
	}
	return img
}

func genImgPhoto(dir string) error {
	img := image.NewRGBA(image.Rect(0, 0, 640, 800))
	draw.Draw(img, img.Bounds(), &image.Uniform{color.RGBA{32, 48, 72, 255}}, image.Point{}, draw.Src)
	for i := 0; i < 24; i++ {
		c := color.RGBA{uint8(40 + i*8), uint8(90 + i*3), uint8(160 - i*4), 255}
		draw.Draw(img, image.Rect(30+i*24, 500, 52+i*24, 560), &image.Uniform{c}, image.Point{}, draw.Src)
	}
	drawText(img, 40, 80, "SYNTHETIC PHOTO INPUT")
	drawText(img, 40, 120, "Standalone image, forced OCR scene.")
	return writePNG(filepath.Join(dir, "img-photo.png"), img)
}

// drawText writes one line with the deterministic basicfont face.
func drawText(img *image.RGBA, x, y int, s string) {
	d := &font.Drawer{
		Dst:  img,
		Src:  image.Black,
		Face: basicfont.Face7x13,
		Dot:  fixed.P(x, y),
	}
	d.DrawString(s)
}

func writePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := png.Encoder{CompressionLevel: png.DefaultCompression}
	return enc.Encode(f, img)
}

// --- comics ---------------------------------------------------------------------------

func comicPage(seed, w, h int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	shade := uint8(200 - (seed*7)%120)
	draw.Draw(img, img.Bounds(), &image.Uniform{color.RGBA{shade, shade, shade, 255}}, image.Point{}, draw.Src)
	drawText(img, 30, 60, fmt.Sprintf("PAGE %04d", seed))
	return img
}

func writeComicZip(path string, pages int) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := zip.NewWriter(f)
	defer w.Close()
	// Page images are rendered once per unique seed and reused: a 500-page archive is 500
	// entries, and rendering 500 PNGs must not take minutes.
	cache := map[int][]byte{}
	for i := 1; i <= pages; i++ {
		seed := i
		data, ok := cache[seed]
		if !ok {
			var buf bytes.Buffer
			if err := png.Encode(&buf, comicPage(seed, 480, 640)); err != nil {
				return err
			}
			data = buf.Bytes()
			cache[seed] = data
		}
		h := &zip.FileHeader{Name: fmt.Sprintf("page%04d.png", i), Method: zip.Deflate, Modified: fixtureTime}
		fw, err := w.CreateHeader(h)
		if err != nil {
			return err
		}
		if _, err := fw.Write(data); err != nil {
			return err
		}
	}
	return nil
}

func genComicSmallCbz(dir string) error {
	return writeComicZip(filepath.Join(dir, "comic-small.cbz"), 10)
}

func genComic500(dir string) error { return writeComicZip(filepath.Join(dir, "comic-500.cbz"), 500) }

// genComicBig2gb writes a stored (uncompressed) zip whose single page entry is ~2 GiB of zeros -
// ticket 12's owed hands-on input. The zeros compress to nothing honest, so the entry is stored,
// and the file on disk really is 2 GiB.
func genComicBig2gb(dir string) error {
	f, err := os.Create(filepath.Join(dir, "comic-big-2gb.cbz"))
	if err != nil {
		return err
	}
	defer f.Close()
	w := zip.NewWriter(f)
	h := &zip.FileHeader{Name: "page0001.png", Method: zip.Store, Modified: fixtureTime}
	// A PNG header so the pipeline sees an image before the budget refusal fires.
	fw, err := w.CreateHeader(h)
	if err != nil {
		return err
	}
	var pngHdr bytes.Buffer
	if err := png.Encode(&pngHdr, comicPage(1, 8, 8)); err != nil {
		return err
	}
	if _, err := fw.Write(pngHdr.Bytes()); err != nil {
		return err
	}
	chunk := make([]byte, 1<<20)
	const targetBytes int64 = 2 << 30
	written := int64(len(pngHdr.Bytes()))
	for written < targetBytes {
		n := len(chunk)
		if rest := targetBytes - written; rest < int64(n) {
			n = int(rest)
		}
		if _, err := fw.Write(chunk[:n]); err != nil {
			return err
		}
		written += int64(n)
	}
	return w.Close()
}

func genComicCbt(dir string) error {
	f, err := os.Create(filepath.Join(dir, "comic.cbt"))
	if err != nil {
		return err
	}
	defer f.Close()
	w := tar.NewWriter(f)
	defer w.Close()
	for i := 1; i <= 10; i++ {
		var buf bytes.Buffer
		if err := png.Encode(&buf, comicPage(i, 480, 640)); err != nil {
			return err
		}
		data := buf.Bytes()
		if err := w.WriteHeader(&tar.Header{Name: fmt.Sprintf("page%04d.png", i), Mode: 0o644, Size: int64(len(data)), ModTime: fixtureTime}); err != nil {
			return err
		}
		if _, err := w.Write(data); err != nil {
			return err
		}
	}
	return nil
}

// genComicCorrupt writes a file that starts like a zip and stops before the central directory -
// the corrupt-archive member of S16.
func genComicCorrupt(dir string) error {
	f, err := os.Create(filepath.Join(dir, "comic-corrupt.cbz"))
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write([]byte("PK\x03\x04" + "truncated synthetic archive - the rest is missing"))
	return err
}

// genMobiProbe writes a PalmDB-shaped header with no real content behind it: enough for the
// missing-Calibre member to meet the helper boundary before any real parsing matters.
func genMobiProbe(dir string) error {
	buf := make([]byte, 4096)
	copy(buf, "BOOKMOBI")
	for i := 32; i < len(buf); i++ {
		buf[i] = byte(i % 251)
	}
	return os.WriteFile(filepath.Join(dir, "mobi-probe.mobi"), buf, 0o644)
}

// genComicProbeCbr writes bytes with only the RAR marker: the missing-7-Zip member's input.
func genComicProbeCbr(dir string) error {
	buf := make([]byte, 2048)
	copy(buf, "Rar!\x1a\x07\x01\x00")
	for i := 8; i < len(buf); i++ {
		buf[i] = byte(i % 241)
	}
	return os.WriteFile(filepath.Join(dir, "comic-probe.cbr"), buf, 0o644)
}

// --- RTF ------------------------------------------------------------------------------

// genRtfNested writes an RTF whose groups nest 500 deep - ticket 81's subject at a depth the
// fixed build must survive, not the unbounded input that broke it.
func genRtfNested(dir string) error {
	var b strings.Builder
	b.WriteString(`{\rtf1\ansi\deff0 {\fonttbl{\f0 Helvetica;}}` + "\n")
	const depth = 500
	for i := 0; i < depth; i++ {
		b.WriteString("{\\b group ")
		b.WriteString(fmt.Sprintf("%d ", i))
	}
	b.WriteString("deepest text")
	for i := 0; i < depth; i++ {
		b.WriteString("}")
	}
	b.WriteString("\n\\par Depth test complete.\n}\n")
	return os.WriteFile(filepath.Join(dir, "rtf-nested.rtf"), []byte(b.String()), 0o644)
}
