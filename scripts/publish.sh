#!/usr/bin/env bash
# publish.sh — attach the artifacts in dist/ to the GitHub release for VERSION.
#
# Shared by `make publish` and CI (.github/workflows/release.yml). The release is
# created when it does not exist and updated in place when it does, so re-running
# it after fixing a build replaces the assets instead of failing.
#
#   scripts/publish.sh                 create the release, or upload with --clobber
#   scripts/publish.sh --prune         also delete assets that are not in this upload
#                                      (only when all three platforms are present)
#   scripts/publish.sh --update-notes  also replace the release notes
#
# Env: VERSION (default: VERSION file)  REPO  TAG (default: v$VERSION)  DIST  GH

set -euo pipefail
cd "$(dirname "$0")/.."

TITLE=BersihDisk
VERSION="${VERSION:-$(tr -d '[:space:]' < VERSION)}"
REPO="${REPO:-cybersafetyid/BersihDisk}"
TAG="${TAG:-v$VERSION}"
DIST="${DIST:-dist}"
GH="${GH:-gh}"
NAME="$TITLE-v$VERSION"

PRUNE=0
UPDATE_NOTES=0
for arg in "$@"; do
	case "$arg" in
		--prune) PRUNE=1 ;;
		--update-notes) UPDATE_NOTES=1 ;;
		*) echo "usage: publish.sh [--prune] [--update-notes]" >&2; exit 2 ;;
	esac
done

die() { echo "✘ $*" >&2; exit 1; }
command -v "$GH" >/dev/null 2>&1 || die "$GH not found (brew install gh && gh auth login)"

# Artifacts: only the formats the updater and the README promise.
artifacts=()
for f in "$DIST/$NAME"-*.dmg "$DIST/$NAME"-*.exe "$DIST/$NAME"-*.zip "$DIST/$NAME"-*.deb "$DIST/$NAME"-*.tar.gz; do
	[ -f "$f" ] && artifacts+=("$f")
done
[ "${#artifacts[@]}" -gt 0 ] || die "no $NAME artifacts in $DIST/ — run 'make package' (or the release workflow) first"

# Checksums for humans; GitHub also exposes a sha256 digest per asset, which is
# what the in-app updater verifies.
sums="$DIST/$NAME-SHA256SUMS.txt"
: > "$sums"
for f in "${artifacts[@]}"; do
	if command -v sha256sum >/dev/null 2>&1; then
		(cd "$DIST" && sha256sum "$(basename "$f")") >> "$sums"
	else
		(cd "$DIST" && shasum -a 256 "$(basename "$f")") >> "$sums"
	fi
done
upload=("${artifacts[@]}" "$sums")

# Release notes: the curated CHANGELOG section for this version.
notes="$DIST/$NAME-notes.md"
if [ ! -f "$notes" ]; then
	{ printf '# BersihDisk v%s\n\n' "$VERSION"; sh scripts/changelog.sh --notes "$VERSION"; } > "$notes"
fi

echo "▶ $TAG → $REPO: ${#upload[@]} file(s)"
printf '   %s\n' "${upload[@]##*/}"

if "$GH" release view "$TAG" -R "$REPO" >/dev/null 2>&1; then
	"$GH" release upload "$TAG" -R "$REPO" --clobber "${upload[@]}"
	if [ "$UPDATE_NOTES" -eq 1 ]; then
		"$GH" release edit "$TAG" -R "$REPO" --title "BersihDisk v$VERSION" --notes-file "$notes"
	fi
else
	"$GH" release create "$TAG" -R "$REPO" --verify-tag \
		--title "BersihDisk v$VERSION" --notes-file "$notes" "${upload[@]}"
fi

if [ "$PRUNE" -eq 1 ]; then
	# Refuse to prune from a partial upload: that would delete the other platforms.
	have() { printf '%s\n' "${upload[@]##*/}" | grep -Eq "$1"; }
	have 'macos.*\.dmg$' && have 'windows.*\.exe$' && have 'linux.*\.deb$' ||
		die "--prune needs the macOS, Windows and Linux artifacts; nothing was deleted"
	keep="$(printf '%s\n' "${upload[@]##*/}")"
	"$GH" release view "$TAG" -R "$REPO" --json assets -q '.assets[].name' | while IFS= read -r asset; do
		if ! printf '%s\n' "$keep" | grep -Fxq "$asset"; then
			echo "   pruning stale asset $asset"
			"$GH" release delete-asset "$TAG" "$asset" -R "$REPO" --yes
		fi
	done
fi

echo "✔ https://github.com/$REPO/releases/tag/$TAG"
