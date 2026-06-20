package httpx

import "strings"

// sensitiveHeaders lists headers whose values must never appear in logs,
// traces, or cURL debug output.
var sensitiveHeaders = map[string]bool{
	"authorization": true,
	"cookie":        true,
	"set-cookie":    true,
	"x-api-key":     true,
	"x-auth-token":  true,
	"x-access-token": true,
}

const masked = "***"

// MaskSensitiveHeaders returns a copy of headers with sensitive values
// replaced by "***". Safe to pass to slog or cURL builders.
func MaskSensitiveHeaders(headers map[string]string) map[string]string {
	out := make(map[string]string, len(headers))
	for k, v := range headers {
		if sensitiveHeaders[strings.ToLower(k)] {
			out[k] = masked
		} else {
			out[k] = v
		}
	}
	return out
}
