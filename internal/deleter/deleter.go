// Package deleter executes bulk deletion with Trash (via go-trash) or
// Permanent (os.RemoveAll) modes.
package deleter

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/laurent22/go-trash"
)

// Deletion modes.
const (
	ModeTrash     = "trash"
	ModePermanent = "permanent"
)

// Item is one path to delete with its size.
type Item struct {
	Path string `json:"path"`
	Size int64  `json:"size"` // bytes
}

// Progress is a deletion progress event sent to the UI.
type Progress struct {
	Done  int    `json:"done"` // items processed so far
	Total int    `json:"total"`
	Bytes int64  `json:"bytes"` // bytes freed so far
	Path  string `json:"path"`  // last processed path
}

// Failure records one failure.
type Failure struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

// Result summarizes the deletion.
type Result struct {
	OK       int       `json:"ok"`
	Failed   int       `json:"failed"`
	Bytes    int64     `json:"bytes"` // total bytes freed
	Failures []Failure `json:"failures"`
	Duration float64   `json:"duration"` // seconds
}

// Deleter executes deletions with cancel support.
type Deleter struct {
	emit   func(Progress)
	closed int32
}

// New creates a deleter with an emit callback.
func New(emit func(Progress)) *Deleter {
	return &Deleter{emit: emit}
}

// Cancel stops the process (unprocessed items are skipped).
func (d *Deleter) Cancel() {
	atomic.StoreInt32(&d.closed, 1)
}

func (d *Deleter) isClosed() bool {
	return atomic.LoadInt32(&d.closed) == 1
}

// TrashSupported reports whether the OS supports trashing.
func TrashSupported() bool {
	return trash.IsAvailable()
}

// isInside reports whether child sits strictly below parent.
func isInside(child, parent string) bool {
	rel, err := filepath.Rel(parent, child)
	if err != nil || rel == "." || rel == ".." || filepath.IsAbs(rel) {
		return false
	}
	return !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// Delete runs the bulk deletion in the given mode.
func (d *Deleter) Delete(items []Item, mode string) Result {
	if mode != ModePermanent {
		mode = ModeTrash
	}
	start := time.Now()
	res := Result{Failures: []Failure{}}

	// Deduplicate paths, then drop any path inside another selected path: the
	// parent removes it, so deleting both would race and count its bytes twice.
	sizes := map[string]int64{}
	unique := make([]string, 0, len(items))
	for _, it := range items {
		p := filepath.Clean(it.Path)
		if _, exists := sizes[p]; !exists {
			unique = append(unique, p)
		}
		sizes[p] = it.Size
	}
	order := make([]string, 0, len(unique))
	for _, p := range unique {
		nested := false
		for _, q := range unique {
			if q != p && isInside(p, q) {
				nested = true
				break
			}
		}
		if !nested {
			order = append(order, p)
		}
	}

	var (
		mu        sync.Mutex
		wg        sync.WaitGroup
		okCount   atomic.Int32
		failCount atomic.Int32
		bytesFree atomic.Int64
	)
	// Conservative parallelism so the disk and system trash API are not hammered.
	par := 4
	if mode == ModeTrash {
		par = 2
	}
	sem := make(chan struct{}, par)

	for _, path := range order {
		if d.isClosed() {
			break
		}
		wg.Add(1)
		go func(path string, size int64) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			// RemoveAll succeeds on a missing path; report it instead of
			// claiming bytes that were never there.
			_, err := os.Lstat(path)
			if err == nil {
				if mode == ModePermanent {
					err = os.RemoveAll(path)
				} else {
					_, err = trash.MoveToTrash(path)
				}
			}

			mu.Lock()
			if err == nil {
				okCount.Add(1)
				bytesFree.Add(size)
			} else {
				failCount.Add(1)
				res.Failures = append(res.Failures, Failure{Path: path, Message: err.Error()})
			}
			done := int(okCount.Load() + failCount.Load())
			// Emit under the lock so progress events reach the UI in order.
			d.emit(Progress{Done: done, Total: len(order), Bytes: bytesFree.Load(), Path: path})
			mu.Unlock()
		}(path, sizes[path])
	}
	wg.Wait()

	res.OK = int(okCount.Load())
	res.Failed = int(failCount.Load())
	res.Bytes = bytesFree.Load()
	res.Duration = time.Since(start).Seconds()
	return res
}
