package runner

import (
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"doc-html-translate/tools/ocrlab/evidence"
)

// A page shaped like the shipped overlay: the same class names, the same container query, the
// same percent-positioned plate. Enough for the probe to have something real to measure without
// needing tesseract.
const fixturePage = `<html><head><style>
.ocr-fig{position:relative;display:block;width:100%;max-width:100%;margin:0 auto;container-type:inline-size;line-height:1.1}
.ocr-fig>img{display:block;width:100%;height:auto;margin:0;max-height:none}
.ocr-box{position:absolute;box-sizing:border-box;overflow:hidden;background:#fff;color:#111;display:flex;align-items:center;justify-content:center;text-align:center}
</style></head><body>
<span class="ocr-fig" style="aspect-ratio:400 / 200"><img src="img.png">
<span class="ocr-box" style="left:10.00%;top:20.00%;width:50.00%;min-height:15.00%;font-size:3cqw">Hello there reader</span>
</span>
</body></html>`

// onePixelPNG is a 400x200 grey PNG, written by the test so the fixture needs no corpus.
func writeFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	img := makeGreyPNG(t, 400, 200)
	if err := os.WriteFile(filepath.Join(dir, "img.png"), img, 0o644); err != nil {
		t.Fatal(err)
	}
	page := filepath.Join(dir, "page_001.html")
	if err := os.WriteFile(page, []byte(fixturePage), 0o644); err != nil {
		t.Fatal(err)
	}
	return page
}

// The probe must complete and the browser must exit. This is the test the first implementation
// would have failed: it waited on requestAnimationFrame, which never fires under
// --virtual-time-budget with no compositor, so the browser hung until the runner killed it.
func TestProbeCollectsPlatesAndExits(t *testing.T) {
	if testing.Short() {
		t.Skip("browser test skipped in short mode")
	}
	browser, err := FindBrowser(filepath.Join(t.TempDir(), "profile"))
	if err != nil {
		t.Skipf("no headless browser available: %v", err)
	}
	// The session outlives a single page now, so the test owns closing it - and must, or the
	// browser keeps the profile open and t.TempDir cannot clean up after itself.
	defer browser.Close()
	page := writeFixture(t)
	probePage, err := injectProbe(page)
	if err != nil {
		t.Fatal(err)
	}

	start := time.Now()
	dom, err := browser.DumpDOM(probePage, "#ocrlab-collect", Viewports[0])
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("after %s: %v", elapsed, err)
	}
	if elapsed > 30*time.Second {
		t.Errorf("the browser took %s for one page - it is waiting on something that never happens", elapsed)
	}

	res, err := extractProbeResult(dom)
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK {
		t.Fatalf("probe reported: %v", res.Errors)
	}
	if res.ImageRect == nil || res.ImageRect.NaturalWidth != 400 {
		t.Fatalf("image rect = %+v, want the 400 px natural width", res.ImageRect)
	}

	// One plate per stress case, and every case must be present.
	byCase := map[string]int{}
	for _, p := range res.Plates {
		byCase[p.StressCase]++
	}
	for _, name := range StressNames() {
		if byCase[name] == 0 {
			t.Errorf("no plate recorded for stress case %q", name)
		}
	}

	// Geometry must come back in the image's own pixels: the plate is at 10%/20% of a 400x200
	// image, so roughly (40,40).
	var primary *probePlate
	for i := range res.Plates {
		if res.Plates[i].StressCase == PrimaryStress {
			primary = &res.Plates[i]
			break
		}
	}
	if primary == nil {
		t.Fatal("no plate for the primary case")
	}
	if primary.Rect.X0 < 30 || primary.Rect.X0 > 50 || primary.Rect.Y0 < 30 || primary.Rect.Y0 > 50 {
		t.Errorf("plate at (%d,%d), want about (40,40) in natural image pixels",
			primary.Rect.X0, primary.Rect.Y0)
	}
	if primary.Text != "Hello there reader" {
		t.Errorf("primary text = %q, want the original", primary.Text)
	}
	if primary.ClientHeight == 0 {
		t.Error("no clientHeight recorded, so clipping could never be detected")
	}

	// A longer replacement must actually be longer, or the stress cases prove nothing.
	for _, p := range res.Plates {
		if p.StressCase == "long-latin" && len([]rune(p.Text)) <= len([]rune(primary.Text)) {
			t.Errorf("long-latin text (%d runes) is not longer than the original (%d)",
				len([]rune(p.Text)), len([]rune(primary.Text)))
		}
	}
}

