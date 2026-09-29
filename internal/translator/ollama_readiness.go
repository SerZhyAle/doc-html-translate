package translator

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// OllamaModelAvailable checks the local model catalog without loading a model or sending text.
func OllamaModelAvailable(ctx context.Context, model string) (bool, error) {
	return ollamaModelAvailableAt(ctx, model, "http://localhost:11434/api/tags")
}

func ollamaModelAvailableAt(ctx context.Context, model, url string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, err
	}
	client := &http.Client{Timeout: 2 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("Ollama returned HTTP %d", resp.StatusCode)
	}
	var catalog struct {
		Models []struct {
			Name  string `json:"name"`
			Model string `json:"model"`
		} `json:"models"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&catalog); err != nil {
		return false, err
	}
	if strings.TrimSpace(model) == "" {
		model = ollamaDefaultModel
	}
	alias := model
	if !strings.Contains(model, ":") {
		alias += ":latest"
	}
	for _, item := range catalog.Models {
		if item.Name == model || item.Model == model || item.Name == alias || item.Model == alias {
			return true, nil
		}
	}
	return false, nil
}
