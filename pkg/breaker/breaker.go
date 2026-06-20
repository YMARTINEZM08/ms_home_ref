package breaker

import (
	"errors"
	"time"

	"github.com/sony/gobreaker/v2"
)

// Settings holds per-breaker configuration. All fields are populated from
// environment variables via internal/config — never hardcoded.
type Settings struct {
	// FailureRatio is the fraction of requests that must fail to trip the breaker.
	// Default (plan decision): 0.05 (5%).
	FailureRatio float64

	// MinRequests is the minimum number of requests in a window before the
	// failure ratio is evaluated. Prevents tripping on a tiny sample.
	MinRequests uint32

	// OpenTimeout is how long the breaker stays open before allowing a
	// half-open probe request.
	OpenTimeout time.Duration
}

// Breaker wraps sony/gobreaker and exposes a single Execute method.
// One Breaker instance must be created per outbound dependency so that a
// failing downstream never trips a sibling dependency's breaker.
type Breaker[T any] struct {
	cb   *gobreaker.CircuitBreaker[T]
	name string
}

// New creates a Breaker for dependency name with the given settings.
// There are no retries — a failed call is counted and fails fast.
func New[T any](name string, s Settings) *Breaker[T] {
	cb := gobreaker.NewCircuitBreaker[T](gobreaker.Settings{
		Name:        name,
		MaxRequests: 1, // single probe in half-open state
		Interval:    10 * time.Second,
		Timeout:     s.OpenTimeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			if counts.Requests < uint32(s.MinRequests) {
				return false
			}
			ratio := float64(counts.TotalFailures) / float64(counts.Requests)
			return ratio >= s.FailureRatio
		},
	})
	return &Breaker[T]{cb: cb, name: name}
}

// Execute runs fn inside the circuit breaker.
// Returns ErrOpen if the breaker is open (dependency considered unavailable).
// There are deliberately no retries — callers handle the open error immediately.
func (b *Breaker[T]) Execute(fn func() (T, error)) (T, error) {
	return b.cb.Execute(fn)
}

// IsOpen reports whether the returned error means the breaker is open.
// Use this in callers to convert to domain.ErrServiceUnavailable.
func IsOpen(err error) bool {
	return errors.Is(err, gobreaker.ErrOpenState) ||
		errors.Is(err, gobreaker.ErrTooManyRequests)
}

// Name returns the dependency name this breaker guards.
func (b *Breaker[T]) Name() string { return b.name }
