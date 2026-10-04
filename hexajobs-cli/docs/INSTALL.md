# Instalasi HexaJobs (Linux & Windows)

Jalur yang disarankan: **binary rilis**. Perintah Linux di bawah dijalankan
dari direktori `hexajobs-cli/` kecuali dinyatakan lain; perintah Windows
dijalankan dari direktori yang sama di **PowerShell**. Butuh terminal
interaktif untuk TUI (min 80×24); `--version` bisa di mana saja.

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

## Jalur E — Windows (tanpa Go, tanpa skrip)

> Berlaku untuk Windows 10/11 64-bit (AMD64). Semua perintah di bawah
> dijalankan manual di **PowerShell** dari direktori `hexajobs-cli/`.
> Instalasi dilakukan manual — tanpa skrip `*.ps1`/`*.bat`/`*.cmd`
> (sesuai keputusan proyek).

### E.1 — Dapatkan biner `hexajobs.exe`

Pilih salah satu:

**A. Dari rilis (tanpa Go):** unduh dua berkas rilis ke satu folder
unduhan, misalnya `Downloads\hexajobs\`:

- `hexajobs.exe`
- `hexajobs.exe.sha256`

**B. Build dari source (perlu Go 1.24+):**

```powershell
go version
make build-windows-amd64
dir dist
```

Hasil yang diharapkan:

- `dist\hexajobs.exe`
- `dist\hexajobs.exe.sha256`

### E.2 — Verifikasi checksum SHA-256

Bandingkan hash lokal dengan isi `.sha256` (sesuaikan path bila dari
folder unduhan):

```powershell
Get-FileHash .\dist\hexajobs.exe -Algorithm SHA256
Get-Content .\dist\hexajobs.exe.sha256
```

Kedua nilai hash **harus sama persis**. Alternatif bawaan Windows
tanpa PowerShell:

```cmd
certutil -hashfile dist\hexajobs.exe SHA256
```

Jika berbeda: **jangan lanjutkan** — unduh ulang / build ulang dan
pastikan file tidak corrupt atau tertukar. Panduan lengkap 5 langkah +
troubleshooting: [INSTALL.md](../INSTALL.md#windows-amd64--instalasi-manual-5-langkah).
Checklist verifikasi: [CHECKLIST_WINDOWS.md](../CHECKLIST_WINDOWS.md).

### E.3 — Instal ke folder permanen + PATH pengguna

```powershell
New-Item -ItemType Directory -Force -Path "$env:USERPROFILE\bin"
Move-Item -Force .\dist\hexajobs.exe "$env:USERPROFILE\bin\hexajobs.exe"
Unblock-File "$env:USERPROFILE\bin\hexajobs.exe"
dir "$env:USERPROFILE\bin\hexajobs.exe"
```

Tambahkan folder instalasi ke `PATH` pengguna **satu kali secara manual**:
Settings → System → About → Advanced system settings →
Environment Variables → User variables → `Path` → New →
`C:\Users\<NamaAnda>\bin` → OK di semua dialog → **tutup dan buka
kembali** PowerShell / Windows Terminal agar `PATH` baru terbaca.
Cek: `$env:Path -split ";"` lalu `Get-Command hexajobs`.

### E.4 — Verifikasi instalasi

```powershell
hexajobs.exe --version
hexajobs.exe --demo
```

Hasil yang diharapkan: `--version` mencetak versi (mis.
`hexajobs.dev v1.1.1`); `--demo` membuka TUI contoh offline bertanda
DEMO (keluar dengan `Ctrl+C` atau `q` dari Dashboard). Butuh terminal
interaktif (Windows Terminal, min 80×24; di bawah 50×16 hanya tampil
permintaan resize).

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
