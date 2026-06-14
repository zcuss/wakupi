package gateway

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestNextFire_Every(t *testing.T) {
	now := time.Date(2026, 6, 15, 10, 7, 30, 0, time.UTC)
	got, ok := nextFire("@every 5m", now)
	if !ok {
		t.Fatal("expected ok")
	}
	want := time.Date(2026, 6, 15, 10, 10, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestNextFire_DailyFuture(t *testing.T) {
	now := time.Date(2026, 6, 15, 8, 0, 0, 0, time.UTC)
	got, ok := nextFire("@daily 09:00", now)
	if !ok {
		t.Fatal("expected ok")
	}
	if got.Hour() != 9 || got.Minute() != 0 {
		t.Fatalf("got %v", got)
	}
}

func TestNextFire_DailyPast(t *testing.T) {
	now := time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC)
	got, ok := nextFire("@daily 09:00", now)
	if !ok {
		t.Fatal("expected ok")
	}
	if got.Day() != now.Day()+1 {
		t.Fatalf("expected next day, got %v", got)
	}
}

func TestNextFire_Invalid(t *testing.T) {
	cases := []string{"", "every 5m", "@bogus foo", "@every notaduration", "@daily 25:00", "@daily foo"}
	for _, c := range cases {
		if _, ok := nextFire(c, time.Now()); ok {
			t.Fatalf("expected invalid: %q", c)
		}
	}
}

func TestSchedulerFires(t *testing.T) {
	var calls int32
	s := NewScheduler(50*time.Millisecond, func(action string, args map[string]string) error {
		atomic.AddInt32(&calls, 1)
		return nil
	})
	_, err := s.Add(CronJob{Cron: "@every 100ms", RunAction: "send_message", RunArgs: map[string]string{"to": "x"}})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	s.Start()
	defer s.Stop()

	// Wait for at least 2 fires
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if atomic.LoadInt32(&calls) >= 2 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if atomic.LoadInt32(&calls) < 2 {
		t.Fatalf("expected >=2 fires, got %d", calls)
	}
}

func TestSchedulerRejectsBadSpec(t *testing.T) {
	s := NewScheduler(50*time.Millisecond, nil)
	_, err := s.Add(CronJob{Cron: "garbage", RunAction: "x"})
	if err == nil {
		t.Fatal("expected error")
	}
	_, err = s.Add(CronJob{Cron: "@every 5m", RunAction: ""})
	if err == nil {
		t.Fatal("expected error on empty action")
	}
}

func TestSchedulerRecordsResult(t *testing.T) {
	s := NewScheduler(30*time.Millisecond, func(action string, args map[string]string) error {
		return nil
	})
	_, _ = s.Add(CronJob{Cron: "@every 50ms", RunAction: "ping"})
	s.Start()
	defer s.Stop()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		jobs := s.List()
		if jobs[0].LastResult == "ok" {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("expected LastResult=ok, got %q", s.List()[0].LastResult)
}
