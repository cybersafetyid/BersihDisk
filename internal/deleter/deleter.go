// Package deleter executes bulk deletion with Trash (via go-trash) or
// Permanent (os.RemoveAll) modes.
package deleter

import (
	"os"
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

// Delete runs the bulk deletion in the given mode.
func (d *Deleter) Delete(items []Item, mode string) Result {
	if mode != ModePermanent {
		mode = ModeTrash
	}
	start := time.Now()
	res := Result{Failures: []Failure{}}

	// Deduplicate paths.
	sizes := map[string]int64{}
	order := make([]string, 0, len(items))
	for _, it := range items {
		if _, exists := sizes[it.Path]; !exists {
			order = append(order, it.Path)
		}
		sizes[it.Path] = it.Size
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

			var err error
			if mode == ModePermanent {
				err = os.RemoveAll(path)
			} else {
				_, err = trash.MoveToTrash(path)
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
			mu.Unlock()

			d.emit(Progress{Done: done, Total: len(order), Bytes: bytesFree.Load(), Path: path})
		}(path, sizes[path])
	}
	wg.Wait()

	res.OK = int(okCount.Load())
	res.Failed = int(failCount.Load())
	res.Bytes = bytesFree.Load()
	res.Duration = time.Since(start).Seconds()
	return res
}
