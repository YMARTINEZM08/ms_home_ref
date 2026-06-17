// Package jewel is the outbound adapter for the Jewel recommendation service.
// Mirrors JewelProvider.getProductsFromModel.
package jewel

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"ms_home/internal/domain"
	"ms_home/internal/ports"
	"ms_home/pkg/httpclient"
)

// Adapter implements ports.JewelPort.
type Adapter struct {
	http *httpclient.Client
	url  string
}

// New builds the adapter. baseURL is the full Jewel recommendation endpoint.
func New(client *httpclient.Client, baseURL string) *Adapter {
	return &Adapter{http: client, url: strings.TrimRight(baseURL, "/")}
}

// GetProductsFromModel GETs products for a Jewel model. Returns an empty slice
// when the model is unset (matching the provider's early return).
func (a *Adapter) GetProductsFromModel(ctx context.Context, cfg ports.JewelModelConfig, min, max int) ([]map[string]any, error) {
	if cfg.Model == "" {
		return nil, nil
	}
	ri := domain.RequestInfoFromContext(ctx)

	q := url.Values{}
	q.Set("model", cfg.Model)
	q.Set("minimum_items", strconv.Itoa(min))
	q.Set("number_of_placements", strconv.Itoa(max))
	if cfg.RequiresUserID && ri.JewelUserID != "" {
		q.Set("user_id", ri.JewelUserID)
	}
	if cfg.RequiresDeviceID && ri.JewelDeviceID != "" {
		q.Set("unique_id", ri.JewelDeviceID)
	}

	headers := map[string]string{}
	if ri.CorrelationID != "" {
		headers["x-correlation-id"] = ri.CorrelationID
	}

	resp, err := a.http.Do(ctx, http.MethodGet, a.url+"?"+q.Encode(), nil, headers)
	if err != nil {
		return nil, fmt.Errorf("jewel: %w", err)
	}
	if resp.Status < 200 || resp.Status >= 300 {
		return nil, fmt.Errorf("jewel: unexpected status %d", resp.Status)
	}

	// The endpoint may return a bare array or a { products: [...] } envelope.
	var arr []map[string]any
	if err := json.Unmarshal(resp.Body, &arr); err == nil {
		return arr, nil
	}
	var wrapped struct {
		Products []map[string]any `json:"products"`
	}
	if err := json.Unmarshal(resp.Body, &wrapped); err != nil {
		return nil, fmt.Errorf("jewel: decode: %w", err)
	}
	return wrapped.Products, nil
}
