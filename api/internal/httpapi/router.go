package httpapi

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
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
	mux.HandleFunc("GET /api/v1/network", func(w http.ResponseWriter, r *http.Request) { serveInventory(w, r, "network", inventory.Network) })
	mux.HandleFunc("GET /api/v1/events", func(w http.ResponseWriter, r *http.Request) { serveInventory(w, r, "events", inventory.Events) })
	mux.HandleFunc("GET /api/v1/observability", func(w http.ResponseWriter, r *http.Request) {
		serveInventory(w, r, "observability", inventory.Observability)
	})
	mux.HandleFunc("GET /api/v1/security", func(w http.ResponseWriter, r *http.Request) { serveInventory(w, r, "security", inventory.Security) })
	mux.HandleFunc("GET /api/v1/cost", func(w http.ResponseWriter, r *http.Request) { serveInventory(w, r, "cost", inventory.Cost) })
	mux.HandleFunc("GET /api/v1/settings", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"cluster": cfg.ClusterName, "environment": cfg.Environment, "version": cfg.Version, "readOnly": true, "refreshSeconds": 15})
	})
	mux.Handle("GET /metrics", promhttp.Handler())
	return securityHeaders(mux)
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
