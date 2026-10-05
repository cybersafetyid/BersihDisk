// Package safety decides how dangerous it is to delete a path.
//
// It has two layers, both shared by the cleaner and the uninstaller:
//
//   - Guard is a hard block. Paths it refuses are never deleted, whatever the UI
//     or a rule says: filesystem roots, the home folder, personal data folders,
//     credentials, and operating-system trees.
//   - Assess grades everything else (safe / caution / danger) from the path's
//     surroundings — inside an app bundle, inside a language runtime, in a global
//     package folder — so the UI can warn before something that is not a
//     regenerable cache is removed.
//
// Reasons are stable codes (no prose); the frontend translates them.
package safety

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Level is how risky deleting a path is.
type Level string

// Levels, ordered from harmless to forbidden.
const (
	Safe    Level = "safe"    // regenerated automatically, no data lost
	Caution Level = "caution" // costs a re-download or may hold something unique
	Danger  Level = "danger"  // can break an installed app, toolchain or user data
	Blocked Level = "blocked" // never deleted
)

func (l Level) rank() int {
	switch l {
	case Caution:
		return 1
	case Danger:
		return 2
	case Blocked:
		return 3
	}
	return 0
}

// Worse returns the more severe of two levels.
func Worse(a, b Level) Level {
	if b.rank() > a.rank() {
		return b
	}
	return a
}

// NeedsAck reports whether deleting at this level needs the user's explicit
// acknowledgement.
func (l Level) NeedsAck() bool { return l.rank() >= Caution.rank() && l != Blocked }

// Assessment is a level together with the reason codes behind it.
type Assessment struct {
	Level   Level    `json:"level"`
	Reasons []string `json:"reasons,omitempty"`
}

// Add raises the level to at least l and records reason (once).
func (a *Assessment) Add(l Level, reason string) {
	if a.Level == "" {
		a.Level = Safe
	}
	a.Level = Worse(a.Level, l)
	if reason == "" {
		return
	}
	for _, r := range a.Reasons {
		if r == reason {
			return
		}
	}
	a.Reasons = append(a.Reasons, reason)
}

// ---- hard guard ----

// caseFold reports whether this OS treats paths case-insensitively by default.
var caseFold = runtime.GOOS == "windows" || runtime.GOOS == "darwin"

func norm(p string) string {
	p = filepath.Clean(p)
	if caseFold {
		p = strings.ToLower(p)
	}
	return p
}

// under reports whether child is parent or lies below it (both normalised).
func under(child, parent string) bool {
	if child == parent {
		return true
	}
	if !strings.HasSuffix(parent, string(filepath.Separator)) {
		parent += string(filepath.Separator)
	}
	return strings.HasPrefix(child, parent)
}

type guardEntry struct {
	path   string
	reason string
}

// protectedSets builds the guard lists for the current user.
//
//	exact: the folder itself, and any folder above it, is off limits — but its
//	       contents are not (a project inside ~/Documents is still cleanable).
//	tree:  nothing inside may be deleted either.
func protectedSets() (exact, tree []guardEntry) {
	home, _ := os.UserHomeDir()
	addExact := func(reason string, paths ...string) {
		for _, p := range paths {
			if p != "" {
				exact = append(exact, guardEntry{norm(p), reason})
			}
		}
	}
	addTree := func(reason string, paths ...string) {
		for _, p := range paths {
			if p != "" {
				tree = append(tree, guardEntry{norm(p), reason})
			}
		}
	}
	inHome := func(rel ...string) []string {
		out := make([]string, 0, len(rel))
		if home == "" {
			return out
		}
		for _, r := range rel {
			out = append(out, filepath.Join(home, filepath.FromSlash(r)))
		}
		return out
	}

	addExact("homeFolder", home)
	addExact("userData", inHome(
		"Documents", "Desktop", "Downloads", "Pictures", "Music", "Movies", "Videos",
		"Public", "Library", "Applications", ".config", ".local", ".local/share",
		"AppData", "AppData/Local", "AppData/Roaming",
	)...)
	addTree("credentials", inHome(
		".ssh", ".gnupg", ".aws", ".azure", ".kube", ".config/gcloud", ".config/gh",
		".password-store", "Library/Keychains", "Library/Mail", "Library/Messages",
	)...)

	if runtime.GOOS == "windows" {
		for _, env := range []string{"SystemRoot", "ProgramFiles", "ProgramFiles(x86)", "ProgramW6432"} {
			addTree("systemFolder", os.Getenv(env))
		}
		addExact("systemFolder", os.Getenv("SystemDrive")+`\Users`, os.Getenv("ProgramData"))
	} else {
		addExact("systemFolder",
			"/Users", "/home", "/Volumes", "/Applications", "/Library", "/opt", "/usr",
			"/usr/local", "/var", "/tmp", "/private", "/root", "/mnt", "/media",
		)
		addTree("systemFolder",
			"/System", "/bin", "/sbin", "/usr/bin", "/usr/sbin", "/usr/lib", "/usr/libexec",
			"/usr/share", "/etc", "/dev", "/proc", "/sys", "/boot", "/private/etc",
			"/private/var/db", "/Library/Apple",
		)
	}
	return withResolved(exact), withResolved(tree)
}

