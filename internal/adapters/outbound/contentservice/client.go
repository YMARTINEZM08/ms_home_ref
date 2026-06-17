// Package contentservice is the outbound adapter for the Content Service proxy.
// It is the only place that knows the proxy's HTTP shape (skill Rules 1, 2).
// Mirrors digital_bff ContentProvider.getContent.
package contentservice

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"ms_home/internal/domain"
	"ms_home/pkg/httpclient"
)

// Adapter implements ports.ContentPort against the Content Service proxy.
type Adapter struct {
	http    *httpclient.Client
	baseURL string
}

// New builds the adapter. baseURL comes from SHARED_CONTENT_SERVICE_URL.
func New(client *httpclient.Client, baseURL string) *Adapter {
	return &Adapter{http: client, baseURL: strings.TrimRight(baseURL, "/")}
}

// GetContent fetches an entry: GET /content/{contentType}/{locale}/{id} with the
// x-brand-id header derived from the per-request context (brand + -PREVIEW).
func (a *Adapter) GetContent(ctx context.Context, ct domain.ContentType, locale, id string) (domain.Document, error) {
	ri := domain.RequestInfoFromContext(ctx)
	// Encode the id segment to match digital_bff's encodeURIComponent (e.g. "/" -> "%2F").
	endpoint := fmt.Sprintf("%s/content/%s/%s/%s", a.baseURL, ct, locale, url.PathEscape(id))

	headers := map[string]string{"x-brand-id": ri.BrandHeader()}
	if ri.CorrelationID != "" {
		headers["x-correlation-id"] = ri.CorrelationID
	}

	resp, err := a.http.Do(ctx, http.MethodGet, endpoint, nil, headers)
	if err != nil {
		return nil, fmt.Errorf("contentservice: get %s: %w", ct, err)
	}
	if resp.Status < 200 || resp.Status >= 300 {
		return nil, fmt.Errorf("contentservice: get %s: unexpected status %d", ct, resp.Status)
	}

	var doc domain.Document
	if err := json.Unmarshal(resp.Body, &doc); err != nil {
		return nil, fmt.Errorf("contentservice: decode %s: %w", ct, err)
	}
	return doc, nil
}
