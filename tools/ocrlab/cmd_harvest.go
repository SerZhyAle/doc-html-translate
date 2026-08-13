package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"doc-html-translate/tools/ocrlab/corpus"
)

// Wikimedia Commons is the harvest source because it is the only large image collection whose
// licence is machine-readable *from the asset's own file page*. The API returns the same
// LicenseShortName a person would read there, so a harvested entry records a real licence rather
// than a guess from a search result - which is what the corpus rule is actually about.
//
// Every harvested scene is stamped `commons-api` in licenceVerifiedBy, never a person's name, so
// a reviewer can always tell machine-read provenance from human-read provenance and re-check any
// entry against its licenceUrl.
const (
	commonsAPI      = "https://commons.wikimedia.org/w/api.php"
	commonsVerifier = "commons-api"
	harvestUA       = "doc-html-translate-ocrlab/1.0 (https://github.com/SerZhyAle/doc-html-translate)"
)

// acceptedCommonsLicence maps what the file page says to our closed licence set. Anything not in
// this table is skipped with its licence named, so a non-commercial or unclear file is refused
// loudly instead of quietly downloaded.
func acceptedCommonsLicence(short string) (corpus.Licence, bool) {
	s := strings.ToLower(strings.TrimSpace(short))
	switch {
	case strings.Contains(s, "public domain"), s == "pd", strings.Contains(s, "pd-"):
		return corpus.LicencePD, true
	case strings.Contains(s, "cc pdm"), strings.Contains(s, "public domain mark"):
		return corpus.LicencePDM, true
	case strings.Contains(s, "cc0"):
		return corpus.LicenceCC0, true
	case s == "cc by 3.0", s == "cc by-sa 3.0":
		// BY-SA carries a share-alike obligation on derivatives. The corpus stores originals and
		// keeps derivatives tied to them, so BY is safe and BY-SA is deliberately not accepted.
		if s == "cc by 3.0" {
			return corpus.LicenceCCBY30, true
		}
		return "", false
	case s == "cc by 4.0":
		return corpus.LicenceCCBY40, true
	default:
		return "", false
	}
}

