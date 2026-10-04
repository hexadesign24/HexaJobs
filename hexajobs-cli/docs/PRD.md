# PRD — HexaJobs

Disusun dari perjalanan build 4 Okt 2026. Acuan versi, bukan chat log.

## 1. Status rilis

| Item | Status |
| ---- | ------ |
| `v1.1.1` (stabil, di `main`) | Blocky ASCII art beranda, guard pesan-ramah tanpa TTY, paket Linux (binary amd64/arm64 + `.deb`), instalasi npm (`GITHUB_TOKEN` selama repo private), `docs/GO-SETUP.md`, versi single-source. 6 asset di `releases/tag/v1.1.1`. |
| `v1.1.0` | Deprecated (pre-release). Binary dari branch, tanpa guard TTY. |
| Repo | Private. PR #1 + #2 merged. |

## 2. Prinsip yang disepakati

1. **Terminal dulu.** Masalah first-run (PATH, TTY, ukuran layar, perintah global) didahulukan dari fitur baru.
2. **Kemudahan > banyak jalur.** Satu pintu masuk yang tuntas lebih berharga dari empat jalur setengah jadi.
3. **Non-goal:** `.desktop` GUI (dikesampingkan), shell selain bash, fitur baru selama Batch T+R berjalan.

## 3. Batch T — terminal (utama)

- **T1** ✅ Pesan ramah tanpa TTY (`main.go` probe `/dev/tty`). Sudah merge.
- **T2** Ukuran sempit: `<80` kolom sidebar bergantian via Tab; `<50×16` pesan resize. Kode ada (`app.go`, `TUI.md`); wajib verifikasi + dokumentasi hasil.
- **T3** Perintah global tanpa `cd`: installer menuntaskan PATH (idempoten, bash), bukan sekadar note. Kriteria: terminal baru bisa `hexajobs --version` dari `$HOME`. Plus matriks uji lintas-direktori.
- **T4** Degradasi terminal minimal (`TERM=dumb`, `NO_COLOR=1`): verifikasi perilaku, nyatakan batas dukungan bila perlu.

## 4. Batch R — README + docs (pelengkap)

- Struktur README: Quickstart 60-detik → Instalasi (1 rekomendasi + alternatif) → Running (syarat TTY + perintah global) → Konfigurasi → tautan docs.
- Baru: `docs/INSTALL.md`, `docs/TROUBLESHOOTING.md` (tabel error→solusi berisi semua jebakan yang ditemukan: direktori kerja, TTY, `--allow-scripts`, token, PATH).
- Pindah detail engine (`README` ~100 baris) → `docs/ENGINE.md` tanpa ubah isi.
- Versi hardcoded → generik + 1 titik versi di atas.
- Semua perintah di docs diverifikasi ulang persis seperti tertulis.

## 5. Backlog (belum diputuskan)

- Core/API V2 aditif (`FetchJobsV2`, `HealthCheck`, `PulseV2`, `Purge`): minor 1.2.0 vs major 2.0.
- Release engineering: CI Actions, aset Windows, publik + `npm publish`.
- Fitur: export json/csv/md, filter-sort lanjutan, notifikasi/scheduler.
