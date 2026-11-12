package report

import (
	"strings"
	"testing"
)

// forgetHidden empties the registry now and when the test ends: it is process-wide state.
func forgetHidden(t *testing.T) {
	t.Helper()
	reset := func() {
		hiddenNames.Lock()
		hiddenNames.names, hiddenNames.re = nil, nil
		hiddenNames.Unlock()
	}
	reset()
	t.Cleanup(reset)
}

func TestHideDocumentNamesInPaths(t *testing.T) {
	forgetHidden(t)
	for _, tc := range []struct{ name, in, want string }{
		{"a spaced name keeps its folder and format", `Opening C:\Docs\My Book.epub with FileDO`, `Opening C:\Docs\<document>.epub with FileDO`},
		{"two paths on one line", `copy D:\In\Dune.pdf -> D:\Out\Emma.fb2 done`, `copy D:\In\<document>.pdf -> D:\Out\<document>.fb2 done`},
		{"slashes", `C:/Library/Part 1/Moby Dick.mobi`, `C:/Library/Part 1/<document>.mobi`},
		{"doubled separators of JSON", `"D:\\Lib\\Dune.epub"`, `"D:\\Lib\\<document>.epub"`},
		{"a file URL", `file:///C:/Lib/Dune%20Messiah.azw3`, `file:///C:/Lib/<document>.azw3`},
		{"a variable root", `%USERPROFILE%\Books\Emma.cbz`, `%USERPROFILE%\Books\<document>.cbz`},
		{"upper-case extension", `C:\In\SCAN.PDF`, `C:\In\<document>.PDF`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := hideDocumentNames(tc.in); got != tc.want {
				t.Errorf("hideDocumentNames(%q)\n got %q\nwant %q", tc.in, got, tc.want)
			}
		})
	}
}

// The app's own names are how the author reads a failure; only the user's documents are hidden.
func TestHideDocumentNamesLeavesTheAppsOwnNames(t *testing.T) {
	forgetHidden(t)
	for _, in := range []string{
		`could not read C:\Out\Book\images\img-012.png`,
		`wrote C:\Out\Book\index.html`,
		`OCR failed on page-0007.png: exit status 1`,
		`C:\Users\x\AppData\Local\doc-html-translate\logs\run-20261030-101500-4242.log`,
		`https://example.com/docs/guide`,
	} {
		if got := hideDocumentNames(in); got != in {
			t.Errorf("hideDocumentNames(%q) = %q, want it unchanged", in, got)
		}
	}
}

func TestHideDocumentNamesHidesTitles(t *testing.T) {
	forgetHidden(t)
	in := "Reading EPUB..\n  Title: War and Peace\n  Chapters: 361\n   Title:\tAnna\n"
	want := "Reading EPUB..\n  Title: <document title>\n  Chapters: 361\n   Title:\t<document title>\n"
	if got := hideDocumentNames(in); got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
}

// A bare name has no path to recognise it by, so the run registers the names it works on.
func TestHideDocumentHidesTheRegisteredNames(t *testing.T) {
	forgetHidden(t)
	HideDocument(`D:\Lib\Dune Messiah.epub`)
	HideFolder(`D:\Out\Dune Messiah (2)`)

	got := hideDocumentNames("Original PDF copied: dune messiah.pdf; Book already extracted: D:\\Out\\Dune Messiah (2); see Dune%20Messiah.epub")
	want := "Original PDF copied: <document>.pdf; Book already extracted: D:\\Out\\<document>; see <document>.epub"
	if got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}
}

// The folder's whole name goes before the file's stem, or "Dune Messiah" would be replaced inside
// "Dune Messiah (2)" and leave " (2)" behind.
func TestHideDocumentReplacesTheLongestNameFirst(t *testing.T) {
	forgetHidden(t)
	HideDocument(`Dune Messiah.epub`)
	HideFolder(`Dune Messiah (2)`)
	if got := hideDocumentNames("into Dune Messiah (2)"); got != "into <document>" {
		t.Errorf("got %q", got)
	}
}

func TestHideDocumentIgnoresNamesThatAreWordsOfTheLog(t *testing.T) {
	forgetHidden(t)
	HideDocument(`C:\In\a.txt`)
	HideFolder(`C:\Out\io`)
	HideFolder(`C:\`)
	if hiddenNames.re != nil {
		t.Fatalf("registered %q", hiddenNames.names)
	}
	if got, in := hideDocumentNames("a b io"), "a b io"; got != in {
		t.Errorf("got %q, want %q", got, in)
	}
}

// Redact runs at write time and again when the archive is built, so a second pass must change
// nothing.
func TestRedactNameHidingIsIdempotent(t *testing.T) {
	forgetHidden(t)
	setHome(t)
	HideDocument(`D:\Lib\Dune.epub`)
	once := Redact("Title: Dune\nOpening D:\\Lib\\Dune.epub, folder D:\\Out\\Dune\n")
	if strings.Contains(once, "Dune") {
		t.Fatalf("a name survived the first pass: %q", once)
	}
	if twice := Redact(once); twice != once {
		t.Errorf("second pass changed the text:\n once %q\ntwice %q", once, twice)
	}
}
