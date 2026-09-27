// Unit tests for the rules package — matchers, content filters, path helpers.
package rules

import (
	"path/filepath"
	"runtime"
	"testing"
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
