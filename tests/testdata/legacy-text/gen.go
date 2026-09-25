//go:build ignore

// gen writes the shared legacy-text fixture set: inputs in the encodings real files arrive in,
// and the text both editions must extract from them. Run from the repository root:
//
//	go run tests/testdata/legacy-text/gen.go
//
// The expected texts are written here by hand, never computed by the readers under test.
// Format of an expected file: paragraphs separated by a blank line, verse lines of one stanza
// by a single newline. tests/legacy_text_test.go and extension/test/legacy-text.test.mjs read
// the same files through cases.json.
package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf16"

	"golang.org/x/text/encoding/charmap"
)

type fixture struct {
	Name     string `json:"name"`
	Format   string `json:"format"`
	Input    string `json:"input"`
	Expected string `json:"expected"`
	About    string `json:"about"`
}

const dir = "tests/testdata/legacy-text"

func main() {
	var cases []fixture
	add := func(name, format, ext, about string, input []byte, expected string) {
		in := name + ext
		exp := name + ".expected.txt"
		must(os.WriteFile(filepath.Join(dir, in), input, 0o644))
		must(os.WriteFile(filepath.Join(dir, exp), []byte(expected+"\n"), 0o644))
		cases = append(cases, fixture{Name: name, Format: format, Input: in, Expected: exp, About: about})
	}

	add("wordpad-ru", "rtf", ".rtf",
		"English-Windows WordPad typing Russian: \\ansicpg1252 with a \\fcharset204 font, font table, colour table, generator and a picture",
		[]byte(`{\rtf1\ansi\ansicpg1252\deff0\nouicompat\deflang1033{\fonttbl{\f0\fnil\fcharset0 Calibri;}{\f1\fnil\fcharset204 Calibri;}}`+"\r\n"+
			`{\colortbl ;\red255\green0\blue0;}`+"\r\n"+
			`{\*\generator Riched20 10.0.19041}\viewkind4\uc1 `+"\r\n"+
			`\pard\sa200\sl276\slmult1\f1\fs22\lang1049 `+hex1251("Привет, мир! Это документ WordPad.")+`\par`+"\r\n"+
			hex1251("Вторая строка с буквами ёЁ и ъЪ.")+`\f0\lang1033  English words.\par`+"\r\n"+
			`{\pict{\*\picprop}\wmetafile8\picw26\pich26\picwgoal15\pichgoal15 `+"\r\n"+
			`0100090000035000000000002700000000000400000003010800050000000b0200000000}\par`+"\r\n"+
			"}\r\n"),
		"Привет, мир! Это документ WordPad.\n\nВторая строка с буквами ёЁ и ъЪ. English words.")

	binData := "}{\\}{\\"
	euro := fmt.Sprintf(`\u%d`, 0x20AC)
	add("libreoffice-uc", "rtf", ".rtf",
		"LibreOffice style: \\uN with \\'XX fallback under \\uc1 and \\uc2, a surrogate pair, a \\bin picture with braces in its data, starred destinations",
		[]byte(`{\rtf1\ansi\deff4\adeflang1025`+"\n"+
			`{\fonttbl{\f0\froman\fprq2\fcharset0 Times New Roman;}{\f4\froman\fprq2\fcharset204 Liberation Serif{\*\falt Times New Roman};}}`+"\n"+
			`{\colortbl;\red0\green0\blue0;\red0\green0\blue255;}`+"\n"+
			`{\stylesheet{\s0\snext0\loch\f4\fs24\lang1049 Normal;}}`+"\n"+
			`{\*\generator LibreOffice/7.6.4.1$Linux_X86_64 LibreOffice_project/}`+"\n"+
			`{\info{\creatim\yr2024\mo3\dy1\hr10\min0}{\revtim\yr2024}}`+"\n"+
			`{\*\userprops{\propname AppVersion}\proptype30{\staticval 16.0}}`+"\n"+
			`\deftab709`+"\n"+
			`\pard\plain \s0\loch\f4\fs24\lang1049{\rtlch \ltrch\loch\uc1`+uniFallback("Привет из LibreOffice «ура»")+"}\n"+
			`\par \pard\plain \s0\loch\f4\fs24{\uc2 `+euro+`\'80\'80 = `+euro+`\'88\'88} {\loch\f4 \u-10179\'3f\u-8704\'3f}\par`+"\n"+
			`{\*\shppict{\pict\pngblip\picw1\pich1\bin6 `+binData+`}}{\nonshppict{\pict\wmetafile8 0102}}`+"\n"+
			`\pard\plain \loch\f4\uc1`+uniFallback("Конец.")+`\par`+"\n"+
			"}\n"),
		"Привет из LibreOffice «ура»\n\n€ = € \U0001F600\n\nКонец.")

	add("cp1252", "rtf", ".rtf",
		"Windows-1252 RTF: \\'XX accents, a raw high byte, control symbols, symbol words, header and footer destinations",
		append(append([]byte(`{\rtf1\ansi\ansicpg1252\deff0{\fonttbl{\f0\fswiss\fcharset0 Arial;}}`+"\r\n"+
			`{\*\generator Msftedit 5.41.21.2510;}\viewkind4\uc1\pard\lang1036\f0\fs20 Caf\'e9, cr\'e8me br\'fbl\'e9e\~et na\'efvet\'e9.\par`+"\r\n"+
			`Z`), 0xFC),
			[]byte(`rich \ldblquote quoted\rdblquote  exc\-ept non\_breaking.\par`+"\r\n"+
				`{\header Page header text}{\footer Footer text}Fin.\par`+"\r\n"+
				"}\r\n")...),
		"Café, crème brûlée\U000000A0et naïveté.\n\nZürich \U0000201Cquoted\U0000201D except non\U00002011breaking.\n\nFin.")

	add("cp1251-poem", "fb2", ".fb2",
		"FB2 declared windows-1251 with an epigraph, a poem (stanzas, text-author), a subtitle and a table",
		enc1251(`<?xml version="1.0" encoding="windows-1251"?>
<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0" xmlns:l="http://www.w3.org/1999/xlink">
<description><title-info><genre>poetry</genre><author><first-name>Александр</first-name><last-name>Пушкин</last-name></author><book-title>Стихи</book-title><lang>ru</lang></title-info></description>
<body>
<section>
<title><p>Зимнее утро</p></title>
<epigraph><p>Эпиграф к стихам.</p><text-author>Автор эпиграфа</text-author></epigraph>
<poem>
<stanza>
<v>Мороз и солнце; день чудесный!</v>
<v>Ещё ты дремлешь, друг прелестный -</v>
</stanza>
<stanza>
<v>Пора, красавица, проснись:</v>
<v>Открой сомкнуты негой взоры</v>
</stanza>
<text-author>А. С. Пушкин</text-author>
</poem>
<subtitle>* * *</subtitle>
<p>Обычный абзац после <emphasis>стихотворения</emphasis>.</p>
<table><tr><td>Ячейка один</td><td>Ячейка два</td></tr></table>
</section>
</body>
</FictionBook>
`),
		"Зимнее утро\n\nЭпиграф к стихам.\n\nАвтор эпиграфа\n\n"+
			"Мороз и солнце; день чудесный!\nЕщё ты дремлешь, друг прелестный -\n\n"+
			"Пора, красавица, проснись:\nОткрой сомкнуты негой взоры\n\n"+
			"А. С. Пушкин\n\n* * *\n\nОбычный абзац после стихотворения.\n\nЯчейка один\n\nЯчейка два")

	truncated := []byte("Первая строка текста.\n\nВторая строка, у которой обрезан последний байт: конец")
	add("utf8-truncated", "txt", ".txt",
		"Russian UTF-8 whose last byte was cut off",
		truncated[:len(truncated)-1],
		"Первая строка текста.\n\nВторая строка, у которой обрезан последний байт: коне\U0000FFFD")

	add("utf16le-nobom", "txt", ".txt",
		"UTF-16LE without a byte-order mark",
		utf16le("Hello, world.\r\nПривет, мир.\r\nTroisième ligne.\r\n"),
		"Hello, world.\n\nПривет, мир.\n\nTroisième ligne.")

	french := "Élément très cher, à côté de l'hôtel où nous étions cet été.\r\n\r\nNoël à Besançon, déjà vu par un garçon."
	add("latin1-fr", "txt", ".txt",
		"French in Latin-1 (read as windows-1252, the Western fallback)",
		enc(charmap.ISO8859_1, french),
		strings.ReplaceAll(french, "\r\n", "\n"))

	manifest, err := json.MarshalIndent(cases, "", "  ")
	must(err)
	must(os.WriteFile(filepath.Join(dir, "cases.json"), append(manifest, '\n'), 0o644))
	fmt.Printf("wrote %d fixtures to %s\n", len(cases), dir)
}

