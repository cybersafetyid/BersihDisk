// Integration tests for the scanner with temporary fixture directories.
package scanner

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"bersihdisk/internal/rules"
)

// fixture builds a dummy project structure for testing.
func fixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	// Node project: app/node_modules/express (small file)
	mk(t, filepath.Join(root, "app", "node_modules", "express"), "index.js", "console.log(1)")
	// Rust: target containing debug
	mk(t, filepath.Join(root, "rs", "target", "debug"), "lib.rlib", "x")
	// Rust: target containing source code -> must not match
	mk(t, filepath.Join(root, "rs2", "target"), "main.rs", "fn main(){}")
	// Gradle build artifacts
	mk(t, filepath.Join(root, "jv", "build", "outputs"), "app.apk", "x")
	// Gradle build with source code -> must not match
	mk(t, filepath.Join(root, "jv2", "build"), "Main.java", "class Main{}")
	// Python
	mk(t, filepath.Join(root, "py", "__pycache__"), "mod.cpython-311.pyc", "x")
	// Skip-list: node_modules inside .Trash must not be detected
	mk(t, filepath.Join(root, ".Trash", "x", "node_modules"), "i.js", "x")

	return root
}

// mk creates a file inside dir.
func mk(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestScanFixture(t *testing.T) {
	root := fixture(t)
	s := New(func(Progress) {})

	categories := []string{rules.CatNodeJS, rules.CatRust, rules.CatGradle, rules.CatPython}
	res := s.Scan([]string{root}, categories)

	if res.Partial {
		t.Fatal("scan should not be cancelled")
	}
	paths := map[string]string{}
	for _, it := range res.Items {
		paths[it.Path] = it.Category
	}

	wantFound := []string{
		filepath.Join(root, "app", "node_modules"),
		filepath.Join(root, "rs", "target"),
		filepath.Join(root, "jv", "build"),
		filepath.Join(root, "py", "__pycache__"),
	}
	for _, p := range wantFound {
		if _, ok := paths[p]; !ok {
			t.Errorf("expected to find %s, got: %v", p, paths)
		}
	}

	wantMissing := []string{
		filepath.Join(root, "rs2", "target"),               // content filter fails
		filepath.Join(root, "jv2", "build"),                // content filter fails
		filepath.Join(root, ".Trash", "x", "node_modules"), // skip-list
	}
	for _, p := range wantMissing {
		if _, ok := paths[p]; ok {
			t.Errorf("must not find %s", p)
		}
	}

	if res.DirsSkipped == 0 {
		t.Error(".Trash should count as skipped")
	}
	if res.TotalBytes <= 0 {
		t.Error("total bytes must be > 0")
	}
}

// System-folder names are skipped only directly under a filesystem root; a
// developer's own "dev" or "System" folder must still be searched.
func TestSkipMatchIsRootOnlyForSystemNames(t *testing.T) {
	sep := string(filepath.Separator)
	root := sep
	if v := filepath.VolumeName(os.TempDir()); v != "" {
		root = v + sep
	}
	cases := []struct {
		dir  string
		want bool
	}{
		{filepath.Join(root, "dev"), true},
		{filepath.Join(root, "Users", "me", "dev"), false},
		{filepath.Join(root, "work", "System"), false},
		{filepath.Join(root, "work", ".Trash"), true},
	}
	for _, c := range cases {
		if got := skipMatch(c.dir, strings.ToLower(filepath.Base(c.dir))); got != c.want {
			t.Errorf("skipMatch(%q) = %v, want %v", c.dir, got, c.want)
		}
	}
}

func TestScanCancelled(t *testing.T) {
	root := fixture(t)
	s := New(func(Progress) {})
	s.Cancel()
	res := s.Scan([]string{root}, []string{rules.CatNodeJS})
	if !res.Partial {
		t.Error("scan after Cancel() must be marked partial")
	}
}

// Cancel during the measure phase must stop the walk and report what was already
// measured instead of running the whole scan to completion.
func TestScanCancelledDuringMeasure(t *testing.T) {
	root := fixture(t)
	var s *Scanner
	var sawMeasure bool
	s = New(func(p Progress) {
		if p.Phase == "measure" && !sawMeasure {
			sawMeasure = true
			s.Cancel()
		}
	})

	res := s.Scan([]string{root}, []string{rules.CatNodeJS, rules.CatRust, rules.CatGradle, rules.CatPython})

	if !res.Partial {
		t.Fatal("cancel during measure must mark the result partial")
	}
	if !sawMeasure {
		t.Fatal("fixture should reach the measure phase")
	}
	if res.ItemCount == 0 {
		t.Error("items measured before the cancel must be reported")
	}
}

// Depth is measured from the scan root, so a target below the rule's maxDepth is
// not reported.
func TestScanRespectsMaxDepth(t *testing.T) {
	root := fixture(t)
	inRange := filepath.Join(root, "p1", "p2", "p3", "p4", "node_modules")
	outOfRange := filepath.Join(root, "q1", "q2", "q3", "q4", "q5", "node_modules")
	mk(t, inRange, "x.js", "1")
	mk(t, outOfRange, "x.js", "1")

	s := New(func(Progress) {})
	res := s.Scan([]string{root}, []string{rules.CatNodeJS})

	found := map[string]bool{}
	for _, it := range res.Items {
		found[it.Path] = true
	}
	if !found[inRange] {
		t.Errorf("node_modules at max depth must be found, got %v", found)
	}
	if found[outOfRange] {
		t.Errorf("node_modules beyond max depth must not be found: %v", found)
	}
}

// The UI shows what the scan is actually doing, so both phases have to publish
// live activity: directories read and the path currently being worked on.
func TestScanReportsLiveActivity(t *testing.T) {
	root := fixture(t)
	var maxSearchDirs, maxMeasureDirs int
	var lastSearchPath, lastMeasurePath string
	var sawTotal bool

	s := New(func(p Progress) {
		switch p.Phase {
		case "search":
			if p.DirsVisited > maxSearchDirs {
				maxSearchDirs = p.DirsVisited
			}
			lastSearchPath = p.Path
		case "measure":
			if p.DirsVisited > maxMeasureDirs {
				maxMeasureDirs = p.DirsVisited
			}
			lastMeasurePath = p.Path
			if p.TotalItems > 0 {
				sawTotal = true
			}
		}
	})
	res := s.Scan([]string{root}, []string{rules.CatNodeJS, rules.CatRust})

	if maxSearchDirs <= res.ItemCount {
		t.Errorf("search phase must report directories read (got %d over %d items)", maxSearchDirs, res.ItemCount)
	}
	if lastSearchPath == "" {
		t.Error("search phase must report the current path")
	}
	if maxMeasureDirs == 0 || lastMeasurePath == "" {
		t.Errorf("measure phase must report directories read and path (dirs=%d path=%q)", maxMeasureDirs, lastMeasurePath)
	}
	if !sawTotal {
		t.Error("measure phase must report the candidate total")
	}
}

// Two scans of the same tree must produce the same list; the walker pool reads
// directories in parallel, so ordering has to be settled at the end.
func TestScanIsDeterministic(t *testing.T) {
	root := fixture(t)
	ids := []string{rules.CatNodeJS, rules.CatRust, rules.CatGradle, rules.CatPython}

	scan := func() string {
		s := New(func(Progress) {})
		res := s.Scan([]string{root}, ids)
		var b strings.Builder
		for _, it := range res.Items {
			b.WriteString(it.Path)
			b.WriteString("=")
			b.WriteString(strconv.FormatInt(it.Size, 10))
			b.WriteString("\n")
		}
		return b.String()
	}

	first := scan()
	if second := scan(); first != second {
		t.Errorf("scan results differ between runs:\n%s\n---\n%s", first, second)
	}
}

func TestWalkPoolWithoutRootsReturns(t *testing.T) {
	done := make(chan struct{})
	go func() {
		walkPool(walkSpec{workers: 2, cancel: make(chan struct{}), visit: func(walkJob, []os.DirEntry) []walkJob { return nil }})
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("walkPool with no roots must return instead of waiting for work")
	}
}

// A multi-segment pattern is more specific than a bare name, so app/build belongs
// to Android while a plain build directory still belongs to Gradle.
func TestScanPrefersSpecificMultiSegmentPattern(t *testing.T) {
	root := t.TempDir()
	appBuild := filepath.Join(root, "proj", "app", "build")
	plainBuild := filepath.Join(root, "plain", "build")
	mk(t, filepath.Join(appBuild, "outputs"), "app.apk", "x")
	mk(t, filepath.Join(plainBuild, "outputs"), "app.apk", "y")

	res := New(func(Progress) {}).Scan([]string{root}, []string{rules.CatGradle, rules.CatAndroid})

	cases := map[string]string{appBuild: rules.CatAndroid, plainBuild: rules.CatGradle}
	for path, want := range cases {
		got := ""
		for _, it := range res.Items {
			if it.Path == path {
				got = it.Category
			}
		}
		if got != want {
			t.Errorf("%s claimed by %q, want %q (items: %v)", path, got, want, res.Items)
		}
	}
}

// A cache moved behind a symlink is measured at its real location — that is where
// the disk actually goes — and the link is kept so the UI can explain it.
func TestScanResolvesSymlinkedHomeCache(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the Xcode home paths are macOS-only, and Windows symlinks need extra privileges")
	}
	root := t.TempDir()
	home := filepath.Join(root, "home")
	link := filepath.Join(home, "Library", "Developer", "Xcode", "DerivedData")
	target := filepath.Join(root, "big-volume", "DerivedData")
	mk(t, target, "index-store", strings.Repeat("x", 128))
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)

	resolved, err := filepath.EvalSymlinks(target)
	if err != nil {
		t.Fatal(err)
	}

	res := New(func(Progress) {}).Scan([]string{home}, []string{rules.CatXcode})

	if res.ItemCount != 1 {
		t.Fatalf("items = %d, want 1: %v", res.ItemCount, res.Items)
	}
	it := res.Items[0]
	if it.Path != resolved {
		t.Errorf("path = %q, want the real target %q", it.Path, resolved)
	}
	if it.LinkFrom != link {
		t.Errorf("linkFrom = %q, want %q", it.LinkFrom, link)
	}
	if it.Size <= 32 {
		t.Errorf("size = %d, want the target contents, not the link", it.Size)
	}
}

