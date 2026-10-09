package tests

// Ticket 111, criterion 12: the discoverability fields agree across channels. docs/discoverability.json
// holds what GitHub, winget, the Store and the extension listing are found by; a tag dropped from one
// channel for a policy reason is dropped from all of them, and the lists the build owns (winget Tags,
// the Store search terms) are held against that file here. The vocabulary is closed on purpose: a
// competitor's name cannot be listed as forbidden in a public file (canon PROMOTION section 1 rule 4),
// so a term enters a list only by being added to allowedTags - which is the review point.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"
)

type discoverability struct {
	GitHub struct {
		Description string   `json:"description"`
		Homepage    string   `json:"homepage"`
		Topics      []string `json:"topics"`
	} `json:"github"`
	Winget struct {
		Moniker string   `json:"moniker"`
		Tags    []string `json:"tags"`
	} `json:"winget"`
	AllowedTags  []string `json:"allowedTags"`
	DroppedTerms []struct {
		Term string `json:"term"`
	} `json:"droppedTerms"`
	DroppedPhrases []struct {
		Phrase string `json:"phrase"`
	} `json:"droppedPhrases"`
	ForbiddenPhrases []string `json:"forbiddenPhrases"`
}

func loadDiscoverability(t *testing.T) discoverability {
	t.Helper()
	var d discoverability
	if err := json.Unmarshal([]byte(readRepoFile(t, "docs", "discoverability.json")), &d); err != nil {
		t.Fatalf("docs/discoverability.json: %v", err)
	}
	return d
}

// retiredNames are the display names docs/positioning.json retired (SITE-REPRESENTATION rule 4).
func retiredNames(t *testing.T) []string {
	t.Helper()
	var p struct {
		RetiredNames []string `json:"retiredNames"`
	}
	if err := json.Unmarshal([]byte(readRepoFile(t, "docs", "positioning.json")), &p); err != nil {
		t.Fatalf("docs/positioning.json: %v", err)
	}
	return p.RetiredNames
}

// bannedIn returns what a piece of text carries that a published discoverability field must not: a
// dropped phrase, a comparison phrase, or a retired name. Matching is case-insensitive.
func (d discoverability) bannedIn(text string, retired []string) []string {
	low := strings.ToLower(text)
	var hits []string
	for _, p := range d.DroppedPhrases {
		if strings.Contains(low, strings.ToLower(p.Phrase)) {
			hits = append(hits, p.Phrase)
		}
	}
	for _, p := range d.ForbiddenPhrases {
		if strings.Contains(low, strings.ToLower(p)) {
			hits = append(hits, strings.TrimSpace(p))
		}
	}
	for _, r := range retired {
		if strings.Contains(low, strings.ToLower(r)) {
			hits = append(hits, r)
		}
	}
	return hits
}

func (d discoverability) isDropped(tag string) bool {
	for _, x := range d.DroppedTerms {
		if strings.EqualFold(x.Term, tag) {
			return true
		}
	}
	return false
}

func (d discoverability) isAllowed(tag string) bool {
	for _, a := range d.AllowedTags {
		if a == tag {
			return true
		}
	}
	return false
}

var topicRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

func TestDiscoverabilityGitHubFields(t *testing.T) {
	d := loadDiscoverability(t)
	if n := utf8.RuneCountInString(d.GitHub.Description); n < 20 || n > 350 {
		t.Errorf("github.description is %d characters, want 20-350", n)
	}
	if hits := d.bannedIn(d.GitHub.Description, retiredNames(t)); len(hits) > 0 {
		t.Errorf("github.description carries %q", hits)
	}
	if !strings.HasPrefix(d.GitHub.Homepage, siteBase) {
		t.Errorf("github.homepage is %q, want the site %s", d.GitHub.Homepage, siteBase)
	}
	if len(d.GitHub.Topics) == 0 || len(d.GitHub.Topics) > 20 {
		t.Errorf("github.topics has %d entries, want 1-20", len(d.GitHub.Topics))
	}
	seen := map[string]bool{}
	for _, topic := range d.GitHub.Topics {
		switch {
		case !topicRe.MatchString(topic) || len(topic) > 50:
			t.Errorf("topic %q is not lowercase letters, digits and hyphens within 50 characters", topic)
		case seen[topic]:
			t.Errorf("topic %q is listed twice", topic)
		case d.isDropped(topic):
			t.Errorf("topic %q is on the dropped list", topic)
		case !d.isAllowed(topic):
			t.Errorf("topic %q is not in allowedTags - add it there deliberately", topic)
		}
		seen[topic] = true
	}
}

func TestDroppedTermsStayDropped(t *testing.T) {
	d := loadDiscoverability(t)
	if !d.isDropped("google-translate") {
		t.Error("google-translate must be on the dropped list (ticket 111, criterion 12)")
	}
	for _, a := range d.AllowedTags {
		if d.isDropped(a) {
			t.Errorf("%q is both allowed and dropped", a)
		}
	}
}

func wingetLocaleFiles(t *testing.T) []string {
	t.Helper()
	files, err := filepath.Glob(filepath.Join("..", "winget", "*.locale.*.yaml"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no winget locale manifests found: %v", err)
	}
	sort.Strings(files)
	return files
}

// yamlList reads the "- item" lines under a top-level "Key:" of a manifest.
func yamlList(text, key string) []string {
	var out []string
	in := false
	for _, line := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		switch {
		case line == key+":":
			in = true
		case in && strings.HasPrefix(line, "- "):
			out = append(out, strings.TrimSpace(strings.TrimPrefix(line, "- ")))
		case in:
			return out
		}
	}
	return out
}

