package panyl

import (
	"context"
	"log/slog"
)

type slogContextKey string

const (
	// slogLoggerCtxKey is the context key used to store the logger
	slogLoggerCtxKey slogContextKey = "logger"
)

var emptySLogger *slog.Logger

// SLogToContext returns a context that carries the logger.
func SLogToContext(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, slogLoggerCtxKey, logger)
}

// SLogFromContext returns the logger stored by SLogToContext, or a logger that discards everything.
func SLogFromContext(ctx context.Context) *slog.Logger {
	v, ok := ctx.Value(slogLoggerCtxKey).(*slog.Logger)
	if ok {
		return v
	}
	return emptySLogger
}

func init() {
	emptySLogger = slog.New(&discardHandler{})
}

type discardHandler struct{}

func (dh discardHandler) Enabled(context.Context, slog.Level) bool  { return false }
func (dh discardHandler) Handle(context.Context, slog.Record) error { return nil }
func (dh discardHandler) WithAttrs(attrs []slog.Attr) slog.Handler  { return dh }
func (dh discardHandler) WithGroup(name string) slog.Handler        { return dh }
