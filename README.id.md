<div align="center">

<img src="assets/logo.png" alt="BersihDisk logo — a blue disk platter with a bright sweep wiping its surface" width="128" height="128" />

# BersihDisk

**Pembersih disk untuk developer** · **Disk cleaner for developers**

Membersihkan gigabyte yang ditinggalkan toolchain-mu: `node_modules`, build artifact, cache paket, cache model AI, dan file temp — dengan UI bilingual modern dan langkah hapus yang kamu kendalikan.

[![release](https://img.shields.io/github/v/release/cybersafetyid/BersihDisk?style=flat-square&color=4f8cff&label=release&logo=github&logoColor=white)](https://github.com/cybersafetyid/BersihDisk/releases)
[![downloads](https://img.shields.io/github/downloads/cybersafetyid/BersihDisk/total?style=flat-square&color=34d399&label=downloads)](https://github.com/cybersafetyid/BersihDisk/releases)
[![last commit](https://img.shields.io/github/last-commit/cybersafetyid/BersihDisk?style=flat-square&color=909090&label=updated)](https://github.com/cybersafetyid/BersihDisk/commits/main)
[![issues](https://img.shields.io/github/issues/cybersafetyid/BersihDisk?style=flat-square&color=fbbf24&label=issues)](https://github.com/cybersafetyid/BersihDisk/issues)
[![stars](https://img.shields.io/github/stars/cybersafetyid/BersihDisk?style=flat-square&color=4f8cff&label=stars&logo=github&logoColor=white)](https://github.com/cybersafetyid/BersihDisk)
[![forks](https://img.shields.io/github/forks/cybersafetyid/BersihDisk?style=flat-square&color=6ba1ff&label=forks)](https://github.com/cybersafetyid/BersihDisk/forks)

[![platform](https://img.shields.io/badge/platform-macOS%20%7C%20Windows%20%7C%20Linux-1c1c1c?style=flat-square)](#-mulai-menggunakan)
[![go](https://img.shields.io/badge/Go-1.23%2B-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev)
[![wails](https://img.shields.io/badge/Wails-v2.12-B00606?style=flat-square)](https://wails.io)
[![react](https://img.shields.io/badge/React-18-1f2b3a?style=flat-square&logo=react&logoColor=61DAFB)](https://react.dev)
[![typescript](https://img.shields.io/badge/TypeScript-4.6-3178C6?style=flat-square&logo=typescript&logoColor=white)](https://www.typescriptlang.org)
[![vite](https://img.shields.io/badge/Vite-3-646CFF?style=flat-square&logo=vite&logoColor=white)](https://vitejs.dev)

[![license](https://img.shields.io/badge/license-MIT-blue?style=flat-square)](LICENSE)
[![tests](https://img.shields.io/badge/tests-34%20Go%20unit%20tests-34d399?style=flat-square)](#-pengujian)
[![categories](https://img.shields.io/badge/cleanup%20categories-24-4f8cff?style=flat-square)](#-kategori-pembersihan)
[![telemetry](https://img.shields.io/badge/telemetry-none-brightgreen?style=flat-square)](#-privasi)
[![languages](https://img.shields.io/badge/UI%20languages-EN%20%7C%20ID-6ba1ff?style=flat-square)](#-bahasa)
[![PRs](https://img.shields.io/badge/PRs-welcome-brightgreen?style=flat-square)](CONTRIBUTING.md)
[![contributor covenant](https://img.shields.io/badge/Contributor%20Covenant-2.1-4baaaa?style=flat-square)](CODE_OF_CONDUCT.md)
[![changelog](https://img.shields.io/badge/changelog-Keep%20a%20Changelog-4f8cff?style=flat-square)](CHANGELOG.md)

**[English](README.md)** · **Bahasa Indonesia**

[Unduh](https://github.com/cybersafetyid/BersihDisk/releases) · [Fitur](#-fitur) · [Mulai](#-mulai-menggunakan) · [Kategori](#-kategori-pembersihan) · [Kontribusi](CONTRIBUTING.md) · [Changelog](#-changelog--rilis) · [Donasi](#-sponsor-donasi)

> Dokumentasi ini tersedia dalam dua bahasa: **[English](README.md)** dan **Bahasa Indonesia** (berkas ini).
</div>

---

## Tentang

BersihDisk adalah pembersih disk desktop untuk developer. Satu mesin yang dipakai
membangun perangkat lunak menumpuk cache dengan cepat — `node_modules` di tiap
proyek, modul cache Go dan Cargo, repositori Gradle dan Maven, layer buildx
Docker, bobot model HuggingFace, DerivedData Xcode — dan sebagian besarnya bisa
dibangun ulang. BersihDisk mencari direktori itu di semua drive yang terpasang,
menampilkan isinya, lalu hanya menghapus apa yang kamu centang.

Aplikasi desktop native (Go + Wails) dengan antarmuka web (React + TypeScript):
scan cepat dan concurrent, UI tetap responsif. Tanpa akun, tanpa server, tanpa
unggah data.

## ✨ Fitur

**Deteksi**
- **Scan multi-drive** — macOS (`/Volumes`), Windows (A:–Z: via WinAPI), Linux (`/proc/mounts`), lengkap dengan kapasitas terpakai/bebas
- **Engine dua fase** — fase *cari* (walking directory tree dengan skip-list proteksi folder sistem) lalu fase *ukur* (menghitung ukuran secara concurrent), dengan progress live dan batal
- **Filter konten cerdas** — folder `build`/`target`/`bin`/`obj` generik hanya dihitung artefak jika isinya bukan kode sumber
- **Telusuri isi folder** — hasil scan bisa dibuka isinya (lazy, ukuran tiap anak dihitung paralel) sehingga file/folder di dalamnya dipilih satu per satu; anak dari folder yang ikut terpilih tidak dihitung ganda

**Pembersihan**
- **24 kategori** dengan ikon merek resmi — lihat [Kategori pembersihan](#-kategori-pembersihan)
- **Bulk delete** — multi-select, pintasan "Pilih semua" / "Pilih ≥ 100 MB", konfirmasi sebelum eksekusi
- **Dua mode hapus** — **Tempat Sampah** (default, bisa dipulihkan, lintas-platform via [go-trash](https://github.com/laurent22/go-trash)) atau **Permanen** (`os.RemoveAll`, dikunci modal merah dengan konfirmasi ganda)
- **Buka di Finder / Explorer** — di tiap baris hasil dan tiap anak folder

**Antarmuka**
- **UI bilingual** — Inggris dan Bahasa Indonesia, diganti saat aplikasi berjalan dan tersimpan
- **Tema Gelap / Terang / Sistem** pada hitam doff netral, plus warna aksen kustom dan pemilih ikon aplikasi saat berjalan
- Animasi di seluruh aplikasi (stagger fade-in, pop, shimmer, indeterminate progress), skeleton loading, toast, modal konfirmasi

**Pemeliharaan**
- **Pembaruan dalam aplikasi** — memeriksa GitHub Releases, menampilkan catatan rilis, mengunduh dengan progress + kecepatan, memverifikasi SHA-256 bila tersedia, lalu tombol Install yang menyerahkan berkas ke OS (`.dmg`/`.pkg`/`.zip`, `.exe`/`.msi`)
- **Panel Pengaturan** — bahasa, tema, warna aksen, ikon, pembaruan otomatis, bersihkan cache, email masukan, kebijakan privasi, dan syarat & ketentuan
- **Info sistem & aplikasi** — OS, arsitektur, core CPU, memori, ukuran aplikasi, dan folder data

## 🧹 Kategori pembersihan

24 aturan tersedia. **Default** = sudah tercentang saat kategori dipilih;
**opt-in** = tidak tercentang sampai kamu memintanya, karena menghapusnya berarti
perlu unduh ulang atau datanya hilang permanen.

| # | Kategori | Target | Status |
|---|---|---|---|
| 1 | Node.js | `node_modules` | Default |
| 2 | Go | `~/go/pkg/mod`, cache unduhan, build cache | Default |
| 3 | Rust | `target/` (filter konten), `~/.cargo/registry` | Default |
| 4 | Gradle | artefak `build/` (filter) + `~/.gradle/caches` | Default |
| 5 | Maven | `~/.m2/repository`, `target/` (filter) | Default |
| 6 | C / C++ | `CMakeFiles`, `cmake-build-*` | Default |
| 7 | Python | `__pycache__`, `.pytest_cache`, `.mypy_cache`, `.ruff_cache`, venv | Default |
| 8 | .NET | `bin/`, `obj/` (filter Debug/Release/.dll) | Default |
| 9 | Build cache frontend | `.next`, `.nuxt`, `.turbo`, `.vite`, `.parcel-cache`, `.svelte-kit`, `.astro`, `coverage`, `storybook-static` | Default |
| 10 | Xcode & iOS | DerivedData, iOS DeviceSupport, CoreSimulator, `.build` SwiftPM | Default |
| 11 | Android | `app/build` (pola multi-segmen), `.cxx`, build-cache | Default |
| 12 | Docker & Buildx | `~/.docker/buildx`, cache & log Docker Desktop | Default |
| 13 | Homebrew | `Library/Caches/Homebrew`, `~/.cache/homebrew` | Default |
| 14 | PHP & Composer | `~/.composer/cache`, `~/.cache/composer` | Default |
| 15 | Cache paket Python | wheel cache pip / uv / Poetry, `~/.conda/pkgs` | Default |
| 16 | Flutter & Pub | `~/.pub-cache/hosted`, engine cache, `.dart_tool` | Default |
| 17 | Terraform | `.terraform/`, plugin cache | Default |
| 18 | .NET NuGet | `~/.nuget/packages`, HTTP cache | Default |
| 19 | JetBrains | cache indeks IDE | Default |
| 20 | Electron & cache browser | electron, npm `_cacache`/`_npx`, yarn, pnpm, Playwright, Cypress | **opt-in** |
| 21 | Temp & cache sistem | `~/.cache`, `%TEMP%`, CrashDumps | **opt-in** |
| 22 | Cache AI/ML | HuggingFace, PyTorch, Ollama, Whisper | **opt-in** |
| 23 | Ruby & Gems | `~/.gem`, cache bundler | **opt-in** |
| 24 | Emulator & simulator | AVD Android, data iOS Simulator | **opt-in** |

## 🚀 Mulai menggunakan

**Unduh** build terbaru dari
[Releases](https://github.com/cybersafetyid/BersihDisk/releases):

| OS | Berkas | Pasang |
|---|---|---|
| macOS (Apple Silicon + Intel) | `BersihDisk-vX.Y.Z-macos-universal.dmg` | buka `.dmg`, seret BersihDisk ke *Applications* |
| Windows 10/11 (x64) | `BersihDisk-vX.Y.Z-windows-x86_64-setup.exe` | jalankan installer (meminta hak admin) |
| Windows, tanpa install | `BersihDisk-vX.Y.Z-windows-x86_64-portable.zip` | ekstrak lalu jalankan `BersihDisk.exe` |
| Linux (Debian/Ubuntu, x64) | `BersihDisk-vX.Y.Z-linux-x86_64.deb` | `sudo apt install ./BersihDisk-*.deb` |
| Linux, distro lain | `BersihDisk-vX.Y.Z-linux-x86_64.tar.gz` | ekstrak lalu jalankan `./BersihDisk` |

`BersihDisk-vX.Y.Z-SHA256SUMS.txt` memuat checksum tiap berkas.

Build **belum ditandatangani/dinotarisasi**. Di macOS berkas hasil unduhan browser
diberi karantina: klik kanan aplikasi → *Buka* untuk pertama kali (berkas dari
pembaruan dalam aplikasi tidak dikarantina). Di Windows, SmartScreen bisa
menampilkan "publisher tidak dikenal": *More info* → *Run anyway*. Linux butuh
GTK 3 dan WebKit2GTK 4.1 (`libgtk-3-0 libwebkit2gtk-4.1-0`, otomatis lewat `.deb`).

**Build dari sumber** (butuh Go 1.23+, Node.js 18+, Wails CLI v2.12+):

```bash
git clone https://github.com/cybersafetyid/BersihDisk.git
cd BersihDisk
make doctor          # cek toolchain
make install-deps    # npm install
make dev             # mode pengembangan hot-reload
make build           # build cepat OS saat ini -> build/bin/
make package         # build + paket OS saat ini -> dist/ (dmg / setup.exe + zip / deb + tar.gz)
make verify          # gerbang CI: gofmt, vet, tes Go, typecheck frontend
```

`make package` menjalankan `scripts/package.sh`, yang membangun dan mengemas **OS
tempat ia dijalankan** (Wails tidak bisa cross-compile aplikasi cgo). Prasyarat
tambahan: macOS — Xcode command line tools; Windows — MinGW-w64 gcc, NSIS
(`choco install nsis`), Git Bash, `make`; Linux —
`sudo apt install libgtk-3-dev libwebkit2gtk-4.1-dev dpkg-dev`. Di Windows tanpa
`make`, jalankan `bash scripts/package.sh` dari Git Bash (skrip yang sama).

> **Catatan Go ≥ 1.24**: CLI `wails` resmi gagal mem-parsing export data Go baru.
> Jalankan `make install-wails-fix` sekali; hasilnya di
> `~/.local/bin/wails-fixed` dan otomatis dipakai Makefile.

## 🛠 Target Makefile

Perintah harian lewat `make` (lihat `make help`):

| Perintah | Fungsi |
|---|---|
| `make run` | build & jalankan aplikasi |
| `make dev` | mode pengembangan hot-reload |
| `make build` | build cepat OS saat ini |
| `make package` | build + paket OS saat ini ke `dist/` (dmg / setup.exe + zip / deb + tar.gz) |
| `make verify` | gerbang CI: gofmt, vet, tes, typecheck (tanpa build) |
| `make build-all` | build semua platform (butuh toolchain silang; CI mengemas tiap OS secara native) |
| `make test` | unit test Go (`-race`) + typecheck frontend |
| `make check` | verifikasi pra-commit penuh (fmt, vet, test, build) |
| `make lint` | vet + typecheck + build frontend |
| `make bump patch\|minor\|major` | bump semantik (1.2.3 → 1.2.4 / 1.3.0 / 2.0.0) di `VERSION`, `wails.json`, `package.json` |
| `make bump VER=1.2.3` | set versi persis |
| `make release` | uji coba lokal: verify + package OS ini (tidak menerbitkan apa pun) |
| `make release-version VER=x.y.z` | bump lalu tulis entri CHANGELOG (lalu commit dan `make publish`) |
| `make changelog-preview` | cetak entri CHANGELOG untuk versi saat ini |
| `make changelog` | sisipkan entri itu ke `CHANGELOG.md` |
| `make tag` | buat git tag anotasi `v<versi>` pada HEAD |
| `make release-notes` | tulis `dist/BersihDisk-v<versi>-notes.md` dari entri |
| `make publish` | tag `v<versi>` lalu dorong: GitHub Actions membangun tiga OS dan membuat release |
| `make publish-local` | unggah artefak hasil build mesin ini (`dist/`) — jalur darurat |
| `make install-deps` / `make doctor` | pasang dependensi frontend / cek toolchain |
| `make install-wails-fix` | build CLI wails yang kompatibel Go ≥ 1.24 |
| `make clean` / `distclean` | bersihkan build / + `node_modules` |

**Versi tidak ada yang di-hardcode**: satu-satunya sumber adalah file **VERSION**,
diinjeksikan ke biner lewat `-ldflags "-X main.appVersion=$(VERSION)"` dan ke
metadata bundle melalui `{{.Info.ProductVersion}}` di `build/darwin/Info.plist`.

**Menerbitkan rilis yang bisa memperbarui diri:**

```bash
make bump patch        # atau minor | major, atau VER=1.2.3 — VERSION, wails.json, package.json
make publish           # release commit + tag + push; GitHub Actions sisanya
```

`make publish` mengerjakan semuanya sampai push, lalu menyerahkan ke CI:

1. `make verify` (gofmt, vet, tes, typecheck).
2. Commit rilis: bila CHANGELOG belum punya entri versi itu, atau bump belum di-commit,
   ia menjalankan `make changelog`, yang menulis entri dari subjek commit lalu
   `git commit -am "chore: release 1.0.1"`. **`-am` meng-commit semua file
   terlacak yang berubah**, jadi commit atau stash dulu pekerjaan lain
   (`make changelog COMMIT=0` menulis entri tanpa commit).
3. Tag `v1.0.1`, dorong branch dan tag. Tag itu menjalankan `release.yml`, yang
   membangun dan mengemas macOS, Windows, Linux lalu membuat release GitHub. Tidak
   ada yang diunggah dari mesinmu.
4. Memantau workflow (`WATCH=0` untuk melewatinya).

Ia menolak file untracked dan tag lokal yang tertinggal dari HEAD (`RETAG=1`
memindahkannya). Bila tag sudah ada di origin, workflow untuk tag itu dijalankan
ulang. `make publish GH=echo GIT_PUSH=echo` mencetak perintah git/gh tanpa
menjalankannya. `make release` membangun OS ini secara lokal untuk mengecek apa yang
akan dibangun CI.

**CI/CD (GitHub Actions)** di `.github/workflows/`: `ci.yml` (tiap push ke `main`
dan pull request: gofmt, vet, tes Go, typecheck di Ubuntu, macOS, Windows) dan
`release.yml` (tag `vX.Y.Z`, atau jalankan manual dengan `tag`: build + paket di
macOS, Windows, Linux lewat `scripts/package.sh`, lalu `scripts/publish.sh`
membuat release atau memperbaruinya di tempat dan menghapus asset usang).
Workflow hanya memanggil skrip repo ini, jadi hasil lokal identik.

**Kontrak penamaan.** `scripts/package.sh` menamai berkas
`BersihDisk-vX.Y.Z-<os>-<arsitektur>[-jenis].<ext>`; `internal/updater` memilih
asset dengan kata OS (`macos`/`windows`/`linux`), arsitektur (`x86_64`; build
macOS `universal` cocok untuk keduanya) dan ekstensi terbaik (`.dmg` › `.exe` ›
`.deb`). `TestPickAssetMatchesPackagedNames` gagal bila keduanya tidak sinkron.

**Pembaruan dalam aplikasi** hanya mengunduh asset dari hasil pengecekan, lewat
HTTPS dari `github.com`/`githubusercontent.com` (tiap redirect dicek), dan menolak
memasang bila SHA-256 `digest` dari GitHub tidak cocok. Lalu ia membuka
installer — `.dmg` di macOS, installer NSIS di Windows (dengan UAC), `.deb` di Linux —
dan di macOS serta Windows aplikasi menutup diri, karena aplikasi yang sedang berjalan
tidak bisa diganti (Finder melaporkan "item sedang digunakan").

## 🏗 Arsitektur

Struktur berkas sama dengan [versi Inggris](README.md#-architecture) — nama paket,
folder, dan fungsi memang berbahasa Inggris. Alur kerjanya:

UI pilih drive & kategori → `StartScan` (goroutine async) → event
`scan:progress` / `scan:finished` → hasil dikelompokkan per kategori → pilih item
→ modal konfirmasi (trash/permanen) → `StartDelete` → event
`delete:progress` / `delete:finished` → toast ringkasan.

`internal/` berisi `rules` (definisi kategori + matcher), `drive` (deteksi drive
per platform), `scanner` (engine dua fase concurrent), `deleter` (hapus massal +
progress + batal), `browser` (daftar isi folder beserta ukuran rekursif),
`reveal` (handoff ke OS), `updater` (cek rilis, unduh, verifikasi sha256),
`appicon` (ikon Dock runtime), dan `sysinfo`. `frontend/src/` tersusun atas
`App.tsx`, `backend.ts`, panel per fitur (`categories/`, `drives/`, `scan/`,
`results/`, `update/`), `common/`, `hooks/`, `i18n/`, `lib/`, dan `styles/`.

## 🔒 Keamanan

- Scan **tidak pernah** menghapus; penghapusan hanya menerima path eksplisit hasil scan
- Skip-list proteksi: `.Trash`, `$Recycle.Bin`, `System Volume Information`, `Windows`, `Program Files`, `Library`, `/usr`, `/etc`, dll.
- Tidak mengikuti symlink (WalkDir tidak menyeberang mount/symlink)
- Nama folder generik wajib lolos filter konten
- Mode permanen disertai modal peringatan merah + konfirmasi ganda
- Kategori AI, emulator, browser, dan temp bersifat opt-in

## 🕸 Privasi

BersihDisk berjalan sepenuhnya di komputermu: tanpa akun, tanpa server, tanpa
telemetri pemakaian. Hasil scan hanya ada di memori selama sesi. Akses jaringan
hanya dipakai fitur pembaruan — satu permintaan baca ke `api.github.com` untuk
informasi rilis publik, lalu unduhan aset rilis dari `github.com`, tanpa data
pribadi. Preferensi (bahasa, tema, warna aksen, ikon, pembaruan otomatis)
disimpan di penyimpanan lokal. Teks lengkap ada di **Pengaturan → Kebijakan
privasi** dalam aplikasi.

## 🌐 Bahasa

Antarmuka tersedia dalam **Bahasa Inggris** dan **Bahasa Indonesia**, diganti
saat aplikasi berjalan dan tersimpan di penyimpanan lokal. Semua string UI hidup
di `frontend/src/i18n/locales/id.ts` dan `en.ts`; keduanya berbagi satu bentuk
sehingga TypeScript menolak kunci yang hilang atau berlebih. Kode (file, folder,
fungsi, variabel, komentar) sepenuhnya berbahasa Inggris.

Dokumentasi ikut terbagi: berkas ini adalah versi Bahasa Indonesia, dan
[README.md](README.md) adalah padanan lengkapnya dalam Bahasa Inggris.

## 🧪 Pengujian

```bash
make test                     # go test -race ./internal/... + typecheck frontend
go test ./internal/... -v     # detail tiap test
go vet ./...
```

34 unit test mencakup matcher aturan (termasuk filter konten yang melindungi
pohon sumber), deteksi drive, scanner dua fase, deleter, browser folder, dan
updater. Kategori baru diharapkan datang bersama test matcher-nya.

## 🩺 Pemecahan masalah

| Gejala | Solusi |
|---|---|
| `wails dev` gagal dengan `internal error: package "…" without types was imported` | Go ≥ 1.24 vs CLI resmi: `make install-wails-fix` |
| `make: wails: No such file` | `go install github.com/wailsapp/wails/v2/cmd/wails@v2.12.0` atau pakai hasil `make install-wails-fix` |
| macOS menolak aplikasi ("pengembang tidak dikenal") | klik kanan → *Buka*, atau `xattr -dr com.apple.quarantine /Applications/BersihDisk.app` |
| Scan melewatkan folder di drive eksternal | cek pilihan drive; walking tidak menyeberang batas mount/symlink |
| Folder yang seharusnya terdeteksi tidak muncul | filter konten menolaknya — buka issue dengan path dan isi level teratasnya |
| Port 34115 sudah dipakai | instance `wails dev` lain memegangnya; hentikan atau ganti port dev server |

## 📑 Changelog & rilis

[CHANGELOG.md](CHANGELOG.md) mencatat setiap rilis dalam format
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) — terbaru di atas — dan
proyek ini mengikuti [Semantic Versioning](https://semver.org/spec/v2.0.0.html):
**MAJOR** untuk perubahan breaking, **MINOR** untuk kemampuan baru, **PATCH**
untuk perbaikan.

Entri dibuat dari subjek commit, bukan ditulis manual jauh setelahnya — itu
sebabnya [conventional commits](CONTRIBUTING.md#8-pull-request-workflow)
dipakai:

| Subjek commit | Masuk ke |
|---|---|
| `feat:` / `feat(scope):` | `### Added` |
| `fix:` | `### Fixed` |
| `perf:` `refactor:` `docs:` `test:` `i18n:` `build:` `ci:` `chore:` `style:` | `### Changed` |
| `remove:` / `deprecate:` / `security:` | `### Removed` / `### Deprecated` / `### Security` |
| tanpa prefiks | `### Other` |
| `feat!:` / `fix(scope)!:` | ditandai `**Breaking**` di sektornya |

Setiap baris membawa short hash commit-nya, jadi entri bisa ditelusuri ke
diff-nya. Workflow rilis mengunggah **entri CHANGELOG yang sudah dikurasi** sebagai
notes release bila versinya sudah tercatat di sana, dan baru memakai hasil generate
commit bila belum.

Setiap rilis juga berupa **git tag anotasi** `vX.Y.Z` — tag itulah yang
dipakai pembaruan dalam aplikasi sebagai acuan, jadi versi tanpa tag belum
menjadi rilis.

```bash
make changelog-preview              # uji cetak: entri untuk VERSION
make changelog                      # sisipkan entri itu ke CHANGELOG.md
make tag                            # git tag -a v<versi> (make publish melakukannya)
make release                        # uji coba lokal: verify + paket OS ini
make publish                        # tag + dorong → GitHub Actions membangun dan merilis
```

`make publish` adalah satu-satunya target yang menulis ke remote: ia mendorong branch
saat commit yang di-tag belum ada di sana dan mendorong tag anotasi; release-nya
dibuat workflow dengan `--verify-tag` sehingga `gh` tidak bisa mengarang tag dari
commit yang bukan maksudmu. Target ini menolak working tree yang kotor kecuali kamu
kirim `ALLOW_DIRTY=1`.

## 🤝 Kontribusi

Laporan bug, terjemahan, kategori baru, dan pekerjaan UI sangat diterima. Baca
[CONTRIBUTING.md](CONTRIBUTING.md) untuk penyiapan lingkungan, konvensi, panduan
langkah demi langkah menambah kategori, dan aturan keamanan penghapusan yang
dipakai saat review. Jalankan `make check` sebelum push.

## 📜 Kode Etik

Semua partisipasi diatur oleh [Code of Conduct](CODE_OF_CONDUCT.md) (Contributor
Covenant 2.1, dengan ringkasan Bahasa Indonesia). Laporan pelanggaran ke
**gorbypermana@gmail.com**.

## 💛 Sponsor & donasi

BersihDisk gratis, open source, dan tanpa iklan. Donasi tidak membuka fitur apa
pun — ia membeli waktu maintainer.

[![Sponsor di GitHub](https://img.shields.io/badge/GitHub%20Sponsors-donasi-ea4aaa?style=for-the-badge&logo=github-sponsors&logoColor=white)](https://github.com/sponsors/cybersafetyid)
[![Saweria](https://img.shields.io/badge/Saweria-donasi-ff5722?style=for-the-badge)](https://saweria.co/cybersafetyid)
[![Trakteer](https://img.shields.io/badge/Trakteer-donasi-red?style=for-the-badge)](https://trakteer.id/cybersafetyid)

Crypto (alamat akan dipublikasikan resmi sebelum dipakai):

```text
BTC  · REPLACE_WITH_BTC_ADDRESS
ETH / USDT (ERC-20) · REPLACE_WITH_ETH_ADDRESS
USDT (TRC-20) · REPLACE_WITH_TRON_ADDRESS
```

## ⚖️ Lisensi

Diterbitkan di bawah [Lisensi MIT](LICENSE). Hak cipta © 2026
[Gorby Permana](https://github.com/cybersafetyid). Ikon merek pihak ketiga tetap
mengikuti merek dan lisensinya masing-masing dan hanya dipakai untuk identifikasi.
