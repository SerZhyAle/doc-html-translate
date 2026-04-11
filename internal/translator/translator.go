// Package translator provides Google Translate v2 API integration.
package translator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// googleAPIKeyFile is the filename looked up next to the executable.
const googleAPIKeyFile = "google_api.key"

const (
	apiURL         = "https://translation.googleapis.com/language/translate/v2"
	maxRetries     = 3
	requestTimeout = 30 * time.Second
	// Google recommends max ~5000 chars per request for v2 API
	maxCharsPerRequest = 5000
)

// Client defines the translation interface (for mocking in tests).
type Client interface {
	Translate(texts []string, sourceLang, targetLang string) ([]string, error)
}

// ProgressReporter is an optional interface for clients that support per-batch progress callbacks.
// done and total are segment counts (done <= total).
type ProgressReporter interface {
	SetProgress(f func(done, total int))
}

// GoogleClient is a real Google Translate v2 API client.
type GoogleClient struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// LoadGoogleAPIKey reads the API key from google_api.key located in the same
// directory as the executable. Returns an error if the file is missing or empty,
// so the caller can inform the user and skip translation gracefully.
func LoadGoogleAPIKey() (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("cannot locate executable: %w", err)
	}
	keyPath := filepath.Join(filepath.Dir(exePath), googleAPIKeyFile)
	data, err := os.ReadFile(keyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("key file not found: %s", keyPath)
		}
		return "", fmt.Errorf("read key file: %w", err)
	}
	key := strings.TrimSpace(string(data))
	if key == "" {
		return "", fmt.Errorf("key file is empty: %s", keyPath)
	}
	return key, nil
}

// NewGoogleClient creates a new Google Translate client with the given API key.
func NewGoogleClient(apiKey string) *GoogleClient {
	return &GoogleClient{
		apiKey:  apiKey,
		baseURL: apiURL,
		httpClient: &http.Client{
			Timeout: requestTimeout,
		},
	}
}

// translateRequest is the JSON body for the v2 API.
type translateRequest struct {
	Q      []string `json:"q"`
	Source string   `json:"source"`
	Target string   `json:"target"`
	Format string   `json:"format"`
}

// translateResponse is the JSON response from the v2 API.
type translateResponse struct {
	Data struct {
		Translations []struct {
			TranslatedText string `json:"translatedText"`
		} `json:"translations"`
	} `json:"data"`
	Error *apiError `json:"error,omitempty"`
}

type apiError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Translate sends texts to Google Translate v2 and returns translated texts.
// Handles batching, retries with exponential backoff for 429/5xx errors.
func (c *GoogleClient) Translate(texts []string, sourceLang, targetLang string) ([]string, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	// Batch texts to respect the char limit
	batches := batchTexts(texts, maxCharsPerRequest)
	var allResults []string

	for _, batch := range batches {
		results, err := c.translateBatch(batch, sourceLang, targetLang)
		if err != nil {
			return nil, err
		}
		allResults = append(allResults, results...)
	}

	return allResults, nil
}

// translateBatch sends a single batch with retries.
func (c *GoogleClient) translateBatch(texts []string, sourceLang, targetLang string) ([]string, error) {
	reqBody := translateRequest{
		Q:      texts,
		Source: sourceLang,
		Target: targetLang,
		Format: "html",
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s?key=%s", c.baseURL, c.apiKey)

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 1s, 2s, 4s
			delay := time.Duration(math.Pow(2, float64(attempt-1))) * time.Second
			time.Sleep(delay)
		}

		req, err := http.NewRequest("POST", url, bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("http request: %w", err)
			continue
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("read response: %w", err)
			continue
		}

		// Retry on 429 (rate limit) and 5xx (server errors)
		if resp.StatusCode == 429 || resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("API returned %d: %s", resp.StatusCode, string(respBody))
			continue
		}

		if resp.StatusCode != 200 {
			return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
		}

		var result translateResponse
		if err := json.Unmarshal(respBody, &result); err != nil {
			return nil, fmt.Errorf("unmarshal response: %w", err)
		}

		if result.Error != nil {
			return nil, fmt.Errorf("API error %d: %s", result.Error.Code, result.Error.Message)
		}

		translated := make([]string, len(result.Data.Translations))
		for i, t := range result.Data.Translations {
			translated[i] = t.TranslatedText
		}
		return translated, nil
	}

	return nil, fmt.Errorf("all %d retries failed: %w", maxRetries, lastErr)
}

// batchTexts splits texts into batches respecting the character limit.
func batchTexts(texts []string, maxChars int) [][]string {
	var batches [][]string
	var currentBatch []string
	currentLen := 0

	for _, t := range texts {
		tLen := len(t)

		// If a single text exceeds the limit, it goes alone in its own batch
		if tLen > maxChars {
			if len(currentBatch) > 0 {
				batches = append(batches, currentBatch)
				currentBatch = nil
				currentLen = 0
			}
			batches = append(batches, []string{t})
			continue
		}

		if currentLen+tLen > maxChars && len(currentBatch) > 0 {
			batches = append(batches, currentBatch)
			currentBatch = nil
			currentLen = 0
		}

		currentBatch = append(currentBatch, t)
		currentLen += tLen
	}

	if len(currentBatch) > 0 {
		batches = append(batches, currentBatch)
	}

	return batches
}
