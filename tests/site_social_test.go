package tests

// Ticket 111, phases 03 and 04: what a crawler or a chat client reads from the head of a page.
//
// Open Graph asks for the image's size and a text alternative beside its address; a page that
// declares a size the file does not have is cropped by the client, so the declaration is held to the
// file itself. The landing carries the one social card (canon PROMOTION section 4 item 2).
//
// Structured data is truthful or absent (canon PROMOTION section 3): no rating or review (there is no
// real one on any channel), no version retyped into a page, and no FAQ or how-to markup added for
// search reach.

import (
	"encoding/json"
	"image/png"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

const landingCard = "assets/social-card-1200x630.png"

func TestSiteOpenGraphImages(t *testing.T) {
	for _, p := range sitePages() {
		doc := parseSitePage(t, p)
		img, ok := metaContent(doc, "property", "og:image")
		if !ok || img == "" {
			t.Errorf("%s: no og:image", p)
			continue
		}
		rel := strings.TrimPrefix(img, siteBase)
		if rel == img {
			t.Errorf("%s: og:image %s is outside the site %s", p, img, siteBase)
			continue
		}
		f, err := os.Open(filepath.Join("..", filepath.FromSlash(rel)))
		if err != nil {
			t.Errorf("%s: og:image names a file that is not in the tree: %v", p, err)
			continue
		}
		cfg, err := png.DecodeConfig(f)
		f.Close()
		if err != nil {
			t.Errorf("%s: og:image %s is not a decodable PNG: %v", p, rel, err)
			continue
		}
		for _, d := range []struct {
			name string
			real int
		}{{"og:image:width", cfg.Width}, {"og:image:height", cfg.Height}} {
			v, ok := metaContent(doc, "property", d.name)
			n, err := strconv.Atoi(v)
			if !ok || err != nil {
				t.Errorf("%s: %s is missing or not a number (%q)", p, d.name, v)
				continue
			}
			if n != d.real {
				t.Errorf("%s: %s is %d, the file %s is %d", p, d.name, n, rel, d.real)
			}
		}
		if alt, _ := metaContent(doc, "property", "og:image:alt"); strings.TrimSpace(alt) == "" {
			t.Errorf("%s: og:image:alt is missing", p)
		}
		if tw, ok := metaContent(doc, "name", "twitter:image"); ok && tw != img {
			t.Errorf("%s: twitter:image %s differs from og:image %s", p, tw, img)
		}
	}
}

func TestLandingUsesTheSocialCard(t *testing.T) {
	img, _ := metaContent(parseSitePage(t, "index.html"), "property", "og:image")
	if img != siteBase+landingCard {
		t.Errorf("index.html og:image is %s, want the social card %s", img, siteBase+landingCard)
	}
}

var jsonLDRe = regexp.MustCompile(`(?s)<script type="application/ld\+json">(.*?)</script>`)

// jsonLDForbidden are the keys and types the campaign never writes (ticket 111 section 2).
var jsonLDForbidden = []string{"aggregateRating", "review", "softwareVersion", "FAQPage", "HowTo"}

func TestSiteJSONLDIsTruthful(t *testing.T) {
	for _, p := range sitePages() {
		blocks := jsonLDRe.FindAllStringSubmatch(readRepoFile(t, strings.Split(p, "/")...), -1)
		if len(blocks) != 1 {
			t.Errorf("%s: %d JSON-LD blocks, want exactly one", p, len(blocks))
			continue
		}
		var v any
		if err := json.Unmarshal([]byte(blocks[0][1]), &v); err != nil {
			t.Errorf("%s: the JSON-LD block is not valid JSON: %v", p, err)
			continue
		}
		found := map[string]bool{}
		collectJSONLDNames(v, found)
		for _, bad := range jsonLDForbidden {
			if found[bad] {
				t.Errorf("%s: the JSON-LD block carries %q (no real rating, review or retyped version exists)", p, bad)
			}
		}
	}
}

// collectJSONLDNames records every key and every @type value of a decoded JSON-LD tree.
func collectJSONLDNames(v any, found map[string]bool) {
	switch x := v.(type) {
	case map[string]any:
		for k, val := range x {
			found[k] = true
			if k == "@type" {
				if s, ok := val.(string); ok {
					found[s] = true
				}
			}
			collectJSONLDNames(val, found)
		}
	case []any:
		for _, val := range x {
			collectJSONLDNames(val, found)
		}
	}
}

// The two software pages carry an address a reader can install from, durable across versions.
func TestSoftwareApplicationCarriesAnInstallAddress(t *testing.T) {
	for _, p := range sitePages() {
		body := readRepoFile(t, strings.Split(p, "/")...)
		m := jsonLDRe.FindStringSubmatch(body)
		if m == nil || !strings.Contains(m[1], `"SoftwareApplication"`) {
			continue
		}
		if !strings.Contains(m[1], `"downloadUrl"`) && !strings.Contains(m[1], `"installUrl"`) {
			t.Errorf("%s: its SoftwareApplication has neither downloadUrl nor installUrl", p)
		}
	}
}
