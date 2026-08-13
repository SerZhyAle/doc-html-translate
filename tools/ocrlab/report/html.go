package report

import (
	"fmt"
	"html"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"doc-html-translate/tools/ocrlab/evidence"
	"doc-html-translate/tools/ocrlab/metrics"
)

// reportCSS keeps the page readable in both themes without loading anything. The strict rule
// for this file is that it references no remote asset at all: a benchmark report that needs the
// network to render is one that stops rendering the moment it matters.
const reportCSS = `
:root{--bg:#fff;--fg:#111;--muted:#666;--line:#ddd;--bad:#b00020;--ok:#1a7f37;--card:#fafafa}
@media (prefers-color-scheme:dark){:root{--bg:#16181c;--fg:#e8e8e8;--muted:#9aa0a6;--line:#333;--bad:#ff6b6b;--ok:#4ade80;--card:#1e2126}}
*{box-sizing:border-box}
body{margin:0;padding:24px;background:var(--bg);color:var(--fg);font:15px/1.5 "Segoe UI",system-ui,sans-serif}
h1{font-size:22px;margin:0 0 4px}
h2{font-size:18px;margin:32px 0 8px;border-bottom:1px solid var(--line);padding-bottom:4px}
.meta{color:var(--muted);font-size:13px;margin-bottom:16px}
table{border-collapse:collapse;width:100%;font-size:13px;margin:8px 0 16px}
th,td{border:1px solid var(--line);padding:4px 8px;text-align:right}
th:first-child,td:first-child{text-align:left}
th{background:var(--card)}
.scene{border:1px solid var(--line);border-radius:6px;padding:12px;margin:16px 0;background:var(--card)}
.scene h3{margin:0 0 6px;font-size:16px}
.verdict{font-weight:600}
.verdict.pass{color:var(--ok)}
.verdict.fail{color:var(--bad)}
.reasons{margin:6px 0 10px;padding-left:18px;color:var(--bad)}
.pair{display:flex;flex-wrap:wrap;gap:12px}
.pair figure{margin:0;flex:1 1 320px;min-width:260px}
.pair figcaption{font-size:12px;color:var(--muted);margin-bottom:4px}
.frame{position:relative;display:block;border:1px solid var(--line);background:#fff}
.frame img{display:block;width:100%;height:auto}
.plate{position:absolute;border:1.5px solid rgba(220,0,60,.9);background:rgba(220,0,60,.06)}
.numbers{font-size:12px;color:var(--muted);margin-top:8px}
.skipped td:first-child{color:var(--muted)}
.stress{display:flex;flex-wrap:wrap;gap:8px;margin-top:10px}
.stress figure{margin:0;flex:1 1 200px;min-width:160px}
.stress figcaption{font-size:11px;color:var(--muted)}
`

// WriteHTML renders report.html: for each scene the source and the result side by side, the
// plate outlines drawn over the result, the stress captures, and the named reason for every
// failure.
func WriteHTML(dir string, d *Data) error {
	var b strings.Builder
	b.WriteString("<!doctype html>\n<html lang=\"en\">\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<meta name=\"viewport\" content=\"width=device-width,initial-scale=1\">\n")
	fmt.Fprintf(&b, "<title>OCR visual-fidelity report - %s</title>\n", html.EscapeString(d.Run.RunID))
	fmt.Fprintf(&b, "<style>%s</style>\n</head>\n<body>\n", reportCSS)

	fmt.Fprintf(&b, "<h1>OCR visual-fidelity report - %s</h1>\n", html.EscapeString(d.Run.RunID))
	fmt.Fprintf(&b, "<div class=\"meta\">%s edition &middot; %s &middot; tessdata %s &middot; %s &middot; %s</div>\n",
		html.EscapeString(string(d.Run.Edition)),
		html.EscapeString(or(d.Run.Engine.Tesseract, "engine unknown")),
		html.EscapeString(or(d.Run.Engine.TessdataVersion, "unknown")),
		html.EscapeString(or(d.Run.Browser.Version, d.Run.Browser.Name)),
		html.EscapeString(viewportList(d.Run)))

	fmt.Fprintf(&b, "<div class=\"meta\">%d scene(s) in the run, %d scored, %d skipped.</div>\n",
		len(d.Run.Scenes), len(d.Scores), len(d.Summary.Skipped))

	writeSummaryTable(&b, "By category", categoryRows(d.Summary))
	writeSummaryTable(&b, "By split", splitRows(d.Summary))
	writeSkipped(&b, d)

	b.WriteString("<h2>Scenes</h2>\n")
	scores := map[string]*metrics.SceneScore{}
	for _, s := range d.Scores {
		scores[s.SceneID] = s
	}
	ids := make([]string, 0, len(d.Run.Scenes))
	for _, sc := range d.Run.Scenes {
		ids = append(ids, sc.SceneID)
	}
	sort.Strings(ids)
	for _, id := range ids {
		writeScene(&b, d, id, scores[id])
	}

	b.WriteString("</body>\n</html>\n")
	return os.WriteFile(filepath.Join(dir, "report.html"), []byte(b.String()), 0o644)
}

