# Panduan Instalasi HexaJobs CLI (INSTALL)

Panduan instalasi manual untuk `hexajobs-cli`. Tidak ada skrip
otomatis (`*.ps1`/`*.bat`/`*.cmd`) — semua langkah di bawah
dilakukan manual oleh pengguna.

- Butuh **Go 1.24+** hanya jika Anda membangun dari source.
- Biner rilis tidak memerlukan Go, hanya OS yang didukung.
- Konfigurasi engine: `~/.config/hexajobs/config.json`
  (`%USERPROFILE%\.config\hexajobs\config.json` di Windows,
  `XDG_CONFIG_HOME` absolut dihormati).
- Cache: `~/.cache/hexajobs/` (`XDG_CACHE_HOME` absolut dihormati).

## macOS (Apple Silicon / ARM64)

```sh
make build-darwin-arm64
./dist/hexajobs-darwin-arm64 --version
./dist/hexajobs-darwin-arm64 --demo
```

Pindahkan biner ke direktori di `PATH` Anda, misalnya
`/usr/local/bin/hexajobs`, lalu `chmod +x` bila diperlukan.
Verifikasi checksum: `shasum -a 256 -c dist/hexajobs-darwin-arm64.sha256`.

## Linux (AMD64)

```sh
make build-linux-amd64
./bin/hexajobs-linux-amd64 --version
./bin/hexajobs-linux-amd64 --demo
```

Pindahkan biner ke direktori di `PATH` Anda, misalnya
`~/.local/bin/hexajobs`, lalu `chmod +x` bila diperlukan.
Verifikasi checksum: `sha256sum -c bin/hexajobs-linux-amd64.sha256`.
Untuk instalasi lengkap (.deb, npm, PATH otomatis), lihat
[docs/INSTALL.md](docs/INSTALL.md).

## Windows (AMD64) — Instalasi Manual 5 Langkah

> Berlaku untuk Windows 10/11 64-bit (AMD64). Semua perintah
> di bawah dijalankan manual di **PowerShell**. Jangan gunakan
> skrip otomatis.

### Langkah 1 — Dapatkan biner `hexajobs.exe`

Pilih salah satu:

**A. Dari rilis (tanpa Go):** unduh dua berkas rilis ke satu
folder unduhan, misalnya `Downloads\hexajobs\`:

- `hexajobs.exe`
- `hexajobs.exe.sha256`

**B. Build dari source (perlu Go 1.24+):** dari root repositori
`hexajobs-cli`:

```powershell
go version
make build-windows-amd64
dir dist
```

Hasil yang diharapkan di `dist\`:

- `dist\hexajobs.exe`
- `dist\hexajobs.exe.sha256`

### Langkah 2 — Verifikasi checksum SHA-256

Masih di PowerShell, bandingkan hash lokal dengan isi `.sha256`.
Contoh bila berkas ada di `dist\` (sesuaikan path bila dari
folder unduhan):

```powershell
Get-FileHash .\dist\hexajobs.exe -Algorithm SHA256
Get-Content .\dist\hexajobs.exe.sha256
```

Kedua nilai hash **harus sama persis**. Alternatif bawaan
Windows tanpa PowerShell:

```cmd
certutil -hashfile dist\hexajobs.exe SHA256
```

Jika berbeda: **jangan lanjutkan**. Unduh ulang / build ulang
dan pastikan file tidak corrupt atau tertukar (`*.exe` dengan
versi yang sesuai di `*.sha256`).

### Langkah 3 — Pindahkan ke folder instalasi

Buat folder instalasi permanen (bukan folder unduhan sementara),
misalnya `%USERPROFILE%\bin`, lalu pindahkan biner ke sana dan
buka blokir keamanan Windows bila diminta:

```powershell
New-Item -ItemType Directory -Force -Path "$env:USERPROFILE\bin"
Move-Item -Force .\dist\hexajobs.exe "$env:USERPROFILE\bin\hexajobs.exe"
Unblock-File "$env:USERPROFILE\bin\hexajobs.exe"
dir "$env:USERPROFILE\bin\hexajobs.exe"
```

> Catatan SmartScreen: pada peluncuran pertama Windows mungkin
> menampilkan peringatan SmartScreen untuk biner yang baru
> diunduh. Pilih opsi untuk melihat detail dan jalankan hanya
> jika hash pada Langkah 2 sudah cocok.

### Langkah 4 — Tambahkan folder instalasi ke `PATH`

Agar perintah `hexajobs` dapat dijalankan dari terminal mana
pun, tambahkan folder instalasi ke `PATH` pengguna secara
manual (satu kali, tanpa skrip):

1. Buka **Settings → System → About → Advanced system settings
   → Environment Variables**.
2. Pada bagian **User variables**, pilih `Path` → **Edit** →
   **New**, lalu isi `C:\Users\<NamaAnda>\bin`
   (sesuai folder pada Langkah 3).
3. Klik **OK** di semua dialog, lalu **tutup dan buka kembali**
   PowerShell / Windows Terminal agar `PATH` baru terbaca.

Alternatif manual via PowerShell (tetap manual, bukan skrip
instalasi — hanya mengatur `PATH` pengguna):

```powershell
[Environment]::SetEnvironmentVariable(
  "Path",
  $env:Path + ";$env:USERPROFILE\bin",
  "User"
)
```

Tutup lalu buka kembali PowerShell, kemudian cek:

```powershell
$env:Path -split ";"
Get-Command hexajobs
```

### Langkah 5 — Verifikasi instalasi

Jalankan versi dan mode demo offline (tanpa request API):

```powershell
hexajobs.exe --version
hexajobs.exe --demo
```

Hasil yang diharapkan:

- `--version` mencetak versi, misalnya `hexajobs.dev v1.1.1`.
- `--demo` membuka TUI dengan contoh bertanda DEMO. Keluar
  dengan `Ctrl+C` atau `q` dari menu Dashboard.

Jika perintah tidak ditemukan, ulangi Langkah 4 (pastikan
terminal sudah dibuka ulang setelah mengubah `PATH`). Isi
hasil verifikasi Anda ke `CHECKLIST_WINDOWS.md`.

## Troubleshooting Windows

| Gejala | Penyebab umum / Solusi |
| --- | --- |
| `Get-Command hexajobs` tidak ditemukan | `PATH` belum termuat ulang — tutup/buka kembali terminal; pastikan path folder benar. |
| Hash tidak cocok | File corrupt/tertukar — unduh ulang pasangan `.exe` + `.sha256` yang satu versi. |
| SmartScreen memblokir | Wajar untuk biner baru — lanjutkan hanya jika hash cocok; gunakan `Unblock-File`. |
| TUI tidak tampil rapi | Gunakan Windows Terminal terbaru; ukuran minimum 80×24 (ideal 100×30 atau 120×40). |
| `go test -race` gagal di Windows | Race detector butuh compiler C yang kompatibel — gunakan `go test ./... -cover` atau instal toolchain C. |

Referensi lanjutan: [README](README.md), [panduan TUI](docs/TUI.md),
[config.example.json](config.example.json).
