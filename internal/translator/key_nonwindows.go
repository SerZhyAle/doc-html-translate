//go:build !windows

package translator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// GoogleAPIKeyPath is the writable per-user location on this platform.
func GoogleAPIKeyPath() string {
	if appData := os.Getenv("LOCALAPPDATA"); appData != "" {
		return filepath.Join(appData, appDataDirName, googleAPIKeyFile)
	}
	paths := GoogleAPIKeyPaths()
	if len(paths) > 0 {
		return paths[len(paths)-1]
	}
	return ""
}

func LoadGoogleAPIKey() (string, error) {
	key, _, err := loadPlainGoogleAPIKey()
	return key, err
}

func SaveGoogleAPIKey(key string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return fmt.Errorf("key is empty")
	}
	path := GoogleAPIKeyPath()
	if path == "" {
		return fmt.Errorf("cannot determine a writable key location")
	}
	return writeKeyFile(path, []byte(key))
}
