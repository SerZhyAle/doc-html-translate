package translator

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	apiURL         = "https://translation.googleapis.com/language/translate/v2"
	requestTimeout = 30 * time.Second
	// maxCharsPerRequest is Google's recommended request size for v2, in characters.
	maxCharsPerRequest = 5000
	// maxSegmentsPerRequest is v2's hard limit on q values per request: a page of many short
	// strings went over it, and the 400 that came back stopped the whole book.
	maxSegmentsPerRequest = 128
	// maxResponseBytes bounds what is read from one reply; a v2 reply to a 5000-character
	// request is tens of kilobytes.
	maxResponseBytes = 16 << 20
)

// GoogleClient is a real Google Translate v2 API client.
type GoogleClient struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	retry      retryPolicy
}

// NewGoogleClient creates a new Google Translate client with the given API key.
func NewGoogleClient(apiKey string) *GoogleClient {
	return &GoogleClient{
		apiKey:     apiKey,
		baseURL:    apiURL,
		httpClient: &http.Client{Timeout: requestTimeout},
		retry:      defaultRetryPolicy(),
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
	Errors  []struct {
		Reason string `json:"reason"`
	} `json:"errors"`
}

// Translate sends texts to Google Translate v2 and returns one translation per text, in order.
// Requests are bounded by segment count and size; throttling is waited out within the retry
// budget; a reply whose shape does not match its request is an error, never a shifted result.
//
// When a later batch fails, the batches already answered are billed, so they come back with a
// *PartialError naming the slots of the failed batch and every one after it.
func (c *GoogleClient) Translate(ctx context.Context, texts []string, sourceLang, targetLang string) ([]string, error) {
	if len(texts) == 0 {
		return nil, nil
	}
	out := make([]string, 0, len(texts))
	for _, batch := range batchTexts(texts, maxCharsPerRequest, maxSegmentsPerRequest) {
		results, err := c.translateBatch(ctx, batch, sourceLang, targetLang)
		if err != nil {
			if len(out) == 0 || ctx.Err() != nil {
				return nil, err
			}
			missing := make([]int, 0, len(texts)-len(out))
			for i := len(out); i < len(texts); i++ {
				missing = append(missing, i)
			}
			return append(out, make([]string, len(missing))...), &PartialError{Missing: missing, Err: err}
		}
		out = append(out, results...)
	}
	return out, nil
}

// translateBatch sends one request's worth of texts.
//
// The texts are plain DOM text, but the request says format=html: in text mode the provider
// may treat line structure and entities differently, while an escaped html request comes back
// with markup characters escaped the same way. So each text is HTML-escaped on the way out and
// the reply is unescaped on the way back - "It's" does not come back as "It&#39;s", and "a<b"
// survives instead of being read as a tag.
func (c *GoogleClient) translateBatch(ctx context.Context, texts []string, sourceLang, targetLang string) ([]string, error) {
	escaped := make([]string, len(texts))
	for i, t := range texts {
		escaped[i] = html.EscapeString(t)
	}
	body, err := json.Marshal(translateRequest{Q: escaped, Source: sourceLang, Target: targetLang, Format: "html"})
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	budget := c.retry.start()
	shapeRetried := false
	for attempt := 1; ; attempt++ {
		translated, advice, err := c.send(ctx, body)
		if err == nil && len(translated) != len(texts) {
			err = fmt.Errorf("reply holds %d translations for %d texts", len(translated), len(texts))
			if !shapeRetried {
				// A malformed reply is retried once, at once: it is not throttling, and a second
				// identical answer means the request itself is the problem.
				shapeRetried = true
				continue
			}
			return nil, fmt.Errorf("google translate: %w", err)
		}
		if err == nil {
			for i, t := range translated {
				translated[i] = html.UnescapeString(t)
			}
			return translated, nil
		}
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if !advice.retryable {
			return nil, fmt.Errorf("google translate: %w", err)
		}
		if werr := budget.wait(ctx, attempt, advice.retryAfter); werr != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			return nil, fmt.Errorf("google translate: %w (%v)", err, werr)
		}
	}
}

