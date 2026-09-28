#!/usr/bin/env bash
# package.sh — build BersihDisk for THIS operating system and package it into dist/.
#
# The same script runs on a developer machine (`make package`) and in CI
# (.github/workflows/release.yml), so a local build and a released build are made
# by identical steps. Wails cannot cross-compile a cgo app, so each OS packages
# itself; CI runs this once per OS.
#
#   macOS    dist/bersihdisk-vX.Y.Z-macos-universal.dmg
#   Windows  dist/bersihdisk-vX.Y.Z-windows-x86_64-setup.exe   (NSIS installer)
#            dist/bersihdisk-vX.Y.Z-windows-x86_64-portable.zip
#   Linux    dist/bersihdisk-vX.Y.Z-linux-x86_64.deb
#            dist/bersihdisk-vX.Y.Z-linux-x86_64.tar.gz
#
# The names are a contract with internal/updater (pickAsset): the OS word, the
# architecture and the extension decide which file the in-app updater downloads.
# internal/updater/updater_test.go pins the same names.
#
# Usage: scripts/package.sh [--skip-build]
# Env:   VERSION (default: VERSION file)  REPO (owner/name)  WAILS (CLI path)
#        DIST (default: dist)

set -euo pipefail
cd "$(dirname "$0")/.."

APP=bersihdisk
TITLE=BersihDisk
VERSION="${VERSION:-$(tr -d '[:space:]' < VERSION)}"
REPO="${REPO:-cybersafetyid/BersihDisk}"
DIST="${DIST:-dist}"
LDFLAGS="-X main.appVersion=$VERSION -X main.releaseRepo=$REPO"
NAME="$APP-v$VERSION"

SKIP_BUILD=0
[ "${1:-}" = "--skip-build" ] && SKIP_BUILD=1

die() { echo "✘ $*" >&2; exit 1; }

# ---- Toolchain ---------------------------------------------------------------

# Wails CLI: explicit WAILS, else the patched CLI from `make install-wails-fix`
# (the published v2.12.0 fails on Go >= 1.24), else whatever is on PATH.
resolve_wails() {
	if [ -n "${WAILS:-}" ]; then echo "$WAILS"; return; fi
	if [ -x "$HOME/.local/bin/wails-fixed" ]; then echo "$HOME/.local/bin/wails-fixed"; return; fi
	if command -v wails >/dev/null 2>&1; then command -v wails; return; fi
	if [ -x "$(go env GOPATH)/bin/wails" ]; then echo "$(go env GOPATH)/bin/wails"; return; fi
	die "wails CLI not found — run 'make install-wails-fix' or 'go install github.com/wailsapp/wails/v2/cmd/wails@v2.12.0'"
}

case "$(uname -s)" in
	Darwin) OS=macos ;;
	Linux) OS=linux ;;
	MINGW* | MSYS* | CYGWIN*) OS=windows ;;
	*) die "unsupported OS: $(uname -s)" ;;
esac

# zip_dir DIR OUT.zip — zip the directory DIR (its own name is kept as the top
# folder). Git Bash on Windows ships no `zip`, so PowerShell does it there.
zip_dir() {
	local dir="$1" out="$2"
	rm -f "$out"
	if command -v zip >/dev/null 2>&1; then
		(cd "$(dirname "$dir")" && zip -qry "$(cd "$(dirname "$out")" && pwd)/$(basename "$out")" "$(basename "$dir")")
	elif command -v powershell.exe >/dev/null 2>&1; then
		powershell.exe -NoProfile -Command \
			"Compress-Archive -Path '$(cygpath -w "$dir")' -DestinationPath '$(cygpath -w "$out")' -Force"
	else
		die "no zip tool available"
	fi
}

mkdir -p "$DIST"
DIST="$(cd "$DIST" && pwd)"
WAILS_BIN="$(resolve_wails)"
# Drop artifacts of an earlier run for this OS, so nothing stale is published.
rm -f "$DIST/$NAME-$OS"-*
echo "▶ Packaging $TITLE v$VERSION for $OS ($($WAILS_BIN version 2>/dev/null | head -1))"

# ---- macOS: universal .app -> .dmg -------------------------------------------

package_macos() {
	local app="build/bin/$APP.app" out="$DIST/$NAME-macos-universal.dmg"
	if [ "$SKIP_BUILD" -eq 0 ]; then
		"$WAILS_BIN" build -platform darwin/universal -ldflags "$LDFLAGS" -m -clean
	fi
	[ -d "$app" ] || die "$app missing — the build failed"

	# Ad-hoc signature: arm64 will not run unsigned code, and a deep signature keeps
	# the bundle consistent. It is not notarisation — see README (first launch).
	codesign --force --deep --sign - "$app"
	codesign --verify --deep --strict "$app"

	local stage
	stage="$(mktemp -d)"
	cp -R "$app" "$stage/"
	ln -s /Applications "$stage/Applications"
	rm -f "$out"
	hdiutil create -volname "$TITLE $VERSION" -srcfolder "$stage" -ov -format UDZO "$out" >/dev/null
	rm -rf "$stage"
	hdiutil verify "$out" >/dev/null
	echo "✔ $out"
}

