//go:build !darwin && !linux

package sysinfo

// totalMemory is not implemented for this platform; the settings page shows the
// field as unavailable rather than guessing.
func totalMemory() uint64 { return 0 }
