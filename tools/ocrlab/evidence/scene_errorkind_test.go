package evidence

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Evidence written before the failure classification existed has neither field and still loads,
// and a scene without an error does not grow them.
func TestSceneErrorKindIsAdditive(t *testing.T) {
	old := `{"schemaVersion":2,"runId":"r","edition":"desktop","scenes":[` +
		`{"sceneId":"a","error":"convert: cannot read input file"},{"sceneId":"b"}]}`
	path := filepath.Join(t.TempDir(), "evidence.json")
	if err := os.WriteFile(path, []byte(old), 0o644); err != nil {
		t.Fatal(err)
	}
	r, err := LoadRun(path)
	if err != nil {
		t.Fatalf("LoadRun of evidence without errorKind: %v", err)
	}
	if got := r.Find("a"); got == nil || got.Error == "" || got.ErrorKind != "" || got.PathChars != 0 {
		t.Errorf("old failed scene = %+v, want the error kept and the new fields empty", got)
	}

	ok, err := json.Marshal(Scene{SceneID: "b"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(ok), "errorKind") || strings.Contains(string(ok), "pathChars") {
		t.Errorf("a clean scene serializes the failure fields: %s", ok)
	}

	failed, err := json.Marshal(Scene{SceneID: "a", Error: "x", ErrorKind: ErrEnginePathIncompatible, PathChars: 301})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"errorKind":"engine-path-incompatible"`, `"pathChars":301`} {
		if !strings.Contains(string(failed), want) {
			t.Errorf("failed scene JSON %s lacks %s", failed, want)
		}
	}
}
