package meter

import (
	"context"
	"errors"
	"log/slog"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var (
	meter metric.Meter
)

func New(name string) {
	meter = otel.Meter(name)
}

type KeyValue struct {
	Key   string
	Value any
}

func transformKeyValue(attrs ...KeyValue) attribute.Set {
	otelAttrs := make([]attribute.KeyValue, len(attrs))

	for _, attr := range attrs {
		otelAttr := attribute.KeyValue{
			Key: attribute.Key(attr.Key),
		}
		switch v := any(attr.Value).(type) {
		case string:
			otelAttr.Value = attribute.StringValue(v)
		case int:
			otelAttr.Value = attribute.IntValue(v)
		case float64:
			otelAttr.Value = attribute.Float64Value(v)
		case bool:
			otelAttr.Value = attribute.BoolValue(v)
		}

		otelAttrs = append(otelAttrs, otelAttr)
	}

	return attribute.NewSet(otelAttrs...)
}

type AttributeGenerator func() []KeyValue

func WithAttribute(key string, value any) AttributeGenerator {
	return func() []KeyValue {
		return []KeyValue{
			{
				Key:   key,
				Value: value,
			},
		}
	}
}

func WithAttributes(attrs ...any) AttributeGenerator {
	if len(attrs)%2 != 0 {
		panic(errors.New("`WithAttributes` needs to receive even number of params, being the first a string key and any value"))
	}
	return func() []KeyValue {
		attrsKeyValues := make([]KeyValue, len(attrs)/2)
		offset := 0
		for i := range len(attrs) / 2 {
			k := attrs[i+offset]
			v := attrs[i+offset+1]
			kv := KeyValue{
				Key:   k.(string),
				Value: v,
			}

			attrsKeyValues = append(attrsKeyValues, kv)

			offset += 1
		}
		return attrsKeyValues
	}
}

func Counter(ctx context.Context, name string, incr int64, attrsGens ...AttributeGenerator) {
	if meter == nil {
		return
	}

	attributes := make([]KeyValue, 0)
	for _, gen := range attrsGens {
		attributes = append(attributes, gen()...)
	}

	counter, err := meter.Int64Counter(name)
	if err != nil {
		slog.Error("failed to use meter", "error", err)
		return
	}

	counter.Add(ctx, incr, metric.WithAttributeSet(transformKeyValue(attributes...)))
}

func Histogram(ctx context.Context, name string, incr int64, attrsGens ...AttributeGenerator) {
	if meter == nil {
		return
	}

	attributes := make([]KeyValue, 0)
	for _, gen := range attrsGens {
		attributes = append(attributes, gen()...)
	}

	counter, err := meter.Int64Histogram(name)
	if err != nil {
		slog.Error("failed to use meter", "error", err)
		return
	}

	counter.Record(ctx, incr, metric.WithAttributeSet(transformKeyValue(attributes...)))
}
