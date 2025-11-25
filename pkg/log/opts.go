package log

import (
	"log/slog"
	"os"
)

type opt func(*params) *params

type params struct {
	fallbackHandler slog.Handler
}

func defaultParams() *params {
	return &params{
		slog.NewJSONHandler(os.Stdout, nil),
	}
}

func WithFallbackHandler(handler slog.Handler) opt {
	return func(p *params) *params {
		p.fallbackHandler = handler
		return p
	}
}
