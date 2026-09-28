package uninstall

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// ---- macOS ----

// macApps lists .app bundles. The bundle identifier is what ties an app to its
// files in ~/Library, so those leftovers are exact; guesses by name are not.
type macApps struct{}

func (macApps) ID() string { return "apps" }

// appExtras are leftovers a bundle identifier cannot reveal: dot-folders in home,
// SDKs, and data folders named after the product.
var appExtras = map[string][]Residue{
	"com.microsoft.VSCode": {
		{Spec: "~/.vscode", Exact: true},
		{Spec: "~/.vscode-shared", Exact: true}, // named in VS Code's own clean-uninstall steps
		{Spec: "~/Library/Application Support/Code", Exact: true},
	},
	"com.microsoft.VSCodeInsiders": {
		{Spec: "~/.vscode-insiders", Exact: true},
		{Spec: "~/Library/Application Support/Code - Insiders", Exact: true},
	},
	"com.todesktop.230313mzl4w4u92": { // Cursor
		{Spec: "~/.cursor", Exact: true},
		{Spec: "~/Library/Application Support/Cursor", Exact: true},
	},
	"com.google.android.studio": {
		{Spec: "~/Library/Application Support/Google/AndroidStudio*", Exact: true},
		{Spec: "~/Library/Caches/Google/AndroidStudio*", Exact: true},
		{Spec: "~/Library/Logs/Google/AndroidStudio*", Exact: true},
		{Spec: "~/Library/Preferences/AndroidStudio*", Exact: true},
		// Shared with command-line builds and other IDEs, and heavy to re-download.
		{Spec: "~/Library/Android/sdk", Exact: true, Reason: "sharedSdk"},
		{Spec: "~/.android", Exact: true, Reason: "emulatorData"},
	},
	// Docker's documented uninstall list (docs.docker.com/desktop/uninstall).
	"com.docker.docker": {
		{Spec: "~/.docker", Exact: true, Reason: "config"},
		{Spec: "~/Library/Application Support/Docker Desktop", Exact: true},
		{Spec: "~/Library/Containers/com.docker.docker", Exact: true, Reason: "containerData"},
		{Spec: "~/Library/Group Containers/group.com.docker", Exact: true},
		{Spec: "~/Library/Logs/Docker Desktop", Exact: true},
		{Spec: "~/Library/Preferences/com.electron.docker-frontend.plist", Exact: true},
		{Spec: "~/Library/Saved Application State/com.electron.docker-frontend.savedState", Exact: true},
		{Spec: "/Library/PrivilegedHelperTools/com.docker.vmnetd", Exact: true, NeedsAdmin: true},
		{Spec: "/Library/LaunchDaemons/com.docker.vmnetd.plist", Exact: true, NeedsAdmin: true},
	},
}

var appExtraWarnings = map[string][]string{
	"com.docker.docker": {"containerData"},
}

// appNative lists vendors that ship their own uninstaller inside the bundle. It
// asks for an administrator password, so it is shown for copying, not run.
var appNative = map[string]string{
	"com.docker.docker": "Contents/MacOS/uninstall",
}

