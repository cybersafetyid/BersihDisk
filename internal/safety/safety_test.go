package safety

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestGuardBlocksProtectedPaths(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	blocked := []string{
		string(filepath.Separator),
		home,
		filepath.Join(home, "Documents"),
		filepath.Join(home, ".ssh"),
		filepath.Join(home, ".ssh", "id_ed25519"),
		filepath.Dir(home), // an ancestor of the home folder
		"relative/path",
		"",
	}
	if runtime.GOOS != "windows" {
		blocked = append(blocked, "/System/Library", "/usr/bin/env", "/etc/hosts", "/Applications")
	}
	for _, p := range blocked {
		if reason, ok := Guard(p); !ok {
			t.Errorf("Guard(%q) allowed the path", p)
		} else if reason == "" {
			t.Errorf("Guard(%q) gave no reason", p)
		}
	}
}

func TestGuardAllowsProjectFolders(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	allowed := []string{
		filepath.Join(home, "Documents", "app", "node_modules"), // a project inside a data folder
		filepath.Join(home, ".cache"),
		filepath.Join(home, ".cargo", "registry"),
		filepath.Join(home, "dev", "app", "target"),
	}
	for _, p := range allowed {
		if reason, ok := Guard(p); ok {
			t.Errorf("Guard(%q) blocked a cleanable path: %s", p, reason)
		}
	}
}

func TestGuardFollowsSymlinkIntoProtectedFolder(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks need privileges on Windows")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	ssh := filepath.Join(home, ".ssh")
	if err := os.Mkdir(ssh, 0o700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "innocent")
	if err := os.Symlink(ssh, link); err != nil {
		t.Fatal(err)
	}
	if _, ok := Guard(link); !ok {
		t.Fatal("a symlink to ~/.ssh must be blocked")
	}
}

func TestContextFlagsInstalledSoftware(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("posix-style paths")
	}
	cases := map[string]string{
		"/Applications/Slack.app/Contents/Resources/node_modules": "appBundle",
		"/Users/a/.nvm/versions/node/v20/lib/node_modules":        "runtimeInstall",
		"/opt/homebrew/lib/node_modules":                          "globalPackages",
		"/Users/a/.vscode/extensions/foo-1.0/node_modules":        "editorExtension",
		"/usr/local/lib/node_modules":                             "systemInstall",
	}
	for path, want := range cases {
		a := Context(path)
		if a.Level != Danger {
			t.Errorf("%s: level = %q, want danger", path, a.Level)
		}
		found := false
		for _, r := range a.Reasons {
			found = found || r == want
		}
		if !found {
			t.Errorf("%s: reasons %v lack %q", path, a.Reasons, want)
		}
	}
	if a := Context("/Users/a/dev/site/node_modules"); a.Level != "" || len(a.Reasons) != 0 {
		t.Errorf("an ordinary project folder was flagged: %+v", a)
	}
}

func TestAssessCombinesBaseAndContext(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	project := filepath.Join(home, "dev", "app", "node_modules")

	if a := Assess(project, Safe); a.Level != Safe || len(a.Reasons) != 0 {
		t.Errorf("plain project = %+v", a)
	}
	a := Assess(project, Caution, "mixedContent")
	if a.Level != Caution || len(a.Reasons) != 1 || a.Reasons[0] != "mixedContent" {
		t.Errorf("caution base = %+v", a)
	}
	if a := Assess(filepath.Join(home, ".ssh"), Safe); a.Level != Blocked {
		t.Errorf("guarded path = %+v, want blocked", a)
	}
}

func TestHasAnySibling(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []string{"Package.JSON", "App.csproj"} {
		if err := os.WriteFile(filepath.Join(dir, n), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if !HasAnySibling(dir, []string{"package.json"}) {
		t.Error("exact match is case-insensitive")
	}
	if !HasAnySibling(dir, []string{"*.csproj"}) {
		t.Error("suffix pattern")
	}
	if HasAnySibling(dir, []string{"Cargo.toml", "*.sln"}) {
		t.Error("no marker present")
	}
}

func TestWorseAndNeedsAck(t *testing.T) {
	if Worse(Safe, Danger) != Danger || Worse(Danger, Caution) != Danger {
		t.Error("Worse")
	}
	if Safe.NeedsAck() || !Caution.NeedsAck() || !Danger.NeedsAck() || Blocked.NeedsAck() {
		t.Error("NeedsAck")
	}
}

func TestPermissionHint(t *testing.T) {
	denied := &fs.PathError{Op: "unlinkat", Path: "/x", Err: errors.New("operation not permitted")}
	if PermissionHint("darwin", denied) != "fullDiskAccess" {
		t.Error("macOS protection must be recognised")
	}
	if PermissionHint("darwin", fs.ErrPermission) != "fullDiskAccess" {
		t.Error("fs.ErrPermission on macOS")
	}
	if PermissionHint("linux", denied) != "" || PermissionHint("darwin", nil) != "" {
		t.Error("the hint is macOS-only and needs an error")
	}
}
