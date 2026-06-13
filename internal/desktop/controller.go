package desktop

// AppInfo represents a running application.
type AppInfo struct {
	Name string `json:"name"`
	PID  int    `json:"pid"`
	Icon string `json:"icon,omitempty"`
}

// MediaInfo represents currently playing media.
type MediaInfo struct {
	Title   string `json:"title"`
	Artist  string `json:"artist"`
	Album   string `json:"album"`
	Playing bool   `json:"playing"`
	Player  string `json:"player"`
}

// Controller provides cross-platform desktop control.
type Controller interface {
	ListRunningApps() ([]AppInfo, error)
	CloseApp(name string) error
	OpenApp(name string) error
	MediaPlayPause() error
	MediaNext() error
	MediaPrev() error
	MediaNowPlaying() (*MediaInfo, error)
	GetVolume() (int, error)
	SetVolume(pct int) error
	TakeScreenshot() (string, error)
	LockScreen() error
}
