package desktop

import (
	"crypto/rand"
	"encoding/hex"
	"sort"
	"strings"
	"sync"
	"time"
)

// SecurityPolicy gates every command. When loaded, it restricts which
// commands run, who can run them, and what they can affect.
//
// Defaults (zero value) block every potentially destructive command.
// This is intentional — desktop control from a chat is a remote-code-
// execution surface and must be opted into explicitly via config.
//
// Wire in app.go: load a Policy from app settings (desktop_config),
// store it on the Controller implementation, and call
// HandleCommand(ctrl, text, sender, policy) instead of the old
// HandleCommand(ctrl, text).
type SecurityPolicy struct {
	// Enabled is the master switch. When false, every command is rejected.
	Enabled bool `json:"enabled"`

	// AllowedSenders is an optional JID/LID allowlist. Empty = allow all
	// (only meaningful when CombinedWith the master switch — i.e. only after
	// the user opts in via Enabled). When set, only matching senders can run
	// commands.
	AllowedSenders []string `json:"allowed_senders"`

	// AllowedOpenApps is the allowlist for !open. Empty = reject all.
	// Entries are matched case-insensitively against the trimmed app name
	// the user sends. The list is also used by !close — only apps in
	// AllowedOpenApps can be closed (don't allow closing apps you couldn't
	// open safely).
	AllowedOpenApps []string `json:"allowed_open_apps"`

	// AllowClose toggles !close. Even with the app in AllowedOpenApps, the
	// caller has to opt into kill commands separately.
	AllowClose bool `json:"allow_close"`

	// AllowLock toggles !lock (locks the Windows session — one-shot, no
	// PowerShell spawns, but still gated because lockouts are annoying).
	AllowLock bool `json:"allow_lock"`

	// AllowScreenshot toggles !screenshot. When false, command is rejected.
	AllowScreenshot bool `json:"allow_screenshot"`

	// AllowMedia toggles play/pause/next/prev/now (least invasive — just
	// media keys via SendKeys).
	AllowMedia bool `json:"allow_media"`

	// AllowVolume toggles !volume. When false, command is rejected.
	AllowVolume bool `json:"allow_volume"`

	// RequireConfirm, when true, makes destructive commands (open/close/
	// lock/screenshot/volume) require the user to reply with a one-time
	// confirmation token. Less invasive commands (media) are exempt.
	// Media playback from a chat is annoying enough without confirmation.
	RequireConfirm bool `json:"require_confirm"`

	// ConfirmTTL is how long a confirmation token is valid. Defaults to 60s.
	ConfirmTTL time.Duration `json:"-"`

	// RateLimitPerMin caps command invocations per sender per minute. 0 = off.
	RateLimitPerMin int `json:"rate_limit_per_min"`
}

// SafeDefaultPolicy is returned by DefaultPolicy(). It blocks everything
// by default — desktop control must be opted into.
func DefaultPolicy() *SecurityPolicy {
	return &SecurityPolicy{
		Enabled:         false,
		AllowedOpenApps: nil,
		AllowClose:      false,
		AllowLock:       false,
		AllowScreenshot: false,
		AllowMedia:      false,
		AllowVolume:     false,
		RequireConfirm:  true,
		ConfirmTTL:      60 * time.Second,
		RateLimitPerMin: 10,
	}
}

// pendingConfirm tracks a !command waiting for a "ya" reply.
type pendingConfirm struct {
	Action   string
	Args     string
	Sender   string
	Token    string
	IssuedAt time.Time
	Policy   *SecurityPolicy
}

// confirmationStore is a tiny thread-safe map of issued tokens.
type confirmationStore struct {
	mu     sync.Mutex
	tokens map[string]pendingConfirm
}

var confirmStore = &confirmationStore{tokens: map[string]pendingConfirm{}}

func (c *confirmationStore) issue(p pendingConfirm) string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	tok := hex.EncodeToString(b)
	p.Token = tok
	c.mu.Lock()
	c.tokens[tok] = p
	c.mu.Unlock()
	return tok
}

