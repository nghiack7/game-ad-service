package logger

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type contextKey string

// String returns contextKey as string.
func (ck contextKey) String() string {
	return string(ck)
}

// Context key definition.
// RqIDCtxKey is request ID context key.
// RqClientIPCtxKey is request client IP context key.
// ExecTimeCtxKey is execute time context key.
var (
	RqIDCtxKey       = contextKey("request_id")
	RqClientIPCtxKey = contextKey("client_ip")
	RqExecTimeCtxKey = contextKey("exec_time")
	RqURICtxKey      = contextKey("request_uri")
)

// GetRqIDFromCtx get request ID in context and returns as string.
func GetRqIDFromCtx(ctx context.Context) string {
	return getStringFromCtx(ctx, RqIDCtxKey)
}

// GetRqClientIPFromCtx get client IP in context and returns as string.
func GetRqClientIPFromCtx(ctx context.Context) string {
	return getStringFromCtx(ctx, RqClientIPCtxKey)
}

// SetRqIDToCtx set request ID to context
func SetRqIDToCtx(ctx context.Context) context.Context {
	return context.WithValue(ctx, RqIDCtxKey, NewRequestID())
}

// GetRqExecTimeFromCtx get exec time in context and returns as string.
func GetRqExecTimeFromCtx(ctx context.Context) float64 {
	if ctx != nil {
		if val, ok := ctx.Value(RqExecTimeCtxKey).(time.Time); ok {
			elapsedTime := time.Now().Sub(val)
			return float64(elapsedTime.Nanoseconds()) / 1000000
		}
	}

	return 0
}

// SetTimeToCtx get exec time in context and returns as string.
func SetTimeToCtx(ctx context.Context) context.Context {
	return context.WithValue(ctx, RqExecTimeCtxKey, time.Now())
}

// GetRqURIFromCtx get exec time in context and returns as string.
func GetRqURIFromCtx(ctx context.Context) string {
	return getStringFromCtx(ctx, RqURICtxKey)
}

// getStringFromCtx returns value string in context.
func getStringFromCtx(ctx context.Context, key contextKey) string {
	if ctx != nil {
		if val, ok := ctx.Value(key).(string); ok {
			return val
		}
	}
	return ""
}

// NewRequestID returns new request ID as string.
func NewRequestID() string {
	return uuid.New().String()
}

// WithRqID returns a context which knows its request ID
func WithRqID(ctx context.Context, rqID string) context.Context {
	return context.WithValue(ctx, RqIDCtxKey, rqID)
}
