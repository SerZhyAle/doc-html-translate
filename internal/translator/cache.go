package translator

import (
	"context"
	"errors"
	"fmt"
)

// CachingClient wraps any Client and deduplicates repeated segments within a run.
// Useful for navigation bars, headers, and repeated phrases across chapters.
type CachingClient struct {
	inner Client
	cache map[string]string
}

// NewCachingClient wraps the given client with an in-memory translation cache.
func NewCachingClient(inner Client) *CachingClient {
	return &CachingClient{
		inner: inner,
		cache: make(map[string]string),
	}
}

// SetProgress implements ProgressReporter by forwarding to the inner client if it supports progress.
func (c *CachingClient) SetProgress(f func(done, total int)) {
	if pr, ok := c.inner.(ProgressReporter); ok {
		pr.SetProgress(f)
	}
}

func (c *CachingClient) Translate(ctx context.Context, texts []string, srcLang, dstLang string) ([]string, error) {
	results := make([]string, len(texts))
	var missTexts []string
	var missIdx []int

	for i, t := range texts {
		key := fmt.Sprintf("%s:%s:%s", srcLang, dstLang, t)
		if v, ok := c.cache[key]; ok {
			results[i] = v
		} else {
			missTexts = append(missTexts, t)
			missIdx = append(missIdx, i)
		}
	}

	if len(missTexts) == 0 {
		return results, nil
	}

	translated, err := c.inner.Translate(ctx, missTexts, srcLang, dstLang)
	var partial *PartialError
	if err != nil && !errors.As(err, &partial) {
		return nil, err
	}
	// Matching by index is only sound when every text got exactly one answer; an engine that
	// broke that promise must not crash the run or put one text's translation on another.
	if len(translated) != len(missTexts) {
		return nil, fmt.Errorf("translation engine returned %d results for %d texts", len(translated), len(missTexts))
	}

	// A slot the inner client could not fill must not be cached: the next page asking for the
	// same text would get the empty string back as if it were the translation.
	skip := map[int]bool{}
	if partial != nil {
		for _, j := range partial.Missing {
			skip[j] = true
		}
	}
	var missing []int
	for j, idx := range missIdx {
		if skip[j] {
			missing = append(missing, idx)
			continue
		}
		results[idx] = translated[j]
		key := fmt.Sprintf("%s:%s:%s", srcLang, dstLang, missTexts[j])
		c.cache[key] = translated[j]
	}
	if partial != nil {
		return results, &PartialError{Missing: missing, Err: partial.Err}
	}
	return results, nil
}

// Stats returns cache hit/miss counters for diagnostics.
func (c *CachingClient) Stats() (cached int) {
	return len(c.cache)
}
