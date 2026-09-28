package uninstall

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"bersihdisk/internal/deleter"
	"bersihdisk/internal/safety"
	"bersihdisk/internal/scanner"
)

// Provider finds installed things of one kind on this machine.
type Provider interface {
	ID() string
	// List returns ErrUnavailable when its tool is missing or the OS does not fit.
	List(ctx context.Context, h Host) ([]Installed, error)
}

// Advisor is an optional Provider extension that warns about one package before
// it is removed (for example "another formula depends on it").
type Advisor interface {
	Advise(ctx context.Context, h Host, p Installed) []Warning
}

// DefaultProviders is every provider the app ships.
func DefaultProviders() []Provider {
	ps := []Provider{macApps{}, winApps{}, appxApps{}, linuxApps{}, toolchains{}, goBins{}}
	for _, s := range cliSpecs() {
		ps = append(ps, cliProvider{s})
	}
	return ps
}

// Inventory lists installed packages and runs uninstall plans for them.
type Inventory struct {
	host      Host
	providers []Provider
	backupDir string // where Windows registry exports and PATH backups go

	mu    sync.Mutex
	pkgs  map[string]Installed
	plans map[string]*Plan
	icons map[string]string // package ID -> data URL

	iconSem chan struct{}
}

// New creates an inventory over the given providers.
func New(h Host, providers ...Provider) *Inventory {
	dir, _ := os.UserConfigDir()
	return &Inventory{
		host: h, providers: providers,
		backupDir: filepath.Join(dir, "BersihDisk", "backups"),
		pkgs:      map[string]Installed{}, plans: map[string]*Plan{},
		icons: map[string]string{}, iconSem: make(chan struct{}, 4),
	}
}

// List runs every provider concurrently. A provider that fails is reported in
// Unavailable; one whose tool is simply not installed is silent.
func (inv *Inventory) List(ctx context.Context) ListResult {
	type answer struct {
		id   string
		pkgs []Installed
		err  error
	}
	ch := make(chan answer, len(inv.providers))
	for _, p := range inv.providers {
		go func(p Provider) {
			c, cancel := context.WithTimeout(ctx, 45*time.Second)
			defer cancel()
			pkgs, err := p.List(c, inv.host)
			ch <- answer{p.ID(), pkgs, err}
		}(p)
	}

	res := ListResult{Packages: []Installed{}, Unavailable: []string{}}
	found := map[string]Installed{}
	failed := map[string]bool{}
	for range inv.providers {
		a := <-ch
		if a.err != nil && !errors.Is(a.err, ErrUnavailable) && !failed[a.id] {
			failed[a.id] = true
			res.Unavailable = append(res.Unavailable, a.id)
		}
		for _, p := range a.pkgs {
			found[p.ID] = p
		}
	}
	for _, p := range found {
		res.Packages = append(res.Packages, p)
	}
	sort.Slice(res.Packages, func(i, j int) bool {
		a, b := res.Packages[i], res.Packages[j]
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		return strings.ToLower(a.Name) < strings.ToLower(b.Name)
	})
	sort.Strings(res.Unavailable)

	inv.mu.Lock()
	inv.pkgs = found
	inv.mu.Unlock()
	return res
}

