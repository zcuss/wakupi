package gateway

import (
	"testing"
)

func TestInboxPushAndPoll(t *testing.T) {
	in := NewInbox(10)
	in.Push(InboxMessage{Chat: "628@a", Text: "halo", MessageID: "m1"})
	in.Push(InboxMessage{Chat: "628@a", Text: "apa", MessageID: "m2"})
	in.Push(InboxMessage{Chat: "999@b", Text: "spam", MessageID: "m3"})

	got := in.Poll("", nil)
	if len(got) != 3 {
		t.Fatalf("expected 3, got %d", len(got))
	}
}

func TestInboxPollWithChatFilter(t *testing.T) {
	in := NewInbox(10)
	in.Push(InboxMessage{Chat: "628@a", MessageID: "m1"})
	in.Push(InboxMessage{Chat: "999@b", MessageID: "m2"})
	got := in.Poll("628@a", nil)
	if len(got) != 1 || got[0].MessageID != "m1" {
		t.Fatalf("filter failed: %+v", got)
	}
}

func TestInboxAckRemoves(t *testing.T) {
	in := NewInbox(10)
	in.Push(InboxMessage{Chat: "x", MessageID: "m1"})
	in.Push(InboxMessage{Chat: "x", MessageID: "m2"})

	// Poll with ack=[m1]: m1 is marked acked before scan, so only m2 returns
	got := in.Poll("", []string{"m1"})
	if len(got) != 1 || got[0].MessageID != "m2" {
		t.Fatalf("expected only m2 after ack m1, got %+v", got)
	}
	// Subsequent poll with no ack: m1 still gone, m2 still unacked → returns m2
	got = in.Poll("", nil)
	if len(got) != 1 || got[0].MessageID != "m2" {
		t.Fatalf("expected m2 still unacked, got %+v", got)
	}
	// Ack m2
	got = in.Poll("", []string{"m2"})
	if len(got) != 0 {
		t.Fatalf("expected empty after ack m2, got %d", len(got))
	}
	// Both acked
	got = in.Poll("", nil)
	if len(got) != 0 {
		t.Fatalf("expected 0 after both acked, got %d", len(got))
	}
}

func TestInboxEvictsOldest(t *testing.T) {
	in := NewInbox(3)
	for i := 0; i < 5; i++ {
		in.Push(InboxMessage{MessageID: "m" + string(rune('0'+i))})
	}
	if got := in.Size(); got != 3 {
		t.Fatalf("expected size 3, got %d", got)
	}
}
