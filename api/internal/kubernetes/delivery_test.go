package kubernetes

import (
	"context"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"testing"
)

func TestDeliveryReadiness(t *testing.T) {
	replicas := int32(2)
	d := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "kubevista-api", Namespace: "kubevista", Generation: 2, Labels: map[string]string{"app.kubernetes.io/name": "kubevista"}}, Spec: appsv1.DeploymentSpec{Replicas: &replicas, Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "api", Image: "api@sha256:abc"}}}}}, Status: appsv1.DeploymentStatus{ObservedGeneration: 1, UpdatedReplicas: 2, AvailableReplicas: 2, ReadyReplicas: 2, Replicas: 2}}
	c := &Client{client: fake.NewSimpleClientset(d)}
	got, err := c.Delivery(context.Background())
	if err != nil || len(got.Workloads) != 1 || got.Workloads[0].State != "progressing" {
		t.Fatalf("stale generation: %+v %v", got, err)
	}
	d.Status.ObservedGeneration = 2
	if _, err = c.client.AppsV1().Deployments("kubevista").UpdateStatus(context.Background(), d, metav1.UpdateOptions{}); err != nil {
		t.Fatal(err)
	}
	got, err = c.Delivery(context.Background())
	if err != nil || got.Workloads[0].State != "ready" {
		t.Fatalf("ready: %+v %v", got, err)
	}
	d.Status.Conditions = []appsv1.DeploymentCondition{{Type: appsv1.DeploymentProgressing, Status: corev1.ConditionFalse, Reason: "ProgressDeadlineExceeded"}}
	_, _ = c.client.AppsV1().Deployments("kubevista").UpdateStatus(context.Background(), d, metav1.UpdateOptions{})
	got, _ = c.Delivery(context.Background())
	if got.Workloads[0].State != "failed" {
		t.Fatal("deadline failure not represented")
	}
}
