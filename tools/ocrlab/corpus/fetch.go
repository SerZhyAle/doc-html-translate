package corpus

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// fetchTimeout bounds a single download. A corpus bootstrap that hangs on one dead mirror is
// worse than one that fails and names it.
const fetchTimeout = 3 * time.Minute

// FetchResult is what happened to one scene, so the caller can summarise without re-deriving it.
type FetchResult struct {
	SceneID string
	Status  string // ok, skip, fail
	Detail  string
}

// Fetch downloads the media for the selected scenes, idempotently.
//
// Three properties matter more than speed. It is *idempotent*: a scene whose file is present
// and hashes correctly is skipped, so re-running costs nothing and repairs a partial run. It is
// *gated*: a scene whose licence no human has verified is refused, because fetching it would
// put unlicensed bytes on disk under the appearance of process. And it is *credential-free*:
// plain GET, no cookie jar, no auth - anything that needs a browser session is not a corpus
// source.
//
// A hash mismatch leaves the downloaded file next to the target with a .bad suffix rather than
// deleting it: when a source silently re-encodes its derivative, the evidence of what arrived
// is the thing you need.
func Fetch(m *Manifest, root string, ids []string, w io.Writer) ([]FetchResult, error) {
	sel, missing := m.Select("all", ids)
	if len(missing) > 0 {
		return nil, fmt.Errorf("unknown scene id(s): %v", missing)
	}
	client := &http.Client{Timeout: fetchTimeout} // no Jar: a corpus source never needs a session

	var out []FetchResult
	var failures int
	for _, s := range sel {
		r := fetchOne(client, s, root)
		out = append(out, r)
		if r.Status == "fail" {
			failures++
		}
		fmt.Fprintf(w, "%-6s %-40s %s\n", r.Status, r.SceneID, r.Detail)
	}
	if failures > 0 {
		return out, fmt.Errorf("%d of %d scene(s) failed to fetch", failures, len(sel))
	}
	return out, nil
}

func fetchOne(client *http.Client, s *Scene, root string) FetchResult {
	res := FetchResult{SceneID: s.ID}
	path := s.Path(root)

	if s.LicenceVerifiedBy == "" {
		res.Status, res.Detail = "fail", "refused: licence not verified by a human"
		return res
	}
	if !s.Licence.NeedsDownload() {
		res.Status, res.Detail = "skip", string(s.Licence)+" media is produced locally, not downloaded"
		return res
	}
	if s.SHA256 == "" {
		res.Status, res.Detail = "fail", "refused: no sha256 to verify the download against"
		return res
	}
	if got, err := HashFile(path); err == nil {
		if got == s.SHA256 {
			res.Status, res.Detail = "skip", "already present and hashes correct"
			return res
		}
		res.Status, res.Detail = "fail", "present but hash differs - delete it deliberately to re-fetch"
		return res
	}
	if s.SourceURL == "" {
		res.Status, res.Detail = "fail", "no sourceUrl"
		return res
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		res.Status, res.Detail = "fail", err.Error()
		return res
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".ocrlab-*.part")
	if err != nil {
		res.Status, res.Detail = "fail", err.Error()
		return res
	}
	tmpName := tmp.Name()

	resp, err := client.Get(s.SourceURL)
	if err != nil {
		tmp.Close()
		os.Remove(tmpName)
		res.Status, res.Detail = "fail", err.Error()
		return res
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		tmp.Close()
		os.Remove(tmpName)
		res.Status, res.Detail = "fail", fmt.Sprintf("HTTP %d from %s", resp.StatusCode, s.SourceURL)
		return res
	}
	_, copyErr := io.Copy(tmp, resp.Body)
	resp.Body.Close()
	closeErr := tmp.Close()
	if copyErr != nil || closeErr != nil {
		os.Remove(tmpName)
		res.Status, res.Detail = "fail", fmt.Sprintf("write: %v %v", copyErr, closeErr)
		return res
	}

	got, err := HashFile(tmpName)
	if err != nil {
		os.Remove(tmpName)
		res.Status, res.Detail = "fail", err.Error()
		return res
	}
	if got != s.SHA256 {
		bad := path + ".bad"
		if renameErr := os.Rename(tmpName, bad); renameErr != nil {
			os.Remove(tmpName)
		}
		res.Status = "fail"
		res.Detail = fmt.Sprintf("hash mismatch: got %s, manifest %s (kept at %s)", short(got), short(s.SHA256), bad)
		return res
	}
	// Only now, with the bytes proven, does the file take its real name.
	if err := os.Rename(tmpName, path); err != nil {
		os.Remove(tmpName)
		res.Status, res.Detail = "fail", err.Error()
		return res
	}
	res.Status, res.Detail = "ok", "downloaded and verified"
	return res
}
