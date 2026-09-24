package httpapi

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/andy98w/Kubernetes-Dashboard/api/internal/config"
	"go.opentelemetry.io/otel/trace"
)

func TestRequestLogCorrelatesWithoutPayloads(t *testing.T) {
	var out bytes.Buffer
	mux := http.NewServeMux()
	mux.HandleFunc("GET /items/{id}", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(503) })
	handler := requestLog(mux, config.Config{Version: "release-123", Environment: "test"}, slog.New(slog.NewJSONHandler(&out, nil)))
	req := httptest.NewRequest("GET", "/items/private-person?token=supersecret", strings.NewReader("private-body"))
	req.Header.Set("Authorization", "Bearer confidential")
	span := trace.NewSpanContext(trace.SpanContextConfig{TraceID: trace.TraceID{1}, SpanID: trace.SpanID{2}})
	req = req.WithContext(trace.ContextWithSpanContext(req.Context(), span))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	for _, secret := range []string{"private-person", "supersecret", "private-body", "confidential"} {
		if strings.Contains(out.String(), secret) {
			t.Fatalf("logged private data: %s", secret)
		}
	}
	var entry map[string]any
	if err := json.Unmarshal(out.Bytes(), &entry); err != nil {
		t.Fatal(err)
	}
	if entry["http.route"] != "GET /items/{id}" || entry["trace_id"] != span.TraceID().String() || entry["level"] != "ERROR" || entry["service.version"] != "release-123" {
		t.Fatalf("unexpected entry: %#v", entry)
	}
	if response.Code != 503 {
		t.Fatal("changed response")
	}
}

func TestRequestLogSuppressesSuccessfulProbe(t *testing.T) {
	var out bytes.Buffer
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	requestLog(next, config.Config{}, slog.New(slog.NewJSONHandler(&out, nil))).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/healthz", nil))
	if out.Len() != 0 {
		t.Fatal("successful probe logged")
	}
}
