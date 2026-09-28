// Package scanner walks drives looking for target directories per rules.
package scanner

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"bersihdisk/internal/rules"
	"bersihdisk/internal/safety"
)

// Item is one found target directory.
type Item struct {
	Path     string `json:"path"`
	Size     int64  `json:"size"`     // bytes
	Category string `json:"category"` // category ID
	// LinkFrom is set when the category's path is a symlink and this item is the
	// real directory behind it.
	LinkFrom string `json:"linkFrom,omitempty"`
	// NestedIn is the path of another item that contains this one, meaning its
	// space is already counted there.
	NestedIn string `json:"nestedIn,omitempty"`
	// Level says how risky deleting the item is; Reasons are the codes behind it.
	// A blocked item is shown but can never be deleted or counted as freeable.
	Level   safety.Level `json:"level"`
	Reasons []string     `json:"reasons,omitempty"`
	// KeepRoot means deleting empties the folder instead of removing it.
	KeepRoot bool `json:"keepRoot,omitempty"`
}

// Progress is a scan progress event sent to the UI.
type Progress struct {
	Phase       string `json:"phase"`       // "search" | "measure"
	Path        string `json:"path"`        // directory being processed right now
	ItemsFound  int    `json:"itemsFound"`  // search: candidates found; measure: candidates measured
	TotalItems  int    `json:"totalItems"`  // candidates to measure (measure phase)
	BytesSoFar  int64  `json:"bytesSoFar"`  // total bytes measured so far
	DirsVisited int    `json:"dirsVisited"` // directories read in the current phase
}

// Result is the final scan result.
type Result struct {
	Items       []Item `json:"items"`
	TotalBytes  int64  `json:"totalBytes"`
	ItemCount   int    `json:"itemCount"`
	DirsSkipped int    `json:"dirsSkipped"`
	Partial     bool   `json:"partial"` // true if the scan was cancelled
}

// candidate is a phase-1 hit: target directory, not yet measured. linkFrom holds
// the symlink a home path was resolved through, empty for a plain directory.
type candidate struct {
	path     string
	category string
	linkFrom string
}

// Scanner runs a scan with cancel and progress events.
type Scanner struct {
	emit func(Progress)

	emitMu  sync.Mutex // serialises events so the UI sees one ordered stream
	stop    chan struct{}
	stopOne sync.Once
	closed  atomic.Bool

	dirsVisited atomic.Int64
	curPath     atomic.Value // string
}

// New creates a scanner with an emit callback for progress events.
func New(emit func(Progress)) *Scanner {
	s := &Scanner{emit: emit, stop: make(chan struct{})}
	s.curPath.Store("")
	return s
}

// isClosed reports whether the scan was cancelled.
func (s *Scanner) isClosed() bool {
	return s.closed.Load()
}

// Cancel aborts a running scan. Walkers stop at the next directory boundary.
func (s *Scanner) Cancel() {
	s.stopOne.Do(func() {
		s.closed.Store(true)
		close(s.stop)
	})
}

// progress delivers one event, serialised against the heartbeat so concurrent
// walkers cannot interleave partial updates on the frontend.
func (s *Scanner) progress(p Progress) {
	s.emitMu.Lock()
	defer s.emitMu.Unlock()
	s.emit(p)
}

// resetCounters starts a fresh per-phase activity counter.
func (s *Scanner) resetCounters() {
	s.dirsVisited.Store(0)
	s.curPath.Store("")
}

// heartbeat keeps publishing the live counters while a phase runs. Without it the
// UI would sit on the last candidate found and look frozen during the long
// stretches of pure directory reading, which is most of a scan.
func (s *Scanner) heartbeat(every time.Duration, phase string, total int, found func() int, bytes func() int64) func() {
	done := make(chan struct{})
	go func() {
		t := time.NewTicker(every)
		defer t.Stop()
		for {
			select {
			case <-done:
				return
			case <-t.C:
				s.progress(Progress{
					Phase:       phase,
					Path:        s.curPath.Load().(string),
					ItemsFound:  found(),
					TotalItems:  total,
					BytesSoFar:  bytes(),
					DirsVisited: int(s.dirsVisited.Load()),
				})
			}
		}
	}()
	return func() { close(done) }
}

// SkipList lists directory names never entered during a scan (protection).
var SkipList = []string{
	// macOS
	".Trash", ".Spotlight-V100", ".fseventsd", ".DocumentRevisions-V100",
	".PKInstallSandboxManager", ".PKInstallSandboxManagerSystemSoftware",
	"Volumes", "System", "private", "cores", "dev", "net",
	// Windows
	"$Recycle.Bin", "Windows", "Program Files", "Program Files (x86)",
	"ProgramData", "System Volume Information", "Recovery", "MSOCache",
	// Linux
	"proc", "sys", "run", "boot", "usr", "etc", "var", "snap", "srv",
	"lost+found",
}

