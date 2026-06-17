// Package groupby is the outbound adapter for the GroupBy search service.
// It owns the GroupBy request/response shape (skill Rule 1). Mirrors
// GroupBySearchProvider.searchProductList.
package groupby

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"ms_home/internal/domain"
	"ms_home/internal/ports"
	"ms_home/pkg/httpclient"
)

// SearchAdapter implements ports.GroupBySearchPort.
type SearchAdapter struct {
	http *httpclient.Client
	url  string
}

// NewSearch builds the adapter. url is the full GroupBy search endpoint.
func NewSearch(client *httpclient.Client, url string) *SearchAdapter {
	return &SearchAdapter{http: client, url: strings.TrimRight(url, "/")}
}

// SearchProductList POSTs the search and decodes records. The mandatory
// price (>=10) and availability (IN_STOCK) refinements, identity, and
// client-metadata fields are added here, matching the provider.
func (a *SearchAdapter) SearchProductList(ctx context.Context, cfg ports.GroupBySearchConfig) (*ports.GroupBySearchResult, error) {
	ri := domain.RequestInfoFromContext(ctx)

	body := map[string]any{
		"loginId":   ri.ProfileID,
		"visitorId": ri.VisitorID,
		"pageSize":  cfg.PageSize,
		"refinements": []any{
			map[string]any{"type": "Value", "navigationName": "attributes.ancestors", "value": cfg.Category, "or": false},
			map[string]any{"type": "Range", "navigationName": "price", "low": "10.0", "displayName": "Precios", "or": false},
			map[string]any{"type": "Value", "navigationName": "availability", "value": "IN_STOCK", "displayName": "Inventario", "or": false},
		},
		"clientMetadata": map[string]any{
			"component": cfg.Component,
			"site":      ri.Brand,
			"page":      ri.ClientPage,
			"channel":   ri.ClientChannel,
			"action":    ri.ClientAction,
		},
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("groupby: marshal: %w", err)
	}

	headers := map[string]string{"Content-Type": "application/json"}
	if ri.CorrelationID != "" {
		headers["x-correlation-id"] = ri.CorrelationID
	}

	resp, err := a.http.Do(ctx, http.MethodPost, a.url, payload, headers)
	if err != nil {
		return nil, fmt.Errorf("groupby: search: %w", err)
	}
	if resp.Status < 200 || resp.Status >= 300 {
		return nil, fmt.Errorf("groupby: search: unexpected status %d", resp.Status)
	}

	var parsed struct {
		Records         []map[string]any `json:"records"`
		OriginalRequest struct {
			ClientMetadata map[string]any `json:"clientMetadata"`
		} `json:"originalRequest"`
	}
	if err := json.Unmarshal(resp.Body, &parsed); err != nil {
		return nil, fmt.Errorf("groupby: decode: %w", err)
	}
	return &ports.GroupBySearchResult{Records: parsed.Records, Metadata: parsed.OriginalRequest.ClientMetadata}, nil
}
