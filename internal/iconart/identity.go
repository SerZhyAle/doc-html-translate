package iconart

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	_ "image/gif"  // demo.gif: read for its size
	_ "image/jpeg" // demo-poster.jpg: read for its size
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// The identity manifest of PRODUCT-IDENTITY 0.1 (section 5): the master and every file that
// shows the product's picture, with its surface, size and SHA-256. It is written by the
// generator after every rendition, never by hand; the catalog's reference checker reads it.
const (
	ManifestPath = "assets/identity/identity-manifest.json"
	// Generator is the one command that writes every rendition and the manifest.
	Generator = "pwsh -NoProfile -File scripts/generate-icon.ps1"
)

// surfaceFile is one manifest row before it is measured.
type surfaceFile struct {
	surface, path, source, licence string
}

const ownWork = "own work of the product, under the repository's licence"

// MSIXCopyDir holds a copy of the package's logos: build-msix.ps1 draws them into the package at
// build time (MSIXOutputs), and the generator writes the same bytes here so the manifest can
// name and hash every MSIX surface.
const MSIXCopyDir = "assets/identity/out/msix"

// msixSurface maps an MSIX logo file to its surface id; the document-type images are a glyph,
// not the product's picture, and are left out.
func msixSurface(name string) string {
	for _, p := range []struct{ prefix, surface string }{
		{"Square44x44Logo.", "msix.square44"},
		{"StoreLogo.", "msix.storelogo"},
		{"Square150x150Logo.", "msix.square150"},
		{"Wide310x150Logo.", "msix.wide310"},
		{"Square71x71Logo.", "other.msix-square71"},
	} {
		if strings.HasPrefix(name, p.prefix) {
			return p.surface
		}
	}
	return ""
}

// MSIXIdentityOutputs are the MSIX logos that show the product, under MSIXCopyDir.
func MSIXIdentityOutputs(root string) ([]Output, error) {
	outs, err := MSIXOutputs(root)
	if err != nil {
		return nil, err
	}
	var keep []Output
	for _, o := range outs {
		if msixSurface(o.Path) != "" {
			keep = append(keep, Output{Path: path.Join(MSIXCopyDir, o.Path), Images: o.Images})
		}
	}
	return keep, nil
}

// identitySurfaces lists every picture of the product by its PRODUCT-IDENTITY surface id. One
// file may serve several surfaces (the program ICO is also the site's favicon and its identity
// block).
func identitySurfaces(root string) ([]surfaceFile, error) {
	rows := []surfaceFile{
		{surface: "app.exe-ico", path: "assets/doc-html-translate.ico"},
		{surface: "app.installer-ico", path: "assets/doc-html-translate.ico"},
		{surface: "app.titlebar", path: "cmd/doc-html-ui/favicon.ico"},
		{surface: "other.book-favicon", path: "internal/htmlgen/favicon.ico", source: "generated from the master by " + Generator + " (the browser-tab icon of a converted book)", licence: ownWork},
		{surface: "web.favicon-ico", path: "assets/doc-html-translate.ico"},
		{surface: "site.identity", path: "assets/doc-html-translate.ico"},
		{surface: "web.og-image", path: "assets/social-card-1200x630.png"},
		{surface: "github.social-preview", path: "assets/social-card-1280x640.png"},
		{surface: "ext.chrome.promo-small", path: "extension/store/chrome-promo-small.png"},
		{surface: "help.head", path: HelpHeadSVGPath},
		{surface: "web.favicon-ico", path: "help/favicon.ico"},
		{surface: "web.favicon-ico", path: "cmd/doc-html-ui/help/favicon.ico"},
		{surface: "other.social-mark", path: SocialMarkPath, source: "generated from the master by " + Generator + " (the plate of the social card)", licence: ownWork},
		{surface: "hub.card", path: HubCardPath},
	}
	for _, s := range ExtensionSizes {
		rows = append(rows, surfaceFile{surface: "ext.chrome.icon", path: fmt.Sprintf("extension/icons/icon%d.png", s)})
	}
	for _, l := range StoreLogos {
		rows = append(rows, surfaceFile{surface: "store.ms.logo", path: l.Path})
	}
	msix, err := MSIXIdentityOutputs(root)
	if err != nil {
		return nil, err
	}
	for _, o := range msix {
		s := surfaceFile{surface: msixSurface(path.Base(o.Path)), path: o.Path}
		if strings.HasPrefix(s.surface, "other.") {
			s.source, s.licence = "generated from the master by "+Generator+" (the MSIX small tile, a size the surface list has no id for)", ownWork
		}
		rows = append(rows, s)
	}
	// Section 3.2 rule 16: every other image the product ships, with its origin.
	demo := "tools/store/make-demo.ps1 - a recording of the real build in headless Chrome"
	rows = append(rows,
		surfaceFile{surface: "other.demo-video", path: "assets/demo.mp4", source: demo, licence: ownWork},
		surfaceFile{surface: "other.demo-gif", path: "assets/demo.gif", source: demo, licence: ownWork},
		surfaceFile{surface: "other.demo-poster", path: "assets/demo-poster.jpg", source: demo, licence: ownWork},
	)
	return rows, nil
}