// nameExtras are leftovers for well-known developer apps on Windows and Linux,
// where there is no bundle identifier to derive them from. They come from each
// vendor's own uninstall documentation. A rule applies when the app name contains
// every word of match and none of exclude (case-insensitive).
var nameExtras = []struct {
	match, exclude []string
	res            []Residue
	warn           []string
}{
	{ // code.visualstudio.com/docs/setup/uninstall#_clean-uninstall
		match: []string{"visual studio code"}, exclude: []string{"insiders"},
		res: []Residue{
			{Spec: "~/.vscode", Exact: true}, {Spec: "~/.vscode-shared", Exact: true},
			{Spec: "win:{APPDATA}/Code", Exact: true}, {Spec: "linux:{XDG_CONFIG_HOME}/Code", Exact: true},
		},
	},
	{
		match: []string{"visual studio code", "insiders"},
		res: []Residue{
			{Spec: "~/.vscode-insiders", Exact: true},
			{Spec: "win:{APPDATA}/Code - Insiders", Exact: true}, {Spec: "linux:{XDG_CONFIG_HOME}/Code - Insiders", Exact: true},
		},
	},
	{ // docs.docker.com/desktop/uninstall
		match: []string{"docker desktop"},
		res: []Residue{
			{Spec: "~/.docker", Exact: true, Reason: "config"},
			{Spec: "win:{PROGRAMDATA}/Docker", Exact: true}, {Spec: "win:{PROGRAMDATA}/DockerDesktop", Exact: true},
			{Spec: "win:{APPDATA}/Docker", Exact: true}, {Spec: "win:{APPDATA}/Docker Desktop", Exact: true},
			{Spec: "win:{LOCALAPPDATA}/Docker", Exact: true, Reason: "containerData"},
		},
		warn: []string{"containerData"},
	},
	{
		match: []string{"android studio"},
		res: []Residue{
			{Spec: "win:{LOCALAPPDATA}/Google/AndroidStudio*", Exact: true},
			{Spec: "win:{APPDATA}/Google/AndroidStudio*", Exact: true},
			{Spec: "linux:{XDG_CONFIG_HOME}/Google/AndroidStudio*", Exact: true},
			{Spec: "linux:{XDG_CACHE_HOME}/Google/AndroidStudio*", Exact: true},
			{Spec: "win:{LOCALAPPDATA}/Android", Exact: true, Reason: "sharedSdk"},
			{Spec: "linux:~/Android", Exact: true, Reason: "sharedSdk"},
			{Spec: "~/.android", Exact: true, Reason: "emulatorData"},
		},
	},
}

// extrasFor returns the documented leftovers for a known developer app.
func extrasFor(name string) (res []Residue, warn []string) {
	lower := strings.ToLower(name)
outer:
	for _, e := range nameExtras {
		for _, m := range e.match {
			if !strings.Contains(lower, m) {
				continue outer
			}
		}
		for _, x := range e.exclude {
			if strings.Contains(lower, x) {
				continue outer
			}
		}
		res, warn = append(res, e.res...), append(warn, e.warn...)
	}
	return res, warn
}

// plutilInfo is the part of Info.plist the list needs.
type plutilInfo struct {
	ID       string `json:"CFBundleIdentifier"`
	Version  string `json:"CFBundleShortVersionString"`
	Name     string `json:"CFBundleName"`
	IconFile string `json:"CFBundleIconFile"`
}

func (macApps) List(ctx context.Context, h Host) ([]Installed, error) {
	if h.OS != "darwin" {
		return nil, ErrUnavailable
	}
	dirs := []string{"/Applications"}
	if h.Home != "" {
		dirs = append(dirs, filepath.Join(h.Home, "Applications"))
	}
	var bundles []string
	for _, d := range dirs {
		ents, err := os.ReadDir(d)
		if err != nil {
			continue
		}
		for _, e := range ents {
			if strings.HasSuffix(e.Name(), ".app") {
				bundles = append(bundles, filepath.Join(d, e.Name()))
			}
		}
	}

	var (
		mu   sync.Mutex
		wg   sync.WaitGroup
		pkgs []Installed
		sem  = make(chan struct{}, 8)
	)
	for _, b := range bundles {
		wg.Add(1)
		go func(bundle string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			if p, ok := macPackage(ctx, h, bundle); ok {
				mu.Lock()
				pkgs = append(pkgs, p)
				mu.Unlock()
			}
		}(b)
	}
	wg.Wait()
	sort.Slice(pkgs, func(i, j int) bool { return strings.ToLower(pkgs[i].Name) < strings.ToLower(pkgs[j].Name) })
	return pkgs, nil
}

