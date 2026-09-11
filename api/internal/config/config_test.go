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
