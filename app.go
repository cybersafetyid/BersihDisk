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

	"bersihdisk/internal/analyzer"
	"bersihdisk/internal/appicon"
	"bersihdisk/internal/browser"
	"bersihdisk/internal/deleter"
	"bersihdisk/internal/drive"
	"bersihdisk/internal/reveal"
	"bersihdisk/internal/rules"
	"bersihdisk/internal/safety"
	"bersihdisk/internal/scanner"
	"bersihdisk/internal/sysinfo"
	"bersihdisk/internal/uninstall"
	"bersihdisk/internal/updater"
	"bersihdisk/internal/winstate"
)

// CategoryUI is one cleanup category. Its display text lives in the frontend
// locales, keyed by ID, so the same build can speak any language.
type CategoryUI struct {
	ID    string `json:"id"`
	Icon  string `json:"icon"`
	OptIn bool   `json:"optIn"`
	// Risk is the worst level any location of the category can reach; the picker
	// shows a warning for anything above "safe".
	Risk safety.Level `json:"risk"`
}

// DriveUI is a drive for the selection cards in the frontend.
type DriveUI struct {
	Name       string `json:"name"`
	MountPoint string `json:"mountPoint"`
	TotalBytes uint64 `json:"totalBytes"`
	FreeBytes  uint64 `json:"freeBytes"`
	UsedBytes  uint64 `json:"usedBytes"` // from Info.UsedBytes, clamped against underflow
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
	// Acknowledged is true when the user ticked "I understand" in the confirmation
	// dialog. Anything above "safe" is refused without it.
	Acknowledged bool `json:"acknowledged"`
}

// App is the main application struct bound to the frontend.
type App struct {
	ctx context.Context

	mu           sync.Mutex
	scanner      *scanner.Scanner
	analyzer     *analyzer.Analyzer
	deleter      *deleter.Deleter
	updateCancel context.CancelFunc
	uninstaller  *uninstall.Inventory
	uninstallRun context.CancelFunc

	// The frontend is not trusted with arbitrary paths: deletion is limited to the
	// last scan's items (and what lies inside them), and only the file this app
	// downloaded can be installed.
	win          winstate.State          // window state loaded at launch, updated on close
	scanItems    map[string]scanner.Item // last scan result by cleaned path
	analyzeItems map[string]scanner.Item // drive-analyzer entries seen this session, by cleaned path
	release      updater.Info            // last CheckUpdate result; the download URL and digest come from here
	updateFile   string
	mounts       []string                // detected drive mount points; analysis and deletion are confined to these
}

// isInside reports whether child is parent itself or lies below it.
func isInside(child, parent string) bool {
	rel, err := filepath.Rel(parent, child)
	if err != nil || filepath.IsAbs(rel) {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

// itemFor returns the known item that path is, or lies inside (the innermost
// one), and whether there is one. Both the cleaner's scan result and the drive
// analyzer's listings count: the analyzer's entries are what lets a folder seen
// in the disk-usage view be deleted through the same guarded path as a scan item.
func (a *App) itemFor(path string) (scanner.Item, bool) {
	path = filepath.Clean(path)
	a.mu.Lock()
	defer a.mu.Unlock()
	var best scanner.Item
	found := false
	consider := func(root string, it scanner.Item) {
		if isInside(path, root) && (!found || len(root) > len(best.Path)) {
			best, found = it, true
		}
	}
	for root, it := range a.scanItems {
		consider(root, it)
	}
	// The analyzer map is keyed by the path itself, so the key is the item's path.
	for root, it := range a.analyzeItems {
		consider(root, it)
	}
	return best, found
}

// deletable reports whether path belongs to the last scan result.
func (a *App) deletable(path string) bool {
	_, ok := a.itemFor(path)
	return ok
}

// isInsideOrEqual reports whether child is parent itself or lies below it.
func isInsideOrEqual(child, parent string) bool {
	return child == parent || isInside(child, parent)
}

// withinDrive reports whether path sits on (or under) one of the detected drive
// mount points. An empty mount set means detection has not run yet and is treated
// as "allowed", so a transient detection failure never wedges the UI.
func withinDrive(path string, mounts []string) bool {
	if len(mounts) == 0 {
		return true
	}
	path = filepath.Clean(path)
	for _, m := range mounts {
		if isInsideOrEqual(path, m) {
			return true
		}
	}
	return false
}

// knownMounts returns the cached mount points of the detected drives, filling the
// cache on first use. Analysis and deletion are confined to these, so the frontend
// cannot name an arbitrary location outside a real volume as a target.
func (a *App) knownMounts() []string {
	a.mu.Lock()
	m := a.mounts
	a.mu.Unlock()
	if len(m) == 0 {
		m = a.driveMounts()
		a.mu.Lock()
		a.mounts = m
		a.mu.Unlock()
	}
	return m
}

// driveMounts lists the mount points of the drives the app can currently see.
func (a *App) driveMounts() []string {
	list, err := drive.Detect()
	if err != nil {
		return nil
	}
	mounts := make([]string, 0, len(list))
	for _, d := range list {
		mounts = append(mounts, filepath.Clean(d.MountPoint))
	}
	return mounts
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
	mounts := make([]string, 0, len(list))
	for _, d := range list {
		out = append(out, DriveUI{
			Name:       d.Name,
			MountPoint: d.MountPoint,
			TotalBytes: d.TotalBytes,
			FreeBytes:  d.FreeBytes,
			UsedBytes:  d.UsedBytes(),
			Root:       d.Root,
			Removable:  d.Removable,
		})
		mounts = append(mounts, filepath.Clean(d.MountPoint))
	}
	a.mu.Lock()
	a.mounts = mounts
	a.mu.Unlock()
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
			Risk:  r.RiskLevel(),
		})
	}
	return out
}

