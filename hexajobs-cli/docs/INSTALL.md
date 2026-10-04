# Instalasi HexaJobs (Linux)

Jalur yang disarankan: **binary rilis**. Semua perintah di bawah dijalankan
dari direktori `hexajobs-cli/` kecuali dinyatakan lain. Butuh terminal
interaktif untuk TUI; `--version` bisa di mana saja.

## Prasyarat

- Terminal min 80×24 (di bawah 50×16 hanya tampil permintaan resize).
- Untuk build dari source: Go 1.24+ (lihat [GO-SETUP.md](GO-SETUP.md)).
- Untuk npm: Node 18+.

## Jalur A — binary rilis (disarankan)

```sh
make build-linux-amd64     # amd64; arm64: make build-linux-arm64
make checksum              # verifikasi: sha256sum -c bin/*.sha256
./scripts/install-linux.sh # -> ~/.local/bin/hexajobs + PATH otomatis
source ~/.bashrc           # atau buka terminal baru
hexajobs --version
hexajobs --demo
```

Installer menambahkan `~/.local/bin` ke PATH di `~/.bashrc` bila belum ada
(idempoten). Alternatif sistem: `./scripts/install-linux.sh --system`
(→ `/usr/local/bin`, perlu sudo).

Uji perintah global (tanpa `cd` ke repo):

```sh
cd /tmp && hexajobs --version
cd ~ && hexajobs --demo
```

## Jalur B — paket .deb (Debian/Ubuntu amd64)

```sh
make deb
sudo dpkg -i dist/hexajobs-cli_v*.deb
hexajobs --version
```

## Jalur C — npm (Linux x64/arm64, Node 18+)

```sh
npm install -g hexajobs-cli --allow-scripts=hexajobs-cli
hexajobs --demo
```

- `--allow-scripts` wajib (npm 10+ memblokir postinstall secara default).
- Postinstall mengunduh binary dari GitHub Releases, atau build dari source
  bila Go + Make tersedia. `GITHUB_TOKEN` opsional (membantu bila kena
  rate-limit API GitHub).
- Belum publish ke registry? Instal dari tarball: `npm pack`, lalu
  `npm install -g ./hexajobs-cli-*.tgz --allow-scripts=hexajobs-cli`.
- Variabel bantu: `HEXAJOBS_BIN=/path/ke/binary` (paksa binary tertentu),
  `HEXAJOBS_SKIP_DOWNLOAD=1` (lewati unduhan postinstall, mis. offline).

## Jalur D — dari source (dev)

```sh
go run ./cmd/hexajobs --demo  # tanpa network
go run ./cmd/hexajobs         # live
go build -o bin/hexajobs ./cmd/hexajobs
```

## Konfigurasi pertama (mode live)

```sh
mkdir -p ~/.config/hexajobs
cp config.example.json ~/.config/hexajobs/config.json
```

Opsional: `ADZUNA_APP_ID`, `ADZUNA_APP_KEY`, `GITHUB_TOKEN` sebagai env.
Tanpa kredensial Adzuna, sumber regional memakai mock berlabel jelas.
Custom config: `hexajobs --config /path/config.json`. Cache di
`~/.cache/hexajobs/`.

Lihat [masalah umum](TROUBLESHOOTING.md) bila ada error, dan
[panduan TUI](TUI.md) untuk navigasi.