// rootOnlySkip are SkipList names that are system folders only directly under a
// filesystem root ("/dev", "C:\Windows"). Elsewhere they are ordinary names —
// ~/dev/app/node_modules must still be found.
var rootOnlySkip = map[string]struct{}{
	"volumes": {}, "system": {}, "private": {}, "cores": {}, "dev": {}, "net": {},
	"windows": {}, "program files": {}, "program files (x86)": {}, "programdata": {},
	"recovery": {}, "msocache": {},
	"proc": {}, "sys": {}, "run": {}, "boot": {}, "usr": {}, "etc": {}, "var": {},
	"snap": {}, "srv": {},
}

// skipSet is the lowercased SkipList, built once so the walker does not rebuild
// it for every directory it passes.
var skipSet = func() map[string]struct{} {
	m := make(map[string]struct{}, len(SkipList))
	for _, s := range SkipList {
		m[strings.ToLower(s)] = struct{}{}
	}
	return m
}()

// skipMatch reports whether dir must not be entered. A root-only name counts only
// when its parent is a filesystem root; any other SkipList name counts anywhere.
func skipMatch(dir, lower string) bool {
	if _, ok := skipSet[lower]; !ok {
		return false
	}
	if _, rootOnly := rootOnlySkip[lower]; rootOnly {
		parent := filepath.Dir(dir)
		return filepath.Dir(parent) == parent
	}
	return true
}

// scanWorkers sizes the walker pool. A scan is syscall-bound (open + readdir +
// lstat per entry), so throughput comes from several reads being in flight at
// once — but measured on an 8-core machine, 16 and 24 walkers were slower than
// one per core, so the pool is capped there.
func scanWorkers() int {
	n := runtime.NumCPU()
	if n < 2 {
		n = 2
	}
	if n > 8 {
		n = 8
	}
	return n
}

// ruleHit is one (rule, pattern) pair the walker looks for. A pattern is usually
// a single directory name but can carry more segments ("app/build"), so the index
// is keyed on the last segment and the full pattern is verified against the path.
type ruleHit struct {
	ruleIdx int
	pattern string
	segs    int
}

// buildNameIndex maps lowercased final directory names to the rules searching for
// them. The walker tests this against every directory on the drive, so it replaces
// a per-directory loop over all rules and all of their patterns. Ties are broken
// by specificity first — "app/build" claims an app/build directory rather than
// leaving it to the looser "build" of another category — and then by rule order.
func buildNameIndex(active []rules.Rule) map[string][]ruleHit {
	idx := make(map[string][]ruleHit)
	for i := range active {
		for _, p := range active[i].DirNames() {
			segs := strings.Split(p, "/")
			key := strings.ToLower(segs[len(segs)-1])
			idx[key] = append(idx[key], ruleHit{ruleIdx: i, pattern: p, segs: len(segs)})
		}
	}
	for key := range idx {
		hits := idx[key]
		sort.SliceStable(hits, func(i, j int) bool { return hits[i].segs > hits[j].segs })
	}
	return idx
}

// patternMatches reports whether dir is the directory this rule's pattern points
// at, within the rule's depth limit. Single-segment patterns keep using
// Rule.HasDirName; multi-segment ones are compared against the trailing path
// segments, case-insensitively.
func patternMatches(r *rules.Rule, pattern, dir, name string, depth int) bool {
	segs := strings.Split(pattern, "/")
	if len(segs) == 1 {
		return r.HasDirName(name, depth)
	}
	if depth < 1 || depth > r.MaxDepth() {
		return false
	}
	rest := dir
	for i := len(segs) - 1; i >= 0; i-- {
		if !strings.EqualFold(filepath.Base(rest), segs[i]) {
			return false
		}
		parent := filepath.Dir(rest)
		if parent == rest {
			return false
		}
		rest = parent
	}
	return true
}

