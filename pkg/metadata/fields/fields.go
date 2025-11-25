package fields

import (
	"context"
	"fmt"

	"github.com/lccmrx/go-swiss-knife/pkg/metadata"
)

type field interface {
	Key() string
	Value(context.Context) string
}

type Field string

func (f Field) Key() string {
	return string(f)
}

func (f Field) Value(ctx context.Context) string {
	metadata := metadata.FromContext(ctx)

	if value, ok := metadata[f.Key()]; ok {
		return fmt.Sprintf("%+v", value)
	}
	return ""
}

var FieldKeysMap = map[field]struct{}{
	RequestID: {},
	TraceID:   {},
	SpanID:    {},
}
