package domain

import "errors"

// Explicit business errors, mirroring digital_bff content/category exceptions.
// Infrastructure errors are wrapped by adapters and preserve their root cause
// (skill Rule 12).
var (
	ErrNoContentType = errors.New("content: response missing _content_type_uid")
	ErrNoCategoryID  = errors.New("content: response missing category_id")
	ErrContentType   = errors.New("content: unsupported content type")
)
