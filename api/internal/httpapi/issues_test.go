package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/andy98w/Kubernetes-Dashboard/api/internal/config"
)

func TestIssuesPersistAuthorizeAndCompareVersions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "issues.json")
	token := strings.Repeat("x", 32)
	t.Setenv("KUBEVISTA_ISSUE_STORE", path)
	t.Setenv("KUBEVISTA_ISSUE_TOKEN", token)
	mux := http.NewServeMux()
	registerIssues(mux, config.Config{Environment: "development"})
	call := func(method, path, body, key string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Authorization", "Bearer "+key)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		return w
	}
	body := `{"service":"web","release":"r1","type":"TypeError","location":"browser/runtime"}`
	if got := call("POST", "/api/v1/issues", body, ""); got.Code != 401 {
		t.Fatal(got.Code)
	}
	got := call("POST", "/api/v1/issues", body, token)
	if got.Code != 201 {
		t.Fatal(got.Body.String())
	}
	var item issue
	if err := json.Unmarshal(got.Body.Bytes(), &item); err != nil {
		t.Fatal(err)
	}
	if call("POST", "/api/v1/issues/"+item.ID+"/resolve", `{"version":0}`, token).Code != 409 {
		t.Fatal("stale resolve accepted")
	}
	if call("POST", "/api/v1/issues/"+item.ID+"/resolve", `{"version":1}`, token).Code != 200 {
		t.Fatal("resolve failed")
	}
	got = call("POST", "/api/v1/issues", body, token)
	json.Unmarshal(got.Body.Bytes(), &item)
	if item.Status != "Regressed" || item.Count != 2 {
		t.Fatal(item)
	}
	reopened, err := newIssueStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if reopened.items[item.ID].Count != 2 {
		t.Fatal("not persisted")
	}
	if call("POST", "/api/v1/issues", strings.Replace(body, "browser/runtime", "https://private.test?secret=abc", 1), token).Code != 400 {
		t.Fatal("URL accepted")
	}
	if call("POST", "/api/v1/issues", strings.TrimSuffix(body, "}")+`,"message":"private"}`, token).Code != 400 {
		t.Fatal("unbounded field accepted")
	}
	production := http.NewServeMux()
	registerIssues(production, config.Config{Environment: "production"})
	w := httptest.NewRecorder()
	production.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/issues", nil))
	if w.Code != 503 {
		t.Fatal("production accidentally enabled")
	}
}

func TestIssueServiceBoundsAndFailedStorage(t *testing.T) {
	token := strings.Repeat("x", 32)
	path := filepath.Join(t.TempDir(), "issues.json")
	t.Setenv("KUBEVISTA_ISSUE_STORE", path)
	t.Setenv("KUBEVISTA_ISSUE_TOKEN", token)
	mux := http.NewServeMux()
	registerIssues(mux, config.Config{Environment: "development"})
	post := func(body string) int {
		r := httptest.NewRequest("POST", "/api/v1/issues", strings.NewReader(body))
		r.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		return w.Code
	}
	body := `{"service":"web","release":"r1","type":"Error","location":"runtime"}`
	if post(strings.Repeat("x", 3000)) != 400 {
		t.Fatal("oversized request accepted")
	}
	if post(strings.Replace(body, "runtime", "https://private.test", 1)) != 400 {
		t.Fatal("URL accepted")
	}
	for i := 0; i < 60; i++ {
		if post(body) != 201 {
			t.Fatal("valid event failed")
		}
	}
	if post(body) != 429 {
		t.Fatal("rate limit missing")
	}

	// A failed rename must never be acknowledged as a successful persisted event.
	brokenPath := filepath.Join(t.TempDir(), "issues.json")
	s, err := newIssueStore(brokenPath)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Mkdir(brokenPath, 0700); err != nil {
		t.Fatal(err)
	}
	if s.save(map[string]issue{"id": {ID: "id"}}) == nil {
		t.Fatal("write failure ignored")
	}
	if len(s.items) != 0 {
		t.Fatal("failed write changed memory")
	}
}
