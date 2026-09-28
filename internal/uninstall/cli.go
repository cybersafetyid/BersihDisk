package uninstall

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// entry is one package a CLI listing reported.
type entry struct{ Name, Version string }

// cliSpec describes a package manager well enough to list and uninstall its
// packages. Adding a manager is adding one spec — no new code path.
type cliSpec struct {
	id, icon string
	kind     Kind
	oses     []string // empty = every OS
	bins     []string // tool names to look for, first found wins
	list     []call   // commands whose outputs are handed to parse
	parse    func(outs [][]byte) ([]entry, error)
	remove   func(e entry) []string // arguments after the tool name
	after    [][]string             // optional follow-up commands (unticked), e.g. autoremove
	admin    bool                   // removal needs root: shown for copying, never run
	skip     map[string]bool        // packages that must not be offered
	warn     []string               // plan warnings for every package of this manager
	notes    []string               // per-package hints
}

// call is one listing command; bin is empty for the spec's own tool.
type call struct {
	bin  string
	args []string
}

func c(args ...string) call              { return call{args: args} }
func cb(bin string, args ...string) call { return call{bin: bin, args: args} }

func (s cliSpec) applies(os string) bool {
	if len(s.oses) == 0 {
		return true
	}
	for _, o := range s.oses {
		if o == os {
			return true
		}
	}
	return false
}

type cliProvider struct{ spec cliSpec }

func (p cliProvider) ID() string { return p.spec.id }

func (p cliProvider) List(ctx context.Context, h Host) ([]Installed, error) {
	s := p.spec
	if !s.applies(h.OS) {
		return nil, ErrUnavailable
	}
	var bin string
	for _, name := range s.bins {
		if found, err := h.Run.Look(name); err == nil {
			bin = found
			break
		}
	}
	if bin == "" {
		return nil, ErrUnavailable
	}
	outs := make([][]byte, 0, len(s.list))
	for _, cl := range s.list {
		tool := bin
		if cl.bin != "" {
			found, err := h.Run.Look(cl.bin)
			if err != nil {
				return nil, ErrUnavailable
			}
			tool = found
		}
		out, err := h.Run.Run(ctx, tool, cl.args...)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", s.id, err)
		}
		outs = append(outs, out)
	}
	entries, err := s.parse(outs)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", s.id, err)
	}
	sort.Slice(entries, func(i, j int) bool { return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name) })

	pkgs := make([]Installed, 0, len(entries))
	for _, e := range entries {
		if s.skip[e.Name] {
			continue
		}
		argv := append([]string{bin}, s.remove(e)...)
		cmds := []Command{{Argv: argv, Label: shellLine(argv), NeedsAdmin: s.admin}}
		for _, a := range s.after {
			extra := append([]string{bin}, a...)
			cmds = append(cmds, Command{Argv: extra, Label: shellLine(extra), NeedsAdmin: s.admin, Optional: true})
		}
		notes := s.notes
		if s.admin {
			notes = append(append([]string{}, notes...), "needsAdmin")
		}
		pkgs = append(pkgs, Installed{
			ID: s.id + ":" + e.Name, Provider: s.id, Kind: s.kind, Name: e.Name,
			Version: e.Version, Icon: s.icon, Notes: notes,
			recipe: Recipe{Commands: cmds, Warnings: s.warn},
		})
	}
	return pkgs, nil
}

// Advise warns when another Homebrew formula depends on the one being removed;
// brew would refuse anyway, but the plan should say why before it fails.
func (p cliProvider) Advise(ctx context.Context, h Host, pkg Installed) []Warning {
	if p.spec.id != "brew" {
		return nil
	}
	bin, err := h.Run.Look("brew")
	if err != nil {
		return nil
	}
	out, err := h.Run.Run(ctx, bin, "uses", "--installed", pkg.Name)
	if err != nil {
		return nil
	}
	if users := strings.Fields(string(out)); len(users) > 0 {
		return []Warning{{Code: "hasDependents", Detail: strings.Join(users, ", ")}}
	}
	return nil
}

// shellLine renders argv the way a user would type it (display only).
func shellLine(argv []string) string {
	parts := make([]string, len(argv))
	for i, a := range argv {
		if i == 0 {
			a = baseName(a)
		}
		parts[i] = quoteArg(a)
	}
	return strings.Join(parts, " ")
}

