// Package rules defines cleanup categories and path matchers.
package rules

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"bersihdisk/internal/safety"
)

// Category IDs.
const (
	CatNodeJS    = "nodejs"
	CatGo        = "go"
	CatRust      = "rust"
	CatGradle    = "gradle"
	CatMaven     = "maven"
	CatCpp       = "cpp"
	CatPython    = "python"
	CatDotnet    = "dotnet"
	CatXcode     = "xcode"
	CatAndroid   = "android"
	CatTemp      = "temp"
	CatAICache   = "ai"
	CatElectron  = "electron"
	CatDocker    = "docker"
	CatBrew      = "brew"
	CatPHP       = "php"
	CatPip       = "pip"
	CatRuby      = "ruby"
	CatFlutter   = "flutter"
	CatTerraform = "terraform"
	CatNuGet     = "nuget"
	CatJetBrains = "jetbrains"
	CatWebBuild  = "webbuild"
	CatEmulators = "emulators"
)

// Icon names resolved to SVGs by the frontend (simple-icons / lucide).
const (
	IconNodeJS    = "nodedotjs"
	IconGo        = "go"
	IconRust      = "rust"
	IconGradle    = "gradle"
	IconMaven     = "apachemaven"
	IconCpp       = "cmake"
	IconPython    = "python"
	IconDotnet    = "dotnet"
	IconXcode     = "xcode"
	IconAndroid   = "android"
	IconTemp      = "eraser"
	IconAICache   = "ollama"
	IconElectron  = "electron"
	IconDocker    = "docker"
	IconBrew      = "homebrew"
	IconPHP       = "php"
	IconPip       = "pypi"
	IconRuby      = "rubygems"
	IconFlutter   = "flutter"
	IconTerraform = "terraform"
	IconNuGet     = "nuget"
	IconJetBrains = "jetbrains"
	IconWebBuild  = "vite"
	IconEmulators = "monitor"
)

// Hint raises the risk of items whose path ends with Suffix ("/" separators,
// case-insensitive). Reason is a code the frontend translates.
type Hint struct {
	Suffix string
	Level  safety.Level
	Reason string
}

// Rule defines one cleanup category.
type Rule struct {
	ID    string // Unique category ID
	Icon  string // Icon key for the frontend
	OptIn bool   // true = unchecked by default (e.g. AI cache)

	// Risk is the level of the whole category (empty = safe); hints raise single
	// locations above it. Non-safe categories need a categories.<id>.risk text in
	// the locales.
	Risk  safety.Level
	hints []Hint
	// markers = per-directory-name files that must sit beside a matched directory
	// to prove it belongs to a real project (key: lowercased last name segment;
	// "*.ext" matches by suffix). Without one the item is flagged, not dropped.
	markers map[string][]string
	// KeepRoot = the folder is a container other software expects to exist (Temp,
	// ~/.cache): deletion empties it instead of removing it.
	KeepRoot bool
	// homeSkip = base names a "*" home path must not expand to.
	homeSkip []string

	// dirNames = directory patterns searched inside projects (e.g. "node_modules").
	// A pattern may hold more than one segment ("app/build"), matched against the
	// trailing path segments; the more specific pattern wins over a bare name.
	dirNames []string
	// contentFilters = per-directory-name content checks; key is the lowercased
	// directory name; nil means no filter for that name.
	contentFilters map[string]func(entries []string) bool
	// homePaths = home-relative paths with "/" separators (e.g. "go/pkg/mod").
	// Prefix "win:" or "unix:" for platform-specific entries.
	homePaths []string
	// maxDepth = maximum search depth for dirNames from the scan root.
	maxDepth int
}

// DirNames returns a copy of the directory patterns this rule searches for.
func (r Rule) DirNames() []string { return append([]string(nil), r.dirNames...) }

// MaxDepth returns the maximum search depth.
func (r Rule) MaxDepth() int { return r.maxDepth }

// HasDirName reports whether name matches one of dirNames at the given depth,
// without content checks (fast pre-filter for the walker). It compares a single
// directory name, so a multi-segment pattern is never matched here.
func (r Rule) HasDirName(name string, depth int) bool {
	if depth < 1 || depth > r.maxDepth {
		return false
	}
	lower := strings.ToLower(name)
	for _, d := range r.dirNames {
		if strings.ToLower(d) == lower {
			return true
		}
	}
	return false
}