// yamlScalar reads a top-level "Key: value" line, unquoting a double-quoted value.
func yamlScalar(text, key string) string {
	for _, line := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		if strings.HasPrefix(line, key+": ") {
			v := strings.TrimSpace(strings.TrimPrefix(line, key+": "))
			if len(v) >= 2 && v[0] == '"' && v[len(v)-1] == '"' {
				v = strings.NewReplacer(`\"`, `"`, `\\`, `\`).Replace(v[1 : len(v)-1])
			}
			return v
		}
	}
	return ""
}

func TestWingetFieldsFollowTheFieldTable(t *testing.T) {
	d := loadDiscoverability(t)
	retired := retiredNames(t)
	if len(d.Winget.Tags) == 0 || len(d.Winget.Tags) > 16 {
		t.Fatalf("winget.tags has %d entries, want 1-16 (the manifest limit)", len(d.Winget.Tags))
	}
	for _, tag := range d.Winget.Tags {
		if d.isDropped(tag) || !d.isAllowed(tag) {
			t.Errorf("winget tag %q is dropped or not in allowedTags", tag)
		}
	}
	want := append([]string(nil), d.Winget.Tags...)
	sort.Strings(want)
	for _, f := range wingetLocaleFiles(t) {
		body, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		text := string(body)
		name := filepath.Base(f)
		got := yamlList(text, "Tags")
		sort.Strings(got)
		if strings.Join(got, ",") != strings.Join(want, ",") {
			t.Errorf("%s: Tags are %v, docs/discoverability.json winget.tags is %v", name, got, want)
		}
		if m := yamlScalar(text, "Moniker"); m != "" && m != d.Winget.Moniker {
			t.Errorf("%s: Moniker is %q, want %q", name, m, d.Winget.Moniker)
		}
		short := yamlScalar(text, "ShortDescription")
		if n := utf8.RuneCountInString(short); n < 3 || n > 256 {
			t.Errorf("%s: ShortDescription is %d characters, want 3-256", name, n)
		}
		if hits := d.bannedIn(short+"\n"+text, retired); len(hits) > 0 {
			t.Errorf("%s carries %q", name, hits)
		}
	}
}

// storeSections reads a tools/store/listing/<code>.txt source: "@@Name" opens a section, comment lines
// before the first section are ignored.
func storeSections(text string) map[string]string {
	out := map[string]string{}
	name := ""
	var body []string
	flush := func() {
		if name != "" {
			out[name] = strings.TrimSpace(strings.Join(body, "\n"))
		}
	}
	for _, line := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		if strings.HasPrefix(line, "@@") {
			flush()
			name, body = strings.TrimSpace(strings.TrimPrefix(line, "@@")), nil
			continue
		}
		if name != "" {
			body = append(body, line)
		}
	}
	flush()
	return out
}

// Partner Center limits, measured on the 2026-07-23 import failure: at most 7 search terms, 40
// characters each, 21 words in all; a feature is at most 200 characters and there are at most 20.
const (
	storeMaxTerms        = 7
	storeMaxTermRunes    = 40
	storeMaxTermWords    = 21
	storeMaxFeatureRunes = 200
)

func TestStoreListingsFollowTheFieldTable(t *testing.T) {
	d := loadDiscoverability(t)
	retired := retiredNames(t)
	files, _ := filepath.Glob(filepath.Join("..", "tools", "store", "listing", "*.txt"))
	if len(files) != 13 {
		t.Fatalf("%d store listing sources, want 13", len(files))
	}
	for _, f := range files {
		body, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		name := filepath.Base(f)
		sec := storeSections(string(body))
		terms, words := 0, 0
		for i := 1; i <= storeMaxTerms+3; i++ {
			v, ok := sec["SearchTerm"+strconv.Itoa(i)]
			if !ok {
				continue
			}
			if v == "" {
				continue
			}
			terms++
			words += len(strings.Fields(v))
			if i > storeMaxTerms {
				t.Errorf("%s: SearchTerm%d is past the limit of %d terms", name, i, storeMaxTerms)
			}
			if n := utf8.RuneCountInString(v); n > storeMaxTermRunes {
				t.Errorf("%s: SearchTerm%d is %d characters, the limit is %d: %q", name, i, n, storeMaxTermRunes, v)
			}
		}
		if terms == 0 {
			t.Errorf("%s: no search terms", name)
		}
		if words > storeMaxTermWords {
			t.Errorf("%s: the search terms hold %d words, the limit is %d (a longer set fails the import for that language)", name, words, storeMaxTermWords)
		}
		features := 0
		for key, v := range sec {
			if strings.HasPrefix(key, "Feature") && v != "" {
				features++
				if n := utf8.RuneCountInString(v); n > storeMaxFeatureRunes {
					t.Errorf("%s: %s is %d characters, the limit is %d", name, key, n, storeMaxFeatureRunes)
				}
			}
		}
		if features > 20 {
			t.Errorf("%s: %d features, the limit is 20", name, features)
		}
		var all []string
		for key, v := range sec {
			if key != "ReleaseNotes" {
				all = append(all, v)
			}
		}
		if hits := d.bannedIn(strings.Join(all, "\n"), retired); len(hits) > 0 {
			t.Errorf("%s carries %q", name, hits)
		}
		for i := 1; i <= storeMaxTerms; i++ {
			if v := sec["SearchTerm"+strconv.Itoa(i)]; v != "" && d.isDropped(strings.ReplaceAll(strings.ToLower(v), " ", "-")) {
				t.Errorf("%s: SearchTerm%d %q is on the dropped list", name, i, v)
			}
		}
	}
}
