package sysinfo

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestHostReportsTheBasics(t *testing.T) {
	m := Host()
	if m.OS != runtime.GOOS || m.Arch != runtime.GOARCH {
		t.Errorf("Host() = %s/%s, want %s/%s", m.OS, m.Arch, runtime.GOOS, runtime.GOARCH)
	}
	if m.CPUCores < 1 {
		t.Errorf("CPUCores = %d, want at least 1", m.CPUCores)
	}
	if m.Home == "" {
		t.Error("Home must be set for the current user")
	}
	if m.GoVersion == "" {
		t.Error("GoVersion must be set")
	}
}

func TestAppLocationAndSize(t *testing.T) {
	path, bytes, err := AppSizeOnDisk()
	if err != nil {
		t.Fatal(err)
	}
	if path == "" || bytes <= 0 {
		t.Errorf("AppSizeOnDisk() = %q, %d — the test binary itself has size", path, bytes)
	}
}

func TestClearCacheOnlyRemovesAppScratch(t *testing.T) {
	mine := filepath.Join(os.TempDir(), "bersihdisk-update-test-1")
	foreign := filepath.Join(os.TempDir(), "someone-elses-scratch-test")
	if err := os.MkdirAll(mine, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mine, "pkg.bin"), make([]byte, 4096), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(foreign, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		os.RemoveAll(mine)
		os.RemoveAll(foreign)
	})

	files, bytes, err := ClearCache()
	if err != nil {
		t.Fatal(err)
	}
	if files == 0 || bytes < 4096 {
		t.Errorf("ClearCache() = %d files, %d bytes; want at least the 4096 byte scratch file", files, bytes)
	}
	if _, err := os.Stat(mine); err == nil {
		t.Error("the app scratch directory must be gone")
	}
	if _, err := os.Stat(foreign); err != nil {
		t.Error("ClearCache must not touch anything outside its own prefix")
	}
}
