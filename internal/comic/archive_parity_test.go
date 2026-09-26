package comic

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type archiveParityCases struct {
	Comics []struct {
		File  string   `json:"file"`
		Pages []string `json:"pages"`
		First string   `json:"first"`
		About string   `json:"about"`
	} `json:"comics"`
}

// TestArchiveParityComics reads the shared comic fixtures extension/test/comic.test.mjs reads, so
// both editions list the same pages from the same archive (docs/PARITY.md, "Comic archive page
// order and entry filter").
func TestArchiveParityComics(t *testing.T) {
	fixtures := filepath.Join("..", "..", "tests", "testdata", "archive-parity")
	data, err := os.ReadFile(filepath.Join(fixtures, "cases.json"))
	if err != nil {
		t.Fatal(err)
	}
	var cases archiveParityCases
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	for _, c := range cases.Comics {
		t.Run(c.File, func(t *testing.T) {
			path := filepath.Join(fixtures, c.File)
			var arc *archive
			switch kind := containerKind(path, strings.ToLower(filepath.Ext(path))); kind {
			case containerZip:
				arc, err = openCBZ(path)
			case containerTar:
				arc, err = openCBT(path)
			default:
				t.Fatalf("container %s: the fixtures hold only ZIP and TAR", kind)
			}
			if err != nil {
				t.Fatal(err)
			}
			defer arc.close()
			pages, err := selectPages(arc)
			if err != nil {
				t.Fatal(err)
			}
			var names []string
			for _, p := range pages {
				names = append(names, p.name)
			}
			if !reflect.DeepEqual(names, c.Pages) {
				t.Fatalf("%s: pages %q, want %q", c.About, names, c.Pages)
			}
			rc, err := pages[0].open()
			if err != nil {
				t.Fatal(err)
			}
			first, err := io.ReadAll(rc)
			_ = rc.Close()
			if err != nil || string(first) != c.First {
				t.Errorf("%s: first page %q (err %v), want %q", c.About, first, err, c.First)
			}
		})
	}
}
