# =============================================================================
# BersihDisk — Makefile
# Run / build / test / release / version management for the Wails app.
#
#   make help            — list all targets
#   make run             — build & launch the app (current OS)
#   make dev             — live-reload development mode
#   make build           — production build for the current OS
#   make build-all       — cross-compile for macOS / Windows / Linux
#   make test            — Go unit tests + frontend typecheck
#   make bump patch|minor|major — bump the version (or: make bump VER=1.2.3)
#   make package         — build + package THIS OS into dist/ (dmg / exe+zip / deb+tar.gz)
#   make release         — local dry run: verify + package this OS (publishes nothing)
#   make changelog       — write the CHANGELOG entry and commit "chore: release X"
#   make publish         — release commit + tag v<VERSION> + push; GitHub Actions
#                          builds every OS and creates the release
#
# Releasing: make bump patch → make publish. `publish` writes the CHANGELOG entry,
# commits "chore: release <version>", tags, and pushes the branch and the tag.
# CI (.github/workflows/release.yml) runs scripts/package.sh on macOS, Windows and
# Linux — the same script as `make package` — then scripts/publish.sh.
# =============================================================================

# ---- Configuration ----------------------------------------------------------
# Recipes are POSIX scripts; without this, make inherits SHELL=/bin/zsh from the
# environment and zsh aborts on any glob that matches nothing (see publish).
SHELL         := /bin/sh

APP_NAME      := bersihdisk
APP_TITLE     := BersihDisk
VERSION_FILE  := VERSION
GO            := go
NPM           := npm
GH            := gh
GIT_PUSH      := git push

# Wails CLI: the published v2.12.0 binary bundles an old golang.org/x/tools and
# aborts on Go >= 1.24 ("internal error: package ... without types"), so prefer
# the locally rebuilt CLI from `make install-wails-fix`; fall back to system wails.
# Override with WAILS=<path> on the command line.
WAILS_HOME    := $(HOME)/.local/bin/wails-fixed
WAILS         ?= $(shell test -x $(WAILS_HOME) && echo $(WAILS_HOME) || command -v wails)

# Version: from VERSION file (or override with VER=x.y.z on the command line).
VERSION       := $(shell cat $(VERSION_FILE) 2>/dev/null || echo 0.0.0)

# Injected into package main (see main.go): the running version, and the GitHub
# "owner/name" whose latest release the in-app updater checks.
REPO          ?= cybersafetyid/BersihDisk
LDFLAGS       := -X main.appVersion=$(VERSION) -X main.releaseRepo=$(REPO)

# Linux links libwebkit2gtk-4.1 (4.0 is gone from Ubuntu 24.04); scripts/package.sh
# uses the same tag, so local and CI builds link the same library.
ifeq ($(shell uname -s),Linux)
GO_TAGS       := -tags webkit2_41
endif

# Release artifact directory.
DIST_DIR      := dist

# Changelog: entries are generated from Conventional Commit subjects.
CHANGELOG     := CHANGELOG.md

# When this make run started: only a Release workflow run created after it is watched.
PUBLISH_START := $(shell date -u +%s)
CHANGELOG_SH  := scripts/changelog.sh

# macOS build targets.
MAC_ARCH      := $(shell uname -m)

# Brand artwork: the logo is the single source, every icon is derived from it.
LOGO_SVG      := assets/logo.svg
LOGO_PNG      := assets/logo.png
BUNDLE_ICON   := build/appicon.png
ICON_VARIANTS := frontend/src/assets/icons

.DEFAULT_GOAL := help
.PHONY: help run dev build package verify build-darwin build-windows build-linux build-all \
        frontend frontend-dev test test-go test-frontend vet lint fmt tidy \
        check clean distclean bump patch minor major release release-version \
        changelog changelog-preview tag release-notes publish publish-local \
        install-deps install-wails-fix doctor icons

# =============================================================================
# Run & Development
# =============================================================================

## run: build and launch the desktop app for the current OS
run: build
	@echo "▶ Launching $(APP_TITLE) v$(VERSION)..."
	@if [ "$(shell uname)" = "Darwin" ]; then \
		open build/bin/$(APP_TITLE).app; \
	else \
		./build/bin/$(APP_TITLE); \
	fi