// quoteArg wraps an argument in quotes when it holds spaces or quote characters.
func quoteArg(a string) string {
	if strings.ContainsAny(a, " \t\"'") {
		return fmt.Sprintf("%q", a)
	}
	return a
}

func baseName(p string) string {
	p = strings.ReplaceAll(p, "\\", "/")
	return p[strings.LastIndex(p, "/")+1:]
}

// cliSpecs are the package managers supported today.
func cliSpecs() []cliSpec {
	return []cliSpec{
		{
			id: "npm", icon: "nodedotjs", kind: KindPackage,
			bins: []string{"npm"}, list: []call{c("ls", "-g", "--depth=0", "--json")},
			parse:  parseNpm,
			remove: func(e entry) []string { return []string{"uninstall", "-g", e.Name} },
			// Removing npm or corepack would break the tool doing the removing.
			skip: map[string]bool{"npm": true, "corepack": true},
		},
		{
			id: "pip", icon: "pypi", kind: KindPackage,
			bins: []string{"python3", "python"}, list: []call{c("-m", "pip", "list", "--format=json")},
			parse:  parsePip,
			remove: func(e entry) []string { return []string{"-m", "pip", "uninstall", "-y", e.Name} },
			skip:   map[string]bool{"pip": true, "setuptools": true, "wheel": true},
			notes:  []string{"pipEnvironment"},
		},
		{
			id: "pipx", icon: "python", kind: KindPackage,
			bins: []string{"pipx"}, list: []call{c("list", "--json")},
			parse:  parsePipx,
			remove: func(e entry) []string { return []string{"uninstall", e.Name} },
		},
		{
			id: "cargo", icon: "rust", kind: KindPackage,
			bins: []string{"cargo"}, list: []call{c("install", "--list")},
			parse:  parseCargo,
			remove: func(e entry) []string { return []string{"uninstall", e.Name} },
		},
		{
			id: "brew", icon: "homebrew", kind: KindPackage, oses: []string{"darwin", "linux"},
			bins:   []string{"brew"},
			list:   []call{c("leaves"), c("list", "--formula", "--versions")},
			parse:  parseBrewLeaves,
			remove: func(e entry) []string { return []string{"uninstall", "--formula", e.Name} },
			// Dependencies the removed formula pulled in become orphans.
			after: [][]string{{"autoremove"}},
			// brew itself refuses when another formula needs the package.
			notes: []string{"brewLeaf"},
		},
		{
			id: "brewcask", icon: "homebrew", kind: KindApp, oses: []string{"darwin"},
			bins: []string{"brew"}, list: []call{c("list", "--cask", "--versions")},
			parse: parseBrewCasks,
			// --zap also removes the cask's own preferences, caches and support files.
			remove: func(e entry) []string { return []string{"uninstall", "--cask", "--zap", e.Name} },
			warn:   []string{"zapRemovesData"},
		},
		{
			id: "gem", icon: "rubygems", kind: KindPackage,
			bins: []string{"gem"}, list: []call{c("list", "--local")},
			parse:  parseGem,
			remove: func(e entry) []string { return []string{"uninstall", "--all", "--executables", e.Name} },
		},
		{
			id: "scoop", icon: "package", kind: KindPackage, oses: []string{"windows"},
			bins: []string{"scoop"}, list: []call{c("export")},
			parse:  parseScoop,
			remove: func(e entry) []string { return []string{"uninstall", e.Name, "--purge"} },
		},
		{
			id: "flatpak", icon: "package", kind: KindApp, oses: []string{"linux"},
			bins: []string{"flatpak"}, list: []call{c("list", "--app", "--columns=application,name,version")},
			parse: parseFlatpak,
			// --delete-data removes ~/.var/app/<id> as well.
			remove: func(e entry) []string { return []string{"uninstall", "-y", "--delete-data", e.Name} },
			warn:   []string{"zapRemovesData"},
		},
		// System package managers need root, so these are listed with the exact
		// command to run yourself. `purge` (not `remove`) also deletes the package's
		// system-wide configuration; `apt remove` leaves it behind.
		{
			id: "apt", icon: "package", kind: KindPackage, oses: []string{"linux"}, admin: true,
			bins: []string{"apt"},
			list: []call{
				cb("apt-mark", "showmanual"),
				cb("dpkg-query", "-W", `-f=${Package}\t${Version}\t${Priority}\t${Essential}\n`),
			},
			parse:  parseApt,
			remove: func(e entry) []string { return []string{"purge", e.Name} },
			after:  [][]string{{"autoremove", "--purge"}},
		},
		{
			id: "snap", icon: "package", kind: KindApp, oses: []string{"linux"}, admin: true,
			bins: []string{"snap"}, list: []call{c("list")},
			parse: parseSnap,
			// --purge skips the automatic data snapshot snap otherwise keeps.
			remove: func(e entry) []string { return []string{"remove", "--purge", e.Name} },
		},
		{
			id: "pacman", icon: "package", kind: KindPackage, oses: []string{"linux"}, admin: true,
			bins: []string{"pacman"}, list: []call{c("-Qe")},
			parse: parseNameVersion,
			// -ns also drops unneeded dependencies and the package's saved config.
			remove: func(e entry) []string { return []string{"-Rns", e.Name} },
		},
		{
			id: "dnf", icon: "package", kind: KindPackage, oses: []string{"linux"}, admin: true,
			bins:   []string{"dnf"},
			list:   []call{c("repoquery", "--userinstalled", `--queryformat=%{name} %{version}\n`)},
			parse:  parseNameVersion,
			remove: func(e entry) []string { return []string{"remove", e.Name} },
			after:  [][]string{{"autoremove"}},
		},
	}
}

