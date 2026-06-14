package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"wakupi/internal/gateway"
)

func TestCronRoutesCRUD(t *testing.T) {
	srv, _, sched, _ := newTestServerWithAll(t)
	defer sched.Stop()

	// Empty list
	rr := httptest.NewRecorder()
	srv.router.ServeHTTP(rr, authReq(httptest.NewRequest(http.MethodGet, "/v1/gateway/cron", nil)))
	if rr.Code != 200 {
		t.Fatalf("list: %d", rr.Code)
	}

	// Create
	body := `{"name":"morning ping","cron":"@every 1h","action":"send_message","args":{"session":"default","jid":"628","text":"hi"}}`
	rr = httptest.NewRecorder()
	req := authReq(httptest.NewRequest(http.MethodPost, "/v1/gateway/cron", strings.NewReader(body)))
	req.Header.Set("Content-Type", "application/json")
	srv.router.ServeHTTP(rr, req)
	if rr.Code != 201 {
		t.Fatalf("create: %d %s", rr.Code, rr.Body.String())
	}
	var created struct {
		OK   bool           `json:"ok"`
		Data gateway.CronJob `json:"data"`
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &created)
	if created.Data.ID == "" {
		t.Fatal("expected ID")
	}

	// List shows it
	rr = httptest.NewRecorder()
	srv.router.ServeHTTP(rr, authReq(httptest.NewRequest(http.MethodGet, "/v1/gateway/cron", nil)))
	var list struct {
		OK   bool            `json:"ok"`
		Data []gateway.CronJob `json:"data"`
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &list)
	if len(list.Data) != 1 {
		t.Fatalf("expected 1, got %d", len(list.Data))
	}

	// Delete
	rr = httptest.NewRecorder()
	srv.router.ServeHTTP(rr, authReq(httptest.NewRequest(http.MethodDelete, "/v1/gateway/cron/"+created.Data.ID, nil)))
	if rr.Code != 200 {
		t.Fatalf("delete: %d", rr.Code)
	}
}

func TestCronRouteRejectsBadSpec(t *testing.T) {
	srv, _, sched, _ := newTestServerWithAll(t)
	defer sched.Stop()
	body := `{"cron":"garbage","action":"send_message"}`
	rr := httptest.NewRecorder()
	req := authReq(httptest.NewRequest(http.MethodPost, "/v1/gateway/cron", strings.NewReader(body)))
	req.Header.Set("Content-Type", "application/json")
	srv.router.ServeHTTP(rr, req)
	if rr.Code != 400 {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestInboxRoutes(t *testing.T) {
	srv, _, _, inbox := newTestServerWithAll(t)

	// Empty size
	rr := httptest.NewRecorder()
	srv.router.ServeHTTP(rr, authReq(httptest.NewRequest(http.MethodGet, "/v1/gateway/inbox/size", nil)))
	if rr.Code != 200 {
		t.Fatalf("size: %d", rr.Code)
	}

	// Push some test data
	inbox.Push(gateway.InboxMessage{Chat: "628@x", Text: "halo", MessageID: "m1"})
	inbox.Push(gateway.InboxMessage{Chat: "628@x", Text: "apa", MessageID: "m2"})
	inbox.Push(gateway.InboxMessage{Chat: "999@y", Text: "spam", MessageID: "m3"})

	// Poll all
	rr = httptest.NewRecorder()
	srv.router.ServeHTTP(rr, authReq(httptest.NewRequest(http.MethodGet, "/v1/gateway/inbox", nil)))
	var poll struct {
		OK   bool                `json:"ok"`
		Data []gateway.InboxMessage `json:"data"`
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &poll)
	if len(poll.Data) != 3 {
		t.Fatalf("expected 3, got %d", len(poll.Data))
	}

	// Poll with chat filter
	rr = httptest.NewRecorder()
	srv.router.ServeHTTP(rr, authReq(httptest.NewRequest(http.MethodGet, "/v1/gateway/inbox?chat=628@x", nil)))
	_ = json.Unmarshal(rr.Body.Bytes(), &poll)
	if len(poll.Data) != 2 {
		t.Fatalf("expected 2, got %d", len(poll.Data))
	}

	// Ack m1
	ack := `{"ids":["m1"]}`
	rr = httptest.NewRecorder()
	req := authReq(httptest.NewRequest(http.MethodPost, "/v1/gateway/inbox/ack", strings.NewReader(ack)))
	req.Header.Set("Content-Type", "application/json")
	srv.router.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("ack: %d", rr.Code)
	}

	// Now poll returns m2,m3
	rr = httptest.NewRecorder()
	srv.router.ServeHTTP(rr, authReq(httptest.NewRequest(http.MethodGet, "/v1/gateway/inbox", nil)))
	_ = json.Unmarshal(rr.Body.Bytes(), &poll)
	if len(poll.Data) != 2 {
		t.Fatalf("expected 2 after ack m1, got %d", len(poll.Data))
	}
}
