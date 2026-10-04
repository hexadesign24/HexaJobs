# hexajobs.dev CLI — v1.1.1

[![CI](https://github.com/hexadesign24/HexaJobs/actions/workflows/ci.yml/badge.svg)](https://github.com/hexadesign24/HexaJobs/actions/workflows/ci.yml)

Agregator lowongan kerja bertenaga TUI: RemoteOK, Remotive, Jobicy, Adzuna,
Hacker News Who's Hiring, forum RSS, GitHub bounty, dan feed Web3 — plus
ScamShield, skor kecocokan skill, dan market pulse. Linux (amd64/arm64) dan
Windows. Butuh **Go 1.24+** hanya untuk build dari source.

> Versi di atas adalah satu-satunya titik versi dokumen ini; nama file
> (`dist/*.deb`, `*.tgz`) mengikuti pola generik.

## Quickstart 60 detik (Linux, tanpa build)

```sh
curl -fsSL https://raw.githubusercontent.com/hexadesign24/HexaJobs/main/hexajobs-cli/scripts/quickstart.sh | sh
```

Hasil yang benar: diakhiri `hexajobs.dev vX.Y.Z` + pesan `Berhasil`.
Lalu di **terminal baru** (butuh terminal interaktif min 80×24), dari
direktori mana pun:

```sh
hexajobs --demo          # coba tanpa network
hexajobs                 # mode live
```

Quickstart mengunduh binary rilis yang cocok (amd64/arm64), memasang ke
`~/.local/bin`, dan menuntaskan PATH otomatis. Butuh `curl`; bila repo
memerlukan auth, set `GITHUB_TOKEN` + pasang `gh` CLI dulu. Alternatif
tanpa `curl|sh`, dari clone lokal: `./scripts/quickstart.sh`.
Detail tiap jalur: [INSTALL.md](docs/INSTALL.md).

## Instalasi

| Jalur | Perintah | Keterangan |
| ----- | -------- | ---------- |
| Binary (disarankan) | `make build-linux-*` + `./scripts/install-linux.sh` | `~/.local/bin`, PATH otomatis |
| .deb | `make deb` + `sudo dpkg -i dist/*.deb` | Debian/Ubuntu amd64 |
| npm | `npm install -g hexajobs-cli --allow-scripts=hexajobs-cli` | Node 18+ |
| Source | `go run ./cmd/hexajobs --demo` | Tanpa instal, butuh Go 1.24+ |

Semua perintah dijalankan dari direktori `hexajobs-cli/`.
Bermasalah? Lihat [INSTALL.md](docs/INSTALL.md) dan
[TROUBLESHOOTING.md](docs/TROUBLESHOOTING.md).

## Menjalankan

```sh
hexajobs --demo                  # contoh offline, tanpa request API
hexajobs                         # live
hexajobs --config /path/config.json
hexajobs --version
```

Wajib terminal interaktif (`/dev/tty`) — tanpa TTY (pipe/redirect) CLI
keluar cepat dengan pesan yang jelas. Navigasi: `Tab` menu/konten,
`↑/↓` atau `k/j` pilih, `Enter` cari/buka, `A` buka link, `P` draft pitch,
`S` simpan, `B`/`Esc` kembali, `Ctrl+C` keluar. Lengkapnya:
[panduan TUI](docs/TUI.md).

Windows (tanpa Go): unduh `.exe` dari Releases lalu ikut
[panduan Windows](INSTALL.md#windows-amd64--instalasi-manual-5-langkah).
Dari source: `go build -o bin/hexajobs.exe ./cmd/hexajobs`, lalu
`bin\hexajobs.exe --demo` di Windows Terminal.

## Konfigurasi singkat

```sh
mkdir -p ~/.config/hexajobs
cp config.example.json ~/.config/hexajobs/config.json
```

Env opsional: `ADZUNA_APP_ID`, `ADZUNA_APP_KEY`, `GITHUB_TOKEN`.
Tanpa kredensial Adzuna, sumber regional memakai mock berlabel jelas.
Cache di `~/.cache/hexajobs/`.

## Untuk developer

```sh
go vet ./...
go test ./... -count=1
```

Versi diinject via ldflags (`-X .../views.AppVersion=...`), default nilainya
sama dengan rilis saat ini. Referensi teknis engine (kontrak, pipeline,
cache, skor, FX, market pulse): [ENGINE.md](docs/ENGINE.md).

## Dokumentasi

- [PRD](docs/PRD.md) — status rilis dan rencana
- [INSTALL.md](docs/INSTALL.md) — instalasi per jalur
- [TROUBLESHOOTING.md](docs/TROUBLESHOOTING.md) — tabel error → solusi
- [TUI.md](docs/TUI.md) — navigasi, keamanan, preferensi, kompatibilitas terminal
- [GO-SETUP.md](docs/GO-SETUP.md) — penyiapan Go
- [ENGINE.md](docs/ENGINE.md) — referensi teknis engine
