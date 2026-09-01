package telemetry

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/andy98w/Kubernetes-Dashboard/api/internal/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type Shutdown func(context.Context) error

func Setup(ctx context.Context, cfg config.Config) (Shutdown, error) {
	if strings.EqualFold(os.Getenv("OTEL_SDK_DISABLED"), "true") || !exportConfigured() {
		return func(context.Context) error { return nil }, nil
	}

	exporter, err := otlptracegrpc.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("create OTLP trace exporter: %w", err)
	}

	serviceResource, err := newServiceResource(cfg)
	if err != nil {
		return nil, fmt.Errorf("create OpenTelemetry resource: %w", err)
	}

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(serviceResource),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(sampleRatio()))),
	)
	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
	return provider.Shutdown, nil
}

func newServiceResource(cfg config.Config) (*resource.Resource, error) {
	// These attributes describe KubeVista itself rather than a specific semantic
	// convention schema. Keeping them schemaless allows them to merge with the
	// SDK's versioned default resource as OpenTelemetry dependencies evolve.
	return resource.Merge(resource.Default(), resource.NewSchemaless(
		attribute.String("service.name", "kubevista-api"),
		attribute.String("service.version", cfg.Version),
		attribute.String("deployment.environment.name", cfg.Environment),
		attribute.String("k8s.cluster.name", cfg.ClusterName),
	))
}

func exportConfigured() bool {
	return os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") != "" || os.Getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT") != ""
}

func sampleRatio() float64 {
	const fallback = 0.25
	value := os.Getenv("KUBEVISTA_TRACE_SAMPLE_RATIO")
	if value == "" {
		return fallback
	}
	ratio, err := strconv.ParseFloat(value, 64)
	if err != nil || ratio < 0 || ratio > 1 {
		return fallback
	}
	return ratio
}
