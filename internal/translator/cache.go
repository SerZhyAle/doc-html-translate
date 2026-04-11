package translator

import "fmt"

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

func (c *CachingClient) Translate(texts []string, srcLang, dstLang string) ([]string, error) {
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

	translated, err := c.inner.Translate(missTexts, srcLang, dstLang)
	if err != nil {
		return nil, err
	}

	for j, idx := range missIdx {
		results[idx] = translated[j]
		key := fmt.Sprintf("%s:%s:%s", srcLang, dstLang, missTexts[j])
		c.cache[key] = translated[j]
	}

	return results, nil
}

// Stats returns cache hit/miss counters for diagnostics.
func (c *CachingClient) Stats() (cached int) {
	return len(c.cache)
}