func TestStressCasesAreDeterministicAndComplete(t *testing.T) {
	want := []string{"none", "short", "long-latin", "long-cyrillic", "rtl-arabic", "cjk"}
	got := StressNames()
	if len(got) != len(want) {
		t.Fatalf("stress cases = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("stress case %d = %q, want %q", i, got[i], want[i])
		}
	}
	if StressCases[0].Text != "" {
		t.Error("the primary case must leave the recognized text alone")
	}
	var sawRTL bool
	for _, c := range StressCases {
		if c.Dir == "rtl" {
			sawRTL = true
		}
	}
	if !sawRTL {
		t.Error("no right-to-left case, so direction is never exercised")
	}
}

func TestConcealmentDiagnosticHidesDesktopTextWithoutLayoutChange(t *testing.T) {
	if testing.Short() {
		t.Skip("browser check")
	}
	browser, err := FindBrowser(filepath.Join(t.TempDir(), "profile"))
	if err != nil {
		t.Skipf("browser absent: %v", err)
	}
	defer browser.Close()
	page := writeFixture(t)
	probe, err := injectProbe(page)
	if err != nil {
		t.Fatal(err)
	}
	var results []*probeResult
	for _, fragment := range []string{"#ocrlab-stress=none", "#ocrlab-stress=none-hidden"} {
		dom, err := browser.DumpDOM(probe, fragment, Viewports[0])
		if err != nil {
			t.Fatal(err)
		}
		r, err := extractProbeResult(dom)
		if err != nil {
			t.Fatal(err)
		}
		results = append(results, r)
	}
	if len(results[0].Plates) != 1 || len(results[1].Plates) != 1 {
		t.Fatal("missing plate observation")
	}
	a, b := results[0].Plates[0], results[1].Plates[0]
	if a.Rect != b.Rect || a.FontPx != b.FontPx || a.ScrollHeight != b.ScrollHeight || a.ScrollWidth != b.ScrollWidth {
		t.Fatal("diagnostic changed layout")
	}
	if b.Ink != "rgba(0, 0, 0, 0)" {
		t.Fatalf("desktop text was not concealed: %q", b.Ink)
	}
}

func TestPartialScreenshotCannotBecomeCompleteEvidence(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "shot.png")
	if err := os.WriteFile(path, makeGreyPNG(t, 100, 100), 0644); err != nil {
		t.Fatal(err)
	}
	r := &probeRect{Left: 0, Top: 0, Width: 100, Height: 200}
	if err := CropToImage(path, r, 1, 100, 200, filepath.Join(dir, "mapped.png")); err == nil || !strings.Contains(err.Error(), "incomplete image capture") {
		t.Fatalf("partial screenshot accepted: %v", err)
	}
}

func TestBandedCapturePreservesTallImageCoordinates(t *testing.T) {
	if testing.Short() {
		t.Skip("browser check")
	}
	dir := t.TempDir()
	browser, err := FindBrowser(filepath.Join(dir, "profile"))
	if err != nil {
		t.Skipf("browser absent: %v", err)
	}
	defer browser.Close()
	if err := os.WriteFile(filepath.Join(dir, "img.png"), makeGreyPNG(t, 400, 2400), 0644); err != nil {
		t.Fatal(err)
	}
	page := filepath.Join(dir, "page.html")
	content := strings.Replace(fixturePage, "aspect-ratio:400 / 200", "aspect-ratio:400 / 2400", 1)
	content = strings.Replace(content, "</head>", "<style>.ocr-fig{width:400px}</style></head>", 1)
	start := strings.Index(content, `<span class="ocr-box"`)
	end := start + strings.Index(content[start:], "</span>") + len("</span>")
	content = content[:start] + content[end:]
	if err := os.WriteFile(page, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	probe, err := injectProbe(page)
	if err != nil {
		t.Fatal(err)
	}
	dom, err := browser.DumpDOM(probe, "#ocrlab-stress=none", Viewports[0])
	if err != nil {
		t.Fatal(err)
	}
	r, err := extractProbeResult(dom)
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "full.png")
	if err := browser.ImageScreenshot(probe, "#ocrlab-stress=none", Viewports[0], r.ImageRect, 400, 2400, out); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(out)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	im, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	if im.Bounds().Dx() != 400 || im.Bounds().Dy() != 2400 {
		t.Fatal(im.Bounds())
	}
	dark, _, _, _ := im.At(100, 605).RGBA()
	bottom, _, _, _ := im.At(100, 1800).RGBA()
	if dark > 60*257 || bottom < 200*257 {
		t.Fatalf("tall source coordinates changed: dark=%d bottom=%d", dark, bottom)
	}
}

