package observability

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

func TestInitTracingDisabledSetsPropagator(t *testing.T) {
	shutdown, err := InitTracing(context.Background(), "ms_home", "v1", "test", false)
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	t.Cleanup(func() { _ = shutdown(context.Background()) })

	// W3C trace context propagation must be installed even when export is disabled,
	// so an inbound traceparent still flows to downstreams.
	prop := otel.GetTextMapPropagator()
	carrier := propagation.MapCarrier{
		"traceparent": "00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01",
	}
	ctx := prop.Extract(context.Background(), carrier)

	out := propagation.MapCarrier{}
	prop.Inject(ctx, out)
	if out["traceparent"] == "" {
		t.Error("expected traceparent to propagate through extract/inject")
	}
}
