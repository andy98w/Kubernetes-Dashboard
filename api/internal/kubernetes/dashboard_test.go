package kubernetes

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestDashboardInventories(t *testing.T) {
	replicas := int32(2)
	privileged := true
	allowEscalation := false
	ingressClass := "alb"
	client := NewClient(fake.NewSimpleClientset(
		&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "kubevista"}, Spec: appsv1.DeploymentSpec{Replicas: &replicas}, Status: appsv1.DeploymentStatus{ReadyReplicas: 2}},
		&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "grafana", Namespace: "observability"}, Spec: appsv1.DeploymentSpec{Replicas: &replicas}, Status: appsv1.DeploymentStatus{ReadyReplicas: 2}},
		&corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: "kubevista"}, Spec: corev1.ServiceSpec{ClusterIP: "172.20.0.10", Ports: []corev1.ServicePort{{Port: 80}}}},
		&networkingv1.Ingress{ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: "kubevista"}, Spec: networkingv1.IngressSpec{IngressClassName: &ingressClass, Rules: []networkingv1.IngressRule{{Host: "kubevista.example.com"}}}},
		&networkingv1.NetworkPolicy{ObjectMeta: metav1.ObjectMeta{Name: "default-deny", Namespace: "kubevista"}, Spec: networkingv1.NetworkPolicySpec{PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress}}},
		&corev1.Event{ObjectMeta: metav1.ObjectMeta{Name: "event", Namespace: "kubevista"}, Type: "Warning", Reason: "Unhealthy", Message: "probe failed", InvolvedObject: corev1.ObjectReference{Kind: "Pod", Name: "api"}},
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "unsafe", Namespace: "kubevista"}, Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "app", Image: "example:latest", SecurityContext: &corev1.SecurityContext{Privileged: &privileged, AllowPrivilegeEscalation: &allowEscalation}}}}},
		&corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node-a", Labels: map[string]string{"node.kubernetes.io/instance-type": "t3.medium", "eks.amazonaws.com/capacityType": "SPOT"}}},
	), "test-cluster")

	ctx := context.Background()
	workloads, err := client.Workloads(ctx)
	if err != nil || len(workloads.Items) != 2 {
		t.Fatalf("Workloads() = %+v, %v", workloads, err)
	}
	network, err := client.Network(ctx)
	if err != nil || len(network.Services) != 1 || len(network.Ingresses) != 1 || len(network.Policies) != 1 {
		t.Fatalf("Network() = %+v, %v", network, err)
	}
	events, err := client.Events(ctx)
	if err != nil || len(events.Items) != 1 || events.Items[0].Type != "Warning" {
		t.Fatalf("Events() = %+v, %v", events, err)
	}
	observability, err := client.Observability(ctx)
	if err != nil || len(observability.Components) != 1 {
		t.Fatalf("Observability() = %+v, %v", observability, err)
	}
	security, err := client.Security(ctx)
	if err != nil || len(security.Findings) == 0 || security.Findings[0].Severity != "Critical" {
		t.Fatalf("Security() = %+v, %v", security, err)
	}
	cost, err := client.Cost(ctx)
	if err != nil || len(cost.Nodes) != 1 || cost.EstimatedHourly <= cost.ControlPlaneHourly {
		t.Fatalf("Cost() = %+v, %v", cost, err)
	}
}
