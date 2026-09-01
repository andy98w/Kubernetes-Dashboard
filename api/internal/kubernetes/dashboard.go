package kubernetes

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Workload struct {
	Kind      string    `json:"kind"`
	Namespace string    `json:"namespace"`
	Name      string    `json:"name"`
	Ready     int32     `json:"ready"`
	Desired   int32     `json:"desired"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
}

type Workloads struct {
	Items      []Workload `json:"items"`
	ObservedAt time.Time  `json:"observedAt"`
}

type Service struct {
	Namespace string   `json:"namespace"`
	Name      string   `json:"name"`
	Type      string   `json:"type"`
	ClusterIP string   `json:"clusterIp"`
	Ports     []string `json:"ports"`
}

type Ingress struct {
	Namespace string   `json:"namespace"`
	Name      string   `json:"name"`
	Class     string   `json:"class"`
	Hosts     []string `json:"hosts"`
	Address   string   `json:"address"`
}

type Policy struct {
	Namespace    string `json:"namespace"`
	Name         string `json:"name"`
	IngressRules int    `json:"ingressRules"`
	EgressRules  int    `json:"egressRules"`
}

type Network struct {
	Services   []Service `json:"services"`
	Ingresses  []Ingress `json:"ingresses"`
	Policies   []Policy  `json:"policies"`
	ObservedAt time.Time `json:"observedAt"`
}

type Event struct {
	Type      string    `json:"type"`
	Reason    string    `json:"reason"`
	Namespace string    `json:"namespace"`
	Object    string    `json:"object"`
	Message   string    `json:"message"`
	Count     int32     `json:"count"`
	LastSeen  time.Time `json:"lastSeen"`
}

type Events struct {
	Items      []Event   `json:"items"`
	ObservedAt time.Time `json:"observedAt"`
}

type Component struct {
	Kind    string `json:"kind"`
	Name    string `json:"name"`
	Ready   int32  `json:"ready"`
	Desired int32  `json:"desired"`
	Status  string `json:"status"`
}

type Observability struct {
	Components []Component `json:"components"`
	Signals    []string    `json:"signals"`
	Namespace  string      `json:"namespace"`
	ObservedAt time.Time   `json:"observedAt"`
}

type Finding struct {
	Severity  string `json:"severity"`
	Category  string `json:"category"`
	Namespace string `json:"namespace"`
	Resource  string `json:"resource"`
	Message   string `json:"message"`
}

type Security struct {
	Findings        []Finding `json:"findings"`
	PodsEvaluated   int       `json:"podsEvaluated"`
	NetworkPolicies int       `json:"networkPolicies"`
	ObservedAt      time.Time `json:"observedAt"`
}

type NodeCost struct {
	Name            string  `json:"name"`
	InstanceType    string  `json:"instanceType"`
	CapacityType    string  `json:"capacityType"`
	EstimatedHourly float64 `json:"estimatedHourly"`
}

type Cost struct {
	Nodes              []NodeCost `json:"nodes"`
	ControlPlaneHourly float64    `json:"controlPlaneHourly"`
	LoadBalancerHourly float64    `json:"loadBalancerHourly"`
	NATGatewayHourly   float64    `json:"natGatewayHourly"`
	EstimatedHourly    float64    `json:"estimatedHourly"`
	Currency           string     `json:"currency"`
	Disclaimer         string     `json:"disclaimer"`
	ObservedAt         time.Time  `json:"observedAt"`
}

func (c *Client) Workloads(ctx context.Context) (Workloads, error) {
	deployments, err := c.client.AppsV1().Deployments("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return Workloads{}, fmt.Errorf("list deployments: %w", err)
	}
	statefulSets, err := c.client.AppsV1().StatefulSets("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return Workloads{}, fmt.Errorf("list statefulsets: %w", err)
	}
	daemonSets, err := c.client.AppsV1().DaemonSets("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return Workloads{}, fmt.Errorf("list daemonsets: %w", err)
	}
	jobs, err := c.client.BatchV1().Jobs("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return Workloads{}, fmt.Errorf("list jobs: %w", err)
	}
	cronJobs, err := c.client.BatchV1().CronJobs("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return Workloads{}, fmt.Errorf("list cronjobs: %w", err)
	}
	items := make([]Workload, 0, len(deployments.Items)+len(statefulSets.Items)+len(daemonSets.Items)+len(jobs.Items)+len(cronJobs.Items))
	for _, d := range deployments.Items {
		items = append(items, workloadFromDeployment(d))
	}
	for _, s := range statefulSets.Items {
		items = append(items, Workload{"StatefulSet", s.Namespace, s.Name, s.Status.ReadyReplicas, s.Status.Replicas, health(s.Status.ReadyReplicas, s.Status.Replicas), s.CreationTimestamp.Time})
	}
	for _, d := range daemonSets.Items {
		items = append(items, Workload{"DaemonSet", d.Namespace, d.Name, d.Status.NumberReady, d.Status.DesiredNumberScheduled, health(d.Status.NumberReady, d.Status.DesiredNumberScheduled), d.CreationTimestamp.Time})
	}
	for _, j := range jobs.Items {
		desired := int32(1)
		if j.Spec.Completions != nil {
			desired = *j.Spec.Completions
		}
		items = append(items, Workload{"Job", j.Namespace, j.Name, j.Status.Succeeded, desired, jobHealth(j, desired), j.CreationTimestamp.Time})
	}
	for _, j := range cronJobs.Items {
		status := "Scheduled"
		if j.Spec.Suspend != nil && *j.Spec.Suspend {
			status = "Suspended"
		}
		items = append(items, Workload{"CronJob", j.Namespace, j.Name, int32(len(j.Status.Active)), 1, status, j.CreationTimestamp.Time})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Namespace == items[j].Namespace {
			return items[i].Name < items[j].Name
		}
		return items[i].Namespace < items[j].Namespace
	})
	return Workloads{items, time.Now().UTC()}, nil
}

func jobHealth(job batchv1.Job, desired int32) string {
	if job.Status.Failed > 0 {
		return "Unavailable"
	}
	if job.Status.Succeeded >= desired {
		return "Healthy"
	}
	return "Progressing"
}

func workloadFromDeployment(d appsv1.Deployment) Workload {
	desired := int32(1)
	if d.Spec.Replicas != nil {
		desired = *d.Spec.Replicas
	}
	return Workload{"Deployment", d.Namespace, d.Name, d.Status.ReadyReplicas, desired, health(d.Status.ReadyReplicas, desired), d.CreationTimestamp.Time}
}

func health(ready, desired int32) string {
	if desired == 0 {
		return "Scaled down"
	}
	if ready == desired {
		return "Healthy"
	}
	if ready == 0 {
		return "Unavailable"
	}
	return "Progressing"
}

func (c *Client) Network(ctx context.Context) (Network, error) {
	services, err := c.client.CoreV1().Services("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return Network{}, fmt.Errorf("list services: %w", err)
	}
	ingresses, err := c.client.NetworkingV1().Ingresses("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return Network{}, fmt.Errorf("list ingresses: %w", err)
	}
	policies, err := c.client.NetworkingV1().NetworkPolicies("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return Network{}, fmt.Errorf("list network policies: %w", err)
	}
	result := Network{Services: []Service{}, Ingresses: []Ingress{}, Policies: []Policy{}, ObservedAt: time.Now().UTC()}
	for _, s := range services.Items {
		ports := make([]string, 0, len(s.Spec.Ports))
		for _, p := range s.Spec.Ports {
			ports = append(ports, fmt.Sprintf("%d/%s", p.Port, p.Protocol))
		}
		result.Services = append(result.Services, Service{s.Namespace, s.Name, string(s.Spec.Type), s.Spec.ClusterIP, ports})
	}
	for _, i := range ingresses.Items {
		hosts := make([]string, 0, len(i.Spec.Rules))
		for _, rule := range i.Spec.Rules {
			if rule.Host != "" {
				hosts = append(hosts, rule.Host)
			}
		}
		class := "default"
		if i.Spec.IngressClassName != nil {
			class = *i.Spec.IngressClassName
		}
		address := "Pending"
		if len(i.Status.LoadBalancer.Ingress) > 0 {
			address = i.Status.LoadBalancer.Ingress[0].Hostname
			if address == "" {
				address = i.Status.LoadBalancer.Ingress[0].IP
			}
		}
		result.Ingresses = append(result.Ingresses, Ingress{i.Namespace, i.Name, class, hosts, address})
	}
	for _, p := range policies.Items {
		result.Policies = append(result.Policies, Policy{p.Namespace, p.Name, len(p.Spec.Ingress), len(p.Spec.Egress)})
	}
	return result, nil
}

func (c *Client) Events(ctx context.Context) (Events, error) {
	list, err := c.client.CoreV1().Events("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return Events{}, fmt.Errorf("list events: %w", err)
	}
	items := make([]Event, 0, len(list.Items))
	for _, e := range list.Items {
		seen := e.LastTimestamp.Time
		if seen.IsZero() {
			seen = e.EventTime.Time
		}
		if seen.IsZero() {
			seen = e.CreationTimestamp.Time
		}
		items = append(items, Event{e.Type, e.Reason, e.Namespace, e.InvolvedObject.Kind + "/" + e.InvolvedObject.Name, e.Message, e.Count, seen})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].LastSeen.After(items[j].LastSeen) })
	if len(items) > 100 {
		items = items[:100]
	}
	return Events{items, time.Now().UTC()}, nil
}

func (c *Client) Observability(ctx context.Context) (Observability, error) {
	deployments, err := c.client.AppsV1().Deployments("observability").List(ctx, metav1.ListOptions{})
	if err != nil {
		return Observability{}, fmt.Errorf("list observability deployments: %w", err)
	}
	statefulSets, err := c.client.AppsV1().StatefulSets("observability").List(ctx, metav1.ListOptions{})
	if err != nil {
		return Observability{}, fmt.Errorf("list observability statefulsets: %w", err)
	}
	daemonSets, err := c.client.AppsV1().DaemonSets("observability").List(ctx, metav1.ListOptions{})
	if err != nil {
		return Observability{}, fmt.Errorf("list observability daemonsets: %w", err)
	}
	components := make([]Component, 0, len(deployments.Items)+len(statefulSets.Items)+len(daemonSets.Items))
	for _, d := range deployments.Items {
		w := workloadFromDeployment(d)
		components = append(components, Component{w.Kind, w.Name, w.Ready, w.Desired, w.Status})
	}
	for _, s := range statefulSets.Items {
		components = append(components, Component{"StatefulSet", s.Name, s.Status.ReadyReplicas, s.Status.Replicas, health(s.Status.ReadyReplicas, s.Status.Replicas)})
	}
	for _, d := range daemonSets.Items {
		components = append(components, Component{"DaemonSet", d.Name, d.Status.NumberReady, d.Status.DesiredNumberScheduled, health(d.Status.NumberReady, d.Status.DesiredNumberScheduled)})
	}
	sort.Slice(components, func(i, j int) bool { return components[i].Name < components[j].Name })
	return Observability{components, []string{"Metrics / Prometheus", "Logs / Loki", "Traces / Tempo", "Telemetry / OpenTelemetry"}, "observability", time.Now().UTC()}, nil
}

func (c *Client) Security(ctx context.Context) (Security, error) {
	pods, err := c.client.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return Security{}, fmt.Errorf("list pods for security scan: %w", err)
	}
	policies, err := c.client.NetworkingV1().NetworkPolicies("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return Security{}, fmt.Errorf("list network policies for security scan: %w", err)
	}
	findings := make([]Finding, 0)
	for _, pod := range pods.Items {
		for _, container := range pod.Spec.Containers {
			resource := "Pod/" + pod.Name + ":" + container.Name
			sc := container.SecurityContext
			if sc != nil && sc.Privileged != nil && *sc.Privileged {
				findings = append(findings, Finding{"Critical", "Privileged container", pod.Namespace, resource, "Container runs with privileged access."})
			}
			if sc == nil || sc.AllowPrivilegeEscalation == nil || *sc.AllowPrivilegeEscalation {
				findings = append(findings, Finding{"Warning", "Privilege escalation", pod.Namespace, resource, "allowPrivilegeEscalation is not explicitly disabled."})
			}
			if strings.HasSuffix(container.Image, ":latest") || !strings.Contains(container.Image, "@sha256:") && !strings.Contains(container.Image, ":") {
				findings = append(findings, Finding{"Info", "Mutable image", pod.Namespace, resource, "Image is not pinned to an immutable digest or version tag."})
			}
		}
	}
	severity := map[string]int{"Critical": 0, "Warning": 1, "Info": 2}
	sort.Slice(findings, func(i, j int) bool {
		if severity[findings[i].Severity] == severity[findings[j].Severity] {
			return findings[i].Resource < findings[j].Resource
		}
		return severity[findings[i].Severity] < severity[findings[j].Severity]
	})
	return Security{findings, len(pods.Items), len(policies.Items), time.Now().UTC()}, nil
}

func (c *Client) Cost(ctx context.Context) (Cost, error) {
	nodes, err := c.client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return Cost{}, fmt.Errorf("list nodes for cost estimate: %w", err)
	}
	result := Cost{Nodes: []NodeCost{}, ControlPlaneHourly: 0.10, LoadBalancerHourly: 0.0225, NATGatewayHourly: 0.045, Currency: "USD", Disclaimer: "Directional estimate for compute and fixed hourly infrastructure only; excludes storage, data processing, logs, taxes, discounts, and free-tier credits.", ObservedAt: time.Now().UTC()}
	result.EstimatedHourly = result.ControlPlaneHourly + result.LoadBalancerHourly + result.NATGatewayHourly
	for _, n := range nodes.Items {
		instanceType := n.Labels["node.kubernetes.io/instance-type"]
		capacity := n.Labels["eks.amazonaws.com/capacityType"]
		rate := hourlyRate(instanceType, capacity)
		result.Nodes = append(result.Nodes, NodeCost{n.Name, instanceType, capacity, rate})
		result.EstimatedHourly += rate
	}
	return result, nil
}

func hourlyRate(instanceType, capacity string) float64 {
	onDemand := map[string]float64{"t3.small": 0.0208, "t3.medium": 0.0416, "t3.large": 0.0832, "m7i.large": 0.1008}
	rate, ok := onDemand[instanceType]
	if !ok {
		rate = 0.05
	}
	if strings.EqualFold(capacity, "SPOT") {
		return rate * 0.35
	}
	return rate
}