// retryAdvice is what one failed exchange says about trying again.
type retryAdvice struct {
	retryable  bool
	retryAfter time.Duration // the server's Retry-After, 0 when it gave none
}

// send makes one request. Its errors never carry the request URL or the key: a transport error
// is reduced to its cause before it is wrapped, because every layer above logs it.
func (c *GoogleClient) send(ctx context.Context, body []byte) ([]string, retryAdvice, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(body))
	if err != nil {
		return nil, retryAdvice{}, errors.New("create request: invalid endpoint")
	}
	req.Header.Set("Content-Type", "application/json")
	// The key travels in a header, never in the URL: URLs end up in errors, proxy logs and
	// crash dumps, and the run log is kept on disk.
	req.Header.Set("X-Goog-Api-Key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, retryAdvice{retryable: true}, fmt.Errorf("http request: %s", c.scrub(transportCause(err)))
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return nil, retryAdvice{retryable: true}, fmt.Errorf("read response: %s", c.scrub(transportCause(err)))
	}

	if resp.StatusCode != http.StatusOK {
		advice := retryAdvice{retryAfter: parseRetryAfter(resp.Header.Get("Retry-After"), c.retry.now())}
		advice.retryable = resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 ||
			(resp.StatusCode == http.StatusForbidden && isRateLimitReason(respBody))
		return nil, advice, fmt.Errorf("API error %d: %s", resp.StatusCode, c.scrub(errorText(respBody)))
	}

	var result translateResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, retryAdvice{}, fmt.Errorf("unmarshal response: %w", err)
	}
	if result.Error != nil {
		return nil, retryAdvice{}, fmt.Errorf("API error %d: %s", result.Error.Code, c.scrub(result.Error.Message))
	}
	translated := make([]string, len(result.Data.Translations))
	for i, t := range result.Data.Translations {
		translated[i] = t.TranslatedText
	}
	return translated, retryAdvice{}, nil
}

// isRateLimitReason tells a throttling 403 from a real refusal (a bad key, a disabled API):
// only the former is worth waiting out.
func isRateLimitReason(body []byte) bool {
	var r translateResponse
	if json.Unmarshal(body, &r) != nil || r.Error == nil {
		return false
	}
	for _, e := range r.Error.Errors {
		if e.Reason == "rateLimitExceeded" || e.Reason == "userRateLimitExceeded" {
			return true
		}
	}
	return false
}

// errorText is the provider's own message when the body carries one, else the body, bounded.
func errorText(body []byte) string {
	var r translateResponse
	if json.Unmarshal(body, &r) == nil && r.Error != nil && r.Error.Message != "" {
		return r.Error.Message
	}
	const maxLen = 512
	s := strings.TrimSpace(string(body))
	if len(s) > maxLen {
		cut := maxLen
		for cut > 0 && !utf8.RuneStart(s[cut]) {
			cut--
		}
		s = s[:cut] + ".."
	}
	return s
}

// transportCause drops the *url.Error wrapper, whose text repeats the full request URL.
func transportCause(err error) string {
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return urlErr.Err.Error()
	}
	return err.Error()
}

// scrub removes the key from provider-supplied text as a last line of defence: the key is not
// in the URL any more, but an error body or a proxy's page could still echo a header back.
func (c *GoogleClient) scrub(s string) string {
	if c.apiKey == "" {
		return s
	}
	return strings.ReplaceAll(s, c.apiKey, "<redacted>")
}

// batchTexts splits texts into batches of at most maxSegs texts and, where possible, at most
// maxChars characters. A text longer than maxChars on its own goes alone in its own batch.
func batchTexts(texts []string, maxChars, maxSegs int) [][]string {
	var batches [][]string
	var current []string
	currentLen := 0
	flush := func() {
		if len(current) > 0 {
			batches = append(batches, current)
			current, currentLen = nil, 0
		}
	}
	for _, t := range texts {
		n := utf8.RuneCountInString(t)
		if n > maxChars {
			flush()
			batches = append(batches, []string{t})
			continue
		}
		if currentLen+n > maxChars || len(current) == maxSegs {
			flush()
		}
		current = append(current, t)
		currentLen += n
	}
	flush()
	return batches
}
