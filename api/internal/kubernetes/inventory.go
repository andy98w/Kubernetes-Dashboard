package kubernetes

import (
	"context"
	"fmt"
	"time"

	"github.com/andy98w/Kubernetes-Dashboard/api/internal/config"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type PodCounts struct {
	Running   int `json:"running"`
	Pending   int `json:"pending"`
	Failed    int `json:"failed"`
	Succeeded int `json:"succeeded"`
	Unknown   int `json:"unknown"`
}

type NodeCounts struct {
	Ready int `json:"ready"`
	Total int `json:"total"`
}

type Summary struct {
	Cluster    string     `json:"cluster"`
	Mode       string     `json:"mode"`
	Nodes      NodeCounts `json:"nodes"`
	Namespaces int        `json:"namespaces"`
	Pods       PodCounts  `json:"pods"`
	ObservedAt time.Time  `json:"observedAt"`
}

type Inventory interface {
	Ready(context.Context) error
	Summary(context.Context) (Summary, error)
	Workloads(context.Context) (Workloads, error)
	Network(context.Context) (Network, error)
	Events(context.Context) (Events, error)
	Observability(context.Context) (Observability, error)
	Security(context.Context) (Security, error)
	Cost(context.Context) (Cost, error)
	WorkloadDetail(context.Context, string, string, string) (WorkloadDetail, error)
	Incidents(context.Context) (Incidents, error)
	Updates(context.Context) (<-chan ClusterUpdate, error)
	PlanOperation(context.Context, OperationRequest) (OperationPlan, error)
	ExecuteOperation(context.Context, OperationRequest, string) (OperationRecord, error)
	RecentOperations(context.Context) ([]OperationRecord, error)
}

type Client struct {
	client      kubernetes.Interface
	clusterName string
	operations  operationPolicy
	audit       *operationAudit
	plans       *operationPlans
}

func New(cfg config.Config) (Inventory, error) {
	if cfg.DemoMode {
		return DemoInventory{ClusterName: cfg.ClusterName}, nil
	}

	restConfig, err := loadRESTConfig(cfg.Kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("load Kubernetes client configuration: %w", err)
	}
	restConfig.UserAgent = "kubevista/" + cfg.Version
	restConfig.Timeout = 10 * time.Second

	client, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("create Kubernetes client: %w", err)
	}
	return NewClientWithOperations(client, cfg.ClusterName, cfg.OperationsEnabled, cfg.OperationNamespaces, cfg.MinReplicas, cfg.MaxReplicas), nil
}

func NewClient(client kubernetes.Interface, clusterName string) *Client {
	return NewClientWithOperations(client, clusterName, false, nil, 1, 6)
}

func NewClientWithOperations(client kubernetes.Interface, clusterName string, enabled bool, namespaces []string, minReplicas, maxReplicas int32) *Client {
	allowed := map[string]bool{}
	for _, namespace := range namespaces {
		allowed[namespace] = true
	}
	return &Client{client: client, clusterName: clusterName, operations: operationPolicy{enabled, allowed, minReplicas, maxReplicas}, audit: &operationAudit{records: []OperationRecord{}}, plans: &operationPlans{entries: map[string]operationPlanEntry{}}}
}

func loadRESTConfig(explicitPath string) (*rest.Config, error) {
	if inCluster, err := rest.InClusterConfig(); err == nil {
		return inCluster, nil
	}

	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	if explicitPath != "" {
		rules.ExplicitPath = explicitPath
	}
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, &clientcmd.ConfigOverrides{}).ClientConfig()
}

func (c *Client) Ready(ctx context.Context) error {
	_, err := c.client.CoreV1().Namespaces().List(ctx, metav1.ListOptions{Limit: 1})
	if err != nil {
		return fmt.Errorf("query Kubernetes API: %w", err)
	}
	return nil
}

func (c *Client) Summary(ctx context.Context) (Summary, error) {
	nodes, err := c.client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return Summary{}, fmt.Errorf("list nodes: %w", err)
	}
	namespaces, err := c.client.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return Summary{}, fmt.Errorf("list namespaces: %w", err)
	}
	pods, err := c.client.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return Summary{}, fmt.Errorf("list pods: %w", err)
	}

	summary := Summary{
		Cluster:    c.clusterName,
		Mode:       "cluster",
		Namespaces: len(namespaces.Items),
		ObservedAt: time.Now().UTC(),
		Nodes:      NodeCounts{Total: len(nodes.Items)},
	}

	for _, node := range nodes.Items {
		if nodeReady(node) {
			summary.Nodes.Ready++
		}
	}
	for _, pod := range pods.Items {
		switch pod.Status.Phase {
		case corev1.PodRunning:
			summary.Pods.Running++
		case corev1.PodPending:
			summary.Pods.Pending++
		case corev1.PodFailed:
			summary.Pods.Failed++
		case corev1.PodSucceeded:
			summary.Pods.Succeeded++
		default:
			summary.Pods.Unknown++
		}
	}
	return summary, nil
}

func nodeReady(node corev1.Node) bool {
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}