// withResolved adds the symlink-resolved twin of every entry that exists, so a
// home folder or system path reached through a link (macOS /var -> /private/var,
// a home on another volume) is guarded under both spellings.
func withResolved(list []guardEntry) []guardEntry {
	out := list
	for _, e := range list {
		if real, err := filepath.EvalSymlinks(e.path); err == nil && norm(real) != e.path {
			out = append(out, guardEntry{norm(real), e.reason})
		}
	}
	return out
}

// guardWith is Guard against an already-built set of protected paths, so a caller
// that checks many paths (a directory listing) can build the set once.
func guardWith(path string, exact, tree []guardEntry) (reason string, blocked bool) {
	if path == "" || !filepath.IsAbs(path) {
		return "relativePath", true
	}
	candidates := []string{norm(path)}
	if real, err := filepath.EvalSymlinks(path); err == nil && norm(real) != candidates[0] {
		candidates = append(candidates, norm(real))
	}
	for _, p := range candidates {
		if filepath.Dir(p) == p {
			return "filesystemRoot", true
		}
		for _, e := range exact {
			if under(e.path, p) { // path is the entry, or an ancestor of it
				return e.reason, true
			}
		}
		for _, e := range tree {
			if under(p, e.path) {
				return e.reason, true
			}
		}
	}
	return "", false
}

// Guard reports whether path must never be deleted, with the reason code. A
// symlink is checked twice — as written and where it points — so a link into a
// protected folder cannot be used to get around the lists.
func Guard(path string) (reason string, blocked bool) {
	exact, tree := protectedSets()
	return guardWith(path, exact, tree)
}

// PermissionHint names a known, fixable cause of a failed delete. On macOS,
// files under ~/Library/Containers and ~/Library/Group Containers are protected
// by the OS (TCC/SIP): removal fails with "Operation not permitted" even for the
// owner and for sudo, until the app is granted Full Disk Access.
func PermissionHint(goos string, err error) string {
	if goos != "darwin" || err == nil {
		return ""
	}
	if errors.Is(err, fs.ErrPermission) || strings.Contains(strings.ToLower(err.Error()), "operation not permitted") {
		return "fullDiskAccess"
	}
	return ""
}

// ---- context assessment ----

// components lowercases and splits a path into its segments.
func components(path string) []string {
	p := strings.ToLower(filepath.ToSlash(filepath.Clean(path)))
	return strings.FieldsFunc(p, func(r rune) bool { return r == '/' })
}

func hasAny(comps []string, names ...string) bool {
	for _, c := range comps {
		for _, n := range names {
			if c == n {
				return true
			}
		}
	}
	return false
}

// hasSeq reports whether the segments contain a, b as consecutive elements.
func hasSeq(comps []string, a, b string) bool {
	for i := 0; i+1 < len(comps); i++ {
		if comps[i] == a && comps[i+1] == b {
			return true
		}
	}
	return false
}

