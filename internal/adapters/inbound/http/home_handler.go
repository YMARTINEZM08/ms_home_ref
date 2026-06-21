package handler

import (
	"log/slog"
	"net/http"
	"regexp"
	"strings"

	domain "github.com/YMARTINEZM08/ms_home_ref/internal/domain/home"
)

// allowedChannels is the inbound allowlist for the channel query parameter.
var allowedChannels = map[string]bool{
	"":      true,
	"pocket": true,
	"kiosk":  true,
	"mpos":   true,
}

// allowedBrand validates the brand value: 1-20 uppercase alphanumeric chars.
var allowedBrand = regexp.MustCompile(`^[A-Z0-9]{1,20}$`)

type homeHandler struct {
	useCase domain.HomeUseCase
	log     *slog.Logger
}

func newHomeHandler(uc domain.HomeUseCase, log *slog.Logger) *homeHandler {
	return &homeHandler{useCase: uc, log: log}
}

// ServeHTTP handles GET /home.
// Inputs: locale (query), x-brand-id (header), channel (query), x-preview (header).
// Output: ordered layout — static blocks inline, dynamic blocks as placeholders.
func (h *homeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	req, appErr := h.parseRequest(r)
	if appErr != nil {
		// Bad input: log at WARN (not ERROR — caller mistake, not service failure).
		h.log.WarnContext(r.Context(), "invalid home request",
			"error_code", string(appErr.Code),
			"detail", appErr.Detail,
		)
		writeError(w, appErr)
		return
	}

	layout, appErr := h.useCase.GetLayout(r.Context(), *req)
	if appErr != nil {
		// Use case / adapter already logged this. Do NOT re-log (log-once rule).
		writeError(w, appErr)
		return
	}

	writeJSON(w, http.StatusOK, toLayoutResponse(layout))
}

func (h *homeHandler) parseRequest(r *http.Request) (*domain.HomeRequest, *domain.AppError) {
	locale := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("locale")))
	if locale == "" {
		locale = "es-mx"
	}

	brand := strings.ToUpper(strings.TrimSpace(r.Header.Get("x-brand-id")))
	// Strip the -PREVIEW suffix before validation; re-apply via Preview flag.
	preview := strings.HasSuffix(brand, "-PREVIEW") || r.Header.Get("x-preview") == "true"
	brand = strings.TrimSuffix(brand, "-PREVIEW")
	if brand == "" {
		brand = "LP"
	}
	if !allowedBrand.MatchString(brand) {
		return nil, domain.ErrBadRequest("x-brand-id contains invalid characters or exceeds 20 characters")
	}

	channel := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("channel")))
	if !allowedChannels[channel] {
		return nil, domain.ErrBadRequest("channel must be one of: pocket, kiosk, mpos, or omitted")
	}

	// x-authenticated is set by the API gateway after token validation.
	// The service trusts this header within the internal network — it never
	// validates the token itself (separation of concerns).
	isLoggedIn := r.Header.Get("x-authenticated") == "true"

	return &domain.HomeRequest{
		Locale:     locale,
		Brand:      brand,
		Channel:    channel,
		Preview:    preview,
		IsLoggedIn: isLoggedIn,
	}, nil
}