// screenshotDirs are the store screenshot folders (PRODUCT-IDENTITY rule 7), each with the
// script that takes them from the real build.
var screenshotDirs = []struct{ dir, surface, source string }{
	{"tools/store", "other.store-screenshot", "tools/store/make-screenshot.ps1 - captured from the real build"},
	{"extension/store/screenshots", "other.ext-screenshot", "extension/scripts/make-screenshots.mjs - captured from the real extension"},
}

type manifestMaster struct {
	SVG        string `json:"svg"`
	Small      string `json:"small"`
	Medium     string `json:"medium"`
	Background string `json:"background"`
	SHA256     string `json:"sha256"`
}

type manifestRendition struct {
	Surface string `json:"surface"`
	Path    string `json:"path"`
	Format  string `json:"format"`
	Width   int    `json:"width,omitempty"`
	Height  int    `json:"height,omitempty"`
	SHA256  string `json:"sha256"`
	Source  string `json:"source,omitempty"`
	Licence string `json:"licence,omitempty"`
}

type manifest struct {
	SchemaVersion int                 `json:"schemaVersion"`
	Product       string              `json:"product"`
	Family        *string             `json:"family"`
	Master        manifestMaster      `json:"master"`
	Generator     string              `json:"generator"`
	Renditions    []manifestRendition `json:"renditions"`
}

// Manifest measures every file of the identity under root and returns the manifest's bytes.
// A file that is missing is an error: the generator writes them all before it runs this.
func Manifest(root string) ([]byte, error) {
	read := func(rel string) ([]byte, error) { return os.ReadFile(filepath.Join(root, filepath.FromSlash(rel))) }
	master, err := read(MasterSVGPath)
	if err != nil {
		return nil, err
	}
	m := manifest{
		SchemaVersion: 1,
		Product:       "doc-html-translate",
		Master: manifestMaster{
			SVG: MasterSVGPath, Small: SmallSVGPath, Medium: MediumSVGPath,
			Background: fmt.Sprintf("#%02X%02X%02X", Navy.R, Navy.G, Navy.B),
			SHA256:     hash(master),
		},
		Generator: Generator,
	}
	rows, err := identitySurfaces(root)
	if err != nil {
		return nil, err
	}
	for _, d := range screenshotDirs {
		entries, err := os.ReadDir(filepath.Join(root, filepath.FromSlash(d.dir)))
		if err != nil {
			return nil, err
		}
		var names []string
		for _, e := range entries {
			if !e.IsDir() && strings.EqualFold(filepath.Ext(e.Name()), ".png") {
				names = append(names, e.Name())
			}
		}
		sort.Strings(names)
		for _, n := range names {
			rows = append(rows, surfaceFile{
				surface: d.surface + "-" + strings.TrimSuffix(n, filepath.Ext(n)),
				path:    path.Join(d.dir, n), source: d.source, licence: ownWork,
			})
		}
	}
	for _, r := range rows {
		raw, err := read(r.path)
		if err != nil {
			return nil, err
		}
		format := strings.TrimPrefix(strings.ToLower(path.Ext(r.path)), ".")
		w, h, err := imageSize(raw, format)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", r.path, err)
		}
		m.Renditions = append(m.Renditions, manifestRendition{
			Surface: r.surface, Path: r.path, Format: format, Width: w, Height: h,
			SHA256: hash(raw), Source: r.source, Licence: r.licence,
		})
	}
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(m); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func hash(b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

// imageSize gives the pixel size the manifest records: an ICO by its largest frame (the size
// the reference checker reads), a PNG, JPEG or GIF by its header, nothing for a vector or a video.
func imageSize(raw []byte, format string) (int, int, error) {
	switch format {
	case "ico":
		if len(raw) < 6 {
			return 0, 0, fmt.Errorf("not an icon file")
		}
		n := int(binary.LittleEndian.Uint16(raw[4:]))
		w, h := 0, 0
		for i := 0; i < n; i++ {
			o := 6 + 16*i
			if o+1 >= len(raw) {
				return 0, 0, fmt.Errorf("icon directory runs past the end of the file")
			}
			fw, fh := int(raw[o]), int(raw[o+1])
			if fw == 0 {
				fw = 256
			}
			if fh == 0 {
				fh = 256
			}
			if fw*fh > w*h {
				w, h = fw, fh
			}
		}
		return w, h, nil
	case "png", "jpg", "jpeg", "gif":
		cfg, _, err := image.DecodeConfig(bytes.NewReader(raw))
		if err != nil {
			return 0, 0, err
		}
		return cfg.Width, cfg.Height, nil
	}
	return 0, 0, nil
}
