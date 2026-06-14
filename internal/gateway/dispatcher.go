package gateway

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// DefaultConfigPath is where the gateway YAML lives relative to the app
// data dir. We resolve to <data>/gateway.yaml at construction.
const DefaultConfigPath = "gateway.yaml"

const (
	defaultTimeout   = 5 * time.Second
	defaultRetry     = 2
	statusRingSize   = 32
)

// Dispatcher loads Hook subscriptions from disk and delivers wa: events
// to external HTTP endpoints. It is safe for concurrent use.
type Dispatcher struct {
	path string
	log  *slog.Logger

	mu     sync.RWMutex
	hooks  []Hook
	status map[string][]deliveryResult

	hc *http.Client
}

// NewDispatcher loads the gateway config from path. If the file is missing
// it creates an empty one and returns a Dispatcher with no hooks.
func NewDispatcher(path string, log *slog.Logger) (*Dispatcher, error) {
	if log == nil {
		log = slog.Default()
	}
	d := &Dispatcher{
		path:   path,
		log:    log,
		hooks:  []Hook{},
		status: map[string][]deliveryResult{},
		hc:     &http.Client{Timeout: defaultTimeout},
	}
	if err := d.load(); err != nil {
		return nil, err
	}
	return d, nil
}

func (d *Dispatcher) load() error {
	raw, err := os.ReadFile(d.path)
	if err != nil {
		if os.IsNotExist(err) {
			return d.save() // create empty file
		}
		return err
	}
	var cfg Config
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return fmt.Errorf("parse %s: %w", d.path, err)
	}
	if cfg.Hooks == nil {
		cfg.Hooks = []Hook{}
	}
	d.mu.Lock()
	d.hooks = cfg.Hooks
	d.mu.Unlock()
	return nil
}