## dev: live-reload development (wails dev)
dev:
	@echo "▶ Starting dev mode (hot reload, v$(VERSION))..."
	$(WAILS) dev $(GO_TAGS) -ldflags "$(LDFLAGS)"

## frontend-dev: run only the Vite dev server (browser, no Wails shell)
frontend-dev:
	cd frontend && $(NPM) run dev

# =============================================================================
# Build
# =============================================================================

## build: production build for the current OS
build:
	@echo "▶ Building $(APP_NAME) v$(VERSION) for current OS..."
	$(WAILS) build $(GO_TAGS) -ldflags "$(LDFLAGS)" -m
	@echo "✔ Built: build/bin/"

## build-darwin: macOS universal binary (arm64 + x86_64), as shipped
build-darwin:
	@echo "▶ Building macOS universal..."
	$(WAILS) build -platform darwin/universal -ldflags "$(LDFLAGS)" -m
	@echo "✔ Built: build/bin/$(APP_TITLE).app"

## build-windows: Windows amd64 (requires CGO + mingw-w64 for the trash lib)
build-windows:
	@echo "▶ Building Windows amd64..."
	GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc \
	$(WAILS) build -platform windows/amd64 -ldflags "$(LDFLAGS)" -m -nsis
	@echo "✔ Built: build/bin/ (exe + installer)"

## build-linux: Linux amd64 (requires libgtk-3-dev libwebkit2gtk-4.1-dev)
build-linux:
	@echo "▶ Building Linux amd64..."
	$(WAILS) build -platform linux/amd64 -tags webkit2_41 -ldflags "$(LDFLAGS)" -m
	@echo "✔ Built: build/bin/"

## build-all: build every platform (cross toolchains required — CI packages each OS natively instead)
build-all: build-darwin build-windows build-linux

## frontend: build only the frontend bundle
frontend:
	cd frontend && $(NPM) run build

## frontend-dev-install: (re)install frontend dependencies
install-deps:
	cd frontend && $(NPM) install

## icons: derive the edge-to-edge bundle icon and the in-app icon variants from the brand logo
icons:
	@echo "▶ Generating icons from $(LOGO_PNG)..."
	@test -f $(LOGO_PNG) || { echo "✘ $(LOGO_PNG) is missing"; exit 1; }
	$(GO) run ./tools/iconvars -src $(LOGO_PNG) -out $(ICON_VARIANTS) -bundle $(BUNDLE_ICON)
	@echo "✔ Next: make build (or make run) to ship the new icon"

# =============================================================================
# Quality: test / vet / fmt / lint
# =============================================================================

## test: Go tests + frontend typecheck
test: test-go test-frontend

## test-go: run Go unit tests with -race
test-go:
	@echo "▶ Go tests..."
	$(GO) test -race $(GO_TAGS) ./internal/... ./tools/...

## test-frontend: TypeScript typecheck
test-frontend:
	@echo "▶ Frontend typecheck..."
	cd frontend && npx tsc --noEmit

# main.go embeds frontend/dist, so vet cannot even compile a fresh checkout without it.
frontend/dist/index.html:
	cd frontend && $(NPM) install && $(NPM) run build

## vet: go vet on all packages (builds the frontend first when frontend/dist is missing)
vet: frontend/dist/index.html
	$(GO) vet $(GO_TAGS) ./...

## fmt: format all Go files
fmt:
	$(GO) fmt ./...

## lint: vet + typecheck + build check (fast CI gate)
lint: vet test-frontend frontend
	@echo "✔ Lint passed"

## verify: the CI gate without a build (gofmt, vet, Go tests, frontend typecheck)
verify:
	@echo "▶ Checking gofmt..."
	@test -z "$$(gofmt -l . 2>/dev/null | grep -v node_modules)" || \
		{ echo "✘ Not gofmt-ed:"; gofmt -l . | grep -v node_modules; exit 1; }
	$(MAKE) --no-print-directory vet test

## check: full pre-commit verification (verify + build)
check: verify build

