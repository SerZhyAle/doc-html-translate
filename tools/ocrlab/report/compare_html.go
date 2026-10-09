package report

import (
	"encoding/base64"
	"fmt"
	"html"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"doc-html-translate/tools/ocrlab/evidence"
)

// WriteComparisonHTML embeds the selected renders so this comparison travels independently.
func WriteComparisonHTML(beforeDir, afterDir string, before, after *Data, c Comparison) error {
	var b strings.Builder
	fmt.Fprintf(&b, `<!doctype html><html lang="en"><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Run comparison</title><style>%s</style><h1>%s to %s (%s)</h1><p>Regression acceptance eligible: %t; baseline approval remains separate.</p>`, reportCSS, html.EscapeString(c.Before), html.EscapeString(c.After), html.EscapeString(c.Edition), c.Compatible)
	for _, lim := range c.Limitations {
		fmt.Fprintf(&b, "<p class=\"reasons\">%s</p>", html.EscapeString(lim))
	}
	b.WriteString(reportControls)
	for _, s := range c.Scenes {
		finding := "clear"
		if s.State == "regressed" {
			finding = "fail"
		}
		if s.State == "unavailable" || s.State == "changed-input" || s.State == "incompatible" || s.State == "removed" {
			finding = "unavailable"
		}
		fmt.Fprintf(&b, "<section class=\"scene\" data-finding=\"%s\"><h2>%s: %s</h2><p>%s</p>", finding, html.EscapeString(s.SceneID), html.EscapeString(s.State), html.EscapeString(strings.Join(s.Findings, "; ")))
		br, ar := before.Run.Find(s.SceneID), after.Run.Find(s.SceneID)
		if br != nil && ar != nil {
			for _, o := range ar.Observations {
				var prior *evidence.Observation
				for i := range br.Observations {
					v := &br.Observations[i]
					if v.Viewport == o.Viewport && v.StressCase == o.StressCase {
						prior = v
						break
					}
				}
				if prior == nil {
					continue
				}
				fmt.Fprintf(&b, "<details data-viewport=\"%s\" data-stress=\"%s\"><summary>%s / %s</summary><div class=\"pair\">", html.EscapeString(o.Viewport), html.EscapeString(o.StressCase), html.EscapeString(o.Viewport), html.EscapeString(o.StressCase))
				writeFigure(&b, "Before", embeddedShot(beforeDir, prior.Rendered), br, br.PlatesFor(o.Viewport, o.StressCase))
				writeFigure(&b, "After", embeddedShot(afterDir, o.Rendered), ar, ar.PlatesFor(o.Viewport, o.StressCase))
				b.WriteString("</div></details>")
			}
		}
		b.WriteString("</section>")
	}
	b.WriteString(reportScript + "</html>")
	return os.WriteFile(filepath.Join(afterDir, "comparison.html"), []byte(b.String()), 0644)
}

func embeddedShot(dir, rel string) string {
	path, err := evidence.SafePath(dir, rel)
	if err != nil {
		return ""
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return "data:" + http.DetectContentType(data) + ";base64," + base64.StdEncoding.EncodeToString(data)
}
