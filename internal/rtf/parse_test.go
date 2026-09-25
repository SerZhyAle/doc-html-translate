package rtf

import (
	"strings"
	"testing"
	"time"
)

func TestStripRTF(t *testing.T) {
	cases := []struct {
		name, in, want string
	}{
		{"unicode escape with fallback", "{\\rtf1\\uc1\\u1055?\\u1088?}", "Пр"},
		{"unicode escape with hex fallback", "{\\rtf1\\ansicpg1251\\uc1\\u1055\\'cf\\u1088\\'f0}", "Пр"},
		{"uc2 skips two fallback bytes", "{\\rtf1{\\uc2\\u1055\\'cf\\'cf}x}", "Пx"},
		{"uc scoped to its group", "{\\rtf1{\\uc0\\u1055}\\u1088?}", "Пр"},
		{"negative value", `{\rtf1\u-3913?}`, "\U0000F0B7"},
		{"surrogate pair", `{\rtf1\u-10179?\u-8704?}`, "\U0001F600"},
		{"lone surrogate", `{\rtf1\u-10179?x}`, "\U0000FFFDx"},
		{"font table skipped", `{\rtf1{\fonttbl{\f0\fnil Calibri;}}Body}`, "Body"},
		{"colour table and info skipped", `{\rtf1{\colortbl;\red0\green0\blue0;}{\info{\author Me}}Body}`, "Body"},
		{"starred destination skipped", `{\rtf1{\*\generator Riched20;}Body}`, "Body"},
		{"picture skipped", `{\rtf1{\pict\wmetafile8 0102abcdef}Body}`, "Body"},
		{"bin data skipped even with braces", "{\\rtf1{\\pict\\bin4 {}\\}}Body}", "Body"},
		{"control symbols", `{\rtf1 a\~b\_c\-d\{\}\\}`, "a\U000000A0b\U00002011cd{}\\"},
		{"symbol words", `{\rtf1\ldblquote x\rdblquote\emdash}`, "\U0000201Cx\U0000201D\U00002014"},
		{"paragraph and tab", `{\rtf1 a\par b\tab c}`, "a\n\nb\tc"},
		{"source line breaks are not text", "{\\rtf1 ab\r\ncd}", "abcd"},
		{"default code page is 1252", `{\rtf1\ansi caf\'e9}`, "café"},
		{"ansicpg honoured", `{\rtf1\ansi\ansicpg1251 \'cf\'f0}`, "Пр"},
		{"font charset beats ansicpg", `{\rtf1\ansi\ansicpg1252{\fonttbl{\f0\fcharset0 A;}{\f1\fcharset204 B;}}\f1\'cf\f0\'e9}`, "Пé"},
		{"font change inside a group is scoped", `{\rtf1{\fonttbl{\f1\fcharset204 B;}}{\f1\'cf}\'cf}`, "ПÏ"},
		{"raw high byte uses the code page", "{\\rtf1\\ansicpg1252 Z\xfcrich}", "Zürich"},
		{"double-byte code page", `{\rtf1\ansicpg932 \'82\'a0}`, "あ"},
		{"field instruction hidden, result kept", `{\rtf1{\field{\*\fldinst HYPERLINK "x"}{\fldrslt link}}}`, "link"},
		{"unknown code page falls back", `{\rtf1\ansicpg437 \'e9}`, "é"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := stripRTF([]byte(c.in)); got != c.want {
				t.Errorf("stripRTF(%q)\n got %q\nwant %q", c.in, got, c.want)
			}
		})
	}
}

// largeRTF builds a document of n paragraphs that is mostly \'XX escapes, the worst case for
// the code-page path.
func largeRTF(n int) []byte {
	var sb strings.Builder
	sb.WriteString(`{\rtf1\ansi\ansicpg1251{\fonttbl{\f0\fcharset204 Times;}}\f0 `)
	line := "\\'cf\\'f0\\'e8\\'e2\\'e5\\'f2, \\'ec\\'e8\\'f0! \\u1055\\'cf\\u1088\\'f0\\par\r\n"
	for i := 0; i < n; i++ {
		sb.WriteString(line)
	}
	sb.WriteString("}")
	return []byte(sb.String())
}

// A decoder used to be created per byte; a multi-megabyte Russian RTF must stay fast.
func TestStripRTFLargeInput(t *testing.T) {
	data := largeRTF(50000) // about 5 MB
	start := time.Now()
	out := stripRTF(data)
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("stripRTF took %v on %d bytes", elapsed, len(data))
	}
	if got := strings.Count(out, "Привет, мир! Пр"); got != 50000 {
		t.Errorf("decoded %d paragraphs, want 50000", got)
	}
}

func BenchmarkStripRTF(b *testing.B) {
	data := largeRTF(5000)
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stripRTF(data)
	}
}
