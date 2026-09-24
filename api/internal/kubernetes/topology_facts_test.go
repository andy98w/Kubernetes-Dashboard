package kubernetes

import (
	"context"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"testing"
)

func TestNodeStorageAndSchedulingFacts(t *testing.T) {
	client := fake.NewSimpleClientset(
		&core.Node{ObjectMeta: meta.ObjectMeta{Name: "worker", Labels: map[string]string{"topology.kubernetes.io/zone": "us-west-2a"}}, Status: core.NodeStatus{Allocatable: core.ResourceList{core.ResourceCPU: resource.MustParse("2"), core.ResourceMemory: resource.MustParse("4Gi")}, Conditions: []core.NodeCondition{{Type: core.NodeMemoryPressure, Status: core.ConditionTrue}}}},
		&core.PersistentVolumeClaim{ObjectMeta: meta.ObjectMeta{Name: "data", Namespace: "a"}, Status: core.PersistentVolumeClaimStatus{Phase: core.ClaimBound, Capacity: core.ResourceList{core.ResourceStorage: resource.MustParse("10Gi")}}},
		&core.PersistentVolumeClaim{ObjectMeta: meta.ObjectMeta{Name: "data", Namespace: "b"}, Status: core.PersistentVolumeClaimStatus{Phase: core.ClaimPending}},
	)
	pods := []core.Pod{{ObjectMeta: meta.ObjectMeta{Name: "pending", Namespace: "a"}, Spec: core.PodSpec{
		Containers: []core.Container{{Name: "app", VolumeMounts: []core.VolumeMount{{Name: "disk"}, {Name: "cache"}}}},
		Volumes:    []core.Volume{{Name: "disk", VolumeSource: core.VolumeSource{PersistentVolumeClaim: &core.PersistentVolumeClaimVolumeSource{ClaimName: "data"}}}, {Name: "cache", VolumeSource: core.VolumeSource{EmptyDir: &core.EmptyDirVolumeSource{}}}, {Name: "unused", VolumeSource: core.VolumeSource{EmptyDir: &core.EmptyDirVolumeSource{}}}},
	}, Status: core.PodStatus{Phase: core.PodPending, Conditions: []core.PodCondition{{Type: core.PodScheduled, Status: core.ConditionFalse, Reason: "Unschedulable", Message: "Insufficient cpu"}}}},
		{ObjectMeta: meta.ObjectMeta{Name: "finished"}, Status: core.PodStatus{Phase: core.PodSucceeded}}}
	out := Topology{}
	(&Client{client: client}).topologyFacts(context.Background(), pods, &out)
	if len(out.Nodes) != 1 || out.Nodes[0].Zone != "us-west-2a" || out.Nodes[0].CPUCapacity != 2000 || out.Nodes[0].MemoryCapacity != 4294967296 {
		t.Fatalf("nodes: %+v", out.Nodes)
	}
	if out.Nodes[0].CPUMilli != nil || out.Nodes[0].MemoryBytes != nil {
		t.Fatal("missing metrics must not become zero")
	}
	if len(out.Nodes[0].Pressure) != 1 {
		t.Fatal("missing pressure")
	}
	if len(out.Volumes) != 2 || out.Volumes[0].Phase != "Bound" || out.Volumes[0].Capacity != "10Gi" || out.Volumes[1].Kind != "emptyDir" {
		t.Fatalf("volumes: %+v", out.Volumes)
	}
	if len(out.Pending) != 1 || out.Pending[0].Reason != "Unschedulable" {
		t.Fatalf("pending: %+v", out.Pending)
	}
}
