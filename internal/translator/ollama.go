package translator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"doc-html-translate/internal/logging"
)

const (
	ollamaDefaultURL     = "http://localhost:11434/api/generate"
	ollamaDefaultModel   = "gemma3:12b"
	ollamaDefaultNumCtx  = 8192 // far below default 128K; our batches need <4K tokens
	ollamaTimeout        = 300 * time.Second
	// Smaller batch keeps the model focused and reduces echo-back failures
	ollamaBatchSize      = 20
	// Max retry passes for segments that came back untranslated (echo)
	ollamaMaxRetries     = 2
)

// OllamaClient translates text using a local Ollama instance.
type OllamaClient struct {
	baseURL     string
	model       string
	numCtx      int
	parallelism int
	httpClient  *http.Client
	onProgress  func(done, total int) // optional — called after each batch completes
	// first-request detection (thread-safe)
	firstMu   sync.Mutex
	firstDone bool
	loadStart time.Time
}

// SetProgress implements ProgressReporter. f is called after each batch with (done, total) segment counts.
func (c *OllamaClient) SetProgress(f func(done, total int)) {
	c.onProgress = f
}

// Unload asks Ollama to release the model from VRAM (keep_alive: 0).
// Safe to call from a signal handler — uses a short 5-second timeout.
func (c *OllamaClient) Unload() {
	body, err := json.Marshal(map[string]interface{}{
		"model":      c.model,
		"prompt":     "",
		"stream":     false,
		"keep_alive": 0,
	})
	if err != nil {
		return
	}
	req, err := http.NewRequest("POST", c.baseURL, bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err == nil {
		resp.Body.Close()
	}
}

// NewOllamaClient creates a client pointing at local Ollama.
func NewOllamaClient(model string) *OllamaClient {
	if model == "" {
		model = ollamaDefaultModel
	}
	return &OllamaClient{
		baseURL:     ollamaDefaultURL,
		model:       model,
		numCtx:      ollamaDefaultNumCtx,
		parallelism: 1,
		httpClient: &http.Client{
			Timeout: ollamaTimeout,
		},
	}
}

// SetParallelism sets how many batch requests are sent to Ollama concurrently.
// Set OLLAMA_NUM_PARALLEL env var to the same value before starting Ollama.
func (c *OllamaClient) SetParallelism(n int) {
	if n < 1 {
		n = 1
	}
	c.parallelism = n
}

// SetNumCtx overrides the context window size sent to Ollama (tokens).
// Smaller values = faster inference. Default 8192 is safe for batches of 20 segments.
func (c *OllamaClient) SetNumCtx(n int) {
	if n < 512 {
		n = 512
	}
	c.numCtx = n
}

// Translate implements the Client interface using Ollama.
// Batches are sent concurrently (up to c.parallelism) for better GPU utilization
// when OLLAMA_NUM_PARALLEL is set accordingly.
func (c *OllamaClient) Translate(texts []string, sourceLang, targetLang string) ([]string, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	numBatches := (len(texts) + ollamaBatchSize - 1) / ollamaBatchSize
	batchResults := make([][]string, numBatches) // indexed by batch number, no races

	var (
		firstErr error
		errMu    sync.Mutex
		doneSegs int64
		wg       sync.WaitGroup
	)

	sem := make(chan struct{}, c.parallelism)

	for b := 0; b < numBatches; b++ {
		// Early abort if a previous batch failed.
		errMu.Lock()
		abort := firstErr != nil
		errMu.Unlock()
		if abort {
			break
		}

		start := b * ollamaBatchSize
		end := start + ollamaBatchSize
		if end > len(texts) {
			end = len(texts)
		}
		batchTexts := texts[start:end]
		bIdx, bStart, bEnd := b, start, end

		sem <- struct{}{} // acquire concurrency slot
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			translated, err := c.translateBatch(batchTexts, sourceLang, targetLang)
			if err != nil {
				errMu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				errMu.Unlock()
				return
			}

			// Retry echo-backs with a simpler single-item prompt.
			for attempt := 0; attempt < ollamaMaxRetries; attempt++ {
				anyRetried := false
				for j, orig := range batchTexts {
					if !isEchoBack(translated[j], orig) {
						continue
					}
					retried, err := c.translateSingle(orig, sourceLang, targetLang)
					if err != nil {
						break
					}
					if !isEchoBack(retried, orig) {
						translated[j] = retried
						anyRetried = true
					}
				}
				if !anyRetried {
					break
				}
			}

			batchResults[bIdx] = translated

			if c.onProgress != nil {
				done := int(atomic.AddInt64(&doneSegs, int64(bEnd-bStart)))
				c.onProgress(done, len(texts))
			}
		}()
	}

	wg.Wait()

	if firstErr != nil {
		return nil, firstErr
	}

	results := make([]string, len(texts))
	for b, tr := range batchResults {
		if tr == nil {
			continue
		}
		copy(results[b*ollamaBatchSize:], tr)
	}
	return results, nil
}

