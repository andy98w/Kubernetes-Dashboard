package kubernetes

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestGuardedRestartPlanAndExecute(t *testing.T) {
	replicas := int32(2)
	clientset := fake.NewSimpleClientset(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "kubevista", ResourceVersion: "7"},
		Spec:       appsv1.DeploymentSpec{Replicas: &replicas, Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "api"}}, Template: corev1.PodTemplateSpec{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "api"}}, Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "api", Image: "example/api:test"}}}}},
	})
	client := NewClientWithOperations(clientset, "test", true, []string{"kubevista"}, 1, 6)
	request := OperationRequest{Action: OperationRestart, Namespace: "kubevista", Kind: "Deployment", Name: "api", Reason: "Recover a stalled rollout"}
	plan, err := client.PlanOperation(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if !plan.DryRun || !plan.Authorized || plan.ResourceVersion != "7" || plan.ID == "" {
		t.Fatalf("unexpected plan: %+v", plan)
	}
	request.PlanID, request.ExpectedResourceVersion = plan.ID, plan.ResourceVersion
	record, err := client.ExecuteOperation(context.Background(), request, "operator-subject")
	if err != nil {
		t.Fatal(err)
	}
	if record.Status != "Accepted" || record.Actor != "operator-subject" || record.Target != "kubevista/Deployment/api" {
		t.Fatalf("unexpected record: %+v", record)
	}
	history, err := client.RecentOperations(context.Background())
	if err != nil || len(history) != 1 || history[0].ID != plan.ID {
		t.Fatalf("unexpected history: %+v, %v", history, err)
	}
	if _, err := client.ExecuteOperation(context.Background(), request, "operator-subject"); err == nil {
		t.Fatal("expected consumed plan to reject replay")
	}
}

func TestGuardedOperationRejectsNamespaceAndReplicaRange(t *testing.T) {
	client := NewClientWithOperations(fake.NewSimpleClientset(), "test", true, []string{"kubevista"}, 1, 6)
	request := OperationRequest{Action: OperationRestart, Namespace: "kube-system", Kind: "Deployment", Name: "api", Reason: "Restart for testing"}
	if _, err := client.PlanOperation(context.Background(), request); err == nil {
		t.Fatal("expected namespace denial")
	}
	replicas := int32(9)
	request = OperationRequest{Action: OperationScale, Namespace: "kubevista", Kind: "Deployment", Name: "api", Reason: "Scale for testing", Replicas: &replicas}
	if _, err := client.PlanOperation(context.Background(), request); err == nil {
		t.Fatal("expected replica guardrail denial")
	}
}

func TestGuardedOperationBindsPlanToReviewedRequest(t *testing.T) {
	replicas := int32(2)
	clientset := fake.NewSimpleClientset(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "kubevista", ResourceVersion: "7"},
		Spec:       appsv1.DeploymentSpec{Replicas: &replicas, Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "api"}}, Template: corev1.PodTemplateSpec{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "api"}}, Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "api", Image: "example/api:test"}}}}},
	})
	client := NewClientWithOperations(clientset, "test", true, []string{"kubevista"}, 1, 6)
	request := OperationRequest{Action: OperationRestart, Namespace: "kubevista", Kind: "Deployment", Name: "api", Reason: "Recover a stalled rollout"}
	plan, err := client.PlanOperation(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	request.PlanID, request.ExpectedResourceVersion = plan.ID, plan.ResourceVersion
	request.Reason = "A different operation reason"
	if _, err := client.ExecuteOperation(context.Background(), request, "operator-subject"); err == nil {
		t.Fatal("expected changed request to reject reviewed plan")
	}
	request.PlanID = "forged-plan"
	request.ExpectedResourceVersion = "7"
	if _, err := client.ExecuteOperation(context.Background(), request, "operator-subject"); err == nil {
		t.Fatal("expected unknown plan ID to be rejected")
	}
}