// isInside reports whether child sits strictly below parent.
func isInside(child, parent string) bool {
	rel, err := filepath.Rel(parent, child)
	if err != nil || rel == "." || rel == ".." || filepath.IsAbs(rel) {
		return false
	}
	return !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// markNested labels items contained in another item. Their space is already
// counted by the parent, which also removes them when deleted, so leaving them
// in the total would promise disk that can never be freed twice.
func markNested(items []Item) {
	for i := range items {
		for j := range items {
			if i == j || !isInside(items[i].Path, items[j].Path) {
				continue
			}
			if items[i].NestedIn == "" || len(items[j].Path) < len(items[i].NestedIn) {
				items[i].NestedIn = items[j].Path
			}
		}
	}
}

// walkJob is one directory handed to the walker pool. depth is its distance from
// the scan root; tag is the index of the candidate being measured and is unused
// during the search phase.
type walkJob struct {
	dir   string
	tag   int
	depth int
}

// walkSpec configures one run of walkPool.
type walkSpec struct {
	roots   []walkJob
	workers int
	cancel  <-chan struct{}
	// visit gets the entries of dir, already read once, and returns the
	// subdirectories the walk should continue into.
	visit func(walkJob, []os.DirEntry) []walkJob
	// settled is called after every visit with the number of children enqueued,
	// letting a caller know when one subtree has been fully walked.
	settled func(walkJob, int)
}

// walkPool walks directory trees with a fixed set of workers sharing one stack of
// pending directories, and returns when the whole frontier is done. Workers pop
// from the end of the stack, so the pending set stays close to a depth-first
// frontier instead of holding a whole level of the tree.
func walkPool(spec walkSpec) {
	if len(spec.roots) == 0 {
		return
	}
	p := newDirStack()
	p.push(spec.roots)

	var wg sync.WaitGroup
	for i := 0; i < spec.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-spec.cancel:
					p.abort()
					return
				default:
				}
				j, ok := p.pop()
				if !ok {
					return
				}
				var subs []walkJob
				if ents, err := readDir(j.dir); err == nil {
					subs = spec.visit(j, ents)
				}
				// Book children before handing them to the stack: a worker that
				// picked one up must never observe its parent's subtree as done.
				if spec.settled != nil {
					spec.settled(j, len(subs))
				}
				p.push(subs)
				p.finish(1)
			}
		}()
	}
	wg.Wait()
}

// readDir lists a directory without the lexical sort os.ReadDir applies. Neither
// phase depends on entry order — results are sorted once at the end — and the
// sort showed up in the scan's CPU profile.
func readDir(path string) ([]os.DirEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return f.ReadDir(-1)
}

// descendJobs turns directory entries into child jobs of parent. The tag is
// inherited so a measured subtree stays attributed to its own candidate, and
// symlinks are not followed: DirEntry.IsDir is false for them, which is how the
// walk has always treated a link — following one could also leave the drive.
func descendJobs(ents []os.DirEntry, parent walkJob) []walkJob {
	var out []walkJob
	for _, e := range ents {
		if e.IsDir() {
			out = append(out, walkJob{
				dir:   filepath.Join(parent.dir, e.Name()),
				tag:   parent.tag,
				depth: parent.depth + 1,
			})
		}
	}
	return out
}

// dirStack is the pending-directory stack shared by one walker pool. It counts
// unfinished jobs so the last worker to finish can wake the rest.
type dirStack struct {
	mu      sync.Mutex
	cond    sync.Cond
	items   []walkJob
	pending int
	closed  bool
}

func newDirStack() *dirStack {
	d := &dirStack{}
	d.cond.L = &d.mu
	return d
}

func (d *dirStack) push(jobs []walkJob) {
	if len(jobs) == 0 {
		return
	}
	d.mu.Lock()
	d.pending += len(jobs)
	d.items = append(d.items, jobs...)
	d.mu.Unlock()
	for i := 0; i < len(jobs); i++ {
		d.cond.Signal()
	}
}

func (d *dirStack) pop() (walkJob, bool) {
	d.mu.Lock()
	for len(d.items) == 0 && !d.closed {
		d.cond.Wait()
	}
	if len(d.items) == 0 {
		d.mu.Unlock()
		return walkJob{}, false
	}
	last := len(d.items) - 1
	j := d.items[last]
	d.items = d.items[:last]
	d.mu.Unlock()
	return j, true
}

// finish marks n jobs as fully processed; with nothing left it closes the stack.
func (d *dirStack) finish(n int) {
	d.mu.Lock()
	d.pending -= n
	if d.pending == 0 {
		d.closed = true
		d.cond.Broadcast()
	}
	d.mu.Unlock()
}

// abort ends the walk early on cancellation; workers leave without draining.
func (d *dirStack) abort() {
	d.mu.Lock()
	d.closed = true
	d.cond.Broadcast()
	d.mu.Unlock()
}

// Sizes returns the recursive apparent size of each path using one shared pool of
// walkers, in the order given. The folder browser uses it to size the child
// directories of a result without reading them one at a time.
func Sizes(paths []string) []int64 {
	return treeSizes(paths, scanWorkers(), nil, nil, nil)
}