// ---- parsers (pure, tested against real tool output) ----

func parseNpm(outs [][]byte) ([]entry, error) {
	var doc struct {
		Dependencies map[string]struct {
			Version string `json:"version"`
		} `json:"dependencies"`
	}
	if err := json.Unmarshal(outs[0], &doc); err != nil {
		return nil, fmt.Errorf("unexpected npm output: %w", err)
	}
	out := make([]entry, 0, len(doc.Dependencies))
	for name, d := range doc.Dependencies {
		out = append(out, entry{name, d.Version})
	}
	return out, nil
}

func parsePip(outs [][]byte) ([]entry, error) {
	var rows []struct{ Name, Version string }
	if err := json.Unmarshal(outs[0], &rows); err != nil {
		return nil, fmt.Errorf("unexpected pip output: %w", err)
	}
	out := make([]entry, len(rows))
	for i, r := range rows {
		out[i] = entry{r.Name, r.Version}
	}
	return out, nil
}

func parsePipx(outs [][]byte) ([]entry, error) {
	var doc struct {
		Venvs map[string]struct {
			Metadata struct {
				MainPackage struct {
					Version string `json:"package_version"`
				} `json:"main_package"`
			} `json:"metadata"`
		} `json:"venvs"`
	}
	if err := json.Unmarshal(outs[0], &doc); err != nil {
		return nil, fmt.Errorf("unexpected pipx output: %w", err)
	}
	out := make([]entry, 0, len(doc.Venvs))
	for name, v := range doc.Venvs {
		out = append(out, entry{name, v.Metadata.MainPackage.Version})
	}
	return out, nil
}

var cargoLine = regexp.MustCompile(`^(\S+) v(\S+?)( \(.*\))?:$`)

func parseCargo(outs [][]byte) ([]entry, error) {
	var out []entry
	for _, line := range strings.Split(string(outs[0]), "\n") {
		if m := cargoLine.FindStringSubmatch(strings.TrimRight(line, "\r")); m != nil {
			out = append(out, entry{m[1], m[2]})
		}
	}
	return out, nil
}

// parseBrewLeaves keeps only the formulae the user asked for (`brew leaves`), not
// the dependencies pulled in with them.
func parseBrewLeaves(outs [][]byte) ([]entry, error) {
	leaves := map[string]bool{}
	for _, l := range strings.Fields(string(outs[0])) {
		leaves[l] = true
	}
	var out []entry
	for _, line := range strings.Split(string(outs[1]), "\n") {
		f := strings.Fields(line)
		if len(f) >= 2 && leaves[f[0]] {
			out = append(out, entry{f[0], f[len(f)-1]})
		}
	}
	return out, nil
}

func parseBrewCasks(outs [][]byte) ([]entry, error) {
	var out []entry
	for _, line := range strings.Split(string(outs[0]), "\n") {
		if f := strings.Fields(line); len(f) >= 2 {
			out = append(out, entry{f[0], f[len(f)-1]})
		}
	}
	return out, nil
}

var gemLine = regexp.MustCompile(`^(\S+) \((.+)\)$`)

