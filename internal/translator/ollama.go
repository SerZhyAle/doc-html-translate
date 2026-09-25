package translator

import (
	"bytes"
	"context"
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
	ollamaDefaultURL    = "http://localhost:11434/api/generate"
	ollamaDefaultModel  = "gemma3:12b"
	ollamaDefaultNumCtx = 8192 // far below default 128K; our batches need <4K tokens
	// ollamaRequestTimeout bounds one request once the model is in memory.
	ollamaRequestTimeout = 300 * time.Second
	// ollamaLoadTimeout bounds the requests sent before the model has answered once: a cold
	// load of a large model from disk can take minutes on its own, before any token is produced.
	ollamaLoadTimeout = 15 * time.Minute
	// Smaller batch keeps the model focused and reduces echo-back failures
	ollamaBatchSize = 20
	// Max retry passes for segments that came back untranslated (echo)
	ollamaMaxRetries = 2
)

// OllamaClient translates text using a local Ollama instance.
type OllamaClient struct {
	baseURL     string
	model       string
	numCtx      int
	parallelism int
	httpClient  *http.Client
	onProgress  func(done, total int) // optional - called after each batch completes
	// first-request detection (thread-safe)
	firstMu   sync.Mutex
	firstDone bool
	ready     atomic.Bool // the model has answered once, so it is loaded
}

// SetProgress implements ProgressReporter. f is called after each batch with (done, total) segment counts.
func (c *OllamaClient) SetProgress(f func(done, total int)) {
	c.onProgress = f
}

// Unload asks Ollama to release the model from VRAM (keep_alive: 0).
// It uses its own short timeout, so it works after the run's context was cancelled.
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
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	if resp, err := c.httpClient.Do(req); err == nil {
		_ = resp.Body.Close()
	}
}

// NewOllamaClient creates a client pointing at local Ollama. Timeouts are per request (see
// call), not on the http.Client, because the first request needs a much longer one.
func NewOllamaClient(model string) *OllamaClient {
	if model == "" {
		model = ollamaDefaultModel
	}
	return &OllamaClient{
		baseURL:     ollamaDefaultURL,
		model:       model,
		numCtx:      ollamaDefaultNumCtx,
		parallelism: 1,
		httpClient:  &http.Client{},
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

// ollamaJob is one request's worth of texts: a numbered batch of one-line texts, or a single
// text sent on its own.
type ollamaJob struct {
	idx    []int // positions in the caller's slice
	single bool
}

// planJobs groups texts into requests. A text containing a line break is sent on its own: the
// numbered-list prompt is one line per text, so its second line came back as an unnumbered line
// the parser dropped - or, if it began with a number ("1984. It was.."), as another slot's answer.
func planJobs(texts []string) []ollamaJob {
	var jobs []ollamaJob
	var batch []int
	for i, t := range texts {
		if strings.ContainsAny(t, "\r\n") {
			jobs = append(jobs, ollamaJob{idx: []int{i}, single: true})
			continue
		}
		batch = append(batch, i)
		if len(batch) == ollamaBatchSize {
			jobs = append(jobs, ollamaJob{idx: batch})
			batch = nil
		}
	}
	if len(batch) > 0 {
		jobs = append(jobs, ollamaJob{idx: batch})
	}
	return jobs
}

// Translate implements the Client interface using Ollama.
// Batches are sent concurrently (up to c.parallelism) for better GPU utilization
// when OLLAMA_NUM_PARALLEL is set accordingly.
//
// A batch that fails does not discard the batches that succeeded: their translations come back
// with a *PartialError naming the untranslated slots. No new batch starts after a failure or a
// cancelled ctx.
func (c *OllamaClient) Translate(ctx context.Context, texts []string, sourceLang, targetLang string) ([]string, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	jobs := planJobs(texts)
	results := make([]string, len(texts))
	done := make([]bool, len(jobs)) // each goroutine writes only its own job's slots

	var (
		firstErr error
		errMu    sync.Mutex
		doneSegs int64
		wg       sync.WaitGroup
	)
	sem := make(chan struct{}, c.parallelism)

	for j, job := range jobs {
		errMu.Lock()
		abort := firstErr != nil
		errMu.Unlock()
		if abort || ctx.Err() != nil {
			break
		}

		sem <- struct{}{} // acquire concurrency slot
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			translated, err := c.runJob(ctx, texts, job, sourceLang, targetLang)
			if err != nil {
				errMu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				errMu.Unlock()
				return
			}
			for k, i := range job.idx {
				results[i] = translated[k]
			}
			done[j] = true

			if c.onProgress != nil {
				n := int(atomic.AddInt64(&doneSegs, int64(len(job.idx))))
				c.onProgress(n, len(texts))
			}
		}()
	}

	wg.Wait()

	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if firstErr == nil {
		return results, nil
	}
	var missing []int
	for j, job := range jobs {
		if !done[j] {
			missing = append(missing, job.idx...)
		}
	}
	if len(missing) == len(texts) {
		return nil, firstErr
	}
	return results, &PartialError{Missing: missing, Err: firstErr}
}

// runJob translates one job, then retries its echo-backs one text at a time.
func (c *OllamaClient) runJob(ctx context.Context, texts []string, job ollamaJob, src, dst string) ([]string, error) {
	jobTexts := make([]string, len(job.idx))
	for k, i := range job.idx {
		jobTexts[k] = texts[i]
	}

	var translated []string
	if job.single {
		t, err := c.translateSingle(ctx, jobTexts[0], src, dst)
		if err != nil {
			return nil, err
		}
		translated = []string{t}
	} else {
		var err error
		if translated, err = c.translateBatch(ctx, jobTexts, src, dst); err != nil {
			return nil, err
		}
	}

	// Retry echo-backs with a simpler single-item prompt.
	for attempt := 0; attempt < ollamaMaxRetries; attempt++ {
		anyRetried := false
		for k, orig := range jobTexts {
			if !isEchoBack(translated[k], orig) {
				continue
			}
			retried, err := c.translateSingle(ctx, orig, src, dst)
			if err != nil {
				break
			}
			if !isEchoBack(retried, orig) {
				translated[k] = retried
				anyRetried = true
			}
		}
		if !anyRetried {
			break
		}
	}
	return translated, nil
}

// translateSingle translates one text string using a simple, direct prompt, keeping its line
// breaks. Used for texts that span lines and for retry passes where the numbered-batch format
// failed.
func (c *OllamaClient) translateSingle(ctx context.Context, text, srcLang, dstLang string) (string, error) {
	prompt := fmt.Sprintf(
		"Translate the following text from %s to %s.\n"+
			"Output ONLY the translation. Keep the line breaks. Do not add explanations or repeat the original.\n\n%s",
		langName(srcLang), langName(dstLang), text,
	)
	out, err := c.generate(ctx, prompt)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
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

func (c *OllamaClient) translateBatch(ctx context.Context, texts []string, srcLang, dstLang string) ([]string, error) {
	// Build numbered list prompt
	var sb strings.Builder
	fmt.Fprintf(&sb,
		"Translate each line from %s to %s.\n"+
			"Rules:\n"+
			"- Output ONLY the translated lines, numbered exactly as input.\n"+
			"- NEVER leave a line in the original language.\n"+
			"- Do NOT add explanations, comments, or extra text.\n"+
			"- If a line contains quoted speech, translate the speech too.\n\n",
		langName(srcLang), langName(dstLang),
	)
	for i, t := range texts {
		fmt.Fprintf(&sb, "%d. %s\n", i+1, t)
	}
	out, err := c.generate(ctx, sb.String())
	if err != nil {
		return nil, err
	}
	return parseNumberedResponse(out, len(texts)), nil
}

// generate runs one prompt and returns the model's answer.
func (c *OllamaClient) generate(ctx context.Context, prompt string) (string, error) {
	body, err := json.Marshal(ollamaRequest{
		Model:   c.model,
		Prompt:  prompt,
		Stream:  false,
		Options: ollamaOptions{NumCtx: c.numCtx, Temperature: 0},
	})
	if err != nil {
		return "", fmt.Errorf("ollama: marshal request: %w", err)
	}

	isFirst := false
	c.firstMu.Lock()
	if !c.firstDone {
		c.firstDone = true
		isFirst = true
	}
	c.firstMu.Unlock()
	if isFirst {
		logging.Printf("  Loading model %s into VRAM..\n", c.model)
	}

	t0 := time.Now()
	status, respBody, err := c.call(ctx, body)
	if isFirst {
		logging.Printf("  Model ready in %s\n", formatLoadTime(time.Since(t0)))
	}
	if err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", fmt.Errorf("ollama: http request: %w (is Ollama running?)", err)
	}
	if status != http.StatusOK {
		return "", fmt.Errorf("ollama: HTTP %d: %s", status, strings.TrimSpace(string(respBody)))
	}
	var result ollamaResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("ollama: unmarshal response: %w", err)
	}
	if result.Error != "" {
		return "", fmt.Errorf("ollama: model error: %s", result.Error)
	}
	c.ready.Store(true)
	return result.Response, nil
}

