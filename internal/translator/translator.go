// Package translator provides the translation engines: Google Cloud Translation v2 and a local
// Ollama model, behind one Client interface.
package translator

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// googleAPIKeyFile is the filename of the Google API key.
const googleAPIKeyFile = "google_api.key"

// appDataDirName is the per-user folder under %LOCALAPPDATA% where a writable
// copy of the key may live. Needed for the Microsoft Store (MSIX) build, whose
// install directory is read-only, so the key cannot sit next to the executable.
const appDataDirName = "doc-html-translate"

// Client defines the translation interface (for mocking in tests). ctx cancels the work:
// requests in flight are aborted and no new one is started.
type Client interface {
	Translate(ctx context.Context, texts []string, sourceLang, targetLang string) ([]string, error)
}

// PartialError is returned together with a full-length result slice when only some texts were
// translated. Missing lists the indexes whose slot is empty because their request failed; every
// other slot holds a real translation. A caller that can use part of a page keeps what did
// arrive instead of discarding it.
type PartialError struct {
	Missing []int
	Err     error
}

func (e *PartialError) Error() string {
	return fmt.Sprintf("%d text(s) untranslated: %v", len(e.Missing), e.Err)
}

func (e *PartialError) Unwrap() error { return e.Err }

// ProgressReporter is an optional interface for clients that support per-batch progress callbacks.
// done and total are segment counts (done <= total).
type ProgressReporter interface {
	SetProgress(f func(done, total int))
}

// GoogleAPIKeyPaths returns the candidate locations for google_api.key, in the
// order they are tried: next to the executable first (the unpackaged build), then
// %LOCALAPPDATA%\doc-html-translate\google_api.key (a writable per-user path that
// also works under the read-only Microsoft Store/MSIX install directory).
func GoogleAPIKeyPaths() []string {
	var paths []string
	if exePath, err := os.Executable(); err == nil {
		paths = append(paths, filepath.Join(filepath.Dir(exePath), googleAPIKeyFile))
	}
	if appData := os.Getenv("LOCALAPPDATA"); appData != "" {
		paths = append(paths, filepath.Join(appData, appDataDirName, googleAPIKeyFile))
	}
	return paths
}

// LoadGoogleAPIKey reads the API key from the first of GoogleAPIKeyPaths that
// holds a non-empty key. Returns an error if no location has a usable key, so the
// caller can inform the user and skip translation gracefully.
func LoadGoogleAPIKey() (string, error) {
	candidates := GoogleAPIKeyPaths()
	if len(candidates) == 0 {
		return "", fmt.Errorf("cannot locate executable or %%LOCALAPPDATA%%")
	}
	var lastErr error
	for _, keyPath := range candidates {
		data, err := os.ReadFile(keyPath)
		if err != nil {
			if os.IsNotExist(err) {
				lastErr = fmt.Errorf("key file not found: %s", keyPath)
				continue
			}
			return "", fmt.Errorf("read key file: %w", err)
		}
		key := strings.TrimSpace(string(data))
		if key == "" {
			lastErr = fmt.Errorf("key file is empty: %s", keyPath)
			continue
		}
		return key, nil
	}
	return "", lastErr
}
