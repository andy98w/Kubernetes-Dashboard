package httpapi

import (
	"bytes"
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
	New(config.Config{Version: "test", Environment: "test"}, kubernetes.DemoInventory{ClusterName: "kubevista-demo"}).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var summary kubernetes.Summary
	if err := json.NewDecoder(w.Body).Decode(&summary); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if summary.Cluster != "kubevista-demo" || summary.Mode != "demo" || summary.Nodes.Ready != 3 {
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
	for _, path := range []string{"/api/v1/workloads", "/api/v1/workloads/kubevista/Deployment/kubevista-api", "/api/v1/network", "/api/v1/events", "/api/v1/observability", "/api/v1/security", "/api/v1/cost", "/api/v1/incidents", "/api/v1/settings"} {
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

func TestDemoOperationRequiresReviewHeader(t *testing.T) {
	handler := New(config.Config{DemoMode: true}, kubernetes.DemoInventory{ClusterName: "test"})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/operations/plan", bytes.NewBufferString(`{"action":"restart","namespace":"kubevista","kind":"Deployment","name":"kubevista-api","reason":"Validate recovery"}`))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDemoOperationReviewAndExecute(t *testing.T) {
	handler := New(config.Config{DemoMode: true}, kubernetes.DemoInventory{ClusterName: "test"})
	body := `{"action":"scale","namespace":"kubevista","kind":"Deployment","name":"kubevista-api","replicas":3,"reason":"Validate guarded scaling"}`
	r := httptest.NewRequest(http.MethodPost, "/api/v1/operations/plan", bytes.NewBufferString(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-KubeVista-Operation", "reviewed")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var plan kubernetes.OperationPlan
	if err := json.NewDecoder(w.Body).Decode(&plan); err != nil {
		t.Fatal(err)
	}
	if !plan.Simulated || !plan.DryRun || plan.DesiredReplicas != 3 || plan.ID == "" {
		t.Fatalf("unexpected plan: %+v", plan)
	}

	execute := map[string]any{"action": "scale", "namespace": "kubevista", "kind": "Deployment", "name": "kubevista-api", "replicas": 3, "reason": "Validate guarded scaling", "planId": plan.ID, "expectedResourceVersion": plan.ResourceVersion}
	payload, _ := json.Marshal(execute)
	r = httptest.NewRequest(http.MethodPost, "/api/v1/operations/execute", bytes.NewReader(payload))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-KubeVista-Operation", "reviewed")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", w.Code, w.Body.String())
	}
	var record kubernetes.OperationRecord
	if err := json.NewDecoder(w.Body).Decode(&record); err != nil {
		t.Fatal(err)
	}
	if !record.Simulated || record.Status != "Simulated" || record.After != "3 replicas" {
		t.Fatalf("unexpected record: %+v", record)
	}
}
