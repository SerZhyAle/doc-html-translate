package pdf

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The same fixture drives extension/test/reflow.test.mjs, so the two editions classify the same
// blocks alike (docs/PARITY.md, "PDF reflow heuristics"; ticket 73).
func TestClassifyBlockSharedCases(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "tests", "testdata", "pdf_heading_cases.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Cases []struct {
			Text          string `json:"text"`
			LeadingSpaces int    `json:"leadingSpaces"`
			PageMargin    int    `json:"pageMargin"`
			Tag           string `json:"tag"`
			Why           string `json:"why"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	if len(fixture.Cases) == 0 {
		t.Fatal("no cases in the shared fixture")
	}
	for _, c := range fixture.Cases {
		if got := classifyBlock(c.Text, c.LeadingSpaces, c.PageMargin); got != c.Tag {
			t.Errorf("classifyBlock(%q, %d, %d) = %q, want %q (%s)", c.Text, c.LeadingSpaces, c.PageMargin, got, c.Tag, c.Why)
		}
	}
}

func TestScriptWords(t *testing.T) {
	cases := []struct {
		text string
		want int
	}{
		{"", 0},
		{"   ", 0},
		{"hello world", 2},
		{"人类", 1}, // ceil(2/2)
		{"个", 1},  // ceil(1/2)
		{"个健康的环境中充分发挥自己的潜能。", 9},   // ceil(16/2) + the full stop
		{"2015 年 9 月 25 日大会决议", 8}, // digits split the runs
		{"2015年9月可持续发展议程与宣言", 9},   // one token, runs counted on their own
		{"2015年9月可持续发展议程", 7},      // the same shape one run shorter
		{"はじめに", 2},                // hiragana
		{"カタカナ", 2},                // katakana
		{"제1조 ① 대한민국은 민주공화국이다", 4}, // Hangul is one word per token, never halved
		{"3/32   15-16301 (C)", 3}, // a footer keeps its three tokens
		{"ГЛАВА ПЕРВАЯ", 2},        // Cyrillic unchanged
		{"We are determined to end poverty and hunger", 8},
	}
	for _, c := range cases {
		if got := scriptWords(c.text); got != c.want {
			t.Errorf("scriptWords(%q) = %d, want %d", c.text, got, c.want)
		}
	}
}

func TestPageLeftMargin(t *testing.T) {
	cases := []struct {
		name string
		page string
		want int
	}{
		{"flush page", "First line\nSecond line\n", 0},
		{"header pins the baseline to column 0", "变革我们的世界：2030 年可持续发展议程\n            人类\n                  我们决心消除\n", 0},
		{"page inset to twelve", "            Chapter One\n            Body text follows here.\n", 12},
		{"blank and whitespace-only lines do not set the margin", "\n            Only line\n\n     \n", 12},
		{"tabs are not spaces", "\tTabbed line\n            Indented line\n", 0},
		{"empty page", "", 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := pageLeftMargin(c.page); got != c.want {
				t.Errorf("pageLeftMargin(%q) = %d, want %d", c.page, got, c.want)
			}
		})
	}
}

// TestParsePDFLayoutPage_CJKPage mirrors page 2 of the corpus case zh-textpdf-un-a-res-70-1
// (ticket 73): a running header at column 0, section headings at column 12, body paragraphs
// whose first lines sit at column 18. Every block used to be "centred" and every block short
// (a Han paragraph is one whitespace word), so the whole page became h2 headings.
func TestParsePDFLayoutPage_CJKPage(t *testing.T) {
	page := strings.Join([]string{
		"变革我们的世界：2030 年可持续发展议程                                     A/RES/70/1",
		"",
		"            人类",
		"",
		"                  我们决心消除一切形式和表现的贫困与饥饿，让所有人平等和有尊严地在一",
		"            个健康的环境中充分发挥自己的潜能。",
		"",
		"            地球",
		"",
		"                  我们决心阻止地球的退化，包括以可持续的方式进行消费和生产，管理地球",
		"            的自然资源，在气候变化问题上立即采取行动，使地球能够满足今世后代的需求。",
		"",
		"2/32                                                 15-16301 (C)",
	}, "\n")

	items := parsePDFLayoutPage(page)
	want := []pageItem{
		{"变革我们的世界：2030 年可持续发展议程                                     A/RES/70/1", "p"},
		{"人类", "h2"},
		{"我们决心消除一切形式和表现的贫困与饥饿，让所有人平等和有尊严地在一 个健康的环境中充分发挥自己的潜能。", "p"},
		{"地球", "h2"},
		{"我们决心阻止地球的退化，包括以可持续的方式进行消费和生产，管理地球 的自然资源，在气候变化问题上立即采取行动，使地球能够满足今世后代的需求。", "p"},
		{"2/32                                                 15-16301 (C)", "h2"},
	}
	if len(items) != len(want) {
		t.Fatalf("got %d items, want %d:\n%#v", len(items), len(want), items)
	}
	for i, w := range want {
		if items[i] != w {
			t.Errorf("item %d = %+v, want %+v", i, items[i], w)
		}
	}
}
