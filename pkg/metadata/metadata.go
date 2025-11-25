package metadata

import "context"

type contextMetadataKey string

const metadataKey contextMetadataKey = "metadata"

type Metadata map[string]any

func FromContext(ctx context.Context) Metadata {
	if v := ctx.Value(metadataKey); v != nil {
		if md, ok := v.(Metadata); ok {
			return md
		}
	}

	return Metadata{}
}

func NewContext(ctx context.Context, md Metadata) context.Context {
	return context.WithValue(ctx, metadataKey, md)
}