func TestViewportsArePinnedAndVaried(t *testing.T) {
	if len(Viewports) < 2 {
		t.Fatal("drift cannot be measured from a single viewport")
	}
	seen := map[string]bool{}
	var sawHiDPI bool
	for _, v := range Viewports {
		if seen[v.Name] {
			t.Errorf("duplicate viewport name %q", v.Name)
		}
		seen[v.Name] = true
		if v.Width <= 0 || v.Height <= 0 {
			t.Errorf("viewport %q has no size", v.Name)
		}
		if v.DeviceScaleFactor > 1 {
			sawHiDPI = true
		}
	}
	if !sawHiDPI {
		t.Error("no high-DPI viewport, so device-pixel rounding is never exercised")
	}
	if Viewports[0].Name != "desktop" {
		t.Errorf("the primary viewport is %q; the report and the screenshots assume desktop", Viewports[0].Name)
	}
}

func TestBandedCaptureFractionalPhoneGeometry(t *testing.T) {
	if testing.Short() {
		t.Skip("browser check")
	}
	dir := t.TempDir()
	browser, err := FindBrowser(filepath.Join(dir, "profile"))
	if err != nil {
		t.Skipf("browser absent: %v", err)
	}
	defer browser.Close()
	if err := os.WriteFile(filepath.Join(dir, "img.png"), makeGreyPNG(t, 400, 658), 0644); err != nil {
		t.Fatal(err)
	}
	content := strings.Replace(fixturePage, "aspect-ratio:400 / 200", "aspect-ratio:400 / 658", 1)
	content = strings.Replace(content, "</head>", "<style>body{margin:0}.ocr-fig{width:370.5px;max-width:none;margin-left:9.75px;margin-top:16.25px}.ocr-box{display:none}</style></head>", 1)
	page := filepath.Join(dir, "page.html")
	if err := os.WriteFile(page, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	probe, err := injectProbe(page)
	if err != nil {
		t.Fatal(err)
	}
	v := Viewports[len(Viewports)-1]
	dom, err := browser.DumpDOM(probe, "#ocrlab-stress=none", v)
	if err != nil {
		t.Fatal(err)
	}
	r, err := extractProbeResult(dom)
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "full.png")
	if err := browser.ImageScreenshot(probe, "#ocrlab-stress=none", v, r.ImageRect, 400, 658, out); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(out)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	im, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	dark, _, _, _ := im.At(100, 169).RGBA()
	bottom, _, _, _ := im.At(100, 640).RGBA()
	if im.Bounds().Dx() != 400 || im.Bounds().Dy() != 658 || dark > 60*257 || bottom < 200*257 {
		t.Fatalf("fractional capture lost source coordinates: %v, dark=%d bottom=%d", im.Bounds(), dark, bottom)
	}
}

// A relative --user-data-dir makes headless Edge hang forever rather than fail, and the only
// symptom is a timeout minutes later. Cheap to assert, expensive to rediscover.
func TestBrowserProfileDirIsAbsolute(t *testing.T) {
	b, err := FindBrowser(filepath.Join("temp", "ocrlab", "profile-guard"))
	if err != nil {
		t.Skipf("no headless browser available: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(filepath.Join("temp", "ocrlab", "profile-guard")) })

	if !filepath.IsAbs(b.ProfileDir()) {
		t.Fatalf("profile dir %q is relative; headless Edge hangs on that", b.ProfileDir())
	}
	var found string
	for _, a := range launchArgs(b.ProfileDir()) {
		if strings.HasPrefix(a, "--user-data-dir=") {
			found = strings.TrimPrefix(a, "--user-data-dir=")
		}
	}
	if found == "" {
		t.Fatal("no --user-data-dir in the browser arguments")
	}
	if !filepath.IsAbs(found) {
		t.Errorf("--user-data-dir=%q is relative", found)
	}
	if strings.Contains(found, "/") && filepath.Separator == '\\' {
		t.Errorf("--user-data-dir=%q uses forward slashes on Windows", found)
	}
}

func TestExtractProbeResultReportsAMissingProbe(t *testing.T) {
	if _, err := extractProbeResult("<html><body>nothing here</body></html>"); err == nil {
		t.Fatal("a page without the probe element must be an error, not an empty result")
	} else if !strings.Contains(err.Error(), ProbeElementID) {
		t.Errorf("the error must name what was missing, got %v", err)
	}
}

func TestInjectProbeKeepsTheOriginalPage(t *testing.T) {
	page := writeFixture(t)
	before, err := os.ReadFile(page)
	if err != nil {
		t.Fatal(err)
	}
	out, err := injectProbe(page)
	if err != nil {
		t.Fatal(err)
	}
	if out == page {
		t.Fatal("the probe must go into a copy, never into the converted page itself")
	}
	after, err := os.ReadFile(page)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Error("injecting the probe modified the original page")
	}
	injected, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(injected), ProbeElementID) {
		t.Error("the copy does not carry the probe")
	}
	if !strings.Contains(string(injected), "ocr-fig") {
		t.Error("the copy lost the page it was supposed to measure")
	}
}