func writeSummaryTable(b *strings.Builder, title string, rows []bucketRow) {
	fmt.Fprintf(b, "<h2>%s</h2>\n<table>\n<tr><th>Bucket</th><th>Scenes</th><th>Recall</th><th>CER</th>"+
		"<th>IoU</th><th>Covered</th><th>Worst residual</th><th>Merges</th><th>Damage px</th>"+
		"<th>Clipped</th><th>Cross-group</th><th>Drift</th><th>Failing</th></tr>\n", title)
	if len(rows) == 0 {
		b.WriteString("<tr><td colspan=\"13\">no scored scenes</td></tr>\n</table>\n")
		return
	}
	for _, r := range rows {
		k := r.bucket
		fmt.Fprintf(b, "<tr><td>%s</td><td>%d</td><td>%.2f</td><td>%.2f</td><td>%.2f</td><td>%.2f</td>"+
			"<td>%.2f</td><td>%d</td><td>%d</td><td>%d</td><td>%d</td><td>%.3f</td><td>%d</td></tr>\n",
			html.EscapeString(strings.Trim(r.name, "*")), k.Scenes, k.MeanRecall, k.MeanCER, k.MeanIoU,
			k.MeanCovered, k.WorstResidual, k.Merges, k.ProtectedHitPx, k.Clipped, k.CrossGroup,
			k.WorstDrift, k.FailingScenes)
	}
	b.WriteString("</table>\n")
}

func writeSkipped(b *strings.Builder, d *Data) {
	if len(d.Summary.Skipped) == 0 {
		return
	}
	b.WriteString("<h2>Skipped</h2>\n<table class=\"skipped\">\n<tr><th>Scene</th><th>Why it was not scored</th></tr>\n")
	for _, s := range d.Summary.Skipped {
		fmt.Fprintf(b, "<tr><td>%s</td><td style=\"text-align:left\">%s</td></tr>\n",
			html.EscapeString(s.SceneID), html.EscapeString(s.Reason))
	}
	b.WriteString("</table>\n")
}

