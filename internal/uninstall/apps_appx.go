package uninstall

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// appxApps lists Microsoft Store / MSIX apps. They have no key under
// ...\CurrentVersion\Uninstall, so the registry listing never shows them; the
// supported interface is PowerShell's Get-AppxPackage / Remove-AppxPackage.
// Frameworks, non-removable and system-signed packages are left out, and
// Remove-AppxPackage deletes the app's data under %LOCALAPPDATA%\Packages itself.
type appxApps struct{}

func (appxApps) ID() string { return "appx" }

// appxScript prints the removable packages as JSON. Version is converted to a
// string because Windows PowerShell 5.1 would serialise it as an object.
const appxScript = `Get-AppxPackage | Where-Object { -not $_.IsFramework -and -not $_.NonRemovable -and $_.SignatureKind -ne 'System' } | ` +
	`Select-Object Name, PackageFullName, @{n='Version';e={$_.Version.ToString()}}, InstallLocation | ConvertTo-Json -Compress`

// appxSystemPrefixes are inbox components that are removable on paper but part of Windows.
// The dot matters: "Microsoft.Windows." is inbox components, while
// "Microsoft.WindowsTerminal" is an app a developer may well remove.
var appxSystemPrefixes = []string{"Microsoft.Windows.", "MicrosoftWindows.", "Windows.", "Microsoft.UI.", "Microsoft.VCLibs", "Microsoft.NET.", "Microsoft.WindowsAppRuntime"}

type appxEntry struct {
	Name            string `json:"Name"`
	PackageFullName string `json:"PackageFullName"`
	Version         string `json:"Version"`
	InstallLocation string `json:"InstallLocation"`
}

// parseAppx accepts both shapes ConvertTo-Json produces: an array, or a bare
// object when exactly one package matched.
func parseAppx(out []byte) ([]appxEntry, error) {
	out = []byte(strings.TrimSpace(string(out)))
	if len(out) == 0 {
		return nil, nil
	}
	var many []appxEntry
	if out[0] == '[' {
		if err := json.Unmarshal(out, &many); err != nil {
			return nil, fmt.Errorf("unexpected Get-AppxPackage output: %w", err)
		}
	} else {
		var one appxEntry
		if err := json.Unmarshal(out, &one); err != nil {
			return nil, fmt.Errorf("unexpected Get-AppxPackage output: %w", err)
		}
		many = []appxEntry{one}
	}
	var keep []appxEntry
	for _, e := range many {
		if e.Name != "" && e.PackageFullName != "" && !hasAnyPrefix(e.Name, appxSystemPrefixes) {
			keep = append(keep, e)
		}
	}
	return keep, nil
}

func (appxApps) List(ctx context.Context, h Host) ([]Installed, error) {
	if h.OS != "windows" {
		return nil, ErrUnavailable
	}
	ps, err := h.Run.Look("powershell")
	if err != nil {
		return nil, ErrUnavailable
	}
	out, err := h.Run.Run(ctx, ps, "-NoProfile", "-NonInteractive", "-Command", appxScript)
	if err != nil {
		return nil, fmt.Errorf("appx: %w", err)
	}
	entries, err := parseAppx(out)
	if err != nil {
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool { return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name) })

	pkgs := make([]Installed, 0, len(entries))
	for _, e := range entries {
		argv := []string{ps, "-NoProfile", "-NonInteractive", "-Command", "Remove-AppxPackage -Package '" + strings.ReplaceAll(e.PackageFullName, "'", "''") + "'"}
		pkgs = append(pkgs, Installed{
			ID: "appx:" + e.PackageFullName, Provider: "appx", Kind: KindApp, Name: e.Name,
			Version: e.Version, Path: e.InstallLocation, Icon: "package",
			recipe: Recipe{Commands: []Command{{
				Argv: argv, Label: "Remove-AppxPackage " + e.PackageFullName,
			}}},
		})
	}
	return pkgs, nil
}
