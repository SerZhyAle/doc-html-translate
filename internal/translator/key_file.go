package translator

import (
	"fmt"
	"os"
	"path/filepath"
)

func writeKeyFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create key folder: %w", err)
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".google-key-*")
	if err != nil {
		return fmt.Errorf("create key file: %w", err)
	}
	defer os.Remove(f.Name())
	if err := f.Chmod(0o600); err != nil {
		f.Close()
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(f.Name(), path); err != nil {
		return fmt.Errorf("replace key file: %w", err)
	}
	return nil
}
