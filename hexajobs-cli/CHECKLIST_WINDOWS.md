# Checklist Verifikasi Windows (AMD64)

Template verifikasi manual oleh pengguna untuk rilis Windows
AMD64 `hexajobs-cli`. Salin checklist ini ke issue/PR rilis
Anda dan centang satu per satu. Instalasi dilakukan **manual**
mengikuti `INSTALL.md` (bagian Windows 5 langkah) — tanpa
skrip `*.ps1`/`*.bat`/`*.cmd`.

## Informasi Lingkungan

- [ ] Versi Windows: _(mis. Windows 11 Pro 23H2, build 22631)_
- [ ] Arsitektur: `AMD64` _(cek via `systeminfo` atau Settings → System → About)_
- [ ] Terminal: _(mis. Windows Terminal 1.21 / PowerShell 5.1 / PowerShell 7.4)_
- [ ] Ukuran terminal: _(mis. 100×30; minimum 80×24)_
- [ ] Sumber biner: _(rilis `hexajobs.exe` versi ___ / build dari source Go ___)_
- [ ] Versi biner (`hexajobs.exe --version`): _(mis. `hexajobs.dev v1.1.1`)_

## Verifikasi Build / Unduhan

- [ ] Target `build-windows-amd64` tersedia: `make help` menampilkannya
      (khusus build dari source).
- [ ] Output build ada: `dist\hexajobs.exe` dan
      `dist\hexajobs.exe.sha256` (khusus build dari source).
- [ ] Pasangan rilis lengkap: `hexajobs.exe` + `hexajobs.exe.sha256`
      satu versi (khusus instalasi dari rilis).
- [ ] `GOOS=windows`, `GOARCH=amd64`, `CGO_ENABLED=0` dipakai saat build
      (khusus build dari source — lihat `Makefile`).

## Verifikasi Instalasi (5 Langkah INSTALL.md)

- [ ] Langkah 1 — Biner diperoleh (unduhan rilis / `make build-windows-amd64`).
- [ ] Langkah 2 — Checksum SHA-256 **cocok**:
      - [ ] Perintah: `Get-FileHash .\dist\hexajobs.exe -Algorithm SHA256`
      - [ ] Hash lokal: `___`
      - [ ] Hash `.sha256`: `___`
      - [ ] Keduanya sama persis (bila tidak, STOP dan unduh/build ulang).
- [ ] Langkah 3 — Biner dipindah ke folder permanen
      (mis. `%USERPROFILE%\bin\hexajobs.exe`), `Unblock-File` bila perlu.
- [ ] Langkah 4 — Folder instalasi masuk `PATH` pengguna, terminal
      dibuka ulang, `Get-Command hexajobs` menemukan biner.
- [ ] Langkah 5 — `hexajobs.exe --version` mencetak versi yang diharapkan.
- [ ] Langkah 5 — `hexajobs.exe --demo` membuka TUI DEMO tanpa request API.

## Verifikasi Fungsional TUI

- [ ] Dashboard tampil rapi (tidak ada karakter rusak / layout pecah).
- [ ] Navigasi: `Tab` pindah fokus, `↑/↓` memilih, `Enter` membuka detail.
- [ ] Pencarian demo berfungsi, `Esc`/`B` kembali ke Dashboard.
- [ ] `Ctrl+C` atau `q` keluar dengan bersih (terminal kembali normal).
- [ ] `--config <path>` custom dapat dimuat (opsional; gagal dengan pesan
      yang jelas bila file tidak valid).

## Catatan / Temuan

_Tulis temuan, screenshot, atau error message di sini:_

1. _
2. _

## Hasil Akhir

- [ ] **LULUS** — semua checklist di atas tercentang, siap rilis Windows AMD64.
- [ ] **GAGAL** — ada item belum lulus (jelaskan di Catatan/Temuan).

Nama penguji: ___ · Tanggal: ___ · Tanda tangan: ___
