package transport

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRetrySuccessOnFirstAttempt(t *testing.T) {
	called := 0
	next := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		called++
		return newTestResponse(http.StatusOK, "ok"), nil
	})

	rt := NewRetry(next, 3, time.Millisecond, time.Millisecond)
	resp, err := rt.RoundTrip(httptest.NewRequest(http.MethodGet, "http://example.com", nil))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, 1, called)
}

func TestRetryNetworkError(t *testing.T) {
	called := 0
	next := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		called++
		if called < 3 {
			return nil, assert.AnError
		}
		return newTestResponse(http.StatusOK, "ok"), nil
	})

	rt := NewRetry(next, 3, time.Millisecond, time.Millisecond)
	resp, err := rt.RoundTrip(httptest.NewRequest(http.MethodGet, "http://example.com", nil))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, 3, called)
}

func TestRetry5xx(t *testing.T) {
	called := 0
	next := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		called++
		if called < 3 {
			return newTestResponse(http.StatusServiceUnavailable, "unavailable"), nil
		}
		return newTestResponse(http.StatusOK, "ok"), nil
	})

	rt := NewRetry(next, 3, time.Millisecond, time.Millisecond)
	resp, err := rt.RoundTrip(httptest.NewRequest(http.MethodGet, "http://example.com", nil))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, 3, called)
}

func TestRetryExhaustedReturnsLastError(t *testing.T) {
	next := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return nil, assert.AnError
	})

	rt := NewRetry(next, 2, time.Millisecond, time.Millisecond)
	_, err := rt.RoundTrip(httptest.NewRequest(http.MethodGet, "http://example.com", nil))
	require.Error(t, err)
	assert.ErrorIs(t, err, assert.AnError)
}

func TestRetryExhaustedReturnsLast5xx(t *testing.T) {
	called := 0
	next := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		called++
		return newTestResponse(http.StatusBadGateway, "bad gateway"), nil
	})

	rt := NewRetry(next, 2, time.Millisecond, time.Millisecond)
	resp, err := rt.RoundTrip(httptest.NewRequest(http.MethodGet, "http://example.com", nil))
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadGateway, resp.StatusCode)
	assert.Equal(t, 2, called)
}

func TestRetry4xxNotRetried(t *testing.T) {
	called := 0
	next := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		called++
		return newTestResponse(http.StatusNotFound, "not found"), nil
	})

	rt := NewRetry(next, 3, time.Millisecond, time.Millisecond)
	resp, err := rt.RoundTrip(httptest.NewRequest(http.MethodGet, "http://example.com", nil))
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	assert.Equal(t, 1, called)
}

func TestRetryBackoff(t *testing.T) {
	called := 0
	delays := []time.Duration{}
	var last time.Time
	next := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		called++
		now := time.Now()
		if !last.IsZero() {
			delays = append(delays, now.Sub(last))
		}
		last = now
		return nil, assert.AnError
	})

	rt := NewRetry(next, 4, 10*time.Millisecond, 50*time.Millisecond).(*retry)
	_, _ = rt.RoundTrip(httptest.NewRequest(http.MethodGet, "http://example.com", nil))
	require.Len(t, delays, 3)
	assert.GreaterOrEqual(t, delays[0], 10*time.Millisecond)
	assert.GreaterOrEqual(t, delays[1], 20*time.Millisecond)
	assert.GreaterOrEqual(t, delays[2], 40*time.Millisecond)
}

func TestRetryRespectsMaxDelay(t *testing.T) {
	rt := NewRetry(nil, 5, 100*time.Millisecond, 150*time.Millisecond).(*retry)
	assert.Equal(t, 100*time.Millisecond, rt.backoff(1))
	assert.Equal(t, 150*time.Millisecond, rt.backoff(2))
	assert.Equal(t, 150*time.Millisecond, rt.backoff(3))
}

func TestRetryClosesBodiesBeforeRetry(t *testing.T) {
	var closed atomic.Int32
	next := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		body := &closeRecorder{closed: &closed}
		return &http.Response{
			StatusCode: http.StatusServiceUnavailable,
			Body:       body,
			Header:     http.Header{},
		}, nil
	})

	rt := NewRetry(next, 2, time.Millisecond, time.Millisecond)
	resp, err := rt.RoundTrip(httptest.NewRequest(http.MethodGet, "http://example.com", nil))
	require.NoError(t, err)
	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
	// Only the first failed response body is closed before the retry; the
	// final response body is returned to the caller.
	assert.Equal(t, int32(1), closed.Load())
	_ = resp.Body.Close()
	assert.Equal(t, int32(2), closed.Load())
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newTestResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     http.Header{},
	}
}

type closeRecorder struct {
	closed *atomic.Int32
}

func (c *closeRecorder) Read([]byte) (int, error) { return 0, io.EOF }
func (c *closeRecorder) Close() error {
	c.closed.Add(1)
	return nil
}
