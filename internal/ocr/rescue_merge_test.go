package ocr

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
)

func mergeLine(text string, y int) Block {
	return Block{Text: text, X0: 100, Y0: y, X1: 300, Y1: y + 20, LineH: 20, TypeH: 18,
		Conf: 95, tokens: 2, Lines: []LineBox{{100, y, 300, y + 20}},
		LineContent: []LineContent{{Text: text, TypeH: 18, Conf: 95, Tokens: 2}}}
}

func mergeParagraph(texts ...string) Block {
	b := Block{}
	for i, text := range texts {
		l := mergeLine(text, 100+i*40)
		b.Lines = append(b.Lines, l.Lines...)
		b.LineContent = append(b.LineContent, l.LineContent...)
	}
	return linePart(b, 0, len(texts))
}

func TestRescueMergeRetainsUncoveredLines(t *testing.T) {
	full := mergeParagraph("first line", "second line", "third line", "fourth line", "fifth line")
	for _, tc := range []struct {
		name string
		kept []Block
		want []string
		drop []string
	}{
		{"missing-tail", []Block{linePart(full, 0, 3)}, []string{"first line second line third line", "fourth line fifth line"}, []string{"first line second line third line"}},
		{"interior-gap", []Block{linePart(full, 0, 1), linePart(full, 2, 5)}, []string{"first line", "second line", "third line fourth line fifth line"}, []string{"first line", "third line fourth line fifth line"}},
		{"full-duplicate", []Block{full}, []string{full.Text}, []string{full.Text}},
		{"unread-prefix", []Block{linePart(full, 2, 5)}, []string{"first line second line", "third line fourth line fifth line"}, []string{"third line fourth line fifth line"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, dropped := mergeScreenBlocks(tc.kept, []Block{full})
			texts := func(bs []Block) []string {
				out := []string{}
				for _, b := range bs {
					out = append(out, b.Text)
				}
				return out
			}
			if !reflect.DeepEqual(texts(got), tc.want) || !reflect.DeepEqual(texts(dropped), tc.drop) {
				t.Fatalf("accepted %q, rejected %q", texts(got), texts(dropped))
			}
			for _, old := range tc.kept {
				found := false
				for _, b := range got {
					if reflect.DeepEqual(old, b) {
						found = true
					}
				}
				if !found {
					t.Fatalf("accepted block was changed: %+v", old)
				}
			}
			for _, b := range got {
				if !hasLineContent(b) {
					t.Fatal("lost line association")
				}
			}
			// The catalog artifact is generated from actual results, never hand-edited.
			if dir := os.Getenv("DOCHT_TEST_MERGE_VECTORS"); dir != "" {
				row := map[string]any{"case": tc.name, "kept": tc.kept, "found": []Block{full}, "accepted": got, "rejected": dropped}
				data, err := json.MarshalIndent(row, "", "  ")
				if err != nil {
					t.Fatal(err)
				}
				if err = os.WriteFile(dir+"/"+tc.name+".json", append(data, '\n'), 0600); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestRescueMergeFallbackAndDisjointColumns(t *testing.T) {
	full := mergeParagraph("first line", "second line")
	partial := linePart(full, 0, 1)
	full.LineContent = nil
	got, rejected := mergeScreenBlocks([]Block{partial}, []Block{full})
	if len(got) != 1 || len(rejected) != 1 {
		t.Fatal("missing metadata must keep the conservative fallback")
	}
	// Existing engine column order is preserved even when a later column starts higher.
	left, right := mergeLine("left column", 300), mergeLine("right column", 100)
	right.X0, right.X1 = 500, 700
	added := mergeLine("left prefix", 200)
	got, _ = mergeScreenBlocks([]Block{left, right}, []Block{added})
	if got[0].Text != added.Text || got[1].Text != left.Text || got[2].Text != right.Text {
		t.Fatalf("columns reordered: %+v", got)
	}
}

func TestRescueMergeLineDuplicatesWithLowBlockCoverage(t *testing.T) {
	full := mergeParagraph("first line", "second line", "third line", "fourth line", "fifth line")
	got, rejected := mergeScreenBlocks([]Block{linePart(full, 0, 1)}, []Block{full})
	if len(got) != 2 || len(rejected) != 1 || strings.Contains(got[1].Text, "first") {
		t.Fatal("block whitespace must not admit a duplicate line")
	}
	res := Result{Width: 800, Height: 800, Blocks: []Block{full}}
	scaleDown(&res, 2)
	if res.Blocks[0].LineContent[0].TypeH != 9 || res.Blocks[0].Lines[0].X0 != 50 || res.Blocks[0].LineContent[0].Text != "first line" {
		t.Fatal("line association did not survive coordinate scaling")
	}
}

func TestMaskExtentAndSupportedAlignment(t *testing.T) {
	b := mergeParagraph("first line", "second line")
	b.Lines[1].X1 = 200
	if !leftAligned(b) {
		t.Fatal("ragged right with a shared left edge should align left")
	}
	b.Lines[1].X0 = 150
	if leftAligned(b) {
		t.Fatal("centered text should stay centered")
	}
	p := maskPlateBounds(b, ModeMask, 800, 800)
	if p.X0 >= b.X0 || p.Y0 >= b.Y0 || p.X1 <= b.X1 || p.Y1 <= b.Y1 {
		t.Fatal("mask padding clipped")
	}
	if !reflect.DeepEqual(p.Lines, b.Lines) {
		t.Fatal("concealment changed source line coordinates")
	}
	edge := Block{X0: 0, Y0: 0, X1: 800, Y1: 800, LineH: 20, Lines: b.Lines}
	if p = maskPlateBounds(edge, ModeMask, 800, 800); p.X0 != 0 || p.Y0 != 0 || p.X1 != 800 || p.Y1 != 800 {
		t.Fatal("padding escaped image")
	}
}

func TestRescueCorroboratesTighterGeometryWithoutRewritingText(t *testing.T) {
	good := mergeLine("same transcript", 100)
	bad := good
	bad.X0 = 40
	bad.Lines = []LineBox{{40, 100, 300, 120}}
	bad.LineContent = []LineContent{{Text: good.Text, TypeH: 18, Conf: 80, Tokens: 2}}
	got, _ := mergeScreenBlocks([]Block{bad}, []Block{good})
	if len(got) != 1 || got[0].X0 != 100 || got[0].Text != bad.Text || bad.Lines[0].X0 != 40 {
		t.Fatal("same-text rescue did not repair geometry, or mutated accepted input")
	}
	good.LineContent[0].Text = "different transcript"
	got, _ = mergeScreenBlocks([]Block{bad}, []Block{good})
	if got[0].X0 != 40 {
		t.Fatal("different text cannot corroborate a crop")
	}
}
