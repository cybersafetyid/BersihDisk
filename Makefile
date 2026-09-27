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
#   make bump VER=1.2.3  — set version in VERSION, wails.json, package.json
#   make release         — bump, build, package artifacts into dist/ + tag
#   make changelog       — prepend the CHANGELOG entry built from git commits
#   make publish         — create the GitHub release with artifacts + notes
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

# Release artifact directory.
DIST_DIR      := dist

# Changelog: entries are generated from Conventional Commit subjects.
CHANGELOG     := CHANGELOG.md
CHANGELOG_SH  := scripts/changelog.sh

# macOS build targets.
MAC_ARCH      := $(shell uname -m)

# Brand artwork: the logo is the single source, every icon is derived from it.
LOGO_SVG      := assets/logo.svg
LOGO_PNG      := assets/logo.png
BUNDLE_ICON   := build/appicon.png
ICON_VARIANTS := frontend/src/assets/icons

.DEFAULT_GOAL := help
.PHONY: help run dev build build-darwin build-windows build-linux build-all \
        frontend frontend-dev test test-go test-frontend vet lint fmt tidy \
        check clean distclean bump patch minor major release release-version \
        changelog changelog-preview tag release-notes publish \
        install-deps install-wails-fix doctor icons

# =============================================================================
# Run & Development
# =============================================================================

## run: build and launch the desktop app for the current OS
run: build
	@echo "▶ Launching $(APP_TITLE) v$(VERSION)..."
	@if [ "$(shell uname)" = "Darwin" ]; then \
		open build/bin/$(APP_NAME).app; \
	else \
		./build/bin/$(APP_NAME); \
	fi

## dev: live-reload development (wails dev)
dev:
	@echo "▶ Starting dev mode (hot reload, v$(VERSION))..."
	$(WAILS) dev -ldflags "$(LDFLAGS)"

## frontend-dev: run only the Vite dev server (browser, no Wails shell)
frontend-dev:
	cd frontend && $(NPM) run dev

# =============================================================================
# Build
# =============================================================================

## build: production build for the current OS
build:
	@echo "▶ Building $(APP_NAME) v$(VERSION) for current OS..."
	$(WAILS) build -ldflags "$(LDFLAGS)" -m
	@echo "✔ Built: build/bin/"

## build-darwin: macOS (arm64 + amd64 universal on Apple Silicon)
build-darwin:
	@echo "▶ Building macOS..."
	$(WAILS) build -platform darwin/$(MAC_ARCH) -ldflags "$(LDFLAGS)" -m
	@echo "✔ Built: build/bin/$(APP_NAME).app"

## build-windows: Windows amd64 (requires CGO + mingw-w64 for the trash lib)
build-windows:
	@echo "▶ Building Windows amd64..."
	GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc \
	$(WAILS) build -platform windows/amd64 -ldflags "$(LDFLAGS)" -m -nsis
	@echo "✔ Built: build/bin/ (exe + installer)"

## build-linux: Linux amd64 (requires GTK/webkit dev headers)
build-linux:
	@echo "▶ Building Linux amd64..."
	$(WAILS) build -platform linux/amd64 -ldflags "$(LDFLAGS)" -m
	@echo "✔ Built: build/bin/"

## build-all: build for every supported platform (native tools required)
build-all: build-darwin build-windows build-linux

## frontend: build only the frontend bundle
frontend:
	cd frontend && $(NPM) run build

## frontend-dev-install: (re)install frontend dependencies
install-deps:
	cd frontend && $(NPM) install

