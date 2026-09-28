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
