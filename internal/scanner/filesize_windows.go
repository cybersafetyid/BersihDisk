//go:build windows

package scanner

import "os"

// FileBytes is the file's size; Windows hard links are not detected here.
func FileBytes(info os.FileInfo) int64 { return info.Size() }

// fsDevice does not detect mount boundaries on Windows (each drive is its own
// root and nested volume mount points are rare), so it reports "unknown" and the
// walk stays unpruned — the same behaviour as before per-drive analysis.
func fsDevice(string) (uint64, bool) { return 0, false }
