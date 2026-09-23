package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/andy98w/Kubernetes-Dashboard/api/internal/config"
	cluster "github.com/andy98w/Kubernetes-Dashboard/api/internal/kubernetes"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type status struct {
	Status      string `json:"status"`
	Version     string `json:"version"`
	Environment string `json:"environment"`
	Timestamp   string `json:"timestamp"`
}

func New(cfg config.Config, inventory cluster.Inventory) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/delivery", func(w http.ResponseWriter, r *http.Request) {
		if source, ok := inventory.(interface {
			Delivery(context.Context) (cluster.Delivery, error)
		}); ok {
			serveInventory(w, r, "delivery", source.Delivery)
			return
		}
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "delivery observations unavailable"})
	})
	mux.HandleFunc("GET /api/v1/topology", func(w http.ResponseWriter, r *http.Request) {
		if source, ok := inventory.(interface {
			Topology(context.Context) (cluster.Topology, error)
		}); ok {
			serveInventory(w, r, "topology", source.Topology)
			return
		}
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "topology unavailable"})
	})
	mux.HandleFunc("GET /api/v1/station-health", func(w http.ResponseWriter, r *http.Request) {
		if source, ok := inventory.(interface {
			StationHealth(context.Context) (cluster.StationHealth, error)
		}); ok {
			serveInventory(w, r, "station-health", source.StationHealth)
			return
		}
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "station health unavailable"})
	})
	claimsVerifier := newALBClaimsVerifier(cfg.ALBSignerARN, cfg.AWSRegion)
	mux.HandleFunc("POST /api/v1/investigate/{namespace}/{kind}/{name}", investigationHandler(cfg, inventory))
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, status{"ok", cfg.Version, cfg.Environment, time.Now().UTC().Format(time.RFC3339)})
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		if err := inventory.Ready(ctx); err != nil {
			slog.Warn("readiness check failed", "error", err)
			writeJSON(w, http.StatusServiceUnavailable, status{"not-ready", cfg.Version, cfg.Environment, time.Now().UTC().Format(time.RFC3339)})
			return
		}
		writeJSON(w, http.StatusOK, status{"ready", cfg.Version, cfg.Environment, time.Now().UTC().Format(time.RFC3339)})
	})
	mux.HandleFunc("GET /api/v1/summary", func(w http.ResponseWriter, r *http.Request) {
		serveInventory(w, r, "summary", inventory.Summary)
	})
	mux.HandleFunc("GET /api/v1/workloads", func(w http.ResponseWriter, r *http.Request) { serveInventory(w, r, "workloads", inventory.Workloads) })
	mux.HandleFunc("GET /api/v1/workloads/{namespace}/{kind}/{name}", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
		defer cancel()
		value, err := inventory.WorkloadDetail(ctx, r.PathValue("namespace"), r.PathValue("kind"), r.PathValue("name"))
		if err != nil {
			slog.Warn("workload detail unavailable", "error", err)
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "workload not found"})
			return
		}
		if r.URL.Query().Get("evidence") == "true" {
			if richer, ok := inventory.(interface {
				Enrich(context.Context, *cluster.WorkloadDetail)
			}); ok {
				richer.Enrich(ctx, &value)
			}
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("GET /api/v1/network", func(w http.ResponseWriter, r *http.Request) { serveInventory(w, r, "network", inventory.Network) })
	mux.HandleFunc("GET /api/v1/events", func(w http.ResponseWriter, r *http.Request) { serveInventory(w, r, "events", inventory.Events) })
	mux.HandleFunc("GET /api/v1/observability", func(w http.ResponseWriter, r *http.Request) {
		serveInventory(w, r, "observability", inventory.Observability)
	})
	mux.HandleFunc("GET /api/v1/security", func(w http.ResponseWriter, r *http.Request) { serveInventory(w, r, "security", inventory.Security) })
	mux.HandleFunc("GET /api/v1/cost", func(w http.ResponseWriter, r *http.Request) { serveInventory(w, r, "cost", inventory.Cost) })
	mux.HandleFunc("GET /api/v1/incidents", func(w http.ResponseWriter, r *http.Request) { serveInventory(w, r, "incidents", inventory.Incidents) })
	mux.HandleFunc("GET /api/v1/operations", func(w http.ResponseWriter, r *http.Request) {
		serveInventory(w, r, "operations", inventory.RecentOperations)
	})
	mux.HandleFunc("POST /api/v1/operations/plan", func(w http.ResponseWriter, r *http.Request) {
		request, ok := decodeOperation(w, r)
		if !ok {
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
		defer cancel()
		plan, err := inventory.PlanOperation(ctx, request)
		if err != nil {
			writeOperationError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, plan)
	})
	mux.HandleFunc("POST /api/v1/operations/execute", func(w http.ResponseWriter, r *http.Request) {
		request, ok := decodeOperation(w, r)
		if !ok {
			return
		}
		actor, ok := operationActor(cfg, claimsVerifier, r)
		if !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authenticated operator identity is required", "code": "actor_required"})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
		defer cancel()
		record, err := inventory.ExecuteOperation(ctx, request, actor)
		if err != nil {
			writeOperationError(w, err)
			return
		}
		writeJSON(w, http.StatusAccepted, record)
	})
	mux.HandleFunc("GET /api/v1/stream", func(w http.ResponseWriter, r *http.Request) { streamUpdates(w, r, inventory) })
	mux.HandleFunc("GET /api/v1/settings", func(w http.ResponseWriter, _ *http.Request) {
		mode := "disabled"
		if cfg.OperationsEnabled {
			mode = "guarded"
		}
		if cfg.DemoMode {
			mode = "simulation"
		}
		writeJSON(w, http.StatusOK, map[string]any{"cluster": cfg.ClusterName, "environment": cfg.Environment, "version": cfg.Version, "readOnly": !cfg.OperationsEnabled, "operationsMode": mode, "operationNamespaces": cfg.OperationNamespaces, "minReplicas": cfg.MinReplicas, "maxReplicas": cfg.MaxReplicas, "refreshSeconds": 15})
	})
	mux.Handle("GET /metrics", promhttp.Handler())
	return securityHeaders(mux)
}

func decodeOperation(w http.ResponseWriter, r *http.Request) (cluster.OperationRequest, bool) {
	if r.Header.Get("X-KubeVista-Operation") != "reviewed" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "operation review header is required", "code": "review_required"})
		return cluster.OperationRequest{}, false
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var request cluster.OperationRequest
	if err := decoder.Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid operation request", "code": "invalid_request"})
		return cluster.OperationRequest{}, false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "request must contain one JSON object", "code": "invalid_request"})
		return cluster.OperationRequest{}, false
	}
	return request, true
}

