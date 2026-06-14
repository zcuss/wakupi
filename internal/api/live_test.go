package api

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/coder/websocket"
)

// pick a free localhost port
func freeAddr(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := l.Addr().String()
	_ = l.Close()
	return addr
}

func TestLiveStartStopAndWS(t *testing.T) {
	addr := freeAddr(t)
	hub := NewHub()
	srv := New(Config{Enabled: true, Addr: addr, Token: "live"}, &mockWA{}, hub, nil)
	errc := srv.Start()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = srv.Stop(ctx)
	}()

	// wait for listener
	base := "http://" + addr
	if !waitReady(base+"/v1/health", time.Second) {
		t.Fatal("server did not become ready")
	}

	select {
	case e := <-errc:
		if e != nil {
			t.Fatalf("server error: %v", e)
		}
	default:
	}

	// connect WS with token query param, expect connection.state greeting
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, "ws://"+addr+"/v1/events?token=live", nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "done")

	_, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("ws read: %v", err)
	}
	if !contains(string(data), "connection.state") {
		t.Fatalf("expected connection.state greeting, got %s", string(data))
	}

	// broadcast an event, expect it to arrive
	hub.Broadcast(Event{Name: "wa:message", Data: []interface{}{map[string]string{"text": "yo"}}})
	_, data2, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("ws read 2: %v", err)
	}
	if !contains(string(data2), "wa:message") {
		t.Fatalf("expected wa:message, got %s", string(data2))
	}
}

func TestWSUnauthorized(t *testing.T) {
	addr := freeAddr(t)
	srv := New(Config{Enabled: true, Addr: addr, Token: "live"}, &mockWA{}, NewHub(), nil)
	srv.Start()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = srv.Stop(ctx)
	}()
	if !waitReady("http://"+addr+"/v1/health", time.Second) {
		t.Fatal("not ready")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, _, err := websocket.Dial(ctx, "ws://"+addr+"/v1/events", nil)
	if err == nil {
		t.Fatal("expected WS dial to fail without token")
	}
}

func waitReady(url string, d time.Duration) bool {
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			_ = resp.Body.Close()
			return true
		}
		time.Sleep(20 * time.Millisecond)
	}
	return false
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
