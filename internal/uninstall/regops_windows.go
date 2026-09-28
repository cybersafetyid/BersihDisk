//go:build windows

package uninstall

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

type regApp struct {
	Key, Name, Version, Publisher, InstallLocation, Uninstall, QuietUninstall, DisplayIcon string
	SizeKB                                                                                 uint32
}

const uninstallPath = `SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`

// listRegistryApps reads the installed-programs list the Settings app shows: the
// 64-bit and 32-bit machine views and the current user's own installs.
func listRegistryApps() ([]regApp, error) {
	views := []struct {
		label  string
		root   registry.Key
		access uint32
	}{
		{"HKLM", registry.LOCAL_MACHINE, registry.READ | registry.WOW64_64KEY},
		{"HKLM", registry.LOCAL_MACHINE, registry.READ | registry.WOW64_32KEY},
		{"HKCU", registry.CURRENT_USER, registry.READ},
	}
	seen := map[string]bool{}
	var apps []regApp
	for _, v := range views {
		parent, err := registry.OpenKey(v.root, uninstallPath, v.access)
		if err != nil {
			continue
		}
		names, _ := parent.ReadSubKeyNames(-1)
		for _, n := range names {
			sub, err := registry.OpenKey(parent, n, registry.QUERY_VALUE)
			if err != nil {
				continue
			}
			str := func(name string) string { s, _, _ := sub.GetStringValue(name); return s }
			num := func(name string) uint32 { v, _, _ := sub.GetIntegerValue(name); return uint32(v) }
			a := regApp{
				Key:             v.label + `\` + uninstallPath + `\` + n,
				Name:            str("DisplayName"),
				Version:         str("DisplayVersion"),
				Publisher:       str("Publisher"),
				InstallLocation: strings.Trim(str("InstallLocation"), `"`),
				Uninstall:       str("UninstallString"),
				QuietUninstall:  str("QuietUninstallString"),
				DisplayIcon:     str("DisplayIcon"),
				SizeKB:          num("EstimatedSize"),
			}
			skip := a.Name == "" || (a.Uninstall == "" && a.QuietUninstall == "") ||
				num("SystemComponent") == 1 || str("ParentKeyName") != "" ||
				strings.Contains(strings.ToLower(str("ReleaseType")), "update") ||
				strings.Contains(strings.ToLower(str("ReleaseType")), "hotfix")
			sub.Close()
			id := strings.ToLower(a.Name + "|" + a.Version)
			if skip || seen[id] {
				continue
			}
			seen[id] = true
			apps = append(apps, a)
		}
		parent.Close()
	}
	return apps, nil
}

// splitKey separates "HKCU\Software\X" into the root key and the sub path.
func splitKey(key string) (registry.Key, string, bool) {
	i := strings.Index(key, `\`)
	if i < 0 {
		return 0, "", false
	}
	switch strings.ToUpper(key[:i]) {
	case "HKCU":
		return registry.CURRENT_USER, key[i+1:], true
	case "HKLM":
		return registry.LOCAL_MACHINE, key[i+1:], true
	}
	return 0, "", false
}

func registryKeyExists(key string) bool {
	root, sub, ok := splitKey(key)
	if !ok {
		return false
	}
	k, err := registry.OpenKey(root, sub, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	k.Close()
	return true
}

// exportKey writes a .reg backup that the user can double-click to restore.
func exportKey(key, backupDir string) (string, error) {
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return "", err
	}
	name := strings.NewReplacer(`\`, "_", ` `, "_", `:`, "_").Replace(key)
	if len(name) > 80 {
		name = name[len(name)-80:]
	}
	file := filepath.Join(backupDir, fmt.Sprintf("%s-%s.reg", time.Now().Format("20060102-150405"), name))
	if out, err := exec.Command("reg", "export", key, file, "/y").CombinedOutput(); err != nil {
		return "", fmt.Errorf("could not back up %s: %v %s", key, err, strings.TrimSpace(string(out)))
	}
	return file, nil
}

// deleteRegistryKey backs the key up, then removes it and everything below it.
// Only the current user's hive is deleted here; machine-wide keys need an
// administrator and are reported for manual removal.
func deleteRegistryKey(key, backupDir string) error {
	root, sub, ok := splitKey(key)
	if !ok || root != registry.CURRENT_USER {
		return fmt.Errorf("only HKCU keys are removed automatically")
	}
	if strings.Count(sub, `\`) < 1 {
		return fmt.Errorf("refusing to delete a top-level key")
	}
	if _, err := exportKey(key, backupDir); err != nil {
		return err
	}
	return deleteTree(root, sub)
}

func deleteTree(root registry.Key, sub string) error {
	k, err := registry.OpenKey(root, sub, registry.ENUMERATE_SUB_KEYS|registry.QUERY_VALUE)
	if err != nil {
		return err
	}
	names, _ := k.ReadSubKeyNames(-1)
	k.Close()
	for _, n := range names {
		if err := deleteTree(root, sub+`\`+n); err != nil {
			return err
		}
	}
	return registry.DeleteKey(root, sub)
}

// userPathHas returns which of the wanted entries the user's PATH contains.
func userPathHas(want []string) []string {
	k, err := registry.OpenKey(registry.CURRENT_USER, `Environment`, registry.QUERY_VALUE)
	if err != nil {
		return nil
	}
	defer k.Close()
	value, _, err := k.GetStringValue("Path")
	if err != nil {
		return nil
	}
	_, gone := dropPathEntries(value, want, ";")
	return gone
}

// removeUserPathEntries drops entries from the user's PATH after backing the
// Environment key up, then tells running programs the environment changed.
func removeUserPathEntries(drop []string, backupDir string) ([]string, error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, `Environment`, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return nil, err
	}
	defer k.Close()
	value, kind, err := k.GetStringValue("Path")
	if err != nil {
		return nil, err
	}
	next, gone := dropPathEntries(value, drop, ";")
	if len(gone) == 0 {
		return nil, nil
	}
	if _, err := exportKey(`HKCU\Environment`, backupDir); err != nil {
		return nil, err
	}
	// Keep the value's type: PATH is normally REG_EXPAND_SZ (it holds %USERPROFILE%
	// style references), and rewriting it as REG_SZ would freeze them. Writing the
	// registry directly also avoids `setx`, which silently truncates at 1024 characters.
	if kind == registry.EXPAND_SZ {
		err = k.SetExpandStringValue("Path", next)
	} else {
		err = k.SetStringValue("Path", next)
	}
	if err != nil {
		return nil, err
	}
	broadcastEnvChange()
	return gone, nil
}

func broadcastEnvChange() {
	proc := windows.NewLazySystemDLL("user32.dll").NewProc("SendMessageTimeoutW")
	env, _ := syscall.UTF16PtrFromString("Environment")
	const hwndBroadcast, wmSettingChange, smtoAbortIfHung = 0xffff, 0x001A, 0x0002
	proc.Call(hwndBroadcast, wmSettingChange, 0, uintptr(unsafe.Pointer(env)), smtoAbortIfHung, 2000, 0)
}

// runRaw starts a command line taken from the registry and waits for it. `start
// /wait` goes through ShellExecute, so an uninstaller that asks for elevation
// shows its normal UAC prompt instead of failing with error 740.
func runRaw(ctx context.Context, cmdline string) error {
	comspec := os.Getenv("COMSPEC")
	if comspec == "" {
		comspec = `C:\Windows\System32\cmd.exe`
	}
	c := exec.CommandContext(ctx, comspec)
	c.SysProcAttr = &syscall.SysProcAttr{CmdLine: `/S /C "start "" /wait ` + cmdline + `"`}
	return c.Run()
}
