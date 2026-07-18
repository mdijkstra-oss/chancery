package logging

import (
	"context"
	"log/slog"
)

type ctxKey struct{}

func WithAttr(ctx context.Context, key, value string) context.Context {
	return WithAttrs(ctx, slog.String(key, value))
}

func WithAttrs(ctx context.Context, added ...slog.Attr) context.Context {
	existing := AttrsFromContext(ctx)
	attrs := make([]slog.Attr, 0, len(existing)+len(added))
	attrs = append(attrs, existing...)
	attrs = append(attrs, added...)
	return context.WithValue(ctx, ctxKey{}, attrs)
}

func AttrsFromContext(ctx context.Context) []slog.Attr {
	attrs, _ := ctx.Value(ctxKey{}).([]slog.Attr)
	return attrs
}

type ContextHandler struct {
	inner slog.Handler
}

func NewContextHandler(inner slog.Handler) *ContextHandler {
	return &ContextHandler{inner: inner}
}

func (h *ContextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *ContextHandler) Handle(ctx context.Context, record slog.Record) error {
	for _, attr := range AttrsFromContext(ctx) {
		record.AddAttrs(attr)
	}
	return h.inner.Handle(ctx, record)
}

func (h *ContextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ContextHandler{inner: h.inner.WithAttrs(attrs)}
}

func (h *ContextHandler) WithGroup(name string) slog.Handler {
	return &ContextHandler{inner: h.inner.WithGroup(name)}
}
