package transport

import (
	"log/slog"
	"net/http"
	"time"
)

func Log(next http.RoundTripper) http.RoundTripper {
	if next == nil {
		next = http.DefaultTransport
	}
	return &log{next}
}

type log struct {
	next http.RoundTripper
}

func (l *log) RoundTrip(r *http.Request) (*http.Response, error) {
	start := time.Now()

	keys := []any{
		"path", r.URL.Path,
	}

	slog.Log(r.Context(), slog.LevelInfo, "starting request", keys...)
	resp, err := l.next.RoundTrip(r)
	keys = append(keys, "duration", time.Since(start))
	if err != nil {
		keys = append(keys, "err", err.Error())
		slog.Log(r.Context(), slog.LevelError, "request failed", keys...)
	} else if resp != nil {
		keys = append(keys, "status", resp.StatusCode)
		keys = append(keys, "cached", cacheUsed(resp.Header))
		slog.Log(r.Context(), slog.LevelInfo, "finished request", keys...)
	}
	return resp, err
}