func newID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// Plan works out what uninstalling a listed package would do and stores it, so
// Execute can later run exactly those steps and nothing else.
func (inv *Inventory) Plan(ctx context.Context, packageID string) (Plan, error) {
	inv.mu.Lock()
	pkg, ok := inv.pkgs[packageID]
	inv.mu.Unlock()
	if !ok {
		return Plan{}, fmt.Errorf("%q is not in the last listing; refresh the list first", packageID)
	}
	rec := pkg.recipe
	plan := Plan{ID: newID(), PackageID: packageID, Name: pkg.Name}

	for _, w := range rec.Warnings {
		plan.Warnings = append(plan.Warnings, Warning{Code: w})
	}
	for _, p := range inv.providers {
		if adv, ok := p.(Advisor); ok && p.ID() == pkg.Provider {
			plan.Warnings = append(plan.Warnings, adv.Advise(ctx, inv.host, pkg)...)
		}
	}

	add := func(s Step) {
		s.ID = fmt.Sprintf("s%d", len(plan.Steps))
		plan.Steps = append(plan.Steps, s)
	}
	for _, c := range rec.Commands {
		s := Step{Kind: StepCommand, Label: c.Label, Level: safety.Safe, Selected: !c.Optional, argv: c.Argv, raw: c.Raw}
		if c.NeedsAdmin {
			s.Manual = "sudo " + c.Label // the label is the full, copy-ready command line
		}
		add(s)
	}
	for _, s := range inv.pathSteps(rec.Paths) {
		add(s)
	}
	for _, e := range rec.Profiles {
		jobs, previews := profileJobs(inv.host.Home, e)
		for i, j := range jobs {
			add(Step{Kind: StepProfile, Label: j.file, Detail: strings.Join(capLines(previews[i], 8), "\n"),
				Level: safety.Safe, Selected: true, profile: j})
		}
	}
	for _, key := range rec.Registry {
		if !registryKeyExists(key) {
			continue
		}
		s := Step{Kind: StepRegistry, Label: key, Level: safety.Safe, Selected: true, key: key}
		if !strings.HasPrefix(strings.ToUpper(key), "HKCU") {
			s.Manual = fmt.Sprintf(`reg delete "%s" /f`, key)
		}
		add(s)
	}
	if len(rec.PathEntries) > 0 {
		var want []string
		for _, spec := range rec.PathEntries {
			want = append(want, inv.host.Expand(spec)...)
		}
		if present := userPathHas(want); len(present) > 0 {
			add(Step{Kind: StepEnvPath, Label: strings.Join(present, "; "), Level: safety.Safe, Selected: true, entries: want})
		}
	}

	for _, s := range plan.Steps {
		if s.Kind == StepPath && s.Selected && s.Manual == "" {
			plan.TotalBytes += s.Size
		}
	}
	inv.mu.Lock()
	if len(inv.plans) >= 20 {
		inv.plans = map[string]*Plan{} // plans are short-lived; drop the old ones
	}
	inv.plans[plan.ID] = &plan
	inv.mu.Unlock()
	return plan, nil
}

func capLines(lines []string, n int) []string {
	if len(lines) <= n {
		return lines
	}
	return append(append([]string{}, lines[:n]...), fmt.Sprintf("… +%d", len(lines)-n))
}

// pathSteps expands the residue specs into the paths that exist, drops those
// covered by another, measures them, and grades each: a guarded path is blocked,
// a guess or a location that may hold the user's own work is a caution that
// starts unchecked.
func (inv *Inventory) pathSteps(res []Residue) []Step {
	type found struct {
		path string
		r    Residue
	}
	var all []found
	seen := map[string]bool{}
	for _, r := range res {
		for _, p := range inv.host.Expand(r.Spec) {
			p = filepath.Clean(p)
			if seen[p] {
				continue
			}
			if _, err := os.Lstat(p); err != nil {
				continue
			}
			seen[p] = true
			all = append(all, found{p, r})
		}
	}
	// A path inside another listed path goes with its parent.
	kept := all[:0:0]
	for _, f := range all {
		covered := false
		for _, g := range all {
			if g.path != f.path && strings.HasPrefix(f.path, g.path+string(filepath.Separator)) {
				covered = true
				break
			}
		}
		if !covered {
			kept = append(kept, f)
		}
	}

	paths := make([]string, len(kept))
	for i, f := range kept {
		paths[i] = f.path
	}
	sizes := sizeOf(paths)

	steps := make([]Step, 0, len(kept))
	for i, f := range kept {
		s := Step{Kind: StepPath, Label: f.path, Size: sizes[i], Level: safety.Safe, path: f.path}
		switch reason, blocked := safety.Guard(f.path); {
		case blocked:
			s.Level, s.Reasons = safety.Blocked, []string{reason}
		case f.r.Reason != "":
			s.Level, s.Reasons, s.Selected = safety.Caution, []string{f.r.Reason}, false
		default:
			s.Selected = f.r.Exact
		}
		if f.r.NeedsAdmin && s.Level != safety.Blocked {
			s.Manual = manualRemove(f.path)
		}
		steps = append(steps, s)
	}
	sort.SliceStable(steps, func(i, j int) bool { return steps[i].Size > steps[j].Size })
	return steps
}

// sizeOf measures files directly and folders through the scanner's shared pool.
func sizeOf(paths []string) []int64 {
	out := make([]int64, len(paths))
	var dirs []string
	var slots []int
	for i, p := range paths {
		st, err := os.Lstat(p)
		switch {
		case err != nil:
		case st.IsDir():
			dirs = append(dirs, p)
			slots = append(slots, i)
		default:
			out[i] = scanner.FileBytes(st)
		}
	}
	for j, n := range scanner.Sizes(dirs) {
		out[slots[j]] = n
	}
	return out
}

