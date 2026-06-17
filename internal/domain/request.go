package domain

import (
	"context"
	"sync"
)

// RequestInfo is the explicit per-request context replacing digital_bff's
// async-local RequestContext. Carried via context.Context (no global state) so
// the service stays stateless and Cloud Run friendly.
type RequestInfo struct {
	RequestID     string
	CorrelationID string
	Brand         string
	Source        Source
	Preview       bool
	LoggedIn      bool
	Locale        string

	// Identity used by personalization / GroupBy (loginId, gbi_visitorId cookie).
	ProfileID string
	VisitorID string

	// Jewel identity (x-jml-user-id / x-jml-device-id headers).
	JewelUserID   string
	JewelDeviceID string

	// Cookie is the raw inbound Cookie header, forwarded to ATG.
	Cookie string

	// Client metadata headers (x-client-*) forwarded to GroupBy metrics.
	ClientPage      string
	ClientChannel   string
	ClientAction    string
	ClientComponent string

	// FeatureFlags carries the effective flags (env gate already applied) so
	// strategies can read them without re-fetching GLOBAL.
	FeatureFlags map[string]bool

	// State is per-request mutable scratch shared across goroutines (event index
	// counter, selected store). Pointer so copies of RequestInfo share it.
	State *RequestState
}

// SelectedStore is the user's favorite store (mirrors reqContext.cache.selectedStore).
type SelectedStore struct {
	ID   int
	Name string
}

// RequestState holds per-request mutable values used during populate. Safe for
// concurrent use (populate runs block containers in parallel).
type RequestState struct {
	mu               sync.Mutex
	tagIndex         int
	selectedStore    *SelectedStore
	cartHeader       map[string]any
	cartHeaderLoaded bool

	sfMu   sync.Mutex
	sfOnce map[string]*sfEntry
}

type sfEntry struct {
	once sync.Once
	data map[string]any
	err  error
}

// NewRequestState returns state with tag_index initialized to 1 (context-defaults.middleware.ts).
func NewRequestState() *RequestState {
	return &RequestState{tagIndex: 1}
}

// NextTagIndex returns the current tag index then increments it.
func (s *RequestState) NextTagIndex() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	v := s.tagIndex
	s.tagIndex++
	return v
}

// SetSelectedStore records the resolved favorite store.
func (s *RequestState) SetSelectedStore(store *SelectedStore) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.selectedStore = store
}

// SelectedStore returns the resolved favorite store (nil if none).
func (s *RequestState) SelectedStore() *SelectedStore {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.selectedStore
}

// SetCartHeader memoizes the ATG cart header details for the request.
func (s *RequestState) SetCartHeader(m map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cartHeader = m
	s.cartHeaderLoaded = true
}

// CartHeader returns the memoized cart header details and whether it was loaded.
func (s *RequestState) CartHeader() (map[string]any, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cartHeader, s.cartHeaderLoaded
}

// SalesforceAction runs compute once per action per request (mirrors digital_bff's
// reqContext.cache.salesforce dedup), so repeated blocks share a single call.
func (s *RequestState) SalesforceAction(action string, compute func() (map[string]any, error)) (map[string]any, error) {
	s.sfMu.Lock()
	if s.sfOnce == nil {
		s.sfOnce = make(map[string]*sfEntry)
	}
	e := s.sfOnce[action]
	if e == nil {
		e = &sfEntry{}
		s.sfOnce[action] = e
	}
	s.sfMu.Unlock()

	e.once.Do(func() { e.data, e.err = compute() })
	return e.data, e.err
}

// BrandHeader returns the value for the x-brand-id header, appending -PREVIEW in
// preview mode (mirrors content.provider.ts).
func (r RequestInfo) BrandHeader() string {
	if r.Preview {
		return r.Brand + "-PREVIEW"
	}
	return r.Brand
}

// Flag reports whether the named effective feature flag is enabled.
func (r RequestInfo) Flag(name string) bool {
	return r.FeatureFlags[name]
}

type ctxKey struct{}

// WithRequestInfo stores per-request info on the context.
func WithRequestInfo(ctx context.Context, ri RequestInfo) context.Context {
	return context.WithValue(ctx, ctxKey{}, ri)
}

// RequestInfoFromContext retrieves per-request info, or a zero value if absent.
func RequestInfoFromContext(ctx context.Context) RequestInfo {
	if ri, ok := ctx.Value(ctxKey{}).(RequestInfo); ok {
		return ri
	}
	return RequestInfo{}
}
