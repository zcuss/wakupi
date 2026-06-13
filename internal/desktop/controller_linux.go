//go:build linux

package desktop

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/godbus/dbus/v5"
)

type linuxController struct{}

// New returns the platform-appropriate controller.
func New() Controller {
	return &linuxController{}
}

// ListRunningApps reads /proc to find running processes with a window.
func (l *linuxController) ListRunningApps() ([]AppInfo, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, err
	}

	seen := make(map[string]bool)
	var apps []AppInfo

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}

		// Read process name from /proc/PID/comm
		commPath := filepath.Join("/proc", entry.Name(), "comm")
		data, err := os.ReadFile(commPath)
		if err != nil {
			continue
		}
		name := strings.TrimSpace(string(data))
		if name == "" || seen[name] {
			continue
		}

		// Skip kernel threads and our own process
		if pid <= 10 || pid == os.Getpid() {
			continue
		}

		// Skip system processes
		skip := []string{"kworker", "ksoftirqd", "migration", "rcu_", "watchdog",
			"kthread", "init", "systemd", "dbus-daemon", "networkd", "journald",
			"logind", "udevd", "polkitd", "accounts-daemon", "colord", "rtkit-daemon",
			"power-profiles", "udisksd", "fwupd", "packagekitd", "snapd", "crond",
			"agetty", "sshd", "cron", "at-spi", "dconf-service", "gvfsd", "xdg-permission",
			"pipewire", "wireplumber", "pulseaudio", "tracker-miner", "xdg-desktop-portal",
			"gnome-shell", "Xwayland", "mutter", "kwin_wayland", "plasmashell",
		}
		skipThis := false
		for _, s := range skip {
			if strings.HasPrefix(name, s) {
				skipThis = true
				break
			}
		}
		if skipThis {
			continue
		}

		seen[name] = true
		apps = append(apps, AppInfo{Name: name, PID: pid})
	}

	sort.Slice(apps, func(i, j int) bool {
		return apps[i].Name < apps[j].Name
	})

	return apps, nil
}

// CloseApp kills a process by name.
func (l *linuxController) CloseApp(name string) error {
	// Find PID by name
	apps, err := l.ListRunningApps()
	if err != nil {
		return err
	}
	for _, app := range apps {
		if app.Name == name {
			return syscall.Kill(app.PID, syscall.SIGTERM)
		}
	}
	return fmt.Errorf("app not found: %s", name)
}

// OpenApp launches an application.
func (l *linuxController) OpenApp(name string) error {
	// Try gtk-launch first (works with .desktop files)
	cmd := exec.Command("gtk-launch", name)
	if err := cmd.Start(); err == nil {
		return nil
	}

	// Fallback: try directly
	cmd = exec.Command(name)
	cmd.Env = os.Environ()
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("cannot open %s: %v", name, err)
	}
	return nil
}

// mprisCall makes a D-Bus call to MPRIS2.
func mprisCall(method string) error {
	conn, err := dbus.SessionBus()
	if err != nil {
		return fmt.Errorf("dbus connect: %v", err)
	}
	defer conn.Close()

	// Find available MPRIS2 players
	var names []string
	err = conn.Object("org.freedesktop.DBus", "/org/freedesktop/DBus").Call(
		"org.freedesktop.DBus.ListNames", 0).Store(&names)
	if err != nil {
		return err
	}

	var player string
	for _, n := range names {
		if strings.HasPrefix(n, "org.mpris.MediaPlayer2.") {
			player = n
			break
		}
	}
	if player == "" {
		return fmt.Errorf("no media player found")
	}

	obj := conn.Object(player, "/org/mpris/MediaPlayer2")
	call := obj.Call("org.mpris.MediaPlayer2.Player."+method, 0)
	return call.Err
}

// MediaPlayPause toggles play/pause via MPRIS2.
func (l *linuxController) MediaPlayPause() error {
	return mprisCall("PlayPause")
}

// MediaNext skips to next track via MPRIS2.
func (l *linuxController) MediaNext() error {
	return mprisCall("Next")
}

// MediaPrev skips to previous track via MPRIS2.
func (l *linuxController) MediaPrev() error {
	return mprisCall("Previous")
}

