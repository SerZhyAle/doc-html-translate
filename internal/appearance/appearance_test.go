package appearance

import (
	"regexp"
	"strings"
	"testing"
)

// The desktop names, repeated here rather than imported: internal/ocr and internal/htmlgen import
// this package, and the pin is about what these builders emit for those names.
var (
	desktopOverlay = OverlayNames{
		Container:   ".ocr-fig",
		Image:       ".ocr-fig>img",
		Plate:       ".ocr-box",
		HiddenPlate: "html.dht-ocr-off .ocr-box",
	}
	desktopPalette = PaletteNames{Prefix: "dht-", Attr: "data-dht-theme"}
)

var cssComment = regexp.MustCompile(`(?s)/\*.*?\*/`)

// rules parses a flat stylesheet into selector -> property -> value. Values are compared as
// written, apart from surrounding whitespace.
func rules(t *testing.T, css string) map[string]map[string]string {
	t.Helper()
	out := map[string]map[string]string{}
	css = cssComment.ReplaceAllString(css, "")
	for _, r := range strings.Split(css, "}") {
		sel, body, ok := strings.Cut(r, "{")
		if !ok {
			if strings.TrimSpace(r) != "" {
				t.Fatalf("unparsed CSS fragment %q", r)
			}
			continue
		}
		sel = strings.TrimSpace(sel)
		if out[sel] == nil {
			out[sel] = map[string]string{}
		}
		for _, d := range strings.Split(body, ";") {
			p, v, ok := strings.Cut(d, ":")
			if !ok {
				continue
			}
			out[sel][strings.TrimSpace(p)] = strings.TrimSpace(v)
		}
	}
	return out
}

func compare(t *testing.T, what string, got, want map[string]map[string]string) {
	t.Helper()
	for sel, decls := range want {
		for p, v := range decls {
			if g, ok := got[sel][p]; !ok {
				t.Errorf("%s: %s lost %s:%s", what, sel, p, v)
			} else if g != v {
				t.Errorf("%s: %s %s is %q, shipped %q", what, sel, p, g, v)
			}
		}
		for p := range got[sel] {
			if _, ok := decls[p]; !ok {
				t.Errorf("%s: %s gained %s:%s", what, sel, p, got[sel][p])
			}
		}
	}
	for sel := range got {
		if _, ok := want[sel]; !ok {
			t.Errorf("%s: new rule %s", what, sel)
		}
	}
}

// TestOverlayCSSMatchesShipped: the overlay stylesheet the desktop pages carried on 2026-08-15,
// before it was derived. Deriving it must not move a rendered pixel; the one textual change is
// that the measurement notes now ship as comments. Declaration order may differ, content may not.
func TestOverlayCSSMatchesShipped(t *testing.T) {
	const shipped = `.ocr-fig{position:relative;display:block;width:100%;max-width:100%;margin:0 auto;container-type:inline-size;line-height:1.1}
.ocr-fig>img{display:block;width:100%;height:auto;margin:0;max-height:none}
.ocr-box{position:absolute;box-sizing:border-box;overflow:hidden;background:#fff;border-radius:0.35em;padding:0.08em 0.28em;color:#111;display:flex;align-items:center;justify-content:center;text-align:center;white-space:pre-wrap;overflow-wrap:anywhere;word-break:break-word;font-family:"Segoe UI",system-ui,Arial,sans-serif;-webkit-print-color-adjust:exact;print-color-adjust:exact}
html.dht-ocr-off .ocr-box{display:none}`
	compare(t, "overlay", rules(t, OverlayCSS(desktopOverlay)), rules(t, shipped))
}

// TestPaletteCSSMatchesShipped: the four reader themes as the desktop pages carried them on
// 2026-08-15.
func TestPaletteCSSMatchesShipped(t *testing.T) {
	const shipped = `:root{--dht-bg:#faf9f7;--dht-fg:#1b1b1b;--dht-muted:#6b6b6b;--dht-bar-bg:#ffffff;--dht-bar-fg:#222222;--dht-border:#e2e0db;--dht-accent:#2563eb;--dht-link:#1a4fb4}
html[data-dht-theme="sepia"]{--dht-bg:#f4ecd8;--dht-fg:#4a3f2f;--dht-muted:#7a6c54;--dht-bar-bg:#efe6cf;--dht-bar-fg:#4a3f2f;--dht-border:#ddd0b0;--dht-accent:#8a5a2b;--dht-link:#7a4a1b}
html[data-dht-theme="dark"]{--dht-bg:#1a1a1c;--dht-fg:#e6e4df;--dht-muted:#9a9893;--dht-bar-bg:#232327;--dht-bar-fg:#e6e4df;--dht-border:#36363b;--dht-accent:#5b8dff;--dht-link:#8fb4ff}
html[data-dht-theme="night"]{--dht-bg:#0a0a0b;--dht-fg:#9a9a9a;--dht-muted:#6a6a6a;--dht-bar-bg:#131315;--dht-bar-fg:#b8b8b8;--dht-border:#262629;--dht-accent:#5599d6;--dht-link:#6aa8e0}`
	compare(t, "palette", rules(t, PaletteCSS(desktopPalette)), rules(t, shipped))
}

// TestSourceShape: the schema rules the generators rely on. A note containing "*/" would end its
// comment early and ship the rest as CSS; a colour outside 6-digit lowercase hex would compare
// unequal to the same colour written the other way on the other edition.
func TestSourceShape(t *testing.T) {
	s := Load()
	hex := regexp.MustCompile(`^#[0-9a-f]{6}$`)
	for role, decls := range s.Roles {
		seen := map[string]bool{}
		for _, d := range decls {
			if seen[d.Property] {
				t.Errorf("role %s declares %s twice", role, d.Property)
			}
			seen[d.Property] = true
			if strings.Contains(d.Note, "*/") || strings.ContainsAny(d.Value, ";{}") {
				t.Errorf("role %s %s: value or note would break the emitted CSS", role, d.Property)
			}
		}
	}
	want := []string{"bg", "fg", "muted", "barBg", "barFg", "border", "accent", "link"}
	for _, th := range s.Themes {
		var got []string
		for _, c := range th.Colors {
			got = append(got, c.Token)
			if !hex.MatchString(c.Value) {
				t.Errorf("theme %s %s = %q, want 6-digit lowercase hex", th.Name, c.Token, c.Value)
			}
		}
		if strings.Join(got, ",") != strings.Join(want, ",") {
			t.Errorf("theme %s tokens %v, want %v", th.Name, got, want)
		}
	}
	for _, d := range s.Divergences {
		if d.What == "" || d.Reason == "" {
			t.Errorf("divergence %+v: what and reason are both required", d)
		}
	}
}

func TestTokenProperty(t *testing.T) {
	for in, want := range map[string]string{"bg": "bg", "barBg": "bar-bg", "barFg": "bar-fg"} {
		if got := TokenProperty(in); got != want {
			t.Errorf("TokenProperty(%q) = %q, want %q", in, got, want)
		}
	}
}
