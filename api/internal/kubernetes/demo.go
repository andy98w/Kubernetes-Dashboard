package kubernetes

import (
	"context"
	"fmt"
	"strings"
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

func (d DemoInventory) WorkloadDetail(ctx context.Context, namespace, kind, name string) (WorkloadDetail, error) {
	workloads, _ := d.Workloads(ctx)
	for _, w := range workloads.Items {
		if w.Namespace == namespace && strings.EqualFold(w.Kind, kind) && w.Name == name {
			now := time.Now().UTC()
			return WorkloadDetail{Workload: w, Strategy: "RollingUpdate", Selector: map[string]string{"app.kubernetes.io/name": name}, Labels: map[string]string{"app.kubernetes.io/name": name, "environment": "production"}, Images: []ContainerImage{{"app", "public.ecr.aws/kubevista/" + name + ":sha-8f04c2a"}}, Pods: []PodDetail{{name + "-7d8c9b-x2m4p", "Running", 1, 1, 0, "ip-10-0-11-42", now.Add(-3 * time.Hour)}, {name + "-7d8c9b-z9q7n", "Running", 1, 1, 0, "ip-10-0-21-18", now.Add(-3 * time.Hour)}}, Services: []Service{{namespace, name, "ClusterIP", "172.20.4.18", []string{"80/TCP"}}}, Policies: []Policy{{namespace, name, 2, 1}}, Events: []Event{{"Normal", "ScalingReplicaSet", namespace, "Deployment/" + name, "Scaled up replica set to 2.", 1, now.Add(-22 * time.Minute)}}, ObservedAt: now}, nil
		}
	}
	return WorkloadDetail{}, fmt.Errorf("workload not found")
}

func (d DemoInventory) Incidents(context.Context) (Incidents, error) {
	now := time.Now().UTC()
	return Incidents{[]Incident{{"test-2026-08-31-api-loss", "Controlled", "Resolved", "Controlled API pod-loss recovery", "A running API pod was removed while Fortio generated traffic to validate disruption tolerance.", "kubevista", "Deployment/kubevista-api", now.Add(-47 * time.Minute), []IncidentEvidence{{"Load generator", "Fortio maintained 20 requests per second during the test", now.Add(-47 * time.Minute)}, {"Kubernetes controller", "Replacement API pod became Ready in 2 seconds", now.Add(-46 * time.Minute)}, {"Verification", "896/896 requests returned HTTP 200; p99 was approximately 2.78 ms", now.Add(-43 * time.Minute)}}}}, now}, nil
}

func (d DemoInventory) PlanOperation(_ context.Context, request OperationRequest) (OperationPlan, error) {
	if !strings.EqualFold(request.Kind, "Deployment") {
		return OperationPlan{}, &OperationError{"kind_denied", "only Deployments support guarded operations"}
	}
	if len(strings.TrimSpace(request.Reason)) < 8 {
		return OperationPlan{}, &OperationError{"invalid_reason", "reason must be at least 8 characters"}
	}
	current, desired := int32(2), int32(2)
	if request.Action == OperationScale {
		if request.Replicas == nil || *request.Replicas < 1 || *request.Replicas > 6 {
			return OperationPlan{}, &OperationError{"invalid_replicas", "replicas must be between 1 and 6"}
		}
		desired = *request.Replicas
	} else if request.Action != OperationRestart {
		return OperationPlan{}, &OperationError{"action_denied", "operation is not supported"}
	}
	impact := "Simulates a rolling restart without changing a cluster."
	if request.Action == OperationScale {
		impact = fmt.Sprintf("Simulates changing desired replicas from %d to %d within the 1–6 guardrail. A HorizontalPodAutoscaler, if present, may subsequently reconcile this value.", current, desired)
	}
	return OperationPlan{newOperationID(), request.Action, target(request), request.Reason, current, desired, impact, "demo-resource-version", true, true, true, time.Now().UTC()}, nil
}

func (d DemoInventory) ExecuteOperation(_ context.Context, request OperationRequest, actor string) (OperationRecord, error) {
	plan, err := d.PlanOperation(context.Background(), request)
	if err != nil {
		return OperationRecord{}, err
	}
	if request.PlanID == "" {
		return OperationRecord{}, &OperationError{"plan_required", "review the operation before executing it"}
	}
	after := fmt.Sprintf("%d replicas", plan.CurrentReplicas)
	if request.Action == OperationScale {
		after = fmt.Sprintf("%d replicas", plan.DesiredReplicas)
	}
	return OperationRecord{request.PlanID, request.Action, plan.Target, actor, request.Reason, "Simulated", fmt.Sprintf("%d replicas", plan.CurrentReplicas), after, true, time.Now().UTC()}, nil
}

func (DemoInventory) RecentOperations(context.Context) ([]OperationRecord, error) {
	return []OperationRecord{}, nil
}

func (DemoInventory) Updates(ctx context.Context) (<-chan ClusterUpdate, error) {
	ch := make(chan ClusterUpdate, 1)
	go func() {
		defer close(ch)
		ticker := time.NewTicker(8 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case t := <-ticker.C:
				ch <- ClusterUpdate{"pods", "modified", "kubevista", "kubevista-api", t.UTC()}
			}
		}
	}()
	return ch, nil
}