// Items inside another item must not be counted twice, and a sibling whose name
// merely starts with the parent's name is not nested.
func TestFinalizeCountsNestedOnce(t *testing.T) {
	res := finalize([]Item{
		{Path: "/home/.cache", Size: 5000},
		{Path: "/home/.cache/yarn", Size: 200},
		{Path: "/home/.cache/huggingface", Size: 300},
		{Path: "/home/.cacheX", Size: 7},
	}, 0, false)

	if res.TotalBytes != 5007 {
		t.Errorf("total = %d, want 5007 (nested bytes counted once)", res.TotalBytes)
	}
	nested := map[string]string{}
	for _, it := range res.Items {
		nested[it.Path] = it.NestedIn
	}
	for p := range nested {
		want := "/home/.cache"
		if p == "/home/.cache" || p == "/home/.cacheX" {
			want = ""
		}
		if nested[p] != want {
			t.Errorf("nestedIn(%s) = %q, want %q", p, nested[p], want)
		}
	}
}

// A hard-linked file keeps its data through the other link, so it must not be
// promised as freed space.
func TestSizesIgnoreHardLinks(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "d")
	mk(t, dir, "own.bin", strings.Repeat("x", 100))
	mk(t, root, "shared.bin", strings.Repeat("y", 500))
	if err := os.Link(filepath.Join(root, "shared.bin"), filepath.Join(dir, "link.bin")); err != nil {
		t.Skip("hard links unavailable:", err)
	}
	if runtime.GOOS == "windows" {
		t.Skip("hard links are not detected on Windows")
	}
	if got := Sizes([]string{dir})[0]; got != 100 {
		t.Errorf("size = %d, want 100 (hard link excluded)", got)
	}
}

