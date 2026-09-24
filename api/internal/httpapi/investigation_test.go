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
	t.Setenv("KUBEVISTA_INVESTIGATION_TOKEN", strings.Repeat("a", 32))
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
			r.Header.Set("Authorization", "Bearer "+strings.Repeat("a", 32))
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

func TestInvestigationRequiresCredential(t *testing.T) {
	t.Setenv("KUBEVISTA_INVESTIGATION_ENABLED", "true")
	t.Setenv("KUBEVISTA_INVESTIGATION_TOKEN", strings.Repeat("a", 32))
	for _, authorization := range []string{"", "Bearer wrong"} {
		r := httptest.NewRequest("POST", "/api/v1/investigate/kubevista/Deployment/kubevista-api", nil)
		r.Header.Set("X-KubeVista-Investigation", "reviewed")
		r.Header.Set("Authorization", authorization)
		w := httptest.NewRecorder()
		New(config.Config{Environment: "development", DemoMode: true, OperationNamespaces: []string{"kubevista"}}, cluster.DemoInventory{}).ServeHTTP(w, r)
		if w.Code != 401 {
			t.Fatalf("expected 401, got %d", w.Code)
		}
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

type enrichmentSpy struct {
	cluster.DemoInventory
	calls int
}

func (s *enrichmentSpy) Enrich(_ context.Context, d *cluster.WorkloadDetail) {
	s.calls++
	d.Logs = []cluster.LogExcerpt{{Pod: "test", Container: "web", Text: "password=must-not-leak"}}
}

func TestInvestigationLogsRequireExplicitOptIn(t *testing.T) {
	t.Setenv("KUBEVISTA_INVESTIGATION_TOKEN", strings.Repeat("a", 32))
	t.Setenv("KUBEVISTA_INVESTIGATION_ENABLED", "true")
	t.Setenv("KUBEVISTA_LOCAL_MODEL", "")
	for _, include := range []bool{false, true} {
		inventory := &enrichmentSpy{}
		cfg := config.Config{Environment: "development", OperationNamespaces: []string{"kubevista"}}
		r := httptest.NewRequest("POST", "/api/v1/investigate/kubevista/Deployment/kubevista-api", nil)
		r.Header.Set("X-KubeVista-Investigation", "reviewed")
		r.Header.Set("Authorization", "Bearer "+strings.Repeat("a", 32))
		if include {
			r.Header.Set("X-KubeVista-Include-Logs", "true")
		}
		w := httptest.NewRecorder()
		New(cfg, inventory).ServeHTTP(w, r)
		if w.Code != 200 {
			t.Fatalf("unexpected status: %d", w.Code)
		}
		if (inventory.calls == 1) != include {
			t.Fatal("enrichment consent not respected")
		}
		if strings.Contains(w.Body.String(), "must-not-leak") {
			t.Fatal("credential leaked")
		}
		if w.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("sensitive evidence may be cached")
		}
	}
}
