// Package rules defines cleanup categories and path matchers.
package rules

import (
	"path/filepath"
	"runtime"
	"strings"
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

// Rule defines one cleanup category.
type Rule struct {
	ID    string // Unique category ID
	Icon  string // Icon key for the frontend
	OptIn bool   // true = unchecked by default (e.g. AI cache)

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

// dotnetFilter: .NET "bin"/"obj" contains Debug/Release/.dll.
func dotnetFilter(entries []string) bool {
	for _, e := range entries {
		l := strings.ToLower(e)
		if l == "debug" || l == "release" || strings.HasSuffix(l, ".dll") ||
			strings.HasSuffix(l, ".exe") || l == "ref" || strings.HasSuffix(l, ".cache") {
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

// All returns every category rule.
func All() []Rule {
	return []Rule{
		{
			ID:       CatNodeJS,
			Icon:     IconNodeJS,
			dirNames: []string{"node_modules"},
			maxDepth: 5,
		},
		{
			ID:   CatGo,
			Icon: IconGo,
			homePaths: []string{
				"go/pkg/mod",
				"go/pkg/mod/cache/download",
				"unix:.cache/go-build",
				"unix:Library/Caches/go-build",
			},
		},
		{
			ID:             CatRust,
			Icon:           IconRust,
			dirNames:       []string{"target"},
			maxDepth:       4,
			contentFilters: map[string]func([]string) bool{"target": rustTargetFilter},
			homePaths:      []string{"unix:.cargo/registry", "win:AppData/Local/Cargo/registry"},
		},
		{
			ID:             CatGradle,
			Icon:           IconGradle,
			dirNames:       []string{"build"},
			maxDepth:       4,
			contentFilters: map[string]func([]string) bool{"build": buildFilter},
			homePaths:      []string{"unix:.gradle/caches", "win:.gradle/caches"},
		},
		{
			ID:             CatMaven,
			Icon:           IconMaven,
			homePaths:      []string{".m2/repository"},
			dirNames:       []string{"target"},
			maxDepth:       4,
			contentFilters: map[string]func([]string) bool{"target": mavenTargetFilter},
		},
		{
			ID:       CatCpp,
			Icon:     IconCpp,
			dirNames: []string{"CMakeFiles", "cmake-build-debug", "cmake-build-release"},
			maxDepth: 4,
		},
		{
			ID:       CatPython,
			Icon:     IconPython,
			dirNames: []string{"__pycache__", ".pytest_cache", ".mypy_cache", ".ruff_cache", ".venv", "venv"},
			maxDepth: 5,
		},
		{
			ID:             CatDotnet,
			Icon:           IconDotnet,
			dirNames:       []string{"bin", "obj"},
			maxDepth:       4,
			contentFilters: map[string]func([]string) bool{"bin": dotnetFilter, "obj": dotnetFilter},
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
		},
		{
			ID:        CatAndroid,
			Icon:      IconAndroid,
			homePaths: []string{"unix:.android/build-cache", "win:.android/build-cache"},
			dirNames:  []string{"app/build", ".cxx"},
			maxDepth:  4,
		},
		{
			ID:        CatTemp,
			Icon:      IconTemp,
			OptIn:     true,
			homePaths: []string{".cache", "win:AppData/Local/Temp", "win:AppData/Local/CrashDumps", "win:AppData/Local/Microsoft/Windows/INetCache"},
		},
		{
			ID:    CatAICache,
			Icon:  IconAICache,
			OptIn: true,
			homePaths: []string{
				".cache/huggingface",
				".cache/torch",
				".cache/whisper",
				".cache/clip",
				".cache/modelscope",
				"unix:.ollama/models",
				"win:.ollama/models",
				"win:AppData/Local/pip/cache",
			},
		},
		{
			ID:    CatElectron,
			Icon:  IconElectron,
			OptIn: true,
			homePaths: []string{
				"unix:Library/Caches/electron",
				"win:AppData/Local/electron/Cache",
				".npm/_cacache",
				".npm/_npx",
				".cache/yarn",
				".cache/pnpm",
				"unix:.local/share/pnpm/store",
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
				"win:AppData/Local/Docker",
			},
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
				"win:AppData/Local/JetBrains",
				".cache/JetBrains",
			},
		},
		{
			ID:       CatWebBuild,
			Icon:     IconWebBuild,
			dirNames: []string{".next", ".nuxt", ".turbo", ".vite", ".parcel-cache", ".svelte-kit", ".astro", "coverage", "storybook-static", "docusaurus-plugin-debug"},
			maxDepth: 4,
		},
		{
			ID:    CatEmulators,
			Icon:  IconEmulators,
			OptIn: true,
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
	return filepath.Join(homeDir, filepath.FromSlash(rel))
}
