// Package httpclient is a reusable outbound HTTP client shared by all outbound
// adapters. It enforces keep-alive, context propagation, structured logging with
// masked secrets, and cURL emission at DEBUG (skill Rules 4, 10, 11).
package httpclient

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// sensitiveHeaders are masked in logs and generated cURL commands.
var sensitiveHeaders = map[string]struct{}{
	"authorization": {},
	"cookie":        {},
	"set-cookie":    {},
	"x-api-key":     {},
	"access_token":  {},
}

// Client wraps a reused *http.Client with observability.
type Client struct {
	hc  *http.Client
	log *slog.Logger
}

// Response is the minimal decoded result returned to adapters.
type Response struct {
	Status int
	Body   []byte
}

// New builds a Client with a tuned, connection-pooling transport for Cloud Run.
func New(timeout time.Duration, log *slog.Logger) *Client {
	t := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
		ForceAttemptHTTP2:   true,
	}
	return &Client{
		hc:  &http.Client{Timeout: timeout, Transport: t},
		log: log,
	}
}

// Do executes an HTTP request, propagating ctx and logging the call. The caller
// owns interpreting status codes and decoding the body.
func (c *Client) Do(ctx context.Context, method, url string, body []byte, headers map[string]string) (*Response, error) {
	var rdr io.Reader
	if body != nil {
		rdr = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, rdr)
	if err != nil {
		return nil, fmt.Errorf("httpclient: build request: %w", err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	if c.log.Enabled(ctx, slog.LevelDebug) {
		c.log.DebugContext(ctx, "outbound request curl", slog.String("curl", toCurl(method, url, headers, body)))
	}

	start := time.Now()
	resp, err := c.hc.Do(req)
	latency := time.Since(start)
	if err != nil {
		c.log.ErrorContext(ctx, "outbound request failed",
			logAttrs(method, url, headers, 0, latency, err)...)
		return nil, fmt.Errorf("httpclient: %s %s: %w", method, url, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("httpclient: read body: %w", err)
	}

	c.log.InfoContext(ctx, "outbound request",
		logAttrs(method, url, headers, resp.StatusCode, latency, nil)...)

	return &Response{Status: resp.StatusCode, Body: data}, nil
}

func logAttrs(method, url string, headers map[string]string, status int, latency time.Duration, err error) []any {
	attrs := []any{
		slog.String("method", method),
		slog.String("url", url),
		slog.Int("status", status),
		slog.Duration("latency", latency),
	}
	if err != nil {
		attrs = append(attrs, slog.String("error", err.Error()))
	}
	return attrs
}

// toCurl builds an equivalent cURL command with sensitive headers masked.
func toCurl(method, url string, headers map[string]string, body []byte) string {
	var b strings.Builder
	fmt.Fprintf(&b, "curl -X %s '%s'", method, url)
	for k, v := range headers {
		if _, ok := sensitiveHeaders[strings.ToLower(k)]; ok {
			v = "***"
		}
		fmt.Fprintf(&b, " -H '%s: %s'", k, v)
	}
	if len(body) > 0 {
		fmt.Fprintf(&b, " --data '%s'", string(body))
	}
	return b.String()
}
