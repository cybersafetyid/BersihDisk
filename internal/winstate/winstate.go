// Package winstate remembers the main window's size, position and maximised state
// between runs, in the per-user config directory of each OS (Application Support
// on macOS, %AppData% on Windows, ~/.config on Linux).
package winstate

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// State is what is restored on the next launch. Width and Height are the window's
// normal (un-maximised) bounds, so leaving maximised mode later returns to them.
type State struct {
	Width      int  `json:"width"`
	Height     int  `json:"height"`
	X          int  `json:"x"`
	Y          int  `json:"y"`
	HasPos     bool `json:"hasPos"`
	Maximised  bool `json:"maximised"`
	Fullscreen bool `json:"fullscreen"`
}

// Default is used on first launch: a maximised window that falls back to
// width x height when the user restores it.
func Default(width, height int) State {
	return State{Width: width, Height: height, Maximised: true}
}

// Path is the state file location.
func Path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "BersihDisk", "window.json"), nil
}

// Load reads the saved state. It reports false when nothing usable is saved
// (first run, unreadable or corrupt file, or bounds smaller than the minimum),
// so the caller falls back to Default.
func Load(minWidth, minHeight int) (State, bool) {
	path, err := Path()
	if err != nil {
		return State{}, false
	}
	return loadFrom(path, minWidth, minHeight)
}

func loadFrom(path string, minWidth, minHeight int) (State, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return State{}, false
	}
	var s State
	if json.Unmarshal(data, &s) != nil || s.Width < minWidth || s.Height < minHeight {
		return State{}, false
	}
	return s, true
}

// Save writes the state atomically, so a crash mid-write cannot leave a
// half-written file behind.
func Save(s State) error {
	path, err := Path()
	if err != nil {
		return err
	}
	return saveTo(path, s)
}

func saveTo(path string, s State) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

// PositionOnScreen reports whether the saved top-left corner still lies inside a
// screen of the given size, so a window last used on a disconnected monitor is
// not restored off-screen. Only the primary screen is known, so a position on a
// secondary monitor is conservatively treated as not restorable.
func (s State) PositionOnScreen(screenW, screenH int) bool {
	const margin = 100 // enough of the title bar must stay reachable
	return s.HasPos && s.X >= 0 && s.Y >= 0 && s.X <= screenW-margin && s.Y <= screenH-margin
}

// Merge builds the state to save from the window as it is now. While maximised or
// fullscreen the live bounds are the screen's, so the previous normal bounds are
// kept instead.
func Merge(prev State, w, h, x, y int, maximised, fullscreen bool) State {
	if maximised || fullscreen {
		prev.Maximised, prev.Fullscreen = maximised, fullscreen
		return prev
	}
	return State{Width: w, Height: h, X: x, Y: y, HasPos: true}
}