// parseGem skips default gems ("default: 1.2"): they ship with Ruby and cannot be
// uninstalled.
func parseGem(outs [][]byte) ([]entry, error) {
	var out []entry
	for _, line := range strings.Split(string(outs[0]), "\n") {
		m := gemLine.FindStringSubmatch(strings.TrimRight(line, "\r"))
		if m == nil {
			continue
		}
		versions := strings.Split(m[2], ", ")
		var installed []string
		for _, v := range versions {
			if !strings.HasPrefix(v, "default: ") {
				installed = append(installed, v)
			}
		}
		if len(installed) > 0 {
			out = append(out, entry{m[1], installed[0]})
		}
	}
	return out, nil
}

func parseScoop(outs [][]byte) ([]entry, error) {
	var doc struct {
		Apps []struct{ Name, Version string } `json:"apps"`
	}
	if err := json.Unmarshal(outs[0], &doc); err != nil {
		return nil, fmt.Errorf("unexpected scoop output: %w", err)
	}
	out := make([]entry, len(doc.Apps))
	for i, a := range doc.Apps {
		out[i] = entry{a.Name, a.Version}
	}
	return out, nil
}

// parseFlatpak reads tab-separated "application id, name, version" rows. The
// entry name is the application ID, which is what uninstall needs.
func parseFlatpak(outs [][]byte) ([]entry, error) {
	var out []entry
	for _, line := range strings.Split(string(outs[0]), "\n") {
		f := strings.Split(strings.TrimRight(line, "\r"), "\t")
		if len(f) >= 1 && strings.Contains(f[0], ".") {
			ver := ""
			if len(f) >= 3 {
				ver = f[2]
			}
			out = append(out, entry{f[0], ver})
		}
	}
	return out, nil
}

// aptProtectedPrefixes are packages a manual-install list contains but a desktop
// cannot lose: the kernel, the distribution's meta-packages, boot and session.
var aptProtectedPrefixes = []string{
	"linux-", "ubuntu-", "debian-", "grub", "systemd", "libc6", "apt", "dpkg", "sudo",
	"network-manager", "snapd", "xserver-xorg", "xorg", "gnome-shell", "gdm3", "lightdm",
	"sddm", "plasma-desktop", "kde-", "initramfs-tools", "cryptsetup", "openssh-server",
}

// parseApt joins `apt-mark showmanual` (what the user asked for) with dpkg's
// metadata, and drops what is essential to the system. apt's own list output is
// documented as unstable for scripts; these two tools are the stable interface.
func parseApt(outs [][]byte) ([]entry, error) {
	manual := map[string]bool{}
	for _, l := range strings.Fields(string(outs[0])) {
		manual[l] = true
	}
	var out []entry
	for _, line := range strings.Split(string(outs[1]), "\n") {
		f := strings.Split(strings.TrimRight(line, "\r"), "\t")
		if len(f) < 4 || !manual[f[0]] {
			continue
		}
		name, version, priority, essential := f[0], f[1], f[2], f[3]
		if essential == "yes" || priority == "required" || priority == "important" || hasAnyPrefix(name, aptProtectedPrefixes) {
			continue
		}
		out = append(out, entry{name, version})
	}
	return out, nil
}

func hasAnyPrefix(s string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

// parseSnap reads the table `snap list` prints and skips runtimes: bases, core
// and snapd are shared, and snap itself refuses to remove one that is in use.
func parseSnap(outs [][]byte) ([]entry, error) {
	var out []entry
	for i, line := range strings.Split(string(outs[0]), "\n") {
		f := strings.Fields(line)
		if i == 0 || len(f) < 2 || f[0] == "Name" {
			continue
		}
		notes := f[len(f)-1]
		if strings.Contains(notes, "base") || strings.Contains(notes, "core") || strings.Contains(notes, "snapd") ||
			strings.Contains(notes, "disabled") || f[0] == "bare" {
			continue
		}
		out = append(out, entry{f[0], f[1]})
	}
	return out, nil
}

// parseNameVersion reads "name version" lines (pacman -Qe, dnf repoquery).
func parseNameVersion(outs [][]byte) ([]entry, error) {
	var out []entry
	for _, line := range strings.Split(string(outs[0]), "\n") {
		if f := strings.Fields(line); len(f) >= 1 {
			ver := ""
			if len(f) >= 2 {
				ver = f[1]
			}
			out = append(out, entry{f[0], ver})
		}
	}
	return out, nil
}
