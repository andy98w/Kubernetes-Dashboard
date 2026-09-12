package main

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDiagnoseAndVerifyCLI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/workloads/demo/Deployment/app" {
			t.Errorf("wrong target: %s", r.URL.Path)
		}
		fmt.Fprint(w, `{"diagnoses":[],"recovery":{"status":"Recovered","generation":2,"observedGeneration":2}}`)
	}))
	defer server.Close()
	for _, command := range []string{"diagnose", "verify"} {
		var out bytes.Buffer
		if err := run([]string{command, "--api", server.URL, "--namespace", "demo", "--name", "app"}, &out); err != nil {
			t.Fatal(err)
		}
		if !bytes.Contains(out.Bytes(), []byte("Recovered")) {
			t.Fatal(out.String())
		}
	}
}

func TestCLIPropagatesDeniedRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "denied", 403) }))
	defer server.Close()
	var out bytes.Buffer
	if err := run([]string{"plan", "--api", server.URL, "--reason", "test deny"}, &out); err == nil {
		t.Fatal("denial swallowed")
	}
}