## tidy: go mod tidy
tidy:
	$(GO) mod tidy

# =============================================================================
# Version management
# =============================================================================

# `make bump patch|minor|major` bumps by level; `make bump VER=1.2.3` sets it exactly.
# The level word after `bump` is an argument, not a second goal, so it is computed
# into VER here and the matching aliases below are replaced by a no-op.
BUMP_ARG   := $(if $(filter bump,$(firstword $(MAKECMDGOALS))),$(word 2,$(MAKECMDGOALS)))
BUMP_LEVEL := $(filter patch minor major,$(BUMP_ARG))
ifneq ($(BUMP_ARG),)
ifeq ($(BUMP_LEVEL),)
$(error Unknown bump level '$(BUMP_ARG)' — use: make bump patch|minor|major, or make bump VER=1.2.3)
endif
VER ?= $(shell awk -F. -v level=$(BUMP_LEVEL) 'BEGIN { OFS = "." } \
	level == "major" { print $$1 + 1, 0, 0 } \
	level == "minor" { print $$1, $$2 + 1, 0 } \
	level == "patch" { print $$1, $$2, $$3 + 1 }' $(VERSION_FILE))
endif

## bump: set the version — make bump patch|minor|major, or make bump VER=1.2.3 (VERSION, wails.json, package.json)
bump:
ifndef VER
	$(error Usage: make bump patch|minor|major   or   make bump VER=1.2.3)
endif
	@echo "▶ Bumping version $(VERSION) -> $(VER)"
	printf '%s\n' '$(VER)' > $(VERSION_FILE)
	@# wails.json Info.productVersion
	@$(NPM) --prefix frontend exec --yes json@2 \
		-I wails.json -e 'this.Info.productVersion="$(VER)"' >/dev/null 2>&1 || \
		sed -i.bak 's/"productVersion": *"[^"]*"/"productVersion": "$(VER)"/' wails.json
	@rm -f wails.json.bak
	@# frontend package.json version (informational)
	@cd frontend && $(NPM) version $(VER) --no-git-tag-version --allow-same-version >/dev/null
	@echo "✔ Version is now $(VERSION_FILE)=`cat $(VERSION_FILE)`"

ifeq ($(BUMP_LEVEL),)
## patch: bump PATCH digit (1.2.3 -> 1.2.4); same as make bump patch
patch:
	$(MAKE) bump patch

## minor: bump MINOR digit, reset patch (1.2.3 -> 1.3.0); same as make bump minor
minor:
	$(MAKE) bump minor

## major: bump MAJOR digit, reset minor+patch (1.2.3 -> 2.0.0); same as make bump major
major:
	$(MAKE) bump major
else
# `make bump patch`: the word `patch` is consumed above, so it must do nothing.
$(BUMP_LEVEL):
	@:
endif

# =============================================================================
# Release & packaging
# =============================================================================

## package: build + package THIS OS into dist/ (macOS .dmg, Windows setup .exe + portable .zip, Linux .deb + .tar.gz)
package:
	WAILS="$(WAILS)" VERSION="$(VERSION)" REPO="$(REPO)" DIST="$(DIST)" bash scripts/package.sh

## release: local dry run — verify + package THIS OS, exactly what CI builds (publishes nothing)
release: verify package

## changelog: write the CHANGELOG entry for VERSION, then commit everything as "chore: release <VERSION>" (COMMIT=0 to skip the commit)
changelog:
	@$(CHANGELOG_SH) --write $(VERSION)
	@if [ "$(COMMIT)" = "0" ]; then \
		echo "ℹ COMMIT=0 — leaving the changes uncommitted"; \
	elif [ -z "$$(git status --porcelain --untracked-files=no)" ]; then \
		echo "✔ Nothing to commit — v$(VERSION) is already committed"; \
	else \
		echo "▶ git commit -am \"chore: release $(VERSION)\":"; \
		git status --short --untracked-files=no; \
		git commit -q -am "chore: release $(VERSION)" && echo "✔ Committed chore: release $(VERSION)"; \
	fi

