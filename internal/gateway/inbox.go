package gateway

import (
	"sync"
	"time"
)

// InboxMessage is a small record of a received wa:message event, kept in
// memory for short-poll consumption by external clients. Older entries
// are evicted when the buffer exceeds MaxSize.
type InboxMessage struct {
	Event     string                 `json:"event"`
	Timestamp time.Time              `json:"timestamp"`
	Chat      string                 `json:"chat"`
	Sender    string                 `json:"sender,omitempty"`
	Text      string                 `json:"text,omitempty"`
	MessageID string                 `json:"message_id,omitempty"`
	FromMe    bool                   `json:"from_me"`
	Extra     map[string]interface{} `json:"extra,omitempty"`
}

// Inbox is a fixed-size ring buffer of recent incoming messages. External
// consumers can POST /v1/gateway/inbox/poll (or GET) to fetch unacked
// messages and ACK them so they don't reappear.
type Inbox struct {
	mu      sync.Mutex
	buf     []InboxMessage
	maxSize int
	acked   map[string]struct{} // messageIDs that have been acked
}

// NewInbox returns a buffer that holds up to maxSize messages.
func NewInbox(maxSize int) *Inbox {
	if maxSize <= 0 {
		maxSize = 256
	}
	return &Inbox{
		buf:     make([]InboxMessage, 0, maxSize),
		maxSize: maxSize,
		acked:   map[string]struct{}{},
	}
}

// Push appends a message and returns the stored record. Older entries are
// evicted when the buffer overflows.
func (i *Inbox) Push(m InboxMessage) InboxMessage {
	i.mu.Lock()
	defer i.mu.Unlock()
	if m.Timestamp.IsZero() {
		m.Timestamp = time.Now()
	}
	if m.Event == "" {
		m.Event = "wa:message"
	}
	i.buf = append(i.buf, m)
	if len(i.buf) > i.maxSize {
		// Drop from the front
		i.buf = i.buf[len(i.buf)-i.maxSize:]
	}
	return m
}

// Poll returns all unacked messages, optionally filtered by chat. ackIDs
// (if non-empty) marks the listed messageIDs as acked and they will not
// appear in future polls.
func (i *Inbox) Poll(chatFilter string, ackIDs []string) []InboxMessage {
	i.mu.Lock()
	defer i.mu.Unlock()
	for _, id := range ackIDs {
		if id != "" {
			i.acked[id] = struct{}{}
		}
	}
	out := []InboxMessage{}
	for _, m := range i.buf {
		if m.MessageID != "" {
			if _, ok := i.acked[m.MessageID]; ok {
				continue
			}
		}
		if chatFilter != "" && m.Chat != chatFilter {
			continue
		}
		out = append(out, m)
	}
	return out
}

// Size returns the current count.
func (i *Inbox) Size() int {
	i.mu.Lock()
	defer i.mu.Unlock()
	return len(i.buf)
}
