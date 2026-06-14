package desktop

import (
	"strings"
	"sync/atomic"
	"testing"
)

type fakeCtrl struct {
	opened    atomic.Int32
	closed    atomic.Int32
	lastOpen  atomic.Value
	lastClose atomic.Value
}

func (f *fakeCtrl) ListRunningApps() ([]AppInfo, error)        { return nil, nil }
func (f *fakeCtrl) CloseApp(name string) error                { f.closed.Add(1); f.lastClose.Store(name); return nil }
func (f *fakeCtrl) OpenApp(name string) error                 { f.opened.Add(1); f.lastOpen.Store(name); return nil }
func (f *fakeCtrl) MediaPlayPause() error                     { return nil }
func (f *fakeCtrl) MediaNext() error                         { return nil }
func (f *fakeCtrl) MediaPrev() error                         { return nil }
func (f *fakeCtrl) MediaNowPlaying() (*MediaInfo, error)     { return nil, nil }
func (f *fakeCtrl) GetVolume() (int, error)                   { return 0, nil }
func (f *fakeCtrl) SetVolume(pct int) error                   { return nil }
func (f *fakeCtrl) TakeScreenshot() (string, error)           { return "", nil }
func (f *fakeCtrl) LockScreen() error                         { return nil }

func TestDefaultPolicyRejectsAll(t *testing.T) {
	ctrl := &fakeCtrl{}
	p := DefaultPolicy()
	for _, cmd := range []string{"!open firefox", "!close notepad", "!lock", "!screenshot", "!volume 50", "!play", "!apps"} {
		out := HandleCommandWithPolicy(ctrl, cmd, "[email protected]", p)
		if !strings.Contains(out, "diblokir") && !strings.Contains(out, "Allowed") {
			t.Errorf("%s should be blocked by default, got %q", cmd, out)
		}
	}
	if ctrl.opened.Load() != 0 || ctrl.closed.Load() != 0 {
		t.Errorf("default policy must not invoke controller, got open=%d close=%d", ctrl.opened.Load(), ctrl.closed.Load())
	}
}

func TestAllowlistEnforced(t *testing.T) {
	ctrl := &fakeCtrl{}
	p := &SecurityPolicy{
		Enabled:         true,
		AllowedOpenApps: []string{"firefox", "code"},
		AllowClose:      true,
		AllowLock:       true,
		AllowScreenshot: true,
		AllowMedia:      true,
		AllowVolume:     true,
		RequireConfirm:  false,
		ConfirmTTL:      60_000_000_000, // 60s
		RateLimitPerMin: 100,
	}
	if out := HandleCommandWithPolicy(ctrl, "!open firefox", "x", p); !strings.Contains(out, "dibuka") {
		t.Errorf("!open firefox should succeed, got %q", out)
	}
	if out := HandleCommandWithPolicy(ctrl, "!open cmd", "x", p); !strings.Contains(out, "tidak di-allow") {
		t.Errorf("!open cmd should be rejected, got %q", out)
	}
	if last, _ := ctrl.lastOpen.Load().(string); last != "firefox" {
		t.Errorf("expected last open=firefox, got %q", last)
	}
}

func TestConfirmFlow(t *testing.T) {
	ctrl := &fakeCtrl{}
	p := &SecurityPolicy{
		Enabled:         true,
		AllowedOpenApps: []string{"firefox"},
		RequireConfirm:  true,
		ConfirmTTL:      60_000_000_000,
		RateLimitPerMin: 100,
	}
	out := HandleCommandWithPolicy(ctrl, "!open firefox", "x", p)
	if !strings.Contains(out, "!confirm ") {
		t.Fatalf("expected confirm prompt, got %q", out)
	}
	tokLine := out[strings.Index(out, "!confirm "):]
	tok := strings.Fields(tokLine)[1]
	if tok == "" {
		t.Fatalf("could not extract token from %q", out)
	}
	out2 := HandleCommandWithPolicy(ctrl, "!confirm "+tok, "x", p)
	if !strings.Contains(out2, "dibuka") {
		t.Errorf("confirm reply should run command, got %q", out2)
	}
	if ctrl.opened.Load() != 1 {
		t.Errorf("expected exactly one open, got %d", ctrl.opened.Load())
	}
}

func TestSenderAllowlist(t *testing.T) {
	ctrl := &fakeCtrl{}
	p := &SecurityPolicy{
		Enabled:         true,
		AllowedSenders:  []string{"admin-jid"},
		AllowedOpenApps: []string{"firefox"},
		RequireConfirm:  false,
		ConfirmTTL:      60_000_000_000,
		RateLimitPerMin: 100,
	}
	if out := HandleCommandWithPolicy(ctrl, "!open firefox", "admin-jid", p); !strings.Contains(out, "dibuka") {
		t.Errorf("admin should run, got %q", out)
	}
	if out := HandleCommandWithPolicy(ctrl, "!open firefox", "other-jid", p); !strings.Contains(out, "diblokir") {
		t.Errorf("non-admin should be blocked, got %q", out)
	}
}
