package main

import (
	"os"
	"path/filepath"
	"time"
)

func preserveScores(dir string) error {
	if _, err := os.Stat(filepath.Join(dir, "scores.json")); os.IsNotExist(err) {
		return nil
	}
	archive := filepath.Join(dir, "score-history", time.Now().UTC().Format("20060102T150405.000000000"))
	if err := os.MkdirAll(archive, 0755); err != nil {
		return err
	}
	for _, name := range []string{"scores.json", "summary.json", "selftest.json", "gate.json"} {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(archive, name), data, 0644); err != nil {
			return err
		}
	}
	return nil
}
