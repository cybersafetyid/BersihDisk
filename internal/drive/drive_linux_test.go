//go:build linux

package drive

import "testing"

// parentDisk must resolve a partition to the whole-disk name /sys/block is keyed
// by, including the nvme/mmcblk families whose disk names already contain digits.
func TestParentDisk(t *testing.T) {
	cases := []struct {
		dev  string
		want string
	}{
		{"sdb1", "sdb"},
		{"sda", "sda"}, // no partition suffix — leave the disk name intact
		{"vda2", "vda"},
		{"xvda1", "xvda"},
		{"nvme0n1", "nvme0n1"}, // bare namespace: the digits belong to the disk name
		{"nvme0n1p2", "nvme0n1"},
		{"mmcblk0", "mmcblk0"},
		{"mmcblk0p1", "mmcblk0"},
	}
	for _, c := range cases {
		if got := parentDisk(c.dev); got != c.want {
			t.Errorf("parentDisk(%q) = %q, want %q", c.dev, got, c.want)
		}
	}
}

// Loop and ram disks are pseudo devices and must never be listed as drives.
func TestIsRealDeviceFiltersPseudo(t *testing.T) {
	for _, dev := range []string{"loop0", "ram1", "zram0", "sr0", "fd0"} {
		if isRealDevice("/dev/" + dev) {
			t.Errorf("isRealDevice(%q) = true, want false", dev)
		}
	}
	for _, dev := range []string{"sda1", "nvme0n1p2", "vdb"} {
		if !isRealDevice("/dev/" + dev) {
			t.Errorf("isRealDevice(%q) = false, want true", dev)
		}
	}
}
