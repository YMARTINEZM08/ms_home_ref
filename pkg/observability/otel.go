package observability

import (
	"context"
	"fmt"
	"net/url"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// Init initialises the global OpenTelemetry TracerProvider and returns a
// shutdown function that flushes pending spans before process exit.
//
// If endpoint is empty, a noop provider is installed — the service runs
// normally without emitting traces. OTel failure is never fatal.
func Init(ctx context.Context, serviceName, endpoint, env string, sampleRatio float64) (func(context.Context) error, error) {
	if endpoint == "" {
		otel.SetTracerProvider(noop.NewTracerProvider())
		return func(_ context.Context) error { return nil }, nil
	}

	host, insecure := parseEndpoint(endpoint)

	opts := []otlptracehttp.Option{otlptracehttp.WithEndpoint(host)}
	if insecure {
		opts = append(opts, otlptracehttp.WithInsecure())
	}

	exporter, err := otlptracehttp.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("otlp exporter: %w", err)
	}

	res, _ := resource.Merge(
		resource.Default(),
		resource.NewSchemaless(
			attribute.String("service.name", serviceName),
			attribute.String("deployment.environment", env),
		),
	)

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(sampleRatio))),
	)

	otel.SetTracerProvider(tp)
	return tp.Shutdown, nil
}

// parseEndpoint splits a raw endpoint string (which may include a scheme)
// into host:port and an insecure flag. WithEndpoint expects host:port only.
func parseEndpoint(raw string) (host string, insecure bool) {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return raw, true
	}
	return u.Host, u.Scheme != "https"
}
