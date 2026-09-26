package pdf

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/go-pdf/fpdf"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// writeFixturePNG writes a small opaque PNG. Each call gets its own colour so pdfcpu keeps
// the images as distinct objects rather than one shared resource.
func writeFixturePNG(t *testing.T, path string, shade uint8) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 40, 60))
	for y := 0; y < 60; y++ {
		for x := 0; x < 40; x++ {
			img.Set(x, y, color.RGBA{R: shade, G: 80, B: 160, A: 255})
		}
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
}

// buildFixturePDF writes a PDF of pages pages. Pages in textPages carry a line of text,
// pages in imagePages carry a picture; any other page is blank.
func buildFixturePDF(t *testing.T, pdfPath string, pages int, textPages, imagePages map[int]bool) {
	t.Helper()
	dir := t.TempDir()
	doc := fpdf.New("P", "mm", "A4", "")
	for p := 1; p <= pages; p++ {
		doc.AddPage()
		doc.SetFont("Helvetica", "", 12.0+float64(p%2)*0.1)
		if textPages[p] {
			doc.Text(20, 30, fmt.Sprintf("Paragraph text on page %d of the document.", p))
		}
		if imagePages[p] {
			imgPath := filepath.Join(dir, fmt.Sprintf("img%d.png", p))
			writeFixturePNG(t, imgPath, uint8(20*p))
			doc.ImageOptions(imgPath, 20, 50, 80, 0, false, fpdf.ImageOptions{ImageType: "PNG"}, 0, "")
		}
	}
	if err := doc.OutputFileAndClose(pdfPath); err != nil {
		t.Fatal(err)
	}
}

func pageSet(pages ...int) map[int]bool {
	m := make(map[int]bool, len(pages))
	for _, p := range pages {
		m[p] = true
	}
	return m
}

func allPages(n int) map[int]bool {
	m := make(map[int]bool, n)
	for p := 1; p <= n; p++ {
		m[p] = true
	}
	return m
}

func sortedKeys(m map[int][]string) []int {
	keys := make([]int, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	return keys
}

// A PDF whose name carries a number used to put every image on that page: the page was read
// back out of the image file name, and the first number in it was the one from "Volume_3".
func TestExtractImages_PageComesFromThePageNotTheFileName(t *testing.T) {
	tmp := t.TempDir()
	pdfPath := filepath.Join(tmp, "Volume_3.pdf")
	buildFixturePDF(t, pdfPath, 10, allPages(10), pageSet(1, 5, 9))

	out := filepath.Join(tmp, "out")
	got := extractImages(pdfPath, out)

	if got.pageCount != 10 {
		t.Errorf("pageCount = %d, want 10", got.pageCount)
	}
	if keys := sortedKeys(got.byPage); fmt.Sprint(keys) != "[1 5 9]" {
		t.Fatalf("images landed on pages %v, want [1 5 9]", keys)
	}
	for page, imgs := range got.byPage {
		if len(imgs) != 1 {
			t.Errorf("page %d: %d images, want 1", page, len(imgs))
		}
		for _, rel := range imgs {
			if _, err := os.Stat(filepath.Join(out, filepath.FromSlash(rel))); err != nil {
				t.Errorf("page %d: listed image %s is not on disk: %v", page, rel, err)
			}
		}
	}
}

// The same fixture end to end: the emitted page for PDF page N shows the image of page N.
func TestExtract_Volume3ImagesOnTheirPages(t *testing.T) {
	tmp := t.TempDir()
	pdfPath := filepath.Join(tmp, "Volume_3.pdf")
	buildFixturePDF(t, pdfPath, 10, allPages(10), pageSet(1, 5, 9))
	out := filepath.Join(tmp, "out")
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatal(err)
	}

	book, err := Extract(pdfPath, out)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if len(book.Spine) != 10 {
		t.Fatalf("spine has %d pages, want 10", len(book.Spine))
	}
	for i, href := range book.SpineHrefs() {
		data, err := os.ReadFile(filepath.Join(out, href))
		if err != nil {
			t.Fatal(err)
		}
		hasImg := strings.Contains(string(data), "<img ")
		want := pageSet(1, 5, 9)[i+1]
		if hasImg != want {
			t.Errorf("page %d: has image = %v, want %v", i+1, hasImg, want)
		}
	}
}

// Two scanned plates at the end have no text, so pdftotext's output ends before them and its
// trimmed page list used to be the page count. A stub pdftotext stands in for the real one so
// the test runs where Poppler is not installed.
func TestExtractWithPDFToText_KeepsTrailingImageOnlyPages(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the stub pdftotext is a shell script")
	}
	tmp := t.TempDir()
	pdfPath := filepath.Join(tmp, "plates.pdf")
	buildFixturePDF(t, pdfPath, 5, pageSet(1, 2, 3), pageSet(4, 5))

	stub := filepath.Join(tmp, "pdftotext")
	script := "#!/bin/sh\nprintf '  Opening chapter paragraph\\n\\f  Second chapter paragraph\\n\\f  Closing chapter paragraph\\n\\f\\f\\f'\n"
	if err := os.WriteFile(stub, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	out := filepath.Join(tmp, "out")
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatal(err)
	}
	book, err := extractWithPDFToText(stub, pdfPath, out)
	if err != nil {
		t.Fatalf("extractWithPDFToText: %v", err)
	}
	hrefs := book.SpineHrefs()
	if len(hrefs) != 5 {
		t.Fatalf("got %d pages, want 5 (three text pages and two plates)", len(hrefs))
	}
	for _, i := range []int{3, 4} {
		data, err := os.ReadFile(filepath.Join(out, hrefs[i]))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), "<img ") {
			t.Errorf("page %d has no image", i+1)
		}
	}
}

// The page count from pdfcpu only extends the page list: a document it cannot read (count 0)
// keeps every page the text extractor produced.
func TestExtractWithPDFToText_UnreadableCountNeverShrinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the stub pdftotext is a shell script")
	}
	tmp := t.TempDir()
	pdfPath := filepath.Join(tmp, "not-really.pdf")
	if err := os.WriteFile(pdfPath, []byte("not a pdf"), 0o644); err != nil {
		t.Fatal(err)
	}
	stub := filepath.Join(tmp, "pdftotext")
	script := "#!/bin/sh\nprintf '  One\\n\\f  Two\\n\\f  Three\\n'\n"
	if err := os.WriteFile(stub, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(tmp, "out")
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatal(err)
	}
	book, err := extractWithPDFToText(stub, pdfPath, out)
	if err != nil {
		t.Fatalf("extractWithPDFToText: %v", err)
	}
	if n := len(book.Spine); n != 3 {
		t.Errorf("got %d pages, want 3", n)
	}
}

func TestImageFileNameAvoidsCollisions(t *testing.T) {
	used := map[string]bool{}
	var names []string
	for _, objNr := range []int{7, 9} {
		img := model.Image{ObjNr: objNr, Name: "Im0", FileType: "png"}
		name := imageFileName("Volume_3", 5, img, used)
		used[strings.ToLower(name)] = true
		names = append(names, name)
	}
	if names[0] == names[1] {
		t.Fatalf("two images got the same name %q", names[0])
	}
	if !regexp.MustCompile(`^Volume_3_5_Im0\.png$`).MatchString(names[0]) {
		t.Errorf("first name = %q, want pdfcpu's shape Volume_3_5_Im0.png", names[0])
	}
}
