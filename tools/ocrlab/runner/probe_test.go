package runner

import (
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
