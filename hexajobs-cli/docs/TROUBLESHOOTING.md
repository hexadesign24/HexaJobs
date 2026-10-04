# Troubleshooting HexaJobs

Gejala → penyebab → solusi. Semua path relatif ke `hexajobs-cli/`.

| # | Gejala | Penyebab umum | Solusi |
| - | ------ | ------------- | ------ |
| 1 | `go: perintah tidak ditemukan` | `bin/` Go belum di PATH | Lihat [GO-SETUP.md](GO-SETUP.md); `source ~/.bashrc` atau buka terminal baru |
| 2 | `hexajobs: command not found` setelah install | `~/.local/bin` belum di PATH | Installer terbaru menambahkannya otomatis — `source ~/.bashrc` / terminal baru; atau `./scripts/install-linux.sh --system` |
| 3 | `could not open a new TTY` (versi lama) / `need an interactive terminal` | Dijalankan tanpa TTY (pipe, redirect, panel non-terminal) | Jalankan di terminal asli; `--version` bisa di mana saja |
| 4 | `No such file or directory` untuk `go run ./cmd/...`, `make`, `./bin/...` | Direktori kerja bukan `hexajobs-cli/` | `cd` ke `hexajobs-cli/` dulu (perintah relatif terhadap modul Go) |
| 5 | `missing bin/hexajobs-linux-* - run 'make build-linux-*' first` | Binary belum di-build | `make build-linux-amd64` (atau `arm64`) dulu |
| 6 | Postinstall npm tidak mengunduh (hening) | npm 10+ memblokir script | Tambah `--allow-scripts=hexajobs-cli` |
| 7 | Postinstall: `HTTP 404` | Tag/asset belum ada, atau kena rate-limit API | Pastikan tag + asset ada; coba lagi nanti atau sertakan `GITHUB_TOKEN` |
| 8 | Wrapper: `native binary not found` | Binary tidak terunduh & tidak ada hasil build | `HEXAJOBS_BIN=$PWD/bin/hexajobs-linux-amd64 hexajobs --demo`, atau build dari source |
| 9 | `Engine unavailable` / `Engine is not ready` | Config invalid / bootstrap gagal | Cek JSON config (lihat `config.example.json`), lalu coba `--demo` untuk isolasi |
| 10 | Hasil regional kosong / mock semua | `ADZUNA_APP_ID` kosong (mock by design) atau negara tak didukung | Isi kredensial Adzuna; negara di luar daftar → sumber lain tetap jalan |
| 11 | `ScamShield blocked this risky link` | Link kena aturan RED | `Enter` untuk inspect alasan; ini proteksi, bukan error |
| 12 | Tampilan rusak di terminal kecil | Di bawah 80 kolom / 50×16 | Perbesar ke min 80×24; logo otomatis sembunyi di layar sempit |
| 13 | Warna hilang (`TERM=dumb`, `NO_COLOR`) | Degradasi otomatis Lip Gloss | Normal dan didukung; butuh warna penuh → pakai terminal 256-warna/truecolor |
| 14 | `dpkg` konflik / permission cache | Instal ganda atau direktori milik root | `sudo dpkg -i --force-overwrite` bila perlu; cache/config harus milik user (`0600`/`0700`) |
