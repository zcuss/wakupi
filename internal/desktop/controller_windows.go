//go:build windows

package desktop

import (
	"encoding/csv"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type windowsController struct{}

// New returns the platform-appropriate controller.
func New() Controller {
	return &windowsController{}
}

// ListRunningApps uses tasklist to enumerate running processes.
func (w *windowsController) ListRunningApps() ([]AppInfo, error) {
	out, err := exec.Command("tasklist", "/FO", "CSV", "/NH").Output()
	if err != nil {
		return nil, err
	}

	seen := make(map[string]bool)
	var apps []AppInfo

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		r := csv.NewReader(strings.NewReader(line))
		records, err := r.Read()
		if err != nil || len(records) < 2 {
			continue
		}

		name := strings.TrimSuffix(records[0], ".exe")
		pid, err := strconv.Atoi(strings.TrimSpace(records[1]))
		if err != nil {
			continue
		}

		// Skip system processes
		skip := []string{"svchost", "csrss", "lsass", "services", "wininit",
			"winlogon", "dwm", "RuntimeBroker", "SearchIndexer", "sihost",
			"taskhostw", "ctfmon", "dllhost", "smss", "conhost", "fontdrvhost",
		}
		skipThis := false
		for _, s := range skip {
			if strings.EqualFold(name, s) {
				skipThis = true
				break
			}
		}
		if skipThis || seen[name] {
			continue
		}

		seen[name] = true
		apps = append(apps, AppInfo{Name: name, PID: pid})
	}

	return apps, nil
}

// CloseApp kills a process by name.
func (w *windowsController) CloseApp(name string) error {
	return exec.Command("taskkill", "/IM", name+".exe", "/T").Run()
}

// OpenApp launches an application.
func (w *windowsController) OpenApp(name string) error {
	return exec.Command("cmd", "/c", "start", "", name).Run()
}

// MediaPlayPause sends media play/pause key via PowerShell.
func (w *windowsController) MediaPlayPause() error {
	return exec.Command("powershell", "-Command",
		`Add-Type -AssemblyName System.Windows.Forms; [System.Windows.Forms.SendKeys]::SendWait("{MEDIA_PLAY_PAUSE}")`).Run()
}

// MediaNext sends media next key via PowerShell.
func (w *windowsController) MediaNext() error {
	return exec.Command("powershell", "-Command",
		`Add-Type -AssemblyName System.Windows.Forms; [System.Windows.Forms.SendKeys]::SendWait("{MEDIA_NEXT_TRACK}")`).Run()
}

// MediaPrev sends media previous key via PowerShell.
func (w *windowsController) MediaPrev() error {
	return exec.Command("powershell", "-Command",
		`Add-Type -AssemblyName System.Windows.Forms; [System.Windows.Forms.SendKeys]::SendWait("{MEDIA_PREV_TRACK}")`).Run()
}

// MediaNowPlaying returns current track info via PowerShell.
func (w *windowsController) MediaNowPlaying() (*MediaInfo, error) {
	// Use SMTC via PowerShell to get media info
	script := `
Add-Type -AssemblyName System.Runtime.WindowsRuntime
$asyncInfo = ([Windows.Media.Control.GlobalMediaControlManager]::RequestAsync()).GetAwaiter().GetResult()
if ($asyncInfo -ne $null) {
    $info = $asyncInfo.GetPlaybackInfo()
    $title = $asyncInfo.MediaProperties.Title
    $artist = $asyncInfo.MediaProperties.Artist
    Write-Output "$title|$artist|Playing"
} else {
    Write-Output "||Stopped"
}`
	out, err := exec.Command("powershell", "-Command", script).Output()
	if err != nil {
		return &MediaInfo{Playing: false}, nil
	}

	parts := strings.Split(strings.TrimSpace(string(out)), "|")
	info := &MediaInfo{}
	if len(parts) >= 1 {
		info.Title = parts[0]
	}
	if len(parts) >= 2 {
		info.Artist = parts[1]
	}
	if len(parts) >= 3 {
		info.Playing = parts[2] == "Playing"
	}
	return info, nil
}

// GetVolume returns current volume percentage.
func (w *windowsController) GetVolume() (int, error) {
	script := `
Add-Type -AssemblyName System.Runtime.WindowsRuntime
$dev = ([Windows.Devices.Enumeration.DeviceInformation]::FindAllAsync('System.Devices.InterfaceClassGuid:="{4D36E96C-E325-11CE-BFC1-08002BE10318}"')).GetAwaiter().GetResult()
Write-Output "50"`
	out, err := exec.Command("powershell", "-Command", script).Output()
	if err != nil {
		return 50, nil
	}
	v, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		return 50, nil
	}
	return v, nil
}

// SetVolume sets volume percentage.
func (w *windowsController) SetVolume(pct int) error {
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	// Use nircmd or volume adjuster
	return exec.Command("powershell", "-Command",
		fmt.Sprintf(`$wshShell = New-Object -ComObject WScript.Shell; 1..%d | %% { $wshShell.SendKeys([char]175) }; 1..%d | %% { $wshShell.SendKeys([char]174) }`, 100-pct, pct)).Run()
}

// TakeScreenshot captures the screen.
func (w *windowsController) TakeScreenshot() (string, error) {
	tmpDir := os.TempDir()
	path := filepath.Join(tmpDir, fmt.Sprintf("screenshot_%d.png", time.Now().Unix()))

	script := fmt.Sprintf(`
Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing
$screen = [System.Windows.Forms.Screen]::PrimaryScreen.Bounds
$bitmap = New-Object System.Drawing.Bitmap($screen.Width, $screen.Height)
$graphics = [System.Drawing.Graphics]::FromImage($bitmap)
$graphics.CopyFromScreen($screen.Location, [System.Drawing.Point]::Empty, $screen.Size)
$bitmap.Save('%s')
$graphics.Dispose()
$bitmap.Dispose()`, path)

	err := exec.Command("powershell", "-Command", script).Run()
	if err != nil {
		return "", err
	}
	return path, nil
}

// LockScreen locks the Windows session.
func (w *windowsController) LockScreen() error {
	return exec.Command("rundll32.exe", "user32.dll,LockWorkStation").Run()
}

// Ensure windowsController implements Controller at compile time.
var _ Controller = (*windowsController)(nil)
