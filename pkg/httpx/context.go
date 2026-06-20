package httpx

import "context"

type ctxKey string

const (
	CtxKeyRequestID     ctxKey = "request_id"
	CtxKeyCorrelationID ctxKey = "correlation_id"
)

func RequestIDFromCtx(ctx context.Context) string {
	v, _ := ctx.Value(CtxKeyRequestID).(string)
	return v
}

func CorrelationIDFromCtx(ctx context.Context) string {
	v, _ := ctx.Value(CtxKeyCorrelationID).(string)
	return v
}

func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, CtxKeyRequestID, id)
}

func WithCorrelationID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, CtxKeyCorrelationID, id)
}
