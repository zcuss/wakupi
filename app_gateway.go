package main

import (
	"fmt"
	"time"

	"wakupi/internal/gateway"
)

// captureInbox pulls the standard wa.MessageInfo fields out of an emit
// payload and pushes it into the gateway inbox. Best-effort: missing
// fields just stay empty.
func (a *App) captureInbox(payload interface{}) {
	m, ok := payload.(map[string]interface{})
	if !ok {
		return
	}
	rec := gateway.InboxMessage{
		Event:     "wa:message",
		Timestamp: time.Now(),
	}
	if v, ok := m["chat"].(string); ok {
		rec.Chat = v
	} else if v, ok := m["Chat"].(string); ok {
		rec.Chat = v
	}
	if v, ok := m["sender"].(string); ok {
		rec.Sender = v
	} else if v, ok := m["Sender"].(string); ok {
		rec.Sender = v
	}
	if v, ok := m["text"].(string); ok {
		rec.Text = v
	} else if v, ok := m["Text"].(string); ok {
		rec.Text = v
	}
	if v, ok := m["id"].(string); ok {
		rec.MessageID = v
	} else if v, ok := m["ID"].(string); ok {
		rec.MessageID = v
	}
	if v, ok := m["from_me"].(bool); ok {
		rec.FromMe = v
	} else if v, ok := m["FromMe"].(bool); ok {
		rec.FromMe = v
	}
	if len(m) > 0 {
		rec.Extra = m
	}
	a.inbox.Push(rec)
}

// runGatewayAction is the cron ActionFunc. Supported verbs:
//
//   "send_message"  args: session, jid, text
//   "mark_read"     args: session, jid, sender
//   "set_presence"  args: session, state ("available"|"unavailable")
//
// Anything else returns an error so the scheduler records it.
func (a *App) runGatewayAction(action string, args map[string]string) error {
	if a.wa == nil {
		return fmt.Errorf("wa manager not ready")
	}
	switch action {
	case "send_message":
		session := args["session"]
		jid := args["jid"]
		text := args["text"]
		if session == "" || jid == "" || text == "" {
			return fmt.Errorf("send_message requires session, jid, text")
		}
		_, err := a.wa.SendText(a.ctx, session, jid, text, nil)
		return err
	case "mark_read":
		session := args["session"]
		jid := args["jid"]
		sender := args["sender"]
		if session == "" || jid == "" {
			return fmt.Errorf("mark_read requires session, jid")
		}
		return a.wa.MarkRead(a.ctx, session, jid, sender, nil)
	case "send_presence":
		session := args["session"]
		state := args["state"]
		if session == "" {
			return fmt.Errorf("send_presence requires session")
		}
		// Chat presence state. The manager accepts a state string.
		return a.wa.SendChatPresence(a.ctx, session, "", state)
	}
	return fmt.Errorf("unknown action: %s", action)
}

// initScheduler wires the gateway scheduler. Called from startAPI so the
// ActionFunc closure can capture a.wa (which may not exist at struct-init
// time on first run).
func (a *App) initScheduler() {
	if a.scheduler != nil {
		return
	}
	a.scheduler = gateway.NewScheduler(30*time.Second, a.runGatewayAction)
	a.scheduler.Start()
}
