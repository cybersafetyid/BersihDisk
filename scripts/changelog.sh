#!/bin/sh
# changelog.sh — build a Keep a Changelog entry from Conventional Commit subjects.
#
#   changelog.sh VERSION                 print the entry for VERSION to stdout
#   changelog.sh --write VERSION         prepend that entry into CHANGELOG.md
#   changelog.sh --notes VERSION         print only the entry body (for gh release)
#   changelog.sh [--write|--notes] VERSION SINCE_REF
#
# SINCE_REF defaults to the newest tag that is not VERSION, so the commit range
# stays correct even when the tag for this version already exists.

set -eu

MODE=print
case "${1:-}" in
	--write) MODE=write; shift ;;
	--notes) MODE=notes; shift ;;
esac

VERSION="${1:-}"
SINCE="${2:-}"

if [ -z "$VERSION" ]; then
	echo "usage: changelog.sh [--write|--notes] VERSION [SINCE_REF]" >&2
	exit 2
fi

command -v git >/dev/null 2>&1 || { echo "git is not installed" >&2; exit 1; }
git rev-parse --is-inside-work-tree >/dev/null 2>&1 || {
	echo "not a git repository — run 'git init' and commit before generating a changelog" >&2
	exit 1
}
git rev-parse --verify HEAD >/dev/null 2>&1 || {
	echo "the repository has no commits yet" >&2
	exit 1
}

if [ -z "$SINCE" ]; then
	SINCE=$(git tag --list 'v[0-9]*' --sort=-v:refname | grep -v "^v${VERSION}\$" | head -1 || true)
fi

if [ -n "$SINCE" ]; then
	RANGE="${SINCE}..HEAD"
else
	RANGE="HEAD"
fi

WORK=$(mktemp -d)
trap 'rm -rf "$WORK"' EXIT INT TERM

ENTRY="$WORK/entry.md"
git log --no-merges --reverse --pretty=format:'%s%x1f%h' "$RANGE" |
	awk -F'\037' -v ver="$VERSION" -v date="$(date -u +%Y-%m-%d)" '
	function clean(s) {
		sub(/^[a-zA-Z][a-zA-Z0-9]*(\([^)]*\))?!?:[ ]*/, "", s)
		gsub(/[ \t]+$/, "", s)
		return s
	}
	{
		subject = $1
		hash = $2
		text = clean(subject)
		breaking = (subject ~ /!:/)

		group = "Other"
		if (subject ~ /^feat/)                                                           group = "Added"
		else if (subject ~ /^fix/)                                                       group = "Fixed"
		else if (subject ~ /^perf|^refactor|^i18n|^build|^ci|^chore|^style|^test|^docs/) group = "Changed"
		else if (subject ~ /^security/)                                                  group = "Security"
		else if (subject ~ /^remove|^rm/)                                                group = "Removed"
		else if (subject ~ /^deprecat/)                                                  group = "Deprecated"

		line = "- " (breaking ? "**Breaking** — " : "") text " (`" hash "`)"
		if (!(line in seen)) { seen[line] = 1; buckets[group] = buckets[group] line "\n" }
	}
	END {
		printf "## [%s] - %s\n", ver, date
		split("Added Changed Deprecated Removed Fixed Security Other", order, " ")
		for (i = 1; i <= 7; i++) {
			g = order[i]
			if (buckets[g] == "") continue
			printf "\n### %s\n\n", g
			printf "%s", buckets[g]
		}
	}' > "$ENTRY"

case "$MODE" in
	notes) tail -n +2 "$ENTRY"; exit 0 ;;
	print) cat "$ENTRY"; exit 0 ;;
esac

CHANGELOG=${CHANGELOG:-CHANGELOG.md}
if [ ! -f "$CHANGELOG" ]; then
	printf '# Changelog\n\nAll notable changes to BersihDisk are documented here.\nThe format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).\n' > "$CHANGELOG"
fi

if grep -q "^## \[${VERSION}\]" "$CHANGELOG"; then
	echo "⚠ v${VERSION} is already listed in ${CHANGELOG} — left untouched" >&2
	exit 0
fi

ENTRY_FILE="$ENTRY" awk '
	BEGIN { while ((getline l < ENVIRON["ENTRY_FILE"]) > 0) entry = entry l "\n" }
	!done && /^## \[/ { printf "%s\n", entry; done = 1 }
	{ print }
	END { if (!done) printf "\n%s", entry }
' "$CHANGELOG" > "$WORK/out"

cp "$WORK/out" "$CHANGELOG"
echo "✔ ${CHANGELOG}: added v${VERSION} ($(grep -c '^- ' "$ENTRY" || true) entries)"
