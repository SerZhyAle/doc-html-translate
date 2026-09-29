package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// SchemaVersion is the version of DEV/doccorpus/cases.json and of every expectation file. The Node
// runner (run.mjs) refuses any other value, so a reshape is a bump on both sides or neither.
const SchemaVersion = 1

// Languages and Classes are the ticket 68 coverage matrix, in the order the report prints it.
var (
	Languages = []string{"en", "fr", "ru", "ja", "zh", "ko"}
	Classes   = []string{"comic-image", "epub", "text-pdf", "scanned-pdf", "illustrated-pdf", "comic-archive"}
	Roles     = []string{"primary", "supplementary", "windows-only"}
)

// Manifest is the document-level corpus record. It is a layer beside the OCR lab's scene manifest
// (DEV/ocrlab/corpus.json), not a second account of the same asset: a case that is also an OCR
// scene names the scene, and the lab stays the owner of its annotations.
type Manifest struct {
	SchemaVersion int    `json:"schemaVersion"`
	Ticket        string `json:"ticket"`
	Root          string `json:"root"`
	SplitRule     string `json:"splitRule"`
	Cases         []Case `json:"cases"`
}

// Case is one accepted-or-candidate source document of the campaign.
type Case struct {
	ID       string   `json:"id"`
	File     string   `json:"file"`
	Class    string   `json:"class"`
	Language string   `json:"language"`
	Script   string   `json:"script"`
	OCRLang  string   `json:"ocrLang"`
	Role     string   `json:"role"`
	Split    string   `json:"split"`
	Features []string `json:"features"`
	SHA256   string   `json:"sha256"`
	Bytes    int64    `json:"bytes"`
	OCRScene string   `json:"ocrScene,omitempty"`
	Source   Source   `json:"source"`
	Rights   Rights   `json:"rights"`
	Expect   string   `json:"expect"`
}

// Source records where the bytes came from.
type Source struct {
	Title            string `json:"title"`
	Page             string `json:"page"`
	DownloadURL      string `json:"downloadUrl"`
	FinalURL         string `json:"finalUrl"`
	RetrievedOn      string `json:"retrievedOn"`
	OriginalFilename string `json:"originalFilename"`
	Transformation   any    `json:"transformation"`
}

// Rights is the licence record. ReviewedBy and ReviewedOn are written by a person after opening
// the asset's own rights page; no code path in this tool writes them (the OCR lab's rule 3).
type Rights struct {
	Licence       string `json:"licence"`
	Evidence      string `json:"evidence"`
	Attribution   string `json:"attribution"`
	AgentReview   string `json:"agentReview"`
	ReviewedBy    string `json:"reviewedBy"`
	ReviewedOn    string `json:"reviewedOn"`
	PermitsCommit bool   `json:"permitsCommit"`
	Restrictions  string `json:"restrictions"`
}

// HumanReviewed reports whether a person has signed the rights record.
func (r Rights) HumanReviewed() bool {
	return strings.TrimSpace(r.ReviewedBy) != "" && dateRE.MatchString(r.ReviewedOn)
}

var (
	idRE   = regexp.MustCompile(`^[a-z]{2}-[a-z0-9]+(-[a-z0-9]+)*$`)
	hashRE = regexp.MustCompile(`^[0-9a-f]{64}$`)
	dateRE = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
)

// LoadManifest reads and parses the manifest without judging it.
func LoadManifest(path string) (*Manifest, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return &m, nil
}

// Validate returns every structural problem of the manifest; an empty list means it is usable.
func (m *Manifest) Validate() []string {
	var bad []string
	if m.SchemaVersion != SchemaVersion {
		bad = append(bad, fmt.Sprintf("schemaVersion %d, this tool understands %d", m.SchemaVersion, SchemaVersion))
	}
	if m.Root == "" {
		bad = append(bad, "no root")
	}
	seen := map[string]bool{}
	files := map[string]string{}
	for _, c := range m.Cases {
		at := c.ID
		if !idRE.MatchString(c.ID) {
			bad = append(bad, fmt.Sprintf("%q: id is not lower-case, language-prefixed kebab case", c.ID))
		}
		if seen[c.ID] {
			bad = append(bad, at+": duplicate id")
		}
		seen[c.ID] = true
		if !contains(Classes, c.Class) {
			bad = append(bad, fmt.Sprintf("%s: unknown class %q", at, c.Class))
		}
		if !contains(Languages, c.Language) {
			bad = append(bad, fmt.Sprintf("%s: language %q is outside the matrix", at, c.Language))
		}
		if !strings.HasPrefix(c.ID, c.Language+"-") {
			bad = append(bad, at+": id does not start with its language")
		}
		if !contains(Roles, c.Role) {
			bad = append(bad, fmt.Sprintf("%s: unknown role %q", at, c.Role))
		}
		if c.Split != "dev" && c.Split != "holdout" {
			bad = append(bad, fmt.Sprintf("%s: split %q is neither dev nor holdout", at, c.Split))
		}
		if !hashRE.MatchString(c.SHA256) {
			bad = append(bad, at+": sha256 is not 64 lower-case hex digits")
		}
		if c.Bytes <= 0 {
			bad = append(bad, at+": bytes not recorded")
		}
		if c.File == "" || strings.Contains(c.File, "\\") || strings.HasPrefix(c.File, "/") || strings.Contains(c.File, "..") {
			bad = append(bad, at+": file must be a forward-slash path under root")
		}
		// One file cannot fill two cells by relabelling (the ticket's own rule).
		if other, ok := files[c.File]; ok {
			bad = append(bad, fmt.Sprintf("%s: same file as %s", at, other))
		}
		files[c.File] = c.ID
		if c.Expect != "expect/"+c.ID+".json" {
			bad = append(bad, at+": expect must be expect/<id>.json")
		}
		if c.Rights.Licence == "" {
			bad = append(bad, at+": no licence recorded")
		}
		if (c.Rights.ReviewedBy == "") != (c.Rights.ReviewedOn == "") {
			bad = append(bad, at+": reviewedBy and reviewedOn are filled together or not at all")
		}
		if c.Rights.ReviewedOn != "" && !dateRE.MatchString(c.Rights.ReviewedOn) {
			bad = append(bad, at+": reviewedOn is not YYYY-MM-DD")
		}
		if c.Rights.PermitsCommit && !c.Rights.HumanReviewed() {
			bad = append(bad, at+": permitsCommit without a human rights review")
		}
	}
	sort.Strings(bad)
	return bad
}

// MediaState is what the bytes on disk say about one case.
type MediaState string

const (
	MediaOK      MediaState = "ok"
	MediaMissing MediaState = "missing"
	MediaChanged MediaState = "hash-changed"
)

// CheckMedia hashes a case's file under root.
func CheckMedia(root string, c Case) (MediaState, error) {
	f, err := os.Open(filepath.Join(root, filepath.FromSlash(c.File)))
	if os.IsNotExist(err) {
		return MediaMissing, nil
	}
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	if hex.EncodeToString(h.Sum(nil)) != c.SHA256 {
		return MediaChanged, nil
	}
	return MediaOK, nil
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}