## icons: derive the bundle icon and the in-app icon variants from the brand logo
icons:
	@echo "▶ Generating icons from $(LOGO_PNG)..."
	@test -f $(LOGO_PNG) || { echo "✘ $(LOGO_PNG) is missing"; exit 1; }
	@tmp=$$(mktemp -d); \
	if command -v qlmanage >/dev/null 2>&1 && qlmanage -t -s 1024 -o $$tmp $(LOGO_SVG) >/dev/null 2>&1 \
		&& [ -f $$tmp/$(notdir $(LOGO_SVG)).png ]; then \
			mv $$tmp/$(notdir $(LOGO_SVG)).png $(BUNDLE_ICON); \
			echo "✔ $(BUNDLE_ICON) rendered at 1024 from $(LOGO_SVG)"; \
		else \
			cp $(LOGO_PNG) $(BUNDLE_ICON); \
			echo "✔ $(BUNDLE_ICON) copied from $(LOGO_PNG) (no SVG renderer here)"; \
		fi; \
	rm -rf $$tmp
	$(GO) run ./tools/iconvars -src $(LOGO_PNG) -out $(ICON_VARIANTS)
	@echo "✔ Next: make build (or make run) to ship the new icon"

# =============================================================================
# Quality: test / vet / fmt / lint
# =============================================================================

## test: Go tests + frontend typecheck
test: test-go test-frontend

## test-go: run Go unit tests with -race
test-go:
	@echo "▶ Go tests..."
	$(GO) test -race ./internal/... ./tools/...

## test-frontend: TypeScript typecheck
test-frontend:
	@echo "▶ Frontend typecheck..."
	cd frontend && npx tsc --noEmit

## vet: go vet on all packages
vet:
	$(GO) vet ./...

## fmt: format all Go files
fmt:
	$(GO) fmt ./...

## lint: vet + typecheck + build check (fast CI gate)
lint: vet test-frontend frontend
	@echo "✔ Lint passed"

## check: full pre-commit verification (fmt check, vet, tests, build)
check:
	@echo "▶ Checking gofmt..."
	@test -z "$$(gofmt -l . 2>/dev/null | grep -v node_modules)" || \
		{ echo "✘ Not gofmt-ed:"; gofmt -l . | grep -v node_modules; exit 1; }
	$(MAKE) vet test build

## tidy: go mod tidy
tidy:
	$(GO) mod tidy

# =============================================================================
# Version management
# =============================================================================

## bump: set version to VER=x.y.z in VERSION, wails.json, and app.go
bump:
ifndef VER
	$(error Usage: make bump VER=1.2.3)
endif
	@echo "▶ Bumping version -> $(VER)"
	printf '%s\n' '$(VER)' > $(VERSION_FILE)
	@# wails.json Info.productVersion
	@$(NPM) --prefix frontend exec --yes json@2 \
		-I wails.json -e 'this.Info.productVersion="$(VER)"' >/dev/null 2>&1 || \
		sed -i.bak 's/"productVersion": *"[^"]*"/"productVersion": "$(VER)"/' wails.json
	@rm -f wails.json.bak
	@# frontend package.json version (informational)
	@cd frontend && $(NPM) version $(VER) --no-git-tag-version --allow-same-version >/dev/null
	@echo "✔ Version is now $(VERSION_FILE)=`cat $(VERSION_FILE)`"

## patch: bump PATCH digit (1.2.3 -> 1.2.4)
patch:
	$(MAKE) bump VER=$$(awk -F. '{printf "%d.%d.%d", $$1, $$2, $$3+1}' $(VERSION_FILE))

## minor: bump MINOR digit, reset patch (1.2.3 -> 1.3.0)
minor:
	$(MAKE) bump VER=$$(awk -F. '{printf "%d.%d.0", $$1, $$2+1}' $(VERSION_FILE))

## major: bump MAJOR digit, reset minor+patch (1.2.3 -> 2.0.0)
major:
	$(MAKE) bump VER=$$(awk -F. '{printf "%d.0.0", $$1+1}' $(VERSION_FILE))

# =============================================================================
# Release & packaging
# =============================================================================

