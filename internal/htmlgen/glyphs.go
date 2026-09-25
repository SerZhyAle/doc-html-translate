package htmlgen

import (
	"fmt"
	"strings"
)

// Glyphs holds the path data of every vocabulary glyph the reader chrome draws, keyed by the
// meaning's id (ICON-SET section 2). Each value is the `d` of the catalog's 24 x 24 drawing,
// copied verbatim from assets/glyphs/<id>.svg; tests/iconography_test.go holds this table to
// those files and to the extension's own copy (extension/src/glyphs.js), so the two editions draw
// one picture per meaning. The drawings derive from Material Icons (Apache-2.0,
// THIRD-PARTY-NOTICES.txt). A new control picks a meaning that is already here or in the
// catalog - never a private picture (ICON-SET rule 5); docs/GLYPH-MAP.md maps every glyph.
var Glyphs = map[string]string{
	"media.previous":           "M6 6h2v12H6zm3.5 6l8.5 6V6z",
	"media.next":               "M6 18l8.5-6L6 6v12zM16 6v12h2V6h-2z",
	"nav.contents":             "M3 7h2v2H3v-2zm0 4h2v2H3v-2zm0 4h2v2H3v-2zm4-8h14v2H7V7zm0 4h14v2H7v-2zm0 4h14v2H7v-2z",
	"action.text-smaller":      "M.99 19h2.42l1.27-3.58h5.65L11.59 19h2.42L8.75 5h-2.5L.99 19zm4.42-5.61L7.44 7.6h.12l2.03 5.79H5.41zM23 11v2h-8v-2h8z",
	"action.text-larger":       "M.99 19h2.42l1.27-3.58h5.65L11.59 19h2.42L8.75 5h-2.5L.99 19zm4.42-5.61L7.44 7.6h.12l2.03 5.79H5.41zM20 11h3v2h-3v3h-2v-3h-3v-2h3V8h2v3z",
	"view.text-layer":          "M20 4H4c-1.1 0-2 .9-2 2v12c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V6c0-1.1-.9-2-2-2zM4 12h4v2H4v-2zm10 6H4v-2h10v2zm6 0h-4v-2h4v2zm0-4H10v-2h10v2z",
	"app.theme":                "M12 22c5.52 0 10-4.48 10-10S17.52 2 12 2 2 6.48 2 12s4.48 10 10 10zm1-17.93c3.94.49 7 3.85 7 7.93s-3.05 7.44-7 7.93V4.07z",
	"nav.go-to-page":           "M7 15H5.5v-4.5H4V9h3v6zm6.5-1.5h-3v-1h2c.55 0 1-.45 1-1V10c0-.55-.45-1-1-1H9v1.5h3v1h-2c-.55 0-1 .45-1 1V15h4.5v-1.5zm6 .5v-4c0-.55-.45-1-1-1H15v1.5h3v1h-2v1h2v1h-3V15h3.5c.55 0 1-.45 1-1z",
	"nav.expand":               "M16.59 8.59L12 13.17 7.41 8.59 6 10l6 6 6-6z",
	"nav.collapse":             "M12 8l-6 6 1.41 1.41L12 10.83l4.59 4.58L18 14z",
	"feature.continue-reading": "M21,5c-1.11,-0.35 -2.33,-0.5 -3.5,-0.5 -1.95,0 -4.05,0.4 -5.5,1.5 -1.45,-1.1 -3.55,-1.5 -5.5,-1.5S2.45,4.9 1,6v14.65c0,0.25 0.25,0.5 0.5,0.5 0.1,0 0.15,-0.05 0.25,-0.05C3.1,20.45 5.05,20 6.5,20c1.95,0 4.05,0.4 5.5,1.5 1.35,-0.85 3.8,-1.5 5.5,-1.5 1.65,0 3.35,0.3 4.75,1.05 0.1,0.05 0.15,0.05 0.25,0.05 0.25,0 0.5,-0.25 0.5,-0.5V6c-0.6,-0.45 -1.25,-0.75 -2,-1zM10,17.5l-3,-2v-7l3,2v7zM21,17.5l-3,2v-7l3,-2v7z",
}

// glyphSVG returns the inline drawing of a vocabulary glyph for the chrome. It is decorative
// (aria-hidden): the control around it carries the meaning's name as text or as aria-label
// (ICON-RENDER rule 8). The colour is the surrounding text colour (ICON-RENDER rule 2), and the
// size is 1.15em of the chrome's own font, which lands on the 16 px tier in the 14 px bar.
//
// Inline SVG rather than a stylesheet mask: a converted book is a folder of self-contained
// pages opened from file://, and one drawing costs about 150 bytes a page.
func glyphSVG(id string) string {
	d, ok := Glyphs[id]
	if !ok {
		panic("htmlgen: no vocabulary glyph " + id)
	}
	return fmt.Sprintf(`<svg class="dht-glyph" viewBox="0 0 24 24" aria-hidden="true" focusable="false"><path fill="currentColor" d="%s"/></svg>`, d)
}

// glyphMaskURL returns a vocabulary glyph as a CSS url() for a mask, for a marker the stylesheet
// draws (a ::before), where no element can hold an inline drawing. Painted with background-color,
// the glyph still takes the text colour around it.
func glyphMaskURL(id string) string {
	d, ok := Glyphs[id]
	if !ok {
		panic("htmlgen: no vocabulary glyph " + id)
	}
	svg := `<svg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24'><path d='` + d + `'/></svg>`
	return `url("data:image/svg+xml,` + strings.NewReplacer("<", "%3C", ">", "%3E", "#", "%23", " ", "%20").Replace(svg) + `")`
}