// macPackage builds the package for one bundle, or reports false for an app that
// must not be offered (Apple's own, or this app).
func macPackage(ctx context.Context, h Host, bundle string) (Installed, bool) {
	name := strings.TrimSuffix(filepath.Base(bundle), ".app")
	if name == "BersihDisk" {
		return Installed{}, false
	}
	var info plutilInfo
	if plutil, err := h.Run.Look("plutil"); err == nil {
		if out, err := h.Run.Run(ctx, plutil, "-convert", "json", "-o", "-", filepath.Join(bundle, "Contents", "Info.plist")); err == nil {
			_ = json.Unmarshal(out, &info)
		}
	}
	// Apple's apps are part of the system; Xcode is the exception developers do remove.
	if strings.HasPrefix(info.ID, "com.apple.") && info.ID != "com.apple.dt.Xcode" {
		return Installed{}, false
	}
	var notes []string
	if _, err := os.Stat(filepath.Join(bundle, "Contents", "_MASReceipt")); err == nil {
		notes = append(notes, "appStore")
	}
	rec := Recipe{
		Paths:    append([]Residue{{Spec: bundle, Exact: true}}, macResidue(name, info.ID)...),
		Warnings: appExtraWarnings[info.ID],
	}
	if rel, ok := appNative[info.ID]; ok {
		if tool := filepath.Join(bundle, rel); fileExists(tool) {
			rec.Commands = []Command{{Argv: []string{tool}, Label: quoteArg(tool), NeedsAdmin: true, Optional: true}}
		}
	}
	if info.ID == "com.apple.dt.Xcode" {
		rec.Paths = append(rec.Paths, xcodeResidue...)
		rec.Warnings = append(rec.Warnings, "xcodeUserData")
	}
	return Installed{
		ID: "apps:" + bundle, Provider: "apps", Kind: KindApp, Name: name,
		Version: info.Version, Path: bundle, Icon: "package", Notes: notes, recipe: rec,
		iconSrc: iconSource{kind: "icns", ref: macIconFile(bundle, info.IconFile)},
	}, true
}

// xcodeResidue: DerivedData and simulators regenerate, but Archives and UserData
// (key bindings, snippets, signing) are the developer's own work.
var xcodeResidue = []Residue{
	{Spec: "~/Library/Developer/Xcode/DerivedData", Exact: true},
	{Spec: "~/Library/Developer/Xcode/iOS DeviceSupport", Exact: true},
	{Spec: "~/Library/Developer/CoreSimulator", Exact: true, Reason: "emulatorData"},
	{Spec: "~/Library/Developer/Xcode/Archives", Exact: true, Reason: "userData"},
	{Spec: "~/Library/Developer/Xcode/UserData", Exact: true, Reason: "userData"},
}

// macResidue lists the per-user locations macOS apps leave behind. Locations
// derived from the bundle identifier are exact; those derived from the display
// name are guesses (Exact=false) and start unchecked. System-wide locations need
// an administrator, so they are reported for manual removal.
func macResidue(name, bundleID string) []Residue {
	var out []Residue
	if validBundleID(bundleID) {
		for _, spec := range []string{
			"~/Library/Application Support/" + bundleID,
			"~/Library/Caches/" + bundleID,
			"~/Library/Preferences/" + bundleID + ".plist",
			"~/Library/Preferences/ByHost/" + bundleID + ".*",
			"~/Library/Saved Application State/" + bundleID + ".savedState",
			"~/Library/Containers/" + bundleID,
			"~/Library/Group Containers/*." + bundleID,
			"~/Library/Group Containers/group." + bundleID,
			"~/Library/HTTPStorages/" + bundleID,
			"~/Library/HTTPStorages/" + bundleID + ".binarycookies",
			"~/Library/WebKit/" + bundleID,
			"~/Library/Cookies/" + bundleID + ".binarycookies",
			"~/Library/Application Scripts/" + bundleID,
			"~/Library/LaunchAgents/" + bundleID + "*.plist",
		} {
			out = append(out, Residue{Spec: spec, Exact: true})
		}
		for _, spec := range []string{
			"/Library/LaunchDaemons/" + bundleID + "*.plist",
			"/Library/PrivilegedHelperTools/" + bundleID + "*",
		} {
			out = append(out, Residue{Spec: spec, Exact: true, NeedsAdmin: true})
		}
		out = append(out, appExtras[bundleID]...)
	}
	if usableName(name) {
		for _, spec := range []string{
			"~/Library/Application Support/" + name,
			"~/Library/Caches/" + name,
			"~/Library/Logs/" + name,
		} {
			out = append(out, Residue{Spec: spec, Reason: "nameMatch"})
		}
	}
	return out
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// validBundleID rejects identifiers that would turn a spec into a wide glob.
func validBundleID(id string) bool {
	return len(id) >= 5 && strings.Contains(id, ".") && !strings.ContainsAny(id, `*?[]\/`)
}

func usableName(name string) bool {
	return len(name) >= 3 && !strings.ContainsAny(name, `*?[]\/`)
}

// ---- Linux ----

// linuxApps lists per-user launchers ($XDG_DATA_HOME/applications, default
// ~/.local/share/applications), where AppImage integrators and manual installs put
// their .desktop files. Distribution packages are handled by the apt/dnf/pacman/
// snap/flatpak providers.
type linuxApps struct{}

func (linuxApps) ID() string { return "apps" }

// desktopEntry is the useful part of a .desktop file.
type desktopEntry struct{ Name, Exec, Icon string }

// parseDesktop reads the [Desktop Entry] group of a .desktop file.
func parseDesktop(content string) (d desktopEntry, ok bool) {
	inGroup := false
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "["):
			inGroup = line == "[Desktop Entry]"
		case inGroup && strings.HasPrefix(line, "Name=") && d.Name == "":
			d.Name = strings.TrimPrefix(line, "Name=")
		case inGroup && strings.HasPrefix(line, "Exec="):
			d.Exec = strings.TrimPrefix(line, "Exec=")
		case inGroup && strings.HasPrefix(line, "Icon="):
			d.Icon = strings.TrimPrefix(line, "Icon=")
		}
	}
	return d, d.Name != ""
}

