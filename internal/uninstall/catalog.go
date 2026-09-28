package uninstall

import (
	"context"
	"os"
	"regexp"
)

// tool describes a developer toolchain that is not a package of some manager: a
// version manager, SDK or CLI installed by a script. Adding one is adding one
// entry — detection, native uninstall, leftovers, shell lines and PATH entries
// are all data.
type tool struct {
	id, name, icon string
	kind           Kind
	detect         []string // path specs; the tool is listed when any exists
	native         []nativeCmd
	paths          []Residue
	profile        []*regexp.Regexp
	blocks         []Block
	winPath        []string // path specs dropped from the Windows user PATH
	warn           []string
}

// nativeCmd is the tool's own uninstall command, used when its binary exists.
type nativeCmd struct {
	bin   string // a tool name, or a path spec
	args  []string
	label string
}

func exact(spec string) Residue    { return Residue{Spec: spec, Exact: true} }
func config(spec string) Residue   { return Residue{Spec: spec, Exact: true, Reason: "config"} }
func userData(spec string) Residue { return Residue{Spec: spec, Exact: true, Reason: "userData"} }
func admin(spec string) Residue    { return Residue{Spec: spec, Exact: true, NeedsAdmin: true} }
func residues(specs ...string) []Residue {
	out := make([]Residue, len(specs))
	for i, s := range specs {
		out[i] = exact(s)
	}
	return out
}

func join(groups ...[]Residue) []Residue {
	var out []Residue
	for _, g := range groups {
		out = append(out, g...)
	}
	return out
}

