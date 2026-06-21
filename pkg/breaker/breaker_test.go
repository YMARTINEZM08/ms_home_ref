package breaker_test

import (
	"errors"
	"testing"
	"time"

	"github.com/YMARTINEZM08/ms_home_ref/pkg/breaker"
)

func settings() breaker.Settings {
	return breaker.Settings{
		FailureRatio: 0.05,
		MinRequests:  20,
		OpenTimeout:  1 * time.Second,
	}
}

func TestBreaker_TripsAt5PercentFailureRatio(t *testing.T) {
	b := breaker.New[string]("test-dep", settings())

	// 19 successes
	for range 19 {
		_, err := b.Execute(func() (string, error) { return "ok", nil })
		if err != nil {
			t.Fatalf("unexpected error on success call: %v", err)
		}
	}
	// 20th request fails → ratio = 1/20 = 5% → breaker trips
	_, _ = b.Execute(func() (string, error) { return "", errors.New("downstream down") })

	// 21st call — breaker must now be open
	_, err := b.Execute(func() (string, error) { return "ok", nil })
	if !breaker.IsOpen(err) {
		t.Fatalf("expected breaker to be open after 5%% failure ratio, got: %v", err)
	}
}

func TestBreaker_DoesNotTripBelowMinRequests(t *testing.T) {
	b := breaker.New[string]("test-dep", breaker.Settings{
		FailureRatio: 0.05,
		MinRequests:  20,
		OpenTimeout:  1 * time.Second,
	})

	// 19 requests: all fail — but minimum not reached yet, so breaker stays closed
	for range 19 {
		_, _ = b.Execute(func() (string, error) { return "", errors.New("fail") })
	}
	_, err := b.Execute(func() (string, error) { return "still-closed", nil })
	if breaker.IsOpen(err) {
		t.Fatal("breaker should not open before MinRequests threshold is reached")
	}
}

func TestBreaker_NoRetries(t *testing.T) {
	b := breaker.New[string]("test-dep", breaker.Settings{
		FailureRatio: 0.99, // high threshold so breaker stays closed
		MinRequests:  100,
		OpenTimeout:  1 * time.Second,
	})

	calls := 0
	_, _ = b.Execute(func() (string, error) {
		calls++
		return "", errors.New("fail")
	})

	if calls != 1 {
		t.Errorf("expected exactly 1 call (no retries), got %d", calls)
	}
}

func TestBreaker_IsOpen_ReturnsFalseOnNilError(t *testing.T) {
	if breaker.IsOpen(nil) {
		t.Error("IsOpen(nil) must return false")
	}
}

func TestBreaker_IsOpen_ReturnsFalseOnOtherErrors(t *testing.T) {
	if breaker.IsOpen(errors.New("some other error")) {
		t.Error("IsOpen must return false for non-breaker errors")
	}
}
