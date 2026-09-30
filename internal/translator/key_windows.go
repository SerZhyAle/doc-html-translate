package translator

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unsafe"

	"doc-html-translate/internal/logging"
	"golang.org/x/sys/windows"
)

const protectedKeyFile = "google_api.key.dpapi"

// GoogleAPIKeyPath is the protected per-user file, including under MSIX.
func GoogleAPIKeyPath() string {
	if appData := os.Getenv("LOCALAPPDATA"); appData != "" {
		return filepath.Join(appData, appDataDirName, protectedKeyFile)
	}
	return ""
}

func LoadGoogleAPIKey() (string, error) {
	path := GoogleAPIKeyPath()
	if path == "" {
		return "", fmt.Errorf("cannot locate %%LOCALAPPDATA%%")
	}
	data, err := os.ReadFile(path)
	if err == nil {
		key, err := unprotectKey(data)
		if err != nil || strings.TrimSpace(key) == "" {
			return "", ErrGoogleKeyUnusable
		}
		return key, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("read protected key: %w", err)
	}
	key, oldPath, err := loadPlainGoogleAPIKey()
	if err != nil {
		return "", err
	}
	if err := SaveGoogleAPIKey(key); err != nil {
		return "", fmt.Errorf("migrate Google API key: %w", err)
	}
	if err := os.Remove(oldPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("remove migrated plaintext key: %w", err)
	}
	logging.Printf("Migrated Google API key to Windows DPAPI storage; removed plaintext key file.\n")
	return key, nil
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
	data, err := protectKey(key)
	if err != nil {
		return fmt.Errorf("protect Google API key: %w", err)
	}
	if err := writeKeyFile(path, data); err != nil {
		return err
	}
	// A previous GUI save may have left plaintext in the per-user directory.
	legacy := filepath.Join(filepath.Dir(path), googleAPIKeyFile)
	if err := os.Remove(legacy); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove plaintext key: %w", err)
	}
	return nil
}

func protectKey(key string) ([]byte, error) {
	input := []byte(key)
	in := windows.DataBlob{Size: uint32(len(input)), Data: &input[0]}
	var out windows.DataBlob
	if err := windows.CryptProtectData(&in, nil, nil, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &out); err != nil {
		return nil, err
	}
	defer func() { _, _ = windows.LocalFree(windows.Handle(unsafe.Pointer(out.Data))) }()
	return append([]byte(nil), unsafe.Slice(out.Data, out.Size)...), nil
}

func unprotectKey(data []byte) (string, error) {
	if len(data) == 0 {
		return "", ErrGoogleKeyUnusable
	}
	in := windows.DataBlob{Size: uint32(len(data)), Data: &data[0]}
	var out windows.DataBlob
	if err := windows.CryptUnprotectData(&in, nil, nil, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &out); err != nil {
		return "", err
	}
	defer func() { _, _ = windows.LocalFree(windows.Handle(unsafe.Pointer(out.Data))) }()
	return string(unsafe.Slice(out.Data, out.Size)), nil
}