// tools is the catalog. Paths were taken from each tool's own uninstall
// documentation; anything that may hold the developer's own work (config, source
// folders, local artifacts) carries a Reason so it starts unchecked.
func tools() []tool {
	return []tool{
		{
			id: "rustup", name: "Rust (rustup + cargo)", icon: "rust", kind: KindRuntime,
			detect:  []string{"~/.rustup", "~/.cargo/bin/rustup"},
			native:  []nativeCmd{{bin: "~/.cargo/bin/rustup", args: []string{"self", "uninstall", "-y"}, label: "rustup self uninstall -y"}},
			paths:   residues("~/.cargo", "~/.rustup"),
			profile: []*regexp.Regexp{sources(".cargo/env"), pathMentions(".cargo/bin")},
			winPath: []string{"~/.cargo/bin"},
			warn:    []string{"removesInstalledBinaries"},
		},
		{
			id: "nvm", name: "Node.js (nvm)", icon: "nodedotjs", kind: KindRuntime,
			detect: []string{"~/.nvm"},
			paths: join(residues("~/.nvm", "~/.node_repl_history"),
				[]Residue{{Spec: "~/.npm", Reason: "nameMatch"}, config("~/.npmrc")}),
			profile: []*regexp.Regexp{exportsVar("NVM_DIR"), sources("NVM_DIR/nvm.sh"), sources("NVM_DIR/bash_completion")},
			warn:    []string{"removesGlobalPackages"},
		},
		{
			id: "fnm", name: "Node.js (fnm)", icon: "nodedotjs", kind: KindRuntime,
			detect:  []string{"{XDG_DATA_HOME}/fnm", "~/.fnm", "mac:~/Library/Application Support/fnm", "win:{LOCALAPPDATA}/fnm"},
			paths:   residues("{XDG_DATA_HOME}/fnm", "~/.fnm", "mac:~/Library/Application Support/fnm", "win:{LOCALAPPDATA}/fnm"),
			profile: []*regexp.Regexp{sources("fnm env"), pathMentions("fnm")},
			warn:    []string{"removesGlobalPackages"},
		},
		{
			id: "volta", name: "Node.js (Volta)", icon: "nodedotjs", kind: KindRuntime,
			detect:  []string{"~/.volta", "win:{LOCALAPPDATA}/Volta"},
			paths:   residues("~/.volta", "win:{LOCALAPPDATA}/Volta"),
			profile: []*regexp.Regexp{exportsVar("VOLTA_HOME"), pathMentions("VOLTA_HOME")},
			winPath: []string{"{LOCALAPPDATA}/Volta/bin"},
			warn:    []string{"removesGlobalPackages"},
		},
		{
			id: "pyenv", name: "Python (pyenv)", icon: "python", kind: KindRuntime,
			detect:  []string{"~/.pyenv", "win:~/.pyenv/pyenv-win"},
			paths:   residues("~/.pyenv"),
			profile: []*regexp.Regexp{exportsVar("PYENV_ROOT"), sources("pyenv init"), sources("pyenv virtualenv-init"), pathMentions("PYENV_ROOT")},
			warn:    []string{"removesVirtualenvs"},
		},
		{
			id: "conda", name: "Python (conda / Anaconda / Miniconda)", icon: "python", kind: KindRuntime,
			detect: []string{"~/miniconda3", "~/anaconda3", "~/miniforge3", "~/mambaforge", "~/opt/anaconda3", "~/opt/miniconda3", "~/.conda"},
			native: []nativeCmd{
				{bin: "~/miniconda3/bin/conda", args: []string{"init", "--reverse", "--all"}, label: "conda init --reverse --all"},
				{bin: "~/anaconda3/bin/conda", args: []string{"init", "--reverse", "--all"}, label: "conda init --reverse --all"},
				{bin: "~/miniforge3/bin/conda", args: []string{"init", "--reverse", "--all"}, label: "conda init --reverse --all"},
			},
			paths: join(residues("~/miniconda3", "~/anaconda3", "~/miniforge3", "~/mambaforge",
				"~/opt/anaconda3", "~/opt/miniconda3", "~/.conda", "~/.continuum"), []Residue{config("~/.condarc")}),
			blocks: []Block{
				{"# >>> conda initialize >>>", "# <<< conda initialize <<<"},
				{"# >>> mamba initialize >>>", "# <<< mamba initialize <<<"},
			},
			warn: []string{"removesVirtualenvs"},
		},
		{
			id: "sdkman", name: "SDKMAN! (Java, Kotlin, Gradle…)", icon: "openjdk", kind: KindRuntime,
			detect:  []string{"~/.sdkman"},
			paths:   residues("~/.sdkman"),
			profile: []*regexp.Regexp{exportsVar("SDKMAN_DIR"), sources("sdkman-init.sh")},
			blocks:  []Block{{"#THIS MUST BE AT THE END OF THE FILE FOR SDKMAN TO WORK", "sdkman-init.sh"}},
		},
		{
			id: "bun", name: "Bun", icon: "bun", kind: KindRuntime,
			detect:  []string{"~/.bun"},
			paths:   residues("~/.bun"),
			profile: []*regexp.Regexp{exportsVar("BUN_INSTALL"), pathMentions("BUN_INSTALL"), sources(".bun/_bun")},
			winPath: []string{"~/.bun/bin"},
		},
		{
			id: "deno", name: "Deno", icon: "deno", kind: KindRuntime,
			detect:  []string{"~/.deno", "mac:~/Library/Caches/deno", "linux:{XDG_CACHE_HOME}/deno", "win:{LOCALAPPDATA}/deno"},
			paths:   residues("~/.deno", "mac:~/Library/Caches/deno", "linux:{XDG_CACHE_HOME}/deno", "win:{LOCALAPPDATA}/deno"),
			profile: []*regexp.Regexp{exportsVar("DENO_INSTALL"), pathMentions("DENO_INSTALL")},
			winPath: []string{"~/.deno/bin"},
		},
		{
			id: "pnpm", name: "pnpm", icon: "pnpm", kind: KindRuntime,
			detect: []string{"{XDG_DATA_HOME}/pnpm", "mac:~/Library/pnpm", "win:{LOCALAPPDATA}/pnpm"},
			paths: residues("{XDG_DATA_HOME}/pnpm", "mac:~/Library/pnpm", "mac:~/Library/Caches/pnpm",
				"linux:{XDG_CACHE_HOME}/pnpm", "win:{LOCALAPPDATA}/pnpm", "win:{LOCALAPPDATA}/pnpm-cache"),
			profile: []*regexp.Regexp{exportsVar("PNPM_HOME"), pathMentions("PNPM_HOME")},
			winPath: []string{"{LOCALAPPDATA}/pnpm"},
			warn:    []string{"removesGlobalPackages"},
		},
		{
			id: "poetry", name: "Poetry", icon: "poetry", kind: KindRuntime,
			detect: []string{"{XDG_DATA_HOME}/pypoetry", "mac:~/Library/Application Support/pypoetry", "~/.poetry", "win:{APPDATA}/pypoetry"},
			paths: join(
				residues("{XDG_DATA_HOME}/pypoetry", "mac:~/Library/Application Support/pypoetry", "~/.poetry",
					"mac:~/Library/Caches/pypoetry", "linux:{XDG_CACHE_HOME}/pypoetry", "win:{APPDATA}/pypoetry", "win:{LOCALAPPDATA}/pypoetry"),
				[]Residue{config("{XDG_CONFIG_HOME}/pypoetry"), config("mac:~/Library/Preferences/pypoetry")}),
			profile: []*regexp.Regexp{pathMentions(".poetry/bin")},
		},
		{
			id: "uv", name: "uv (Python)", icon: "astral", kind: KindRuntime,
			detect: []string{"~/.local/bin/uv", "{XDG_DATA_HOME}/uv", "~/.cargo/bin/uv"},
			paths: residues("~/.local/bin/uv", "~/.local/bin/uvx", "{XDG_DATA_HOME}/uv",
				"mac:~/Library/Caches/uv", "linux:{XDG_CACHE_HOME}/uv", "win:{LOCALAPPDATA}/uv"),
		},
		{
			id: "go", name: "Go (SDK + GOPATH)", icon: "go", kind: KindRuntime,
			detect: []string{"/usr/local/go", "~/go", "~/sdk"},
			paths: join(
				[]Residue{admin("unix:/usr/local/go")},
				residues("~/sdk", "~/go/bin", "~/go/pkg", "mac:~/Library/Caches/go-build", "linux:{XDG_CACHE_HOME}/go-build", "win:{LOCALAPPDATA}/go-build"),
				[]Residue{userData("~/go/src"), config("{XDG_CONFIG_HOME}/go")}),
			profile: []*regexp.Regexp{pathMentions("/usr/local/go/bin"), pathMentions("GOPATH"), pathMentions("go env GOPATH"), exportsVar("GOPATH")},
			warn:    []string{"gopathHoldsSource"},
		},
		{
			id: "gradle", name: "Gradle (user home)", icon: "gradle", kind: KindRuntime,
			detect: []string{"~/.gradle"},
			paths:  join(residues("~/.gradle/caches", "~/.gradle/wrapper", "~/.gradle/daemon", "~/.gradle/native"), []Residue{config("~/.gradle/gradle.properties"), config("~/.gradle/init.d")}),
		},
		{
			id: "maven", name: "Maven (user home)", icon: "apachemaven", kind: KindRuntime,
			detect: []string{"~/.m2"},
			paths:  []Residue{{Spec: "~/.m2/repository", Exact: true, Reason: "localArtifacts"}, config("~/.m2/settings.xml"), config("~/.m2/settings-security.xml"), exact("~/.m2/wrapper")},
		},
		{
			id: "flutter", name: "Flutter / Dart (user data)", icon: "flutter", kind: KindRuntime,
			detect:  []string{"~/.pub-cache", "~/.flutter", "~/.dart", "~/.dartServer"},
			paths:   join(residues("~/.pub-cache", "~/.flutter", "~/.dart", "~/.dartServer", "mac:~/Library/Caches/flutter_engine"), []Residue{config("{XDG_CONFIG_HOME}/flutter")}),
			profile: []*regexp.Regexp{pathMentions("flutter/bin"), pathMentions(".pub-cache/bin")},
			warn:    []string{"sdkLocationUnknown"},
		},
		{
			id: "composer", name: "PHP Composer", icon: "composer", kind: KindRuntime,
			detect: []string{"~/.composer", "{XDG_CONFIG_HOME}/composer", "win:{APPDATA}/Composer"},
			paths: join(residues("~/.composer/cache", "~/.composer/vendor", "mac:~/Library/Caches/composer", "linux:{XDG_CACHE_HOME}/composer", "win:{LOCALAPPDATA}/Composer"),
				[]Residue{config("~/.composer/auth.json"), config("{XDG_CONFIG_HOME}/composer")}),
			profile: []*regexp.Regexp{pathMentions(".composer/vendor/bin"), pathMentions(".config/composer/vendor/bin")},
			warn:    []string{"removesGlobalPackages"},
		},
		{
			id: "dotnet", name: ".NET SDK (user data)", icon: "dotnet", kind: KindRuntime,
			detect: []string{"~/.dotnet", "~/.nuget", "unix:/usr/local/share/dotnet"},
			paths: join(residues("~/.dotnet", "~/.nuget", "~/.templateengine"),
				[]Residue{admin("mac:/usr/local/share/dotnet")}),
			profile: []*regexp.Regexp{pathMentions(".dotnet/tools"), exportsVar("DOTNET_ROOT"), pathMentions("DOTNET_ROOT")},
			winPath: []string{"~/.dotnet/tools"},
			warn:    []string{"removesGlobalPackages"},
		},
		{
			id: "rbenv", name: "Ruby (rbenv)", icon: "ruby", kind: KindRuntime,
			detect:  []string{"~/.rbenv"},
			paths:   residues("~/.rbenv"),
			profile: []*regexp.Regexp{sources("rbenv init"), pathMentions(".rbenv/bin"), pathMentions(".rbenv/shims")},
			warn:    []string{"removesGlobalPackages"},
		},
		{
			id: "rvm", name: "Ruby (RVM)", icon: "ruby", kind: KindRuntime,
			detect:  []string{"~/.rvm"},
			native:  []nativeCmd{{bin: "~/.rvm/bin/rvm", args: []string{"implode", "--force"}, label: "rvm implode --force"}},
			paths:   residues("~/.rvm", "~/.rvmrc"),
			profile: []*regexp.Regexp{sources(".rvm/scripts/rvm"), pathMentions(".rvm/bin")},
			warn:    []string{"removesGlobalPackages"},
		},
	}
}

