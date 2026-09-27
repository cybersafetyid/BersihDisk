// Package sysinfo reports machine and application facts for the settings page.
package sysinfo

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Machine describes the computer the app runs on.
type Machine struct {
	OS          string `json:"os"`
	Arch        string `json:"arch"`
	CPUCores    int    `json:"cpuCores"`
	MemoryBytes uint64 `json:"memoryBytes"`
	Hostname    string `json:"hostname"`
	User        string `json:"user"`
	Home        string `json:"home"`
	GoVersion   string `json:"goVersion"`
}

// Host reads the current machine. Individual fields stay zero when the OS does
// not expose them, so one unreadable fact never fails the whole report.
func Host() Machine {
	m := Machine{
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
		CPUCores:  runtime.NumCPU(),
		GoVersion: runtime.Version(),
	}
	m.MemoryBytes = totalMemory()
	if h, err := os.Hostname(); err == nil {
		m.Hostname = h
	}
	if u, err := os.UserHomeDir(); err == nil {
		m.Home = u
		m.User = filepath.Base(u)
	}
	if name := os.Getenv("USER"); name != "" {
		m.User = name
	}
	return m
}

// AppLocation returns the bundle directory (macOS) or the executable path.
func AppLocation() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	if resolved, rerr := filepath.EvalSymlinks(exe); rerr == nil {
		exe = resolved
	}
	if i := strings.LastIndex(exe, ".app/"); i >= 0 {
		return exe[:i+4], nil
	}
	return exe, nil
}

// AppSizeOnDisk sums the files of the running application.
func AppSizeOnDisk() (path string, bytes int64, err error) {
	path, err = AppLocation()
	if err != nil {
		return "", 0, err
	}
	st, serr := os.Stat(path)
	if serr != nil {
		return path, 0, serr
	}
	if !st.IsDir() {
		return path, st.Size(), nil
	}
	err = filepath.WalkDir(path, func(_ string, d os.DirEntry, werr error) error {
		if werr != nil || d.IsDir() {
			return werr
		}
		if info, ierr := d.Info(); ierr == nil {
			bytes += info.Size()
		}
		return nil
	})
	return path, bytes, err
}

// ClearCache removes the scratch files the app itself created — update downloads
// in the OS temp directory — and reports what was freed.
func ClearCache() (files int, bytes int64, err error) {
	patterns := []string{
		filepath.Join(os.TempDir(), "bersihdisk-update-*"),
	}
	var firstErr error
	for _, p := range patterns {
		matches, merr := filepath.Glob(p)
		if merr != nil {
			firstErr = merr
			continue
		}
		for _, m := range matches {
			size := dirSize(m)
			if rerr := os.RemoveAll(m); rerr != nil {
				if firstErr == nil {
					firstErr = rerr
				}
				continue
			}
			files++
			bytes += size
		}
	}
	return files, bytes, firstErr
}

func dirSize(path string) int64 {
	var total int64
	st, err := os.Stat(path)
	if err != nil {
		return 0
	}
	if !st.IsDir() {
		return st.Size()
	}
	_ = filepath.WalkDir(path, func(_ string, d os.DirEntry, werr error) error {
		if werr != nil || d.IsDir() {
			return werr
		}
		if info, ierr := d.Info(); ierr == nil {
			total += info.Size()
		}
		return nil
	})
	return total
}
