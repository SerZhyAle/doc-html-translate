package report

import (
	"encoding/json"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"doc-html-translate/tools/ocrlab/evidence"
	"doc-html-translate/tools/ocrlab/metrics"
	"doc-html-translate/tools/ocrlab/truth"
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
.reused{margin:2px 0 6px;font-size:12px;color:var(--muted);border-left:3px solid var(--muted);padding-left:8px}
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
button,input,select{font:inherit;padding:5px;margin:4px}input:focus-visible,select:focus-visible{outline:3px solid #075dbb}
.pair figure{overflow:auto}.frame{transform-origin:top left}.frame svg{position:absolute;inset:0;width:100%;height:100%;pointer-events:none}.truth{fill:none;stroke:#0068cc;stroke-width:2}.protected{fill:none;stroke:#b02000;stroke-width:2;stroke-dasharray:6 4}
[hidden]{display:none!important}
`

const reportControls = `<label>Search scene, category, language, split or finding <input id="filter" type="search"></label>
<label>Finding <select id="finding"><option value="">All</option><option value="fail">Hard failure</option><option value="unavailable">Unavailable</option><option value="clear">No hard failure</option></select></label>
<label>Viewport <select id="viewport"><option value="">All</option></select></label><label>Stress <select id="stress"><option value="">All</option></select></label>
<label>Synchronized zoom <input id="zoom" type="range" min="1" max="4" step="0.25" value="1"></label>
<p>Blue outlines: reference reading groups. Red dashed outlines: protected artwork. Pixel diagnostics remain uncalibrated.</p>`

const reportScript = `<script>(function(){const el=id=>document.getElementById(id),cards=[...document.querySelectorAll('.scene')],observations=[...document.querySelectorAll('[data-viewport]')];for(const key of ['viewport','stress']){for(const value of [...new Set(observations.map(n=>n.dataset[key]))].sort()){const o=document.createElement('option');o.value=value;o.textContent=value;el(key).appendChild(o);}}function filter(){const q=el('filter').value.toLowerCase();for(const c of cards)c.hidden=!(c.textContent.toLowerCase().includes(q)&&(!el('finding').value||c.dataset.finding===el('finding').value));for(const o of observations)o.hidden=!!((el('viewport').value&&o.dataset.viewport!==el('viewport').value)||(el('stress').value&&o.dataset.stress!==el('stress').value));}for(const key of ['filter','finding','viewport','stress'])el(key).addEventListener('input',filter);el('zoom').oninput=()=>{for(const f of document.querySelectorAll('.pair .frame'))f.style.width=(Number(el('zoom').value)*100)+'%';};for(const pair of document.querySelectorAll('.pair')){let busy=false;const figs=[...pair.querySelectorAll('figure')];for(const f of figs)f.onscroll=()=>{if(busy)return;busy=true;for(const other of figs){if(other===f)continue;other.scrollLeft=f.scrollLeft;other.scrollTop=f.scrollTop;}busy=false;};}})();</script>`

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
	fmt.Fprintf(&b, "<div class=\"meta\">Evidence: %s</div>\n", html.EscapeString(evidenceLine(d.Run)))
	fmt.Fprintf(&b, "<div class=\"meta\">Not measured: concealment in %d scene(s), painted damage in %d scene(s); they carry no number below.</div>\n",
		d.Summary.Overall.UnmeasuredConcealment, d.Summary.Overall.UnmeasuredDamage)
	fmt.Fprintf(&b, "<p>Scope: %s; procedure: %s. No hard failure is not an acceptance PASS.</p>", html.EscapeString(or(d.Summary.Purpose, "legacy unspecified")), html.EscapeString(or(d.Summary.Procedure, "legacy")))
	for _, issue := range d.Summary.EvidenceIssues {
		fmt.Fprintf(&b, "<p class=\"reasons\">Unavailable evidence: %s</p>", html.EscapeString(issue))
	}
	b.WriteString(reportControls)

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
	sort.SliceStable(ids, func(i, j int) bool {
		priority := func(id string) int {
			if s := scores[id]; s != nil && len(s.Failures) > 0 {
				return 0
			}
			if scores[id] == nil {
				return 1
			}
			return 2
		}
		if priority(ids[i]) != priority(ids[j]) {
			return priority(ids[i]) < priority(ids[j])
		}
		return ids[i] < ids[j]
	})
	for _, id := range ids {
		writeScene(&b, d, id, scores[id], dir)
	}
	b.WriteString(reportScript)

	b.WriteString("</body>\n</html>\n")
	return os.WriteFile(filepath.Join(dir, "report.html"), []byte(b.String()), 0o644)
}

func writeReference(b *strings.Builder, sc *evidence.Scene, a *truth.Annotation) {
	if sc.Screenshots.Source == "" {
		return
	}
	b.WriteString("<details><summary>Reference geometry and transcripts: " + html.EscapeString(sc.SceneID) + "</summary>")
	fmt.Fprintf(b, "<span class=\"frame\"><img loading=\"lazy\" src=\"%s\" alt=\"Reference regions\"><svg viewBox=\"0 0 %d %d\" aria-hidden=\"true\">", html.EscapeString(sc.Screenshots.Source), a.ImageWidth, a.ImageHeight)
	draw := func(r truth.Region, cls string) {
		var points []string
		for _, p := range r.Points {
			points = append(points, fmt.Sprintf("%d,%d", p[0], p[1]))
		}
		if r.Kind == truth.RegionBox && len(r.Points) == 2 {
			p, q := r.Points[0], r.Points[1]
			points = []string{fmt.Sprintf("%d,%d %d,%d %d,%d %d,%d", p[0], p[1], q[0], p[1], q[0], q[1], p[0], q[1])}
		}
		fmt.Fprintf(b, "<polygon class=\"%s\" points=\"%s\"/>", cls, strings.Join(points, " "))
	}
	for _, g := range a.Groups {
		draw(g.Bounds, "truth")
	}
	for _, r := range a.Protected {
		draw(r, "protected")
	}
	b.WriteString("</svg></span>")
	for _, g := range a.Groups {
		fmt.Fprintf(b, "<p>Group %s; language %s; direction %s; order %d: %s</p>", html.EscapeString(g.ID), html.EscapeString(g.Language), html.EscapeString(string(g.Direction)), g.ReadingOrder, html.EscapeString(g.Transcript))
	}
	b.WriteString("</details>")
}

func writeSummaryTable(b *strings.Builder, title string, rows []bucketRow) {
	fmt.Fprintf(b, "<h2>%s</h2>\n<table>\n<tr><th>Bucket</th><th>Scenes</th><th>Recall</th><th>CER</th>"+
		"<th>IoU</th><th>Covered</th><th>Worst residual (measured)</th><th>Merges</th><th>Painted damage px</th>"+
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
func writeScene(b *strings.Builder, d *Data, id string, score *metrics.SceneScore, dir string) {
	sc := d.Run.Find(id)
	if sc == nil {
		return
	}
	finding := "clear"
	if score == nil || sc.Error != "" {
		finding = "unavailable"
	} else if len(score.Failures) > 0 {
		finding = "fail"
	}
	fmt.Fprintf(b, "<div class=\"scene\" data-finding=\"%s\">\n", finding)
	fmt.Fprintf(b, "<h3>%s</h3>\n", html.EscapeString(id))
	b.WriteString(reusedHTML(sc))

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
		if score.Loss != nil {
			fmt.Fprintf(b, "<li>loss point: %s</li>\n", html.EscapeString(score.Loss.String()))
		}
		b.WriteString("</ul>\n")
	default:
		b.WriteString("<div class=\"verdict\">No hard failure observed; acceptance requires the scoped gate</div>\n")
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
				if r, ok := score.Stress[primaryViewportOf(d.Run, sc)+"/"+n]; ok {
					label = fmt.Sprintf("%s - %d clipped, %d cross-group", n, r.Clipped, r.CrossGroupOverlap)
				}
			}
			fmt.Fprintf(b, "<figure><figcaption>%s</figcaption><span class=\"frame\"><img loading=\"lazy\" src=\"%s\" alt=\"%s\"></span></figure>\n",
				html.EscapeString(label), html.EscapeString(sc.Screenshots.Stress[n]), html.EscapeString(n))
		}
		b.WriteString("</div>\n")
	}
	if score != nil {
		fmt.Fprintf(b, "<p>Categories: %s; split: %s</p>", html.EscapeString(fmt.Sprint(score.Categories)), html.EscapeString(string(score.Split)))
	}
	for _, o := range sc.Observations {
		fmt.Fprintf(b, "<details data-viewport=\"%s\" data-stress=\"%s\"><summary>%s / %s</summary><div class=\"pair\">", html.EscapeString(o.Viewport), html.EscapeString(o.StressCase), html.EscapeString(o.Viewport), html.EscapeString(o.StressCase))
		writeFigure(b, "Rendered", o.Rendered, sc, sc.PlatesFor(o.Viewport, o.StressCase))
		writeFigure(b, "Replacement text hidden, concealment retained (diagnostic)", o.Concealed, sc, nil)
		b.WriteString("</div>")
		if score != nil {
			if pixel, ok := score.PixelDiagnostics[o.Viewport+"/"+o.StressCase]; ok {
				data, _ := json.Marshal(pixel)
				fmt.Fprintf(b, "<pre>%s</pre>", html.EscapeString(string(data)))
			}
		}
		b.WriteString("</details>")
	}

	if score != nil {
		fmt.Fprintf(b, "<div class=\"numbers\">recall %.2f &middot; precision %.2f &middot; CER %.2f &middot; "+
			"IoU %.2f (worst %.2f) &middot; covered %.2f &middot; residual %s (halo %s) &middot; "+
			"background contrast %.0f &middot; replacement glyphs: %s &middot; "+
			"painted damage %s px &middot; rectangle intrusion %d px &middot; drift %.3f &middot; ocr %d ms</div>\n",
			score.Detection.Recall, score.Detection.Precision, score.Text.MeanCER,
			score.Placement.MeanIoU, score.Placement.WorstIoU, score.Covered,
			residualCell(score.Residual), haloCell(score.Residual), score.BackgroundContrast.MinLuma,
			html.EscapeString(readabilityCell(score.Readability)),
			damageCell(score.Damage), score.RectangleIntrusion.ProtectedHit, score.Placement.Drift, score.Cost.OcrMs)
	}
	if a, err := truth.Load(truth.FinalPath(filepath.Join(dir, "truth"), id)); err == nil {
		writeReference(b, sc, a)
	}
	if score != nil {
		if score.Text.Measured {
			fmt.Fprintf(b, "<p>Strict CER %.4f (%d errors / %d characters); strict WER %.4f (%d errors / %d words); normalized CER %.4f, WER %.4f; matched transcripts %d, skipped %d. Detection TP %d, FP %d, FN %d.</p>", score.Text.StrictCER, score.Text.StrictCharErrors, score.Text.ReferenceChars, score.Text.StrictWER, score.Text.StrictWordErrors, score.Text.ReferenceWords, score.Text.MeanCER, score.Text.MeanWER, score.Text.Compared, score.Text.Skipped, score.Detection.TP, score.Detection.FP, score.Detection.FN)
		} else {
			b.WriteString("<p>Text accuracy unavailable: no comparable matched transcript.</p>")
		}
		fmt.Fprintf(b, "<p>Conversion/recognition %d ms; browser rendering %d ms; memory: %s, %d bytes (snapshot; process-tree peak unavailable).</p>", score.Cost.OcrMs, score.Cost.RenderMs, html.EscapeString(score.Cost.MemoryKind), score.Cost.MemoryBytes)
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
