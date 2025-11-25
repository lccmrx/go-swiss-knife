package telemetry

import (
	"net/url"
	"os"

	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
)

type opt func(*config) *config

type config struct {
	serviceName       string
	attributes        []attribute.KeyValue
	collectorEndpoint *url.URL

	enabledMeterProvider  bool
	enabledTraceProvider  bool
	enabledLoggerProvider bool
}

func WithServiceName(name string) opt {
	return func(config *config) *config {
		config.serviceName = name
		os.Setenv("OTEL_SERVICE_NAME", name)
		return config
	}
}

func WithResourceAttributes(attrs ...attribute.KeyValue) opt {
	return func(config *config) *config {
		config.attributes = append(config.attributes, attrs...)
		return config
	}
}

func WithEnvironment(env string) opt {
	return func(config *config) *config {
		config.attributes = append(config.attributes,
			semconv.DeploymentEnvironmentName(env),
		)
		return config
	}
}

func WithEnabledMeterProvider() opt {
	return func(config *config) *config {
		config.enabledMeterProvider = true
		return config
	}
}

func WithEnabledTraceProvider() opt {
	return func(config *config) *config {
		config.enabledTraceProvider = true
		return config
	}

}
func WithEnabledLoggerProvider() opt {
	return func(config *config) *config {
		config.enabledLoggerProvider = true
		return config
	}
}
