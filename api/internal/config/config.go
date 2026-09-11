package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Address             string
	Environment         string
	Version             string
	ClusterName         string
	Kubeconfig          string
	DemoMode            bool
	OperationsEnabled   bool
	OperationNamespaces []string
	MinReplicas         int32
	MaxReplicas         int32
	ALBSignerARN        string
	AWSRegion           string
}

func FromEnv() (Config, error) {
	cfg := Config{
		Address:             envOr("KUBEVISTA_ADDRESS", ":8080"),
		Environment:         envOr("KUBEVISTA_ENVIRONMENT", "development"),
		Version:             envOr("KUBEVISTA_VERSION", "dev"),
		ClusterName:         envOr("KUBEVISTA_CLUSTER_NAME", "kubevista-dev"),
		Kubeconfig:          os.Getenv("KUBECONFIG"),
		DemoMode:            strings.EqualFold(os.Getenv("KUBEVISTA_DEMO_MODE"), "true"),
		OperationsEnabled:   strings.EqualFold(os.Getenv("KUBEVISTA_OPERATIONS_ENABLED"), "true"),
		OperationNamespaces: splitList(envOr("KUBEVISTA_OPERATION_NAMESPACES", "kubevista")),
		ALBSignerARN:        os.Getenv("KUBEVISTA_ALB_SIGNER_ARN"),
		AWSRegion:           envOr("AWS_REGION", "us-west-2"),
	}
	minReplicas, err := envInt32("KUBEVISTA_MIN_REPLICAS", 1)
	if err != nil {
		return Config{}, err
	}
	maxReplicas, err := envInt32("KUBEVISTA_MAX_REPLICAS", 6)
	if err != nil {
		return Config{}, err
	}
	cfg.MinReplicas, cfg.MaxReplicas = minReplicas, maxReplicas
	if !strings.HasPrefix(cfg.Address, ":") && !strings.Contains(cfg.Address, ":") {
		return Config{}, fmt.Errorf("KUBEVISTA_ADDRESS must be host:port or :port")
	}
	if cfg.MinReplicas < 0 || cfg.MaxReplicas < cfg.MinReplicas {
		return Config{}, fmt.Errorf("replica limits must satisfy 0 <= min <= max")
	}
	if cfg.OperationsEnabled && strings.EqualFold(cfg.Environment, "production") && cfg.ALBSignerARN == "" {
		return Config{}, fmt.Errorf("KUBEVISTA_ALB_SIGNER_ARN is required for production operations")
	}
	return cfg, nil
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func splitList(value string) []string {
	items := []string{}
	for _, item := range strings.Split(value, ",") {
		if trimmed := strings.TrimSpace(item); trimmed != "" {
			items = append(items, trimmed)
		}
	}
	return items
}

func envInt32(key string, fallback int32) (int32, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer", key)
	}
	return int32(parsed), nil
}