var _ = evidence.SchemaVersion // the runner and the schema move together

// shippedFitScript returns the desktop re-fit script exactly as the converted pages carry it. The
// constant is unexported in internal/ocr, and a copy here would let this test pass against a script
// nobody ships.
func shippedFitScript(t *testing.T) string {
	t.Helper()
	src, err := os.ReadFile(filepath.Join("..", "..", "..", "internal", "ocr", "overlay.go"))
	if err != nil {
		t.Fatal(err)
	}
	const open = "const ocrScript = `"
	i := strings.Index(string(src), open)
	if i < 0 {
		t.Fatal("internal/ocr/overlay.go no longer declares ocrScript as a raw string")
	}
	body := string(src)[i+len(open):]
	j := strings.Index(body, "`")
	if j < 0 {
		t.Fatal("ocrScript raw string is not terminated")
	}
	return body[:j]
}

// A plate's min-height is a percentage of its figure and Chrome reports the computed value as that
// percentage, so a fit that parses it as a pixel count pins a 34.29% plate to 34.29 px: two thirds of
// a 51 px phone figure, hanging past its bottom edge. This is that figure, at every stress case,
// under the shipped fit script.
func TestPlateStaysInsideAShortPhoneFigure(t *testing.T) {
	if testing.Short() {
		t.Skip("browser check")
	}
	dir := t.TempDir()
	browser, err := FindBrowser(filepath.Join(dir, "profile"))
	if err != nil {
		t.Skipf("browser absent: %v", err)
	}
	defer browser.Close()
	const w, h = 390, 51
	if err := os.WriteFile(filepath.Join(dir, "img.png"), makeGreyPNG(t, w, h), 0644); err != nil {
		t.Fatal(err)
	}
	content := strings.Replace(fixturePage, "aspect-ratio:400 / 200", "aspect-ratio:390 / 51", 1)
	content = strings.Replace(content, "left:10.00%;top:20.00%;width:50.00%;min-height:15.00%;font-size:3cqw",
		"left:5.00%;top:65.71%;width:90.00%;min-height:34.29%;font-size:2cqw", 1)
	content = strings.Replace(content, "</head>", "<style>body{margin:0}</style></head>", 1)
	content = strings.Replace(content, "</body>", "<script>"+shippedFitScript(t)+"</script></body>", 1)
	page := filepath.Join(dir, "page.html")
	if err := os.WriteFile(page, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	probe, err := injectProbe(page)
	if err != nil {
		t.Fatal(err)
	}
	dom, err := browser.DumpDOM(probe, "#ocrlab-collect", Viewports[len(Viewports)-1])
	if err != nil {
		t.Fatal(err)
	}
	res, err := extractProbeResult(dom)
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK {
		t.Fatalf("probe reported: %v", res.Errors)
	}
	if res.ImageRect == nil || res.ImageRect.Height < h-1 || res.ImageRect.Height > h+1 {
		t.Fatalf("image rect = %+v, want a figure about %d css px tall", res.ImageRect, h)
	}
	seen := map[string]bool{}
	for _, p := range res.Plates {
		seen[p.StressCase] = true
		// Rects are in the image's own pixels, which are css pixels here (390 px wide at 390 px).
		if p.Rect.Y1 > h+1 {
			t.Errorf("stress %q: the plate ends at y=%d, past the %d px figure", p.StressCase, p.Rect.Y1, h)
		}
		if p.Rect.Y0 < 0 {
			t.Errorf("stress %q: the plate starts at y=%d, above the figure", p.StressCase, p.Rect.Y0)
		}
	}
	for _, name := range StressNames() {
		if !seen[name] {
			t.Errorf("no plate recorded for stress case %q", name)
		}
	}
}
