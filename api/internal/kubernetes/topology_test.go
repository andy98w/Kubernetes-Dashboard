package kubernetes

import (
	"context"
	core "k8s.io/api/core/v1"
	discovery "k8s.io/api/discovery/v1"
	networking "k8s.io/api/networking/v1"
	rbac "k8s.io/api/rbac/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"testing"
)

func TestTopologyEvidence(t *testing.T) {
	ready := false
	client := fake.NewSimpleClientset(
		&core.Pod{ObjectMeta: meta.ObjectMeta{Name: "api", Namespace: "shop", Labels: map[string]string{"app": "api"}}, Spec: core.PodSpec{ServiceAccountName: "reader"}},
		&networking.Ingress{ObjectMeta: meta.ObjectMeta{Name: "entry", Namespace: "shop"}, Spec: networking.IngressSpec{DefaultBackend: &networking.IngressBackend{Service: &networking.IngressServiceBackend{Name: "api", Port: networking.ServiceBackendPort{Number: 8080}}}}},
		&discovery.EndpointSlice{ObjectMeta: meta.ObjectMeta{Name: "api", Namespace: "shop", Labels: map[string]string{"kubernetes.io/service-name": "api"}}, Endpoints: []discovery.Endpoint{{Addresses: []string{"10.0.0.1"}, Conditions: discovery.EndpointConditions{Ready: &ready}, TargetRef: &core.ObjectReference{Kind: "Pod", Name: "api", Namespace: "shop"}}}},
		&networking.NetworkPolicy{ObjectMeta: meta.ObjectMeta{Name: "isolate", Namespace: "shop"}, Spec: networking.NetworkPolicySpec{PodSelector: meta.LabelSelector{MatchLabels: map[string]string{"app": "api"}}}},
		&rbac.Role{ObjectMeta: meta.ObjectMeta{Name: "reader", Namespace: "shop"}, Rules: []rbac.PolicyRule{{APIGroups: []string{""}, Resources: []string{"pods"}, Verbs: []string{"get"}}}},
		&rbac.RoleBinding{ObjectMeta: meta.ObjectMeta{Name: "read", Namespace: "shop"}, Subjects: []rbac.Subject{{Kind: "ServiceAccount", Name: "reader", Namespace: "shop"}}, RoleRef: rbac.RoleRef{APIGroup: "rbac.authorization.k8s.io", Kind: "Role", Name: "reader"}},
	)
	out, err := (&Client{client: client}).Topology(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Routes) != 1 || out.Routes[0].Port != "8080" {
		t.Fatalf("routes: %+v", out.Routes)
	}
	if len(out.Endpoints) != 1 || out.Endpoints[0].Ready == nil || *out.Endpoints[0].Ready {
		t.Fatal("lost unready endpoint evidence")
	}
	if len(out.Identities) != 1 || out.Identities[0].ServiceAccount != "reader" || len(out.Identities[0].Grants) != 1 || len(out.Identities[0].Policies) != 1 {
		t.Fatalf("identity: %+v", out.Identities)
	}
}
