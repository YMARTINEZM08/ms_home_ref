package home

import "fmt"

// Category classifies the nature of an error for routing and logging decisions.
type Category string

const (
	CategoryValidation      Category = "VALIDATION"
	CategoryBusiness        Category = "BUSINESS"
	CategoryResourceNotFound Category = "RESOURCE_NOT_FOUND"
	CategoryExternalService  Category = "EXTERNAL_SERVICE"
	CategoryTimeout          Category = "TIMEOUT"
	CategoryConfiguration    Category = "CONFIGURATION"
	CategoryInfrastructure   Category = "INFRASTRUCTURE"
	CategoryUnexpected       Category = "UNEXPECTED"
)

// ErrorCode is a stable, machine-readable identifier sent to API consumers.
type ErrorCode string

const (
	ErrCodeBlockDisabled    ErrorCode = "BLOCK_TEMPORARILY_DISABLED"
	ErrCodeServiceUnavailable ErrorCode = "SERVICE_UNAVAILABLE"
	ErrCodeNotFound         ErrorCode = "NOT_FOUND"
	ErrCodeBadRequest       ErrorCode = "BAD_REQUEST"
	ErrCodeTimeout          ErrorCode = "TIMEOUT"
	ErrCodeUnexpected       ErrorCode = "UNEXPECTED_ERROR"
	ErrCodeConfiguration    ErrorCode = "CONFIGURATION_ERROR"
)

// AppError is the single error type used throughout the service.
// - Message is safe to send to API consumers.
// - Detail explains the root cause for developers/operators (never exposed in responses).
// - Cause is the original error and is only used for internal logging (never exposed).
type AppError struct {
	Code     ErrorCode
	Category Category
	Status   int
	Message  string
	Detail   string
	Retryable bool
	Cause    error
}

func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Detail, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Detail)
}

func (e *AppError) Unwrap() error { return e.Cause }

// Constructors — each one is a named, explicit error. Never use generic messages.

func ErrBlockDisabled(blockType BlockType, flagID string) *AppError {
	return &AppError{
		Code:      ErrCodeBlockDisabled,
		Category:  CategoryBusiness,
		Status:    423,
		Message:   "This section is currently turned off.",
		Detail:    fmt.Sprintf("block type %q is disabled via feature flag %q", blockType, flagID),
		Retryable: false,
	}
}

func ErrServiceUnavailable(dependency string, cause error) *AppError {
	return &AppError{
		Code:      ErrCodeServiceUnavailable,
		Category:  CategoryInfrastructure,
		Status:    503,
		Message:   "service not available at this moment",
		Detail:    fmt.Sprintf("circuit breaker is open for dependency %q — too many recent failures", dependency),
		Retryable: true,
		Cause:     cause,
	}
}

func ErrNotFound(resource string) *AppError {
	return &AppError{
		Code:      ErrCodeNotFound,
		Category:  CategoryResourceNotFound,
		Status:    404,
		Message:   fmt.Sprintf("%s could not be found.", resource),
		Detail:    fmt.Sprintf("%s was not found in the content-service response", resource),
		Retryable: false,
	}
}

func ErrBadRequest(reason string) *AppError {
	return &AppError{
		Code:      ErrCodeBadRequest,
		Category:  CategoryValidation,
		Status:    400,
		Message:   "The request contains invalid parameters.",
		Detail:    reason,
		Retryable: false,
	}
}

func ErrRequestTimeout(dependency string, cause error) *AppError {
	return &AppError{
		Code:      ErrCodeTimeout,
		Category:  CategoryTimeout,
		Status:    504,
		Message:   "The request timed out. Please try again.",
		Detail:    fmt.Sprintf("outbound call to %q exceeded the configured timeout", dependency),
		Retryable: true,
		Cause:     cause,
	}
}

func ErrConfiguration(field string, cause error) *AppError {
	return &AppError{
		Code:      ErrCodeConfiguration,
		Category:  CategoryConfiguration,
		Status:    500,
		Message:   "The service is not correctly configured.",
		Detail:    fmt.Sprintf("configuration field %q is missing or invalid", field),
		Retryable: false,
		Cause:     cause,
	}
}

func ErrUnexpected(op string, cause error) *AppError {
	return &AppError{
		Code:      ErrCodeUnexpected,
		Category:  CategoryUnexpected,
		Status:    500,
		Message:   "An unexpected error occurred. Please try again later.",
		Detail:    fmt.Sprintf("unexpected failure in operation %q", op),
		Retryable: false,
		Cause:     cause,
	}
}
