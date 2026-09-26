package htmlproc

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ExtractTexts trims every Unicode space, so ReplaceTexts must put back the same
// set: a no-break space before an inline element ("Mr.&nbsp;<i>Smith</i>") is
// what keeps the translated word apart from the next one.
func TestReplaceTextsKeepsNoBreakSpaceEdges(t *testing.T) {
	p := filepath.Join(t.TempDir(), "nbsp.html")
	src := "<html><body><p>Chapter <b>1</b></p><p><i>Mr.</i> Smith</p></body></html>"
	if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	segs, doc, err := ExtractTexts(p, nil)
	if err != nil {
		t.Fatal(err)
	}
	var tr []string
	for _, s := range segs {
		tr = append(tr, "<"+s.Text+">")
	}
	ReplaceTexts(segs, tr)
	if err := RenderToFile(doc, p); err != nil {
		t.Fatal(err)
	}
	out, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"&lt;Chapter&gt; <b>", "</i> &lt;Smith&gt;"} {
		if !strings.Contains(string(out), want) {
			t.Errorf("output lost a space edge, want %q in:\n%s", want, out)
		}
	}
}
