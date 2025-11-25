package telemetry

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"time"

	"github.com/lccmrx/go-swiss-knife/pkg/telemetry/logger"
	"github.com/lccmrx/go-swiss-knife/pkg/telemetry/meter"
	"github.com/lccmrx/go-swiss-knife/pkg/telemetry/tracer"

	"go.opentelemetry.io/contrib/instrumentation/host"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
	nooplog "go.opentelemetry.io/otel/log/noop"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"

	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/metric"
	noopmetric "go.opentelemetry.io/otel/metric/noop"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	nooptrace "go.opentelemetry.io/otel/trace/noop"

	"go.opentelemetry.io/otel/sdk/resource"
)

var telemetryInstance *telemetry

type telemetry struct {
	serviceName string

	meterExporter  *otlpmetricgrpc.Exporter
	traceExporter  *otlptrace.Exporter
	loggerExporter *otlploggrpc.Exporter
}

func New(collectorEndpoint *url.URL, opts ...opt) (err error) {
	if os.Getenv("OTEL_SERVICE_NAME") == "" {
		slog.Warn("environment variable `OTEL_SERVICE_NAME` is not set; telemetry may not function as expected")
	}

	if collectorEndpoint == nil {
		collectorEndpoint, err = url.Parse("localhost:4317")
		if err != nil {
			return err
		}
	}

	config := &config{
		collectorEndpoint: collectorEndpoint,
	}
	for _, opt := range opts {
		config = opt(config)
	}

	res, err := resource.New(context.Background(),
		resource.WithFromEnv(),
		resource.WithProcess(),
		resource.WithTelemetrySDK(),
		resource.WithHost(),
		resource.WithAttributes(config.attributes...),
	)
	if err != nil {
		return fmt.Errorf("failed to create resource for telemetry: %w", err)
	}

	res, err = resource.Merge(
		resource.Default(),
		res,
	)
	if err != nil {
		return fmt.Errorf("failed to merge resource for telemetry: %w", err)
	}

	telemetryInstance = &telemetry{
		serviceName: os.Getenv("OTEL_SERVICE_NAME"),
	}

	telemetryInstance.setupProviders(config, res)

	err = host.Start()
	if err != nil {
		return fmt.Errorf("failed to start host instrumentation: %w", err)
	}

	err = runtime.Start()
	if err != nil {
		return fmt.Errorf("failed to start runtime instrumentation: %w", err)
	}

	return nil
}

func (o *telemetry) setupProviders(config *config, res *resource.Resource) error {
	var meterProvider metric.MeterProvider = noopmetric.MeterProvider{}
	if config.enabledMeterProvider {
		meterExporter, err := otlpmetricgrpc.New(context.Background(),
			otlpmetricgrpc.WithEndpoint(config.collectorEndpoint.String()),
		)
		if err != nil {
			return fmt.Errorf("failed to create meter exporter: %w", err)
		}

		meterProvider = sdkmetric.NewMeterProvider(
			sdkmetric.WithResource(res),
			sdkmetric.WithReader(
				sdkmetric.NewPeriodicReader(meterExporter,
					sdkmetric.WithInterval(10*time.Second),
				),
			),
		)
		o.meterExporter = meterExporter

		meter.New(o.serviceName)
	}
	otel.SetMeterProvider(meterProvider)

	var tracerProvider trace.TracerProvider = nooptrace.NewTracerProvider()
	if config.enabledTraceProvider {
		traceExporter, err := otlptracegrpc.New(context.Background(),
			otlptracegrpc.WithEndpoint(config.collectorEndpoint.String()),
		)
		if err != nil {
			return fmt.Errorf("failed to create trace exporter: %w", err)
		}

		tracerProvider = sdktrace.NewTracerProvider(
			sdktrace.WithResource(res),
			sdktrace.WithSampler(
				sdktrace.ParentBased(sdktrace.TraceIDRatioBased(.5)),
			),
			sdktrace.WithBatcher(traceExporter),
		)

		otel.SetTextMapPropagator(
			propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}),
		)
		o.traceExporter = traceExporter

		tracer.New(o.serviceName)
	}
	otel.SetTracerProvider(tracerProvider)

	var loggerProvider log.LoggerProvider = nooplog.NewLoggerProvider()
	if config.enabledLoggerProvider {
		logExporter, err := otlploggrpc.New(context.Background(),
			otlploggrpc.WithEndpoint(config.collectorEndpoint.String()),
		)
		if err != nil {
			return fmt.Errorf("failed to create log exporter: %w", err)
		}

		loggerProvider = sdklog.NewLoggerProvider(
			sdklog.WithResource(res),
			sdklog.WithProcessor(
				sdklog.NewBatchProcessor(logExporter)),
		)

		handler := slog.Default().Handler()
		logger.NewOtelHandler(
			loggerProvider.Logger(o.serviceName),
			handler,
		)
	}
	global.SetLoggerProvider(loggerProvider)

	return nil
}

func Shutdown(ctx context.Context) error {
	cxt, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if telemetryInstance == nil {
		return nil
	}

	if telemetryInstance.meterExporter != nil {
		telemetryInstance.meterExporter.Shutdown(cxt)
	}

	if telemetryInstance.traceExporter != nil {
		telemetryInstance.traceExporter.Shutdown(cxt)
	}

	if telemetryInstance.loggerExporter != nil {
		telemetryInstance.loggerExporter.Shutdown(cxt)
	}

	return nil
}