// MatchDir reports whether a directory named name at depth matches this rule.
// If a content filter exists for that name, the directory entries must pass it.
func (r Rule) MatchDir(name string, depth int, entries []string) bool {
	if !r.HasDirName(name, depth) {
		return false
	}
	if f := r.ContentFilterFor(name); f != nil && !f(entries) {
		return false
	}
	return true
}

// ContentFilterFor returns the content filter for a directory name (may be nil).
func (r Rule) ContentFilterFor(name string) func(entries []string) bool {
	return r.contentFilters[strings.ToLower(name)]
}

// RiskLevel is the worst level the category can produce, for the category picker.
func (r Rule) RiskLevel() safety.Level {
	worst := safety.Safe
	if r.Risk != "" {
		worst = r.Risk
	}
	for _, h := range r.hints {
		worst = safety.Worse(worst, h.Level)
	}
	return worst
}

// Assess grades one found path: the guard and context checks first, then the
// category level, hints for specific locations, and the project-marker check.
func (r Rule) Assess(path string) safety.Assessment {
	a := safety.Assess(path, r.Risk, "categoryNote")
	if a.Level == safety.Blocked {
		return a
	}
	slash := strings.ToLower(filepath.ToSlash(path))
	for _, h := range r.hints {
		if strings.HasSuffix(slash, strings.ToLower(h.Suffix)) {
			a.Add(h.Level, h.Reason)
		}
	}
	if m := r.markers[strings.ToLower(filepath.Base(path))]; len(m) > 0 {
		if !safety.HasAnySibling(filepath.Dir(path), m) {
			a.Add(safety.Caution, "noProjectFile")
		}
	}
	return a
}

// ExpandHome returns the absolute locations of the rule's home paths on this OS,
// with "*" segments expanded and homeSkip names removed. A path that does not
// exist is still returned; the caller decides what to do with it.
func (r Rule) ExpandHome(home string) []string {
	var out []string
	seen := map[string]bool{}
	add := func(p string) {
		if !seen[p] {
			seen[p] = true
			out = append(out, p)
		}
	}
	for _, hp := range r.HomePaths() {
		// Linux tools follow $XDG_*_HOME when the user moved those folders, so a
		// default-location entry also gets its relocated twin.
		for _, alt := range xdgTwins(hp) {
			if !strings.Contains(alt, "*") {
				add(HomeRelToAbs(home, alt))
			}
		}
		abs := HomeRelToAbs(home, hp)
		if !strings.Contains(hp, "*") {
			add(abs)
			continue
		}
		matches, _ := filepath.Glob(abs)
		for _, m := range matches {
			if !r.skipsHome(filepath.Base(m)) {
				add(m)
			}
		}
	}
	return out
}

// xdgBases maps a default home-relative base to the variable that relocates it.
var xdgBases = []struct{ rel, env string }{
	{".cache/", "XDG_CACHE_HOME"},
	{".local/share/", "XDG_DATA_HOME"},
	{".config/", "XDG_CONFIG_HOME"},
}

// xdgTwins returns hp re-based under the XDG variable that governs it, when that
// variable is set to an absolute path. The result is an absolute path, which
// HomeRelToAbs would join to home, so it is returned relative to nothing: callers
// use filepath.IsAbs to tell.
func xdgTwins(hp string) []string {
	for _, b := range xdgBases {
		root := strings.TrimSuffix(b.rel, "/")
		if hp != root && !strings.HasPrefix(hp, b.rel) {
			continue
		}
		base := os.Getenv(b.env)
		if base == "" || !filepath.IsAbs(base) {
			return nil
		}
		if hp == root {
			return []string{filepath.ToSlash(base)}
		}
		return []string{filepath.ToSlash(base) + "/" + strings.TrimPrefix(hp, b.rel)}
	}
	return nil
}

func (r Rule) skipsHome(name string) bool {
	for _, s := range r.homeSkip {
		if strings.EqualFold(s, name) {
			return true
		}
	}
	return false
}

