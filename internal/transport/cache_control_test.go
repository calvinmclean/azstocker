package transport

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gregjones/httpcache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCacheControlSetsStaleIfError(t *testing.T) {
	var received http.Header
	next := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		received = req.Header
		return newCacheableResponse(http.StatusOK, "fresh"), nil
	})

	rt := NewCacheControl(time.Hour, next)
	_, err := rt.RoundTrip(httptest.NewRequest(http.MethodGet, "http://example.com", nil))
	require.NoError(t, err)

	cc := received.Get("Cache-Control")
	assert.Contains(t, cc, "max-age=3600")
	assert.Contains(t, cc, "stale-if-error")
}

func TestStaleCacheReturnedOnUpstreamError(t *testing.T) {
	calls := atomic.Int32{}
	next := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		if calls.Add(1) == 1 {
			return newCacheableResponse(http.StatusOK, "cached-value"), nil
		}
		return nil, assert.AnError
	})

	cache := httpcache.NewMemoryCache()
	cacheRT := httpcache.NewTransport(cache)
	cacheRT.Transport = next
	// maxAge=0 forces the cached response to be treated as stale, so the
	// second request will contact the upstream and fall back to the cache.
	rt := &cacheControl{rt: cacheRT, maxAge: 0}

	req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	resp, err := rt.RoundTrip(req)
	require.NoError(t, err)
	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, "cached-value", string(body))

	req2 := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	resp2, err := rt.RoundTrip(req2)
	require.NoError(t, err)
	body2, _ := io.ReadAll(resp2.Body)
	assert.Equal(t, "cached-value", string(body2))
	assert.Equal(t, "1", resp2.Header.Get(httpcache.XFromCache))
	assert.Equal(t, int32(2), calls.Load())
}

func TestStaleCacheWithRetryOnUpstreamError(t *testing.T) {
	calls := atomic.Int32{}
	next := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		if calls.Add(1) == 1 {
			return newCacheableResponse(http.StatusOK, "cached-value"), nil
		}
		return nil, assert.AnError
	})

	cache := httpcache.NewMemoryCache()
	cacheRT := httpcache.NewTransport(cache)
	cacheRT.Transport = NewRetry(next, 3, time.Millisecond, time.Millisecond)
	rt := &cacheControl{rt: cacheRT, maxAge: 0}

	req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	resp, err := rt.RoundTrip(req)
	require.NoError(t, err)
	_, _ = io.ReadAll(resp.Body)

	req2 := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	resp2, err := rt.RoundTrip(req2)
	require.NoError(t, err)
	body2, _ := io.ReadAll(resp2.Body)
	assert.Equal(t, "cached-value", string(body2))
	assert.Equal(t, "1", resp2.Header.Get(httpcache.XFromCache))
	// initial request + 3 retries on the second stale request
	assert.Equal(t, int32(4), calls.Load())
}

func TestStaleCacheWithRetryOnUpstream5xx(t *testing.T) {
	calls := atomic.Int32{}
	next := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		if calls.Add(1) == 1 {
			return newCacheableResponse(http.StatusOK, "cached-value"), nil
		}
		return newCacheableResponse(http.StatusServiceUnavailable, "unavailable"), nil
	})

	cache := httpcache.NewMemoryCache()
	cacheRT := httpcache.NewTransport(cache)
	cacheRT.Transport = NewRetry(next, 3, time.Millisecond, time.Millisecond)
	rt := &cacheControl{rt: cacheRT, maxAge: 0}

	req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	resp, err := rt.RoundTrip(req)
	require.NoError(t, err)
	_, _ = io.ReadAll(resp.Body)

	req2 := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	resp2, err := rt.RoundTrip(req2)
	require.NoError(t, err)
	body2, _ := io.ReadAll(resp2.Body)
	assert.Equal(t, "cached-value", string(body2))
	assert.Equal(t, "1", resp2.Header.Get(httpcache.XFromCache))
	// initial request + 3 retry attempts on the second stale request
	assert.Equal(t, int32(4), calls.Load())
}

func TestFreshCacheReturnedWithoutUpstreamRequest(t *testing.T) {
	calls := atomic.Int32{}
	next := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		calls.Add(1)
		return newCacheableResponse(http.StatusOK, "cached-value"), nil
	})

	cache := httpcache.NewMemoryCache()
	cacheRT := httpcache.NewTransport(cache)
	cacheRT.Transport = next
	// maxAge > 0 means the cached response is considered fresh and will be
	// returned on the second request without contacting the upstream server.
	rt := &cacheControl{rt: cacheRT, maxAge: time.Hour}

	req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	resp, err := rt.RoundTrip(req)
	require.NoError(t, err)
	_, _ = io.ReadAll(resp.Body)

	req2 := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	resp2, err := rt.RoundTrip(req2)
	require.NoError(t, err)
	body2, _ := io.ReadAll(resp2.Body)
	assert.Equal(t, "cached-value", string(body2))
	assert.Equal(t, "1", resp2.Header.Get(httpcache.XFromCache))
	assert.Equal(t, int32(1), calls.Load())
}

func TestStaleCacheReturnedOnUpstream5xx(t *testing.T) {
	calls := atomic.Int32{}
	next := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		if calls.Add(1) == 1 {
			return newCacheableResponse(http.StatusOK, "cached-value"), nil
		}
		return newCacheableResponse(http.StatusServiceUnavailable, "unavailable"), nil
	})

	cache := httpcache.NewMemoryCache()
	cacheRT := httpcache.NewTransport(cache)
	cacheRT.Transport = next
	rt := &cacheControl{rt: cacheRT, maxAge: 0}

	req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	resp, err := rt.RoundTrip(req)
	require.NoError(t, err)
	_, _ = io.ReadAll(resp.Body)

	req2 := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	resp2, err := rt.RoundTrip(req2)
	require.NoError(t, err)
	body2, _ := io.ReadAll(resp2.Body)
	assert.Equal(t, "cached-value", string(body2))
	assert.Equal(t, "1", resp2.Header.Get(httpcache.XFromCache))
}

func newCacheableResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header: http.Header{
			"Date":          []string{time.Now().UTC().Format(http.TimeFormat)},
			"Cache-Control": []string{"max-age=0"},
			"Content-Type":  []string{"text/plain"},
		},
	}
}
