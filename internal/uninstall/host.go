package uninstall

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// ErrUnavailable is returned by a provider whose tool is not installed or does
// not apply to this OS. It is not an error worth showing.
var ErrUnavailable = errors.New("provider unavailable")

// Env is what path specs need to know about the machine. Tests replace it.
type Env struct {
	OS   string
	Home string
	Vars map[string]string // {NAME} placeholders (APPDATA, LOCALAPPDATA, ...)
}

// NewEnv reads the real machine.
func NewEnv() Env {
	home, _ := os.UserHomeDir()
	vars := map[string]string{}
	for _, k := range []string{
		"APPDATA", "LOCALAPPDATA", "PROGRAMDATA", "PROGRAMFILES", "PROGRAMFILES(X86)", "USERPROFILE",
		"XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_CACHE_HOME", "XDG_STATE_HOME",
	} {
		vars[k] = os.Getenv(k)
	}
	return Env{OS: runtime.GOOS, Home: home, Vars: vars}
}

// Expand turns a path spec into concrete paths for this OS:
//
//	"~/x"            home-relative
//	"{APPDATA}/x"    environment placeholder
//	"mac:", "win:", "linux:", "unix:"   only on that OS (unix = mac or linux)
//	"*"              glob within one segment
//
// It returns nil when the spec belongs to another OS or a placeholder is unset.
func (e Env) Expand(spec string) []string {
	for prefix, ok := range map[string]bool{
		"mac:":   e.OS == "darwin",
		"win:":   e.OS == "windows",
		"linux:": e.OS == "linux",
		"unix:":  e.OS != "windows",
	} {
		if strings.HasPrefix(spec, prefix) {
			if !ok {
				return nil
			}
			spec = strings.TrimPrefix(spec, prefix)
			break
		}
	}
	switch {
	case strings.HasPrefix(spec, "~/"):
		if e.Home == "" {
			return nil
		}
		spec = filepath.Join(e.Home, filepath.FromSlash(spec[2:]))
	case strings.HasPrefix(spec, "{"):
		end := strings.Index(spec, "}")
		if end < 0 {
			return nil
		}
		name := strings.ToUpper(spec[1:end])
		base := e.Vars[name]
		if base == "" {
			base = e.xdgDefault(name)
		}
		if base == "" {
			return nil
		}
		spec = filepath.Join(base, filepath.FromSlash(strings.TrimLeft(spec[end+1:], "/\\")))
	default:
		spec = filepath.FromSlash(spec)
	}
	if !strings.Contains(spec, "*") {
		return []string{spec}
	}
	matches, _ := filepath.Glob(spec)
	return matches
}

// xdgDefault is the XDG Base Directory Specification's fallback for an unset or
// empty variable. Linux tools honour $XDG_*_HOME when the user moved them, so
// specs use {XDG_CONFIG_HOME} and friends instead of a hard-coded ~/.config.
func (e Env) xdgDefault(name string) string {
	rel := map[string]string{
		"XDG_CONFIG_HOME": ".config",
		"XDG_DATA_HOME":   ".local/share",
		"XDG_CACHE_HOME":  ".cache",
		"XDG_STATE_HOME":  ".local/state",
	}[name]
	if rel == "" || e.Home == "" {
		return ""
	}
	return filepath.Join(e.Home, filepath.FromSlash(rel))
}

// Runner starts processes. The real one resolves tools against the user's shell
// PATH; tests use a fake.
type Runner interface {
	// Look returns the absolute path of a tool, or an error when it is not installed.
	Look(name string) (string, error)
	// Run executes a program without a shell and returns its standard output.
	Run(ctx context.Context, name string, args ...string) ([]byte, error)
}

// ExecRunner is the Runner that starts real processes.
type ExecRunner struct {
	once sync.Once
	path string
}

var _ Runner = (*ExecRunner)(nil)

// searchPath is the PATH a GUI app should use. An app started from Finder or the
// Start menu inherits a minimal PATH, without Homebrew, nvm or cargo — so the
// login shell is asked for the PATH the user's terminal has.
func (r *ExecRunner) searchPath() string {
	r.once.Do(func() {
		r.path = os.Getenv("PATH")
		if runtime.GOOS == "windows" {
			return
		}
		home, _ := os.UserHomeDir()
		extra := []string{
			"/opt/homebrew/bin", "/opt/homebrew/sbin", "/usr/local/bin", "/usr/local/sbin",
			filepath.Join(home, ".cargo", "bin"), filepath.Join(home, ".volta", "bin"),
			filepath.Join(home, "go", "bin"), filepath.Join(home, ".local", "bin"),
		}
		r.path = strings.Join(append([]string{r.path}, extra...), string(os.PathListSeparator))
		if sh := os.Getenv("SHELL"); sh != "" {
			ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
			defer cancel()
			out, err := exec.CommandContext(ctx, sh, "-ilc", `printf '__BD__%s__BD__' "$PATH"`).Output()
			if err == nil {
				s := string(out)
				if i := strings.Index(s, "__BD__"); i >= 0 {
					if j := strings.Index(s[i+6:], "__BD__"); j >= 0 {
						r.path = s[i+6:i+6+j] + string(os.PathListSeparator) + r.path
					}
				}
			}
		}
	})
	return r.path
}

// Look implements Runner.
func (r *ExecRunner) Look(name string) (string, error) {
	if filepath.IsAbs(name) {
		if isExec(name) {
			return name, nil
		}
		return "", fmt.Errorf("%s: %w", name, ErrUnavailable)
	}
	exts := []string{""}
	if runtime.GOOS == "windows" {
		exts = []string{".exe", ".cmd", ".bat", ".com"}
	}
	for _, dir := range filepath.SplitList(r.searchPath()) {
		for _, ext := range exts {
			if p := filepath.Join(dir, name+ext); isExec(p) {
				return p, nil
			}
		}
	}
	return "", fmt.Errorf("%s: %w", name, ErrUnavailable)
}

func isExec(p string) bool {
	st, err := os.Stat(p)
	if err != nil || st.IsDir() {
		return false
	}
	return runtime.GOOS == "windows" || st.Mode()&0o111 != 0
}

// Run implements Runner.
func (r *ExecRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = append(os.Environ(), "PATH="+r.searchPath())
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			if len(msg) > 400 {
				msg = msg[len(msg)-400:]
			}
			return out, fmt.Errorf("%w: %s", err, msg)
		}
	}
	return out, err
}

// Host is what providers get to work with.
type Host struct {
	Env
	Run Runner
}
