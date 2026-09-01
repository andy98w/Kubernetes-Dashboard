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

func (DemoInventory) Workloads(context.Context) (Workloads, error) {
	now := time.Now().UTC()
	return Workloads{Items: []Workload{
		{"Deployment", "kubevista", "kubevista-api", 2, 2, "Healthy", now.Add(-4 * time.Hour)},
		{"Deployment", "kubevista", "kubevista-web", 2, 2, "Healthy", now.Add(-4 * time.Hour)},
		{"StatefulSet", "observability", "loki", 1, 1, "Healthy", now.Add(-3 * time.Hour)},
	}, ObservedAt: now}, nil
}

func (DemoInventory) Network(context.Context) (Network, error) {
	return Network{
		Services:  []Service{{"kubevista", "kubevista-api", "ClusterIP", "172.20.4.18", []string{"80/TCP"}}, {"kubevista", "kubevista-web", "ClusterIP", "172.20.8.42", []string{"80/TCP"}}},
		Ingresses: []Ingress{{"kubevista", "kubevista", "alb", []string{"kubevista.example.com"}, "demo-alb.us-west-2.elb.amazonaws.com"}},
		Policies:  []Policy{{"kubevista", "kubevista-api", 3, 3}, {"kubevista", "kubevista-web", 2, 1}}, ObservedAt: time.Now().UTC(),
	}, nil
}

func (DemoInventory) Events(context.Context) (Events, error) {
	now := time.Now().UTC()
	return Events{Items: []Event{{"Normal", "ScalingReplicaSet", "kubevista", "Deployment/kubevista-api", "Scaled up replica set to 2.", 1, now.Add(-2 * time.Minute)}, {"Warning", "Unhealthy", "demo", "Pod/example", "Readiness probe failed during rollout.", 2, now.Add(-18 * time.Minute)}}, ObservedAt: now}, nil
}

func (DemoInventory) Observability(context.Context) (Observability, error) {
	return Observability{Components: []Component{{"StatefulSet", "prometheus", 1, 1, "Healthy"}, {"StatefulSet", "loki", 1, 1, "Healthy"}, {"StatefulSet", "tempo", 1, 1, "Healthy"}, {"Deployment", "otel-gateway", 2, 2, "Healthy"}}, Signals: []string{"Metrics / Prometheus", "Logs / Loki", "Traces / Tempo", "Telemetry / OpenTelemetry"}, Namespace: "observability", ObservedAt: time.Now().UTC()}, nil
}

func (DemoInventory) Security(context.Context) (Security, error) {
	return Security{Findings: []Finding{{"Warning", "Privilege escalation", "demo", "Pod/legacy-worker:worker", "allowPrivilegeEscalation is not explicitly disabled."}}, PodsEvaluated: 42, NetworkPolicies: 8, ObservedAt: time.Now().UTC()}, nil
}

func (DemoInventory) Cost(context.Context) (Cost, error) {
	return Cost{Nodes: []NodeCost{{"demo-node-a", "t3.medium", "SPOT", 0.0146}, {"demo-node-b", "t3.medium", "SPOT", 0.0146}}, ControlPlaneHourly: 0.10, LoadBalancerHourly: 0.0225, NATGatewayHourly: 0.045, EstimatedHourly: 0.1967, Currency: "USD", Disclaimer: "Directional estimate for compute and fixed hourly infrastructure only; excludes storage, data processing, logs, taxes, discounts, and free-tier credits.", ObservedAt: time.Now().UTC()}, nil
}
