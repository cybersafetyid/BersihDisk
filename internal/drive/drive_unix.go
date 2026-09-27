//go:build !windows

// Shared non-Windows helpers: mount detection primitives plus a Windows stub.
package drive

import (
	"errors"
	"path/filepath"

	"golang.org/x/sys/unix"
)

func detectWindows() ([]Info, error) {
	return nil, errors.New("detectWindows is only available on Windows")
}

// isMountPoint reports whether a candidate volume path is an actual mount point,
// detected by its device differing from its parent directory's device.
//
// This is the accurate filter for directories that merely survive under
// /Volumes after their disk was unmounted: statfs on such a directory succeeds
// and reports the *parent* volume's identity and capacities, which is what makes
// a phantom duplicate of another drive appear in the list.
func isMountPoint(path string) bool {
	var self, parent unix.Stat_t
	if err := unix.Stat(path, &self); err != nil {
		return false
	}
	if err := unix.Stat(filepath.Dir(path), &parent); err != nil {
		return false
	}
	return self.Dev != parent.Dev
}