func TestScanGradesItems(t *testing.T) {
	root := t.TempDir()
	// A real project (package.json beside node_modules) and a loose folder without one.
	mk(t, filepath.Join(root, "app", "node_modules", "x"), "i.js", "x")
	mk(t, filepath.Join(root, "app"), "package.json", "{}")
	mk(t, filepath.Join(root, "loose", "node_modules", "x"), "i.js", "x")
	// A dependency folder inside an app bundle belongs to the installed app.
	mk(t, filepath.Join(root, "Tool.app", "Contents", "Resources", "node_modules", "x"), "i.js", "x")

	res := New(func(Progress) {}).Scan([]string{root}, []string{rules.CatNodeJS})
	levels := map[string]Item{}
	for _, it := range res.Items {
		levels[it.Path] = it
	}
	app := levels[filepath.Join(root, "app", "node_modules")]
	if app.Level != "safe" {
		t.Errorf("project node_modules level = %q (%v), want safe", app.Level, app.Reasons)
	}
	loose := levels[filepath.Join(root, "loose", "node_modules")]
	if loose.Level != "caution" || len(loose.Reasons) == 0 || loose.Reasons[0] != "noProjectFile" {
		t.Errorf("loose node_modules = %q %v, want caution/noProjectFile", loose.Level, loose.Reasons)
	}
	bundle, found := levels[filepath.Join(root, "Tool.app", "Contents", "Resources", "node_modules")]
	if !found || bundle.Level != "danger" || bundle.Reasons[0] != "appBundle" {
		t.Errorf("app-bundle node_modules = %+v (found %v), want danger/appBundle", bundle, found)
	}
}
