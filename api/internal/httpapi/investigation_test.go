package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/andy98w/Kubernetes-Dashboard/api/internal/config"
	cluster "github.com/andy98w/Kubernetes-Dashboard/api/internal/kubernetes"
)

func TestInvestigationAccess(t *testing.T) {
	for _, tc := range []struct {
		name, enabled, environment, namespace, header string
		code                                          int
	}{
		{"disabled", "", "development", "kubevista", "reviewed", 404},
		{"production", "true", "production", "kubevista", "reviewed", 404},
		{"namespace", "true", "development", "other", "reviewed", 403},
		{"header", "true", "development", "kubevista", "", 403},
		{"preview", "true", "development", "kubevista", "reviewed", 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("KUBEVISTA_INVESTIGATION_ENABLED", tc.enabled)
			t.Setenv("KUBEVISTA_LOCAL_MODEL", "")
			cfg := config.Config{Environment: tc.environment, DemoMode: true, OperationNamespaces: []string{"kubevista"}}
			r := httptest.NewRequest("POST", "/api/v1/investigate/"+tc.namespace+"/Deployment/kubevista-api", nil)
			r.Header.Set("X-KubeVista-Investigation", tc.header)
			w := httptest.NewRecorder()
			New(cfg, cluster.DemoInventory{}).ServeHTTP(w, r)
			if w.Code != tc.code {
				t.Fatalf("%d: %s", w.Code, w.Body.String())
			}
			if tc.code == 200 && !strings.Contains(w.Body.String(), "evidence-only") {
				t.Fatal("preview mislabeled")
			}
		})
	}
}

func TestEvidenceBoundsAndRedaction(t *testing.T) {
	d := cluster.WorkloadDetail{}
	for i := 0; i < 100; i++ {
		d.Logs = append(d.Logs, cluster.LogExcerpt{Text: "password=hunter2\nAuthorization: Bearer abc\n" + strings.Repeat("x", 5000)})
	}
	packet := evidencePacket(d)
	if len(packet.Evidence) != 3 {
		t.Fatal("unbounded logs")
	}
	for _, e := range packet.Evidence {
		if len(e.Text) > 1600 || strings.Contains(e.Text, "hunter2") || strings.Contains(e.Text, "abc") {
			t.Fatal("redaction/bound failure")
		}
	}
}

func TestModelCitationValidation(t *testing.T) {
	for _, tc := range []struct {
		content string
		valid   bool
	}{
		{`{"hypotheses":[{"explanation":"Possible probe failure; check timing","evidenceIds":["E1"]}]}`, true},
		{`{"hypotheses":[]}`, true},
		{`{"hypotheses":[{"explanation":"Invented evidence","evidenceIds":["E99"]}]}`, false},
		{`{"hypotheses":[{"explanation":"Uncited","evidenceIds":[]}]}`, false},
		{`not json`, false},
	} {
		t.Run(tc.content, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var request map[string]any
				json.NewDecoder(r.Body).Decode(&request)
				if _, ok := request["tools"]; ok {
					t.Error("model has tools")
				}
				json.NewEncoder(w).Encode(map[string]any{"message": map[string]string{"content": tc.content}})
			}))
			defer server.Close()
			_, err := explain(context.Background(), server.Client(), server.URL, "fixture", investigation{Evidence: []evidenceItem{{"E1", "event", "probe failed"}}})
			if (err == nil) != tc.valid {
				t.Fatalf("unexpected validation: %v", err)
			}
		})
	}
}
