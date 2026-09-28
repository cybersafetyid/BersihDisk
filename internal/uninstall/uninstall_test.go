package uninstall

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"bersihdisk/internal/safety"
)

// fakeRunner answers Look/Run from tables and records every command.
type fakeRunner struct {
	bins    map[string]string // tool name -> absolute path
	outputs map[string]string // "name arg arg" -> stdout
	failOn  string            // a command line that returns an error
	calls   []string
}

func (f *fakeRunner) Look(name string) (string, error) {
	if p, ok := f.bins[name]; ok {
		return p, nil
	}
	return "", ErrUnavailable
}

func (f *fakeRunner) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	line := strings.Join(append([]string{filepath.Base(name)}, args...), " ")
	f.calls = append(f.calls, line)
	if line == f.failOn {
		return nil, errors.New("boom")
	}
	return []byte(f.outputs[line]), nil
}

// sandbox points HOME at a temp folder so the safety guard and every path spec
// resolve inside it.
func sandbox(t *testing.T) (home string, host Host, run *fakeRunner) {
	t.Helper()
	home = t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	run = &fakeRunner{bins: map[string]string{}, outputs: map[string]string{}}
	return home, Host{Env: Env{OS: "darwin", Home: home, Vars: map[string]string{}}, Run: run}, run
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// staticProvider serves fixed packages.
type staticProvider struct {
	id   string
	pkgs []Installed
}

func (s staticProvider) ID() string { return s.id }
func (s staticProvider) List(context.Context, Host) ([]Installed, error) {
	return s.pkgs, nil
}

func TestParsers(t *testing.T) {
	npm, _ := parseNpm([][]byte{[]byte(`{"dependencies":{"typescript":{"version":"5.4.5"},"npm":{"version":"10.2.0"}}}`)})
	if len(npm) != 2 {
		t.Errorf("npm = %v", npm)
	}
	pip, _ := parsePip([][]byte{[]byte(`[{"name":"requests","version":"2.31.0"}]`)})
	if len(pip) != 1 || pip[0].Name != "requests" || pip[0].Version != "2.31.0" {
		t.Errorf("pip = %v", pip)
	}
	pipx, _ := parsePipx([][]byte{[]byte(`{"venvs":{"black":{"metadata":{"main_package":{"package_version":"24.1.0"}}}}}`)})
	if len(pipx) != 1 || pipx[0].Version != "24.1.0" {
		t.Errorf("pipx = %v", pipx)
	}
	cargo, _ := parseCargo([][]byte{[]byte("ripgrep v14.1.0:\n    rg\ncargo-edit v0.12.2 (/home/u/src):\n    cargo-add\n")})
	if len(cargo) != 2 || cargo[0].Name != "ripgrep" || cargo[1].Version != "0.12.2" {
		t.Errorf("cargo = %v", cargo)
	}
	brew, _ := parseBrewLeaves([][]byte{[]byte("wget\nnode\n"), []byte("openssl@3 3.3.0\nwget 1.24.5\nnode 21.7.1 20.1.0\n")})
	if len(brew) != 2 {
		t.Errorf("brew leaves must skip dependencies: %v", brew)
	}
	casks, _ := parseBrewCasks([][]byte{[]byte("docker 4.30.0\nvisual-studio-code 1.90.0\n")})
	if len(casks) != 2 {
		t.Errorf("casks = %v", casks)
	}
	gems, _ := parseGem([][]byte{[]byte("bigdecimal (default: 3.1.3)\nrake (13.1.0, 12.3.3)\nbundler (default: 2.4.10, 2.5.0)\n")})
	if len(gems) != 2 || gems[0].Name != "rake" || gems[1].Name != "bundler" || gems[1].Version != "2.5.0" {
		t.Errorf("default gems must be skipped, others kept: %v", gems)
	}
	scoop, _ := parseScoop([][]byte{[]byte(`{"apps":[{"Name":"git","Version":"2.45.0"}]}`)})
	if len(scoop) != 1 {
		t.Errorf("scoop = %v", scoop)
	}
	fp, _ := parseFlatpak([][]byte{[]byte("org.gimp.GIMP\tGIMP\t2.10\nnot-an-id\tx\ty\n")})
	if len(fp) != 1 || fp[0].Name != "org.gimp.GIMP" {
		t.Errorf("flatpak = %v", fp)
	}
}

func TestCLIProviderListsAndSkipsProtected(t *testing.T) {
	_, host, run := sandbox(t)
	run.bins["npm"] = "/bin/npm"
	run.outputs["npm ls -g --depth=0 --json"] = `{"dependencies":{"typescript":{"version":"5.4.5"},"npm":{"version":"10"},"corepack":{"version":"0.2"}}}`

	pkgs, err := cliProvider{cliSpecs()[0]}.List(context.Background(), host)
	if err != nil {
		t.Fatal(err)
	}
	if len(pkgs) != 1 || pkgs[0].Name != "typescript" {
		t.Fatalf("npm and corepack must never be offered: %v", pkgs)
	}
	cmd := pkgs[0].recipe.Commands[0]
	if strings.Join(cmd.Argv[1:], " ") != "uninstall -g typescript" {
		t.Errorf("argv = %v", cmd.Argv)
	}

	// A missing tool is silent, not an error.
	delete(run.bins, "npm")
	if _, err := (cliProvider{cliSpecs()[0]}).List(context.Background(), host); !errors.Is(err, ErrUnavailable) {
		t.Errorf("err = %v, want ErrUnavailable", err)
	}
}

func TestExpand(t *testing.T) {
	e := Env{OS: "darwin", Home: "/h", Vars: map[string]string{"APPDATA": `C:\Users\u\AppData\Roaming`}}
	cases := map[string][]string{
		"~/x/y":       {filepath.Join("/h", "x", "y")},
		"mac:~/a":     {filepath.Join("/h", "a")},
		"win:~/a":     nil,
		"linux:~/a":   nil,
		"unix:~/a":    {filepath.Join("/h", "a")},
		"{APPDATA}/z": {filepath.Join(`C:\Users\u\AppData\Roaming`, "z")},
		"{NOPE}/z":    nil,
	}
	for spec, want := range cases {
		got := e.Expand(spec)
		if len(got) != len(want) || (len(got) == 1 && got[0] != want[0]) {
			t.Errorf("Expand(%q) = %v, want %v", spec, got, want)
		}
	}
}

func TestExpandGlob(t *testing.T) {
	home, host, _ := sandbox(t)
	write(t, filepath.Join(home, "Library", "Caches", "Google", "AndroidStudio2024.1", "x"), "1")
	write(t, filepath.Join(home, "Library", "Caches", "Google", "Chrome", "x"), "1")
	got := host.Expand("~/Library/Caches/Google/AndroidStudio*")
	if len(got) != 1 || !strings.HasSuffix(got[0], "AndroidStudio2024.1") {
		t.Errorf("glob = %v", got)
	}
}

func TestCleanProfileRemovesOnlyInstallerLines(t *testing.T) {
	rc := strings.Join([]string{
		`export EDITOR=vim`,
		`. "$HOME/.cargo/env"`,
		`export PATH="$HOME/.cargo/bin:$PATH"`,
		`export PATH="$HOME/bin:$PATH"`,
		`export NVM_DIR="$HOME/.nvm"`,
		`[ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"  # This loads nvm`,
		`# >>> conda initialize >>>`,
		`__conda_setup="$('/x/bin/conda' 'shell.zsh' 'hook')"`,
		`# <<< conda initialize <<<`,
		`alias ll='ls -l'`,
		``,
	}, "\n")
	edit := ProfileEdit{
		Lines:  []*regexp.Regexp{sources(".cargo/env"), pathMentions(".cargo/bin"), exportsVar("NVM_DIR"), sources("NVM_DIR/nvm.sh")},
		Blocks: []Block{{"# >>> conda initialize >>>", "# <<< conda initialize <<<"}},
	}
	out, removed := CleanProfile(rc, edit)
	if len(removed) != 7 {
		t.Errorf("removed %d lines, want 7: %q", len(removed), removed)
	}
	for _, keep := range []string{"EDITOR=vim", `$HOME/bin:$PATH`, "alias ll"} {
		if !strings.Contains(out, keep) {
			t.Errorf("unrelated line %q was removed:\n%s", keep, out)
		}
	}
	for _, gone := range []string{"cargo", "NVM_DIR", "conda"} {
		if strings.Contains(out, gone) {
			t.Errorf("%q survived:\n%s", gone, out)
		}
	}
}

func TestCleanProfileLeavesUnclosedBlockAlone(t *testing.T) {
	rc := "a\n# >>> conda initialize >>>\nb\nc\n"
	out, removed := CleanProfile(rc, ProfileEdit{Blocks: []Block{{"# >>> conda initialize >>>", "# <<< conda initialize <<<"}}})
	if out != rc || len(removed) != 0 {
		t.Errorf("an unclosed block must not delete the rest of the file: %q", out)
	}
}

func TestDropPathEntries(t *testing.T) {
	got, gone := dropPathEntries(`C:\a;C:\Users\u\.cargo\bin\;C:\b`, []string{`c:/users/u/.cargo/bin`}, ";")
	if got != `C:\a;C:\b` || len(gone) != 1 {
		t.Errorf("got %q gone %v", got, gone)
	}
}

func TestParseDesktop(t *testing.T) {
	d, ok := parseDesktop("[Desktop Entry]\nName=Foo App\nExec=\"/home/u/Apps/Foo.AppImage\" %U\nIcon=/home/u/icons/foo.png\n[Desktop Action x]\nName=Other\n")
	if !ok || d.Name != "Foo App" || d.Icon != "/home/u/icons/foo.png" {
		t.Fatalf("desktop = %+v", d)
	}
	if execTarget(d.Exec) != "/home/u/Apps/Foo.AppImage" {
		t.Errorf("exec target = %q", execTarget(d.Exec))
	}
}

func TestMacResidueSeparatesExactFromGuessed(t *testing.T) {
	res := macResidue("Slack", "com.tinyspeck.slackmacgap")
	var exactN, guessN, adminN int
	for _, r := range res {
		switch {
		case r.NeedsAdmin:
			adminN++
		case r.Exact:
			exactN++
		case r.Reason == "nameMatch":
			guessN++
		}
	}
	if exactN == 0 || guessN == 0 || adminN == 0 {
		t.Errorf("exact=%d guessed=%d admin=%d, want all three kinds", exactN, guessN, adminN)
	}
	// A short or wildcard identifier must not become a wide glob.
	for _, id := range []string{"", "a.b", "com.*", "../x.y.z"} {
		if validBundleID(id) {
			t.Errorf("%q must not be a valid bundle id", id)
		}
	}
	if len(macResidue("ab", "")) != 0 {
		t.Error("a two-letter name is too generic to match")
	}
}

func TestMacPackageSkipsAppleAppsAndReadsBundleID(t *testing.T) {
	home, host, run := sandbox(t)
	plutil := "/usr/bin/plutil"
	run.bins["plutil"] = plutil
	bundle := filepath.Join(home, "Applications", "Slack.app")
	write(t, filepath.Join(bundle, "Contents", "Info.plist"), "x")
	run.outputs["plutil -convert json -o - "+filepath.Join(bundle, "Contents", "Info.plist")] =
		`{"CFBundleIdentifier":"com.tinyspeck.slackmacgap","CFBundleShortVersionString":"4.38"}`

	p, ok := macPackage(context.Background(), host, bundle)
	if !ok || p.Version != "4.38" || p.recipe.Paths[0].Spec != bundle {
		t.Fatalf("package = %+v ok=%v", p, ok)
	}

	safari := filepath.Join(home, "Applications", "Safari.app")
	write(t, filepath.Join(safari, "Contents", "Info.plist"), "x")
	run.outputs["plutil -convert json -o - "+filepath.Join(safari, "Contents", "Info.plist")] = `{"CFBundleIdentifier":"com.apple.Safari"}`
	if _, ok := macPackage(context.Background(), host, safari); ok {
		t.Error("Apple's own apps must not be offered")
	}
}

// ---- plan + execute, end to end in a sandbox ----

func TestPlanAndExecuteRemovesEverythingSelected(t *testing.T) {
	home, host, run := sandbox(t)
	run.bins["rustup"] = "/x/rustup"

	write(t, filepath.Join(home, ".cargo", "bin", "cargo"), strings.Repeat("x", 4096))
	write(t, filepath.Join(home, ".rustup", "toolchains", "t", "f"), "x")
	write(t, filepath.Join(home, ".zshrc"), "export A=1\n. \"$HOME/.cargo/env\"\n")
	write(t, filepath.Join(home, ".npmrc"), "//registry:_authToken=secret\n")

	pkg := Installed{ID: "t:rust", Provider: "t", Name: "Rust", recipe: Recipe{
		Commands: []Command{{Argv: []string{"/x/rustup", "self", "uninstall", "-y"}, Label: "rustup self uninstall -y"}},
		Paths: []Residue{
			exact("~/.cargo"), exact("~/.rustup"), config("~/.npmrc"),
			{Spec: "~/.cargo/bin", Exact: true}, // nested: goes with ~/.cargo
			exact("~/.nothing-here"),            // absent: not offered
		},
		Profiles: []ProfileEdit{{Lines: []*regexp.Regexp{sources(".cargo/env")}}},
	}}
	inv := New(host, staticProvider{"t", []Installed{pkg}})
	inv.backupDir = filepath.Join(home, "backups")
	inv.List(context.Background())

	plan, err := inv.Plan(context.Background(), "t:rust")
	if err != nil {
		t.Fatal(err)
	}
	byLabel := map[string]Step{}
	for _, s := range plan.Steps {
		byLabel[s.Label] = s
	}
	if _, ok := byLabel[filepath.Join(home, ".cargo", "bin")]; ok {
		t.Error("a path inside another step must not be a step of its own")
	}
	if _, ok := byLabel[filepath.Join(home, ".nothing-here")]; ok {
		t.Error("a missing path must not be offered")
	}
	npmrc := byLabel[filepath.Join(home, ".npmrc")]
	if npmrc.Selected || npmrc.Level != safety.Caution || npmrc.Reasons[0] != "config" {
		t.Errorf("user config must start unchecked as a caution: %+v", npmrc)
	}
	if !byLabel[filepath.Join(home, ".cargo")].Selected || byLabel[filepath.Join(home, ".cargo")].Size < 4096 {
		t.Errorf(".cargo step = %+v", byLabel[filepath.Join(home, ".cargo")])
	}
	if plan.Steps[0].Kind != StepCommand {
		t.Error("the native command must run first")
	}

	var ids []string
	for _, s := range plan.Steps {
		if s.Selected {
			ids = append(ids, s.ID)
		}
	}
	var events int
	res, err := inv.Execute(context.Background(), Request{PlanID: plan.ID, Steps: ids, Mode: "permanent"}, func(Progress) { events++ })
	if err != nil {
		t.Fatal(err)
	}
	if res.Failed != 0 || res.Aborted {
		t.Fatalf("result = %+v", res)
	}
	if len(run.calls) != 1 || run.calls[0] != "rustup self uninstall -y" {
		t.Errorf("commands run = %v", run.calls)
	}
	for _, gone := range []string{".cargo", ".rustup"} {
		if _, err := os.Stat(filepath.Join(home, gone)); !os.IsNotExist(err) {
			t.Errorf("%s should be gone", gone)
		}
	}
	if _, err := os.Stat(filepath.Join(home, ".npmrc")); err != nil {
		t.Error("the unchecked config file must survive")
	}
	rc, _ := os.ReadFile(filepath.Join(home, ".zshrc"))
	if strings.Contains(string(rc), "cargo") || !strings.Contains(string(rc), "export A=1") {
		t.Errorf("profile = %q", rc)
	}
	if baks, _ := filepath.Glob(filepath.Join(home, ".zshrc.bersihdisk-*.bak")); len(baks) != 1 {
		t.Errorf("the profile must be backed up before it is edited: %v", baks)
	}
	if events != len(ids) || res.Bytes < 4096 {
		t.Errorf("events=%d (want %d) bytes=%d", events, len(ids), res.Bytes)
	}
}

func TestFailedCommandStopsTheRun(t *testing.T) {
	home, host, run := sandbox(t)
	write(t, filepath.Join(home, ".tool", "f"), "x")
	run.failOn = "tool uninstall"
	pkg := Installed{ID: "t:tool", Provider: "t", Name: "Tool", recipe: Recipe{
		Commands: []Command{{Argv: []string{"/x/tool", "uninstall"}, Label: "tool uninstall"}},
		Paths:    []Residue{exact("~/.tool")},
	}}
	inv := New(host, staticProvider{"t", []Installed{pkg}})
	inv.List(context.Background())
	plan, _ := inv.Plan(context.Background(), "t:tool")

	res, err := inv.Execute(context.Background(), Request{PlanID: plan.ID, Steps: []string{"s0", "s1"}, Mode: "permanent"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Aborted || res.Failed != 1 || res.Skipped != 1 {
		t.Errorf("result = %+v", res)
	}
	if _, err := os.Stat(filepath.Join(home, ".tool")); err != nil {
		t.Error("leftovers of a tool that refused to uninstall must stay")
	}
}

func TestExecuteRejectsForeignStepsAndMissingAck(t *testing.T) {
	home, host, _ := sandbox(t)
	write(t, filepath.Join(home, ".cfg"), "x")
	pkg := Installed{ID: "t:x", Provider: "t", Name: "X", recipe: Recipe{Paths: []Residue{config("~/.cfg")}}}
	inv := New(host, staticProvider{"t", []Installed{pkg}})
	inv.List(context.Background())
	plan, _ := inv.Plan(context.Background(), "t:x")

	if _, err := inv.Execute(context.Background(), Request{PlanID: plan.ID, Steps: []string{"s99"}}, nil); err == nil {
		t.Error("a step that is not in the plan must be rejected")
	}
	if _, err := inv.Execute(context.Background(), Request{PlanID: plan.ID, Steps: []string{"s0"}}, nil); err == nil {
		t.Error("a caution step needs the user's acknowledgement")
	}
	if _, err := inv.Execute(context.Background(), Request{PlanID: "nope", Steps: nil}, nil); err == nil {
		t.Error("an unknown plan must be rejected")
	}
	if _, err := os.Stat(filepath.Join(home, ".cfg")); err != nil {
		t.Error("nothing may be deleted by a rejected request")
	}
	if _, err := inv.Execute(context.Background(), Request{PlanID: plan.ID, Steps: []string{"s0"}, Acknowledged: true, Mode: "permanent"}, nil); err != nil {
		t.Errorf("acknowledged run: %v", err)
	}
}

func TestProtectedPathBecomesABlockedStep(t *testing.T) {
	home, host, _ := sandbox(t)
	write(t, filepath.Join(home, ".ssh", "id_rsa"), "key")
	pkg := Installed{ID: "t:x", Provider: "t", Name: "X", recipe: Recipe{Paths: []Residue{exact("~/.ssh")}}}
	inv := New(host, staticProvider{"t", []Installed{pkg}})
	inv.List(context.Background())
	plan, _ := inv.Plan(context.Background(), "t:x")

	if len(plan.Steps) != 1 || plan.Steps[0].Level != safety.Blocked {
		t.Fatalf("steps = %+v", plan.Steps)
	}
	res, _ := inv.Execute(context.Background(), Request{PlanID: plan.ID, Steps: []string{"s0"}, Mode: "permanent", Acknowledged: true}, nil)
	if res.Failed != 1 {
		t.Errorf("result = %+v", res)
	}
	if _, err := os.Stat(filepath.Join(home, ".ssh", "id_rsa")); err != nil {
		t.Fatal("the credentials folder must survive")
	}
}

func TestAdminLocationIsShownNotRun(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("posix paths")
	}
	home, host, _ := sandbox(t)
	dir := filepath.Join(home, "opt it's")
	write(t, filepath.Join(dir, "f"), "x")
	pkg := Installed{ID: "t:x", Provider: "t", Name: "X", recipe: Recipe{Paths: []Residue{{Spec: dir, Exact: true, NeedsAdmin: true}}}}
	inv := New(host, staticProvider{"t", []Installed{pkg}})
	inv.List(context.Background())
	plan, _ := inv.Plan(context.Background(), "t:x")

	if plan.Steps[0].Manual != "sudo rm -rf '"+dir[:len(home)]+`/opt it'\''s'` {
		t.Errorf("manual command = %s", plan.Steps[0].Manual)
	}
	res, _ := inv.Execute(context.Background(), Request{PlanID: plan.ID, Steps: []string{"s0"}, Mode: "permanent"}, nil)
	if len(res.Manual) != 1 || res.Failed != 0 {
		t.Errorf("result = %+v", res)
	}
	if _, err := os.Stat(dir); err != nil {
		t.Fatal("an administrator-only location must not be touched by the app")
	}
}

func TestCatalogIsSound(t *testing.T) {
	envs := []Env{
		{OS: "darwin", Home: "/home/u", Vars: map[string]string{}},
		{OS: "linux", Home: "/home/u", Vars: map[string]string{}},
		{OS: "windows", Home: `C:\Users\u`, Vars: map[string]string{
			"APPDATA": `C:\Users\u\AppData\Roaming`, "LOCALAPPDATA": `C:\Users\u\AppData\Local`,
		}},
	}
	seen := map[string]bool{}
	for _, tl := range tools() {
		if tl.id == "" || tl.name == "" || seen[tl.id] || len(tl.detect) == 0 {
			t.Errorf("tool %q is malformed or duplicated", tl.id)
		}
		seen[tl.id] = true
		for _, r := range tl.paths {
			if r.Spec == "" || r.Spec == "~" || r.Spec == "~/" {
				t.Errorf("%s: a residue must never be the home folder itself", tl.id)
			}
			if strings.Contains(strings.TrimPrefix(r.Spec, "~/"), "..") {
				t.Errorf("%s: %q climbs out of its folder", tl.id, r.Spec)
			}
			if r.Spec == "~/go" || r.Spec == "~/.gradle" || r.Spec == "~/.m2" {
				t.Errorf("%s: %q is too broad; list its cache folders instead (it holds source or settings)", tl.id, r.Spec)
			}
		}
		slash := func(p string) string { return strings.ReplaceAll(p, `\`, "/") }
		for _, e := range envs {
			for _, r := range tl.paths {
				for _, p := range e.Expand(r.Spec) {
					if !r.NeedsAdmin && !strings.HasPrefix(slash(p), slash(e.Home)+"/") {
						t.Errorf("%s (%s): %q lies outside the home folder", tl.id, e.OS, p)
					}
				}
			}
		}
	}
}

// ---- cases taken from vendor documentation and real tool output ----

func TestParseApt(t *testing.T) {
	manual := "bash\ncurl\nlinux-generic\nubuntu-desktop\nhtop\nsudo\nnginx\n"
	dpkg := "bash\t5.2\trequired\tno\n" +
		"curl\t8.5\toptional\t\n" +
		"linux-generic\t6.8\toptional\t\n" +
		"ubuntu-desktop\t1.5\toptional\t\n" +
		"htop\t3.3\toptional\t\n" +
		"sudo\t1.9\toptional\t\n" +
		"nginx\t1.24\toptional\t\n" +
		"libfoo\t1\toptional\t\n" // installed as a dependency, not in the manual list
	got, _ := parseApt([][]byte{[]byte(manual), []byte(dpkg)})
	names := map[string]bool{}
	for _, e := range got {
		names[e.Name] = true
	}
	for _, want := range []string{"curl", "htop", "nginx"} {
		if !names[want] {
			t.Errorf("%s should be offered: %v", want, got)
		}
	}
	for _, bad := range []string{"bash", "linux-generic", "ubuntu-desktop", "sudo", "libfoo"} {
		if names[bad] {
			t.Errorf("%s must not be offered", bad)
		}
	}
}

func TestParseSnapSkipsRuntimes(t *testing.T) {
	out := "Name        Version   Rev   Tracking       Publisher  Notes\n" +
		"bare        1.0       5     latest/stable  canonical✓ base\n" +
		"core22      20250923  2139  latest/stable  canonical✓ base\n" +
		"firefox     144.0     7177  latest/stable  mozilla✓   -\n" +
		"snapd       2.72      25577 latest/stable  canonical✓ snapd\n" +
		"slack       4.46      216   latest/stable  slack✓     -\n"
	got, _ := parseSnap([][]byte{[]byte(out)})
	if len(got) != 2 || got[0].Name != "firefox" || got[1].Name != "slack" {
		t.Errorf("snaps = %v", got)
	}
}

func TestParseNameVersionAndAppx(t *testing.T) {
	got, _ := parseNameVersion([][]byte{[]byte("git 2.45.0-1\nneovim 0.10\n")})
	if len(got) != 2 || got[1].Name != "neovim" || got[1].Version != "0.10" {
		t.Errorf("pacman/dnf = %v", got)
	}

	one, err := parseAppx([]byte(`{"Name":"Python.3.12","PackageFullName":"PythonSoftwareFoundation.Python.3.12_3.12.1520.0_x64__qbz5n2kfra8p0","Version":"3.12","InstallLocation":"C:\\x"}`))
	if err != nil || len(one) != 1 {
		t.Fatalf("a single package is serialised as an object: %v %v", one, err)
	}
	many, _ := parseAppx([]byte(`[{"Name":"Microsoft.WindowsTerminal","PackageFullName":"a"},{"Name":"Microsoft.Windows.Photos","PackageFullName":"b"},{"Name":"Spotify","PackageFullName":"c"}]`))
	if len(many) != 2 {
		t.Errorf("inbox Windows components must be dropped: %v", many)
	}
	if got, _ := parseAppx([]byte("  ")); got != nil {
		t.Errorf("empty output = %v", got)
	}
}

func TestSystemPackageManagersAreShownNotRun(t *testing.T) {
	_, host, run := sandbox(t)
	host.OS = "linux"
	run.bins["apt"] = "/usr/bin/apt"
	run.bins["apt-mark"] = "/usr/bin/apt-mark"
	run.bins["dpkg-query"] = "/usr/bin/dpkg-query"
	run.outputs["apt-mark showmanual"] = "htop\n"
	run.outputs[`dpkg-query -W -f=${Package}\t${Version}\t${Priority}\t${Essential}\n`] = "htop\t3.3\toptional\t\n"

	var spec cliSpec
	for _, s := range cliSpecs() {
		if s.id == "apt" {
			spec = s
		}
	}
	pkgs, err := cliProvider{spec}.List(context.Background(), host)
	if err != nil || len(pkgs) != 1 {
		t.Fatalf("apt listing = %v, %v", pkgs, err)
	}
	cmds := pkgs[0].recipe.Commands
	if !cmds[0].NeedsAdmin || cmds[0].Label != "apt purge htop" || !cmds[1].Optional || cmds[1].Label != "apt autoremove --purge" {
		t.Errorf("commands = %+v", cmds)
	}
	inv := New(host, staticProvider{"apt", pkgs})
	inv.List(context.Background())
	plan, _ := inv.Plan(context.Background(), "apt:htop")
	if plan.Steps[0].Manual != "sudo apt purge htop" {
		t.Errorf("manual = %q", plan.Steps[0].Manual)
	}
	res, _ := inv.Execute(context.Background(), Request{PlanID: plan.ID, Steps: []string{"s0", "s1"}}, nil)
	if len(run.calls) != 2 || res.Failed != 0 { // the two listing calls only; nothing was run as root
		t.Errorf("calls = %v result = %+v", run.calls, res)
	}
	if len(res.Manual) != 2 {
		t.Errorf("manual commands = %v", res.Manual)
	}
}

func TestExpandUsesXDGDefaultsAndOverrides(t *testing.T) {
	e := Env{OS: "linux", Home: "/home/u", Vars: map[string]string{}}
	if got := e.Expand("{XDG_CONFIG_HOME}/Code"); len(got) != 1 || got[0] != filepath.Join("/home/u", ".config", "Code") {
		t.Errorf("default = %v", got)
	}
	e.Vars["XDG_CONFIG_HOME"] = "/data/cfg"
	if got := e.Expand("{XDG_CONFIG_HOME}/Code"); len(got) != 1 || got[0] != filepath.Join("/data/cfg", "Code") {
		t.Errorf("override = %v", got)
	}
}

func TestNvmGuardedSourceLinesAreRemoved(t *testing.T) {
	rc := "export NVM_DIR=\"$HOME/.nvm\"\n" +
		"[ -s \"$NVM_DIR/nvm.sh\" ] && \\. \"$NVM_DIR/nvm.sh\"  # This loads nvm\n" +
		"[[ -r $NVM_DIR/bash_completion ]] && \\. $NVM_DIR/bash_completion\n" +
		"alias k=kubectl\n"
	var nvm tool
	for _, tl := range tools() {
		if tl.id == "nvm" {
			nvm = tl
		}
	}
	out, removed := CleanProfile(rc, ProfileEdit{Lines: nvm.profile})
	if len(removed) != 3 || out != "alias k=kubectl\n" {
		t.Errorf("removed %d, left %q", len(removed), out)
	}
}

func TestKnownAppsGetTheirDocumentedLeftovers(t *testing.T) {
	res, _ := extrasFor("Microsoft Visual Studio Code (User)")
	has := func(rs []Residue, spec string) bool {
		for _, r := range rs {
			if r.Spec == spec {
				return true
			}
		}
		return false
	}
	if !has(res, "~/.vscode-shared") || !has(res, "win:{APPDATA}/Code") || has(res, "~/.vscode-insiders") {
		t.Errorf("stable VS Code extras = %v", res)
	}
	ins, _ := extrasFor("Microsoft Visual Studio Code - Insiders")
	if !has(ins, "~/.vscode-insiders") {
		t.Errorf("insiders extras = %v", ins)
	}
	docker, warn := extrasFor("Docker Desktop")
	if !has(docker, "win:{PROGRAMDATA}/DockerDesktop") || len(warn) == 0 {
		t.Errorf("docker extras = %v %v", docker, warn)
	}
	if got, _ := extrasFor("Some Random App"); len(got) != 0 {
		t.Error("an unknown app gets no extras")
	}
}

// ---- application icons ----

func TestIconLookups(t *testing.T) {
	home, host, _ := sandbox(t)

	bundle := filepath.Join(home, "Applications", "Foo.app")
	write(t, filepath.Join(bundle, "Contents", "Resources", "foo.icns"), "x")
	write(t, filepath.Join(bundle, "Contents", "Resources", "AppIcon.icns"), "x")
	if got := macIconFile(bundle, "foo"); filepath.Base(got) != "foo.icns" {
		t.Errorf("CFBundleIconFile without extension: %q", got)
	}
	if got := macIconFile(bundle, "missing.icns"); filepath.Base(got) != "AppIcon.icns" {
		t.Errorf("fallback to AppIcon.icns: %q", got)
	}
	if macIconFile(bundle, "../../etc/passwd") != "" && !strings.HasPrefix(macIconFile(bundle, "../../etc/passwd"), bundle) {
		t.Error("an icon name must never escape the bundle")
	}
	if macIconFile(filepath.Join(home, "Nope.app"), "x") != "" {
		t.Error("no icon file, no result")
	}

	host.OS = "linux"
	write(t, filepath.Join(home, ".local", "share", "icons", "hicolor", "48x48", "apps", "foo.png"), "x")
	write(t, filepath.Join(home, ".local", "share", "icons", "hicolor", "64x64", "apps", "foo.png"), "x")
	if got := linuxIconFile(host.Env, "foo"); !strings.Contains(got, "64x64") {
		t.Errorf("prefers a size near 64px: %q", got)
	}
	if got := linuxIconFile(host.Env, "/no/such.png"); got != "" {
		t.Errorf("missing absolute path: %q", got)
	}

	if winIconExe(`"C:\Program Files\App\app.exe",0`) != `C:\Program Files\App\app.exe` || winIconExe(`C:\x\readme.txt`) != "" {
		t.Error("DisplayIcon parsing")
	}
	if s := exeScript(`C:\it's\a.exe`); !strings.Contains(s, `it''s`) {
		t.Errorf("a quote in the path must be doubled, not break out: %s", s)
	}
}

func TestIconIsServedFromCacheAndOnlyForListedPackages(t *testing.T) {
	home, host, _ := sandbox(t)
	png := filepath.Join(home, "app.png")
	write(t, png, "\x89PNG")
	pkg := Installed{ID: "t:app", Provider: "t", Name: "App", iconSrc: iconSource{kind: "file", ref: png}}
	inv := New(host, staticProvider{"t", []Installed{pkg, {ID: "t:none", Provider: "t", Name: "None"}}})
	inv.List(context.Background())

	url, err := inv.Icon(context.Background(), "t:app")
	if err != nil || !strings.HasPrefix(url, "data:image/png;base64,") {
		t.Fatalf("icon = %q, %v", url, err)
	}
	os.Remove(png) // the second answer must come from the cache
	if again, err := inv.Icon(context.Background(), "t:app"); err != nil || again != url {
		t.Errorf("cached icon: %v", err)
	}
	if _, err := inv.Icon(context.Background(), "t:none"); err == nil {
		t.Error("a package without an icon source has no icon")
	}
	if _, err := inv.Icon(context.Background(), "../../etc/passwd"); err == nil {
		t.Error("an ID that was never listed must be refused")
	}
}