// versionManagers hold whole language runtimes; anything inside is installed
// software, not a project.
var versionManagers = []string{
	".nvm", ".fnm", ".volta", ".rustup", ".pyenv", ".rbenv", ".asdf", ".sdkman",
	".jenv", ".gvm", ".nodenv", ".rvm", "pyenv-win", "cellar",
}

// editorHomes are per-user editor folders whose "extensions" are installed plug-ins.
var editorHomes = []string{
	".vscode", ".vscode-insiders", ".vscode-oss", ".cursor", ".windsurf",
}

// installPrefixes are directory trees where package managers put installed
// software; a dependency folder here belongs to an installed tool.
var installPrefixes = [][]string{
	{"opt"}, {"usr", "local"}, {"home", "linuxbrew"}, {"nix"}, {"snap"},
}

// Context grades a path from where it sits. It never returns Blocked — that is
// Guard's job — and returns an empty assessment for an ordinary project path.
func Context(path string) Assessment {
	var a Assessment
	comps := components(path)
	base := ""
	parent := ""
	if n := len(comps); n > 0 {
		base = comps[n-1]
		if n > 1 {
			parent = comps[n-2]
		}
	}

	for _, c := range comps {
		if strings.HasSuffix(c, ".app") {
			a.Add(Danger, "appBundle")
			break
		}
	}
	if hasAny(comps, versionManagers...) {
		a.Add(Danger, "runtimeInstall")
	}
	if hasAny(comps, ".vscode-server") {
		a.Add(Danger, "editorExtension")
	}
	for _, e := range editorHomes {
		if hasSeq(comps, e, "extensions") {
			a.Add(Danger, "editorExtension")
		}
	}
	// <prefix>/lib/node_modules and %APPDATA%\npm\node_modules are global installs.
	if base == "node_modules" && (parent == "lib" || parent == "lib64" || parent == "npm") {
		a.Add(Danger, "globalPackages")
	}
	if len(comps) > 0 && runtime.GOOS != "windows" {
		for _, pre := range installPrefixes {
			if len(comps) >= len(pre) && equal(comps[:len(pre)], pre) {
				a.Add(Danger, "systemInstall")
			}
		}
	}
	return a
}

func equal(a, b []string) bool {
	for i := range b {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// assess is Assess against an already-built set of protected paths.
func assess(path string, base Level, baseReasons []string, exact, tree []guardEntry) Assessment {
	if reason, blocked := guardWith(path, exact, tree); blocked {
		return Assessment{Level: Blocked, Reasons: []string{reason}}
	}
	a := Assessment{Level: Safe}
	if base != "" && base != Safe {
		for _, r := range baseReasons {
			a.Add(base, r)
		}
		a.Add(base, "")
	}
	for _, r := range Context(path).Reasons {
		a.Add(Danger, r)
	}
	return a
}

// Assess combines the hard guard, the context checks and a starting level from
// the rule that found the path.
func Assess(path string, base Level, baseReasons ...string) Assessment {
	exact, tree := protectedSets()
	return assess(path, base, baseReasons, exact, tree)
}

// AssessBatch grades many plain paths (no rule-derived starting level) with the
// protected-path set built once. The drive analyzer lists directories that can
// hold thousands of children, and rebuilding the set for each entry is the
// difference between a responsive listing and a slow one.
func AssessBatch(paths []string) []Assessment {
	exact, tree := protectedSets()
	out := make([]Assessment, len(paths))
	for i, p := range paths {
		out[i] = assess(p, "", nil, exact, tree)
		if out[i].Level == "" {
			out[i].Level = Safe
		}
	}
	return out
}

// HasAnySibling reports whether dir holds an entry matching one of the patterns
// (exact names, or "*.ext" suffix patterns), compared case-insensitively.
func HasAnySibling(dir string, patterns []string) bool {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range ents {
		name := strings.ToLower(e.Name())
		for _, p := range patterns {
			p = strings.ToLower(p)
			if strings.HasPrefix(p, "*") {
				if strings.HasSuffix(name, p[1:]) {
					return true
				}
			} else if name == p {
				return true
			}
		}
	}
	return false
}