// MediaNowPlaying returns current track info via MPRIS2.
func (l *linuxController) MediaNowPlaying() (*MediaInfo, error) {
	conn, err := dbus.SessionBus()
	if err != nil {
		return nil, fmt.Errorf("dbus connect: %v", err)
	}
	defer conn.Close()

	// Find MPRIS2 player
	var names []string
	err = conn.Object("org.freedesktop.DBus", "/org/freedesktop/DBus").Call(
		"org.freedesktop.DBus.ListNames", 0).Store(&names)
	if err != nil {
		return nil, err
	}

	var player string
	for _, n := range names {
		if strings.HasPrefix(n, "org.mpris.MediaPlayer2.") {
			player = n
			break
		}
	}
	if player == "" {
		return nil, fmt.Errorf("no media player found")
	}

	obj := conn.Object(player, "/org/mpris/MediaPlayer2")

	// Get PlaybackStatus
	status, err := obj.GetProperty("org.mpris.MediaPlayer2.Player.PlaybackStatus")
	if err != nil {
		return nil, err
	}
	playing := status.String() == "\"Playing\""

	// Get Metadata
	metadata, err := obj.GetProperty("org.mpris.MediaPlayer2.Player.Metadata")
	if err != nil {
		return &MediaInfo{Playing: playing, Player: player}, nil
	}

	info := &MediaInfo{Playing: playing, Player: player}

	// Parse metadata map
	if md, ok := metadata.Value().(map[string]dbus.Variant); ok {
		if v, ok := md["xesam:title"]; ok {
			info.Title = strings.Trim(v.String(), "\"")
		}
		if v, ok := md["xesam:artist"]; ok {
			// Artist is usually an array
			s := v.String()
			s = strings.TrimPrefix(s, "[")
			s = strings.TrimSuffix(s, "]")
			s = strings.Trim(s, "\"")
			info.Artist = s
		}
		if v, ok := md["xesam:album"]; ok {
			info.Album = strings.Trim(v.String(), "\"")
		}
	}

	return info, nil
}

// GetVolume returns current volume percentage.
func (l *linuxController) GetVolume() (int, error) {
	// Try pactl (PipeWire/PulseAudio)
	out, err := exec.Command("pactl", "get-sink-volume", "@DEFAULT_SINK@").Output()
	if err == nil {
		// Parse: "Volume: front-left: 32768 /  50% / -18.06 dB ..."
		s := string(out)
		for _, part := range strings.Fields(s) {
			if strings.HasSuffix(part, "%") {
				pct := strings.TrimSuffix(part, "%")
				if v, err := strconv.Atoi(pct); err == nil {
					return v, nil
				}
			}
		}
	}

	// Fallback: amixer
	out, err = exec.Command("amixer", "get", "Master").Output()
	if err != nil {
		return 0, fmt.Errorf("cannot get volume: %v", err)
	}
	s := string(out)
	for _, line := range strings.Split(s, "\n") {
		if strings.Contains(line, "%") {
			idx := strings.Index(line, "[")
			end := strings.Index(line, "%]")
			if idx >= 0 && end > idx {
				pct := line[idx+1 : end]
				if v, err := strconv.Atoi(pct); err == nil {
					return v, nil
				}
			}
		}
	}
	return 0, fmt.Errorf("cannot parse volume")
}

// SetVolume sets volume percentage.
func (l *linuxController) SetVolume(pct int) error {
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}

	// Try pactl first
	err := exec.Command("pactl", "set-sink-volume", "@DEFAULT_SINK@", fmt.Sprintf("%d%%", pct)).Run()
	if err == nil {
		return nil
	}

	// Fallback: amixer
	return exec.Command("amixer", "set", "Master", fmt.Sprintf("%d%%", pct)).Run()
}

// TakeScreenshot captures the screen.
func (l *linuxController) TakeScreenshot() (string, error) {
	tmpDir := os.TempDir()
	path := filepath.Join(tmpDir, fmt.Sprintf("screenshot_%d.png", time.Now().Unix()))

	// Try gnome-screenshot
	err := exec.Command("gnome-screenshot", "-f", path).Run()
	if err == nil {
		return path, nil
	}

	// Try scrot
	err = exec.Command("scrot", path).Run()
	if err == nil {
		return path, nil
	}

	// Try grim (Wayland)
	err = exec.Command("grim", path).Run()
	if err == nil {
		return path, nil
	}

	return "", fmt.Errorf("no screenshot tool available (install gnome-screenshot, scrot, or grim)")
}

// LockScreen locks the screen via D-Bus.
func (l *linuxController) LockScreen() error {
	conn, err := dbus.SessionBus()
	if err != nil {
		return fmt.Errorf("dbus connect: %v", err)
	}
	defer conn.Close()

	// Try org.freedesktop.ScreenSaver first
	obj := conn.Object("org.freedesktop.ScreenSaver", "/org/freedesktop/ScreenSaver")
	err = obj.Call("org.freedesktop.ScreenSaver.Lock", 0).Err
	if err == nil {
		return nil
	}

	// Try org.gnome.ScreenSaver
	obj = conn.Object("org.gnome.ScreenSaver", "/org/gnome/ScreenSaver")
	return obj.Call("org.gnome.ScreenSaver.Lock", 0).Err
}

// Ensure linuxController implements Controller at compile time.
var _ Controller = (*linuxController)(nil)

// Keep context import used
var _ = context.Background
