package main

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"golang.org/x/net/html"
)

// Expectation is the reviewed account of what a reader must get from one case. A draft is written
// by `doccorpus draft` from the source's own bytes - a PDF's text layer, an EPUB's XHTML, the
// author's lettering files - and never from either edition's output, so neither edition can be
// graded against itself. It becomes truth only when a person has checked it against the original,
// set origin to "human" and signed reviewedBy / reviewedOn.
type Expectation struct {
	SchemaVersion int    `json:"schemaVersion"`
	CaseID        string `json:"caseId"`
	Origin        string `json:"origin"`
	DraftSource   string `json:"draftSource"`
	ReviewedBy    string `json:"reviewedBy"`
	ReviewedOn    string `json:"reviewedOn"`

	// Facts measured from the bytes (test_doc/INVENTORY.json, the corpus inventory).
	Pages           int    `json:"pages"`
	TextLayer       string `json:"textLayer,omitempty"`
	SourceTextChars int    `json:"sourceTextChars,omitempty"`
	Images          int    `json:"images,omitempty"`

	// What the reader must get.
	Lang      string   `json:"lang"`
	Snippets  []string `json:"snippets"`
	Lettering []string `json:"lettering,omitempty"`
	Omissions []string `json:"omissions"`
	Notes     string   `json:"notes"`
}

// IsTruth reports whether a person has reviewed the expectation.
func (e Expectation) IsTruth() bool {
	return e.Origin == "human" && strings.TrimSpace(e.ReviewedBy) != "" && dateRE.MatchString(e.ReviewedOn)
}

// LoadExpectation reads one expectation file.
func LoadExpectation(p string) (*Expectation, error) {
	raw, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	var e Expectation
	if err := json.Unmarshal(raw, &e); err != nil {
		return nil, fmt.Errorf("%s: %w", p, err)
	}
	return &e, nil
}

// inventoryItem is the part of test_doc/INVENTORY.json a draft reads.
type inventoryItem struct {
	Path          string `json:"path"`
	SHA256        string `json:"sha256"`
	Pages         int    `json:"pages"`
	TextChars     int    `json:"textChars"`
	PagesWithText int    `json:"pagesWithText"`
	TextLayer     string `json:"textLayer"`
	ImagePages    int    `json:"imagePages"`
	Width         int    `json:"width"`
	Images        int    `json:"images"`
}

func loadInventory(root string) (map[string]inventoryItem, error) {
	raw, err := os.ReadFile(filepath.Join(root, "INVENTORY.json"))
	if err != nil {
		return nil, err
	}
	var inv struct {
		Items []inventoryItem `json:"items"`
	}
	if err := json.Unmarshal(raw, &inv); err != nil {
		return nil, err
	}
	out := make(map[string]inventoryItem, len(inv.Items))
	for _, it := range inv.Items {
		out[it.Path] = it
	}
	return out, nil
}

// langPackDir maps a case language to the Pepper&Carrot lang-pack folder.
var langPackDir = map[string]string{"en": "en", "fr": "fr", "ru": "ru", "ja": "ja", "zh": "cn", "ko": "kr"}

const langPack = "multilingual_2026-09-29/xx-peppercarrot-e06-lang-pack.zip"

