package desktop

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// HandleCommand is kept for old callers. It now uses the safe default policy,
// which rejects desktop control unless the caller explicitly opts in via
// HandleCommandWithPolicy.
func HandleCommand(ctrl Controller, text string) string {
	return HandleCommandWithPolicy(ctrl, text, "", DefaultPolicy())
}

// HandleCommandWithPolicy parses a !command and executes it only when allowed
// by SecurityPolicy. This prevents chat-side desktop control from silently
// spawning PowerShell/cmd/taskkill for any incoming WhatsApp message.
func HandleCommandWithPolicy(ctrl Controller, text, sender string, policy *SecurityPolicy) string {
	text = strings.TrimSpace(text)
	confirmPrefix := "!confirm "
	if strings.HasPrefix(strings.ToLower(text), confirmPrefix) {
		tok := strings.TrimSpace(text[len(confirmPrefix):])
		pending, ok := confirmStore.consume(tok, sender)
		if !ok {
			return "❌ Konfirmasi tidak valid / expired"
		}
		return runDesktopCommand(ctrl, pending.Action, strings.Fields(pending.Args), pending.Policy)
	}

	// Remove leading ! and trim
	cmd := strings.TrimSpace(strings.TrimPrefix(text, "!"))
	if cmd == "" {
		return ""
	}

	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return ""
	}
	action := strings.ToLower(parts[0])

	// Help is safe even when desktop control is disabled.
	if action == "help" || action == "bantuan" || action == "cmd" {
		return desktopHelp(policy)
	}

	if policy == nil {
		policy = DefaultPolicy()
	}
	if policy.ConfirmTTL == 0 {
		policy.ConfirmTTL = 60 * time.Second
	}
	cmdLimiter.perMin = policy.RateLimitPerMin
	if !cmdLimiter.allow(sender) {
		return "❌ Terlalu banyak command. Coba lagi nanti."
	}

	decision := policy.classify(action, sender)
	if decision == policyReject {
		return "🔒 Desktop Controller diblokir. Aktifkan dan allowlist command/sender dulu di Settings."
	}

	if action == "open" || action == "launch" || action == "start" || action == "close" || action == "kill" || action == "stop" {
		if len(parts) < 2 {
			return "❌ Format: !open <nama_app> / !close <nama_app>"
		}
		if !policy.allowOpenApp(parts[1]) {
			return fmt.Sprintf("🔒 App tidak di-allow: %s\nAllowed: %s", parts[1], policy.denyOpenList())
		}
	}

	if decision == policyConfirm {
		tok := confirmStore.issue(pendingConfirm{
			Action:   action,
			Args:     strings.Join(parts[1:], " "),
			Sender:   sender,
			IssuedAt: time.Now(),
			Policy:   policy,
		})
		confirmStore.gc()
		return fmt.Sprintf("⚠️ Konfirmasi command desktop: !%s\nBalas: !confirm %s\nExpired: %s", cmd, tok, policy.ConfirmTTL)
	}

	return runDesktopCommand(ctrl, action, parts[1:], policy)
}

