// Unit tests for the rules package — matchers, content filters, path helpers.
package rules

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"bersihdisk/internal/safety"
)

func TestMatchDirNodeModules(t *testing.T) {
	r := ByID(CatNodeJS)
	if r == nil {
		t.Fatal("nodejs rule not found")
	}
	if !r.MatchDir("node_modules", 1, nil) {
		t.Error("node_modules at depth 1 should match")
	}
	if !r.MatchDir("Node_MODULES", 3, nil) {
		t.Error("matching should be case-insensitive")
	}
	if r.MatchDir("node_modules", 6, nil) {
		t.Error("node_modules at depth 6 should exceed maxDepth")
	}
	if r.MatchDir("node_modules", 0, nil) {
		t.Error("depth must start at 1")
	}
}

func TestMatchDirRustTargetFilter(t *testing.T) {
	r := ByID(CatRust)
	cargo := []string{"debug", "release", "CACHEDIR.TAG"}
	if !r.MatchDir("target", 1, cargo) {
		t.Error("target containing debug/release should match")
	}
	if r.MatchDir("target", 1, []string{"main.rs"}) {
		t.Error("target containing source code must not match")
	}
}

func TestMatchDirGradleBuildFilter(t *testing.T) {
	r := ByID(CatGradle)
	if !r.MatchDir("build", 1, []string{"outputs", "tmp", "intermediates"}) {
		t.Error("build containing APK artifacts should match")
	}
	if r.MatchDir("build", 1, []string{"src", "main.ts"}) {
		t.Error("build containing source code must not match")
	}
	if r.MatchDir("build", 1, []string{}) {
		t.Error("empty build must not match")
	}
}

func TestMatchHomePathPerOS(t *testing.T) {
	// CatGo has "go/pkg/mod" without a prefix (all OSes) and two unix-only paths.
	gr := ByID(CatGo)
	if gr == nil {
		t.Fatal("go rule not found")
	}
	if !gr.MatchHomePath("go/pkg/mod") {
		t.Error("go/pkg/mod should match on every OS")
	}
	if runtime.GOOS != "windows" && !gr.MatchHomePath(".cache/go-build") {
		t.Error(".cache/go-build should match on unix")
	}
	if runtime.GOOS == "windows" && gr.MatchHomePath(".cache/go-build") {
		t.Error(".cache/go-build must not match on Windows")
	}
}

func TestHomeRelToAbs(t *testing.T) {
	abs := HomeRelToAbs("/home/u", "go/pkg/mod")
	if abs != filepath.Join("/home/u", "go", "pkg", "mod") {
		t.Errorf("HomeRelToAbs wrong: %s", abs)
	}
}

func TestAllRuleIDsUnique(t *testing.T) {
	seen := map[string]bool{}
	for _, r := range All() {
		if seen[r.ID] {
			t.Errorf("duplicate ID: %s", r.ID)
		}
		seen[r.ID] = true
		if r.Icon == "" {
			t.Errorf("rule %s: an icon is required", r.ID)
		}
		if len(r.dirNames) == 0 && len(r.homePaths) == 0 {
			t.Errorf("rule %s: needs dirNames or homePaths", r.ID)
		}
	}
}

