//go:build !windows

package drive

import (
	"os"
	"testing"
)

func mountPoints(in []Info) []string {
	out := make([]string, len(in))
	for i, d := range in {
		out[i] = d.MountPoint
	}
	return out
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// A mount-point directory left behind by an unmounted disk still reports the
// parent volume's statfs identity, so it must not be listed twice.
func TestUniqueVolumesDropsSameFilesystemAndKeepsCanonical(t *testing.T) {
	in := []Info{
		{MountPoint: "/", Name: "Macintosh HD", Root: true, fsKey: "16777233:0"},
		{MountPoint: "/Volumes/Data", Name: "Data", fsKey: "16777242:0"},
		{MountPoint: "/Volumes/ExternalSSD", Name: "ExternalSSD", fsKey: "16777233:0"},
		{MountPoint: "/Volumes/Gorby", Name: "Gorby", fsKey: "16777245:0"},
	}

	got := mountPoints(uniqueVolumes(in))
	want := []string{"/", "/Volumes/Data", "/Volumes/Gorby"}
	if !equal(got, want) {
		t.Fatalf("uniqueVolumes() = %v, want %v", got, want)
	}
}

// Dedupe keeps the first entry, so it has to run after the root is sorted first.
func TestUniqueVolumesAfterSortByRootKeepsSystemDrive(t *testing.T) {
	in := []Info{
		{MountPoint: "/Volumes/ExternalSSD", Name: "ExternalSSD", fsKey: "16777233:0"},
		{MountPoint: "/", Name: "Macintosh HD", Root: true, fsKey: "16777233:0"},
	}
	sortByRoot(in)

	got := mountPoints(uniqueVolumes(in))
	want := []string{"/"}
	if !equal(got, want) {
		t.Fatalf("uniqueVolumes(sortByRoot(...)) = %v, want %v", got, want)
	}
}

// Unknown identity must never collapse distinct drives into one.
func TestUniqueVolumesKeepsEntriesWithoutIdentity(t *testing.T) {
	in := []Info{
		{MountPoint: "/", Name: "System", Root: true},
		{MountPoint: "/mnt/other", Name: "Other"},
	}

	got := mountPoints(uniqueVolumes(in))
	want := []string{"/", "/mnt/other"}
	if !equal(got, want) {
		t.Fatalf("uniqueVolumes() = %v, want %v", got, want)
	}
}

// A leftover directory on an already-listed volume is not a mount point, so it
// never reaches the statfs call that would report the parent's capacities.
func TestIsMountPointRejectsPlainDirectory(t *testing.T) {
	dir := t.TempDir()
	sub := dir + "/ExternalSSD"
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if isMountPoint(sub) {
		t.Fatalf("isMountPoint(%q) = true, want false for a plain directory", sub)
	}
}
