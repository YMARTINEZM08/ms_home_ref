// Package atg is the outbound adapter for the ATG cart-header endpoint.
// Mirrors LiverpoolATGProvider.getCartHeaderDetails.
package atg

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"ms_home/internal/domain"
	"ms_home/pkg/httpclient"
)

// CartHeaderAdapter implements ports.CartHeaderPort.
type CartHeaderAdapter struct {
	http *httpclient.Client
	url  string
}

// NewCartHeader builds the adapter. url is the full cart-header endpoint.
func NewCartHeader(client *httpclient.Client, url string) *CartHeaderAdapter {
	return &CartHeaderAdapter{http: client, url: strings.TrimRight(url, "/")}
}

// GetCartHeaderDetails POSTs to ATG and returns the decoded cartHeaderDetails object.
// brand/channel/cookie headers are forwarded from the per-request context.
func (a *CartHeaderAdapter) GetCartHeaderDetails(ctx context.Context) (map[string]any, error) {
	ri := domain.RequestInfoFromContext(ctx)

	body, err := json.Marshal(map[string]any{
		"fromBuyNow": "false",
		"rearrange":  "false",
	})
	if err != nil {
		return nil, fmt.Errorf("atg: marshal: %w", err)
	}

	headers := map[string]string{
		"Content-Type": "application/json",
		"brand":        ri.Brand,
		"channel":      string(ri.Source),
	}
	if ri.Cookie != "" {
		headers["cookie"] = ri.Cookie
	}
	if ri.CorrelationID != "" {
		headers["x-correlation-id"] = ri.CorrelationID
	}

	resp, err := a.http.Do(ctx, http.MethodPost, a.url, body, headers)
	if err != nil {
		return nil, fmt.Errorf("atg: cart header: %w", err)
	}
	if resp.Status < 200 || resp.Status >= 300 {
		return nil, fmt.Errorf("atg: cart header: unexpected status %d", resp.Status)
	}

	var parsed struct {
		CartHeaderDetails map[string]any `json:"cartHeaderDetails"`
	}
	if err := json.Unmarshal(resp.Body, &parsed); err != nil {
		return nil, fmt.Errorf("atg: decode: %w", err)
	}
	return parsed.CartHeaderDetails, nil
}
