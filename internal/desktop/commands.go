package desktop

import (
	"fmt"
	"strconv"
	"strings"
)

// HandleCommand parses a !command and executes it, returning a response string.
func HandleCommand(ctrl Controller, text string) string {
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

	switch action {
	case "open", "launch", "start":
		if len(parts) < 2 {
			return "❌ Format: !open <nama_app>\nContoh: !open terminal, !open firefox"
		}
		appName := parts[1]
		err := ctrl.OpenApp(appName)
		if err != nil {
			return fmt.Sprintf("❌ Gagal membuka %s: %v", appName, err)
		}
		return fmt.Sprintf("✅ %s dibuka!", appName)

	case "close", "kill", "stop":
		if len(parts) < 2 {
			return "❌ Format: !close <nama_app>\nContoh: !close firefox"
		}
		appName := parts[1]
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
		if len(parts) < 2 {
			// Get current volume
			vol, err := ctrl.GetVolume()
			if err != nil {
				return fmt.Sprintf("❌ Gagal: %v", err)
			}
			return fmt.Sprintf("🔊 Volume: %d%%", vol)
		}
		pct, err := strconv.Atoi(parts[1])
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

	case "help", "bantuan", "cmd":
		return `🤖 Desktop Commands:
!open <app> - Buka aplikasi
!close <app> - Tutup aplikasi
!apps - List app berjalan
!play - Play/Pause media
!next - Next track
!prev - Previous track
!now - Lagu sekarang
!volume [0-100] - Get/Set volume
!screenshot - Ambil screenshot
!lock - Lock screen`

	default:
		return fmt.Sprintf("❓ Command tidak dikenali: !%s\nKetik !help untuk daftar command", cmd)
	}
}