// Draft builds the draft expectation of one case. It never overwrites a reviewed one.
func Draft(root string, inv map[string]inventoryItem, c Case) (*Expectation, error) {
	e := &Expectation{SchemaVersion: SchemaVersion, CaseID: c.ID, Origin: "draft", Lang: c.Language, Snippets: []string{}, Omissions: []string{}}
	it, ok := inv[c.File]
	if ok && it.SHA256 != c.SHA256 {
		return nil, fmt.Errorf("%s: test_doc/INVENTORY.json measured different bytes", c.ID)
	}
	src := filepath.Join(root, filepath.FromSlash(c.File))
	switch c.Class {
	case "comic-image":
		e.Pages = 1
	case "comic-archive":
		e.Pages = it.ImagePages
	default:
		e.Pages = it.Pages
	}
	e.TextLayer = it.TextLayer
	e.SourceTextChars = it.TextChars
	// The inventory calls any layer that yields text "full"; a scan carrying someone else's OCR is
	// recorded as a weak layer by the case itself, and that text is never truth.
	if hasFeature(c, "weak-text-layer") {
		e.TextLayer = "weak"
		e.SourceTextChars = 0
	}

	var err error
	switch {
	case strings.Contains(c.File, "peppercarrot"):
		e.Lettering, err = peppercarrotLettering(root, c)
		e.DraftSource = "author lettering: " + langPack + " (lang/" + langPackDir[c.Language] + "/E06P*.svg)"
		e.Notes = "Lettering lines are the author's own SVG text in reading order of the file, not OCR. " +
			"The painted sign on P03 is part of the artwork; check whether it must stay unplated."
	case strings.HasSuffix(c.File, ".epub"):
		e.Snippets, err = epubSnippets(src)
		e.DraftSource = "source XHTML of the EPUB spine"
	case strings.HasSuffix(c.File, ".pdf") && e.TextLayer == "full":
		e.Snippets, err = pdfSnippets(root, src)
		e.DraftSource = "the PDF's own text layer (bundled pdftotext)"
	default:
		e.DraftSource = "none"
		e.Notes = "No independent transcription exists yet: a person transcribes representative lettering or text " +
			"from the original page. OCR output must not seed it."
	}
	if err != nil {
		return nil, fmt.Errorf("%s: %w", c.ID, err)
	}
	if e.TextLayer == "weak" {
		e.Notes = strings.TrimSpace(e.Notes + " The text layer is someone else's OCR, so it is not used as truth.")
	}
	return e, nil
}

// ---- PDF --------------------------------------------------------------------

func pdfSnippets(root, src string) ([]string, error) {
	tool := filepath.Join(filepath.Dir(root), "internal", "bundledtools", "pdftotext", "pdftotext.exe")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	out, err := exec.CommandContext(ctx, tool, "-enc", "UTF-8", "-f", "1", "-l", "4", src, "-").Output()
	if err != nil {
		return nil, fmt.Errorf("pdftotext: %w", err)
	}
	var lines []string
	for _, l := range strings.Split(string(out), "\n") {
		lines = append(lines, strings.TrimSpace(strings.Trim(l, "\f")))
	}
	return pickSnippets(lines, 4), nil
}

// ---- EPUB -------------------------------------------------------------------

