//go:build windows

package scanner

import "os"

// FileBytes is the file's size; Windows hard links are not detected here.
func FileBytes(info os.FileInfo) int64 { return info.Size() }
