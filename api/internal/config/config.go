package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	Address     string
	Environment string
	Version     string
	DemoMode    bool
}

func FromEnv() (Config, error) {
	cfg := Config{
		Address:     envOr("KUBEVISTA_ADDRESS", ":8080"),
		Environment: envOr("KUBEVISTA_ENVIRONMENT", "development"),
		Version:     envOr("KUBEVISTA_VERSION", "dev"),
		DemoMode:    strings.EqualFold(os.Getenv("KUBEVISTA_DEMO_MODE"), "true"),
	}
	if !strings.HasPrefix(cfg.Address, ":") && !strings.Contains(cfg.Address, ":") {
		return Config{}, fmt.Errorf("KUBEVISTA_ADDRESS must be host:port or :port")
	}
	return cfg, nil
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
