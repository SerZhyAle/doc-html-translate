package ocr

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// TestOrderColumnsSharedCases runs the column-order fixture extension/test/ocr-cluster.test.mjs
// runs: the floor a pass hands orderColumns decides which lines may form a column (audit finding
// B64; tests/ocr_floor_parity_test.go holds each pass to handing over its own floor).
func TestOrderColumnsSharedCases(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "tests", "testdata", "ocr_column_order_cases.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fx struct {
		Runs []struct {
			Text string  `json:"text"`
			Conf float64 `json:"conf"`
			BBox struct {
				X0, Y0, X1, Y1 int
			} `json:"bbox"`
		} `json:"runs"`
		Orders []struct {
			Floor float64  `json:"floor"`
			Want  []string `json:"want"`
		} `json:"orders"`
	}
	if err := json.Unmarshal(data, &fx); err != nil {
		t.Fatal(err)
	}
	for _, o := range fx.Orders {
		var runs []*ocrLine
		for _, r := range fx.Runs {
			l := &ocrLine{x0: r.BBox.X0, y0: r.BBox.Y0, x1: r.BBox.X1, y1: r.BBox.Y1, confSum: r.Conf, confN: 1}
			l.text.WriteString(r.Text)
			runs = append(runs, l)
		}
		var got []string
		for _, l := range orderColumns(runs, o.Floor) {
			got = append(got, l.text.String())
		}
		if !reflect.DeepEqual(got, o.Want) {
			t.Errorf("floor %v: order %q, want %q", o.Floor, got, o.Want)
		}
	}
}
