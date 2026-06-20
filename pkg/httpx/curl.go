package httpx

import (
	"fmt"
	"strings"
)

// BuildCurlCommand produces a masked, replayable cURL string for DEBUG logging.
// Sensitive headers are automatically redacted.
func BuildCurlCommand(method, rawURL string, headers map[string]string) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "curl -X %s '%s'", method, rawURL)
	for k, v := range MaskSensitiveHeaders(headers) {
		fmt.Fprintf(&sb, " -H '%s: %s'", k, v)
	}
	return sb.String()
}