// HomePaths returns home-relative paths valid on the current OS.
func (r Rule) HomePaths() []string {
	var out []string
	for _, hp := range r.homePaths {
		switch {
		case strings.HasPrefix(hp, "win:"):
			if runtime.GOOS == "windows" {
				out = append(out, strings.TrimPrefix(hp, "win:"))
			}
		case strings.HasPrefix(hp, "unix:"):
			if runtime.GOOS != "windows" {
				out = append(out, strings.TrimPrefix(hp, "unix:"))
			}
		default:
			out = append(out, hp)
		}
	}
	return out
}

// MatchHomePath reports whether a home-relative path (with "/" separators)
// exactly matches one of the paths valid on this OS.
func (r Rule) MatchHomePath(rel string) bool {
	for _, hp := range r.HomePaths() {
		if rel == hp {
			return true
		}
	}
	return false
}

// ---- content filters ----

// buildFilter: a generic "build" directory counts as an artifact only when its
// contents look like build output, not source code.
func buildFilter(entries []string) bool {
	if len(entries) == 0 {
		return false
	}
	score := 0
	for _, e := range entries {
		l := strings.ToLower(e)
		switch {
		case strings.HasSuffix(l, ".class"), strings.HasSuffix(l, ".jar"),
			strings.HasSuffix(l, ".apk"), strings.HasSuffix(l, ".aab"),
			strings.HasSuffix(l, ".dex"), strings.HasSuffix(l, ".o"),
			strings.HasSuffix(l, ".obj"), strings.HasSuffix(l, ".a"),
			strings.HasSuffix(l, ".so"), strings.HasSuffix(l, ".dylib"),
			strings.HasSuffix(l, ".dll"), strings.HasSuffix(l, ".exe"),
			strings.HasSuffix(l, ".wasm"), strings.HasSuffix(l, ".tsbuildinfo"),
			l == "cmakefiles", l == "generated", l == "intermediates",
			l == "tmp", l == "kotlin", l == "res", l == "outputs", l == "assets":
			score++
		case strings.HasSuffix(l, ".go"), strings.HasSuffix(l, ".rs"),
			strings.HasSuffix(l, ".swift"), strings.HasSuffix(l, ".kt"),
			strings.HasSuffix(l, ".java"), strings.HasSuffix(l, ".c"),
			strings.HasSuffix(l, ".h"), strings.HasSuffix(l, ".cpp"),
			strings.HasSuffix(l, ".py"), strings.HasSuffix(l, ".rb"),
			strings.HasSuffix(l, ".php"), strings.HasSuffix(l, ".ts"),
			strings.HasSuffix(l, ".tsx"), strings.HasSuffix(l, ".vue"),
			strings.HasSuffix(l, ".dart"):
			score -= 3
		}
	}
	return score > 0
}

// rustTargetFilter: a cargo "target" dir contains debug/release/CACHEDIR.TAG.
func rustTargetFilter(entries []string) bool {
	for _, e := range entries {
		switch strings.ToLower(e) {
		case "debug", "release", "cachedir.tag", ".rustc_info.json":
			return true
		}
	}
	return false
}

// mavenTargetFilter: a Maven "target" dir contains .jar/.class/classes.
func mavenTargetFilter(entries []string) bool {
	for _, e := range entries {
		l := strings.ToLower(e)
		if strings.HasSuffix(l, ".jar") || strings.HasSuffix(l, ".class") ||
			l == "classes" || l == "test-classes" {
			return true
		}
	}
	return false
}

// dotnetFilter: .NET "bin"/"obj" contains Debug/Release/.dll. A bare .exe is not
// enough — hand-made tool folders named "bin" hold those too.
func dotnetFilter(entries []string) bool {
	for _, e := range entries {
		l := strings.ToLower(e)
		if l == "debug" || l == "release" || strings.HasSuffix(l, ".dll") ||
			l == "project.assets.json" || strings.HasSuffix(l, ".csproj.nuget.cache") {
			return true
		}
	}
	return false
}

// venvFilter: only a real virtualenv (it always has pyvenv.cfg), not any folder
// that happens to be called "venv".
func venvFilter(entries []string) bool {
	for _, e := range entries {
		if strings.EqualFold(e, "pyvenv.cfg") {
			return true
		}
	}
	return false
}

