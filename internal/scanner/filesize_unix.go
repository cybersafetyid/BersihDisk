//go:build !windows

package scanner

import (
	"os"
	"syscall"
)

// FileBytes is what deleting the file can free. A file with more than one hard
// link (pnpm's node_modules, for one) keeps its data alive through the other
// links, so it frees nothing.
func FileBytes(info os.FileInfo) int64 {
	if st, ok := info.Sys().(*syscall.Stat_t); ok && st.Nlink > 1 {
		return 0
	}
	return info.Size()
}

// fsDevice returns an identifier for the filesystem dir lives on, so a walk can
// stop at a mount boundary instead of following into another volume. ok is false
// when it cannot be determined; callers then treat everything as one filesystem.
func fsDevice(dir string) (id uint64, ok bool) {
	st, err := os.Lstat(dir)
	if err != nil {
		return 0, false
	}
	if sys, isStat := st.Sys().(*syscall.Stat_t); isStat {
		return uint64(sys.Dev), true
	}
	return 0, false
}