// execTarget returns the program a desktop Exec line starts, when it is a file.
func execTarget(exec string) string {
	exec = strings.TrimSpace(exec)
	if strings.HasPrefix(exec, `"`) {
		if end := strings.Index(exec[1:], `"`); end >= 0 {
			return exec[1 : 1+end]
		}
	}
	if i := strings.IndexAny(exec, " \t"); i >= 0 {
		exec = exec[:i]
	}
	return exec
}

func (linuxApps) List(_ context.Context, h Host) ([]Installed, error) {
	if h.OS != "linux" || h.Home == "" {
		return nil, ErrUnavailable
	}
	dirs := h.Expand("{XDG_DATA_HOME}/applications")
	if len(dirs) == 0 {
		return nil, ErrUnavailable
	}
	files, _ := filepath.Glob(filepath.Join(dirs[0], "*.desktop"))
	var pkgs []Installed
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		d, ok := parseDesktop(string(data))
		if !ok {
			continue
		}
		rec := Recipe{Paths: []Residue{{Spec: f, Exact: true}}}
		// A launcher's program and icon are removed only when they live in the home folder.
		for _, p := range []string{execTarget(d.Exec), d.Icon} {
			if filepath.IsAbs(p) && strings.HasPrefix(p, h.Home+string(filepath.Separator)) {
				rec.Paths = append(rec.Paths, Residue{Spec: p, Exact: true})
			}
		}
		if usableName(d.Name) {
			for _, base := range []string{"{XDG_CONFIG_HOME}/", "{XDG_DATA_HOME}/", "{XDG_CACHE_HOME}/", "{XDG_STATE_HOME}/"} {
				rec.Paths = append(rec.Paths, Residue{Spec: base + d.Name, Reason: "nameMatch"})
			}
		}
		extra, warn := extrasFor(d.Name)
		rec.Paths, rec.Warnings = append(rec.Paths, extra...), append(rec.Warnings, warn...)
		pkgs = append(pkgs, Installed{
			ID: "apps:" + f, Provider: "apps", Kind: KindApp, Name: d.Name, Path: f,
			Icon: "package", recipe: rec,
			iconSrc: iconSource{kind: "file", ref: linuxIconFile(h.Env, d.Icon)},
		})
	}
	sort.Slice(pkgs, func(i, j int) bool { return strings.ToLower(pkgs[i].Name) < strings.ToLower(pkgs[j].Name) })
	return pkgs, nil
}
