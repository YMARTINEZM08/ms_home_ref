// Package searchfacade is the outbound adapter for the Liverpool Search Facade.
// Mirrors LiverpoolSearchFacadeProvider.getMultiProductDetails.
package searchfacade

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

// MultiProductAdapter implements ports.SearchFacadePort.
type MultiProductAdapter struct {
	http    *httpclient.Client
	baseURL string
}

// NewMultiProduct builds the adapter. baseURL is the Search Facade base.
func NewMultiProduct(client *httpclient.Client, baseURL string) *MultiProductAdapter {
	return &MultiProductAdapter{http: client, baseURL: strings.TrimRight(baseURL, "/")}
}

// GetMultiProductDetails POSTs /getMultiProduct with the searchFacadeConfig
// (dataCenter/brand/channel) plus the multi-product query, returning records.
func (a *MultiProductAdapter) GetMultiProductDetails(ctx context.Context, productIDs []string, favoriteStore string) (*ports.MultiProductResult, error) {
	ri := domain.RequestInfoFromContext(ctx)

	body := map[string]any{
		"dataCenter":            "SiteA", // searchFacadeConfig (request-context.ts)
		"brand":                 ri.Brand,
		"channel":               string(ri.Source),
		"productIds":            productIDs,
		"storeTypeResponseFlag": false,
	}
	if favoriteStore != "" {
		body["favoriteStore"] = favoriteStore
		body["nearByStore"] = favoriteStore
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("searchfacade: marshal: %w", err)
	}
	headers := map[string]string{"Content-Type": "application/json"}
	if ri.CorrelationID != "" {
		headers["x-correlation-id"] = ri.CorrelationID
	}

	resp, err := a.http.Do(ctx, http.MethodPost, a.baseURL+"/getMultiProduct", payload, headers)
	if err != nil {
		return nil, fmt.Errorf("searchfacade: multi-product: %w", err)
	}
	if resp.Status < 200 || resp.Status >= 300 {
		return nil, fmt.Errorf("searchfacade: multi-product: unexpected status %d", resp.Status)
	}

	var parsed struct {
		Records []map[string]any `json:"records"`
	}
	if err := json.Unmarshal(resp.Body, &parsed); err != nil {
		return nil, fmt.Errorf("searchfacade: decode: %w", err)
	}
	return &ports.MultiProductResult{Records: parsed.Records}, nil
}
