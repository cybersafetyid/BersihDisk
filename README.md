<div align="center">

<img src="assets/logo.png" alt="BersihDisk logo — a blue disk platter with a bright sweep wiping its surface" width="128" height="128" />

# BersihDisk

**Disk cleaner for developers** · **Pembersih disk untuk developer**

Frees the gigabytes your toolchain leaves behind: `node_modules`, build artifacts, package caches, AI model caches and temp files — with a modern bilingual UI and a delete step you control.

[![release](https://img.shields.io/github/v/release/cybersafetyid/BersihDisk?style=flat-square&color=4f8cff&label=release&logo=github&logoColor=white)](https://github.com/cybersafetyid/BersihDisk/releases)
[![downloads](https://img.shields.io/github/downloads/cybersafetyid/BersihDisk/total?style=flat-square&color=34d399&label=downloads)](https://github.com/cybersafetyid/BersihDisk/releases)
[![last commit](https://img.shields.io/github/last-commit/cybersafetyid/BersihDisk?style=flat-square&color=909090&label=updated)](https://github.com/cybersafetyid/BersihDisk/commits/main)
[![issues](https://img.shields.io/github/issues/cybersafetyid/BersihDisk?style=flat-square&color=fbbf24&label=issues)](https://github.com/cybersafetyid/BersihDisk/issues)
[![stars](https://img.shields.io/github/stars/cybersafetyid/BersihDisk?style=flat-square&color=4f8cff&label=stars&logo=github&logoColor=white)](https://github.com/cybersafetyid/BersihDisk)
[![forks](https://img.shields.io/github/forks/cybersafetyid/BersihDisk?style=flat-square&color=6ba1ff&label=forks)](https://github.com/cybersafetyid/BersihDisk/forks)

[![platform](https://img.shields.io/badge/platform-macOS%20%7C%20Windows%20%7C%20Linux-1c1c1c?style=flat-square)](#-getting-started)
[![go](https://img.shields.io/badge/Go-1.23%2B-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev)
[![wails](https://img.shields.io/badge/Wails-v2.12-B00606?style=flat-square)](https://wails.io)
[![react](https://img.shields.io/badge/React-18-1f2b3a?style=flat-square&logo=react&logoColor=61DAFB)](https://react.dev)
[![typescript](https://img.shields.io/badge/TypeScript-4.6-3178C6?style=flat-square&logo=typescript&logoColor=white)](https://www.typescriptlang.org)
[![vite](https://img.shields.io/badge/Vite-3-646CFF?style=flat-square&logo=vite&logoColor=white)](https://vitejs.dev)

[![license](https://img.shields.io/badge/license-MIT-blue?style=flat-square)](LICENSE)
[![tests](https://img.shields.io/badge/tests-87%20Go%20unit%20tests-34d399?style=flat-square)](#-testing)
[![categories](https://img.shields.io/badge/cleanup%20categories-24-4f8cff?style=flat-square)](#-cleanup-categories)
[![telemetry](https://img.shields.io/badge/telemetry-none-brightgreen?style=flat-square)](#-privacy)
[![languages](https://img.shields.io/badge/UI%20languages-EN%20%7C%20ID-6ba1ff?style=flat-square)](README.id.md)
[![PRs](https://img.shields.io/badge/PRs-welcome-brightgreen?style=flat-square)](CONTRIBUTING.md)
[![contributor covenant](https://img.shields.io/badge/Contributor%20Covenant-2.1-4baaaa?style=flat-square)](CODE_OF_CONDUCT.md)
[![changelog](https://img.shields.io/badge/changelog-Keep%20a%20Changelog-4f8cff?style=flat-square)](CHANGELOG.md)

**English** · **[Bahasa Indonesia](README.id.md)**

[Download](https://github.com/cybersafetyid/BersihDisk/releases) · [Features](#-features) · [Getting started](#-getting-started) · [Architecture](#-architecture) · [Contributing](CONTRIBUTING.md) · [Changelog](#-changelog--releases) · [Sponsor](#-sponsorship--donations)
</div>

---

## About

BersihDisk is a desktop disk cleaner **and uninstaller** aimed at developers. A single machine that
builds software accumulates caches fast — a `node_modules` per project, Go and
Cargo module caches, Gradle and Maven repositories, Docker buildx layers,
HuggingFace weights, Xcode DerivedData — and most of it is regenerable.
BersihDisk finds those directories across every mounted drive, shows you what is
inside them, and deletes only what you ticked. Every result is graded for risk,
so a folder that only *looks* like junk — a global `node_modules`, an app bundle,
the Ruby gems you installed — is flagged (or refused) before it can hurt. The
second half of the app removes whole tools — Node, Rust, Python, Docker, VS Code
and friends — together with the caches, configuration, shell-profile lines, PATH
entries and registry keys they leave behind.

It is a native desktop app (Go + Wails) with a web-tech UI (React + TypeScript),
so scanning is fast and concurrent while the interface stays responsive and
animated. No accounts, no server, no upload.

## ✨ Features

**Detection**
- **Multi-drive scanning** — macOS (`/Volumes`), Windows (A:–Z: via WinAPI), Linux (`/proc/mounts`), each with used/free capacity
- **Two-phase engine** — a *find* phase walking the tree with a system-folder skip-list, then a *measure* phase sizing candidates concurrently, with live progress and cancellation
- **Content-aware matching** — generic `build/`, `target/`, `bin/`, `obj/` folders only count as artifacts when their contents prove it, so source trees are never matched
- **Folder browser** — open any scan result to see its children (sizes computed in parallel, lazily) and pick individual entries; children inside a selected folder are not double-counted

**Safety** — see [Safety](#-safety)
- **Risk grading** — every result is *safe*, *caution*, *danger* or *protected*, with the reason in plain words; risky results are never ticked automatically and need an "I understand" acknowledgement
- **Hard guard** — home folder, personal data folders, credentials (`~/.ssh`, `~/.aws`, …) and operating-system trees can never be deleted, enforced again inside the deleter
- **Context checks** — a `node_modules` inside an app bundle, an editor extension, `nvm`/`pyenv`, or a global `lib/` folder is danger; one without a `package.json` beside it is caution

**Uninstall** — see [Uninstaller](#-uninstaller)
- **Apps, runtimes and packages** — macOS apps, Windows *Installed apps*, Linux launchers and Flatpaks; Node/Python/Rust/Go/Ruby/Java toolchains; packages of npm, pip, pipx, cargo, Homebrew, RubyGems, Scoop and `go install`
- **Clean, not just removed** — the tool's own uninstall command first, then caches, config, shell-profile lines, Windows PATH entries and registry keys — every step reviewable and individually tickable

**Cleanup**
- **24 categories** with official brand icons — see [Cleanup categories](#-cleanup-categories)
- **Bulk delete** — multi-select, "select all" / "select ≥ 100 MB" shortcuts, confirmation before anything is removed
- **Two delete modes** — **Trash** (default, restorable, cross-platform via [go-trash](https://github.com/laurent22/go-trash)) or **Permanent** (`os.RemoveAll`, guarded by a red double-confirm modal)
- **Reveal in Finder / Explorer** — from every result row and every folder child

**Interface**
- **Bilingual UI** — English and Bahasa Indonesia, switchable at runtime and persisted
- **Dark / Light / System** themes on neutral matte black, plus a custom accent colour and a runtime app-icon picker
- Motion throughout (stagger fade-in, pop, shimmer, indeterminate progress), skeleton loading, toasts, modal confirmations

**Maintenance**
- **In-app updates** — checks GitHub Releases, shows the changelog, downloads with progress and speed, verifies SHA-256 when GitHub publishes a digest, then hands the file to the OS installer (`.dmg`/`.pkg`/`.zip`, `.exe`/`.msi`)
- **Settings panel** — language, theme, accent, icon, automatic updates, cache clear, feedback email, privacy policy and terms
- **System & app info** — OS, architecture, cores, memory, app size and data folder, in-app

## 🧹 Cleanup categories

24 rules ship today. **Default** = pre-checked when you pick the category;
**opt-in** = unchecked until you ask for it, because deleting it costs a
re-download or is unrecoverable. **Risk** is the worst grade any location of the
category can reach (see [Safety](#-safety)); an empty cell means every location
is safe.

| # | Category | Targets | Default | Risk |
|---|---|---|---|---|
| 1 | Node.js | `node_modules` | Default |  |
| 2 | Go | `~/go/pkg/mod`, download cache, build cache | Default |  |
| 3 | Rust | `target/` (content-filtered), `~/.cargo/registry` | Default |  |
| 4 | Gradle | `build/` artifacts (filtered), `~/.gradle/caches` | Default |  |
| 5 | Maven | `~/.m2/repository`, `target/` (filtered) | Default | caution — `~/.m2/repository` may hold locally installed artifacts |
| 6 | C / C++ | `CMakeFiles`, `cmake-build-*` | Default |  |
| 7 | Python | `__pycache__`, `.pytest_cache`, `.mypy_cache`, `.ruff_cache`, virtualenvs | Default | caution — virtualenvs |
| 8 | .NET | `bin/`, `obj/` (Debug/Release/.dll filter) | Default |  |
| 9 | Frontend build cache | `.next`, `.nuxt`, `.turbo`, `.vite`, `.parcel-cache`, `.svelte-kit`, `.astro`, `coverage`, `storybook-static` | Default |  |
| 10 | Xcode & iOS | DerivedData, iOS DeviceSupport, CoreSimulator caches, SwiftPM `.build` | Default |  |
| 11 | Android | `app/build` (multi-segment pattern), `.cxx`, build-cache | Default |  |
| 12 | Docker & Buildx | `~/.docker/buildx`, Docker Desktop cache and logs (never the Windows WSL disk image) | Default | caution — buildx builder definitions |
| 13 | Homebrew | `Library/Caches/Homebrew`, `~/.cache/homebrew` | Default |  |
| 14 | PHP & Composer | `~/.composer/cache`, `~/.cache/composer` | Default |  |
| 15 | Python package cache | pip / uv / Poetry wheel caches, `~/.conda/pkgs` | Default |  |
| 16 | Flutter & Pub | `~/.pub-cache/hosted`, engine cache, `.dart_tool` | Default | caution — globally activated tools |
| 17 | Terraform | `.terraform/`, plugin cache | Default |  |
| 18 | .NET NuGet | `~/.nuget/packages`, HTTP cache | Default |  |
| 19 | JetBrains | IDE index caches (never the Toolbox folder that holds installed IDEs) | Default |  |
| 20 | Electron & browser cache | electron, npm `_cacache`/`_npx`, yarn, pnpm, Playwright, Cypress | **opt-in** | caution |
| 21 | Temp & system cache | `~/.cache`, `%TEMP%`, CrashDumps | **opt-in** | caution — emptied, never removed |
| 22 | AI / ML cache | HuggingFace, PyTorch, Ollama, Whisper | **opt-in** | caution — very large downloads |
| 23 | Ruby & Gems | `~/.gem`, Bundler cache | **opt-in** | **danger** — `~/.gem` holds installed gems |
| 24 | Emulators & simulators | Android AVDs, iOS Simulator data | **opt-in** | **danger** — AVDs and simulators hold app data |

## 🗑 Uninstaller

Switch to **Uninstall** in the header. BersihDisk lists what is installed, and
picking an entry opens its **plan** — nothing runs from the list itself.

| Group | Found through | Removed with |
|---|---|---|
| Applications | macOS `/Applications` + `~/Applications` (bundle ID from `Info.plist`); Windows *Installed apps* registry keys; Linux `~/.local/share/applications` launchers; Flatpak; Homebrew casks | macOS: the bundle plus its `~/Library` files; Windows: the vendor's own uninstaller; casks: `brew uninstall --cask --zap`; Flatpak: `--delete-data` |
| Runtimes & SDKs | a data catalog: rustup, nvm, fnm, Volta, pyenv, conda, SDKMAN!, Bun, Deno, pnpm, Poetry, uv, Go, Gradle, Maven, Flutter/Dart, Composer, .NET, rbenv, RVM | the tool's own command when it has one (`rustup self uninstall`, `conda init --reverse`, `rvm implode`), then its folders |
| Packages | `npm ls -g`, `pip list`, `pipx list`, `cargo install --list`, `brew leaves`, `gem list`, `scoop export`, `~/go/bin`; Microsoft Store apps (`Get-AppxPackage`); Linux `apt-mark showmanual` + `dpkg-query`, `snap list`, `pacman -Qe`, `dnf repoquery --userinstalled` | `npm uninstall -g`, `pip uninstall`, `cargo uninstall`, `brew uninstall`, `Remove-AppxPackage`, … Linux system packages: `apt purge`, `snap remove --purge`, `pacman -Rns`, `dnf remove` shown for you to run with `sudo` |

**What "clean" means.** A plan can contain five kinds of step, all listed before
you confirm:

1. **Command** — the native uninstall. If it fails, the run stops: leftovers of a
   tool that refused to uninstall are never deleted.
2. **Files & folders** — caches, data and configuration, moved to the Trash by
   default. Locations found through a stable identifier (a bundle ID, a tool's
   own folder) are pre-ticked; name-based guesses, config files that may hold
   tokens, and anything that may be your own work (`~/go/src`, Xcode Archives, Docker
   volumes) start **unticked** with a warning.
3. **Shell configuration** — the exact lines the installer added to `~/.zshrc`,
   `~/.bashrc`, `~/.profile`, fish config and friends (`. "$HOME/.cargo/env"`,
   `export NVM_DIR=…`, the `# >>> conda initialize >>>` block). The file is
   backed up next to itself as `*.bersihdisk-<time>.bak` before it is rewritten.
4. **Windows registry** — leftover uninstall keys; the key is exported to a `.reg`
   file under `%AppData%\BersihDisk\backups` before it is deleted. Only `HKCU` is
   changed automatically.
5. **PATH** — entries of the Windows user `PATH` that point at the removed tool.

**Administrator rights.** BersihDisk never elevates itself. A step that needs it
(`/usr/local/go`, `/usr/local/share/dotnet`, machine-wide registry keys) is shown
with the exact command to copy and run yourself, and is reported as *manual*.

**Platform details** — each choice below comes from the vendor's or tool's own
documentation (see [Research basis](#research-basis)):

| | macOS | Windows | Linux |
|---|---|---|---|
| App list | `.app` bundles, bundle ID via `plutil` | Uninstall registry keys in **three** places: `HKLM`, `HKLM\WOW6432Node`, and `HKCU` (per-user installs such as Chrome, Teams, Zoom) — not `Win32_Product`, which triggers MSI reconfiguration | `$XDG_DATA_HOME/applications` launchers, Flatpak, Snap |
| Uninstall | move the bundle; Docker's own `uninstall` binary is shown (it asks for a password) | `QuietUninstallString` if present, else `UninstallString`; MSI `/I{GUID}` is rewritten to `/X{GUID}`; run through `start /wait` so UAC prompts normally | `flatpak uninstall --delete-data`; `apt purge` (not `remove`, which keeps config), `snap remove --purge` |
| Leftovers | `~/Library` locations derived from the bundle ID are exact; name guesses start unticked; **Group Containers are only removed when the group ID is the app's own**, because macOS shares them between apps | `%APPDATA%`, `%LOCALAPPDATA%`, `%PROGRAMDATA%` per vendor docs (VS Code, Docker, Android Studio); registry key backed up to `.reg` first | `$XDG_CONFIG_HOME`, `$XDG_DATA_HOME`, `$XDG_CACHE_HOME`, `$XDG_STATE_HOME` — honoured when the user relocated them, defaults from the XDG spec otherwise |
| PATH | shell profile lines, backed up | user `PATH` written **directly in the registry**, keeping `REG_EXPAND_SZ`, then broadcast `WM_SETTINGCHANGE` — never `setx`, which truncates at 1024 characters | shell profile lines (`.profile`, `.bashrc`, fish config), backed up |
| Known trap | deleting `~/Library/Containers/*` fails with *Operation not permitted* even with `sudo` until the app has **Full Disk Access** — BersihDisk detects this and says so | Store/MSIX apps are not in the Uninstall registry at all | `dnf`'s "user installed" list can contain everything after a `system-upgrade`; check before removing |

**Platform coverage.** macOS is exercised end to end. The Windows registry and
Linux launcher code is written against each platform's documented layout and
covered by unit tests and cross-platform compilation in CI, but has had less
real-machine use — review the plan before confirming, as always. System packages
managed by `apt`/`dnf`/`snap` need root and are left to those tools.

## 📚 Research basis

Uninstall and cache locations were checked against vendor documentation and
community practice rather than guessed. Corrections that came out of it: Cargo's
home is `%USERPROFILE%\.cargo` on Windows (not `%LocalAppData%`), npm's Windows
cache is `%LocalAppData%\npm-cache`, Go's is `%LocalAppData%\go-build`, Yarn keeps
`~/Library/Caches/Yarn` on macOS, the `pip` cache belongs to the pip category, and
VS Code's clean uninstall also removes `~/.vscode-shared`.

- [Docker Desktop — uninstall](https://docs.docker.com/desktop/uninstall/) (per-OS file lists, the macOS Full Disk Access caveat)
- [VS Code — uninstall / clean uninstall](https://code.visualstudio.com/docs/setup/uninstall)
- [conda — uninstalling](https://docs.conda.io/projects/conda/en/latest/user-guide/install/macos.html#uninstalling-anaconda-or-miniconda) (`conda init --reverse --all`, `~/.condarc`, `~/.conda`, `~/.continuum`)
- [nvm — manual uninstall](https://github.com/nvm-sh/nvm#manual-uninstall), [rustup — self uninstall](https://rust-lang.github.io/rustup/installation/index.html), [Cargo home](https://doc.rust-lang.org/cargo/guide/cargo-home.html)
- [Homebrew Cask Cookbook — `zap`](https://docs.brew.sh/Cask-Cookbook#stanza-zap) (may remove shared resources; that is why the plan warns)
- [XDG Base Directory Specification](https://specifications.freedesktop.org/basedir/latest/)
- [Windows Installer — Uninstall registry key](https://learn.microsoft.com/en-us/windows/win32/msi/uninstall-registry-key), [winget `uninstall`](https://learn.microsoft.com/en-us/windows/package-manager/winget/uninstall), [`Remove-AppxPackage`](https://learn.microsoft.com/en-us/powershell/module/appx/remove-appxpackage)
- [npm cache defaults](https://docs.npmjs.com/cli/v8/commands/npm-cache/), [Yarn cache](https://classic.yarnpkg.com/lang/en/docs/cli/cache/)
- Community: Stack Overflow on `UninstallString`/`QuietUninstallString` and the `HKCU` per-user hive, `setx` truncation reports, `pkgutil --forget`, and macOS leftover-cleaner write-ups on Group Containers ownership.

**Deliberately not done:** elevating privileges. `osascript … with administrator
privileges`, `pkexec` and scripted UAC are exactly the patterns security tools
flag, and a wrong command as root cannot be undone — so root-only steps are shown
for you to run.

## 🚀 Getting started

### Download

Grab the latest build for your platform from
[Releases](https://github.com/cybersafetyid/BersihDisk/releases):

| OS | File | Install |
|---|---|---|
| macOS (Apple Silicon + Intel) | `BersihDisk-vX.Y.Z-macos-universal.dmg` | open the `.dmg`, drag BersihDisk to *Applications* |
| Windows 10/11 (x64) | `BersihDisk-vX.Y.Z-windows-x86_64-setup.exe` | run the installer (asks for admin rights) |
| Windows, no install | `BersihDisk-vX.Y.Z-windows-x86_64-portable.zip` | extract and run `BersihDisk.exe` |
| Linux (Debian/Ubuntu, x64) | `BersihDisk-vX.Y.Z-linux-x86_64.deb` | `sudo apt install ./BersihDisk-*.deb` |
| Linux, other distros | `BersihDisk-vX.Y.Z-linux-x86_64.tar.gz` | extract and run `./BersihDisk` |

`BersihDisk-vX.Y.Z-SHA256SUMS.txt` lists a checksum for every file.

The builds are **not code-signed or notarised**. On macOS a file downloaded in a
browser is quarantined: right-click the app → *Open* the first time (files fetched
by the in-app updater are not quarantined). On Windows, SmartScreen may show
"unknown publisher": *More info* → *Run anyway*. Linux needs GTK 3 and
WebKit2GTK 4.1 (`libgtk-3-0 libwebkit2gtk-4.1-0`, installed by the `.deb`).

### Build from source

Prerequisites: **Go 1.23+**, **Node.js 18+**, **Wails CLI v2.12+**, plus per OS:

| OS | Also needed |
|---|---|
| macOS | Xcode command line tools (`xcode-select --install`) |
| Windows | MinGW-w64 gcc (cgo, for the Trash library), [NSIS](https://nsis.sourceforge.io) (`choco install nsis`), Git Bash, `make` (`choco install make`) |
| Linux | `sudo apt install libgtk-3-dev libwebkit2gtk-4.1-dev dpkg-dev` |

```bash
git clone https://github.com/cybersafetyid/BersihDisk.git
cd BersihDisk
make doctor          # check the toolchain
make install-deps    # npm install
make dev             # hot-reload development window
```

> **Go ≥ 1.24**: the published `wails` CLI bundles an old `golang.org/x/tools`
> and aborts on Go's new export data (`internal error: package "…" without types
> was imported`). Build a patched CLI once — `make install-wails-fix` installs it
> to `~/.local/bin/wails-fixed`, and the Makefile uses it automatically.

### Package a release build

```bash
make build      # current OS → build/bin/ (quick, for testing)
make package    # current OS → dist/: the installer formats below
make verify     # what CI runs: gofmt, vet, Go tests, frontend typecheck
```

`make package` calls `scripts/package.sh`, which builds and packages **the OS it
runs on** (Wails cannot cross-compile a cgo app):

| Run on | Output in `dist/` |
|---|---|
| macOS | `BersihDisk-vX.Y.Z-macos-universal.dmg` (arm64 + x86_64, ad-hoc signed) |
| Windows | `…-windows-x86_64-setup.exe` (NSIS) and `…-windows-x86_64-portable.zip` |
| Linux | `…-linux-x86_64.deb` and `…-linux-x86_64.tar.gz` |

On Windows without `make`, run `bash scripts/package.sh` from Git Bash — it is the
same script.

## 🛠 Make targets

| Target | Purpose |
|---|---|
| `make run` | build and launch the app |
| `make dev` | development mode with hot reload |
| `make build` | production build for the current OS |
| `make package` | build + package the current OS into `dist/` (dmg / setup.exe + zip / deb + tar.gz) |
| `make verify` | CI gate: gofmt, vet, tests, typecheck (no build) |
| `make build-all` | build every platform (needs cross toolchains; CI packages each OS natively) |
| `make test` | Go unit tests (`-race`) + frontend typecheck |
| `make check` | full pre-commit gate: fmt, vet, tests, build |
| `make lint` | vet + typecheck + frontend build |
| `make fmt` / `make vet` / `make tidy` | Go hygiene |
| `make bump patch\|minor\|major` | semantic bump (1.2.3 → 1.2.4 / 1.3.0 / 2.0.0) in `VERSION`, `wails.json`, `package.json` |
| `make bump VER=1.2.3` | set an exact version |
| `make release` | local dry run: verify + package this OS (publishes nothing) |
| `make release-version VER=1.2.3` | bump and write the CHANGELOG entry (then commit and `make publish`) |
| `make changelog-preview` | print the CHANGELOG entry for the current version |
| `make changelog` | prepend that entry into `CHANGELOG.md` |
| `make tag` | create the annotated git tag `v<version>` on HEAD |
| `make release-notes` | write `dist/BersihDisk-v<version>-notes.md` from the entry |
| `make publish` | tag `v<version>` and push it: GitHub Actions builds all three OSes and creates the release |
| `make publish-local` | upload the artifacts built on this machine (`dist/`) — escape hatch |
| `make install-deps` | install frontend dependencies |
| `make doctor` | verify the toolchain (go, node, wails) |
| `make install-wails-fix` | build the Go ≥ 1.24 compatible Wails CLI |
| `make icons` | derive `build/appicon.png` + the in-app icon variants from `assets/logo.*` |
| `make clean` / `distclean` | remove build output / plus `node_modules` |

### Versioning

Nothing is hard-coded. The **`VERSION`** file is the single source of truth: it is
injected into the binary via `-ldflags "-X main.appVersion=$(VERSION)"` (used by
`make dev`, `make build*`, `make release`) and into the bundle metadata
(`CFBundleShortVersionString` / file version) through `{{.Info.ProductVersion}}`
in `build/darwin/Info.plist`. `make bump VER=x.y.z` updates `VERSION`,
`wails.json`, and `frontend/package.json` together.

### Publishing an updatable release

**CI/CD (GitHub Actions).** Two workflows in `.github/workflows/`:

- `ci.yml` — every push to `main` and every pull request: gofmt, `go vet`, Go
  tests and the frontend typecheck on Ubuntu, macOS and Windows.
- `release.yml` — pushing a tag `vX.Y.Z` (or running the workflow by hand with a
  `tag`) builds and packages on macOS, Windows and Linux runners with
  `scripts/package.sh`, then `scripts/publish.sh` creates the release, or updates
  it in place (`--clobber`) and prunes stale assets.

The workflows only call the repository's own scripts, so a local run produces the
same artifacts:

```bash
make bump patch        # or minor | major, or VER=1.2.3 — VERSION, wails.json, package.json
make publish           # release commit + tag + push; GitHub Actions does the rest
```

`make publish` does everything up to the push, then hands over to CI:

1. `make verify` (gofmt, vet, tests, typecheck).
2. The release commit: when the CHANGELOG has no entry for the version, or the bump
   is uncommitted, it runs `make changelog`, which writes the entry from the commit
   subjects and runs `git commit -am "chore: release 1.0.1"`. **`-am` commits every
   modified tracked file**, so commit or stash unrelated work first (`make changelog
   COMMIT=0` writes the entry without committing).
3. Tag `v1.0.1`, push the branch and the tag. The tag starts `release.yml`, which
   builds and packages macOS, Windows and Linux and creates the GitHub release.
   Nothing is uploaded from your machine.
4. Watch the workflow (`WATCH=0` skips this).

It refuses untracked files and a local tag that lags HEAD (`RETAG=1` moves it). If
the tag is already on origin it re-runs the workflow for that tag instead.
`make publish GH=echo GIT_PUSH=echo` prints the git/gh commands without running
them. `make release` builds this OS locally, to check what CI will build.

**The naming contract.** `scripts/package.sh` names files
`BersihDisk-vX.Y.Z-<os>-<arch>[-kind].<ext>`; `internal/updater` picks the asset
whose name has the OS word (`macos`/`windows`/`linux`), the architecture
(`x86_64`; the macOS build is `universal` and matches both) and the best extension
(`.dmg` › `.exe` › `.deb`). `TestPickAssetMatchesPackagedNames` fails if the two
drift apart.

**In-app updates.** The updater downloads only the asset returned by the update
check, from `github.com`/`githubusercontent.com` over HTTPS (every redirect is
checked), and refuses to install unless GitHub's SHA-256 `digest` for that asset
matches. It then opens the installer — the `.dmg` on macOS, the NSIS installer on
Windows (with the UAC prompt), the `.deb` on Linux — and on macOS and Windows the
app quits, because a running app cannot be replaced (Finder reports "the item is in
use"). Assets uploaded to a release always get a digest from GitHub.

## 🏗 Architecture

```
.
├── main.go                 window bootstrap (Wails), ldflags-injected version
├── app.go                  frontend bindings: DetectDrives, StartScan, StartDelete,
│                           ListPackages, PlanUninstall, StartUninstall, …
│                           (re-checks every delete against the scan's risk grades)
├── VERSION                 single source of truth for the version
├── Makefile                run / build / test / release / bump
├── assets/                 brand logo (SVG source + PNG renders)
├── CHANGELOG.md            release history, newest first
├── scripts/changelog.sh    Conventional Commits -> CHANGELOG entry
├── tools/iconvars/         derives the app-icon variants from assets/logo.png
└── internal/
    ├── safety/             risk levels, the hard delete guard, context checks
    ├── rules/              category definitions + matchers (dirNames, homePaths,
    │                       per-OS content filters, risk hints, project markers)
    ├── drive/              per-platform drive detection (darwin / windows / linux)
    ├── scanner/            two-phase concurrent scan engine + risk grading + progress
    ├── deleter/            bulk delete (trash / permanent), guard, keep-root, cancel
    ├── uninstall/          providers (package managers, apps, toolchain catalog),
    │                       plans, execution, profile/PATH/registry cleanup
    ├── browser/            folder listing with recursive sizes
    ├── reveal/             OS handoff: Open (Finder/Explorer), Launch (installer)
    ├── updater/            GitHub Releases check, download with progress, sha256
    ├── appicon/            runtime dock/taskbar icon swap (macOS)
    └── sysinfo/            OS, architecture, cores, memory, host identity
```

```
frontend/src/
├── main.tsx / App.tsx      entry + main flow
├── backend.ts              Wails binding wrapper + event listeners
├── categories/ drives/ scan/ results/ uninstall/ update/   feature panels
├── common/                 Icon (SVG engine), Toast, ThemeSwitch, ModeNav,
│                           RiskBadge / RiskAck (shared warning components)
├── hooks/                  useTheme, useAppliedSettings
├── i18n/                   lookup + interpolation, locales/id.ts, locales/en.ts
├── lib/                    types, format, risk, brandIcons, appIcons, settings, links
└── styles/                 tokens, layout, cards, panels, overlays, utils
```

**Flow:** pick drives and categories → `StartScan` (async goroutine) →
`scan:progress` / `scan:finished` events → results grouped by category → select
items → confirmation modal (trash or permanent) → `StartDelete` →
`delete:progress` / `delete:finished` → summary toast.

**Uninstall flow:** `ListPackages` (providers run concurrently) → pick one →
`PlanUninstall` (steps, sizes, warnings; stored server-side) → review and
acknowledge → `StartUninstall` with step IDs → `uninstall:progress` /
`uninstall:finished`.

## 🔒 Safety

A tool that deletes files has to earn trust, so the destructive path is narrow,
and the *backend* — not the UI — enforces every rule below.

**Every result carries a grade**, decided in Go and shown with its reason:

| Grade | Meaning | What the app does |
|---|---|---|
| **Safe** | regenerated automatically, nothing lost | pre-ticked, one confirmation |
| **Caution** | costs a re-download, may hold something unique (`~/.m2/repository`, a virtualenv, a folder with no project file beside it) | never ticked automatically, listed with reasons, needs an "I understand" checkbox |
| **Danger** | can break an installed app or tool, or destroy data (a global `node_modules`, anything inside a `.app`, an editor extension, `nvm`/`pyenv` installs, `~/.gem`, simulator devices) | as caution, plus **Trash only** — Permanent is disabled |
| **Protected** | never deleted: filesystem roots, the home folder, `Documents`/`Desktop`/…, `~/.ssh`, `~/.aws`, `~/.kube`, `/System`, `/usr/bin`, `C:\Windows`, `Program Files` | shown greyed out, checkbox disabled, refused again by the deleter |

How the grade is produced (`internal/safety`, `internal/rules`):

- **Hard guard** — a path is checked as written *and* after resolving symlinks, so
  a link into `~/.ssh` cannot smuggle it in. The deleter calls the same guard, so
  even a bug elsewhere cannot delete a protected path.
- **Context** — the surroundings of a match matter more than its name: inside an
  app bundle, an editor's `extensions/`, a version manager, a `lib/node_modules`
  global folder, or a package-manager prefix (`/opt/homebrew`, `/usr/local`).
- **Project markers** — `node_modules` needs `package.json` beside it, `target/`
  needs `Cargo.toml` or `pom.xml`, `bin/obj` need a `.csproj`, and so on. Without
  one the item is *caution*, not silently deleted.
- **Rule hints** — specific locations are raised above their category
  (`.m2/repository`, `~/.gem`, `.android/avd`, buildx builders, `.pub-cache/hosted`).
- **Server-side enforcement** — `StartDelete` accepts only paths from the last
  scan, re-derives their grade, refuses protected ones, and refuses anything
  above safe unless the acknowledgement came with the request. Danger forces the
  Trash whatever mode was asked for.
- **Containers are emptied, not removed** — `%TEMP%` and `~/.cache` keep the
  folder itself; entries that are in use are kept and reported.
- **Narrow paths** — the Windows Docker rule targets only `Docker\log` (the WSL disk
  image with every image and volume lives beside it); JetBrains targets one folder
  per IDE and never Toolbox, where the installed IDEs live.

Still true from the start:

- The scanner **never deletes**. Deletion accepts only explicit paths produced by a scan.
- A skip-list keeps the walker out of system locations on every platform: `.Trash`,
  `$Recycle.Bin`, `System Volume Information`, `Windows`, `Program Files`,
  `Library`, `/usr`, `/etc`, and friends.
- Symlinks are never followed — the walk does not cross mounts or link targets.
- Generic directory names (`build`, `target`, `bin`, `coverage`) require a content filter.
- Permanent mode always shows a red warning modal with a second confirmation.
- Categories that cost a large re-download or are unrecoverable are opt-in.

**Uninstaller safety** follows the same guard plus its own rules: it runs only
steps of a plan it stored itself (the UI sends step IDs, never commands or paths),
commands run without a shell from argument lists built by the backend, a failed
command stops the run, edited shell profiles and exported registry keys are
backed up first, and administrator-only steps are shown, never run.

## 🕸 Privacy

BersihDisk runs entirely on your machine: no account, no server, no
developer-side database, no usage telemetry. Scan results live in memory for the
session only. The one network call is the update check — a read request to
`api.github.com` for public release metadata, then the release asset download
from `github.com`; it carries no personal data. Preferences (language, theme,
accent, icon, automatic updates) are stored locally. The full text ships in-app
under **Settings → Privacy policy**.

## 🌐 Languages

The interface ships in **English** and **Bahasa Indonesia**, switchable at
runtime and remembered between launches. Every UI string lives in
`frontend/src/i18n/locales/id.ts` or `en.ts`; both catalogs share one shape and
the `Dictionary` type makes the compiler reject a missing or extra key, so a
half-translated build cannot ship. All code — files, functions, variables,
comments, commits — is in English.

Documentation is split the same way: this file is the English reference, and
[README.id.md](README.id.md) is its full Bahasa Indonesia counterpart.

## 🧪 Testing

```bash
make test        # go test -race ./internal/... + frontend typecheck
go test ./internal/... -v
go vet ./...
```

87 unit tests cover the safety guard (protected paths, symlink escapes, context
grading), the rules matchers and risk hints, the two-phase scanner, the deleter
(guard, keep-root), server-side delete vetting, the uninstaller (parsers for every
package manager's real output, shell-profile cleaning, plan building and
execution in a sandboxed `$HOME`, abort-on-failure, acknowledgement and
admin-only handling, catalog sanity) and the updater. The safety-critical path —
what may be matched and what may be deleted — is where the test density is
highest, and new categories and catalog entries are expected to arrive with tests.

## 🩺 Troubleshooting

| Symptom | Fix |
|---|---|
| `wails dev` dies with `internal error: package "…" without types was imported` | Go ≥ 1.24 vs the published CLI: `make install-wails-fix` |
| `make: wails: No such file` | `go install github.com/wailsapp/wails/v2/cmd/wails@v2.12.0`, or use the patched binary |
| macOS blocks the app ("unidentified developer") | right-click the app → *Open*, or `xattr -dr com.apple.quarantine /Applications/BersihDisk.app` |
| Scan misses folders under iCloud / external drives | check drive selection; the walk does not cross mount boundaries or symlinks |
| A directory that should be cleanable is not listed | the content filter rejected it — open an issue with the path and its top-level contents |
| Port 34115 already in use | another `wails dev` instance owns it; quit it or change the dev server port |

## 📑 Changelog & releases

[CHANGELOG.md](CHANGELOG.md) lists every release in
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) format, newest first,
and the project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html):
**MAJOR** for breaking changes, **MINOR** for new capabilities, **PATCH** for fixes.

Entries are generated from commit subjects rather than written by hand after the
fact, which is why [conventional commits](CONTRIBUTING.md#8-pull-request-workflow)
are required:

| Commit subject | Lands in |
|---|---|
| `feat:` / `feat(scope):` | `### Added` |
| `fix:` | `### Fixed` |
| `perf:` `refactor:` `docs:` `test:` `i18n:` `build:` `ci:` `chore:` `style:` | `### Changed` |
| `remove:` / `deprecate:` / `security:` | `### Removed` / `### Deprecated` / `### Security` |
| anything else | `### Other` |
| `feat!:` / `fix(scope)!:` | marked `**Breaking**` in its section |

Each line keeps its short commit hash, so any entry can be traced back to the
diff. The release workflow uploads the **curated CHANGELOG section** as the release
notes when that version is already listed, and falls back to the freshly generated
commit list otherwise.

Every release is also an **annotated git tag** `vX.Y.Z` — that tag is what
the in-app updater compares against, so a version without a tag is not a release.

```bash
make changelog-preview              # dry run: print the entry for VERSION
make changelog                      # prepend it into CHANGELOG.md
make tag                            # git tag -a v<version> (make publish does this)
make release                        # local dry run: verify + package this OS
make publish                        # tag + push → GitHub Actions builds and releases
```

`make publish` is the only target that writes to the remote: it pushes the current
branch when the tagged commit is missing there and pushes the annotated tag; the
release itself is created by the workflow with `--verify-tag`, so `gh` can never
invent a tag from a commit you did not intend. It refuses a dirty working tree
unless you pass `ALLOW_DIRTY=1`.

## 🤝 Contributing

Bug reports, translations, new cleanup categories, and UI work are all welcome.
Read [CONTRIBUTING.md](CONTRIBUTING.md) for the setup, conventions, the
step-by-step guide to adding a category, and the deletion-safety rules a pull
request is reviewed against. Run `make check` before pushing.

Good starting points: [`good first issue`](https://github.com/cybersafetyid/BersihDisk/issues?q=is%3Aissue+is%3Aopen+label%3A%22good+first+issue%22),
translation gaps, and missing cache paths for tools you actually use.

## 📜 Code of Conduct

Participants are expected to follow the [Code of Conduct](CODE_OF_CONDUCT.md)
(Contributor Covenant 2.1, with an Indonesian summary). Report issues to
**gorbypermana@gmail.com**.

## 🎖 Sponsorship & donations

BersihDisk is free, open source, ad-free, and will stay that way. Donations
unlock nothing — they simply buy the maintainer time.

[![Sponsor on GitHub](https://img.shields.io/badge/GitHub%20Sponsors-donate-ea4aaa?style=for-the-badge&logo=github-sponsors&logoColor=white)](https://github.com/sponsors/cybersafetyid)
[![Saweria](https://img.shields.io/badge/Saweria-donate-ff5722?style=for-the-badge)](https://saweria.co/cybersafetyid)
[![Trakteer](https://img.shields.io/badge/Trakteer-donate-red?style=for-the-badge)](https://trakteer.id/cybersafetyid)

Crypto — a wallet address is a long-lived commitment, so these are placeholders
until published officially:

```text
BTC  · REPLACE_WITH_BTC_ADDRESS
ETH / USDT (ERC-20) · REPLACE_WITH_ETH_ADDRESS
USDT (TRC-20) · REPLACE_WITH_TRON_ADDRESS
```

Prefer a pull request over a payment? New cleanup rules, translations, and bug
reports are worth just as much.

## 🙏 Acknowledgements

- [Wails](https://wails.io) — Go + webview desktop framework
- [go-trash](https://github.com/laurent22/go-trash) — cross-platform recycle bin
- [simple-icons](https://github.com/simple-icons/simple-icons) — the official brand SVGs on category cards
- [Lucide](https://lucide.dev) — UI icon set
- [React](https://react.dev), [TypeScript](https://www.typescriptlang.org), [Vite](https://vitejs.dev)

## ⚖️ License

Released under the [MIT License](LICENSE). Copyright © 2026
[Gorby Permana](https://github.com/cybersafetyid). Third-party brand icons
remain under their own trademarks and licenses and are used for identification
only.
