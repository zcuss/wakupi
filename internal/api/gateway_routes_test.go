package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"wakupi/internal/gateway"
)

func newTestServerWithGateway(t *testing.T) (*Server, *gateway.Dispatcher) {
	t.Helper()
	gw, err := gateway.NewDispatcher(t.TempDir()+"/g.yaml", nil)
	if err != nil {
		t.Fatalf("gateway: %v", err)
	}
	srv := New(Config{
		Enabled: true,
		Addr:    "127.0.0.1:0",
		Token:   "secret",
	}, &mockWA{}, NewHub(), gw, nil, nil)
	return srv, gw
}

func newTestServerWithAll(t *testing.T) (*Server, *gateway.Dispatcher, *gateway.Scheduler, *gateway.Inbox) {
	t.Helper()
	gw, _ := gateway.NewDispatcher(t.TempDir()+"/g.yaml", nil)
	sched := gateway.NewScheduler(50*time.Millisecond, func(action string, args map[string]string) error { return nil })
	inbox := gateway.NewInbox(16)
	srv := New(Config{
		Enabled: true, Addr: "127.0.0.1:0", Token: "secret",
	}, &mockWA{}, NewHub(), gw, sched, inbox)
	return srv, gw, sched, inbox
}

func authReq(r *http.Request) *http.Request {
	r.Header.Set("Authorization", "Bearer secret")
	return r
}

func TestGatewayRoutesCRUD(t *testing.T) {
	srv, _ := newTestServerWithGateway(t)

	// Empty list
	rr := httptest.NewRecorder()
	srv.router.ServeHTTP(rr, authReq(httptest.NewRequest(http.MethodGet, "/v1/gateway/webhooks", nil)))
	if rr.Code != 200 {
		t.Fatalf("GET list: %d %s", rr.Code, rr.Body.String())
	}
	var resp struct {
		OK   bool          `json:"ok"`
		Data []gateway.Hook `json:"data"`
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	if !resp.OK || len(resp.Data) != 0 {
		t.Fatalf("expected empty list, got %+v", resp)
	}

	// Create
	body := `{"event":"wa:message","url":"http://n8n.local/webhook/x","chat_filter":"628","timeout":1000000000,"retry":1}`
	rr = httptest.NewRecorder()
	req := authReq(httptest.NewRequest(http.MethodPost, "/v1/gateway/webhooks", strings.NewReader(body)))
	req.Header.Set("Content-Type", "application/json")
	srv.router.ServeHTTP(rr, req)
	if rr.Code != 201 {
		t.Fatalf("POST create: %d %s", rr.Code, rr.Body.String())
	}
	var created struct {
		OK   bool        `json:"ok"`
		Data gateway.Hook `json:"data"`
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &created)
	if created.Data.ID == "" {
		t.Fatal("expected ID assigned")
	}
	if !created.Data.Enabled {
		t.Fatal("expected Enabled=true on new hook")
	}
	id := created.Data.ID

	// Get
	rr = httptest.NewRecorder()
	srv.router.ServeHTTP(rr, authReq(httptest.NewRequest(http.MethodGet, "/v1/gateway/webhooks/"+id, nil)))
	if rr.Code != 200 {
		t.Fatalf("GET one: %d", rr.Code)
	}

	// Update
	upd := `{"event":"wa:message","url":"http://n8n.local/webhook/y","enabled":false}`
	rr = httptest.NewRecorder()
	req = authReq(httptest.NewRequest(http.MethodPut, "/v1/gateway/webhooks/"+id, strings.NewReader(upd)))
	req.Header.Set("Content-Type", "application/json")
	srv.router.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("PUT update: %d %s", rr.Code, rr.Body.String())
	}

	// Delete
	rr = httptest.NewRecorder()
	srv.router.ServeHTTP(rr, authReq(httptest.NewRequest(http.MethodDelete, "/v1/gateway/webhooks/"+id, nil)))
	if rr.Code != 200 {
		t.Fatalf("DELETE: %d", rr.Code)
	}

	// Confirm 404 after delete
	rr = httptest.NewRecorder()
	srv.router.ServeHTTP(rr, authReq(httptest.NewRequest(http.MethodGet, "/v1/gateway/webhooks/"+id, nil)))
	if rr.Code != 404 {
		t.Fatalf("expected 404 after delete, got %d", rr.Code)
	}
}

func TestGatewayRoutesRequireAuth(t *testing.T) {
	srv, _ := newTestServerWithGateway(t)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/gateway/webhooks", nil)
	// no Authorization header
	srv.router.ServeHTTP(rr, req)
	if rr.Code != 401 {
		t.Fatalf("expected 401 without auth, got %d", rr.Code)
	}
}

func TestGatewayEventsList(t *testing.T) {
	srv, _ := newTestServerWithGateway(t)
	rr := httptest.NewRecorder()
	srv.router.ServeHTTP(rr, authReq(httptest.NewRequest(http.MethodGet, "/v1/gateway/events", nil)))
	if rr.Code != 200 {
		t.Fatalf("GET events: %d", rr.Code)
	}
	var resp struct {
		OK   bool     `json:"ok"`
		Data []string `json:"data"`
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	if !resp.OK || len(resp.Data) < 5 {
		t.Fatalf("expected several events, got %+v", resp)
	}
	if resp.Data[0] != "wa:message" {
		t.Fatalf("expected first event wa:message, got %q", resp.Data[0])
	}
}

func TestGatewayTestEndpointDispatches(t *testing.T) {
	srv, gw := newTestServerWithGateway(t)
	h, _ := gw.Add(gateway.Hook{Event: "wa:message", URL: "http://example.invalid/x"})

	rr := httptest.NewRecorder()
	srv.router.ServeHTTP(rr, authReq(httptest.NewRequest(http.MethodPost, "/v1/gateway/webhooks/"+h.ID+"/test", nil)))
	if rr.Code != 202 {
		t.Fatalf("test endpoint: %d %s", rr.Code, rr.Body.String())
	}
	// give the async dispatch a moment to record failure
	// (http://example.invalid won't resolve)
	deadline := waitForStatus(gw, h.ID, 1, 3.0)
	if !deadline {
		t.Fatal("expected at least 1 status entry after test dispatch")
	}
}

// waitForStatus polls for at least n recorded deliveries for the given hook.
func waitForStatus(gw *gateway.Dispatcher, id string, n int, seconds float64) bool {
	deadline := time.Now().Add(time.Duration(seconds * float64(time.Second)))
	for time.Now().Before(deadline) {
		if len(gw.Status(id)) >= n {
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return len(gw.Status(id)) >= n
}