# ---- Windows: NSIS installer + portable zip -----------------------------------

package_windows() {
	local setup="$DIST/$NAME-windows-x86_64-setup.exe" portable="$DIST/$NAME-windows-x86_64-portable.zip"
	if [ "$SKIP_BUILD" -eq 0 ]; then
		command -v makensis >/dev/null 2>&1 || die "makensis not found (choco install nsis)"
		# The trash library is cgo, so the Windows build needs a C compiler (mingw-w64).
		CGO_ENABLED=1 "$WAILS_BIN" build -platform windows/amd64 -ldflags "$LDFLAGS" -m -nsis -clean
	fi
	[ -f "build/bin/$APP.exe" ] || die "build/bin/$APP.exe missing — the build failed"
	[ -f "build/bin/$APP-amd64-installer.exe" ] || die "NSIS installer missing — is makensis installed?"

	cp "build/bin/$APP-amd64-installer.exe" "$setup"
	local stage
	stage="$(mktemp -d)"
	mkdir "$stage/$NAME"
	cp "build/bin/$APP.exe" "$stage/$NAME/"
	cp LICENSE "$stage/$NAME/"
	zip_dir "$stage/$NAME" "$portable"
	rm -rf "$stage"
	echo "✔ $setup"
	echo "✔ $portable"
}

# ---- Linux: .deb + tar.gz -------------------------------------------------------

package_linux() {
	local deb="$DIST/$NAME-linux-x86_64.deb" tgz="$DIST/$NAME-linux-x86_64.tar.gz"
	if [ "$SKIP_BUILD" -eq 0 ]; then
		# webkit2_41 links libwebkit2gtk-4.1, which every supported Ubuntu/Debian ships
		# (4.0 was dropped from Ubuntu 24.04).
		"$WAILS_BIN" build -platform linux/amd64 -tags webkit2_41 -ldflags "$LDFLAGS" -m -clean
	fi
	[ -f "build/bin/$APP" ] || die "build/bin/$APP missing — the build failed"

	local stage
	stage="$(mktemp -d)"

	# tar.gz: the bare binary plus licence, for distros without dpkg.
	mkdir "$stage/$NAME-linux-x86_64"
	cp "build/bin/$APP" LICENSE "$stage/$NAME-linux-x86_64/"
	tar -czf "$tgz" -C "$stage" "$NAME-linux-x86_64"

	# .deb
	command -v dpkg-deb >/dev/null 2>&1 || { echo "⚠ dpkg-deb not found — skipped the .deb (tar.gz only)"; rm -rf "$stage"; echo "✔ $tgz"; return; }
	local root="$stage/deb"
	install -Dm755 "build/bin/$APP" "$root/usr/bin/$APP"
	install -Dm644 build/appicon.png "$root/usr/share/pixmaps/$APP.png"
	install -Dm644 /dev/stdin "$root/usr/share/applications/$APP.desktop" <<DESKTOP
[Desktop Entry]
Type=Application
Name=$TITLE
Comment=Disk cleaner for developers
Exec=/usr/bin/$APP
Icon=$APP
Terminal=false
Categories=Utility;System;
DESKTOP
	install -Dm644 LICENSE "$root/usr/share/doc/$APP/copyright"
	install -d "$root/DEBIAN"
	cat > "$root/DEBIAN/control" <<CONTROL
Package: $APP
Version: $VERSION
Section: utils
Priority: optional
Architecture: amd64
Depends: libgtk-3-0, libwebkit2gtk-4.1-0
Installed-Size: $(du -sk "$root/usr" | cut -f1)
Maintainer: Gorby Permana <gorbypermana@gmail.com>
Homepage: https://github.com/$REPO
Description: Disk cleaner for developers
 $TITLE finds build caches and dependency folders (node_modules, target,
 DerivedData, ...) and removes them to the trash or permanently.
CONTROL
	rm -f "$deb"
	dpkg-deb --root-owner-group --build "$root" "$deb" >/dev/null
	rm -rf "$stage"
	echo "✔ $deb"
	echo "✔ $tgz"
}

"package_$OS"

echo "▶ Artifacts in $DIST:"
ls -lh "$DIST" | grep "v$VERSION" || true
