package transport

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

// retry wraps an http.RoundTripper and retries requests that fail due to network
// errors or receive a 5xx response. It uses exponential backoff capped at a
// maximum delay.
type retry struct {
	next     http.RoundTripper
	attempts int
	base     time.Duration
	max      time.Duration
}

// NewRetry returns a RoundTripper that retries failed requests. It will attempt
// the request up to attempts times total (including the initial request) and
// waits base * 2^attempt between retries, capped at max.
func NewRetry(next http.RoundTripper, attempts int, base, max time.Duration) http.RoundTripper {
	if next == nil {
		next = http.DefaultTransport
	}
	if attempts < 1 {
		attempts = 1
	}
	if base <= 0 {
		base = 100 * time.Millisecond
	}
	if max <= 0 {
		max = time.Second
	}
	return &retry{next: next, attempts: attempts, base: base, max: max}
}

func (r *retry) RoundTrip(req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error

	for attempt := 0; attempt < r.attempts; attempt++ {
		if attempt > 0 {
			time.Sleep(r.backoff(attempt))
		}

		resp, err = r.next.RoundTrip(req)
		if err != nil {
			continue
		}

		if resp.StatusCode < http.StatusInternalServerError {
			return resp, nil
		}

		if attempt < r.attempts-1 && resp.Body != nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
		}
	}

	if err != nil {
		return nil, fmt.Errorf("request failed after %d attempts: %w", r.attempts, err)
	}

	return resp, nil
}

func (r *retry) backoff(attempt int) time.Duration {
	d := r.base * (1 << (attempt - 1))
	if d > r.max {
		return r.max
	}
	return d
}