// translateSingle translates one text string using a simple, direct prompt.
// Used for retry passes where the numbered-batch format failed.
func (c *OllamaClient) translateSingle(text, srcLang, dstLang string) (string, error) {
	prompt := fmt.Sprintf(
		"Translate the following text from %s to %s.\n"+
			"Output ONLY the translation. Do not add explanations or repeat the original.\n\n%s",
		langName(srcLang), langName(dstLang), text,
	)
	reqBody := ollamaRequest{
		Model:  c.model,
		Prompt: prompt,
		Stream: false,
		Options: ollamaOptions{NumCtx: c.numCtx, Temperature: 0},
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}
	resp, err := c.httpClient.Post(c.baseURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("ollama: HTTP %d", resp.StatusCode)
	}
	var result ollamaResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", err
	}
	if result.Error != "" {
		return "", fmt.Errorf("ollama: %s", result.Error)
	}
	return strings.TrimSpace(result.Response), nil
}

// isEchoBack returns true if the model returned the original text unchanged.
func isEchoBack(translated, original string) bool {
	if translated == "" {
		return true
	}
	t := strings.TrimSpace(translated)
	o := strings.TrimSpace(original)
	if len(o) < 10 {
		return false // short strings — don't retry
	}
	return strings.EqualFold(t, o)
}

type ollamaOptions struct {
	NumCtx      int     `json:"num_ctx"`
	Temperature float64 `json:"temperature"` // 0 = greedy (deterministic, slightly faster)
}

type ollamaRequest struct {
	Model   string        `json:"model"`
	Prompt  string        `json:"prompt"`
	Stream  bool          `json:"stream"`
	Options ollamaOptions `json:"options"`
}

type ollamaResponse struct {
	Response string `json:"response"`
	Error    string `json:"error,omitempty"`
}

func (c *OllamaClient) translateBatch(texts []string, srcLang, dstLang string) ([]string, error) {
	// Build numbered list prompt
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(
		"Translate each line from %s to %s.\n"+
			"Rules:\n"+
			"- Output ONLY the translated lines, numbered exactly as input.\n"+
			"- NEVER leave a line in the original language.\n"+
			"- Do NOT add explanations, comments, or extra text.\n"+
			"- If a line contains quoted speech, translate the speech too.\n\n",
		langName(srcLang), langName(dstLang),
	))
	for i, t := range texts {
		fmt.Fprintf(&sb, "%d. %s\n", i+1, t)
	}

	reqBody := ollamaRequest{
		Model:   c.model,
		Prompt:  sb.String(),
		Stream:  false,
		Options: ollamaOptions{NumCtx: c.numCtx, Temperature: 0},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("ollama: marshal request: %w", err)
	}

	// Thread-safe first-request load detection.
	isFirst := false
	c.firstMu.Lock()
	if !c.firstDone {
		c.firstDone = true
		isFirst = true
		c.loadStart = time.Now()
	}
	c.firstMu.Unlock()
	if isFirst {
		logging.Printf("  Loading model %s into VRAM...\n", c.model)
	}

	t0 := time.Now()
	resp, err := c.httpClient.Post(c.baseURL, "application/json", bytes.NewReader(body))
	if isFirst {
		logging.Printf("  Model ready in %s\n", formatLoadTime(time.Since(t0)))
	}
	if err != nil {
		return nil, fmt.Errorf("ollama: http request: %w (is Ollama running?)", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ollama: read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("ollama: HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var result ollamaResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("ollama: unmarshal response: %w", err)
	}
	if result.Error != "" {
		return nil, fmt.Errorf("ollama: model error: %s", result.Error)
	}

	return parseNumberedResponse(result.Response, len(texts)), nil
}

// parseNumberedResponse extracts "N. text" lines from the model output.
// Falls back to full response split by newlines if parsing fails.
var numberedLineRe = regexp.MustCompile(`(?m)^\s*(\d+)\.\s*(.+)$`)

func parseNumberedResponse(response string, expected int) []string {
	matches := numberedLineRe.FindAllStringSubmatch(response, -1)

	// Build map by number
	parsed := make(map[int]string, len(matches))
	for _, m := range matches {
		n, err := strconv.Atoi(m[1])
		if err == nil && n >= 1 && n <= expected {
			parsed[n] = strings.TrimSpace(m[2])
		}
	}

	results := make([]string, expected)
	for i := range results {
		if v, ok := parsed[i+1]; ok {
			results[i] = v
		} else {
			// Missing number — use empty string (original will be kept by caller)
			results[i] = ""
		}
	}
	return results
}

// langName returns a human-readable language name for the prompt.
func langName(code string) string {
	m := map[string]string{
		"en": "English",
		"ru": "Russian",
		"de": "German",
		"fr": "French",
		"es": "Spanish",
		"zh": "Chinese",
		"ja": "Japanese",
		"ko": "Korean",
		"it": "Italian",
		"pt": "Portuguese",
		"pl": "Polish",
		"uk": "Ukrainian",
	}
	if name, ok := m[code]; ok {
		return name
	}
	return code
}

// formatLoadTime formats a duration for the "Model ready in X" message.
func formatLoadTime(d time.Duration) string {
	if d < time.Second {
		return "< 1s (already loaded)"
	}
	s := int(d.Seconds())
	if s < 60 {
		return fmt.Sprintf("%ds", s)
	}
	return fmt.Sprintf("%dm%ds", s/60, s%60)
}
