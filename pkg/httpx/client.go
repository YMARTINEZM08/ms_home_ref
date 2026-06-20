package httpx

import (
	"net/http"
	"time"
)

// NewClient returns a reusable *http.Client configured for keep-alive and
// low-latency outbound calls. Each outbound adapter should share one instance.
func NewClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 20,
			IdleConnTimeout:     90 * time.Second,
			DisableKeepAlives:   false,
		},
	}
}
