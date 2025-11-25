package log

import (
	"context"
	"log/slog"

	"github.com/lccmrx/go-swiss-knife/pkg/metadata"
	"github.com/lccmrx/go-swiss-knife/pkg/metadata/fields"
)

func NewHandler(opts ...opt) slog.Handler {
	p := defaultParams()
	for _, opt := range opts {
		p = opt(p)
	}

	return &handler{
		fallbackHandler: p.fallbackHandler,
	}
}

type handler struct {
	fallbackHandler slog.Handler
}

func (h *handler) Handle(ctx context.Context, record slog.Record) error {
	md := metadata.FromContext(ctx)

	attrs := make([]slog.Attr, 0, len(md))
	additionalAttrs := make([]slog.Attr, 0, len(md))
	for k, v := range md {
		if _, ok := fields.FieldKeysMap[fields.Field(k)]; !ok {
			additionalAttrs = append(additionalAttrs, slog.Any(k, v))
			continue
		}
		attr := slog.Any(k, v)
		attrs = append(attrs, attr)
	}

	record.AddAttrs(attrs...)
	record.AddAttrs(slog.GroupAttrs("additional", additionalAttrs...))

	return h.fallbackHandler.Handle(ctx, record)
}

func (h *handler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.fallbackHandler.Enabled(ctx, level)
}

func (h *handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &handler{
		h.fallbackHandler.WithAttrs(attrs),
	}
}

func (h *handler) WithGroup(name string) slog.Handler {
	return &handler{
		h.fallbackHandler.WithGroup(name),
	}
}
