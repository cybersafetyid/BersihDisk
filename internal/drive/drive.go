// Package drive detects mounted drives/volumes across platforms
// (macOS, Windows, Linux).
package drive

import (
	"fmt"
	"runtime"
	"sort"
)

// Info holds details of a single drive/volume.
type Info struct {
	Name       string `json:"name"`       // Display name (e.g. "Macintosh HD", "Data (D:)")
	MountPoint string `json:"mountPoint"` // Absolute mount path
	TotalBytes uint64 `json:"totalBytes"` // Total capacity
	FreeBytes  uint64 `json:"freeBytes"`  // Free space
	Root       bool   `json:"root"`       // true for the main system drive
	Removable  bool   `json:"removable"`  // true for removable media (USB etc.)

	// fsKey identifies the underlying filesystem (from statfs). Unexported so it
	// never reaches the frontend; empty means the identity is unknown.
	fsKey string
}

// UsedBytes returns the used space.
func (d Info) UsedBytes() uint64 {
	if d.TotalBytes < d.FreeBytes {
		return 0
	}
	return d.TotalBytes - d.FreeBytes
}

// Detect lists all scannable drives/volumes.
func Detect() ([]Info, error) {
	switch runtime.GOOS {
	case "darwin":
		return detectDarwin()
	case "windows":
		return detectWindows()
	case "linux":
		return detectLinux()
	default:
		return nil, fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

func sortByRoot(out []Info) {
	sort.Slice(out, func(i, j int) bool {
		if out[i].Root != out[j].Root {
			return out[i].Root
		}
		return out[i].MountPoint < out[j].MountPoint
	})
}

// uniqueVolumes drops entries backed by a filesystem that is already listed,
// keeping the first occurrence. Run it after sortByRoot so the canonical mount
// point (the system root, or the real device mount) is the one that survives.
// This is what stops a stale, never-mounted mount-point directory such as
// /Volumes/ExternalSSD from showing up as a second copy of the system drive —
// statfs on it reports the parent volume's identity and capacities.
// Entries with an unknown identity are always kept.
func uniqueVolumes(in []Info) []Info {
	out := make([]Info, 0, len(in))
	seen := make(map[string]bool, len(in))
	for _, d := range in {
		if d.fsKey == "" {
			out = append(out, d)
			continue
		}
		if seen[d.fsKey] {
			continue
		}
		seen[d.fsKey] = true
		out = append(out, d)
	}
	return out
}