## release: build + package artifacts, then write the CHANGELOG entry and tag
release: check build
	@echo "▶ Packaging release v$(VERSION)..."
	@mkdir -p $(DIST_DIR)/$(APP_NAME)-v$(VERSION)
	@if [ -d build/bin/$(APP_NAME).app ]; then \
		cp -R build/bin/$(APP_NAME).app $(DIST_DIR)/$(APP_NAME)-v$(VERSION)/; \
		cd $(DIST_DIR) && zip -qry \
			$(APP_NAME)-v$(VERSION)-macos-$(MAC_ARCH).zip $(APP_NAME)-v$(VERSION)/$(APP_NAME).app; \
	elif [ -f build/bin/$(APP_NAME).exe ]; then \
		cp build/bin/$(APP_NAME).exe $(DIST_DIR)/$(APP_NAME)-v$(VERSION)/; \
		cd $(DIST_DIR) && zip -qry \
			$(APP_NAME)-v$(VERSION)-windows-amd64.zip $(APP_NAME)-v$(VERSION)/; \
	else \
		cp -R build/bin/$(APP_NAME) $(DIST_DIR)/$(APP_NAME)-v$(VERSION)/; \
		cd $(DIST_DIR) && tar czf \
			$(APP_NAME)-v$(VERSION)-linux-amd64.tar.gz $(APP_NAME)-v$(VERSION)/; \
	fi
	@printf '%s v%s (%s)\nBuilt: %s\n' \
		"$(APP_TITLE)" "$(VERSION)" "$$(date -u +%Y-%m-%d)" \
		"$$(ls $(DIST_DIR)/*v$(VERSION)*.zip $(DIST_DIR)/*v$(VERSION)*.tar.gz 2>/dev/null)" \
		> $(DIST_DIR)/$(APP_NAME)-v$(VERSION)-release-notes.txt
	@echo "✔ Release ready:"
	@ls -lh $(DIST_DIR)/ | grep "v$(VERSION)" || true
	@$(MAKE) --no-print-directory release-finalize

## release-finalize: CHANGELOG entry + git tag + GitHub notes, when history exists
release-finalize:
	@if git rev-parse --verify HEAD >/dev/null 2>&1; then \
		$(MAKE) --no-print-directory changelog tag release-notes; \
	else \
		echo "⚠ No git history — skipped CHANGELOG entry and v$(VERSION) tag."; \
		echo "  Fix with: git init && git add -A && git commit -m 'chore: initial commit' && make release"; \
	fi

## changelog: prepend the generated entry for the current version into CHANGELOG.md
changelog:
	@$(CHANGELOG_SH) --write $(VERSION)

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
		> $(DIST_DIR)/$(APP_NAME)-v$(VERSION)-notes.md
	@echo "✔ Notes: $(DIST_DIR)/$(APP_NAME)-v$(VERSION)-notes.md"

## publish: create the GitHub release with dist artifacts + CHANGELOG notes
publish: release-notes
	@command -v $(GH) >/dev/null 2>&1 || { echo "✘ $(GH) not found (brew install gh && gh auth login)"; exit 1; }
	@set --; \
	for f in $(DIST_DIR)/$(APP_NAME)-v$(VERSION)*.zip $(DIST_DIR)/$(APP_NAME)-v$(VERSION)*.tar.gz; do \
		[ -f "$$f" ] && set -- "$$@" "$$f"; \
	done; \
	if [ "$$#" -eq 0 ]; then \
		echo "✘ No v$(VERSION) artifacts in $(DIST_DIR)/ — run 'make release' first"; exit 1; \
	fi; \
	echo "▶ Releasing v$(VERSION) with $$# artifact(s): $$*"; \
	$(GH) release create "v$(VERSION)" -R $(REPO) \
		--title "$(APP_TITLE) v$(VERSION)" \
		--notes-file $(DIST_DIR)/$(APP_NAME)-v$(VERSION)-notes.md "$$@"
	@echo "✔ Published https://github.com/$(REPO)/releases/tag/v$(VERSION)"

## release-version: bump to VER=x.y.z then release (one-shot)
release-version: bump
	$(MAKE) release

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
