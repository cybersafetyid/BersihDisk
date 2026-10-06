//go:build windows

// Windows-specific drive detection using golang.org/x/sys/windows.
package drive

import (
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"golang.org/x/sys/windows"
)

// detectWindows returns drives A: through Z: whose type is scannable
// (fixed or removable).
func detectWindows() ([]Info, error) {
	var out []Info
	for ch := byte('A'); ch <= 'Z'; ch++ {
		root := string(ch) + `:\`
		driveType := windows.GetDriveType(windows.StringToUTF16Ptr(root))
		if driveType != windows.DRIVE_FIXED && driveType != windows.DRIVE_REMOVABLE {
			continue
		}

		var free, total uint64
		if err := windows.GetDiskFreeSpaceEx(
			windows.StringToUTF16Ptr(root), nil, &total, &free); err != nil {
			continue // drive without media (empty card reader)
		}

		volBuf := make([]uint16, 261)
		_ = windows.GetVolumeInformation(
			windows.StringToUTF16Ptr(root),
			&volBuf[0], uint32(len(volBuf)), nil, nil, nil,
			nil, 0)
		// GetVolumeInformationW writes a null-terminated label and returns no length,
		// so read up to the terminator; slicing by an (always-zero) counter yielded "".
		label := syscall.UTF16ToString(volBuf)

		d := Info{
			Name:       label,
			MountPoint: filepath.Clean(root),
			TotalBytes: total,
			FreeBytes:  free,
			Removable:  driveType == windows.DRIVE_REMOVABLE,
		}
		if d.Name == "" {
			d.Name = "Drive " + string(ch)
		}
		if ch == 'C' {
			d.Root = true
			d.Name = strings.TrimSpace(d.Name + " (System C:)")
		}
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Root != out[j].Root {
			return out[i].Root
		}
		return out[i].MountPoint < out[j].MountPoint
	})
	return out, nil
}