// manualRemove is the command to show for a location that needs administrator
// rights. The path is single-quoted, so it cannot be split or expanded.
func manualRemove(path string) string {
	return "sudo rm -rf '" + strings.ReplaceAll(path, "'", `'\''`) + "'"
}

// Execute runs the chosen steps of a stored plan, in plan order: commands first,
// then files, profiles, registry and PATH. A failed command stops the run —
// deleting leftovers of a tool that refused to uninstall would break it.
func (inv *Inventory) Execute(ctx context.Context, req Request, emit func(Progress)) (Result, error) {
	inv.mu.Lock()
	plan := inv.plans[req.PlanID]
	inv.mu.Unlock()
	if plan == nil {
		return Result{}, fmt.Errorf("this plan has expired; open the package again")
	}

	chosen := map[string]bool{}
	for _, id := range req.Steps {
		chosen[id] = true
	}
	var run []Step
	needAck := false
	for _, s := range plan.Steps {
		if chosen[s.ID] {
			run = append(run, s)
			delete(chosen, s.ID)
			needAck = needAck || s.Level.NeedsAck()
		}
	}
	if len(chosen) > 0 {
		return Result{}, fmt.Errorf("steps are not part of this plan")
	}
	if needAck && !req.Acknowledged {
		return Result{}, fmt.Errorf("the risks of the selected steps were not acknowledged")
	}
	mode := deleter.ModeTrash
	if req.Mode == deleter.ModePermanent {
		mode = deleter.ModePermanent
	}

	start := time.Now()
	res := Result{Steps: []StepResult{}}
	record := func(s Step, status, msg, hint string) {
		res.Steps = append(res.Steps, StepResult{ID: s.ID, Status: status, Message: msg, Hint: hint})
		switch status {
		case StatusDone:
			res.OK++
		case StatusFailed:
			res.Failed++
		default:
			res.Skipped++
		}
		if status == StatusManual && s.Manual != "" {
			res.Manual = append(res.Manual, s.Manual)
		}
		if emit != nil {
			emit(Progress{Done: len(res.Steps), Total: len(run), Step: s.Label, Bytes: res.Bytes})
		}
	}

	for _, s := range run {
		if res.Aborted {
			record(s, StatusSkipped, "an earlier command failed", "")
			continue
		}
		if ctx.Err() != nil {
			res.Aborted = true
			record(s, StatusSkipped, "cancelled", "")
			continue
		}
		switch {
		case s.Level == safety.Blocked:
			record(s, StatusFailed, "protected location ("+strings.Join(s.Reasons, ", ")+")", "")
		case s.Manual != "":
			record(s, StatusManual, "needs administrator rights", "")
		default:
			status, msg, hint, freed := inv.runStep(ctx, s, mode)
			res.Bytes += freed
			record(s, status, msg, hint)
			if s.Kind == StepCommand && status == StatusFailed {
				res.Aborted = true
			}
		}
	}
	res.Duration = time.Since(start).Seconds()
	return res, nil
}

// runStep executes one step and reports how it went.
func (inv *Inventory) runStep(ctx context.Context, s Step, mode string) (status, msg, hint string, freed int64) {
	fail := func(err error) (string, string, string, int64) {
		return StatusFailed, err.Error(), safety.PermissionHint(inv.host.OS, err), 0
	}
	switch s.Kind {
	case StepCommand:
		c, cancel := context.WithTimeout(ctx, 15*time.Minute)
		defer cancel()
		var err error
		if s.raw != "" {
			err = runRaw(c, s.raw)
		} else {
			_, err = inv.host.Run.Run(c, s.argv[0], s.argv[1:]...)
		}
		if err != nil {
			return fail(err)
		}
	case StepPath:
		if _, err := os.Lstat(s.path); os.IsNotExist(err) {
			return StatusSkipped, "already gone", "", 0 // the native uninstall removed it
		}
		r := deleter.New(func(deleter.Progress) {}).Delete([]deleter.Item{{Path: s.path, Size: s.Size}}, mode)
		if r.Failed > 0 {
			return StatusFailed, r.Failures[0].Message, r.Failures[0].Hint, 0
		}
		freed = r.Bytes
	case StepProfile:
		if _, err := s.profile.apply(); err != nil {
			return fail(err)
		}
	case StepRegistry:
		if err := deleteRegistryKey(s.key, inv.backupDir); err != nil {
			return fail(err)
		}
	case StepEnvPath:
		if _, err := removeUserPathEntries(s.entries, inv.backupDir); err != nil {
			return fail(err)
		}
	}
	return StatusDone, "", "", freed
}
