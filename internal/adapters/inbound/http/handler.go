// Package http is the inbound adapter exposing HOME over net/http (stdlib
// ServeMux, Go 1.22+ path patterns). It translates HTTP <-> domain only.
package http

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"ms_home/internal/application"
	"ms_home/internal/domain"
)

// TokenVerifier validates a bearer token and returns its claims. Implemented by
// internal/auth.Verifier; nil in dev (identity falls back to the x-profile-id header).
type TokenVerifier interface {
	Verify(ctx context.Context, token string) (map[string]any, error)
}

// Handler serves the HOME content endpoint.
type Handler struct {
	home         *application.HomeService
	verifier     TokenVerifier
	profileClaim string
	log          *slog.Logger
	defaultBrand string
}

// NewHandler builds the inbound handler. verifier may be nil (dev mode).
func NewHandler(home *application.HomeService, verifier TokenVerifier, profileClaim, defaultBrand string, log *slog.Logger) *Handler {
	return &Handler{home: home, verifier: verifier, profileClaim: profileClaim, defaultBrand: defaultBrand, log: log}
}

// GetContent handles GET /content/{contentType}/{locale}/{path...}.
// Path defaulting mirrors digital_bff: screen "" -> "home", otherwise "" -> "/".
func (h *Handler) GetContent(w http.ResponseWriter, r *http.Request) {
	ct := domain.ContentType(r.PathValue("contentType"))
	locale := r.PathValue("locale")
	path := r.PathValue("path")

	if path == "" {
		if ct.IsScreen() {
			path = "home"
		} else {
			path = "/"
		}
	}

	ri := h.requestInfo(r, ct, locale)
	ctx := domain.WithRequestInfo(r.Context(), ri)

	doc, err := h.home.GetHome(ctx, ct, locale, path)
	if err != nil {
		h.log.ErrorContext(ctx, "get home failed", slog.String("error", err.Error()),
			slog.String("content_type", string(ct)), slog.String("locale", locale))
		writeJSON(w, http.StatusFailedDependency, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, doc)
}

// requestInfo derives the per-request context from headers and the verified token.
func (h *Handler) requestInfo(r *http.Request, ct domain.ContentType, locale string) domain.RequestInfo {
	brand := r.Header.Get("x-brand-id")
	if brand == "" {
		brand = h.defaultBrand
	}
	corr := r.Header.Get("x-correlation-id")
	if corr == "" {
		corr = newID()
	}
	source := domain.SourceWeb
	if ct.IsScreen() {
		source = domain.SourcePocket
	}

	profileID, claims := h.identity(r)

	visitorID := ""
	if c, err := r.Cookie("gbi_visitorId"); err == nil {
		visitorID = c.Value
	}

	return domain.RequestInfo{
		RequestID:       newID(),
		CorrelationID:   corr,
		Brand:           brand,
		Source:          source,
		Preview:         r.Header.Get("x-preview") != "",
		Locale:          locale,
		ProfileID:       profileID,
		VisitorID:       visitorID,
		LoggedIn:        profileID != "",
		Claims:          claims,
		JewelUserID:     r.Header.Get("x-jml-user-id"),
		JewelDeviceID:   r.Header.Get("x-jml-device-id"),
		ClientPage:      r.Header.Get("x-client-page"),
		ClientChannel:   r.Header.Get("x-client-channel"),
		ClientAction:    r.Header.Get("x-client-action"),
		ClientComponent: r.Header.Get("x-client-component"),
		Cookie:          r.Header.Get("Cookie"),
		State:           domain.NewRequestState(),
	}
}

// identity resolves the profile id (and JWT claims) for the request. When a
// verifier is configured, identity comes ONLY from a valid bearer token (the
// x-profile-id header is ignored for security); otherwise the header is the dev
// fallback. An invalid/absent token yields an anonymous request.
func (h *Handler) identity(r *http.Request) (string, map[string]any) {
	if h.verifier == nil {
		return r.Header.Get("x-profile-id"), nil
	}
	token := bearerToken(r)
	if token == "" {
		return "", nil
	}
	claims, err := h.verifier.Verify(r.Context(), token)
	if err != nil {
		h.log.DebugContext(r.Context(), "jwt verification failed", slog.String("error", err.Error()))
		return "", nil
	}
	return claimString(claims, h.profileClaim), claims
}

func bearerToken(r *http.Request) string {
	const prefix = "Bearer "
	auth := r.Header.Get("Authorization")
	if len(auth) > len(prefix) && strings.EqualFold(auth[:len(prefix)], prefix) {
		return auth[len(prefix):]
	}
	return ""
}

// claimString reads a claim as a string (numbers are formatted without decimals).
func claimString(claims map[string]any, key string) string {
	switch v := claims[key].(type) {
	case string:
		return v
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	default:
		return ""
	}
}

// newID returns a random 16-byte hex identifier (stdlib, no external deps).
func newID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