// hex1251 writes text as RTF \'XX escapes in Windows-1251, ASCII left as is.
func hex1251(s string) string {
	var sb strings.Builder
	for _, b := range enc(charmap.Windows1251, s) {
		if b < 0x80 {
			sb.WriteByte(b)
		} else {
			fmt.Fprintf(&sb, `\'%02x`, b)
		}
	}
	return sb.String()
}

// uniFallback writes each non-ASCII letter as \uN followed by its Windows-1251 \'XX fallback,
// the way LibreOffice does under \uc1.
func uniFallback(s string) string {
	var sb strings.Builder
	for _, r := range s {
		if r < 0x80 {
			sb.WriteRune(r)
			continue
		}
		fb := enc(charmap.Windows1251, string(r))
		fmt.Fprintf(&sb, `\u%d\'%02x`, r, fb[0])
	}
	return sb.String()
}

func enc1251(s string) []byte { return enc(charmap.Windows1251, s) }

func enc(cm *charmap.Charmap, s string) []byte {
	b, err := cm.NewEncoder().Bytes([]byte(s))
	must(err)
	return b
}

func utf16le(s string) []byte {
	units := utf16.Encode([]rune(s))
	b := make([]byte, 2*len(units))
	for i, u := range units {
		binary.LittleEndian.PutUint16(b[2*i:], u)
	}
	return b
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
