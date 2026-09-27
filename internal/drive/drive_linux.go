//go:build linux

// Drive detection for Linux via /proc/mounts.
package drive

import (
	"bufio"
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
			d.Name, d.Root = "Home (/home)", true
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

// isRemovableLinux reads /sys/block/<dev>/removable when available.
func isRemovableLinux(dev string) bool {
	base := filepath.Base(dev)
	name := strings.TrimLeft(base, "0123456789")
	if i := strings.Index(base, name); i > 0 {
		base = base[i:] // keep the partition number
	}
	f, err := os.Open(filepath.Join("/sys/block", name, "removable"))
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
