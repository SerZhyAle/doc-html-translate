package runner

import (
	"errors"
	"fmt"
	"image"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	xdraw "golang.org/x/image/draw"

	"doc-html-translate/tools/ocrlab/evidence"
)

// BrowserEnvVar overrides the browser the runner drives. Named in the error a missing browser
// produces, so the fix is in the failure rather than in the documentation.
const BrowserEnvVar = "DOCHT_BROWSER"

// Viewports are pinned. A run that let the window float would produce numbers nobody could
// reproduce, and the whole point of the drift measurement is comparing the same plate at
// deliberately different geometries - a desktop window, a tablet portrait, and a phone at 2x
// where a device-pixel rounding error is twice as visible.
var Viewports = []evidence.Viewport{
	{Name: "desktop", Width: 1280, Height: 800, DeviceScaleFactor: 1},
	{Name: "tablet", Width: 768, Height: 1024, DeviceScaleFactor: 1},
	{Name: "phone", Width: 390, Height: 844, DeviceScaleFactor: 2},
}

const browserTimeout = 90 * time.Second

// Browser is a located headless browser, plus the live session once one has been needed.
type Browser struct {
	Path    string
	Name    string
	Version string

	profileDir string
	dt         *devtools
}

// session starts the browser on first use and reuses it afterwards. One process for the whole
// run: the states this runner asks for differ by URL fragment and viewport, neither of which is a
// reason to pay for a launch.
func (b *Browser) session() (*devtools, error) {
	if b.dt != nil {
		return b.dt, nil
	}
	d, err := b.start()
	if err != nil {
		return nil, err
	}
	b.dt = d
	return d, nil
}

// FindBrowser locates Edge or Chrome, in the same order and at the same paths as
// scripts/verify-html.ps1 so the lab and the existing verification tooling agree on what
// "the browser" means.
//
// A missing browser is a hard error. The strategic spec is explicit that a missing dependency
// must fail the run rather than silently reduce its coverage: a benchmark that quietly scored
// nothing would still print a table.
func FindBrowser(profileDir string) (*Browser, error) {
	var candidates []string
	if p := os.Getenv(BrowserEnvVar); p != "" {
		candidates = append(candidates, p)
	}
	if p := os.Getenv("CHROME"); p != "" {
		candidates = append(candidates, p)
	}
	if runtime.GOOS == "windows" {
		pf, pf86, lad := os.Getenv("ProgramFiles"), os.Getenv("ProgramFiles(x86)"), os.Getenv("LOCALAPPDATA")
		candidates = append(candidates,
			filepath.Join(pf, `Microsoft\Edge\Application\msedge.exe`),
			filepath.Join(pf86, `Microsoft\Edge\Application\msedge.exe`),
			filepath.Join(pf, `Google\Chrome\Application\chrome.exe`),
			filepath.Join(pf86, `Google\Chrome\Application\chrome.exe`),
			filepath.Join(lad, `Google\Chrome\Application\chrome.exe`),
		)
	} else {
		candidates = append(candidates,
			"/usr/bin/google-chrome", "/usr/bin/chromium", "/usr/bin/chromium-browser",
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		)
	}
	// The profile path must be absolute and in native separators.
	//
	// Measured on Windows 11 with Edge 141: a *relative* --user-data-dir (or one with forward
	// slashes) makes headless Edge hang forever instead of failing - the process never exits,
	// --dump-dom never returns, and the only symptom is the caller's timeout. An absolute path
	// with backslashes works every time. scripts/verify-html.ps1 gets this right by accident
	// because PowerShell hands it an absolute path; this had to learn it the hard way.
	abs, err := filepath.Abs(profileDir)
	if err != nil {
		return nil, fmt.Errorf("resolve browser profile dir: %w", err)
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return nil, fmt.Errorf("create browser profile dir: %w", err)
	}

	for _, c := range candidates {
		if c == "" {
			continue
		}
		if _, err := os.Stat(c); err == nil {
			b := &Browser{Path: c, profileDir: abs}
			b.Name, b.Version = describe(c)
			return b, nil
		}
	}
	return nil, fmt.Errorf("no headless browser found - install Chrome or Edge, or set %s to its executable", BrowserEnvVar)
}

// ProfileDir is the absolute profile path the browser is driven with. Exported so the guard
// test can assert the property that cost an afternoon.
func (b *Browser) ProfileDir() string { return b.profileDir }

// describe names the browser and its version.
//
// On Windows the version comes from the versioned sub-directory the installer keeps beside the
// executable (Application\141.0.3537.57\) and `--version` is never run. Measured: `msedge.exe
// --version` on Windows does not print a version and exit - it launches a full browser instance
// and stays running, so a naive version probe hangs the caller forever and leaves the browser
// behind. That was the first cause of a "the runner never finishes" symptom that looked like a
// rendering problem for an afternoon.
//
// Elsewhere `--version` is the right answer, but it is still run under a short deadline: a
// reporting field must never be able to stall a benchmark.
func describe(path string) (name, version string) {
	name = "chrome"
	if strings.Contains(strings.ToLower(filepath.Base(path)), "msedge") {
		name = "edge"
	}
	if v := versionFromSiblingDir(path); v != "" {
		return name, name + " " + v
	}
	if runtime.GOOS != "windows" {
		if v := versionFlag(path); v != "" {
			return name, v
		}
	}
	return name, name + " (version unknown)"
}

