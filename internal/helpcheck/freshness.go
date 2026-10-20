package helpcheck

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// The inputs a help page or screenshot is made from, digested the way tools/helpgen/generate.mjs digests them
// (inputDigests): the two must agree on what "stale" means. ui.html counts without its script blocks - what the
// window looks like, not what it does - and every file with line endings normalized.

var scriptBlock = regexp.MustCompile(`(?s)<script\b[^>]*>.*?</script>`)

func sha(b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

func lf(b []byte) []byte { return []byte(strings.ReplaceAll(string(b), "\r\n", "\n")) }

// InputDigests returns the digest of every input and the digest of them all.
func InputDigests(root string) (parts map[string]string, digest string, err error) {
	read := func(rel string) ([]byte, error) { return os.ReadFile(filepath.Join(root, filepath.FromSlash(rel))) }
	parts = map[string]string{}
	ui, err := read(UIPath)
	if err != nil {
		return nil, "", err
	}
	parts["ui.html (markup and style)"] = sha(scriptBlock.ReplaceAll(lf(ui), nil))
	for name, rel := range map[string]string{
		"i18n.js":       "cmd/doc-html-ui/i18n.js",
		"help-map.json": MapPath,
	} {
		b, err := read(rel)
		if err != nil {
			return nil, "", err
		}
		parts[name] = sha(lf(b))
	}
	for name, rel := range map[string]string{
		"sza-kit.css": "assets/sza-kit.css",
		"help.css":    GenDir + "/vendor/help.css",
	} {
		b, err := read(rel)
		if err != nil {
			return nil, "", err
		}
		parts[name] = sha(b)
	}
	entries, err := os.ReadDir(filepath.Join(root, filepath.FromSlash(CopyDir)))
	if err != nil {
		return nil, "", err
	}
	for _, e := range entries {
		b, err := read(path.Join(CopyDir, e.Name()))
		if err != nil {
			return nil, "", err
		}
		parts["copy/"+e.Name()] = sha(lf(b))
	}
	for _, f := range []string{"generate.mjs", "render.mjs", "shots.mjs", "dict.mjs", "page.js"} {
		b, err := read(path.Join(GenDir, f))
		if err != nil {
			return nil, "", err
		}
		parts["generator/"+f] = sha(lf(b))
	}
	names := make([]string, 0, len(parts))
	for n := range parts {
		names = append(names, n)
	}
	sort.Strings(names)
	var all strings.Builder
	for _, n := range names {
		all.WriteString(n + "\x00" + parts[n] + "\n")
	}
	return parts, sha([]byte(all.String())), nil
}

// checkFreshness is rung 2 of the contract in the form a gate can afford (it opens no browser): the set records the
// digests of its inputs, and a difference means an input changed after the last generation. It is an advisory,
// not a failure: the fix is to run the generator, which needs a browser and the real build, so it cannot be demanded
// of every edit of the page; the release walk and the generator's own check close it (CONTEXT-HELP section 4).
func checkFreshness(r *Report, root string) {
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(HelpDir), "freshness.json"))
	if err != nil {
		return // reported as a missing shared file
	}
	var rec struct {
		Digest string            `json:"digest"`
		Inputs map[string]string `json:"inputs"`
	}
	if err := json.Unmarshal(raw, &rec); err != nil {
		r.fail("%s/freshness.json: %v", HelpDir, err)
		return
	}
	parts, digest, err := InputDigests(root)
	if err != nil {
		r.fail("the inputs of the help set cannot be read: %v", err)
		return
	}
	if digest == rec.Digest {
		return
	}
	var changed []string
	for n, h := range parts {
		if rec.Inputs[n] != h {
			changed = append(changed, n)
		}
	}
	for n := range rec.Inputs {
		if _, ok := parts[n]; !ok {
			changed = append(changed, n+" (gone)")
		}
	}
	sort.Strings(changed)
	r.advise("the help set is older than its inputs (%s): run `node tools/helpgen/generate.mjs` (CONTEXT-HELP rules 14 and 18)", strings.Join(changed, ", "))
}