// cmdHarvest pulls a batch of licence-checked images out of Commons and registers them.
func cmdHarvest(args []string) error {
	fs := flag.NewFlagSet("harvest", flag.ExitOnError)
	p := addCommonFlags(fs)
	query := fs.String("search", "", "Commons search terms (required unless -category is given)")
	category := fs.String("category", "", "Commons category to walk, without the Category: prefix")
	cats := fs.String("categories", "document", "lab categories to tag the results with: "+categoryList())
	limit := fs.Int("limit", 20, "how many images to take")
	minPx := fs.Int("min-px", 500, "skip images whose long side is below this")
	maxBytes := fs.Int64("max-bytes", 12<<20, "skip images larger than this")
	split := fs.String("split", "dev", "dev or holdout")
	dry := fs.Bool("dry-run", false, "list what would be taken and stop")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *query == "" && *category == "" {
		return fmt.Errorf("give -search or -category")
	}
	labCats, err := parseCategories(*cats)
	if err != nil {
		return err
	}
	if !corpus.Split(*split).Valid() {
		return fmt.Errorf("split %q is neither dev nor holdout", *split)
	}

	titles, err := commonsTitles(*query, *category, *limit*3)
	if err != nil {
		return err
	}
	if len(titles) == 0 {
		return fmt.Errorf("no files found")
	}
	infos, err := commonsInfo(titles)
	if err != nil {
		return err
	}

	m, err := corpus.Load(p.manifest)
	if err != nil {
		return err
	}
	today := todayStamp()
	root := filepath.Join(p.root, "commons")
	if err := os.MkdirAll(root, 0o755); err != nil {
		return err
	}

	var taken, skipped int
	for _, in := range infos {
		if taken >= *limit {
			break
		}
		licence, ok := acceptedCommonsLicence(in.Licence)
		if !ok {
			fmt.Printf("skip   %-52s licence %q not accepted\n", short(in.Title), in.Licence)
			skipped++
			continue
		}
		if maxSide(in.Width, in.Height) < *minPx {
			fmt.Printf("skip   %-52s only %dx%d\n", short(in.Title), in.Width, in.Height)
			skipped++
			continue
		}
		if in.Size > *maxBytes {
			fmt.Printf("skip   %-52s %.1f MB\n", short(in.Title), float64(in.Size)/(1<<20))
			skipped++
			continue
		}
		id := slug(strings.TrimPrefix(in.Title, "File:"))
		if m.Find(id) != nil {
			fmt.Printf("skip   %-52s already in the manifest\n", short(in.Title))
			skipped++
			continue
		}
		if *dry {
			fmt.Printf("would  %-52s %s %dx%d\n", short(in.Title), licence, in.Width, in.Height)
			taken++
			continue
		}

		ext := strings.ToLower(filepath.Ext(in.URL))
		if i := strings.IndexAny(ext, "?#"); i >= 0 {
			ext = ext[:i]
		}
		rel := filepath.ToSlash(filepath.Join("commons", id+ext))
		dst := filepath.Join(p.root, filepath.FromSlash(rel))
		if err := download(in.URL, dst); err != nil {
			fmt.Printf("FAIL   %-52s %v\n", short(in.Title), err)
			skipped++
			continue
		}
		sum, err := corpus.HashFile(dst)
		if err != nil {
			return err
		}
		info, err := os.Stat(dst)
		if err != nil {
			return err
		}
		m.Upsert(corpus.Scene{
			ID:                id,
			File:              rel,
			SourceURL:         in.URL,
			SourceItem:        in.Title,
			RetrievedOn:       today,
			Licence:           licence,
			LicenceURL:        in.DescriptionURL,
			Attribution:       stripHTML(in.Artist),
			LicenceVerifiedBy: commonsVerifier,
			LicenceVerifiedOn: today,
			SHA256:            sum,
			Bytes:             info.Size(),
			Width:             in.Width,
			Height:            in.Height,
			Categories:        labCats,
			Split:             corpus.Split(*split),
			Note:              "licence read from the Commons file page metadata by API, not by a person",
		})
		fmt.Printf("ok     %-52s %s %dx%d\n", short(in.Title), licence, in.Width, in.Height)
		taken++
	}

	if *dry {
		fmt.Printf("\ndry run: %d would be taken, %d skipped\n", taken, skipped)
		return nil
	}
	if err := corpus.Save(p.manifest, m); err != nil {
		return err
	}
	fmt.Printf("\nharvested %d, skipped %d. Manifest: %s\n", taken, skipped, p.manifest)
	return nil
}

type commonsImage struct {
	Title          string
	URL            string
	DescriptionURL string
	Licence        string
	Artist         string
	Width          int
	Height         int
	Size           int64
}

// commonsTitles finds candidate file titles by search or by category walk.
func commonsTitles(query, category string, limit int) ([]string, error) {
	q := url.Values{"action": {"query"}, "format": {"json"}, "formatversion": {"2"}}
	if category != "" {
		q.Set("list", "categorymembers")
		q.Set("cmtitle", "Category:"+category)
		q.Set("cmtype", "file")
		q.Set("cmlimit", fmt.Sprint(min(limit, 500)))
	} else {
		q.Set("list", "search")
		q.Set("srsearch", "filetype:bitmap "+query)
		q.Set("srnamespace", "6")
		q.Set("srlimit", fmt.Sprint(min(limit, 500)))
	}
	var out struct {
		Query struct {
			Search          []struct{ Title string } `json:"search"`
			CategoryMembers []struct{ Title string } `json:"categorymembers"`
		} `json:"query"`
	}
	if err := getJSON(commonsAPI+"?"+q.Encode(), &out); err != nil {
		return nil, err
	}
	var titles []string
	for _, s := range out.Query.Search {
		titles = append(titles, s.Title)
	}
	for _, s := range out.Query.CategoryMembers {
		titles = append(titles, s.Title)
	}
	return titles, nil
}

