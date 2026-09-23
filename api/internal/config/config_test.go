package config

import "testing"

func TestOperationDefaults(t *testing.T) {
	t.Setenv("KUBEVISTA_OPERATIONS_ENABLED", "")
	t.Setenv("KUBEVISTA_OPERATION_NAMESPACES", "")
	t.Setenv("KUBEVISTA_MIN_REPLICAS", "")
	t.Setenv("KUBEVISTA_MAX_REPLICAS", "")
	cfg, err := FromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.OperationsEnabled || len(cfg.OperationNamespaces) != 1 || cfg.OperationNamespaces[0] != "kubevista" || cfg.MinReplicas != 1 || cfg.MaxReplicas != 6 {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
}

func TestRejectsInvalidReplicaLimits(t *testing.T) {
	t.Setenv("KUBEVISTA_MIN_REPLICAS", "5")
	t.Setenv("KUBEVISTA_MAX_REPLICAS", "2")
	if _, err := FromEnv(); err == nil {
		t.Fatal("expected invalid replica limits to fail")
	}
}

func TestDurableOperationConfiguration(t *testing.T) {
	t.Setenv("KUBEVISTA_DURABLE_OPERATIONS", "true")
	t.Setenv("KUBEVISTA_OPERATIONS_ENABLED", "false")
	t.Setenv("KUBEVISTA_DEMO_MODE", "false")
	t.Setenv("KUBEVISTA_ENVIRONMENT", "development")
	if _, err := FromEnv(); err == nil {
		t.Fatal("durable store without operations accepted")
	}
	t.Setenv("KUBEVISTA_OPERATIONS_ENABLED", "true")
	t.Setenv("KUBEVISTA_DEMO_MODE", "true")
	if _, err := FromEnv(); err == nil {
		t.Fatal("durable demo mode accepted")
	}
	t.Setenv("KUBEVISTA_DEMO_MODE", "false")
	cfg, err := FromEnv()
	if err != nil || !cfg.DurableOperations || cfg.ControllerActiveActive || cfg.OperationStoreNamespace != "kubevista-operations" {
		t.Fatalf("unexpected durable defaults: %+v, %v", cfg, err)
	}
	t.Setenv("KUBEVISTA_ENVIRONMENT", "production")
	t.Setenv("KUBEVISTA_ALB_SIGNER_ARN", "")
	if _, err := FromEnv(); err == nil {
		t.Fatal("durable mode bypassed production authentication")
	}
}
