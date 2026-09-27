// app.go — binding layer between the Go backend and the React frontend via Wails.
package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"runtime"
	"sync"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"bersihdisk/internal/appicon"
	"bersihdisk/internal/browser"
	"bersihdisk/internal/deleter"
	"bersihdisk/internal/drive"
	"bersihdisk/internal/reveal"
	"bersihdisk/internal/rules"
	"bersihdisk/internal/scanner"
	"bersihdisk/internal/sysinfo"
	"bersihdisk/internal/updater"
)

// CategoryUI is one cleanup category. Its display text lives in the frontend
// locales, keyed by ID, so the same build can speak any language.
type CategoryUI struct {
	ID    string `json:"id"`
	Icon  string `json:"icon"`
	OptIn bool   `json:"optIn"`
}

// DriveUI is a drive for the selection cards in the frontend.
type DriveUI struct {
	Name       string `json:"name"`
	MountPoint string `json:"mountPoint"`
	TotalBytes uint64 `json:"totalBytes"`
	FreeBytes  uint64 `json:"freeBytes"`
	Root       bool   `json:"root"`
	Removable  bool   `json:"removable"`
}

// ScanRequest holds the scan request parameters from the UI.
type ScanRequest struct {
	Roots      []string `json:"roots"`
	Categories []string `json:"categories"`
}

// DeleteRequest holds the deletion parameters from the UI.
type DeleteRequest struct {
	Paths []string `json:"paths"`
	Sizes []int64  `json:"sizes"`
	Mode  string   `json:"mode"`
}

// App is the main application struct bound to the frontend.
type App struct {
	ctx context.Context

	mu           sync.Mutex
	scanner      *scanner.Scanner
	deleter      *deleter.Deleter
	updateCancel context.CancelFunc
}

// NewApp creates the App instance.
func NewApp() *App {
	return &App{}
}

// startup is called by Wails when the app starts.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// DetectDrives returns the list of mounted drives.
func (a *App) DetectDrives() []DriveUI {
	list, err := drive.Detect()
	if err != nil {
		wailsruntime.LogErrorf(a.ctx, "drive detection failed: %v", err)
		return []DriveUI{}
	}
	out := make([]DriveUI, 0, len(list))
	for _, d := range list {
		out = append(out, DriveUI{
			Name:       d.Name,
			MountPoint: d.MountPoint,
			TotalBytes: d.TotalBytes,
			FreeBytes:  d.FreeBytes,
			Root:       d.Root,
			Removable:  d.Removable,
		})
	}
	return out
}

// ListCategories returns the cleanup categories for the UI.
func (a *App) ListCategories() []CategoryUI {
	all := rules.All()
	out := make([]CategoryUI, 0, len(all))
	for _, r := range all {
		out = append(out, CategoryUI{
			ID:    r.ID,
			Icon:  r.Icon,
			OptIn: r.OptIn,
		})
	}
	return out
}

// ListFolder returns the children of a folder with their sizes, so individual
// items inside a scan result can be chosen for deletion.
func (a *App) ListFolder(path string) ([]browser.Entry, error) {
	return browser.List(path)
}

// Reveal opens a path in Finder, Explorer, or the default file manager.
func (a *App) Reveal(path string) error {
	return reveal.Open(path)
}

// StartScan runs an asynchronous scan; progress via the "scan:progress" event,
// result via "scan:finished".
func (a *App) StartScan(req ScanRequest) string {
	a.mu.Lock()
	if a.scanner != nil {
		a.scanner.Cancel()
	}
	s := scanner.New(func(p scanner.Progress) {
		wailsruntime.EventsEmit(a.ctx, "scan:progress", p)
	})
	a.scanner = s
	a.mu.Unlock()

	go func() {
		res := s.Scan(req.Roots, req.Categories)
		a.mu.Lock()
		if a.scanner == s {
			a.scanner = nil
		}
		a.mu.Unlock()
		wailsruntime.EventsEmit(a.ctx, "scan:finished", res)
	}()
	return "ok"
}

// CancelScan aborts the running scan.
func (a *App) CancelScan() {
	a.mu.Lock()
	s := a.scanner
	a.mu.Unlock()
	if s != nil {
		s.Cancel()
	}
}

