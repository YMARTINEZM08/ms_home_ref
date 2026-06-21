package handler

import (
	"net/http"

	domain "github.com/YMARTINEZM08/ms_home_ref/internal/domain/home"
)

// errorResponse is the stable, frontend-safe error envelope.
// It never exposes stack traces, Detail, or Cause (logger-handler skill).
type errorResponse struct {
	ErrorCode string `json:"error_code"`
	Message   string `json:"message"`
	Retryable bool   `json:"retryable"`
}

// writeError maps a domain AppError to an HTTP status + JSON body.
// This is the only place domain errors become HTTP responses — no middleware
// or filter added solely for this purpose (logger-handler skill).
func writeError(w http.ResponseWriter, err *domain.AppError) {
	writeJSON(w, err.Status, errorResponse{
		ErrorCode: string(err.Code),
		Message:   err.Message,
		Retryable: err.Retryable,
	})
}
