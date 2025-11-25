package logger

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel/log"
)

type otelHandler struct {
	otelLogger log.Logger

	logHandler slog.Handler
}

func (h *otelHandler) Enabled(_ context.Context, _ slog.Level) bool {
	return true
}

func (h *otelHandler) Handle(ctx context.Context, r slog.Record) error {
	h.logHandler.Handle(ctx, r)

	attrs := make([]log.KeyValue, 0, r.NumAttrs())
	r.Attrs(func(a slog.Attr) bool {
		attrs = append(attrs, log.String(a.Key, a.Value.String()))
		return true
	})

	otelRecord := log.Record{}
	otelRecord.SetTimestamp(r.Time)
	otelRecord.SetSeverityText(r.Level.String())
	otelRecord.SetBody(log.StringValue(r.Message))
	otelRecord.AddAttributes(attrs...)

	h.otelLogger.Emit(ctx, otelRecord)
	return nil
}

func (h *otelHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}

func (h *otelHandler) WithGroup(name string) slog.Handler {
	return h
}

func NewOtelHandler(otelLogger log.Logger, logHandler slog.Handler) {
	slog.SetDefault(
		slog.New(&otelHandler{otelLogger: otelLogger, logHandler: logHandler}),
	)
}
