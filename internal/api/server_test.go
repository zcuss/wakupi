package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"wakupi/internal/wa"
)

// mockWA implements WAService for tests.
type mockWA struct {
	sentText string
	sentJID  string
}

func (m *mockWA) Sessions() []wa.SessionInfo {
	return []wa.SessionInfo{{ID: "628111", Name: "+628111", Connected: true, JID: "628111@s.whatsapp.net", Phone: "+628111"}}
}
func (m *mockWA) StartLogin(ctx context.Context, name string) (string, error) { return "new-1", nil }
func (m *mockWA) Logout(ctx context.Context, id string) error                 { return nil }
func (m *mockWA) LoadChats(ctx context.Context, acct string) ([]wa.ChatInfo, error) {
	return []wa.ChatInfo{{ID: "c1", JID: "628999@s.whatsapp.net", Name: "Budi"}}, nil
}
func (m *mockWA) LoadMessages(ctx context.Context, acct, jid string, limit int, before int64) ([]wa.MessageInfo, error) {
	return []wa.MessageInfo{{ID: "m1", Text: "hi", ChatID: jid}}, nil
}
func (m *mockWA) SendText(ctx context.Context, sid, jid, text string, q *wa.QuotedRef) (string, error) {
	m.sentText = text
	m.sentJID = jid
	return "msg-123", nil
}
func (m *mockWA) SendImage(ctx context.Context, sid, jid, path, cap string, q *wa.QuotedRef) (*wa.SendMediaResult, error) {
	return &wa.SendMediaResult{MessageID: "img-1"}, nil
}
func (m *mockWA) SendDocument(ctx context.Context, sid, jid, path string, q *wa.QuotedRef) (*wa.SendMediaResult, error) {
	return &wa.SendMediaResult{MessageID: "doc-1"}, nil
}
func (m *mockWA) MarkRead(ctx context.Context, sid, jid, sender string, ids []string) error { return nil }
func (m *mockWA) ReactMessage(ctx context.Context, sid, jid, mid, sender, emoji string) error {
	return nil
}

func newTestServer() (*Server, *mockWA) {
	mock := &mockWA{}
	srv := New(Config{Enabled: true, Addr: "127.0.0.1:0", Token: "secret"}, mock, NewHub())
	return srv, mock
}

func TestHealthNoAuth(t *testing.T) {
	srv, _ := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	rr := httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("health: want 200 got %d", rr.Code)
	}
	var body map[string]interface{}
	_ = json.Unmarshal(rr.Body.Bytes(), &body)
	if body["ok"] != true {
		t.Fatalf("health: ok!=true: %v", body)
	}
}

func TestAuthRequired(t *testing.T) {
	srv, _ := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/v1/sessions", nil)
	rr := httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("sessions w/o token: want 401 got %d", rr.Code)
	}
}

func TestAuthBearerAccepted(t *testing.T) {
	srv, _ := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/v1/sessions", nil)
	req.Header.Set("Authorization", "Bearer secret")
	rr := httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("sessions w/ token: want 200 got %d", rr.Code)
	}
}

func TestSendText(t *testing.T) {
	srv, mock := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/v1/chats/628999@s.whatsapp.net/messages",
		strings.NewReader(`{"text":"hello world"}`))
	req.Header.Set("Authorization", "Bearer secret")
	rr := httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("send: want 200 got %d body=%s", rr.Code, rr.Body.String())
	}
	if mock.sentText != "hello world" {
		t.Fatalf("send: text not passed through: %q", mock.sentText)
	}
	if mock.sentJID != "628999@s.whatsapp.net" {
		t.Fatalf("send: jid not passed through: %q", mock.sentJID)
	}
}

func TestSendEmptyRejected(t *testing.T) {
	srv, _ := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/v1/chats/x@s.whatsapp.net/messages",
		strings.NewReader(`{}`))
	req.Header.Set("Authorization", "Bearer secret")
	rr := httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("empty send: want 400 got %d", rr.Code)
	}
}

func TestListChats(t *testing.T) {
	srv, _ := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/v1/chats", nil)
	req.Header.Set("Authorization", "Bearer secret")
	rr := httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("chats: want 200 got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "Budi") {
		t.Fatalf("chats: expected chat in body: %s", rr.Body.String())
	}
}

func TestTokenQueryParam(t *testing.T) {
	srv, _ := newTestServer()
	// WS auth path also accepts ?token= ; verify checkToken honors it.
	req := httptest.NewRequest(http.MethodGet, "/v1/sessions?token=secret", nil)
	if !srv.checkToken(req) {
		t.Fatal("checkToken should accept query param token")
	}
	bad := httptest.NewRequest(http.MethodGet, "/v1/sessions?token=wrong", nil)
	if srv.checkToken(bad) {
		t.Fatal("checkToken should reject wrong token")
	}
}
