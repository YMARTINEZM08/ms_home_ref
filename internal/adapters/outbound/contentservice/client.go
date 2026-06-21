package contentservice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"time"

	"github.com/YMARTINEZM08/ms_home_ref/internal/domain/home"
	"github.com/YMARTINEZM08/ms_home_ref/pkg/breaker"
	"github.com/YMARTINEZM08/ms_home_ref/pkg/httpx"
)

// Config holds the content-service adapter configuration.
// All values come from environment variables — never hardcoded.
type Config struct {
	BaseURL         string
	HomePageID      string // page slug appended to the path, e.g. "tienda/home"
	Timeout         time.Duration
	BreakerSettings breaker.Settings
}

// Client implements domain.ContentPort. It calls the content-service proxy
// over HTTP, wraps calls in a circuit breaker, and emits structured logs.
type Client struct {
	cfg        Config
	httpClient *http.Client
	brk        *breaker.Breaker[[]byte]
	log        *slog.Logger
}

// NewClient constructs a Client. The caller owns the *http.Client lifecycle;
// use httpx.NewClient for a keep-alive-enabled instance.
func NewClient(cfg Config, httpClient *http.Client, log *slog.Logger) *Client {
	brk := breaker.New[[]byte]("content-service", cfg.BreakerSettings)
	return &Client{cfg: cfg, httpClient: httpClient, brk: brk, log: log}
}

// allowedLocale enforces that locale is a two-part IETF tag (e.g. es-mx).
var allowedLocale = regexp.MustCompile(`^[a-z]{2}-[a-z]{2}$`)

// FetchLayout implements domain.ContentPort. It fetches the page layout for
// the given HomeRequest from the content-service and returns normalised RawBlocks.
func (c *Client) FetchLayout(ctx context.Context, req home.HomeRequest) ([]home.RawBlock, *home.AppError) {
	if !allowedLocale.MatchString(req.Locale) {
		return nil, home.ErrBadRequest(fmt.Sprintf("locale %q is not a valid IETF language tag (expected xx-xx)", req.Locale))
	}

	rawURL, err := c.buildURL(req)
	if err != nil {
		return nil, home.ErrConfiguration("content-service base URL", err)
	}

	headers := c.buildHeaders(req)
	start := time.Now()

	// Emit cURL equivalent at DEBUG for incident replay (Rule 10).
	if c.log.Enabled(ctx, slog.LevelDebug) {
		c.log.DebugContext(ctx, "outbound request",
			"curl", httpx.BuildCurlCommand(http.MethodGet, rawURL, headers),
			"dependency", "content-service",
		)
	}

	body, execErr := c.brk.Execute(func() ([]byte, error) {
		return c.do(ctx, rawURL, headers)
	})

	latency := time.Since(start)

	if execErr != nil {
		return nil, c.mapError(ctx, "content-service", rawURL, latency, execErr)
	}

	c.log.InfoContext(ctx, "outbound response",
		"dependency", "content-service",
		"url", rawURL,
		"status", http.StatusOK,
		"latency_ms", latency.Milliseconds(),
		"request_id", httpx.RequestIDFromCtx(ctx),
		"correlation_id", httpx.CorrelationIDFromCtx(ctx),
	)

	var resp contentServiceResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, home.ErrUnexpected("content-service response decode", err)
	}

	blocks := mapToRawBlocks(&resp)
	return blocks, nil
}

// do executes the HTTP request and reads the full body.
// It is called inside the circuit breaker — never retried.
func (c *Client) do(ctx context.Context, rawURL string, headers map[string]string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20)) // 4 MB guard
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 300 {
		return nil, &httpStatusError{code: resp.StatusCode, body: body}
	}
	return body, nil
}

// buildURL constructs the content-service URL from config + validated request
// fields. User input only influences path segments after allowlist checks —
// the host always comes from config (SSRF prevention).
// The page identifier (HomePageID) is an infrastructure config value — it is
// never supplied by the caller and therefore cannot be used for path traversal.
func (c *Client) buildURL(req home.HomeRequest) (string, error) {
	base, err := url.Parse(c.cfg.BaseURL)
	if err != nil {
		return "", err
	}
	contentType := contentTypeFromChannel(req.Channel)
	base.Path = fmt.Sprintf("/content/%s/%s/%s", contentType, req.Locale, c.cfg.HomePageID)

	if req.Channel != "" {
		q := base.Query()
		q.Set("channel", req.Channel)
		base.RawQuery = q.Encode()
	}
	return base.String(), nil
}

// buildHeaders assembles the outbound headers. Only the x-brand-id and
// correlation headers are forwarded — arbitrary client headers are never
// passed through (header trust boundary rule).
func (c *Client) buildHeaders(req home.HomeRequest) map[string]string {
	brandID := req.Brand
	if req.Preview {
		brandID += "-PREVIEW"
	}
	return map[string]string{
		"x-brand-id": brandID,
		"Accept":     "application/json",
	}
}

// contentTypeFromChannel maps the channel parameter to a Contentstack content
// type, mirroring the legacy BFF routing (external contract only).
func contentTypeFromChannel(channel string) string {
	switch channel {
	case "pocket", "kiosk", "mpos":
		return "screen"
	default:
		return "page"
	}
}

// mapError converts transport and HTTP errors to domain AppErrors.
// Breaker-open and timeout errors become SERVICE_UNAVAILABLE / TIMEOUT.
// This is the only place these errors are logged — the caller must not log
// them again (logger-handler skill: log exactly once at the highest layer).
func (c *Client) mapError(ctx context.Context, dep, rawURL string, latency time.Duration, err error) *home.AppError {
	logAttrs := []any{
		"dependency", dep,
		"url", rawURL,
		"latency_ms", latency.Milliseconds(),
		"request_id", httpx.RequestIDFromCtx(ctx),
		"correlation_id", httpx.CorrelationIDFromCtx(ctx),
	}

	if breaker.IsOpen(err) {
		appErr := home.ErrServiceUnavailable(dep, err)
		c.log.ErrorContext(ctx, "circuit breaker open", append(logAttrs, "error_code", string(appErr.Code))...)
		return appErr
	}

	var statusErr *httpStatusError
	if errors.As(err, &statusErr) {
		if statusErr.code == http.StatusNotFound {
			appErr := home.ErrNotFound("home layout")
			c.log.WarnContext(ctx, "content not found", append(logAttrs, "http_status", statusErr.code)...)
			return appErr
		}
		appErr := home.ErrServiceUnavailable(dep, err)
		c.log.ErrorContext(ctx, "content-service error response",
			append(logAttrs, "http_status", statusErr.code, "error_code", string(appErr.Code))...)
		return appErr
	}

	if errors.Is(err, context.DeadlineExceeded) {
		appErr := home.ErrRequestTimeout(dep, err)
		c.log.ErrorContext(ctx, "outbound timeout", append(logAttrs, "error_code", string(appErr.Code))...)
		return appErr
	}

	appErr := home.ErrUnexpected("content-service call", err)
	c.log.ErrorContext(ctx, "unexpected outbound error", append(logAttrs, "error", err.Error())...)
	return appErr
}

// httpStatusError carries the HTTP status code from a non-2xx response so
// callers can distinguish 404 from 5xx without reading the body twice.
type httpStatusError struct {
	code int
	body []byte
}

func (e *httpStatusError) Error() string {
	return fmt.Sprintf("content-service responded with HTTP %d", e.code)
}
