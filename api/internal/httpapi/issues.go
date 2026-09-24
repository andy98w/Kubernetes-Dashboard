package httpapi

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/andy98w/Kubernetes-Dashboard/api/internal/config"
)

type issue struct {
	ID        string    `json:"id"`
	Service   string    `json:"service"`
	Release   string    `json:"release"`
	Type      string    `json:"type"`
	Location  string    `json:"location"`
	Count     int       `json:"count"`
	Version   int       `json:"version"`
	Status    string    `json:"status"`
	FirstSeen time.Time `json:"firstSeen"`
	LastSeen  time.Time `json:"lastSeen"`
}
type issueStore struct {
	mu       sync.Mutex
	path     string
	items    map[string]issue
	window   time.Time
	received int
}

var safeIssueField = regexp.MustCompile(`^[a-zA-Z0-9_.:/-]{1,160}$`)

// Single-process local store. Explicitly disabled in production; not an HA database.
func newIssueStore(path string) (*issueStore, error) {
	s := &issueStore{path: path, items: map[string]issue{}}
	if !filepath.IsAbs(path) {
		return nil, errors.New("absolute issue store path required")
	}
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, 4<<20))
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(raw, &s.items); err != nil {
		return nil, err
	}
	if s.items == nil || len(s.items) > 1000 {
		return nil, errors.New("invalid issue store")
	}
	return s, nil
}
func (s *issueStore) save(next map[string]issue) error {
	raw, err := json.Marshal(next)
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(s.path), 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(s.path), ".issues-*")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	if _, err = f.Write(raw); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	if err = os.Rename(name, s.path); err != nil {
		return err
	}
	s.items = next
	return nil
}
func (s *issueStore) copy() map[string]issue {
	next := map[string]issue{}
	for k, v := range s.items {
		next[k] = v
	}
	return next
}
func decodeIssue(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 2048)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if decoder.Decode(v) != nil || decoder.Decode(&struct{}{}) != io.EOF {
		writeJSON(w, 400, map[string]string{"error": "invalid issue request"})
		return false
	}
	return true
}
func registerIssues(mux *http.ServeMux, cfg config.Config) {
	token, path := os.Getenv("KUBEVISTA_ISSUE_TOKEN"), os.Getenv("KUBEVISTA_ISSUE_STORE")
	var store *issueStore
	if len(token) >= 32 && path != "" && !strings.EqualFold(cfg.Environment, "production") {
		store, _ = newIssueStore(path)
	}
	guard := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "no-store")
			if store == nil {
				writeJSON(w, 503, map[string]string{"error": "local issue service is not configured"})
				return
			}
			got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") || subtle.ConstantTimeCompare([]byte(got), []byte(token)) != 1 {
				writeJSON(w, 401, map[string]string{"error": "issue-service credentials required"})
				return
			}
			next(w, r)
		}
	}
	mux.HandleFunc("GET /api/v1/issues", guard(func(w http.ResponseWriter, r *http.Request) {
		store.mu.Lock()
		defer store.mu.Unlock()
		items := []issue{}
		for _, v := range store.items {
			items = append(items, v)
		}
		sort.Slice(items, func(i, j int) bool { return items[i].LastSeen.After(items[j].LastSeen) })
		writeJSON(w, 200, map[string]any{"items": items, "mode": "local-persisted"})
	}))
	mux.HandleFunc("POST /api/v1/issues", guard(func(w http.ResponseWriter, r *http.Request) {
		var event struct {
			Service  string `json:"service"`
			Release  string `json:"release"`
			Type     string `json:"type"`
			Location string `json:"location"`
		}
		if !decodeIssue(w, r, &event) {
			return
		}
		for _, v := range []string{event.Service, event.Release, event.Type, event.Location} {
			if !safeIssueField.MatchString(v) || strings.Contains(v, "://") {
				writeJSON(w, 400, map[string]string{"error": "use bounded identifiers, not messages, URLs, or personal data"})
				return
			}
		}
		store.mu.Lock()
		defer store.mu.Unlock()
		now := time.Now().UTC()
		if now.Sub(store.window) >= time.Minute {
			store.window = now
			store.received = 0
		}
		if store.received >= 60 {
			w.Header().Set("Retry-After", "60")
			writeJSON(w, 429, map[string]string{"error": "ingestion limit reached"})
			return
		}
		store.received++
		sum := sha256.Sum256([]byte(event.Service + "\x00" + event.Type + "\x00" + event.Location))
		id := hex.EncodeToString(sum[:])
		entry, exists := store.items[id]
		if !exists && len(store.items) >= 1000 {
			writeJSON(w, 507, map[string]string{"error": "issue capacity reached"})
			return
		}
		if !exists {
			entry = issue{ID: id, Service: event.Service, Type: event.Type, Location: event.Location, FirstSeen: now, Status: "Open"}
		}
		if entry.Status == "Resolved" {
			entry.Status = "Regressed"
		}
		entry.Count++
		entry.Version++
		entry.LastSeen = now
		entry.Release = event.Release
		next := store.copy()
		next[id] = entry
		if store.save(next) != nil {
			writeJSON(w, 503, map[string]string{"error": "issue could not be persisted"})
			return
		}
		writeJSON(w, 201, entry)
	}))
	mux.HandleFunc("POST /api/v1/issues/{id}/resolve", guard(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Version int `json:"version"`
		}
		if !decodeIssue(w, r, &request) {
			return
		}
		store.mu.Lock()
		defer store.mu.Unlock()
		entry, ok := store.items[r.PathValue("id")]
		if !ok {
			writeJSON(w, 404, map[string]string{"error": "issue not found"})
			return
		}
		if entry.Version != request.Version {
			writeJSON(w, 409, map[string]string{"error": "issue changed; refresh before resolving"})
			return
		}
		entry.Status = "Resolved"
		entry.Version++
		next := store.copy()
		next[entry.ID] = entry
		if store.save(next) != nil {
			writeJSON(w, 503, map[string]string{"error": "resolution could not be persisted"})
			return
		}
		writeJSON(w, 200, entry)
	}))
}