func runDesktopCommand(ctrl Controller, action string, args []string, policy *SecurityPolicy) string {
	switch action {
	case "open", "launch", "start":
		if len(args) < 1 {
			return "❌ Format: !open <nama_app>\nContoh: !open firefox"
		}
		appName := args[0]
		err := ctrl.OpenApp(appName)
		if err != nil {
			return fmt.Sprintf("❌ Gagal membuka %s: %v", appName, err)
		}
		return fmt.Sprintf("✅ %s dibuka!", appName)

	case "close", "kill", "stop":
		if len(args) < 1 {
			return "❌ Format: !close <nama_app>\nContoh: !close firefox"
		}
		appName := args[0]
		err := ctrl.CloseApp(appName)
		if err != nil {
			return fmt.Sprintf("❌ Gagal menutup %s: %v", appName, err)
		}
		return fmt.Sprintf("✅ %s ditutup!", appName)

	case "apps", "list", "ps":
		apps, err := ctrl.ListRunningApps()
		if err != nil {
			return fmt.Sprintf("❌ Gagal list apps: %v", err)
		}
		if len(apps) == 0 {
			return "📋 Tidak ada app yang berjalan"
		}
		var sb strings.Builder
		sb.WriteString("📋 Running Apps:\n")
		for i, app := range apps {
			if i >= 15 {
				sb.WriteString(fmt.Sprintf("... dan %d lainnya", len(apps)-15))
				break
			}
			sb.WriteString(fmt.Sprintf("• %s (PID %d)\n", app.Name, app.PID))
		}
		return sb.String()

	case "play", "pause", "playpause":
		err := ctrl.MediaPlayPause()
		if err != nil {
			return fmt.Sprintf("❌ Gagal: %v", err)
		}
		return "▶️ Play/Pause"

	case "next", "skip":
		err := ctrl.MediaNext()
		if err != nil {
			return fmt.Sprintf("❌ Gagal: %v", err)
		}
		return "⏭️ Next"

	case "prev", "previous", "back":
		err := ctrl.MediaPrev()
		if err != nil {
			return fmt.Sprintf("❌ Gagal: %v", err)
		}
		return "⏮️ Previous"

	case "now", "nowplaying", "song", "lagu":
		info, err := ctrl.MediaNowPlaying()
		if err != nil {
			return fmt.Sprintf("❌ Gagal: %v", err)
		}
		if info == nil || info.Title == "" {
			return "🎵 Tidak ada lagu yang diputar"
		}
		status := "⏸️"
		if info.Playing {
			status = "▶️"
		}
		result := fmt.Sprintf("%s %s", status, info.Title)
		if info.Artist != "" {
			result += fmt.Sprintf(" - %s", info.Artist)
		}
		return result

	case "volume", "vol":
		if len(args) < 1 {
			vol, err := ctrl.GetVolume()
			if err != nil {
				return fmt.Sprintf("❌ Gagal: %v", err)
			}
			return fmt.Sprintf("🔊 Volume: %d%%", vol)
		}
		pct, err := strconv.Atoi(args[0])
		if err != nil || pct < 0 || pct > 100 {
			return "❌ Format: !volume <0-100>"
		}
		err = ctrl.SetVolume(pct)
		if err != nil {
			return fmt.Sprintf("❌ Gagal set volume: %v", err)
		}
		return fmt.Sprintf("🔊 Volume: %d%%", pct)

	case "screenshot", "ss", "capture":
		path, err := ctrl.TakeScreenshot()
		if err != nil {
			return fmt.Sprintf("❌ Gagal screenshot: %v", err)
		}
		return fmt.Sprintf("📸 Screenshot: %s", path)

	case "lock", "kunci":
		err := ctrl.LockScreen()
		if err != nil {
			return fmt.Sprintf("❌ Gagal lock: %v", err)
		}
		return "🔒 Screen locked!"

	default:
		return fmt.Sprintf("❓ Command tidak dikenali: !%s\nKetik !help untuk daftar command", action)
	}
}

func desktopHelp(policy *SecurityPolicy) string {
	allowed := "(desktop control off)"
	if policy != nil && policy.Enabled {
		allowed = policy.denyOpenList()
	}
	return `🤖 Desktop Commands:
!open <app> - Buka aplikasi (allowlist only)
!close <app> - Tutup aplikasi (opt-in)
!apps - List app berjalan
!play - Play/Pause media (opt-in)
!next - Next track (opt-in)
!prev - Previous track (opt-in)
!now - Lagu sekarang (opt-in)
!volume [0-100] - Get/Set volume (opt-in)
!screenshot - Ambil screenshot (opt-in)
!lock - Lock screen (opt-in)
!confirm <token> - Konfirmasi command berisiko

Allowed apps: ` + allowed
}
