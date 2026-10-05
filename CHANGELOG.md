# Changelog

All notable changes to **BersihDisk** are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).
Every entry is generated from [Conventional Commit](https://www.conventionalcommits.org/)
subjects by `scripts/changelog.sh` (`make changelog`), newest release first, and
each release is tagged `vX.Y.Z`.

[Riwayat versi Bahasa Indonesia →](#catatan-versi-bahasa-indonesia)

## [1.1.1] - 2026-10-05

### Added

- add drive space analyzer page with interactive sunburst chart and deletion support (`4ced20e`)

## [1.1.0] - 2026-09-28

### Added

- add comprehensive application uninstaller and safety risk assessment framework (`c3f6c40`)

## [1.0.3] - 2026-09-28

### Added

- add post-delete path re-measurement and enhanced category panel layout options (`3cb2e23`)

## [1.0.2] - 2026-09-28

### Fixed

- quit app on macOS during updates to allow replacing running bundle (`19d7e0f`)

## [1.0.1] - 2026-09-28

### Added

- add smooth menu animations and capitalize app name to BersihDisk (`90fe9eb`)
- support semantic version bumping in Makefile and bump version to 1.0.1 (`ae25ef9`)

### Changed

- update contributing and README files (`de90764`)
- build and release macOS dmg, Windows exe and Linux deb from the same scripts (`29124d3`)
- skip the macOS-only symlinked cache test on Windows (`3686825`)

### Fixed

- run recipes under /bin/sh and source release notes from CHANGELOG (`983d6f4`)
- harden delete, update and scan paths; remember window state; edge-to-edge icon (`b256ab9`)

## [1.0.0] - 2026-09-27

The first public release. This entry is written by hand because it predates the
git history of the repository; every release after it is generated from commits.

### Added

- Two-phase scan engine: a walking *find* phase with a system-folder skip-list,
  then a concurrent *measure* phase, both reporting live progress and supporting
  cancellation
- 24 cleanup categories — Node.js, Go, Rust, Gradle, Maven, C/C++, Python, .NET,
  frontend build cache, Xcode & iOS, Android, Docker & Buildx, Homebrew,
  PHP & Composer, Python package cache, Flutter & Pub, Terraform, NuGet,
  JetBrains, Electron & browser cache, temp & system cache, AI/ML cache,
  Ruby & Gems, emulators & simulators
- Per-platform drive detection with used/free capacity: macOS (`/Volumes`),
  Windows (A:–Z: via WinAPI), Linux (`/proc/mounts`)
- Content filters so generic `build/`, `target/`, `bin/` and `obj/` directories
  only match real artifacts and never source code
- Bulk delete with multi-select, "select all" and "select ≥ 100 MB" shortcuts
- Two delete modes: restorable **Trash** via
  [go-trash](https://github.com/laurent22/go-trash), or **Permanent** behind a
  red double-confirmation modal
- Folder browser that lists any scan result with per-child sizes computed in
  parallel, without double-counting children of a selected folder
- Reveal in Finder / Explorer from result rows and folder children
- In-app updates: GitHub Releases check, changelog display, download with
  progress and speed, SHA-256 verification when a digest is published, and
  handoff of `.dmg`/`.pkg`/`.zip`/`.exe`/`.msi` to the OS
- Bilingual interface (English, Bahasa Indonesia) with runtime switching
- Dark / Light / System themes on neutral matte black, custom accent colour, and
  a runtime app-icon picker (macOS)
- Settings panel with cache clearing, feedback email, privacy policy and terms
- System info and app info views
- `make` toolchain for the whole lifecycle: `dev`, `build`, `build-all`, `test`,
  `check`, `bump`, `release`, `changelog`, `tag`, `publish`
- Unit tests for the rules matchers, drive detection, scanner, deleter, folder
  browser and updater

### Security

- Deletion accepts only explicit paths produced by a scan; the scanner never
  deletes
- Skip-list protects `.Trash`, `$Recycle.Bin`, `System Volume Information`,
  `Windows`, `Program Files`, `Library`, `/usr`, `/etc` and similar locations
- Directory walking does not follow symlinks or cross mount points
- Data-costly categories (AI model weights, emulator images, browser caches,
  temp) are opt-in and show a risk note
- No telemetry, no accounts, no server-side storage; the only network request is
  the public release check

## Catatan versi Bahasa Indonesia

### [1.0.0] - 2026-09-27

Rilis publik pertama. Entri ini ditulis manual karena mendahului riwayat git
repo; rilis setelahnya dibuat otomatis dari commit oleh `make changelog`.

- **Mesin scan dua fase** — fase cari dengan skip-list folder sistem, lalu fase
  ukur secara concurrent, keduanya melaporkan progress dan bisa dibatalkan
- **24 kategori pembersihan** — Node.js, Go, Rust, Gradle, Maven, C/C++, Python,
  .NET, build cache frontend, Xcode & iOS, Android, Docker & Buildx, Homebrew,
  PHP & Composer, cache paket Python, Flutter & Pub, Terraform, NuGet, JetBrains,
  Electron & cache browser, temp & cache sistem, cache AI/ML, Ruby & Gems,
  emulator & simulator
- **Deteksi drive lintas platform** beserta kapasitas terpakai/bebas: macOS
  (`/Volumes`), Windows (A:–Z: via WinAPI), Linux (`/proc/mounts`)
- **Filter konten** sehingga folder generik `build/`, `target/`, `bin/`, `obj/`
  hanya mengenali artefak sungguhan, bukan kode sumber
- **Hapus massal** dengan multi-select dan pintasan "Pilih semua" / "Pilih ≥ 100 MB"
- **Dua mode hapus** — Tempat Sampah yang bisa dipulihkan (go-trash) atau
  Permanen di balik modal konfirmasi ganda
- **Telusuri isi folder** dengan ukuran tiap anak dihitung paralel, tanpa
  menghitung ganda anak dari folder yang terpilih
- **Buka di Finder / Explorer** dari baris hasil maupun anak folder
- **Pembaruan dalam aplikasi** — cek GitHub Releases, catatan rilis, unduhan
  dengan progress + kecepatan, verifikasi SHA-256 bila tersedia, lalu serahkan
  berkas ke OS
- **UI bilingual** (Inggris, Bahasa Indonesia) yang bisa diganti saat berjalan
- **Tema Gelap / Terang / Sistem**, warna aksen kustom, dan pemilih ikon aplikasi
- **Panel Pengaturan** berisi bersihkan cache, email masukan, kebijakan privasi,
  dan syarat & ketentuan
- **Info sistem & info aplikasi**
- **Perkakas `make`** untuk seluruh daur hidup: `dev`, `build`, `build-all`,
  `test`, `check`, `bump`, `release`, `changelog`, `tag`, `publish`
- **Unit test** untuk matcher aturan, deteksi drive, scanner, deleter, browser
  folder, dan updater

**Keamanan:**

- Penghapusan hanya menerima path eksplisit hasil scan; scanner tidak pernah menghapus
- Skip-list melindungi `.Trash`, `$Recycle.Bin`, `System Volume Information`,
  `Windows`, `Program Files`, `Library`, `/usr`, `/etc`, dan sejenisnya
- Walking direktori tidak mengikuti symlink maupun menyeberang mount
- Kategori yang berbiaya unduh ulang atau menghilangkan data bersifat opt-in dan
  menampilkan catatan risiko
- Tanpa telemetri, tanpa akun, tanpa penyimpanan sisi server; satu-satunya
  permintaan jaringan adalah cek rilis publik

[⬆ Kembali ke atas](#changelog) · [English entry](#100---2026-09-27)
