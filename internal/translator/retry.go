package translator

import (
	"context"
	"fmt"
	"math/rand/v2"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	// maxRetryWait is the most one request may spend waiting between attempts. Throttling that
	// outlasts it is not transient, and the run is better stopped with the book half translated
	// than hung for an unbounded time.
	maxRetryWait = 60 * time.Second
	backoffBase  = 1 * time.Second
	backoffCap   = 16 * time.Second
)

// retryPolicy waits between attempts. sleep and now are fields so the tests can run a minute of
// backoff in no time and check what would have been waited.
type retryPolicy struct {
	budget time.Duration
	sleep  func(ctx context.Context, d time.Duration) error
	now    func() time.Time
	jitter func(d time.Duration) time.Duration
}

func defaultRetryPolicy() retryPolicy {
	return retryPolicy{
		budget: maxRetryWait,
		sleep:  sleepContext,
		now:    time.Now,
		// Half fixed, half random: enough spread that parallel runs throttled together do not
		// come back together, while every wait still grows with the attempt.
		jitter: func(d time.Duration) time.Duration { return d/2 + rand.N(d/2+1) },
	}
}

// retryBudget tracks what one request has waited so far.
type retryBudget struct {
	policy retryPolicy
	waited time.Duration
}

func (p retryPolicy) start() *retryBudget { return &retryBudget{policy: p} }

// wait sleeps before attempt+1: the server's Retry-After when it gave one, otherwise capped
// exponential backoff with jitter. It refuses, without sleeping, a wait that would take the
// request past its budget.
func (b *retryBudget) wait(ctx context.Context, attempt int, retryAfter time.Duration) error {
	d := min(backoffBase<<min(attempt-1, 8), backoffCap)
	d = b.policy.jitter(d)
	if retryAfter > d {
		d = retryAfter
	}
	if b.waited+d > b.policy.budget {
		return fmt.Errorf("gave up after %d attempts and %s of waiting", attempt, b.waited.Round(time.Second))
	}
	if err := b.policy.sleep(ctx, d); err != nil {
		return err
	}
	b.waited += d
	return nil
}

func sleepContext(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// parseRetryAfter reads a Retry-After header, which is either a number of seconds or an HTTP
// date. Anything unreadable, or a date already past, is no advice at all.
func parseRetryAfter(v string, now time.Time) time.Duration {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0
	}
	if secs, err := strconv.Atoi(v); err == nil {
		if secs < 0 {
			return 0
		}
		// Anything past the budget is refused anyway; the cap only keeps the multiplication sane.
		return time.Duration(min(secs, 3600)) * time.Second
	}
	if t, err := http.ParseTime(v); err == nil && t.After(now) {
		return t.Sub(now)
	}
	return 0
}