func (d *Dispatcher) save() error {
	d.mu.RLock()
	cfg := Config{Hooks: d.hooks, UpdatedAt: time.Now()}
	d.mu.RUnlock()
	if err := os.MkdirAll(filepath.Dir(d.path), 0o755); err != nil {
		return err
	}
	raw, err := yaml.Marshal(&cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(d.path, raw, 0o600)
}

// List returns a copy of the current hooks.
func (d *Dispatcher) List() []Hook {
	d.mu.RLock()
	defer d.mu.RUnlock()
	out := make([]Hook, len(d.hooks))
	copy(out, d.hooks)
	return out
}

// Get returns a hook by ID and whether it was found.
func (d *Dispatcher) Get(id string) (Hook, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	for _, h := range d.hooks {
		if h.ID == id {
			return h, true
		}
	}
	return Hook{}, false
}

// Add appends a hook, assigns a new ID if empty, and persists.
func (d *Dispatcher) Add(h Hook) (Hook, error) {
	if h.URL == "" {
		return Hook{}, fmt.Errorf("url is required")
	}
	if h.Event == "" {
		return Hook{}, fmt.Errorf("event is required")
	}
	d.mu.Lock()
	if h.ID == "" {
		h.ID = newID()
	}
	if h.CreatedAt.IsZero() {
		h.CreatedAt = time.Now()
	}
	if h.Timeout == 0 {
		h.Timeout = defaultTimeout
	}
	if h.Retry == 0 {
		h.Retry = defaultRetry
	}
	h.Enabled = true
	d.hooks = append(d.hooks, h)
	d.mu.Unlock()
	return h, d.save()
}

// Update replaces a hook by ID, preserves CreatedAt, and persists.
func (d *Dispatcher) Update(id string, patch Hook) (Hook, error) {
	d.mu.Lock()
	var (
		found  bool
		before Hook
	)
	for i := range d.hooks {
		if d.hooks[i].ID == id {
			found = true
			before = d.hooks[i]
			patch.ID = id
			if patch.CreatedAt.IsZero() {
				patch.CreatedAt = before.CreatedAt
			}
			if patch.Timeout == 0 {
				patch.Timeout = before.Timeout
			}
			if patch.Retry == 0 {
				patch.Retry = before.Retry
			}
			d.hooks[i] = patch
			break
		}
	}
	d.mu.Unlock()
	if !found {
		return Hook{}, fmt.Errorf("hook %q not found", id)
	}
	return patch, d.save()
}

// Remove deletes a hook by ID and persists.
func (d *Dispatcher) Remove(id string) error {
	d.mu.Lock()
	found := false
	out := d.hooks[:0]
	for _, h := range d.hooks {
		if h.ID == id {
			found = true
			continue
		}
		out = append(out, h)
	}
	d.hooks = out
	d.mu.Unlock()
	if !found {
		return fmt.Errorf("hook %q not found", id)
	}
	return d.save()
}

// Status returns the last N delivery results for a hook.
func (d *Dispatcher) Status(id string) []deliveryResult {
	d.mu.RLock()
	defer d.mu.RUnlock()
	src := d.status[id]
	out := make([]deliveryResult, len(src))
	copy(out, src)
	return out
}

// Dispatch is the entry point called by the wa.Manager emit fan-out. It
// runs asynchronously so a slow webhook target never blocks the UI.
func (d *Dispatcher) Dispatch(event string, payload interface{}) {
	hooks := d.match(event, payload)
	if len(hooks) == 0 {
		return
	}
	body, err := json.Marshal(map[string]interface{}{
		"event":     event,
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		"data":      payload,
	})
	if err != nil {
		d.log.Warn("webhook marshal failed", "event", event, "err", err)
		return
	}
	for _, h := range hooks {
		go d.deliver(h, event, body)
	}
}

func (d *Dispatcher) match(event string, payload interface{}) []Hook {
	d.mu.RLock()
	defer d.mu.RUnlock()
	out := []Hook{}
	for _, h := range d.hooks {
		if !h.Enabled {
			continue
		}
		if h.Event != event {
			continue
		}
		if h.ChatFilter != "" {
			if !payloadMatchesChat(payload, h.ChatFilter) {
				continue
			}
		}
		out = append(out, h)
	}
	return out
}

// payloadMatchesChat inspects the event payload for a "chat" or "chat_jid"
// or "from" string field and returns true if it contains the filter
// substring. This is intentionally simple — gateway.yaml is human-edited
// and the filter language is "contains" not regex.
func payloadMatchesChat(payload interface{}, filter string) bool {
	m, ok := payload.(map[string]interface{})
	if !ok {
		return false
	}
	for _, key := range []string{"chat", "chat_jid", "chatJid", "from", "jid"} {
		v, ok := m[key]
		if !ok {
			continue
		}
		s, ok := v.(string)
		if !ok {
			continue
		}
		if contains(s, filter) {
			return true
		}
	}
	return false
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	// stdlib has strings.Contains but avoid the import for clarity
	if len(sub) == 0 {
		return 0
	}
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func (d *Dispatcher) deliver(h Hook, event string, body []byte) {
	timeout := h.Timeout
	if timeout == 0 {
		timeout = defaultTimeout
	}
	attempts := h.Retry + 1
	if attempts < 1 {
		attempts = 1
	}
	var (
		res  deliveryResult
		resp *http.Response
		err  error
	)
	for i := 0; i < attempts; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		req, rerr := http.NewRequestWithContext(ctx, http.MethodPost, h.URL, bytes.NewReader(body))
		if rerr != nil {
			cancel()
			res = deliveryResult{At: time.Now(), OK: false, Error: rerr.Error(), Attempts: i + 1}
			break
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Wakupi-Event", event)
		req.Header.Set("X-Wakupi-Hook", h.ID)
		for k, v := range h.Headers {
			req.Header.Set(k, v)
		}
		resp, err = d.hc.Do(req)
		// Drain & close so the connection can be reused
		if resp != nil {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
		cancel()
		// Retry on transport error OR non-2xx response
		if err == nil && resp.StatusCode < 400 {
			break
		}
		if i < attempts-1 {
			time.Sleep(backoff(i))
		}
	}
	if err != nil {
		res = deliveryResult{At: time.Now(), OK: false, Error: err.Error(), Attempts: attempts}
	} else {
		res = deliveryResult{At: time.Now(), OK: resp.StatusCode < 400, Status: resp.StatusCode, Attempts: attempts}
	}
	d.recordResult(h.ID, res)
	d.log.Info("webhook delivered",
		"hook", h.ID, "event", event, "url", h.URL,
		"ok", res.OK, "status", res.Status, "attempts", res.Attempts, "err", res.Error)
}

func (d *Dispatcher) recordResult(hookID string, r deliveryResult) {
	d.mu.Lock()
	defer d.mu.Unlock()
	ring := d.status[hookID]
	ring = append(ring, r)
	if len(ring) > statusRingSize {
		ring = ring[len(ring)-statusRingSize:]
	}
	d.status[hookID] = ring
}

func backoff(attempt int) time.Duration {
	d := time.Duration(1<<attempt) * 100 * time.Millisecond
	if d > 5*time.Second {
		d = 5 * time.Second
	}
	return d
}

func newID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return "hook_" + hex.EncodeToString(b[:])
}
