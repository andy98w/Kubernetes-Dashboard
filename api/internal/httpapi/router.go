package httpapi

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/andy98w/Kubernetes-Dashboard/api/internal/config"
)

type status struct {
	Status      string `json:"status"`
	Version     string `json:"version"`
	Environment string `json:"environment"`
	Timestamp   string `json:"timestamp"`
}

func New(cfg config.Config) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, status{"ok", cfg.Version, cfg.Environment, time.Now().UTC().Format(time.RFC3339)})
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, status{"ready", cfg.Version, cfg.Environment, time.Now().UTC().Format(time.RFC3339)})
	})
	mux.HandleFunc("GET /api/v1/summary", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"cluster": "portfolio-eks", "mode": map[bool]string{true: "demo", false: "cluster"}[cfg.DemoMode],
			"nodes": 3, "namespaces": 12, "pods": map[string]int{"running": 42, "pending": 1, "failed": 0},
		})
	})
	return securityHeaders(mux)
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
