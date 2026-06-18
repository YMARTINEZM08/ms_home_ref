// Package http is the inbound adapter exposing HOME over net/http (stdlib
// ServeMux, Go 1.22+ path patterns). It translates HTTP <-> domain only.
package http

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"

	"ms_home/internal/application"
	"ms_home/internal/domain"
)

// Handler serves the HOME content endpoint.
type Handler struct {
	home         *application.HomeService
	auth         Authenticator // nil = dev mode (x-profile-id header)
	log          *slog.Logger
	defaultBrand string
}

// NewHandler builds the inbound handler. auth may be nil (dev mode).
func NewHandler(home *application.HomeService, auth Authenticator, defaultBrand string, log *slog.Logger) *Handler {
	return &Handler{home: home, auth: auth, defaultBrand: defaultBrand, log: log}
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

	profileID, loggedIn, claims := h.identity(r)

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
		LoggedIn:        loggedIn,
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

// identity resolves (profileID, loggedIn, claims). With an Authenticator configured,
// identity comes only from it (the x-profile-id header is ignored for security);
// otherwise the header is the dev fallback. Anonymous/invalid → ("", false, nil).
func (h *Handler) identity(r *http.Request) (string, bool, map[string]any) {
	if h.auth == nil {
		p := r.Header.Get("x-profile-id")
		return p, p != "", nil
	}
	return h.auth.Authenticate(r.Context(), r)
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