// ListFolder returns the children of a folder with their sizes, so individual
// items inside a scan result can be chosen for deletion.
func (a *App) ListFolder(path string) ([]browser.Entry, error) {
	return browser.List(path)
}

// MeasurePaths returns the current size of each scan-result path, in the order
// given (0 for one that no longer exists), so the results page can refresh its
// numbers after a delete without a new scan. Paths outside the scan result are
// reported as 0.
func (a *App) MeasurePaths(paths []string) []int64 {
	out := make([]int64, len(paths))
	var ok []string
	var slots []int
	for i, p := range paths {
		if a.deletable(p) {
			ok = append(ok, p)
			slots = append(slots, i)
		}
	}
	for j, size := range scanner.Sizes(ok) {
		out[slots[j]] = size
	}
	return out
}

// Reveal opens a path in Finder, Explorer, or the default file manager.
func (a *App) Reveal(path string) error {
	return reveal.Open(path)
}

// StartScan runs an asynchronous scan; progress via the "scan:progress" event,
// result via "scan:finished".
func (a *App) StartScan(req ScanRequest) string {
	// The frontend is not trusted to name roots outside a real volume: keep only
	// those that lie on a detected drive.
	mounts := a.knownMounts()
	roots := make([]string, 0, len(req.Roots))
	for _, r := range req.Roots {
		if withinDrive(r, mounts) {
			roots = append(roots, r)
		}
	}
	if len(roots) == 0 {
		wailsruntime.EventsEmit(a.ctx, "scan:finished", scanner.Result{})
		return "denied"
	}

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
		res := s.Scan(roots, req.Categories)
		a.mu.Lock()
		current := a.scanner == s
		if current {
			a.scanner = nil
			a.scanItems = make(map[string]scanner.Item, len(res.Items))
			for _, it := range res.Items {
				a.scanItems[filepath.Clean(it.Path)] = it
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

// AnalyzeOutcome is sent on "analyze:finished": the listing, or why it failed.
type AnalyzeOutcome struct {
	analyzer.Result
	Error string `json:"error,omitempty"`
}

// StartAnalyze lists one folder's children with their sizes and risk grades, for
// the drive-usage view. Progress arrives on "analyze:progress" and the result on
// "analyze:finished". It replaces any listing still running.
//
// Entries are remembered for the session, so a folder shown here can be deleted
// through StartDelete — which still re-checks the risk before anything is removed.
func (a *App) StartAnalyze(path string) string {
	// Refuse to analyze (and therefore offer for deletion) any location outside a
	// detected drive — the frontend cannot pick an arbitrary workspace.
	if !withinDrive(path, a.knownMounts()) {
		wailsruntime.EventsEmit(a.ctx, "analyze:finished", AnalyzeOutcome{
			Result: analyzer.Result{Path: filepath.Clean(path)},
			Error:  "path is outside a detected drive",
		})
		return "denied"
	}
	a.mu.Lock()
	if a.analyzer != nil {
		a.analyzer.Cancel()
	}
	an := analyzer.New(func(p analyzer.Progress) {
		wailsruntime.EventsEmit(a.ctx, "analyze:progress", p)
	})
	a.analyzer = an
	a.mu.Unlock()

	go func() {
		res, err := an.List(path)
		a.mu.Lock()
		current := a.analyzer == an
		if current {
			a.analyzer = nil
			if a.analyzeItems == nil {
				a.analyzeItems = make(map[string]scanner.Item)
			}
			for _, e := range res.Entries {
				a.analyzeItems[filepath.Clean(e.Path)] = scanner.Item{
					Path:    e.Path,
					Size:    e.Size,
					Level:   e.Level,
					Reasons: e.Reasons,
				}
			}
		}
		a.mu.Unlock()
		// A listing replaced by a newer one is stale; its result must not reach the UI.
		if !current {
			return
		}
		out := AnalyzeOutcome{Result: res}
		if err != nil {
			out.Error = err.Error()
		}
		wailsruntime.EventsEmit(a.ctx, "analyze:finished", out)
	}()
	return "ok"
}

// CancelAnalyze aborts the running folder analysis.
func (a *App) CancelAnalyze() {
	a.mu.Lock()
	an := a.analyzer
	a.mu.Unlock()
	if an != nil {
		an.Cancel()
	}
}

// vetDelete decides what of a delete request may run: the items to hand to the
// deleter, the paths refused with why, and the mode to use.
func (a *App) vetDelete(req DeleteRequest) (items []deleter.Item, denied []deleter.Failure, mode string) {
	worst := safety.Safe
	mounts := a.knownMounts()
	for i, p := range req.Paths {
		if !withinDrive(p, mounts) {
			denied = append(denied, deleter.Failure{Path: p, Message: "outside a detected drive"})
			continue
		}
		it, ok := a.itemFor(p)
		switch {
		case !ok:
			denied = append(denied, deleter.Failure{Path: p, Message: "not part of the scan result"})
			continue
		case it.Level == safety.Blocked:
			denied = append(denied, deleter.Failure{Path: p, Message: "protected location (" + strings.Join(it.Reasons, ", ") + ")"})
			continue
		}
		worst = safety.Worse(worst, it.Level)
		var size int64
		if i < len(req.Sizes) {
			size = req.Sizes[i]
		}
		// KeepRoot applies to the container itself, not to a child chosen inside it.
		items = append(items, deleter.Item{Path: p, Size: size, KeepRoot: it.KeepRoot && filepath.Clean(p) == filepath.Clean(it.Path)})
	}
	mode = req.Mode
	if worst == safety.Danger {
		mode = deleter.ModeTrash
	}
	if worst.NeedsAck() && !req.Acknowledged {
		for _, it := range items {
			denied = append(denied, deleter.Failure{Path: it.Path, Message: "the risk was not acknowledged"})
		}
		items = nil
	}
	return items, denied, mode
}

// StartDelete runs an asynchronous deletion; progress via the "delete:progress"
// event, result via "delete:finished".
//
// The frontend is not trusted to have read the warnings: every path is checked
// against the scan result and its assessment. A blocked location is refused, and
// anything above "safe" is refused unless the user acknowledged the risk. Danger
// items can only go to the Trash, whatever mode was asked for.
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
		items, denied, mode := a.vetDelete(req)
		res := deleter.Result{Failures: []deleter.Failure{}}
		if len(items) > 0 {
			res = d.Delete(items, mode)
		}
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

// ---- uninstaller ----

// inventory builds the uninstaller on first use.
func (a *App) inventory() *uninstall.Inventory {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.uninstaller == nil {
		a.uninstaller = uninstall.New(
			uninstall.Host{Env: uninstall.NewEnv(), Run: &uninstall.ExecRunner{}},
			uninstall.DefaultProviders()...,
		)
	}
	return a.uninstaller
}

// ListPackages returns every installed app, runtime and package the app knows how
// to remove. It runs the package managers, so it can take a few seconds.
func (a *App) ListPackages() uninstall.ListResult {
	return a.inventory().List(a.ctx)
}

// AppIcon returns an application's icon as a data URL, or "" when it has none the
// app can read. The list asks for it lazily, row by row, as rows scroll into view.
func (a *App) AppIcon(packageID string) string {
	url, err := a.inventory().Icon(a.ctx, packageID)
	if err != nil {
		return ""
	}
	return url
}

// PlanUninstall works out what removing one package would do, without doing it.
func (a *App) PlanUninstall(packageID string) (uninstall.Plan, error) {
	return a.inventory().Plan(a.ctx, packageID)
}

// StartUninstall runs the chosen steps of a plan; progress via "uninstall:progress",
// result via "uninstall:finished". Only steps of a plan this app produced can run.
func (a *App) StartUninstall(req uninstall.Request) string {
	ctx, cancel := context.WithCancel(context.Background())
	a.mu.Lock()
	if a.uninstallRun != nil {
		a.uninstallRun()
	}
	a.uninstallRun = cancel
	a.mu.Unlock()

	go func() {
		defer cancel()
		res, err := a.inventory().Execute(ctx, req, func(p uninstall.Progress) {
			wailsruntime.EventsEmit(a.ctx, "uninstall:progress", p)
		})
		out := UninstallOutcome{Result: res}
		if err != nil {
			out.Error = err.Error()
		}
		wailsruntime.EventsEmit(a.ctx, "uninstall:finished", out)
	}()
	return "ok"
}

// CancelUninstall stops the running uninstall after the current step.
func (a *App) CancelUninstall() {
	a.mu.Lock()
	cancel := a.uninstallRun
	a.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// UninstallOutcome is sent on "uninstall:finished": the result, or why the
// request was refused.
type UninstallOutcome struct {
	uninstall.Result
	Error string `json:"error,omitempty"`
}

// OpenLink hands a web or mail address to the user's default handler.
func (a *App) OpenLink(rawURL string) error {
	return reveal.OpenURL(rawURL)
}
