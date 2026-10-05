// Package analyzer measures what actually takes up space on a drive, one folder
// at a time, so the UI can draw a DaisyDisk-style breakdown of the largest items.
//
// It is deliberately read-only: it never deletes anything. Every entry is graded
// with the same safety rules the cleaner uses, so a system folder, a home folder
// or a credentials directory is flagged (or blocked) before it can be selected.
//
// Sizes are the recursive apparent size of each child, measured with the
// scanner's shared pool of walkers — one pass over a folder's subtree instead of
// one walk per child. Symlinks are never followed (that could leave the drive or
// loop), the walk can be cancelled, and virtual/pseudo filesystems and the
// operating system's own trash folders are skipped so a whole-drive scan stays
// stable on every platform.
package analyzer

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"bersihdisk/internal/safety"
	"bersihdisk/internal/scanner"
)

// Entry is one child of an analyzed directory.
type Entry struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	IsDir  bool   `json:"isDir"`
	IsLink bool   `json:"isLink"` // a symlink: deleting it frees nothing
	Size   int64  `json:"size"`   // bytes; for a directory, its whole subtree
	// Level and Reasons are how risky deleting this entry is, decided by the same
	// rules as the cleaner. A blocked entry is shown but can never be deleted.
	Level   safety.Level `json:"level"`
	Reasons []string     `json:"reasons,omitempty"`
	// Skipped marks a directory that was intentionally not walked (virtual or
	// pseudo filesystem, or the OS trash); its size is unknown and reported as 0.
	Skipped bool `json:"skipped,omitempty"`
}

// Result is the analyzed listing of one directory.
type Result struct {
	Path       string  `json:"path"`
	TotalBytes int64   `json:"totalBytes"`
	Entries    []Entry `json:"entries"`
	Files      int     `json:"files"`
	Dirs       int     `json:"dirs"`
	Skipped    int     `json:"skipped"`
	Partial    bool    `json:"partial"` // true if the listing was cancelled
}

// Progress is an analysis progress event sent to the UI.
type Progress struct {
	Path  string `json:"path"`  // directory being measured right now
	Dirs  int    `json:"dirs"`  // directories read so far
	Total int    `json:"total"` // children being measured
}

// Analyzer lists folders with cancellation and progress events.
type Analyzer struct {
	emit func(Progress)

	emitMu  sync.Mutex // serialises events so the UI sees one ordered stream
	stop    chan struct{}
	stopOne sync.Once
	closed  atomic.Bool

	dirs    atomic.Int64
	curPath atomic.Value // string
}

// New creates an analyzer with an emit callback for progress events.
func New(emit func(Progress)) *Analyzer {
	a := &Analyzer{emit: emit, stop: make(chan struct{})}
	a.curPath.Store("")
	return a
}

// Cancel aborts a running listing. Walkers stop at the next directory boundary.
func (a *Analyzer) Cancel() {
	a.stopOne.Do(func() {
		a.closed.Store(true)
		close(a.stop)
	})
}

func (a *Analyzer) isClosed() bool { return a.closed.Load() }

// progress delivers one event, serialised against the heartbeat.
func (a *Analyzer) progress(p Progress) {
	a.emitMu.Lock()
	defer a.emitMu.Unlock()
	a.emit(p)
}

// heartbeat publishes the live counters while the walk runs, so the UI shows
// movement instead of sitting on the last number until the folder is done.
func (a *Analyzer) heartbeat(every time.Duration) func() {
	done := make(chan struct{})
	go func() {
		t := time.NewTicker(every)
		defer t.Stop()
		for {
			select {
			case <-done:
				return
			case <-t.C:
				a.progress(Progress{
					Path: a.curPath.Load().(string),
					Dirs: int(a.dirs.Load()),
				})
			}
		}
	}()
	return func() { close(done) }
}

