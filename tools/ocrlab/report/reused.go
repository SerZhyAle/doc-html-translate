package report

import (
	"fmt"
	"html"
	"sort"
	"strings"

	"doc-html-translate/tools/ocrlab/evidence"
)

// A reused scene is a record copied from an earlier complete run, not a measurement of this one.
// The report states that wherever the scene appears, so a reader never has to infer it from the
// timings or the paths.

// reuseCounts splits a run's scenes into those it collected and those it copied.
func reuseCounts(r *evidence.Run) (collected, reused int) {
	for i := range r.Scenes {
		if r.Scenes[i].ReusedFrom != nil {
			reused++
		} else {
			collected++
		}
	}
	return collected, reused
}

func evidenceLine(r *evidence.Run) string {
	collected, reused := reuseCounts(r)
	return fmt.Sprintf("%d scene(s) collected in this run, %d reused from earlier runs.", collected, reused)
}

// reusedNote is the one-sentence provenance of a reused scene.
func reusedNote(f *evidence.ReusedFrom) string {
	return fmt.Sprintf("reused from %s, collected %s, copied %s (files checked by %s); not a new measurement",
		f.Bundle, f.CollectedAt, f.ReusedAt, f.Integrity)
}

// writeReusedMarkdown lists every reused scene with its origin. Nothing is written for a run that
// collected everything.
func writeReusedMarkdown(b *strings.Builder, r *evidence.Run) {
	var ids []string
	for i := range r.Scenes {
		if r.Scenes[i].ReusedFrom != nil {
			ids = append(ids, r.Scenes[i].SceneID)
		}
	}
	if len(ids) == 0 {
		return
	}
	sort.Strings(ids)
	b.WriteString("| Reused scene | From | Collected | Copied | Files checked by |\n|---|---|---|---|---|\n")
	for _, id := range ids {
		f := r.Find(id).ReusedFrom
		fmt.Fprintf(b, "| `%s` | `%s` | %s | %s | %s |\n", id, f.Bundle, f.CollectedAt, f.ReusedAt, f.Integrity)
	}
	b.WriteString("\n")
}

func reusedHTML(sc *evidence.Scene) string {
	if sc.ReusedFrom == nil {
		return ""
	}
	return "<p class=\"reused\">" + html.EscapeString(reusedNote(sc.ReusedFrom)) + "</p>\n"
}
