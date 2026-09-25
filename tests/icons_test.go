package tests

// System-surface icon guards (ICON-RENDER rule 9, ticket 32). The program ICO, its favicon
// copies, the Explorer verb and document-type ICOs and the extension's action icons are render
// targets of internal/iconart: a committed file that differs from what it draws was edited by
// hand or not regenerated (scripts/generate-icon.ps1). The executables embed the three ICOs in
// the order internal/windowsreg names them by index.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"doc-html-translate/internal/iconart"
)

func decodeCommitted(t *testing.T, rel string) []image.Image {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Ext(rel) == ".ico" {
		frames, err := iconart.DecodeICO(raw)
		if err != nil {
			t.Fatalf("%s: %v", rel, err)
		}
		return frames
	}
	img, err := png.Decode(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("%s: %v", rel, err)
	}
	return []image.Image{img}
}

func sameImage(a, b image.Image) bool {
	if a.Bounds() != b.Bounds() {
		return false
	}
	r := a.Bounds()
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if color.NRGBAModel.Convert(a.At(x, y)) != color.NRGBAModel.Convert(b.At(x, y)) {
				return false
			}
		}
	}
	return true
}

func TestSystemSurfaceIconsAreRenderTargets(t *testing.T) {
	outs, err := iconart.RepoOutputs("..")
	if err != nil {
		t.Fatal(err)
	}
	for _, o := range outs {
		got := decodeCommitted(t, o.Path)
		if len(got) != len(o.Images) {
			t.Errorf("%s has %d images, the generator draws %d - run scripts/generate-icon.ps1", o.Path, len(got), len(o.Images))
			continue
		}
		for i := range got {
			if !sameImage(got[i], o.Images[i]) {
				t.Errorf("%s image %d (%v) differs from what internal/iconart draws - run scripts/generate-icon.ps1, never edit it by hand",
					o.Path, i, o.Images[i].Bounds().Size())
			}
		}
	}
}

// ICON-RENDER rule 9: the program ICO spans 16 to 256 px, the verb has 16 with 20, 24 and 32.
func TestICOsCarryTheRuleNineSizes(t *testing.T) {
	want := map[string][]int{
		"assets/doc-html-translate.ico": {16, 20, 24, 32, 48, 256},
		"assets/document-type.ico":      {16, 20, 24, 32, 48, 256},
		"assets/convert-verb.ico":       {16, 20, 24, 32},
	}
	for rel, sizes := range want {
		have := map[int]bool{}
		for _, f := range decodeCommitted(t, rel) {
			have[f.Bounds().Dx()] = true
		}
		for _, s := range sizes {
			if !have[s] {
				t.Errorf("%s has no %d px frame", rel, s)
			}
		}
	}
}

// Both executables embed the ICOs in iconart.ExeIcons order - internal/windowsreg writes
// "<exe>",1 for the verb and "<exe>",2 for the document type.
func TestExecutablesEmbedTheIconsInIndexOrder(t *testing.T) {
	var want []string
	for _, p := range iconart.ExeIcons {
		want = append(want, "../../"+p)
	}
	for _, cmd := range []string{"doc-html-translate", "doc-html-ui"} {
		var vi struct{ IconPath string }
		if err := json.Unmarshal([]byte(readRepoFile(t, "cmd", cmd, "versioninfo.json")), &vi); err != nil {
			t.Fatal(err)
		}
		if vi.IconPath != strings.Join(want, ",") {
			t.Errorf("cmd/%s/versioninfo.json IconPath = %q, want %q", cmd, vi.IconPath, strings.Join(want, ","))
		}
	}
	src := readRepoFile(t, "internal", "windowsreg", "register_windows.go")
	for i, name := range []string{"iconMark", "iconVerb", "iconDocumentType"} {
		if !containsConst(src, name, i) {
			t.Errorf("internal/windowsreg: %s is not icon resource %d (%s)", name, i, iconart.ExeIcons[i])
		}
	}
}

func containsConst(src, name string, v int) bool {
	for _, line := range strings.Split(src, "\n") {
		f := strings.Fields(line)
		if len(f) >= 3 && f[0] == name && f[1] == "=" && f[2] == fmt.Sprint(v) {
			return true
		}
	}
	return false
}

// The extension's action icon is the mark on its own plate: the plate holds 3:1 on a light
// toolbar, the sheet on a dark one, and the brackets on the sheet (ICON-RENDER rule 9, 0.13).
func TestExtensionActionIconHoldsOnBothToolbars(t *testing.T) {
	var man struct {
		Action struct {
			DefaultIcon map[string]string `json:"default_icon"`
		} `json:"action"`
	}
	if err := json.Unmarshal([]byte(readRepoFile(t, "extension", "manifest.json")), &man); err != nil {
		t.Fatal(err)
	}
	for _, s := range iconart.ExtensionSizes {
		if got := man.Action.DefaultIcon[fmt.Sprint(s)]; got != fmt.Sprintf("icons/icon%d.png", s) {
			t.Errorf("manifest action.default_icon[%d] = %q", s, got)
		}
	}
	light := []color.Color{color.White, color.NRGBA{0xF1, 0xF3, 0xF4, 0xFF}, color.NRGBA{0xF7, 0xF7, 0xF7, 0xFF}}
	dark := []color.Color{color.NRGBA{0x20, 0x21, 0x24, 0xFF}, color.NRGBA{0x35, 0x36, 0x3A, 0xFF}, color.NRGBA{0x3B, 0x3B, 0x3B, 0xFF}}
	for _, bg := range light {
		if r := iconart.Contrast(iconart.Navy, bg); r < 3 {
			t.Errorf("plate on light toolbar %v: %.2f:1", bg, r)
		}
	}
	for _, bg := range dark {
		if r := iconart.Contrast(iconart.Sheet, bg); r < 3 {
			t.Errorf("sheet on dark toolbar %v: %.2f:1", bg, r)
		}
	}
	if r := iconart.Contrast(iconart.Navy, iconart.Sheet); r < 3 {
		t.Errorf("brackets on the sheet: %.2f:1", r)
	}
	// The 16 px icon is drawn in those colours: the corner is plate, the middle is sheet.
	img := decodeCommitted(t, "extension/icons/icon16.png")[0]
	if c := color.NRGBAModel.Convert(img.At(8, 2)); c != iconart.Sheet {
		t.Errorf("icon16 sheet pixel = %v", c)
	}
	if c := color.NRGBAModel.Convert(img.At(1, 8)); c != iconart.Navy {
		t.Errorf("icon16 plate pixel = %v", c)
	}
}
