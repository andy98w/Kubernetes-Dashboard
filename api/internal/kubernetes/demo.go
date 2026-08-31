package kubernetes

import (
	"context"
	"time"
)

type DemoInventory struct {
	ClusterName string
}

func (DemoInventory) Ready(context.Context) error { return nil }

func (d DemoInventory) Summary(context.Context) (Summary, error) {
	return Summary{
		Cluster:    d.ClusterName,
		Mode:       "demo",
		Nodes:      NodeCounts{Ready: 3, Total: 3},
		Namespaces: 12,
		Pods:       PodCounts{Running: 42, Pending: 1},
		ObservedAt: time.Now().UTC(),
	}, nil
}
