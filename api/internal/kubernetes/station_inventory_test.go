package kubernetes

import (
	"context"
	"fmt"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	"testing"
)

func TestStationInventoryBeyond24(t *testing.T) {
	objects := []runtime.Object{}
	controller := true
	for i := 0; i < 30; i++ {
		name := fmt.Sprintf("app-%d", i)
		uid := types.UID(name)
		objects = append(objects, &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "apps", UID: uid}},
			&appsv1.ReplicaSet{ObjectMeta: metav1.ObjectMeta{Name: name + "-rs", Namespace: "apps", UID: uid + "rs", OwnerReferences: []metav1.OwnerReference{{Kind: "Deployment", Name: name, UID: uid, Controller: &controller}}}},
			&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: name + "-pod", Namespace: "apps", OwnerReferences: []metav1.OwnerReference{{Kind: "ReplicaSet", UID: uid + "rs", Controller: &controller}}}, Spec: corev1.PodSpec{NodeName: "node-1"}})
	}
	objects = append(objects, &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "unowned", Namespace: "apps"}})
	objects = append(objects,
		&appsv1.ReplicaSet{ObjectMeta: metav1.ObjectMeta{Name: "orphan", Namespace: "apps", UID: "orphan-rs", OwnerReferences: []metav1.OwnerReference{{Kind: "Deployment", Name: "app-0", UID: "deleted-deployment", Controller: &controller}}}},
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "orphan", Namespace: "apps", OwnerReferences: []metav1.OwnerReference{{Kind: "ReplicaSet", UID: "orphan-rs", Controller: &controller}}}},
		&corev1.Event{ObjectMeta: metav1.ObjectMeta{Name: "current", Namespace: "apps"}, Reason: "FailedCreate", InvolvedObject: corev1.ObjectReference{Kind: "Deployment", Name: "app-0", UID: "app-0"}},
		&corev1.Event{ObjectMeta: metav1.ObjectMeta{Name: "stale", Namespace: "apps"}, Reason: "FailedCreate", InvolvedObject: corev1.ObjectReference{Kind: "Deployment", Name: "app-0", UID: "deleted-deployment"}},
	)
	client := fake.NewClientset(objects...)
	got, err := NewClient(client, "test").StationInventory(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Residents) != 30 || got.UnmatchedPods != 2 || len(got.Events) != 1 {
		t.Fatalf("incomplete inventory: %+v", got)
	}
	if len(client.Actions()) != 13 {
		t.Fatalf("expected fixed list calls, got %d", len(client.Actions()))
	}
}