// treeSizes walks every root with one pool and reports each root's apparent size.
// onVisit observes progress (current directory), and onDone fires for a root only
// once its whole subtree has been read, so a size is never published half-made.
func treeSizes(roots []string, workers int, cancel <-chan struct{}, onVisit func(dir string), onDone func(index int, size int64)) []int64 {
	sums := make([]atomic.Int64, len(roots))
	outstanding := make([]atomic.Int32, len(roots))
	jobs := make([]walkJob, len(roots))
	for i, r := range roots {
		jobs[i] = walkJob{dir: r, tag: i}
		outstanding[i].Store(1)
	}

	walkPool(walkSpec{
		roots:   jobs,
		workers: workers,
		cancel:  cancel,
		visit: func(j walkJob, ents []os.DirEntry) []walkJob {
			if onVisit != nil {
				onVisit(j.dir)
			}
			var sum int64
			for _, e := range ents {
				if e.IsDir() {
					continue
				}
				if info, ierr := e.Info(); ierr == nil {
					sum += FileBytes(info)
				}
			}
			if sum != 0 {
				sums[j.tag].Add(sum)
			}
			return descendJobs(ents, j)
		},
		settled: func(j walkJob, children int) {
			if outstanding[j.tag].Add(int32(children)-1) > 0 {
				return
			}
			if onDone != nil {
				onDone(j.tag, sums[j.tag].Load())
			}
		},
	})

	out := make([]int64, len(roots))
	for i := range sums {
		out[i] = sums[i].Load()
	}
	return out
}

// Scan scans the given roots for the selected categories. Cancel() aborts it.
func (s *Scanner) Scan(roots []string, ruleIDs []string) Result {
	var active []rules.Rule
	for _, id := range ruleIDs {
		if r := rules.ByID(id); r != nil {
			active = append(active, *r)
		}
	}
	if len(active) == 0 || len(roots) == 0 {
		return Result{}
	}

	// Candidates from home paths (fixed locations, no walk needed). Only
	// included when the home directory sits under one of the chosen roots.
	var phase1 []candidate
	home, herr := os.UserHomeDir()
	if herr == nil {
		var homeInRoot bool
		for _, root := range roots {
			if isUnder(home, root) {
				homeInRoot = true
				break
			}
		}
		if homeInRoot {
			for _, r := range active {
				for _, abs := range r.ExpandHome(home) {
					st, serr := os.Lstat(abs)
					if serr != nil {
						continue
					}
					c := candidate{path: abs, category: r.ID}
					switch {
					case st.Mode()&os.ModeSymlink != 0:
						// A cache moved behind a symlink (DerivedData onto a second
						// drive is the common case) holds its space at the target, and
						// deleting the link would break the tool instead of freeing the
						// disk — so measure and offer the target, and keep the link in
						// the UI as the reason it showed up.
						if tgt, terr := filepath.EvalSymlinks(abs); terr == nil {
							if tst, terr2 := os.Stat(tgt); terr2 == nil && tst.IsDir() {
								c.path, c.linkFrom = tgt, abs
							}
						}
					case !st.IsDir():
						continue
					}
					phase1 = append(phase1, c)
				}
			}
		}
	}

	var (
		mu      sync.Mutex
		skipped atomic.Int32
		found   atomic.Int32
	)
	// The home candidates above are already known, so they count towards the
	// "candidates found" figure the search phase reports.
	found.Add(int32(len(phase1)))

	// Phase 1: find target directories, walking every root with one pool.
	rootsJobs := make([]walkJob, 0, len(roots))
	for _, r := range roots {
		rootsJobs = append(rootsJobs, walkJob{dir: r})
	}
	index := buildNameIndex(active)
	workers := scanWorkers()

	s.resetCounters()
	stopHeartbeat := s.heartbeat(200*time.Millisecond, "search", 0,
		func() int { return int(found.Load()) }, func() int64 { return 0 })

	walkPool(walkSpec{
		roots:   rootsJobs,
		workers: workers,
		cancel:  s.stop,
		visit: func(j walkJob, ents []os.DirEntry) []walkJob {
			s.dirsVisited.Add(1)
			s.curPath.Store(j.dir)

			name := filepath.Base(j.dir)
			lower := strings.ToLower(name)

			if hits, ok := index[lower]; ok {
				for _, h := range hits {
					r := &active[h.ruleIdx]
					if !patternMatches(r, h.pattern, j.dir, name, j.depth) {
						continue
					}
					// Content filter? Reuse the entries already read for this dir.
					if f := r.ContentFilterFor(name); f != nil && !f(entryNames(ents)) {
						continue
					}
					mu.Lock()
					phase1 = append(phase1, candidate{path: j.dir, category: r.ID})
					mu.Unlock()
					s.progress(Progress{
						Phase:       "search",
						Path:        j.dir,
						ItemsFound:  int(found.Add(1)),
						DirsVisited: int(s.dirsVisited.Load()),
					})
					return nil // do not descend into the target
				}
			}

			if j.depth > 0 && skipMatch(j.dir, lower) {
				skipped.Add(1)
				return nil
			}
			return descendJobs(ents, j)
		},
	})
	stopHeartbeat()

	if s.isClosed() {
		return finalize(nil, int(skipped.Load()), true)
	}

	// Phase 2: measure every candidate through one shared pool, so a huge tree
	// such as a large node_modules is read by all workers instead of by one.
	items := make([]Item, len(phase1))
	toMeasure := make([]string, 0, len(phase1))
	indexOf := make([]int, 0, len(phase1))
	var measured, totalAll atomic.Int64

	for i, c := range phase1 {
		// What is left here is a symlink whose target no longer exists: there is
		// nothing to walk, and its size is the dangling link itself.
		if st, serr := os.Lstat(c.path); serr == nil && !st.IsDir() {
			items[i] = Item{Path: c.path, Size: st.Size(), Category: c.category}
			measured.Add(1)
			continue
		}
		toMeasure = append(toMeasure, c.path)
		indexOf = append(indexOf, i)
	}

	s.resetCounters()
	stopMeasureHeartbeat := s.heartbeat(200*time.Millisecond, "measure", len(phase1),
		func() int { return int(measured.Load()) }, func() int64 { return totalAll.Load() })

	treeSizes(toMeasure, workers, s.stop,
		func(dir string) {
			s.dirsVisited.Add(1)
			s.curPath.Store(dir)
		},
		func(done int, size int64) {
			pi := indexOf[done]
			c := phase1[pi]
			items[pi] = Item{Path: c.path, Size: size, Category: c.category, LinkFrom: c.linkFrom}
			s.progress(Progress{
				Phase:       "measure",
				Path:        c.path,
				ItemsFound:  int(measured.Add(1)),
				TotalItems:  len(phase1),
				BytesSoFar:  totalAll.Add(size),
				DirsVisited: int(s.dirsVisited.Load()),
			})
		})
	stopMeasureHeartbeat()

	return finalize(assessAll(items), int(skipped.Load()), s.isClosed())
}