// StartDelete runs an asynchronous deletion; progress via the "delete:progress"
// event, result via "delete:finished".
func (a *App) StartDelete(req DeleteRequest) string {
	a.mu.Lock()
	if a.deleter != nil {
		a.deleter.Cancel()
	}
	d := deleter.New(func(p deleter.Progress) {
		wailsruntime.EventsEmit(a.ctx, "delete:progress", p)
	})
	a.deleter = d
	a.mu.Unlock()

	go func() {
		items := make([]deleter.Item, 0, len(req.Paths))
		for i, p := range req.Paths {
			var size int64
			if i < len(req.Sizes) {
				size = req.Sizes[i]
			}
			items = append(items, deleter.Item{Path: p, Size: size})
		}
		res := d.Delete(items, req.Mode)
		wailsruntime.EventsEmit(a.ctx, "delete:finished", res)
	}()
	return "ok"
}

// CancelDelete aborts the running deletion.
func (a *App) CancelDelete() {
	a.mu.Lock()
	d := a.deleter
	a.mu.Unlock()
	if d != nil {
		d.Cancel()
	}
}

// UpdateDownload is the outcome of a download, sent on the "update:finished"
// event together with the local file to install.
type UpdateDownload struct {
	Path  string `json:"path"`
	Error string `json:"error,omitempty"`
}

// CheckUpdate looks for a newer GitHub release of releaseRepo.
func (a *App) CheckUpdate() (updater.Info, error) {
	return updater.Check(a.ctx, releaseRepo, appVersion)
}

// StartUpdateDownload fetches the release asset asynchronously; progress arrives
// on the "update:progress" event and the result on "update:finished".
func (a *App) StartUpdateDownload(rawURL, digest string) string {
	a.mu.Lock()
	if a.updateCancel != nil {
		a.updateCancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.updateCancel = cancel
	a.mu.Unlock()

	go func() {
		path, err := updater.Download(ctx, rawURL, digest, func(p updater.Progress) {
			wailsruntime.EventsEmit(a.ctx, "update:progress", p)
		})
		a.mu.Lock()
		a.updateCancel = nil
		a.mu.Unlock()

		res := UpdateDownload{Path: path}
		if err != nil {
			res.Error = err.Error()
		}
		wailsruntime.EventsEmit(a.ctx, "update:finished", res)
	}()
	return "ok"
}

// CancelUpdate aborts a download in progress.
func (a *App) CancelUpdate() {
	a.mu.Lock()
	cancel := a.updateCancel
	a.updateCancel = nil
	a.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// InstallUpdate opens the downloaded build so the OS can install it.
func (a *App) InstallUpdate(path string) error {
	return reveal.Launch(path)
}

// AppInfo returns app metadata for the About dialog.
func (a *App) AppInfo() map[string]interface{} {
	return map[string]interface{}{
		"name":    "BersihDisk",
		"version": appVersion,
		"trash":   deleter.TrashSupported(),
		"os":      fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}

// SystemInfo reports the machine the app runs on, for the settings page.
func (a *App) SystemInfo() sysinfo.Machine {
	return sysinfo.Host()
}

// AppInstall describes the running application on disk.
type AppInstall struct {
	Version string `json:"version"`
	Path    string `json:"path"`
	Bytes   int64  `json:"bytes"`
}

// AppDetails returns where the app lives and how large it is on disk.
func (a *App) AppDetails() (AppInstall, error) {
	path, bytes, err := sysinfo.AppSizeOnDisk()
	return AppInstall{Version: appVersion, Path: path, Bytes: bytes}, err
}

// CacheReport is what ClearCache removed.
type CacheReport struct {
	Files int   `json:"files"`
	Bytes int64 `json:"bytes"`
}

// ClearCache removes the temporary files this app created (update downloads).
func (a *App) ClearCache() (CacheReport, error) {
	files, bytes, err := sysinfo.ClearCache()
	return CacheReport{Files: files, Bytes: bytes}, err
}

// AppIconSupported reports whether this OS can swap the app icon at runtime.
func (a *App) AppIconSupported() bool {
	return appicon.Supported()
}

// SetAppIcon applies a PNG (base64) to the running app's icon.
func (a *App) SetAppIcon(pngBase64 string) error {
	png, err := base64.StdEncoding.DecodeString(pngBase64)
	if err != nil {
		return fmt.Errorf("icon data is not valid base64: %w", err)
	}
	return appicon.Set(png)
}

// OpenLink hands a web or mail address to the user's default handler.
func (a *App) OpenLink(rawURL string) error {
	return reveal.OpenURL(rawURL)
}
