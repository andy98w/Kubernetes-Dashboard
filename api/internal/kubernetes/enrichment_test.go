package kubernetes

import (
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPrometheusSampleAndMissingData(t *testing.T) {
	for _, tc := range []struct {
		body string
		ok   bool
	}{
		{`{"status":"success","data":{"result":[{"value":[1000,"0.02"]}]}}`, true},
		{`{"status":"success","data":{"result":[]}}`, false},
		{`{"status":"success","data":{"result":[{"value":[1000,"NaN"]}]}}`, false},
	} {
		t.Run(tc.body, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Query().Get("query") != "sum(test)" {
					t.Error("missing query")
				}
				fmt.Fprint(w, tc.body)
			}))
			defer server.Close()
			_, err := queryMetric(context.Background(), server.URL, "test", "sum(test)")
			if (err == nil) != tc.ok {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func BenchmarkDiagnoseThousandPods(b *testing.B) {
	pods := makeFailurePods(1000)
	b.ReportAllocs()
	for b.Loop() {
		diagnosePods(pods)
	}
}

func makeFailurePods(n int) []corev1.Pod {
	pods := make([]corev1.Pod, n)
	for i := range pods {
		pods[i].Name = fmt.Sprintf("pod-%d", i)
		pods[i].Status.ContainerStatuses = []corev1.ContainerStatus{{Name: "app", State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "CrashLoopBackOff", Message: "exit code 1"}}}}
	}
	return pods
}