func (c *confirmationStore) consume(tok, sender string) (pendingConfirm, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	p, ok := c.tokens[tok]
	if !ok {
		return pendingConfirm{}, false
	}
	delete(c.tokens, tok)
	if p.Sender != sender {
		return pendingConfirm{}, false
	}
	if time.Since(p.IssuedAt) > p.Policy.ConfirmTTL {
		return pendingConfirm{}, false
	}
	return p, true
}

func (c *confirmationStore) gc() {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	for k, p := range c.tokens {
		if now.Sub(p.IssuedAt) > p.Policy.ConfirmTTL {
			delete(c.tokens, k)
		}
	}
}

// rateLimiter is a simple per-sender counter. Resets each minute.
type rateLimiter struct {
	mu      sync.Mutex
	hits    map[string][]time.Time
	perMin  int
}

func newRateLimiter(perMin int) *rateLimiter {
	return &rateLimiter{hits: map[string][]time.Time{}, perMin: perMin}
}

func (r *rateLimiter) allow(sender string) bool {
	if r.perMin <= 0 {
		return true
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	cutoff := now.Add(-time.Minute)
	keep := r.hits[sender][:0]
	for _, t := range r.hits[sender] {
		if t.After(cutoff) {
			keep = append(keep, t)
		}
	}
	if len(keep) >= r.perMin {
		r.hits[sender] = keep
		return false
	}
	keep = append(keep, now)
	r.hits[sender] = keep
	return true
}

// callerContext is the chat-side context we need to enforce policy.
// Sender is the originating JID/LID; text is the raw incoming message.
type callerContext struct {
	Sender string
	Text   string
}

// policyResult tells HandleCommand whether to run, ask, or refuse.
type policyResult int

const (
	policyRun    policyResult = iota // run the command normally
	policyConfirm                    // ask user to confirm with a token
	policyReject                     // refuse with an error message
)

// classifyCommand inspects a parsed command and returns whether the policy
// allows it, requires confirmation, or rejects it outright.
func (p *SecurityPolicy) classify(action, sender string) policyResult {
	if p == nil || !p.Enabled {
		return policyReject
	}
	if len(p.AllowedSenders) > 0 {
		ok := false
		for _, s := range p.AllowedSenders {
			if s == sender {
				ok = true
				break
			}
		}
		if !ok {
			return policyReject
		}
	}
	switch action {
	case "open", "launch", "start":
		// Per-app allowlist happens later. Continue so confirmation can still
		// apply to !open.
	case "close", "kill", "stop":
		if !p.AllowClose {
			return policyReject
		}
	case "lock", "kunci":
		if !p.AllowLock {
			return policyReject
		}
	case "screenshot", "ss", "capture":
		if !p.AllowScreenshot {
			return policyReject
		}
	case "play", "pause", "playpause", "next", "skip", "prev", "previous", "back", "now", "nowplaying", "song", "lagu":
		if !p.AllowMedia {
			return policyReject
		}
	case "volume", "vol":
		if !p.AllowVolume {
			return policyReject
		}
	}
	if p.RequireConfirm {
		switch action {
		case "open", "launch", "start", "close", "kill", "stop",
			"lock", "kunci", "screenshot", "ss", "capture", "volume", "vol":
			return policyConfirm
		}
	}
	return policyRun
}

// allowOpenApp enforces the per-app allowlist.
func (p *SecurityPolicy) allowOpenApp(name string) bool {
	if len(p.AllowedOpenApps) == 0 {
		return false
	}
	lower := strings.ToLower(strings.TrimSpace(name))
	for _, allowed := range p.AllowedOpenApps {
		if strings.ToLower(strings.TrimSpace(allowed)) == lower {
			return true
		}
	}
	return false
}

// denyOpenList returns a sorted, human-readable view of the allowlist for
// the !help response.
func (p *SecurityPolicy) denyOpenList() string {
	if len(p.AllowedOpenApps) == 0 {
		return "(kosong — tidak ada app yang boleh dibuka)"
	}
	out := make([]string, len(p.AllowedOpenApps))
	copy(out, p.AllowedOpenApps)
	sort.Strings(out)
	return strings.Join(out, ", ")
}

// global rate limiter, keyed by sender. Per-policy limits reset the bucket
// when the policy is reloaded, which is fine because policies reload on app
// restart.
var cmdLimiter = newRateLimiter(10)
