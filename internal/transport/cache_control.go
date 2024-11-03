package transport

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gregjones/httpcache"
	"github.com/gregjones/httpcache/diskcache"
)

// cacheControl is intended to be used to wrap the httpcache.Transport and set
// necessary headers used to determine if cache should be used
type cacheControl struct {
	rt     http.RoundTripper
	maxAge time.Duration
}

func NewDiskCacheControl(path string, maxAge time.Duration, next http.RoundTripper) http.RoundTripper {
	cache := diskcache.New(path)
	cacheRT := httpcache.NewTransport(cache)
	cacheRT.Transport = next
	return &cacheControl{cacheRT, maxAge}
}

func NewCacheControl(maxAge time.Duration, next http.RoundTripper) http.RoundTripper {
	cacheRT := httpcache.NewMemoryCacheTransport()
	cacheRT.Transport = next
	return &cacheControl{cacheRT, maxAge}
}

func (h *cacheControl) RoundTrip(r *http.Request) (*http.Response, error) {
	r.Header.Set("Cache-Control", fmt.Sprintf("max-age=%d", int64(h.maxAge.Seconds())))
	return h.rt.RoundTrip(r)
}
