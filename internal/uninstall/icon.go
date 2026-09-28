package uninstall

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// iconSource says where an application's icon can be read from. It stays in the
// backend: the UI only ever asks for "the icon of package X".
type iconSource struct {
	kind string // "icns" (macOS bundle icon), "file" (png/svg on disk), "exe" (Windows executable)
	ref  string
}

const maxIconBytes = 512 << 10 // an icon larger than this is not an icon

// Icon returns the application's icon as a data URL, or an error when it has
// none the app can read (the UI then keeps its generic icon). Results are cached.
func (inv *Inventory) Icon(ctx context.Context, id string) (string, error) {
	inv.mu.Lock()
	if url, ok := inv.icons[id]; ok {
		inv.mu.Unlock()
		return url, nil
	}
	pkg, ok := inv.pkgs[id]
	inv.mu.Unlock()
	if !ok || pkg.iconSrc.ref == "" {
		return "", fmt.Errorf("no icon")
	}
	inv.iconSem <- struct{}{} // a few conversions at a time; a long list scrolls in bursts
	defer func() { <-inv.iconSem }()

	var (
		url string
		err error
	)
	switch pkg.iconSrc.kind {
	case "icns":
		url, err = icnsToDataURL(ctx, inv.host, pkg.iconSrc.ref)
	case "file":
		url, err = fileToDataURL(pkg.iconSrc.ref)
	case "exe":
		url, err = exeToDataURL(ctx, inv.host, pkg.iconSrc.ref)
	default:
		err = fmt.Errorf("unknown icon source")
	}
	if err != nil {
		return "", err
	}
	inv.mu.Lock()
	inv.icons[id] = url
	inv.mu.Unlock()
	return url, nil
}

func dataURL(mime string, data []byte) string {
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(data)
}

// fileToDataURL reads a PNG or SVG icon.
func fileToDataURL(path string) (string, error) {
	st, err := os.Stat(path)
	if err != nil || st.Size() > maxIconBytes {
		return "", fmt.Errorf("icon unavailable")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png":
		return dataURL("image/png", data), nil
	case ".svg":
		return dataURL("image/svg+xml", data), nil
	}
	return "", fmt.Errorf("unsupported icon type")
}

// icnsToDataURL converts a macOS .icns to a small PNG with sips, which ships with
// macOS. A WebView cannot draw .icns itself.
func icnsToDataURL(ctx context.Context, h Host, icns string) (string, error) {
	sips, err := h.Run.Look("sips")
	if err != nil {
		return "", err
	}
	dir, err := os.MkdirTemp("", "bersihdisk-icon-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(dir)
	out := filepath.Join(dir, "icon.png")
	if _, err := h.Run.Run(ctx, sips, "-s", "format", "png", "-Z", "64", icns, "--out", out); err != nil {
		return "", err
	}
	return fileToDataURL(out)
}

// exeScript prints the executable's associated icon as base64 PNG. The path is
// passed inside a single-quoted PowerShell string with quotes doubled, so it is
// data, never code.
func exeScript(path string) string {
	return "Add-Type -AssemblyName System.Drawing; " +
		"$i=[System.Drawing.Icon]::ExtractAssociatedIcon('" + strings.ReplaceAll(path, "'", "''") + "'); " +
		"$m=New-Object IO.MemoryStream; $i.ToBitmap().Save($m,[Drawing.Imaging.ImageFormat]::Png); " +
		"[Convert]::ToBase64String($m.ToArray())"
}

// exeToDataURL extracts a Windows program's icon through System.Drawing.
func exeToDataURL(ctx context.Context, h Host, exe string) (string, error) {
	ps, err := h.Run.Look("powershell")
	if err != nil {
		return "", err
	}
	out, err := h.Run.Run(ctx, ps, "-NoProfile", "-NonInteractive", "-Command", exeScript(exe))
	if err != nil {
		return "", err
	}
	b64 := strings.TrimSpace(string(out))
	if b64 == "" || len(b64) > maxIconBytes {
		return "", fmt.Errorf("icon unavailable")
	}
	return "data:image/png;base64," + b64, nil
}

// macIconFile finds the bundle's .icns from Info.plist's CFBundleIconFile (the
// extension is optional there), falling back to the conventional AppIcon.icns.
func macIconFile(bundle, iconFile string) string {
	res := filepath.Join(bundle, "Contents", "Resources")
	for _, name := range []string{iconFile, "AppIcon"} {
		if name == "" {
			continue
		}
		if filepath.Ext(name) == "" {
			name += ".icns"
		}
		if p := filepath.Join(res, filepath.Base(name)); fileExists(p) {
			return p
		}
	}
	return ""
}

// linuxIconFile resolves a .desktop Icon= value: an absolute path as is, or a
// theme name searched in the standard icon folders, preferring sizes near 64px.
func linuxIconFile(env Env, icon string) string {
	if icon == "" {
		return ""
	}
	if filepath.IsAbs(icon) {
		if fileExists(icon) {
			return icon
		}
		return ""
	}
	var roots []string
	roots = append(roots, env.Expand("{XDG_DATA_HOME}/icons")...)
	roots = append(roots, env.Expand("~/.icons")...)
	roots = append(roots, "/usr/share/icons", "/usr/share/pixmaps")
	sizes := []string{"64x64", "48x48", "128x128", "96x96", "256x256", "scalable"}
	for _, root := range roots {
		for _, sz := range sizes {
			for _, ext := range []string{".png", ".svg"} {
				if p := filepath.Join(root, "hicolor", sz, "apps", icon+ext); fileExists(p) {
					return p
				}
			}
		}
		for _, ext := range []string{".png", ".svg"} {
			if p := filepath.Join(root, icon+ext); fileExists(p) {
				return p
			}
		}
	}
	return ""
}

// winIconExe picks the executable to read an icon from: the registry's
// DisplayIcon ("C:\path\app.exe,0"), else nothing.
func winIconExe(displayIcon string) string {
	p := strings.TrimSpace(displayIcon)
	if i := strings.LastIndex(p, ","); i > 0 {
		p = p[:i]
	}
	p = strings.Trim(p, `"`)
	if strings.HasSuffix(strings.ToLower(p), ".exe") || strings.HasSuffix(strings.ToLower(p), ".ico") {
		return p
	}
	return ""
}
