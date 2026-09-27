//go:build !darwin

package appicon

import "errors"

// Supported reports that this platform has no runtime Dock/taskbar icon API, so
// the settings page hides the picker instead of offering a dead control.
func Supported() bool { return false }

// Set is unavailable here.
func Set([]byte) error {
	return errors.New("changing the app icon at runtime is not supported on this platform")
}