// commonsInfo reads each file's own page metadata, 50 titles per request (the API's limit).
func commonsInfo(titles []string) ([]commonsImage, error) {
	var out []commonsImage
	for i := 0; i < len(titles); i += 50 {
		batch := titles[i:min(i+50, len(titles))]
		q := url.Values{
			"action": {"query"}, "format": {"json"}, "formatversion": {"2"},
			"prop": {"imageinfo"}, "iiprop": {"url|size|dimensions|extmetadata"},
			"titles": {strings.Join(batch, "|")},
		}
		var res struct {
			Query struct {
				Pages []struct {
					Title     string `json:"title"`
					Missing   bool   `json:"missing"`
					ImageInfo []struct {
						URL            string `json:"url"`
						DescriptionURL string `json:"descriptionurl"`
						Size           int64  `json:"size"`
						Width          int    `json:"width"`
						Height         int    `json:"height"`
						// value is a string for most fields and a number for a few (image dates,
						// some ratings), so it is decoded loosely and flattened below rather than
						// failing the whole batch on one numeric field.
						ExtMetadata map[string]struct {
							Value any `json:"value"`
						} `json:"extmetadata"`
					} `json:"imageinfo"`
				} `json:"pages"`
			} `json:"query"`
		}
		if err := getJSON(commonsAPI+"?"+q.Encode(), &res); err != nil {
			return nil, err
		}
		for _, p := range res.Query.Pages {
			if p.Missing || len(p.ImageInfo) == 0 {
				continue
			}
			ii := p.ImageInfo[0]
			out = append(out, commonsImage{
				Title:          p.Title,
				URL:            stripQuery(ii.URL),
				DescriptionURL: ii.DescriptionURL,
				Licence:        metaString(ii.ExtMetadata["LicenseShortName"].Value),
				Artist:         metaString(ii.ExtMetadata["Artist"].Value),
				Width:          ii.Width,
				Height:         ii.Height,
				Size:           ii.Size,
			})
		}
	}
	// Deterministic order, so two harvests of the same query take the same files.
	sort.Slice(out, func(i, j int) bool { return out[i].Title < out[j].Title })
	return out, nil
}

var harvestClient = &http.Client{Timeout: 90 * time.Second}

func getJSON(u string, v any) error {
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", harvestUA)
	resp, err := harvestClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d from %s", resp.StatusCode, u)
	}
	return json.NewDecoder(resp.Body).Decode(v)
}

func download(u, dst string) error {
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", harvestUA)
	resp, err := harvestClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := f.ReadFrom(resp.Body); err != nil {
		f.Close()
		os.Remove(dst)
		return err
	}
	return f.Close()
}

// metaString flattens a loosely-typed extmetadata value to text.
func metaString(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprint(v)
}

func stripQuery(u string) string {
	if i := strings.IndexByte(u, '?'); i >= 0 {
		return u[:i]
	}
	return u
}

// stripHTML flattens the Artist field, which Commons returns as a link.
func stripHTML(s string) string {
	var b strings.Builder
	depth := 0
	for _, r := range s {
		switch r {
		case '<':
			depth++
		case '>':
			if depth > 0 {
				depth--
			}
		default:
			if depth == 0 {
				b.WriteRune(r)
			}
		}
	}
	return strings.TrimSpace(b.String())
}

func short(s string) string {
	s = strings.TrimPrefix(s, "File:")
	r := []rune(s)
	if len(r) <= 50 {
		return s
	}
	return string(r[:48]) + ".."
}

func maxSide(w, h int) int { return max(w, h) }

func todayStamp() string { return time.Now().Format("2006-01-02") }
