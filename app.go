// app.go — binding layer between the Go backend and the React frontend via Wails.
package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

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
	"bersihdisk/internal/winstate"
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

	// The frontend is not trusted with arbitrary paths: deletion is limited to the
	// last scan's items (and what lies inside them), and only the file this app
	// downloaded can be installed.
	win        winstate.State // window state loaded at launch, updated on close
	scanPaths  []string
	release    updater.Info // last CheckUpdate result; the download URL and digest come from here
	updateFile string
}

// isInside reports whether child is parent itself or lies below it.
func isInside(child, parent string) bool {
	rel, err := filepath.Rel(parent, child)
	if err != nil || filepath.IsAbs(rel) {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

// deletable reports whether path belongs to the last scan result.
func (a *App) deletable(path string) bool {
	path = filepath.Clean(path)
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, root := range a.scanPaths {
		if isInside(path, root) {
			return true
		}
	}
	return false
}

// NewApp creates the App instance.
func NewApp() *App {
	return &App{}
}

// startup is called by Wails when the app starts.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	// Wails has no start position option, so a saved position is applied here — only
	// when it still lies on the primary screen (a monitor may have been unplugged).
	if a.win.Maximised || a.win.Fullscreen {
		return
	}
	if screens, err := wailsruntime.ScreenGetAll(ctx); err == nil {
		for _, s := range screens {
			if s.IsPrimary && a.win.PositionOnScreen(s.Width, s.Height) {
				wailsruntime.WindowSetPosition(ctx, a.win.X, a.win.Y)
			}
		}
	}
}

// beforeClose stores the window state so the next launch reopens it as it was.
func (a *App) beforeClose(ctx context.Context) bool {
	w, h := wailsruntime.WindowGetSize(ctx)
	x, y := wailsruntime.WindowGetPosition(ctx)
	a.win = winstate.Merge(a.win, w, h, x, y,
		wailsruntime.WindowIsMaximised(ctx), wailsruntime.WindowIsFullscreen(ctx))
	if err := winstate.Save(a.win); err != nil {
		wailsruntime.LogErrorf(ctx, "could not save window state: %v", err)
	}
	return false // let the window close
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
		current := a.scanner == s
		if current {
			a.scanner = nil
			a.scanPaths = a.scanPaths[:0]
			for _, it := range res.Items {
				a.scanPaths = append(a.scanPaths, filepath.Clean(it.Path))
			}
		}
		a.mu.Unlock()
		// A scan replaced by a newer one is stale; its result must not reach the UI.
		if current {
			wailsruntime.EventsEmit(a.ctx, "scan:finished", res)
		}
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
		var denied []deleter.Failure
		for i, p := range req.Paths {
			if !a.deletable(p) {
				denied = append(denied, deleter.Failure{Path: p, Message: "not part of the scan result"})
				continue
			}
			var size int64
			if i < len(req.Sizes) {
				size = req.Sizes[i]
			}
			items = append(items, deleter.Item{Path: p, Size: size})
		}
		res := d.Delete(items, req.Mode)
		res.Failures = append(res.Failures, denied...)
		res.Failed += len(denied)
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
	info, err := updater.Check(a.ctx, releaseRepo, appVersion)
	if err == nil {
		a.mu.Lock()
		a.release = info
		a.mu.Unlock()
	}
	return info, err
}

// StartUpdateDownload fetches the release asset asynchronously; progress arrives
// on the "update:progress" event and the result on "update:finished".
//
// Only the asset that CheckUpdate returned can be fetched, and its digest is the
// one from that check — the arguments are not trusted to choose either.
func (a *App) StartUpdateDownload(rawURL, _ string) string {
	a.mu.Lock()
	if rawURL == "" || rawURL != a.release.URL {
		a.mu.Unlock()
		wailsruntime.EventsEmit(a.ctx, "update:finished", UpdateDownload{Error: "download is not the release found by the update check"})
		return "denied"
	}
	digest := a.release.Digest
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
		if err == nil {
			a.updateFile = path
		}
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
	a.mu.Lock()
	want := a.updateFile
	a.mu.Unlock()
	if want == "" || filepath.Clean(path) != want {
		return fmt.Errorf("only the downloaded update can be installed")
	}
	if err := reveal.Launch(want); err != nil {
		return err
	}
	// The running app cannot be replaced: Windows locks its executable, and Finder
	// refuses to overwrite a running bundle in /Applications ("the item is in use").
	// So quit once the installer / disk image is open. Linux's package manager can
	// swap the binary underneath a running process, so it stays open there.
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		go func() {
			time.Sleep(1500 * time.Millisecond) // let the installer window appear first
			wailsruntime.Quit(a.ctx)
		}()
	}
	return nil
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
