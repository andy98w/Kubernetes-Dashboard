package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/andy98w/Kubernetes-Dashboard/api/internal/config"
	"github.com/andy98w/Kubernetes-Dashboard/api/internal/httpapi"
	cluster "github.com/andy98w/Kubernetes-Dashboard/api/internal/kubernetes"
	"github.com/andy98w/Kubernetes-Dashboard/api/internal/telemetry"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func main() {
	cfg, err := config.FromEnv()
	if err != nil {
		slog.Error("invalid configuration", "error", err)
		os.Exit(1)
	}
	inventory, err := cluster.New(cfg)
	if err != nil {
		slog.Error("initialize cluster inventory", "error", err)
		os.Exit(1)
	}
	shutdownTelemetry, err := telemetry.Setup(context.Background(), cfg)
	if err != nil {
		slog.Error("initialize telemetry", "error", err)
		os.Exit(1)
	}

	server := &http.Server{
		Addr:              cfg.Address,
		Handler:           otelhttp.NewHandler(httpapi.New(cfg, inventory), "kubevista.http"),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go func() {
		slog.Info("api listening", "address", cfg.Address, "environment", cfg.Environment)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("graceful shutdown failed", "error", err)
		os.Exit(1)
	}
	if err := shutdownTelemetry(shutdownCtx); err != nil {
		slog.Error("telemetry shutdown failed", "error", err)
		os.Exit(1)
	}
}
