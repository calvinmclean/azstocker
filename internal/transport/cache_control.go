package transport

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gregjones/httpcache"
	"github.com/gregjones/httpcache/diskcache"
	"github.com/prometheus/client_golang/prometheus"
)

var httpCacheMetric = prometheus.NewGaugeVec(prometheus.GaugeOpts{
	Namespace: "stocker",
	Name:      "http_client_cache",
	Help:      "gauge of cache usage",
}, []string{"path", "cache_used"})

func init() {
	prometheus.MustRegister(httpCacheMetric)
}

// cacheControl is intended to be used to wrap the httpcache.Transport and set
// necessary headers used to determine if cache should be used
type cacheControl struct {
	rt     http.RoundTripper
	maxAge time.Duration
}

func NewDiskCacheControl(path string, maxAge time.Duration, next http.RoundTripper) http.RoundTripper {
	cache := diskcache.New(path)
	return newCacheControl(cache, maxAge, next)
}

func NewCacheControl(maxAge time.Duration, next http.RoundTripper) http.RoundTripper {
	cache := httpcache.NewMemoryCache()
	return newCacheControl(cache, maxAge, next)
}

func newCacheControl(cache httpcache.Cache, maxAge time.Duration, next http.RoundTripper) http.RoundTripper {
	cacheRT := httpcache.NewTransport(cache)
	cacheRT.Transport = next
	cacheRT.MarkCachedResponses = true
	return &cacheControl{cacheRT, maxAge}
}

func (h *cacheControl) RoundTrip(r *http.Request) (*http.Response, error) {
	r.Header.Set("Cache-Control", fmt.Sprintf("max-age=%d", int64(h.maxAge.Seconds())))
	resp, err := h.rt.RoundTrip(r)
	if resp != nil {
		httpCacheMetric.WithLabelValues(r.URL.Path, fmt.Sprint(cacheUsed(resp.Header))).Inc()
	}
	return resp, err
}

func cacheUsed(headers http.Header) bool {
	return headers.Get(httpcache.XFromCache) == "1"
}
