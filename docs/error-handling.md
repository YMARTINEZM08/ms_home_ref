# Error Handling

## AppError model

```go
type AppError struct {
    Code      ErrorCode  // machine-readable, stable across versions
    Category  Category   // Validation | Business | ResourceNotFound | ExternalService | Timeout | Configuration | Infrastructure | Unexpected
    Status    int        // HTTP status to return
    Message   string     // consumer-safe, never leaks internals
    Detail    string     // developer-facing; never sent in responses
    Retryable bool       // hint to caller / frontend
    Cause     error      // internal; never serialized
}
```

## Error codes

| Code | HTTP | Retryable | Meaning |
|---|---|---|---|
| `BLOCK_TEMPORARILY_DISABLED` | 423 | false | Feature flag is off |
| `SERVICE_UNAVAILABLE` | 503 | true | Circuit breaker open / downstream down |
| `NOT_FOUND` | 404 | false | Content entry not found |
| `BAD_REQUEST` | 400 | false | Invalid input |
| `REQUEST_TIMEOUT` | 504 | true | Downstream timeout |
| `CONFIGURATION_ERROR` | 500 | false | Missing/invalid config at startup |
| `UNEXPECTED_ERROR` | 500 | false | Unclassified failure |

## Logging discipline

**Log each failure exactly once, at the highest responsible layer that has full context.**

- **Outbound adapters** (`contentservice/client.go`) — log the failure once with all outbound context (url, status, latency, error code), then return `*AppError`.
- **Application services** — never re-log errors received from adapters. Log one `INFO` on success with aggregate counts.
- **HTTP handlers** — never re-log errors received from use cases. Log `WARN` for input validation failures (not errors). The error carries its own status.
- **Middleware** — access log captures method/path/status/latency per request; health endpoints excluded.

**What gets logged:**
- Startup and shutdown events (always `INFO`)
- Config / external / unexpected failures (`ERROR`)
- Breaker state transitions (`WARN`)
- Successful layout composition (`INFO`, block counts)
- Input validation rejections (`WARN`)
- Debug: masked cURL of every outbound request (`DEBUG`)

**What never gets logged:**
- `Authorization`, cookies, tokens, or any PII
- `AppError.Detail` or `AppError.Cause` at a level visible to non-engineers
- Per-method calls on hot paths at `INFO` or above

## Runtime log level

`LOG_LEVEL` env var; valid values: `OFF`, `ERROR`, `WARN`, `INFO`, `DEBUG`, `TRACE` (alias for DEBUG).
Change takes effect on next request — no redeploy needed (uses `slog.LevelVar`).

## Response contract

HTTP error responses carry only safe fields:

```json
{
  "error_code": "SERVICE_UNAVAILABLE",
  "message":    "service not available at this moment",
  "retryable":  true
}
```

`Detail`, `Cause`, and stack traces are never included.