// writeScene is the part a reviewer actually opens: what went in, what came out, where the
// plates landed and - in a sentence, not a number - what is wrong with it.
func writeScene(b *strings.Builder, d *Data, id string, score *metrics.SceneScore) {
	sc := d.Run.Find(id)
	if sc == nil {
		return
	}
	b.WriteString("<div class=\"scene\">\n")
	fmt.Fprintf(b, "<h3>%s</h3>\n", html.EscapeString(id))

	switch {
	case sc.Error != "":
		fmt.Fprintf(b, "<div class=\"verdict fail\">run error</div>\n<ul class=\"reasons\"><li>%s</li></ul>\n",
			html.EscapeString(sc.Error))
	case score == nil:
		reason := "not scored"
		for _, s := range d.Summary.Skipped {
			if s.SceneID == id {
				reason = s.Reason
			}
		}
		fmt.Fprintf(b, "<div class=\"verdict fail\">not scored</div>\n<ul class=\"reasons\"><li>%s</li></ul>\n",
			html.EscapeString(reason))
	case len(score.Failures) > 0:
		b.WriteString("<div class=\"verdict fail\">fail</div>\n<ul class=\"reasons\">\n")
		for _, f := range score.Failures {
			fmt.Fprintf(b, "<li>%s</li>\n", html.EscapeString(f))
		}
		b.WriteString("</ul>\n")
	default:
		b.WriteString("<div class=\"verdict pass\">pass</div>\n")
	}

	if sc.Screenshots.Source != "" || sc.Screenshots.Rendered != "" {
		b.WriteString("<div class=\"pair\">\n")
		writeFigure(b, "source", sc.Screenshots.Source, sc, nil)
		writeFigure(b, "rendered, plate outlines drawn", sc.Screenshots.Rendered, sc,
			sc.PlatesFor(primaryViewportOf(d.Run, sc), metrics.PrimaryStressCase))
		b.WriteString("</div>\n")
	}

	if len(sc.Screenshots.Stress) > 0 {
		b.WriteString("<div class=\"stress\">\n")
		names := make([]string, 0, len(sc.Screenshots.Stress))
		for n := range sc.Screenshots.Stress {
			names = append(names, n)
		}
		sort.Strings(names)
		for _, n := range names {
			if n == metrics.PrimaryStressCase {
				continue
			}
			label := n
			if score != nil {
				if r, ok := score.Stress[n]; ok {
					label = fmt.Sprintf("%s - %d clipped, %d cross-group", n, r.Clipped, r.CrossGroupOverlap)
				}
			}
			fmt.Fprintf(b, "<figure><figcaption>%s</figcaption><span class=\"frame\"><img loading=\"lazy\" src=\"%s\" alt=\"%s\"></span></figure>\n",
				html.EscapeString(label), html.EscapeString(sc.Screenshots.Stress[n]), html.EscapeString(n))
		}
		b.WriteString("</div>\n")
	}

	if score != nil {
		fmt.Fprintf(b, "<div class=\"numbers\">recall %.2f &middot; precision %.2f &middot; CER %.2f &middot; "+
			"IoU %.2f (worst %.2f) &middot; covered %.2f &middot; residual %.2f (halo %.2f) &middot; "+
			"contrast %.0f &middot; damage %d px &middot; drift %.3f &middot; ocr %d ms</div>\n",
			score.Detection.Recall, score.Detection.Precision, score.Text.MeanCER,
			score.Placement.MeanIoU, score.Placement.WorstIoU, score.Covered,
			score.Residual.Residual, score.Residual.Halo, score.Contrast.MinLuma,
			score.Damage.ProtectedHit, score.Placement.Drift, score.Cost.OcrMs)
	}
	b.WriteString("</div>\n")
}

// writeFigure draws one picture with the plate rectangles overlaid as percentage-positioned
// outlines, so "the plate is 30 px too low" is something a reviewer sees rather than computes.
func writeFigure(b *strings.Builder, caption, src string, sc *evidence.Scene, plates []evidence.Plate) {
	if src == "" {
		return
	}
	fmt.Fprintf(b, "<figure><figcaption>%s</figcaption><span class=\"frame\">", html.EscapeString(caption))
	fmt.Fprintf(b, "<img loading=\"lazy\" src=\"%s\" alt=\"%s\">", html.EscapeString(src), html.EscapeString(caption))
	if sc.ImageWidth > 0 && sc.ImageHeight > 0 {
		for _, p := range plates {
			fmt.Fprintf(b, "<span class=\"plate\" style=\"left:%.2f%%;top:%.2f%%;width:%.2f%%;height:%.2f%%\" title=\"%s\"></span>",
				pct(p.Rect.X0, sc.ImageWidth), pct(p.Rect.Y0, sc.ImageHeight),
				pct(p.Rect.Width(), sc.ImageWidth), pct(p.Rect.Height(), sc.ImageHeight),
				html.EscapeString(truncate(p.Text, 80)))
		}
	}
	b.WriteString("</span></figure>\n")
}

func primaryViewportOf(r *evidence.Run, sc *evidence.Scene) string {
	for _, v := range r.Viewports {
		if len(sc.PlatesFor(v.Name, metrics.PrimaryStressCase)) > 0 {
			return v.Name
		}
	}
	return ""
}

func pct(v, total int) float64 {
	if total <= 0 {
		return 0
	}
	return float64(v) / float64(total) * 100
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + ".."
}
