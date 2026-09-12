package kubernetes

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	kt "k8s.io/client-go/testing"
)

func rollbackFixture() (*Client, *fake.Clientset, OperationRequest) {
	d := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "kubevista", UID: "deployment-uid", ResourceVersion: "7", Annotations: map[string]string{"deployment.kubernetes.io/revision": "2"}}, Spec: appsv1.DeploymentSpec{Replicas: ptr(int32(1)), Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "web", Image: "nginx:broken"}}}}}}
	rs := &appsv1.ReplicaSet{ObjectMeta: metav1.ObjectMeta{Name: "previous", Namespace: "kubevista", UID: "rs-uid", Annotations: map[string]string{"deployment.kubernetes.io/revision": "1"}, OwnerReferences: []metav1.OwnerReference{{Kind: "Deployment", Name: "api", UID: d.UID, Controller: ptr(true)}}}, Spec: appsv1.ReplicaSetSpec{Template: corev1.PodTemplateSpec{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"pod-template-hash": "old", "app": "api"}}, Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "web", Image: "nginx:good"}}}}}}
	cs := fake.NewSimpleClientset(d, rs)
	// client-go's tracker does not implement admission or dry run. Ensure test
	// dry runs return without mutation; real semantics are exercised in kind CI.
	cs.PrependReactor("patch", "deployments", func(a kt.Action) (bool, runtime.Object, error) {
		opts := a.(interface{ GetPatchOptions() metav1.PatchOptions }).GetPatchOptions()
		if len(opts.DryRun) > 0 {
			obj, e := cs.Tracker().Get(appsv1.SchemeGroupVersion.WithResource("deployments"), "kubevista", "api")
			return true, obj, e
		}
		return false, nil, nil
	})
	c := NewClientWithOperations(cs, "test", true, []string{"kubevista"}, 1, 6)
	return c, cs, OperationRequest{Action: OperationRollback, Namespace: "kubevista", Kind: "Deployment", Name: "api", Reason: "Recover broken image revision"}
}

func TestRollbackPinsReviewedTemplateAndRejectsReplay(t *testing.T) {
	c, cs, r := rollbackFixture()
	ctx := context.Background()
	plan, e := c.PlanOperation(ctx, r)
	if e != nil {
		t.Fatal(e)
	}
	if plan.RollbackRevision != "1" || len(plan.TemplateDiff) == 0 {
		t.Fatalf("missing review: %+v", plan)
	}
	d, _ := cs.AppsV1().Deployments(r.Namespace).Get(ctx, r.Name, metav1.GetOptions{})
	if d.Spec.Template.Spec.Containers[0].Image != "nginx:broken" {
		t.Fatal("dry run mutated deployment")
	}
	// Mutating historical RS after review must not change what execution applies.
	rs, _ := cs.AppsV1().ReplicaSets(r.Namespace).Get(ctx, "previous", metav1.GetOptions{})
	rs.Spec.Template.Spec.Containers[0].Image = "nginx:surprise"
	_, _ = cs.AppsV1().ReplicaSets(r.Namespace).Update(ctx, rs, metav1.UpdateOptions{})
	r.PlanID, r.ExpectedResourceVersion = plan.ID, plan.ResourceVersion
	_, e = c.ExecuteOperation(ctx, r, "test-operator")
	if e != nil {
		t.Fatal(e)
	}
	d, _ = cs.AppsV1().Deployments(r.Namespace).Get(ctx, r.Name, metav1.GetOptions{})
	if d.Spec.Template.Spec.Containers[0].Image != "nginx:good" || d.Spec.Template.Labels["pod-template-hash"] != "" {
		t.Fatalf("wrong rollback: %+v", d.Spec.Template)
	}
	if _, e = c.ExecuteOperation(ctx, r, "test-operator"); e == nil {
		t.Fatal("replay accepted")
	}
}

func TestRollbackRejectsStaleOrUnownedHistory(t *testing.T) {
	c, cs, r := rollbackFixture()
	ctx := context.Background()
	p, e := c.PlanOperation(ctx, r)
	if e != nil {
		t.Fatal(e)
	}
	d, _ := cs.AppsV1().Deployments(r.Namespace).Get(ctx, r.Name, metav1.GetOptions{})
	d.ResourceVersion = "8"
	_, _ = cs.AppsV1().Deployments(r.Namespace).Update(ctx, d, metav1.UpdateOptions{})
	r.PlanID, r.ExpectedResourceVersion = p.ID, p.ResourceVersion
	if _, e = c.ExecuteOperation(ctx, r, "operator"); e == nil {
		t.Fatal("stale accepted")
	}
	rs, _ := cs.AppsV1().ReplicaSets(r.Namespace).Get(ctx, "previous", metav1.GetOptions{})
	rs.OwnerReferences[0].UID = types.UID("other-deployment")
	_, _ = cs.AppsV1().ReplicaSets(r.Namespace).Update(ctx, rs, metav1.UpdateOptions{})
	if _, e = c.PlanOperation(ctx, r); e == nil {
		t.Fatal("unowned history accepted")
	}
}

func TestRecoveryWaitsForCurrentGenerationAndOldPods(t *testing.T) {
	d := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Generation: 4}, Spec: appsv1.DeploymentSpec{Replicas: ptr(int32(1))}, Status: appsv1.DeploymentStatus{ObservedGeneration: 3, UpdatedReplicas: 1, AvailableReplicas: 1, Replicas: 1}}
	if deploymentRecovery(d).Status == "Recovered" {
		t.Fatal("old observation is not recovery")
	}
	d.Status.ObservedGeneration = 4
	d.Status.Replicas = 2
	if deploymentRecovery(d).Status == "Recovered" {
		t.Fatal("old replicas still exist")
	}
	d.Status.Replicas = 1
	if deploymentRecovery(d).Status != "Recovered" {
		t.Fatal("settled rollout should recover")
	}
}

func TestTemplateDiffOmitsValues(t *testing.T) {
	a := corev1.PodTemplateSpec{Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "x", Env: []corev1.EnvVar{{Name: "TOKEN", Value: "old"}}}}}}
	b := a.DeepCopy()
	b.Spec.Containers[0].Env[0].Value = "private-value"
	diff := templateChanges(a, *b)
	raw, _ := json.Marshal(diff)
	if len(diff) != 1 || strings.Contains(string(raw), "private-value") {
		t.Fatalf("unsafe diff: %s", raw)
	}
}
