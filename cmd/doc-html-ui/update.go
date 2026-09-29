package main

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

const latestReleaseAPI = "https://api.github.com/repos/SerZhyAle/doc-html-translate/releases/latest"
const releasePageBase = "https://github.com/SerZhyAle/doc-html-translate/releases/tag/"

var releaseVersion = regexp.MustCompile(`^v?(\d{2})\.(\d{4})\.(\d{4})$`)
var updateMu sync.Mutex
var updateClient = &http.Client{
	Timeout:       8 * time.Second,
	CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
}

type updateCache struct {
	FetchedAt       time.Time `json:"fetchedAt"`
	LastAutoAttempt time.Time `json:"lastAutoAttempt"`
	Tag             string    `json:"tag"`
}

type updateReply struct {
	Status  string `json:"status"`
	Version string `json:"version,omitempty"`
	URL     string `json:"url,omitempty"`
}

func updateCachePath() string {
	return filepath.Join(filepath.Dir(settingsPath()), "update-cache.json")
}

func compareReleaseVersions(a, b string) (int, bool) {
	a = strings.TrimPrefix(a, "v")
	b = strings.TrimPrefix(b, "v")
	if !releaseVersion.MatchString(a) || !releaseVersion.MatchString(b) {
		return 0, false
	}
	// Every field has a fixed width; lexical order is numeric order.
	return strings.Compare(a, b), true
}

func updateResult(tag string) updateReply {
	cmp, ok := compareReleaseVersions(tag, Version)
	if !ok {
		return updateReply{Status: "unavailable"}
	}
	if cmp <= 0 {
		return updateReply{Status: "current"}
	}
	return updateReply{Status: "newer", Version: strings.TrimPrefix(tag, "v"), URL: releasePageBase + "v" + strings.TrimPrefix(tag, "v")}
}

func fetchLatestRelease(r *http.Request) (string, error) {
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, latestReleaseAPI, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := updateClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", io.ErrUnexpectedEOF
	}
	var release struct {
		Tag string `json:"tag_name"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&release); err != nil {
		return "", err
	}
	if !releaseVersion.MatchString(release.Tag) {
		return "", io.ErrUnexpectedEOF
	}
	return release.Tag, nil
}

func handleUpdateCheck(w http.ResponseWriter, r *http.Request) {
	defer busy()()
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if isPackaged() {
		_ = json.NewEncoder(w).Encode(updateReply{Status: "store"})
		return
	}
	var input struct {
		Automatic bool `json:"automatic"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1024)).Decode(&input); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	updateMu.Lock()
	defer updateMu.Unlock()
	if input.Automatic {
		data, _ := readSettings()
		var settings struct {
			AutoUpdates bool `json:"autoUpdates"`
		}
		_ = json.Unmarshal(data, &settings)
		if !settings.AutoUpdates {
			_ = json.NewEncoder(w).Encode(updateReply{Status: "disabled"})
			return
		}
	}
	var cache updateCache
	data, _ := os.ReadFile(updateCachePath())
	_ = json.Unmarshal(data, &cache)
	now := time.Now()
	if input.Automatic && now.Sub(cache.LastAutoAttempt) < 24*time.Hour {
		if now.Sub(cache.FetchedAt) < 24*time.Hour && cache.Tag != "" {
			_ = json.NewEncoder(w).Encode(updateResult(cache.Tag))
			return
		}
		_ = json.NewEncoder(w).Encode(updateReply{Status: "unavailable"})
		return
	}
	if input.Automatic {
		cache.LastAutoAttempt = now
		// Record the attempt before reaching the network. If the per-user cache is
		// unwritable, fail closed instead of repeating an automatic request at launch.
		if err := saveUpdateCache(cache); err != nil {
			_ = json.NewEncoder(w).Encode(updateReply{Status: "unavailable"})
			return
		}
	}
	if cache.Tag == "" || now.Sub(cache.FetchedAt) >= 6*time.Hour {
		tag, err := fetchLatestRelease(r)
		if err != nil {
			_ = json.NewEncoder(w).Encode(updateReply{Status: "unavailable"})
			return
		}
		cache.Tag, cache.FetchedAt = tag, now
	}
	_ = saveUpdateCache(cache)
	_ = json.NewEncoder(w).Encode(updateResult(cache.Tag))
}

func saveUpdateCache(cache updateCache) error {
	data, err := json.Marshal(cache)
	if err != nil {
		return err
	}
	return writeFileAtomic(updateCachePath(), data, 0o600)
}
