//go:build darwin

// Drive detection for macOS.
package drive

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"
)

// statfs fills capacity details for one mount point.
func statfs(d Info) (Info, error) {
	var st unix.Statfs_t
	if err := unix.Statfs(d.MountPoint, &st); err != nil {
		return d, err
	}
	d.TotalBytes = uint64(st.Blocks) * uint64(st.Bsize)
	d.FreeBytes = uint64(st.Bavail) * uint64(st.Bsize)
	d.Removable = st.Flags&unix.MNT_REMOVABLE != 0
	d.fsKey = fmt.Sprintf("%d:%d", st.Fsid.Val[0], st.Fsid.Val[1])
	return d, nil
}

func detectDarwin() ([]Info, error) {
	rootInfo, err := statfs(Info{Name: "Macintosh HD", MountPoint: "/", Root: true})
	if err != nil {
		return nil, err
	}
	out := []Info{rootInfo}

	entries, err := os.ReadDir("/Volumes")
	if err != nil {
		return out, nil // root alone is enough
	}
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), "com.apple.") || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		mp := filepath.Join("/Volumes", e.Name())
		if !isMountPoint(mp) {
			continue // stale mount-point dir of an unmounted disk, not a volume
		}
		d, err := statfs(Info{Name: e.Name(), MountPoint: mp})
		if err != nil {
			continue // dangling/ disconnected disk image
		}
		out = append(out, d)
	}
	sortByRoot(out)
	return uniqueVolumes(out), nil
}