// skipAnywhere are directory names never entered during an analysis, wherever
// they appear: recycle bins and the metadata folders an operating system keeps
// inside every volume.
var skipAnywhere = map[string]struct{}{
	".trash": {}, ".trashes": {}, "$recycle.bin": {}, "system volume information": {},
	".spotlight-v100": {}, ".fseventsd": {}, ".documentrevisions-v100": {},
	".temporaryitems": {}, ".vol": {}, "lost+found": {},
}

// skipAtRoot are directory names skipped only directly under a filesystem root.
// Elsewhere they are ordinary folders (~/dev/project must still be analyzed).
var skipAtRoot = map[string]struct{}{
	"proc": {}, "sys": {}, "dev": {}, "run": {}, "boot": {},
}

// skipDir reports whether dir must not be walked. path is already joined.
func skipDir(path string) bool {
	name := strings.ToLower(filepath.Base(path))
	if _, ok := skipAnywhere[name]; ok {
		return true
	}
	if _, ok := skipAtRoot[name]; !ok {
		return false
	}
	parent := filepath.Dir(path)
	return filepath.Dir(parent) == parent // parent is a filesystem root
}

// List reads dir once and returns its children, largest first, each with its
// recursive size and its deletion risk. It is safe to call repeatedly while
// drilling into a drive; Cancel() aborts the walk in progress.
//
// A directory that cannot be read is reported as an error only when it is the
// root of the listing. Children that cannot be read simply measure as 0, which is
// what a permission-restricted system folder does.
func (a *Analyzer) List(dir string) (Result, error) {
	dir = filepath.Clean(dir)
	ents, err := os.ReadDir(dir)
	if err != nil {
		return Result{Path: dir}, err
	}

	res := Result{Path: dir, Entries: make([]Entry, 0, len(ents))}
	var dirPaths []string
	var dirSlots []int
	for _, de := range ents {
		name := de.Name()
		path := filepath.Join(dir, name)
		e := Entry{Name: name, Path: path}

		// de.Type() comes from the directory read itself; a symlink is not
		// followed, so deleting it frees nothing and we must not measure through it.
		if de.Type()&os.ModeSymlink != 0 {
			e.IsLink = true
		} else if de.IsDir() {
			e.IsDir = true
		}

		if e.IsDir {
			if skipDir(path) {
				e.Skipped = true
				res.Skipped++
			} else {
				dirPaths = append(dirPaths, path)
				dirSlots = append(dirSlots, len(res.Entries))
				res.Dirs++
			}
			res.Entries = append(res.Entries, e)
			continue
		}

		if !e.IsLink {
			if info, ierr := de.Info(); ierr == nil {
				e.Size = scanner.FileBytes(info)
			}
		}
		res.Entries = append(res.Entries, e)
		res.Files++
	}

	if len(dirPaths) > 0 {
		a.dirs.Store(0)
		a.curPath.Store("")
		stopHeartbeat := a.heartbeat(200 * time.Millisecond)
		sizes := scanner.MeasureTree(dirPaths, a.stop, func(d string) {
			a.dirs.Add(1)
			a.curPath.Store(d)
		})
		stopHeartbeat()
		for i, slot := range dirSlots {
			res.Entries[slot].Size = sizes[i]
		}
	}

	// Grade every entry through one batch, so listing a directory with thousands
	// of children does not rebuild the protected-path set for each of them.
	paths := make([]string, len(res.Entries))
	for i := range res.Entries {
		paths[i] = res.Entries[i].Path
	}
	assessments := safety.AssessBatch(paths)
	for i := range res.Entries {
		res.Entries[i].Level = assessments[i].Level
		res.Entries[i].Reasons = assessments[i].Reasons
		if res.Entries[i].Level == "" {
			res.Entries[i].Level = safety.Safe
		}
		res.TotalBytes += res.Entries[i].Size
	}

	sort.Slice(res.Entries, func(i, j int) bool {
		if res.Entries[i].Size != res.Entries[j].Size {
			return res.Entries[i].Size > res.Entries[j].Size
		}
		return res.Entries[i].Name < res.Entries[j].Name
	})

	res.Partial = a.isClosed()
	return res, nil
}
