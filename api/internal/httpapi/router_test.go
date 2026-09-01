package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/andy98w/Kubernetes-Dashboard/api/internal/config"
	"github.com/andy98w/Kubernetes-Dashboard/api/internal/kubernetes"
)

func TestHealth(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	New(config.Config{Version: "test", Environment: "test"}, kubernetes.DemoInventory{ClusterName: "test"}).ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if got := w.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("expected security header, got %q", got)
	}
}

func TestSummary(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/v1/summary", nil)
	w := httptest.NewRecorder()
	New(config.Config{Version: "test", Environment: "test"}, kubernetes.DemoInventory{ClusterName: "recruiter-demo"}).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var summary kubernetes.Summary
	if err := json.NewDecoder(w.Body).Decode(&summary); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if summary.Cluster != "recruiter-demo" || summary.Mode != "demo" || summary.Nodes.Ready != 3 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
}

func TestMetrics(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()
	New(config.Config{}, kubernetes.DemoInventory{}).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if got := w.Header().Get("Content-Type"); got == "" {
		t.Fatal("expected metrics content type")
	}
}

func TestDashboardInventoryRoutes(t *testing.T) {
	handler := New(config.Config{Version: "test", Environment: "test", ClusterName: "test-cluster"}, kubernetes.DemoInventory{ClusterName: "test-cluster"})
	for _, path := range []string{"/api/v1/workloads", "/api/v1/network", "/api/v1/events", "/api/v1/observability", "/api/v1/security", "/api/v1/cost", "/api/v1/settings"} {
		t.Run(path, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, path, nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)
			if w.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
			}
			if got := w.Header().Get("Content-Type"); got != "application/json" {
				t.Fatalf("expected JSON response, got %q", got)
			}
		})
	}
}
