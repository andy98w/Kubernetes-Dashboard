package kubernetes

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestSummaryCountsClusterResources(t *testing.T) {
	client := fake.NewSimpleClientset(
		&corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "ready"}, Status: corev1.NodeStatus{Conditions: []corev1.NodeCondition{{Type: corev1.NodeReady, Status: corev1.ConditionTrue}}}},
		&corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "not-ready"}},
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "default"}},
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "platform"}},
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "running", Namespace: "default"}, Status: corev1.PodStatus{Phase: corev1.PodRunning}},
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "failed", Namespace: "default"}, Status: corev1.PodStatus{Phase: corev1.PodFailed}},
	)

	summary, err := NewClient(client, "test-cluster").Summary(context.Background())
	if err != nil {
		t.Fatalf("Summary() error = %v", err)
	}
	if summary.Cluster != "test-cluster" || summary.Nodes.Total != 2 || summary.Nodes.Ready != 1 {
		t.Fatalf("unexpected node summary: %+v", summary)
	}
	if summary.Namespaces != 2 || summary.Pods.Running != 1 || summary.Pods.Failed != 1 {
		t.Fatalf("unexpected resource summary: %+v", summary)
	}
}