func operationActor(cfg config.Config, verifier *albClaimsVerifier, r *http.Request) (string, bool) {
	if cfg.DemoMode {
		return "demo-operator", true
	}
	if !strings.EqualFold(cfg.Environment, "production") {
		return "local-operator", true
	}
	actor, err := verifier.Verify(r.Context(), r.Header.Get("X-Amzn-Oidc-Data"))
	if err != nil {
		slog.Warn("rejecting operation with unverified ALB identity", "error", err)
		return "", false
	}
	return actor, true
}

func writeOperationError(w http.ResponseWriter, err error) {
	var operationError *cluster.OperationError
	if errors.As(err, &operationError) {
		status := http.StatusBadRequest
		if operationError.Code == "operations_disabled" || operationError.Code == "namespace_denied" || operationError.Code == "kind_denied" || operationError.Code == "action_denied" {
			status = http.StatusForbidden
		}
		if operationError.Code == "target_unavailable" {
			status = http.StatusNotFound
		}
		if operationError.Code == "stale_plan" {
			status = http.StatusConflict
		}
		writeJSON(w, status, map[string]string{"error": operationError.Message, "code": operationError.Code})
		return
	}
	writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "operation failed", "code": "internal_error"})
}

func streamUpdates(w http.ResponseWriter, r *http.Request, inventory cluster.Inventory) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "streaming unsupported"})
		return
	}
	updates, err := inventory.Updates(r.Context())
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "cluster stream unavailable"})
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	fmt.Fprint(w, "event: ready\ndata: {}\n\n")
	flusher.Flush()
	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-heartbeat.C:
			fmt.Fprint(w, ": heartbeat\n\n")
			flusher.Flush()
		case update, ok := <-updates:
			if !ok {
				return
			}
			payload, _ := json.Marshal(update)
			fmt.Fprintf(w, "event: cluster-update\ndata: %s\n\n", payload)
			flusher.Flush()
		}
	}
}

func serveInventory[T any](w http.ResponseWriter, r *http.Request, resource string, query func(context.Context) (T, error)) {
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	value, err := query(ctx)
	if err != nil {
		slog.Error("cluster inventory query failed", "resource", resource, "error", err)
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": resource + " inventory is unavailable"})
		return
	}
	writeJSON(w, http.StatusOK, value)
}

func writeJSON(w http.ResponseWriter, code int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(value)
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}