// ---- provider ----

type toolchains struct{}

func (toolchains) ID() string { return "toolchain" }

func (toolchains) List(_ context.Context, h Host) ([]Installed, error) {
	var pkgs []Installed
	for _, t := range tools() {
		if !t.installed(h) {
			continue
		}
		pkgs = append(pkgs, t.pkg(h))
	}
	return pkgs, nil
}

func (t tool) installed(h Host) bool {
	for _, spec := range t.detect {
		for _, p := range h.Expand(spec) {
			if _, err := os.Lstat(p); err == nil {
				return true
			}
		}
	}
	return false
}

func (t tool) pkg(h Host) Installed {
	rec := Recipe{Paths: t.paths, Warnings: t.warn, PathEntries: t.winPath}
	if len(t.profile) > 0 || len(t.blocks) > 0 {
		rec.Profiles = []ProfileEdit{{Lines: t.profile, Blocks: t.blocks}}
	}
	for _, n := range t.native {
		bin := n.bin
		if exp := h.Expand(bin); len(exp) == 1 {
			bin = exp[0]
		}
		if found, err := h.Run.Look(bin); err == nil {
			rec.Commands = append(rec.Commands, Command{Argv: append([]string{found}, n.args...), Label: n.label})
			break // one native command is enough; the variants only differ by install folder
		}
	}
	return Installed{
		ID: "toolchain:" + t.id, Provider: "toolchain", Kind: t.kind, Name: t.name,
		Icon: t.icon, recipe: rec,
	}
}