func TestAssessFlagsProjectlessAndRiskyItems(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home) // Windows reads its home folder from here

	proj := filepath.Join(home, "dev", "app")
	if err := os.MkdirAll(filepath.Join(proj, "node_modules"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(home, "dev", "loose", "node_modules"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(proj, "package.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	node := ByID(CatNodeJS)
	if a := node.Assess(filepath.Join(proj, "node_modules")); a.Level != safety.Safe {
		t.Errorf("project node_modules = %+v, want safe", a)
	}
	a := node.Assess(filepath.Join(home, "dev", "loose", "node_modules"))
	if a.Level != safety.Caution || a.Reasons[0] != "noProjectFile" {
		t.Errorf("node_modules without package.json = %+v", a)
	}

	if a := ByID(CatRuby).Assess(filepath.Join(home, ".gem")); a.Level != safety.Danger {
		t.Errorf("~/.gem = %+v, want danger", a)
	}
	if a := ByID(CatMaven).Assess(filepath.Join(home, ".m2", "repository")); a.Level != safety.Caution {
		t.Errorf("~/.m2/repository = %+v, want caution", a)
	}
	if a := ByID(CatEmulators).Assess(filepath.Join(home, ".android", "avd")); a.Level != safety.Danger {
		t.Errorf("AVD folder = %+v, want danger", a)
	}
	if a := ByID(CatTemp).Assess(filepath.Join(home, ".cache")); a.Level != safety.Caution {
		t.Errorf("~/.cache = %+v, want caution", a)
	}
}

func TestCategoryRiskAndKeepRoot(t *testing.T) {
	for _, r := range All() {
		if r.Risk != "" && r.Risk != safety.Caution && r.Risk != safety.Danger {
			t.Errorf("%s: category risk %q must be caution or danger", r.ID, r.Risk)
		}
		if r.KeepRoot && r.ID != CatTemp {
			t.Errorf("%s: only Temp & cache is a keep-root container", r.ID)
		}
	}
}

func TestExpandHomeGlobSkipsToolbox(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("the JetBrains glob is a Windows path")
	}
	home := t.TempDir()
	for _, n := range []string{"IntelliJIdea2024.1", "Toolbox"} {
		if err := os.MkdirAll(filepath.Join(home, "AppData", "Local", "JetBrains", n), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, p := range ByID(CatJetBrains).ExpandHome(home) {
		if strings.EqualFold(filepath.Base(p), "Toolbox") {
			t.Errorf("Toolbox must never be offered: %s", p)
		}
	}
}

func TestDockerWindowsPathAvoidsDiskImage(t *testing.T) {
	for _, r := range All() {
		for _, hp := range r.homePaths {
			if hp == "win:AppData/Local/Docker" {
				t.Errorf("%s lists the whole Docker Desktop folder, which holds the WSL disk image", r.ID)
			}
		}
	}
}

func TestCoverageFilterNeedsAReport(t *testing.T) {
	if coverageFilter([]string{"notes.txt", "photos"}) {
		t.Error("a folder called coverage without a report is not build output")
	}
	if !coverageFilter([]string{"lcov.info"}) {
		t.Error("lcov.info marks a coverage report")
	}
}

func TestExpandHomeFollowsXDGRelocation(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("XDG variables are a Linux/macOS convention")
	}
	home := t.TempDir()
	moved := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", moved)

	got := ByID(CatPip).ExpandHome(home)
	want := filepath.Join(moved, "pip")
	var hasDefault, hasMoved bool
	for _, p := range got {
		hasDefault = hasDefault || p == filepath.Join(home, ".cache", "pip")
		hasMoved = hasMoved || p == want
	}
	if !hasDefault || !hasMoved {
		t.Errorf("want both the default and the relocated pip cache, got %v", got)
	}

	// The container itself moves too, and a variable equal to the default adds no duplicate.
	temp := ByID(CatTemp).ExpandHome(home)
	if len(temp) < 2 {
		t.Errorf("~/.cache and its relocated twin expected: %v", temp)
	}
	t.Setenv("XDG_CACHE_HOME", filepath.Join(home, ".cache"))
	seen := map[string]int{}
	for _, p := range ByID(CatPip).ExpandHome(home) {
		seen[p]++
		if seen[p] > 1 {
			t.Errorf("duplicate %s", p)
		}
	}
}

func TestCachePathsMatchVendorDocumentation(t *testing.T) {
	has := func(cat string, want string) bool {
		for _, hp := range ByID(cat).homePaths {
			if hp == want {
				return true
			}
		}
		return false
	}
	// Facts from the npm, Cargo and Go documentation: Cargo's home is ~/.cargo on
	// Windows too, npm keeps its cache in %LocalAppData%\npm-cache, and Go's build
	// cache is %LocalAppData%\go-build.
	for cat, want := range map[string]string{
		CatRust: ".cargo/registry", CatElectron: "win:AppData/Local/npm-cache", CatGo: "win:AppData/Local/go-build",
		CatPip: "win:AppData/Local/pip/Cache",
	} {
		if !has(cat, want) {
			t.Errorf("%s lacks %s", cat, want)
		}
	}
	if has(CatRust, "win:AppData/Local/Cargo/registry") {
		t.Error("Cargo does not use %LocalAppData% on Windows")
	}
}