// versionFlag runs `<browser> --version` under a deadline and returns its first line.
func versionFlag(path string) string {
	cmd := exec.Command(path, "--version")
	out, err := os.CreateTemp("", "ocrlab-ver-*")
	if err != nil {
		return ""
	}
	defer func() { out.Close(); os.Remove(out.Name()) }()
	cmd.Stdout = out
	if err := cmd.Start(); err != nil {
		return ""
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		if err != nil {
			return ""
		}
	case <-time.After(versionTimeout):
		killTree(cmd)
		return ""
	}
	data, err := os.ReadFile(out.Name())
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			return line
		}
	}
	return ""
}

// versionTimeout bounds the version probe. Short on purpose: it is metadata for a report.
const versionTimeout = 5 * time.Second

// versionFromSiblingDir finds the highest version-shaped directory beside the executable.
func versionFromSiblingDir(exe string) string {
	entries, err := os.ReadDir(filepath.Dir(exe))
	if err != nil {
		return ""
	}
	best := ""
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		n := e.Name()
		if !strings.Contains(n, ".") || strings.ContainsFunc(n, func(r rune) bool {
			return r != '.' && (r < '0' || r > '9')
		}) {
			continue
		}
		if n > best {
			best = n
		}
	}
	return best
}

func fileURL(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	u := "file:///" + strings.ReplaceAll(abs, `\`, "/")
	return strings.ReplaceAll(u, " ", "%20"), nil
}

// DumpDOM loads a page at one viewport and returns the DOM once the probe has published its
// measurement. The name is kept from the switch it replaces, because that is what it delivers.
func (b *Browser) DumpDOM(pagePath, fragment string, v evidence.Viewport) (string, error) {
	d, err := b.openProbed(pagePath, fragment, v)
	if err != nil {
		return "", err
	}
	dom, err := d.evaluate("document.documentElement.outerHTML")
	if err != nil {
		return "", err
	}
	if len(dom) < 100 {
		return "", errors.New("browser returned an empty DOM")
	}
	return dom, nil
}

// Screenshot captures one page state to a PNG.
func (b *Browser) Screenshot(pagePath, fragment string, v evidence.Viewport, out string) error {
	d, err := b.openProbed(pagePath, fragment, v)
	if err != nil {
		return err
	}
	png, err := d.capture()
	if err != nil {
		return err
	}
	if len(png) == 0 {
		return fmt.Errorf("browser produced no screenshot for %s", out)
	}
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return err
	}
	return os.WriteFile(out, png, 0o644)
}

// Close ends the browser instance that owns this run's profile.
//
// A headless launch leaves helper processes (gpu, utility, crashpad) alive after the process we
// started has exited, and because every launch in a run shares one --user-data-dir they all
// belong to that one instance. Killing by image name would take the user's own browser with it,
// so the match is on the profile path, which is unique to this run. Best-effort: a leftover
// process is untidy, not wrong, and must never fail a run.
func (b *Browser) Close() {
	if b.dt != nil {
		b.dt.close()
		b.dt = nil
	}
	if runtime.GOOS != "windows" {
		return
	}
	script := fmt.Sprintf(
		`Get-CimInstance Win32_Process -Filter "Name='msedge.exe' OR Name='chrome.exe'" `+
			`| Where-Object { $_.CommandLine -like '*%s*' } `+
			`| ForEach-Object { Stop-Process -Id $_.ProcessId -Force -ErrorAction SilentlyContinue }`,
		strings.ReplaceAll(b.profileDir, `'`, `''`))
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script)
	cmd.Stdout, cmd.Stderr = nil, nil
	_ = cmd.Start()
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case <-done:
	case <-time.After(versionTimeout):
		killTree(cmd)
	}
}

// killTree ends the browser and the helpers it spawned. Killing only the process we started
// leaves the children running, and a corpus run would accumulate hundreds of them.
func killTree(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}
	if runtime.GOOS == "windows" {
		_ = exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(cmd.Process.Pid)).Run()
		return
	}
	_ = cmd.Process.Kill()
}

// CropToImage turns a full-page screenshot into the source image's own pixel space, so the
// residual-ink measurement can compare it with the original pixel for pixel.
//
// Without this every concealment number would be measured against a differently scaled,
// differently offset picture and would mean nothing. The crop rect comes from the probe (CSS
// pixels), scaled by the viewport's device scale factor to reach screenshot pixels.
func CropToImage(shotPath string, r *probeRect, dsf float64, naturalW, naturalH int, out string) error {
	if r == nil || r.Width <= 0 || r.Height <= 0 {
		return errors.New("no image rect recorded, cannot map the screenshot to image space")
	}
	f, err := os.Open(shotPath)
	if err != nil {
		return err
	}
	src, _, err := image.Decode(f)
	f.Close()
	if err != nil {
		return err
	}

	x0 := int(r.Left * dsf)
	y0 := int(r.Top * dsf)
	x1 := int((r.Left + r.Width) * dsf)
	y1 := int((r.Top + r.Height) * dsf)
	b := src.Bounds()
	x0, y0 = max(x0, b.Min.X), max(y0, b.Min.Y)
	x1, y1 = min(x1, b.Max.X), min(y1, b.Max.Y)
	if x1-x0 < 2 || y1-y0 < 2 {
		return fmt.Errorf("the image is not visible in the screenshot (crop %d,%d-%d,%d of %v)", x0, y0, x1, y1, b)
	}

	dst := image.NewRGBA(image.Rect(0, 0, naturalW, naturalH))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, image.Rect(x0, y0, x1, y1), xdraw.Src, nil)

	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return err
	}
	g, err := os.Create(out)
	if err != nil {
		return err
	}
	if err := png.Encode(g, dst); err != nil {
		g.Close()
		return err
	}
	return g.Close()
}