func epubSnippets(src string) ([]string, error) {
	zr, err := zip.OpenReader(src)
	if err != nil {
		return nil, err
	}
	defer zr.Close()
	files := map[string]*zip.File{}
	for _, f := range zr.File {
		files[f.Name] = f
	}
	read := func(name string) ([]byte, error) {
		f, ok := files[name]
		if !ok {
			return nil, fmt.Errorf("no %s in the archive", name)
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		defer rc.Close()
		return io.ReadAll(rc)
	}
	var container struct {
		Rootfiles []struct {
			FullPath string `xml:"full-path,attr"`
		} `xml:"rootfiles>rootfile"`
	}
	raw, err := read("META-INF/container.xml")
	if err != nil {
		return nil, err
	}
	if err := xml.Unmarshal(raw, &container); err != nil || len(container.Rootfiles) == 0 {
		return nil, fmt.Errorf("container.xml: no rootfile")
	}
	opfPath := container.Rootfiles[0].FullPath
	var opf struct {
		Items []struct {
			ID   string `xml:"id,attr"`
			Href string `xml:"href,attr"`
		} `xml:"manifest>item"`
		Spine []struct {
			IDRef string `xml:"idref,attr"`
		} `xml:"spine>itemref"`
	}
	if raw, err = read(opfPath); err != nil {
		return nil, err
	}
	if err := xml.Unmarshal(raw, &opf); err != nil {
		return nil, fmt.Errorf("%s: %w", opfPath, err)
	}
	href := map[string]string{}
	for _, it := range opf.Items {
		href[it.ID] = it.Href
	}
	var docs []string
	for _, s := range opf.Spine {
		if h, ok := href[s.IDRef]; ok {
			docs = append(docs, path.Join(path.Dir(opfPath), unescapePath(h)))
		}
	}
	// The richest spine documents, in spine order, so the snippets also pin reading order.
	var lines []string
	for _, d := range docs {
		raw, err := read(d)
		if err != nil {
			continue
		}
		lines = append(lines, htmlBlocks(raw)...)
	}
	return pickSnippets(lines, 4), nil
}

func unescapePath(h string) string {
	h = strings.SplitN(h, "#", 2)[0]
	if u, err := url.PathUnescape(h); err == nil {
		return u
	}
	return h
}

// htmlBlocks returns the text of each paragraph-level element.
func htmlBlocks(raw []byte) []string {
	doc, err := html.Parse(bytes.NewReader(raw))
	if err != nil {
		return nil
	}
	var out []string
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "script", "style", "rt", "rp", "head":
				return
			case "p", "h1", "h2", "h3", "h4", "li", "blockquote":
				out = append(out, collapse(textOf(n)))
				return
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return out
}

// textOf concatenates a node's text, leaving ruby annotations out: the base text is what a reader
// searches for and what both editions must keep.
func textOf(n *html.Node) string {
	var sb strings.Builder
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && (n.Data == "rt" || n.Data == "rp") {
			return
		}
		if n.Type == html.TextNode {
			sb.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return sb.String()
}

// ---- Pepper&Carrot ------------------------------------------------------------

var svgTextRE = regexp.MustCompile(`<(?:flowPara|tspan)\b[^>]*>([^<]*)<`)

// peppercarrotLettering reads the author's lettering for the case: page 03 for a standalone page,
// every page for the episode archive.
func peppercarrotLettering(root string, c Case) ([]string, error) {
	zr, err := zip.OpenReader(filepath.Join(root, filepath.FromSlash(langPack)))
	if err != nil {
		return nil, err
	}
	defer zr.Close()
	dir := langPackDir[c.Language]
	pages := []string{"E06P03"}
	if c.Class == "comic-archive" {
		pages = []string{"E06P00", "E06P01", "E06P02", "E06P03", "E06P04", "E06P05", "E06P06", "E06P07", "E06P08", "E06P09", "E06P10"}
	}
	var out []string
	for _, p := range pages {
		name := "lang/" + dir + "/" + p + ".svg"
		var f *zip.File
		for _, zf := range zr.File {
			if zf.Name == name {
				f = zf
				break
			}
		}
		if f == nil {
			return nil, fmt.Errorf("no %s in the lang-pack", name)
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		raw, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return nil, err
		}
		for _, m := range svgTextRE.FindAllSubmatch(raw, -1) {
			if s := collapse(html.UnescapeString(string(m[1]))); s != "" {
				out = append(out, s)
			}
		}
	}
	return out, nil
}

// ---- helpers -------------------------------------------------------------------

func collapse(s string) string { return strings.Join(strings.Fields(s), " ") }

// pickSnippets chooses up to n distinct lines that read as prose - long enough to be unique, mostly
// letters - spread over the input so they also pin the reading order.
func pickSnippets(lines []string, n int) []string {
	var cand []string
	seen := map[string]bool{}
	for _, l := range lines {
		l = collapse(l)
		rc := utf8.RuneCountInString(l)
		cjk := hasCJK(l)
		if (cjk && (rc < 8 || rc > 60)) || (!cjk && (rc < 30 || rc > 110)) {
			continue
		}
		letters := 0
		for _, r := range l {
			if unicode.IsLetter(r) {
				letters++
			}
		}
		if float64(letters)/float64(rc) < 0.6 || seen[l] {
			continue
		}
		seen[l] = true
		cand = append(cand, l)
	}
	if len(cand) <= n {
		if cand == nil {
			return []string{}
		}
		return cand
	}
	out := make([]string, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, cand[i*(len(cand)-1)/(n-1)])
	}
	return out
}

func hasCJK(s string) bool {
	for _, r := range s {
		if unicode.In(r, unicode.Han, unicode.Hiragana, unicode.Katakana, unicode.Hangul) {
			return true
		}
	}
	return false
}
