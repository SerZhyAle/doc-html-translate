package pipeline

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"doc-html-translate/internal/config"
	"doc-html-translate/internal/epub"
	"doc-html-translate/internal/htmlgen"
	"doc-html-translate/internal/outputpath"
	"doc-html-translate/internal/translator"
)

// The reference loop preserves the previous eager-loading path. Compare the bytes written by
// both paths, including a page without text and the translated title and TOC labels.
func TestStreamingTranslationMatchesEagerPages(t *testing.T) {
	book := epub.Book{
		Title: "Book title",
		TOC:   []epub.TOCEntry{{Title: "Chapter one"}},
		Manifest: []epub.ManifestItem{
			{ID: "one", Href: "one.html", MediaType: "text/html"},
			{ID: "empty", Href: "empty.html", MediaType: "text/html"},
			{ID: "two", Href: "two.html", MediaType: "text/html"},
		},
	}
	files := map[string]string{
		"one.html":   `<html><body><h1>Chapter one</h1><p>Hello &amp; goodbye.</p></body></html>`,
		"empty.html": `<html><body><img src="cover.png"></body></html>`,
		"two.html":   `<html><body><p>Another page.</p></body></html>`,
	}
	dirs := []string{t.TempDir(), t.TempDir()}
	for _, dir := range dirs {
		for name, body := range files {
			writeFile(t, filepath.Join(dir, name), body)
		}
	}
	r := NewRunner(config.Config{SourceLang: "en", TargetLang: "de"})
	streamBook, eagerBook := book, book
	stream := r.translateContent(context.Background(), &streamBook,
		translator.NewCachingClient(&stubEngine{}), dirs[0])

	pages := make([]contentPage, 0, len(book.Manifest))
	for _, item := range book.ContentFiles() {
		pages = append(pages, loadContentPage(&eagerBook, dirs[1], item))
	}
	eager := translationOutcome{state: outputpath.TranslationFull, snippets: map[string]string{}}
	client := translator.NewCachingClient(&stubEngine{})
	for i, page := range pages {
		if len(page.segments) == 0 {
			eager.snippets[page.item.Href] = htmlgen.ExtractSnippetFromDoc(page.doc)
			continue
		}
		eager.pages++
		if r.translatePage(context.Background(), &eager, client, page, i+1, len(pages)) {
			eager.done++
		}
	}
	r.translateLabels(context.Background(), &eagerBook, client, &eager)
	if !reflect.DeepEqual(stream, eager) || !reflect.DeepEqual(streamBook, eagerBook) {
		t.Fatalf("stream outcome/book differs from eager: stream=%+v, eager=%+v", stream, eager)
	}
	for name := range files {
		got, err := os.ReadFile(filepath.Join(dirs[0], name))
		if err != nil {
			t.Fatal(err)
		}
		want, err := os.ReadFile(filepath.Join(dirs[1], name))
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != string(want) {
			t.Errorf("%s differs from the eager path", name)
		}
	}
}
