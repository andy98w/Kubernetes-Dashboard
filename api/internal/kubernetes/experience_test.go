package kubernetes

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/fake"
)

func TestWorkloadDetailCorrelatesReadOnlyResources(t *testing.T) {
	selector := map[string]string{"app": "api"}
	objects := []runtime.Object{
		&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "platform"}, Spec: appsv1.DeploymentSpec{Replicas: ptr(int32(1)), Selector: &metav1.LabelSelector{MatchLabels: selector}, Template: corev1.PodTemplateSpec{ObjectMeta: metav1.ObjectMeta{Labels: selector}, Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "api", Image: "example/api@sha256:abc"}}}}}, Status: appsv1.DeploymentStatus{ReadyReplicas: 1}},
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "api-123", Namespace: "platform", Labels: selector}, Spec: corev1.PodSpec{NodeName: "node-a", Containers: []corev1.Container{{Name: "api", Image: "example/api@sha256:abc"}}}, Status: corev1.PodStatus{Phase: corev1.PodRunning, ContainerStatuses: []corev1.ContainerStatus{{Name: "api", Ready: true}}}},
		&corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "platform"}, Spec: corev1.ServiceSpec{Selector: selector, ClusterIP: "10.0.0.10", Ports: []corev1.ServicePort{{Port: 80, TargetPort: intstr.FromInt32(8080)}}}},
		&networkingv1.NetworkPolicy{ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "platform"}, Spec: networkingv1.NetworkPolicySpec{PodSelector: metav1.LabelSelector{MatchLabels: selector}}},
		&corev1.Event{ObjectMeta: metav1.ObjectMeta{Name: "event", Namespace: "platform"}, Type: "Warning", Reason: "Unhealthy", Message: "probe failed", InvolvedObject: corev1.ObjectReference{Kind: "Pod", Name: "api-123"}},
	}
	detail, err := NewClient(fake.NewSimpleClientset(objects...), "test").WorkloadDetail(context.Background(), "platform", "Deployment", "api")
	if err != nil {
		t.Fatalf("WorkloadDetail() error = %v", err)
	}
	if len(detail.Pods) != 1 || len(detail.Images) != 1 || len(detail.Services) != 1 || len(detail.Policies) != 1 || len(detail.Events) != 1 {
		t.Fatalf("unexpected correlated detail: %+v", detail)
	}
}

func TestIncidentsCorrelatesWarningEvents(t *testing.T) {
	client := fake.NewSimpleClientset(&corev1.Event{ObjectMeta: metav1.ObjectMeta{Name: "warning", Namespace: "platform"}, Type: "Warning", Reason: "BackOff", Message: "container restart", InvolvedObject: corev1.ObjectReference{Kind: "Pod", Name: "api"}})
	incidents, err := NewClient(client, "test").Incidents(context.Background())
	if err != nil {
		t.Fatalf("Incidents() error = %v", err)
	}
	if len(incidents.Items) != 1 || incidents.Items[0].Severity != "High" {
		t.Fatalf("unexpected incidents: %+v", incidents)
	}
}

func ptr[T any](value T) *T { return &value }

func TestDeploymentEvidenceUsesOwnershipAndLabelExpressions(t *testing.T) {
	owner := []metav1.OwnerReference{{Kind: "Deployment", Name: "api", UID: "d1", Controller: ptr(true)}}
	podOwner := []metav1.OwnerReference{{Kind: "ReplicaSet", Name: "api-rs", UID: "r1", Controller: ptr(true)}}
	d := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "platform", UID: "d1"}, Spec: appsv1.DeploymentSpec{Selector: &metav1.LabelSelector{MatchExpressions: []metav1.LabelSelectorRequirement{{Key: "app", Operator: metav1.LabelSelectorOpIn, Values: []string{"api"}}}}}}
	rs := &appsv1.ReplicaSet{ObjectMeta: metav1.ObjectMeta{Name: "api-rs", Namespace: "platform", UID: "r1", OwnerReferences: owner}}
	pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "owned", Namespace: "platform", UID: "p1", Labels: map[string]string{"app": "api"}, OwnerReferences: podOwner}}
	stranger := pod.DeepCopy()
	stranger.Name, stranger.UID, stranger.OwnerReferences = "stranger", "p2", nil
	stranger.Status.ContainerStatuses = []corev1.ContainerStatus{{State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "CrashLoopBackOff"}}}}
	wrongLabel := pod.DeepCopy()
	wrongLabel.Name, wrongLabel.UID = "wrong-label", "p3"
	wrongLabel.Labels = map[string]string{"app": "other"}
	oldEvent := &corev1.Event{ObjectMeta: metav1.ObjectMeta{Name: "old-pod-event", Namespace: "platform"}, Type: "Warning", Reason: "Unhealthy", Message: "readiness probe failed", InvolvedObject: corev1.ObjectReference{Kind: "Pod", Name: "owned", UID: "old-pod-uid"}}
	c := NewClient(fake.NewSimpleClientset(d, rs, pod, stranger, wrongLabel, oldEvent), "test")
	detail, err := c.WorkloadDetail(context.Background(), "platform", "Deployment", "api")
	if err != nil {
		t.Fatal(err)
	}
	if len(detail.Pods) != 1 || detail.Pods[0].Name != "owned" || len(detail.Diagnoses) != 0 || len(detail.Events) != 0 {
		t.Fatalf("unrelated evidence leaked into workload: %+v", detail)
	}
}