## changelog-preview: print that entry to stdout without writing the file
changelog-preview:
	@$(CHANGELOG_SH) $(VERSION)

## tag: create annotated git tag v$(VERSION) on HEAD (skipped when it exists)
tag:
	@if git rev-parse -q --verify "refs/tags/v$(VERSION)" >/dev/null; then \
		echo "✔ v$(VERSION) already tagged"; \
	else \
		git tag -a "v$(VERSION)" -m "$(APP_TITLE) $(VERSION)" && \
		echo "✔ Tagged v$(VERSION)"; \
	fi

## release-notes: write dist/ release notes markdown from the CHANGELOG entry
release-notes:
	@mkdir -p $(DIST_DIR)
	@{ printf '# $(APP_TITLE) v%s\n\n' "$(VERSION)"; $(CHANGELOG_SH) --notes $(VERSION); } \
		> $(DIST_DIR)/$(APP_TITLE)-v$(VERSION)-notes.md
	@echo "✔ Notes: $(DIST_DIR)/$(APP_TITLE)-v$(VERSION)-notes.md"

## publish: release commit, tag v$(VERSION), push branch + tag — GitHub Actions then builds macOS/Windows/Linux and creates the release (WATCH=0 to not wait)
publish: verify
	@command -v $(GH) >/dev/null 2>&1 || { echo "✘ $(GH) not found (brew install gh && gh auth login)"; exit 1; }
	@# The release commit: a missing CHANGELOG entry, or uncommitted changes such as the
	@# version bump, are written and committed as "chore: release $(VERSION)" first.
	@if ! grep -q '^## \[$(VERSION)\]' $(CHANGELOG) || \
		{ [ -n "$$(git status --porcelain --untracked-files=no)" ] && [ -z "$(ALLOW_DIRTY)" ]; }; then \
		$(MAKE) --no-print-directory changelog; \
	fi
	@if [ -n "$$(git status --porcelain)" ] && [ -z "$(ALLOW_DIRTY)" ]; then \
		echo "✘ Untracked files would not be part of v$(VERSION):"; git status --short; \
		echo "  Add or remove them, or re-run with ALLOW_DIRTY=1."; exit 1; \
	fi
	@# A tag that was never pushed and lags HEAD (made by an earlier run, before the last
	@# commits) would release stale code. Retag it, or stop.
	@if git rev-parse -q --verify "refs/tags/v$(VERSION)" >/dev/null && \
		[ -z "$$(git ls-remote origin -t "refs/tags/v$(VERSION)" | cut -f1)" ] && \
		[ "$$(git rev-list -n1 v$(VERSION))" != "$$(git rev-parse HEAD)" ]; then \
		if [ "$(RETAG)" = "1" ]; then \
			echo "▶ Moving unpushed tag v$(VERSION) to HEAD"; git tag -d "v$(VERSION)" >/dev/null; \
		else \
			echo "✘ Local tag v$(VERSION) is behind HEAD and was never pushed, so it would release older code."; \
			echo "  Move it with: make publish RETAG=1   (or: git tag -d v$(VERSION))"; exit 1; \
		fi; \
	fi
	@$(MAKE) --no-print-directory tag
	@branch=$$(git rev-parse --abbrev-ref HEAD); \
	tag_commit=$$(git rev-list -n1 v$(VERSION)); \
	if ! git merge-base --is-ancestor "$$tag_commit" HEAD; then \
		echo "✘ v$(VERSION) points at $$tag_commit, which is not part of $$branch."; \
		echo "  Delete it (git tag -d v$(VERSION)) and run 'make tag' on the commit to release."; \
		exit 1; \
	fi; \
	remote_head=$$(git ls-remote origin -h "refs/heads/$$branch" | cut -f1); \
	if [ -z "$$remote_head" ] || ! git merge-base --is-ancestor "$$tag_commit" "$$remote_head"; then \
		echo "▶ origin/$$branch does not contain the tagged commit — pushing $$branch"; \
		$(GIT_PUSH) origin "$$branch"; \
	fi; \
	if [ "$$(git ls-remote origin -t "refs/tags/v$(VERSION)" | cut -f1)" = "$$tag_commit" ]; then \
		echo "▶ v$(VERSION) is already on origin — re-running the release workflow"; \
		$(GH) workflow run release.yml -R $(REPO) -f tag=v$(VERSION) -f ref=v$(VERSION); \
	else \
		echo "▶ Pushing annotated tag v$(VERSION) — this starts the Release workflow"; \
		$(GIT_PUSH) origin "refs/tags/v$(VERSION)"; \
	fi
	@if [ "$(WATCH)" != "0" ]; then \
		echo "▶ Waiting for the Release workflow (Ctrl-C stops watching, not the build)..."; \
		id=""; for i in 1 2 3 4 5 6 7 8 9 10; do \
			id=$$($(GH) run list -R $(REPO) --workflow release.yml --limit 5 --json databaseId,createdAt \
				-q '[.[] | select((.createdAt | fromdateiso8601) >= $(PUBLISH_START) - 120)] | sort_by(.createdAt) | reverse | .[0].databaseId // empty' 2>/dev/null); \
			[ -n "$$id" ] && break; sleep 3; \
		done; \
		[ -n "$$id" ] && $(GH) run watch "$$id" -R $(REPO) --exit-status; \
	fi
	@echo "✔ https://github.com/$(REPO)/releases/tag/v$(VERSION)"

