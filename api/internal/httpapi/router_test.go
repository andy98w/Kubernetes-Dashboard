package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/andy98w/Kubernetes-Dashboard/api/internal/config"
)

func TestHealth(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	New(config.Config{Version: "test", Environment: "test"}).ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if got := w.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("expected security header, got %q", got)
	}
}
