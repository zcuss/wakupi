package gateway

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"
)

func newTestDispatcher(t *testing.T) (*Dispatcher, string) {
	t.Helper()
	path := t.TempDir() + "/gateway.yaml"
	d, err := NewDispatcher(path, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("NewDispatcher: %v", err)
	}
	return d, path
}

func TestLoadCreatesEmptyFileIfMissing(t *testing.T) {
	path := t.TempDir() + "/g.yaml"
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected path missing, got %v", err)
	}
	d, err := NewDispatcher(path, nil)
	if err != nil {
		t.Fatalf("NewDispatcher: %v", err)
	}
	if got := len(d.List()); got != 0 {
		t.Fatalf("expected 0 hooks, got %d", got)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected file created, got %v", err)
	}
}

func TestAddAndListHook(t *testing.T) {
	d, _ := newTestDispatcher(t)
	h, err := d.Add(Hook{Event: "wa:message", URL: "http://x", ChatFilter: "628"})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if h.ID == "" {
		t.Fatal("expected ID assigned")
	}
	if !h.Enabled {
		t.Fatal("new hook should default to enabled")
	}
	if h.CreatedAt.IsZero() {
		t.Fatal("expected CreatedAt set")
	}
	if got := len(d.List()); got != 1 {
		t.Fatalf("expected 1 hook, got %d", got)
	}
}

func TestAddRejectsInvalid(t *testing.T) {
	d, _ := newTestDispatcher(t)
	if _, err := d.Add(Hook{URL: "http://x"}); err == nil {
		t.Fatal("expected error when event is empty")
	}
	if _, err := d.Add(Hook{Event: "wa:message"}); err == nil {
		t.Fatal("expected error when URL is empty")
	}
}

func TestRemoveHook(t *testing.T) {
	d, _ := newTestDispatcher(t)
	h, _ := d.Add(Hook{Event: "wa:message", URL: "http://x"})
	if err := d.Remove(h.ID); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if got := len(d.List()); got != 0 {
		t.Fatalf("expected 0 after remove, got %d", got)
	}
	if err := d.Remove("nonexistent"); err == nil {
		t.Fatal("expected error removing unknown id")
	}
}

func TestUpdatePreservesCreatedAt(t *testing.T) {
	d, _ := newTestDispatcher(t)
	h, _ := d.Add(Hook{Event: "wa:message", URL: "http://x"})
	orig := h.CreatedAt
	updated, err := d.Update(h.ID, Hook{Event: "wa:chat", URL: "http://y", Enabled: true})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if !updated.CreatedAt.Equal(orig) {
		t.Fatalf("CreatedAt changed: was %v, now %v", orig, updated.CreatedAt)
	}
}

func TestDispatchFiresMatchingHook(t *testing.T) {
	d, _ := newTestDispatcher(t)
	var got int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&got, 1)
		if r.Header.Get("X-Wakupi-Event") != "wa:message" {
			t.Errorf("expected X-Wakupi-Event=wa:message, got %q", r.Header.Get("X-Wakupi-Event"))
		}
		body, _ := io.ReadAll(r.Body)
		var p map[string]interface{}
		if err := json.Unmarshal(body, &p); err != nil {
			t.Errorf("bad json: %v", err)
		}
		if p["event"] != "wa:message" {
			t.Errorf("expected event=wa:message, got %v", p["event"])
		}
		w.WriteHeader(200)
	}))
	defer srv.Close()

	if _, err := d.Add(Hook{Event: "wa:message", URL: srv.URL}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	d.Dispatch("wa:message", map[string]interface{}{"chat": "628x@s.whatsapp.net", "text": "hi"})

	// async delivery — wait briefly
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if atomic.LoadInt32(&got) >= 1 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if atomic.LoadInt32(&got) != 1 {
		t.Fatalf("expected 1 delivery, got %d", got)
	}
}

func TestDispatchSkipsNonMatching(t *testing.T) {
	d, _ := newTestDispatcher(t)
	var got int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&got, 1)
	}))
	defer srv.Close()

	d.Add(Hook{Event: "wa:message", URL: srv.URL, ChatFilter: "628"})
	d.Dispatch("wa:message", map[string]interface{}{"chat": "999@x"})
	d.Dispatch("wa:chat", map[string]interface{}{"chat": "628@x"}) // wrong event

	time.Sleep(100 * time.Millisecond)
	if atomic.LoadInt32(&got) != 0 {
		t.Fatalf("expected 0 deliveries, got %d", got)
	}
}

func TestDispatchSkipsDisabled(t *testing.T) {
	d, _ := newTestDispatcher(t)
	var got int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&got, 1)
	}))
	defer srv.Close()

	h, _ := d.Add(Hook{Event: "wa:message", URL: srv.URL})
	d.Update(h.ID, Hook{Event: h.Event, URL: h.URL, Enabled: false})
	d.Dispatch("wa:message", nil)
	time.Sleep(100 * time.Millisecond)
	if atomic.LoadInt32(&got) != 0 {
		t.Fatalf("disabled hook still fired")
	}
}

func TestRetriesOnFailure(t *testing.T) {
	d, _ := newTestDispatcher(t)
	var got int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&got, 1)
		w.WriteHeader(500) // always fail
	}))
	defer srv.Close()

	d.Add(Hook{Event: "wa:message", URL: srv.URL, Retry: 2, Timeout: 200 * time.Millisecond})
	d.Dispatch("wa:message", nil)

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if atomic.LoadInt32(&got) >= 3 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if atomic.LoadInt32(&got) != 3 {
		t.Fatalf("expected 3 attempts (retry=2 + initial), got %d", got)
	}
	// status should reflect failure
	h, _ := d.List()[0], true
	_ = h
	status := d.Status(d.List()[0].ID)
	if len(status) == 0 || status[0].OK {
		t.Fatalf("expected status entry marked !OK, got %+v", status)
	}
}

func TestPersistenceReload(t *testing.T) {
	path := t.TempDir() + "/g.yaml"
	d1, _ := NewDispatcher(path, nil)
	d1.Add(Hook{Event: "wa:message", URL: "http://x"})
	// New dispatcher reading same file should see the hook
	d2, err := NewDispatcher(path, nil)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if got := len(d2.List()); got != 1 {
		t.Fatalf("expected 1 hook after reload, got %d", got)
	}
}