// assessAll grades every measured item with its category's rules. It runs once
// over the final list, so the marker check (one directory read) costs nothing
// during the walk.
func assessAll(items []Item) []Item {
	byID := map[string]*rules.Rule{}
	for i := range items {
		it := &items[i]
		if it.Path == "" {
			continue
		}
		r, ok := byID[it.Category]
		if !ok {
			r = rules.ByID(it.Category)
			byID[it.Category] = r
		}
		if r == nil {
			continue
		}
		a := r.Assess(it.Path)
		it.Level, it.Reasons, it.KeepRoot = a.Level, a.Reasons, r.KeepRoot && a.Level != safety.Blocked
	}
	return items
}

// finalize drops empty items, marks nested ones, sorts largest first and totals
// the result. Nested items are excluded from the total because the parent that
// contains them already accounts for the same bytes, and so are blocked ones:
// they can never be freed.
func finalize(items []Item, dirsSkipped int, partial bool) Result {
	final := make([]Item, 0, len(items))
	for _, it := range items {
		if it.Size > 0 {
			final = append(final, it)
		}
	}
	sort.Slice(final, func(i, j int) bool {
		if final[i].Size != final[j].Size {
			return final[i].Size > final[j].Size
		}
		return final[i].Path < final[j].Path
	})
	markNested(final)

	var total int64
	for _, it := range final {
		if it.NestedIn == "" && it.Level != safety.Blocked {
			total += it.Size
		}
	}

	return Result{
		Items:       final,
		TotalBytes:  total,
		ItemCount:   len(final),
		DirsSkipped: dirsSkipped,
		Partial:     partial,
	}
}

// isUnder reports whether path is under (or equal to) root.
func isUnder(path, root string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel == "." || (!strings.HasPrefix(rel, "..") && rel != "")
}

// entryNames returns the list of directory entry names.
func entryNames(entries []os.DirEntry) []string {
	out := make([]string, len(entries))
	for i, e := range entries {
		out[i] = e.Name()
	}
	return out
}
