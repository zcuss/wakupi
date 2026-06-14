package gateway

import (
	"sync"
	"time"
)

// Hook is a single webhook subscription: when an event matching Event and
// (optionally) ChatFilter fires, the dispatcher POSTs the event JSON to URL
// with optional Headers and a per-attempt Timeout. Retry controls how many
// additional times we re-POST on failure.
type Hook struct {
	ID         string            `yaml:"id"           json:"id"`
	Event      string            `yaml:"event"        json:"event"`     // e.g. "wa:message"
	URL        string            `yaml:"url"          json:"url"`
	ChatFilter string            `yaml:"chat_filter"  json:"chat_filter,omitempty"`
	Headers    map[string]string `yaml:"headers"      json:"headers,omitempty"`
	Timeout    time.Duration     `yaml:"timeout"      json:"timeout"`         // 0 = default 5s
	Retry      int               `yaml:"retry"        json:"retry"`           // 0 = no retry, 2 = 3 total attempts
	Enabled    bool              `yaml:"enabled"      json:"enabled"`
	CreatedAt  time.Time         `yaml:"created_at"   json:"created_at"`
}

// Config is the on-disk gateway configuration. It is loaded from and
// persisted to ./data/gateway.yaml.
type Config struct {
	Hooks     []Hook `yaml:"hooks"      json:"hooks"`
	UpdatedAt time.Time `yaml:"updated_at" json:"updated_at"`
}

// dispatcherKey is the per-event subscription key.
type dispatcherKey struct{}

// store is an in-process map of hookID -> last delivery result, for the
// /v1/gateway/webhooks/<id>/status endpoint.
type deliveryResult struct {
	At       time.Time `json:"at"`
	OK       bool      `json:"ok"`
	Status   int       `json:"status"`
	Error    string    `json:"error,omitempty"`
	Attempts int       `json:"attempts"`
}

// webhooksState holds the live dispatcher state protected by a mutex.
type webhooksState struct {
	mu       sync.RWMutex
	hooks    []Hook
	status   map[string][]deliveryResult // hookID -> ring buffer of last N results
	dispatch *Dispatcher                 // back-pointer for re-loading config
}

func newWebhooksState() *webhooksState {
	return &webhooksState{
		hooks:  []Hook{},
		status: map[string][]deliveryResult{},
	}
}
