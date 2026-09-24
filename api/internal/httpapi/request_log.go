package httpapi

import (
	"log/slog"
	"net/http"

	"github.com/andy98w/Kubernetes-Dashboard/api/internal/config"
	"github.com/felixge/httpsnoop"
	"go.opentelemetry.io/otel/trace"
)

// requestLog records bounded request metadata, never URLs, bodies, or headers.
// It runs inside the otelhttp handler so the request span is available.
func requestLog(next http.Handler, cfg config.Config, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		metrics := httpsnoop.CaptureMetrics(next, w, r)
		route := r.Pattern
		if route == "" {
			route = "unmatched"
		}
		attrs := []any{"service.name", "kubevista-api", "service.version", cfg.Version,
			"deployment.environment.name", cfg.Environment, "http.route", route,
			"http.response.status_code", metrics.Code, "duration_ms", float64(metrics.Duration.Microseconds()) / 1000}
		span := trace.SpanContextFromContext(r.Context())
		if span.IsValid() {
			attrs = append(attrs, "trace_id", span.TraceID().String(), "span_id", span.SpanID().String())
		}
		if metrics.Code >= 500 {
			logger.ErrorContext(r.Context(), "HTTP request completed", attrs...)
		} else if metrics.Code >= 400 {
			logger.WarnContext(r.Context(), "HTTP request completed", attrs...)
		} else if r.URL.Path != "/healthz" && r.URL.Path != "/readyz" && r.URL.Path != "/metrics" {
			logger.InfoContext(r.Context(), "HTTP request completed", attrs...)
		}
	})
}