## publish-local: upload the artifacts built on THIS machine (dist/) to the release — an escape hatch, CI is the normal path
publish-local: release-notes
	@VERSION="$(VERSION)" REPO="$(REPO)" DIST="$(DIST_DIR)" GH="$(GH)" bash scripts/publish.sh $(PUBLISH_FLAGS)

## release-version: bump to VER=x.y.z, write the CHANGELOG entry and commit it (then make publish)
release-version:
	@$(MAKE) --no-print-directory bump VER=$(VER)
	@$(MAKE) --no-print-directory changelog
	@echo "✔ $(APP_TITLE) $$(cat $(VERSION_FILE)) is committed. Next: make publish"

# =============================================================================
# Maintenance
# =============================================================================

## clean: remove build output and frontend dist
clean:
	@echo "▶ Cleaning..."
	rm -rf build/bin $(DIST_DIR) frontend/dist frontend/package.json.md5
	@echo "✔ Cleaned"

## distclean: clean + node_modules + Go test cache
distclean: clean
	rm -rf frontend/node_modules
	$(GO) clean -testcache
	@echo "✔ Deep cleaned"

## doctor: verify the toolchain (go, node, wails, cgo deps)
doctor:
	@echo "== Toolchain =="
	@$(GO) version || echo "✘ Go missing"
	@node --version || echo "✘ Node missing"
	@$(NPM) --version || echo "✘ npm missing"
	@echo "Wails CLI : $(WAILS)"
	@$(WAILS) version 2>/dev/null | head -1
	@test "$(WAILS)" = "$(WAILS_HOME)" || \
		echo "⚠ Using system wails; it fails on Go >= 1.24 — run: make install-wails-fix"
	@uname -sm

## install-wails-fix: rebuild the Go>=1.24-compatible Wails CLI into $(HOME)/.local/bin
install-wails-fix:
	@echo "▶ Building patched wails CLI (x/tools latest)..."
	@dir=$$(mktemp -d) && cd $$dir && \
	printf 'module wclifix\n\ngo 1.23.0\n' > go.mod && \
	$(GO) get github.com/wailsapp/wails/v2/cmd/wails@v2.12.0 && \
	$(GO) get golang.org/x/tools@latest && \
	mkdir -p $$(dirname $(WAILS_HOME)) && \
	$(GO) build -o $(WAILS_HOME) github.com/wailsapp/wails/v2/cmd/wails && \
	rm -rf $$dir
	@echo "✔ Installed: $(WAILS_HOME)"

# =============================================================================
# Help
# =============================================================================

help:
	@echo "$(APP_TITLE) v$(VERSION)"
	@echo ""
	@echo "Usage: make [target] [VARS...]"
	@echo ""
	@grep -E '^## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ": "}; \
		{printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}' | \
		sed 's/## //'