// call sends one request under its own timeout: the long one until the model has answered
// once, the ordinary one after. The body is read before the timeout's context is released.
func (c *OllamaClient) call(ctx context.Context, body []byte) (int, []byte, error) {
	timeout := ollamaRequestTimeout
	if !c.ready.Load() {
		timeout = ollamaLoadTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, err
	}
	return resp.StatusCode, respBody, nil
}

// isEchoBack returns true if the model returned the original text unchanged.
func isEchoBack(translated, original string) bool {
	if translated == "" {
		return true
	}
	t := strings.TrimSpace(translated)
	o := strings.TrimSpace(original)
	if len(o) < 10 {
		return false // short strings - don't retry
	}
	return strings.EqualFold(t, o)
}

// numberedLineRe matches one "N. text" answer line of the model output. The blanks around the
// number are spaces and tabs only: \s would also cross a line break, so an empty "2." answer
// took the next line ("3. text") as its own and slot 3 went missing.
var numberedLineRe = regexp.MustCompile(`(?m)^[ \t]*(\d+)\.[ \t]*(.+)$`)

// parseNumberedResponse maps "N. text" lines onto the expected slots. The first answer for a
// number wins and numbers outside 1..expected are ignored, so a translated line that itself
// starts with a number ("1984. It was..") can neither overwrite an earlier slot nor invent one.
// A slot with no answer stays empty and the caller keeps the original text.
func parseNumberedResponse(response string, expected int) []string {
	results := make([]string, expected)
	filled := make([]bool, expected)
	for _, m := range numberedLineRe.FindAllStringSubmatch(response, -1) {
		n, err := strconv.Atoi(m[1])
		if err != nil || n < 1 || n > expected || filled[n-1] {
			continue
		}
		results[n-1] = strings.TrimSpace(m[2])
		filled[n-1] = true
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
