package review

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"doc-html-translate/tools/ocrlab/corpus"
	"doc-html-translate/tools/ocrlab/runner"
	"doc-html-translate/tools/ocrlab/truth"
)

func reviewed() *truth.Annotation {
	return &truth.Annotation{SchemaVersion: truth.SchemaVersion, SceneID: "scene", Origin: truth.OriginHuman, ImageWidth: 10, ImageHeight: 10, Ambiguity: truth.AmbiguityClear, Protected: []truth.Region{}, Groups: []truth.Group{{ID: "g", Type: truth.GroupCaption, Transcript: "Hello", ReadingOrder: 1, Bounds: truth.Box("", 1, 1, 8, 8), Lines: []truth.Region{truth.Box("", 1, 1, 8, 8)}, ReplaceArea: truth.Box("", 1, 1, 8, 8)}}, Review: truth.Review{AnnotatedBy: "Alice", CheckedBy: "Bob", Disagreements: []string{"retain the punctuation dispute"}}}
}

func TestImportPreservesTruthAndInvalidatesSignatures(t *testing.T) {
	dir := t.TempDir()
	a := reviewed()
	final := truth.FinalPath(dir, a.SceneID)
	if err := truth.Save(final, a); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(final)
	input := filepath.Join(t.TempDir(), "input.json")
	a.Groups[0].Transcript = "Corrected!"
	if err := truth.Save(input, a); err != nil {
		t.Fatal(err)
	}
	m := &corpus.Manifest{Scenes: []corpus.Scene{{ID: a.SceneID, Width: 10, Height: 10, Split: corpus.SplitHoldout}}}
	dest, err := Import(input, dir, m)
	if err != nil {
		t.Fatal(err)
	}
	draft, err := truth.Load(dest)
	if err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(final)
	if string(before) != string(after) {
		t.Fatal("reviewed truth overwritten")
	}
	if draft.IsTruth() || draft.Review.CheckedBy != "" || draft.Review.AnnotatedBy != "" || len(draft.Review.Disagreements) != 1 {
		t.Fatalf("approvals or disagreement handling: %+v", draft)
	}
	a.Groups[0].Bounds = truth.Box("", 8, 8, 1, 1)
	if err := truth.Save(input, a); err != nil {
		t.Fatal(err)
	}
	if _, err := Import(input, dir, m); err == nil {
		t.Fatal("invalid geometry imported")
	}
}

func TestOfflineEditorBrowserWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("browser check")
	}
	dir := t.TempDir()
	browser, err := runner.FindBrowser(filepath.Join(dir, "profile"))
	if err != nil {
		t.Skipf("browser absent: %v", err)
	}
	defer browser.Close()
	a := reviewed()
	m := &corpus.Manifest{Scenes: []corpus.Scene{{ID: a.SceneID, Width: 10, Height: 10, Split: corpus.SplitHoldout}}}
	page := filepath.Join(dir, "review.html")
	if err := Write(page, dir, m, map[string]*truth.Annotation{a.SceneID: a}); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(page)
	// Exercise the generated page's actual actions, not a second implementation of its validator.
	script := `<script>window.addEventListener('load',()=>{setTimeout(()=>{let errors=[];try{const ed=document.getElementById('editor'),a=JSON.parse(ed.value);if(a.review.annotatedBy||a.review.checkedBy)errors.push('initial signatures retained');a.groups[0].transcript='Corrected!';ed.value=JSON.stringify(a);document.getElementById('apply').click();if(JSON.parse(ed.value).groups[0].transcript!=='Corrected!')errors.push('apply failed');document.getElementById('undo').click();if(JSON.parse(ed.value).groups[0].transcript!=='Hello')errors.push('undo failed');document.getElementById('save').click();const stored=Object.keys(localStorage).find(k=>k.startsWith('ocrlab-review:'));if(!stored)errors.push('draft not saved');a.groups[0].bounds.points=[[8,8],[1,1]];ed.value=JSON.stringify(a);document.getElementById('apply').click();if(!document.getElementById('status').textContent.includes('inverted'))errors.push('invalid geometry accepted');}catch(e){errors.push(e.message)}const out=document.createElement('script');out.id='ocrlab-evidence';out.type='application/json';out.textContent=JSON.stringify({errors});document.body.appendChild(out);},100);});</script>`
	if err := os.WriteFile(page, []byte(strings.Replace(string(data), "</html>", script+"</html>", 1)), 0644); err != nil {
		t.Fatal(err)
	}
	dom, err := browser.DumpDOM(page, "", runner.Viewports[0])
	if err != nil {
		t.Fatal(err)
	}
	marker := `id="ocrlab-evidence"`
	start := strings.Index(dom, marker)
	if start < 0 {
		t.Fatal("workflow produced no result")
	}
	start += strings.Index(dom[start:], ">") + 1
	end := strings.Index(dom[start:], "</script>")
	var result struct {
		Errors []string `json:"errors"`
	}
	if err := json.Unmarshal([]byte(dom[start:start+end]), &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Errors) > 0 {
		t.Fatal(result.Errors)
	}
}
