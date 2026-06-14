package api

import (
	"context"
	"encoding/json"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
)

// Event is a single event broadcast to WebSocket subscribers. Name mirrors the
// Wails event names already emitted by wa.Manager (e.g. "wa:message").
type Event struct {
	Name string        `json:"name"`
	Data []interface{} `json:"data,omitempty"`
}

const (
	clientBufferSize = 256
	pingInterval     = 20 * time.Second
	writeTimeout     = 10 * time.Second
)

type client struct {
	conn    *websocket.Conn
	send    chan Event
	dropped atomic.Uint64
}

// Hub fans out events to all connected WebSocket clients with bounded
// per-client buffers (slow clients drop oldest events rather than blocking).
type Hub struct {
	mu      sync.RWMutex
	clients map[*client]struct{}
}

// NewHub creates an empty hub.
func NewHub() *Hub {
	return &Hub{clients: make(map[*client]struct{})}
}

// Broadcast delivers ev to every connected client. Non-blocking: if a client's
// buffer is full the event is dropped for that client and a counter increments.
func (h *Hub) Broadcast(ev Event) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients {
		select {
		case c.send <- ev:
		default:
			c.dropped.Add(1)
		}
	}
}

// Serve registers a freshly-upgraded connection and blocks until it closes,
// streaming events and running a heartbeat. Caller must have authenticated.
func (h *Hub) Serve(ctx context.Context, conn *websocket.Conn) {
	c := &client{
		conn: conn,
		send: make(chan Event, clientBufferSize),
	}
	h.mu.Lock()
	h.clients[c] = struct{}{}
	h.mu.Unlock()

	defer func() {
		h.mu.Lock()
		delete(h.clients, c)
		h.mu.Unlock()
		_ = conn.Close(websocket.StatusNormalClosure, "bye")
	}()

	// Greet with a connection.state event so clients know they're live.
	_ = writeEvent(ctx, conn, Event{Name: "connection.state", Data: []interface{}{
		map[string]interface{}{"connected": true},
	}})

	// Reader: drain incoming frames (and detect close) so pings/pongs work.
	readDone := make(chan struct{})
	go func() {
		defer close(readDone)
		for {
			if _, _, err := conn.Read(ctx); err != nil {
				return
			}
		}
	}()

	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-readDone:
			return
		case ev := <-c.send:
			if err := writeEvent(ctx, conn, ev); err != nil {
				return
			}
		case <-ticker.C:
			pctx, cancel := context.WithTimeout(ctx, writeTimeout)
			err := conn.Ping(pctx)
			cancel()
			if err != nil {
				return
			}
		}
	}
}

func writeEvent(ctx context.Context, conn *websocket.Conn, ev Event) error {
	payload, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	wctx, cancel := context.WithTimeout(ctx, writeTimeout)
	defer cancel()
	return conn.Write(wctx, websocket.MessageText, payload)
}

// Count returns the number of connected clients (for diagnostics).
func (h *Hub) Count() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}
