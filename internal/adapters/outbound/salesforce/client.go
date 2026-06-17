// Package salesforce is the outbound adapter for the Salesforce actions service.
// Mirrors SalesforceProvider.getActionFromUser.
package salesforce

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"ms_home/internal/domain"
	"ms_home/pkg/httpclient"
)

// ErrNoUserID mirrors SalesforceNoUserIdError (action requires a profile).
var ErrNoUserID = fmt.Errorf("salesforce: missing profile id")

// Adapter implements ports.SalesforcePort.
type Adapter struct {
	http *httpclient.Client
	url  string
}

// New builds the adapter. url is the full Salesforce actions endpoint.
func New(client *httpclient.Client, url string) *Adapter {
	return &Adapter{http: client, url: strings.TrimRight(url, "/")}
}

// GetActionFromUser POSTs the action for the current profile and decodes the payload.
func (a *Adapter) GetActionFromUser(ctx context.Context, action string) (map[string]any, error) {
	ri := domain.RequestInfoFromContext(ctx)
	if ri.ProfileID == "" {
		return nil, ErrNoUserID
	}

	body := map[string]any{
		"action": action,
		"flags":  map[string]any{"noCampaigns": false},
		"source": map[string]any{
			"channel":     "Server",
			"application": application(ri.Source),
		},
		"user": map[string]any{
			"attributes": map[string]any{"ID_ATG1": ri.ProfileID},
		},
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("salesforce: marshal: %w", err)
	}

	headers := map[string]string{"Content-Type": "application/json"}
	if ri.CorrelationID != "" {
		headers["x-correlation-id"] = ri.CorrelationID
	}

	resp, err := a.http.Do(ctx, http.MethodPost, a.url, payload, headers)
	if err != nil {
		return nil, fmt.Errorf("salesforce: action %s: %w", action, err)
	}
	if resp.Status < 200 || resp.Status >= 300 {
		return nil, fmt.Errorf("salesforce: action %s: unexpected status %d", action, resp.Status)
	}

	var data map[string]any
	if err := json.Unmarshal(resp.Body, &data); err != nil {
		return nil, fmt.Errorf("salesforce: decode %s: %w", action, err)
	}
	return data, nil
}

// application maps the source surface to the Salesforce application name.
func application(s domain.Source) string {
	if s == domain.SourcePocket {
		return "App"
	}
	return "Web"
}