// swiftPMFilter: a SwiftPM ".build" directory.
func swiftPMFilter(entries []string) bool {
	for _, e := range entries {
		switch strings.ToLower(e) {
		case "package.swift", "release.yaml", "debug.yaml", "checkouts",
			"arm64-apple-macosx", "x86_64-apple-macosx", "build.db":
			return true
		}
	}
	return false
}

// coverageFilter: a "coverage" folder counts only when it holds a coverage
// report — the name alone is too generic.
func coverageFilter(entries []string) bool {
	for _, e := range entries {
		switch strings.ToLower(e) {
		case "lcov.info", "lcov-report", "coverage-final.json", "clover.xml",
			"cobertura-coverage.xml", "coverage.json", "index.html":
			return true
		}
	}
	return false
}

// Files that prove a matched directory sits in a real project.
var (
	jsProject     = []string{"package.json"}
	gradleProject = []string{"build.gradle", "build.gradle.kts", "settings.gradle", "settings.gradle.kts"}
	dotnetProject = []string{"*.csproj", "*.fsproj", "*.vbproj", "*.sln"}
)

// All returns every category rule.
func All() []Rule {
	return []Rule{
		{
			ID:       CatNodeJS,
			Icon:     IconNodeJS,
			dirNames: []string{"node_modules"},
			maxDepth: 5,
			markers:  map[string][]string{"node_modules": jsProject},
		},
		{
			ID:   CatGo,
			Icon: IconGo,
			homePaths: []string{
				"go/pkg/mod",
				"go/pkg/mod/cache/download",
				"unix:.cache/go-build",
				"unix:Library/Caches/go-build",
				"win:AppData/Local/go-build",
			},
		},
		{
			ID:             CatRust,
			Icon:           IconRust,
			dirNames:       []string{"target"},
			maxDepth:       4,
			contentFilters: map[string]func([]string) bool{"target": rustTargetFilter},
			markers:        map[string][]string{"target": {"Cargo.toml"}},
			// CARGO_HOME defaults to ~/.cargo on every OS, %USERPROFILE%\.cargo on Windows.
			homePaths: []string{".cargo/registry"},
		},
		{
			ID:             CatGradle,
			Icon:           IconGradle,
			dirNames:       []string{"build"},
			maxDepth:       4,
			contentFilters: map[string]func([]string) bool{"build": buildFilter},
			markers:        map[string][]string{"build": gradleProject},
			homePaths:      []string{"unix:.gradle/caches", "win:.gradle/caches"},
		},
		{
			ID:             CatMaven,
			Icon:           IconMaven,
			homePaths:      []string{".m2/repository"},
			dirNames:       []string{"target"},
			maxDepth:       4,
			contentFilters: map[string]func([]string) bool{"target": mavenTargetFilter},
			markers:        map[string][]string{"target": {"pom.xml"}},
			// Artifacts put there by `mvn install` exist nowhere else.
			hints: []Hint{{".m2/repository", safety.Caution, "localArtifacts"}},
		},
		{
			ID:       CatCpp,
			Icon:     IconCpp,
			dirNames: []string{"CMakeFiles", "cmake-build-debug", "cmake-build-release"},
			maxDepth: 4,
			markers: map[string][]string{
				"cmakefiles":          {"CMakeCache.txt"},
				"cmake-build-debug":   {"CMakeLists.txt"},
				"cmake-build-release": {"CMakeLists.txt"},
			},
		},
		{
			ID:             CatPython,
			Icon:           IconPython,
			dirNames:       []string{"__pycache__", ".pytest_cache", ".mypy_cache", ".ruff_cache", ".venv", "venv"},
			maxDepth:       5,
			contentFilters: map[string]func([]string) bool{".venv": venvFilter, "venv": venvFilter},
			// A virtualenv can hold packages that were never written to a requirements file.
			hints: []Hint{{"/.venv", safety.Caution, "virtualenv"}, {"/venv", safety.Caution, "virtualenv"}},
		},
		{
			ID:             CatDotnet,
			Icon:           IconDotnet,
			dirNames:       []string{"bin", "obj"},
			maxDepth:       4,
			contentFilters: map[string]func([]string) bool{"bin": dotnetFilter, "obj": dotnetFilter},
			markers:        map[string][]string{"bin": dotnetProject, "obj": dotnetProject},
		},
		{
			ID:   CatXcode,
			Icon: IconXcode,
			homePaths: []string{
				"unix:Library/Developer/Xcode/DerivedData",
				"unix:Library/Developer/Xcode/iOS DeviceSupport",
				"unix:Library/Developer/CoreSimulator/Caches",
				"unix:Library/Caches/com.apple.dt.Xcode",
			},
			dirNames:       []string{".build"},
			maxDepth:       3,
			contentFilters: map[string]func([]string) bool{".build": swiftPMFilter},
			markers:        map[string][]string{".build": {"Package.swift"}},
		},
		{
			ID:        CatAndroid,
			Icon:      IconAndroid,
			homePaths: []string{"unix:.android/build-cache", "win:.android/build-cache"},
			dirNames:  []string{"app/build", ".cxx"},
			maxDepth:  4,
			markers:   map[string][]string{"build": gradleProject},
		},
		{
			ID:        CatTemp,
			Icon:      IconTemp,
			OptIn:     true,
			Risk:      safety.Caution, // running apps keep live files here
			KeepRoot:  true,
			homePaths: []string{".cache", "win:AppData/Local/Temp", "win:AppData/Local/CrashDumps", "win:AppData/Local/Microsoft/Windows/INetCache"},
		},
		{
			ID:    CatAICache,
			Icon:  IconAICache,
			OptIn: true,
			Risk:  safety.Caution, // multi-gigabyte downloads; custom Ollama models cannot be re-pulled
			homePaths: []string{
				".cache/huggingface",
				".cache/torch",
				".cache/whisper",
				".cache/clip",
				".cache/modelscope",
				"unix:.ollama/models",
				"win:.ollama/models",
			},
		},
		{
			ID:    CatElectron,
			Icon:  IconElectron,
			OptIn: true,
			Risk:  safety.Caution,
			homePaths: []string{
				"unix:Library/Caches/electron",
				"win:AppData/Local/electron/Cache",
				".npm/_cacache",
				".npm/_npx",
				"win:AppData/Local/npm-cache", // npm's cache default on Windows
				".cache/yarn",
				"unix:Library/Caches/Yarn",
				"win:AppData/Local/Yarn/Cache",
				".yarn/berry/cache",
				".cache/pnpm",
				"unix:.local/share/pnpm/store",
				"unix:Library/pnpm/store",
				"win:AppData/Local/pnpm/store",
				"unix:Library/Caches/ms-playwright",
				".cache/ms-playwright",
				"unix:Library/Caches/Cypress",
				".cache/Cypress",
			},
		},
		{
			ID:   CatDocker,
			Icon: IconDocker,
			homePaths: []string{
				"unix:.docker/buildx",
				"unix:Library/Caches/com.docker.docker",
				"unix:Library/Containers/com.docker.docker/Data/log",
				// Not the whole AppData/Local/Docker: its wsl/ folder holds the disk
				// image with every image, container and volume.
				"win:AppData/Local/Docker/log",
			},
			// buildx keeps the definitions of custom builders next to its cache.
			hints: []Hint{{".docker/buildx", safety.Caution, "builderConfig"}},
		},
		{
			ID:   CatBrew,
			Icon: IconBrew,
			homePaths: []string{
				"unix:Library/Caches/Homebrew",
				"unix:.cache/homebrew",
			},
		},
		{
			ID:   CatPHP,
			Icon: IconPHP,
			homePaths: []string{
				"unix:.composer/cache",
				"unix:Library/Caches/composer",
				"win:AppData/Local/Composer",
			},
		},
		{
			ID:   CatPip,
			Icon: IconPip,
			homePaths: []string{
				".cache/pip",
				"unix:Library/Caches/pip",
				"win:AppData/Local/pip/Cache",
				"win:AppData/Local/uv/cache",
				"win:AppData/Local/pypoetry/Cache",
				".cache/uv",
				"unix:Library/Caches/uv",
				".local/share/uv/cache",
				"unix:Library/Caches/pypoetry",
				".cache/pypoetry",
				".conda/pkgs",
			},
		},
		{
			ID:    CatRuby,
			Icon:  IconRuby,
			OptIn: true,
			Risk:  safety.Caution,
			// ~/.gem holds the gems installed with --user-install: not a cache.
			hints: []Hint{{"/.gem", safety.Danger, "installedPackages"}},
			homePaths: []string{
				"unix:Library/Caches/Gem",
				"unix:.gem",
				"win:AppData/Local/RubyGems",
			},
		},
		{
			ID:       CatFlutter,
			Icon:     IconFlutter,
			dirNames: []string{".dart_tool"},
			maxDepth: 4,
			markers:  map[string][]string{".dart_tool": {"pubspec.yaml"}},
			// Tools activated with `dart pub global activate` live under .pub-cache.
			hints: []Hint{{".pub-cache/hosted", safety.Caution, "globalTools"}},
			homePaths: []string{
				".pub-cache/hosted",
				"unix:Library/Caches/flutter_engine",
				"unix:.flutter/bin/cache/artifacts",
			},
		},
		{
			ID:       CatTerraform,
			Icon:     IconTerraform,
			dirNames: []string{".terraform"},
			maxDepth: 4,
			markers:  map[string][]string{".terraform": {"*.tf", "*.tf.json"}},
			homePaths: []string{
				".terraform.d/plugin-cache",
				"unix:.cache/tf-plugin-fetch",
			},
		},
		{
			ID:   CatNuGet,
			Icon: IconNuGet,
			homePaths: []string{
				".nuget/packages",
				"unix:.local/share/NuGet",
				"win:AppData/NuGet",
				"win:AppData/Local/NuGet/v3-cache",
			},
		},
		{
			ID:   CatJetBrains,
			Icon: IconJetBrains,
			homePaths: []string{
				"unix:Library/Caches/JetBrains",
				// One folder per IDE — never the parent: Toolbox keeps the installed IDEs there.
				"win:AppData/Local/JetBrains/*",
				".cache/JetBrains",
			},
			homeSkip: []string{"Toolbox"},
		},
		{
			ID:             CatWebBuild,
			Icon:           IconWebBuild,
			dirNames:       []string{".next", ".nuxt", ".turbo", ".vite", ".parcel-cache", ".svelte-kit", ".astro", "coverage", "storybook-static", "docusaurus-plugin-debug"},
			maxDepth:       4,
			contentFilters: map[string]func([]string) bool{"coverage": coverageFilter},
			markers: map[string][]string{
				".next": jsProject, ".nuxt": jsProject, ".turbo": jsProject, ".vite": jsProject,
				".parcel-cache": jsProject, ".svelte-kit": jsProject, ".astro": jsProject,
				"storybook-static": jsProject, "docusaurus-plugin-debug": jsProject,
			},
		},
		{
			ID:    CatEmulators,
			Icon:  IconEmulators,
			OptIn: true,
			Risk:  safety.Caution,
			// Virtual devices carry installed apps and user data; only the dyld cache regenerates.
			hints: []Hint{
				{".android/avd", safety.Danger, "emulatorData"},
				{"Library/Developer/CoreSimulator/Devices", safety.Danger, "emulatorData"},
			},
			homePaths: []string{
				".android/avd",
				"unix:Library/Developer/CoreSimulator/Devices",
				"unix:Library/Developer/CoreSimulator/Caches/dyld",
				"win:AppData/Local/Android/avd",
			},
		},
	}
}

// ByID looks up a rule by ID; nil when missing.
func ByID(id string) *Rule {
	for _, r := range All() {
		if r.ID == id {
			return &r
		}
	}
	return nil
}

// HomeRelToAbs converts a home-relative path ("/" separators) to absolute.
func HomeRelToAbs(homeDir, rel string) string {
	if filepath.IsAbs(filepath.FromSlash(rel)) {
		return filepath.Clean(filepath.FromSlash(rel)) // an XDG-relocated location
	}
	return filepath.Join(homeDir, filepath.FromSlash(rel))
}
