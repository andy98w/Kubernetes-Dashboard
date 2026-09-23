package kubernetes

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"testing"
)

func TestStationEndpointReadiness(t *testing.T) {
	for _, ready := range []bool{true, false} {
		client := fake.NewSimpleClientset(&corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "test"}, Spec: corev1.ServiceSpec{Selector: map[string]string{"app": "api"}}}, &discoveryv1.EndpointSlice{ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "test", Labels: map[string]string{"kubernetes.io/service-name": "api"}}, Endpoints: []discoveryv1.Endpoint{{Addresses: []string{"10.0.0.1"}, Conditions: discoveryv1.EndpointConditions{Ready: &ready}}}})
		result, err := (&Client{client: client}).StationHealth(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		want := "warning"
		if ready {
			want = "ready"
		}
		if result.Services.State != want {
			t.Fatalf("got %s want %s", result.Services.State, want)
		}
		if result.Control.State != "unknown" || result.Telemetry.State != "unknown" {
			t.Fatal("missing probes must not report ready")
		}
	}
}
