//go:build linux

// Drive detection for Linux via /proc/mounts.
package drive

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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
	d.fsKey = fmt.Sprintf("%d:%d", st.Fsid.Val[0], st.Fsid.Val[1])
	return d, nil
}

func detectLinux() ([]Info, error) {
	var out []Info
	home, _ := os.UserHomeDir()
	seen := map[string]bool{}

	f, err := os.Open("/proc/mounts")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) < 3 {
			continue
		}
		dev, mp := fields[0], unescapeMount(fields[1])
		if !strings.HasPrefix(dev, "/dev/") || seen[mp] || !isRealDevice(dev) {
			continue
		}
		seen[mp] = true

		d := Info{MountPoint: mp, Removable: isRemovableLinux(dev)}
		switch {
		case mp == "/":
			d.Name, d.Root = "System (/)", true
		case mp == home:
			// Home gets a friendly name but is NOT the system drive. Marking it Root
			// would auto-select it next to "/", and since "/" is a parent mount of
			// /home the scan would then walk /home twice and double-count it.
			d.Name = "Home (/home)"
		default:
			d.Name = filepath.Base(mp)
			if len(d.Name) > 24 {
				d.Name = d.Name[:24]
			}
		}
		if d2, err := statfs(d); err == nil {
			out = append(out, d2)
		}
	}
	if len(out) == 0 {
		if d, err := statfs(Info{Name: "System (/)", MountPoint: "/", Root: true}); err == nil {
			out = append(out, d)
		}
	}
	sortByRoot(out)
	return uniqueVolumes(out), nil
}

// isRealDevice filters loop/ramdisk/sr/fd devices.
func isRealDevice(dev string) bool {
	name := strings.TrimLeft(filepath.Base(dev), "0123456789")
	for _, p := range []string{"loop", "ram", "zram", "sr", "fd"} {
		if name == p || strings.HasPrefix(name, p) {
			return false
		}
	}
	return true
}

// partitionedDiskRe matches an nvme or mmcblk whole-disk name, optionally
// followed by a "p<partition>" suffix. Their disk names embed digits (nvme0n1,
// mmcblk0), so only the "p<digits>" tail marks a partition.
var partitionedDiskRe = regexp.MustCompile(`^(nvme\d+n\d+|mmcblk\d+)(p\d+)?$`)

// parentDisk maps a partition device name to its whole-disk block name, which is
// what /sys/block is keyed by: "sdb1" -> "sdb", "nvme0n1p2" -> "nvme0n1",
// "mmcblk0p1" -> "mmcblk0". A name without a partition suffix is returned as-is.
func parentDisk(name string) string {
	if m := partitionedDiskRe.FindStringSubmatch(name); m != nil {
		return m[1] // nvme/mmcblk: the disk name, minus any trailing "p<digits>"
	}
	// SCSI / virtio / Xen style (sda1, vdb2, xvda1): trailing digits are the
	// partition number; a name ending in a letter (sda) is left untouched.
	return strings.TrimRight(name, "0123456789")
}

// isRemovableLinux reports whether the parent disk of a device is removable by
// reading /sys/block/<parent>/removable. Checking the partition itself (e.g.
// "sdb1") always misses it, because /sys/block lists whole disks only.
func isRemovableLinux(dev string) bool {
	f, err := os.Open(filepath.Join("/sys/block", parentDisk(filepath.Base(dev)), "removable"))
	if err != nil {
		return false
	}
	defer f.Close()
	b, err := bufio.NewReader(f).ReadString('\n')
	if err != nil {
		return false
	}
	return strings.TrimSpace(b) == "1"
}

// unescapeMount converts octal escapes in /proc/mounts (e.g. \040) to bytes.
func unescapeMount(s string) string {
	if !strings.Contains(s, "\\") {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+3 < len(s) {
			var v int
			if _, err := fmt.Sscanf(s[i+1:i+4], "%o", &v); err == nil {
				b.WriteByte(byte(v))
				i += 3
				continue
			}
		}
		b.WriteByte(s[i])
	}
	return b.String()
}
