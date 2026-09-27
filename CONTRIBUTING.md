# Contributing to BersihDisk

Thank you for helping make BersihDisk better. This guide explains how to set up
the project, where to look, and how your pull request gets reviewed.

Terima kasih sudah berkontribusi. Panduan ini menjelaskan cara menyiapkan
lingkungan kerja, struktur kode, dan alur review pull request.

[English](#how-to-contribute) · [Ringkasan Bahasa Indonesia](#cara-berkontribusi-ringkasan-bahasa-indonesia)

Docs live in two files: [README.md](README.md) (English) and
[README.id.md](README.id.md) (Bahasa Indonesia) — change both when you change behaviour.

## How to contribute

### 1. Ways to help

| Want to… | Open a | Notes |
|---|---|---|
| Report a bug or a wrong deletion | **bug report issue** | Include OS, app version, and the exact path involved |
| Ask for a new cleanup category | **feature issue** | Link the tool's official docs that document the cache path |
| Fix a typo, translation, or small bug | **pull request** | Look for [`good first issue`](https://github.com/cybersafetyid/BersihDisk/issues?q=is%3Aissue+is%3Aopen+label%3A%22good+first+issue%22) |
| Add a cleaning rule | **pull request** | Read [Adding a cleanup category](#adding-a-cleanup-category) first |

A disk cleaner can destroy data, so issues about **false positives** (something
flagged that should not be) and **false negatives** (a known cache directory
that is missed) are always treated as high priority.

### 2. Development setup

Prerequisites: **Go 1.23+**, **Node.js 18+**, and the **Wails CLI v2.12+**.

```bash
git clone https://github.com/cybersafetyid/BersihDisk.git
cd BersihDisk
make doctor        # verify go / node / wails are present and usable
make install-deps  # npm install for the frontend
make dev           # hot-reload development window
```

`make doctor` prints what is missing. If you are on **Go ≥ 1.24**, the published
`wails` binary fails to parse Go's new export data; build a patched CLI instead:

```bash
make install-wails-fix   # writes ~/.local/bin/wails-fixed, Makefile picks it up automatically
```

### 3. Daily commands

Everything is driven by `make` (run `make help` for the full list).

| Command | What it does |
|---|---|
| `make dev` | hot-reload development mode |
| `make run` | build and launch the app |
| `make test` | Go unit tests with `-race` + frontend typecheck |
| `make vet` / `make fmt` | `go vet ./...` / `gofmt -w` |
| `make lint` | vet + frontend typecheck + frontend build |
| `make check` | full gate: fmt, vet, tests, build — **run this before pushing** |
| `make build` | production build for the current OS into `build/bin/` |
| `make build-all` | cross-compile macOS / Windows / Linux |
| `make bump VER=1.2.3` | update `VERSION`, `wails.json`, `frontend/package.json` |
| `make release` | `make check` + build + package artifacts into `dist/` |

### 4. Project layout

```
main.go / app.go        window bootstrap + the Wails bindings the UI calls
internal/
  rules/                category definitions and path matchers (unit tested)
  drive/                per-platform drive detection (darwin / windows / linux)
  scanner/              two-phase concurrent scan + progress events
  deleter/              bulk delete via trash or permanent removal
  browser/              folder listing with per-child sizes
  reveal/               OS handoff (Finder / Explorer, launching installers)
  updater/              GitHub Releases check + download + sha256 verify
  appicon/ sysinfo/     runtime dock icon, system info
frontend/src/
  App.tsx               main flow
  backend.ts            Wails binding + event wrapper
  categories/ drives/ scan/ results/ update/ common/
  hooks/                useTheme, useAppliedSettings
  i18n/locales/         id.ts and en.ts translation catalogs
  lib/                  types, formatting, icon curation, settings
  styles/               tokens, layout, cards, panels, overlays, utils
build/                  platform manifests, icons, installer scripts
```

### 5. Code conventions

- **Go code is written in English** — file names, package names, functions,
  variables, comments, and commit messages.
- **User-facing text is never hard-coded in a component.** Add the string to
  both `frontend/src/i18n/locales/id.ts` and `en.ts`. The catalogs share one
  shape, and the `Dictionary` type makes TypeScript reject a missing or extra
  key, so add to both files in the same commit.
- Category names, descriptions, and risk notes are keyed by category ID:
  `categories.<id>.name`, `.desc`, `.risk`.
- Comments are for *why*, not *what*: a hidden constraint, an invariant, a
  workaround. Most functions need none.
- `gofmt` output is authoritative; `make check` fails on unformatted files.
- Keep changes surgical. A bug fix does not need surrounding cleanup.

### 6. Adding a cleanup category

A category touches four places. Do them together so the UI never renders a
category with missing copy:

1. **Rule** — `internal/rules/rules.go`: add the `Cat…` ID and `Icon…` name, then
   append a `Rule` with its `dirNames` (directory patterns, multi-segment allowed),
   `homePaths` (home-relative, `win:` / `unix:` prefixes for per-OS entries),
   `contentFilters` (per-name content check) and `maxDepth`. Generic directory
   names (`build`, `target`, `bin`, `obj`) **must** carry a content filter so
   source code is never matched.
2. **Tests** — `internal/rules/rules_test.go`: assert the matcher hits real
   artifact paths and rejects look-alikes.
3. **Frontend metadata** — `frontend/src/lib/types.ts` (`CategoryUI`) and the
   category list the backend exposes; set `optIn: true` whenever deleting the
   data costs a re-download or is unrecoverable.
4. **Copy + icon** — add `categories.<id>.*` to both locale catalogs, and map
   the icon in `frontend/src/lib/brandIcons.ts` (simple-icons) or use a Lucide
   name.

Default state for a new category is **off**. Opt-in categories stay unchecked
because the user should decide to pay a re-download.

### 7. Safety rules for deletions

Non-negotiable, and a common reason a PR is asked for changes:

- The scanner never deletes; deletion only accepts explicit paths produced by a
  scan.
- The skip-list (`.Trash`, `$Recycle.Bin`, `System Volume Information`,
  `Windows`, `Program Files`, `Library`, `/usr`, `/etc`, …) must keep protecting
  system locations on every platform.
- Never follow symlinks into a target the user did not select, and never cross
  a mount point during the walk.
- Anything that can lose user data (emulator images, AI model weights, browser
  profiles) is opt-in and shows a risk note.

### 8. Pull request workflow

```bash
git checkout -b feat/your-topic      # branch from main
# … commit …
make check                           # fmt + vet + tests + build
git push -u origin feat/your-topic
```

- One logical change per pull request; keep the diff reviewable.
- Commit messages follow [Conventional Commits](https://www.conventionalcommits.org/):
  `feat:`, `fix:`, `refactor:`, `docs:`, `test:`, `chore:`, `i18n:`.
- In the description, explain **why**, link the issue (`Fixes #123`), and —
  for UI work — attach a screenshot or short recording.
- If your change affects scanning or deletion, state which OSes you tested.

### 9. Changelog and release tags

`CHANGELOG.md` is not maintained by hand. `scripts/changelog.sh` builds each
entry from the commit subjects since the previous tag, and `make release` runs it
for you:

```bash
make changelog-preview   # see exactly what would be written for VERSION
make changelog           # prepend the entry into CHANGELOG.md
make tag                 # annotate HEAD as v<version>
make release             # check + build + package, then changelog + tag + notes
make publish             # push the tag, then the GitHub release from dist/ artifacts
```

This is why the commit prefixes in step 8 are mandatory rather than stylistic:
`feat:` lands under **Added**, `fix:` under **Fixed**, and
`perf|refactor|docs|test|i18n|build|ci|chore|style` under **Changed**; a `!`
before the colon marks the line **Breaking**. A subject with no prefix still
appears, but under **Other** — so tag your work.

`make publish` is the step that touches the remote: it pushes the branch and the
annotated tag when they are missing, and releases with `--verify-tag` so a tag you
never pushed cannot be replaced by one gh invents from the default branch. It also
stops on a dirty working tree unless you pass `ALLOW_DIRTY=1`; use
`make publish GH=echo GIT_PUSH=echo` to preview the commands.

The notes uploaded to GitHub come from the CHANGELOG section when the version is
already listed there, so editing that entry before `make publish` is enough to
fix the wording of a release.

A version without a `vX.Y.Z` tag is not a release: the in-app updater compares
against tags.

### 10. Review process

- A maintainer reviews within a few days; CI-equivalent checks are `make check`.
- Reviews look for: correctness of the matcher, data-safety (section 7), tests
  for new behavior, and both locale catalogs updated.
- Address feedback by pushing new commits; the maintainer squash-merges, so you
  do not need to force-push during review.
- Merged pull requests are included in the next release notes and credited in
  the release.

### 11. Reporting a bug well

Include: app version (shown in the Settings panel), OS and version, the
categories selected, the paths that were wrong, and — for deletion issues —
whether Trash or Permanent mode was used. Please **redact** anything in the
path that you do not want public.

## Cara berkontribusi (ringkasan Bahasa Indonesia)

1. **Siapkan lingkungan**: butuh Go 1.23+, Node.js 18+, Wails CLI v2.12+.
   Jalankan `make doctor`; jika memakai Go ≥ 1.24 jalankan
   `make install-wails-fix`.
2. **Kode harian**: `make dev` (hot-reload), `make test`, dan `make check`
   (gerbang penuh: fmt, vet, test, build) sebelum push.
3. **Konvensi**: seluruh kode dan komentar berbahasa Inggris; teks UI tidak
   pernah ditulis langsung di komponen — tambahkan kunci string ke `id.ts`
   **dan** `en.ts` pada commit yang sama.
4. **Menambah kategori**: ubah `internal/rules/rules.go`, tambah test di
   `rules_test.go`, metadata di `frontend/src/lib/types.ts`, lalu salinan teks
   `categories.<id>.name/.desc/.risk` di kedua katalog. Nama folder generik
   (`build`, `target`, `bin`, `obj`) wajib punya filter konten. Kategori baru
   default-nya **tidak tercentang**.
5. **Keamanan hapus**: scanner tidak pernah menghapus; hanya path hasil scan
   yang boleh dihapus; skip-list sistem harus tetap terlindungi di semua
   platform; jangan mengikuti symlink atau menyeberang mount; fitur yang bisa
   membuat data hilang bersifat opt-in dan menampilkan catatan risiko.
6. **Pull request**: satu perubahan logis per PR, branch dari `main`, commit
   memakai Conventional Commits, jelaskan *mengapa*, tautkan issue
   (`Fixes #123`), dan sertakan tangkapan layar untuk perubahan UI.
7. **Laporan bug**: sebutkan versi aplikasi, OS, kategori yang dipilih, path
   yang bermasalah, dan mode hapus (Tempat Sampah / Permanen). Sensor bagian
   path yang bersifat privat.

Perilaku dalam komunitas diatur oleh [Code of Conduct](CODE_OF_CONDUCT.md).
