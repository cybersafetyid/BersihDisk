package uninstall

import (
	"context"
	"regexp"
	"sort"
	"strings"

	"bersihdisk/internal/safety"
)

// winApps lists what Windows shows under "Installed apps". The vendor's own
// uninstaller does the removal — it knows its files — and the leftovers are
// swept afterwards.
type winApps struct{}

func (winApps) ID() string { return "apps" }

var msiInstall = regexp.MustCompile(`(?i)msiexec(\.exe)?\s+/I\s*(\{[0-9A-F-]+\})`)

// uninstallLine picks what to run: the quiet form when the vendor provides one,
// and turns an MSI "/I" (install/repair) entry into "/X" (uninstall).
func uninstallLine(a regApp) string {
	line := a.QuietUninstall
	if line == "" {
		line = a.Uninstall
	}
	return msiInstall.ReplaceAllString(line, "MsiExec.exe /X$2")
}

func (winApps) List(_ context.Context, h Host) ([]Installed, error) {
	if h.OS != "windows" {
		return nil, ErrUnavailable
	}
	apps, err := listRegistryApps()
	if err != nil {
		return nil, err
	}
	var pkgs []Installed
	for _, a := range apps {
		rec := Recipe{
			Commands: []Command{{Raw: uninstallLine(a), Label: uninstallLine(a)}},
			Registry: []string{a.Key}, // left behind by an uninstaller that forgot to clean up
		}
		// Under Program Files the vendor's uninstaller owns the folder; the guard
		// would only show it as protected.
		if _, blocked := safety.Guard(a.InstallLocation); a.InstallLocation != "" && !blocked {
			rec.Paths = append(rec.Paths, Residue{Spec: a.InstallLocation, Exact: true})
		}
		extra, warn := extrasFor(a.Name)
		rec.Paths, rec.Warnings = append(rec.Paths, extra...), append(rec.Warnings, warn...)
		if usableName(a.Name) {
			for _, base := range []string{"{APPDATA}/", "{LOCALAPPDATA}/"} {
				rec.Paths = append(rec.Paths, Residue{Spec: base + a.Name, Reason: "nameMatch"})
			}
			if a.Publisher != "" && usableName(a.Publisher) {
				rec.Paths = append(rec.Paths, Residue{Spec: "{LOCALAPPDATA}/" + a.Publisher + "/" + a.Name, Reason: "nameMatch"})
			}
		}
		pkgs = append(pkgs, Installed{
			ID: "apps:" + a.Key, Provider: "apps", Kind: KindApp, Name: a.Name, Version: a.Version,
			Path: a.InstallLocation, Icon: "package", recipe: rec,
			iconSrc: iconSource{kind: "exe", ref: winIconExe(a.DisplayIcon)},
		})
	}
	sort.Slice(pkgs, func(i, j int) bool { return strings.ToLower(pkgs[i].Name) < strings.ToLower(pkgs[j].Name) })
	return pkgs, nil
}
