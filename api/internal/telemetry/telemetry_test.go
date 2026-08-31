package telemetry

import (
	"context"
	"testing"

	"github.com/andy98w/Kubernetes-Dashboard/api/internal/config"
)

func TestSetupWithoutExporterIsNoop(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")
	t.Setenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", "")

	shutdown, err := Setup(context.Background(), config.Config{})
	if err != nil {
		t.Fatalf("Setup() error = %v", err)
	}
	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown() error = %v", err)
	}
}

func TestSampleRatioRejectsInvalidValues(t *testing.T) {
	t.Setenv("KUBEVISTA_TRACE_SAMPLE_RATIO", "2")
	if got := sampleRatio(); got != 0.25 {
		t.Fatalf("sampleRatio() = %v, want 0.25", got)
	}
}
