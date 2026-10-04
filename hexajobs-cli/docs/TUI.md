# Terminal UI, Security Shield & Interaction

Entry point: `cmd/hexajobs/main.go`. Implementasi memakai Bubble Tea v1.3.10, Lip Gloss v1.1.0, dan Bubbles v0.21.0, dipin di `go.mod`/`go.sum`. Versi minimum Go naik menjadi 1.24 untuk dependensi tersebut. Engine Peran 1 tidak diubah.

## Menjalankan

```sh
go run ./cmd/hexajobs
go run ./cmd/hexajobs --demo
go run ./cmd/hexajobs --config /path/to/config.json
```

Inisialisasi konfigurasi engine, pembuatan engine, panggilan eksplisit `PurgeExpiredCache`, pemuatan preferensi UI, dan pembacaan ringkasan pasar berjalan dalam `Init` → `tea.Cmd`. Konfigurasi harus dimuat sebelum engine dapat dibuat. `NewModel` sendiri tidak melakukan I/O. Bubble Tea menggunakan alternate screen, renderer 30 FPS, dan clipping berdasarkan lebar sel terminal. Ukuran utama yang diuji: 80×24, 100×30, 120×40; di bawah 80 kolom sidebar bergantian dengan konten saat Tab, dan di bawah 50×16 tampil permintaan resize.

## Navigasi

| Tombol | Aksi |
| --- | --- |
| Ctrl+C | Keluar dari mana saja |
| q | Keluar saat fokus menu di Dashboard; tetap menjadi teks di input pencarian |
| Tab | Pindah fokus sidebar/konten |
| ↑/↓ atau k/j | Pilih menu, lowongan, atau preferensi; tetap menjadi teks saat mengetik |
| Enter | Pilih menu, cari, atau buka detail |
| Esc | Tutup panel detail/pitch; jika tidak ada panel, kembali ke Dashboard dan batalkan fetch |
| A | Buka URL lowongan terpilih, atau link sponsor di Donate jika dikonfigurasi |
| P | Buat dan tampilkan draft pitch lokal |
| S | Simpan listing dalam history |
| B | Kembali dari hasil, history, radar, atau panel |
| R | Muat ulang hasil, history, atau radar |
| ←/→ atau Enter | Ubah field Settings yang dipilih |

Menu `Web3 & Bounty` memakai dua panggilan kontrak engine (`Web3`, lalu `Bounty`), tanpa implementasi ingestion tambahan. History menampilkan event terbaru terlebih dahulu dan mendukung inspect, pitch, save, serta buka link. Membuka link hanya mencatat `opened`, tidak pernah mengklaim lamaran telah dikirim.

Fetch memiliki context 60 detik; browser helper memiliki context 15 detik. Request lama dibatalkan pada navigasi dan response dengan ID lama diabaikan. Hasil parsial tetap ditampilkan bersama status kegagalan. Semua akses disk, request engine, dan peluncuran browser berada dalam `tea.Cmd`; `View` hanya merender state.

## Preferensi dan riwayat

- `~/.config/hexajobs/ui.json`: bahasa `en`, `id`, `jp`, region `global`, `us`, `eu`, `asia`, `id`, dan toggle ScamShield. Header, menu, judul view, serta label utama diterjemahkan; penjelasan teknis dan status diagnostik menggunakan bahasa Inggris.
- `~/.cache/hexajobs/history.json`: array event `{job, action, at}` untuk `saved`, `opened`, dan `pitch`. Metadata deskripsi dipertahankan agar listing tersimpan tetap bisa diperiksa ScamShield.
- `XDG_CONFIG_HOME`/`XDG_CACHE_HOME` absolut dihormati. UI tidak menulis `config.json` milik engine. File history bukan nama hash cache engine dan tidak menjadi sasaran purge engine.
- Penulisan atomik via temporary file dan rename; mode `0600`/direktori `0700` pada sistem Unix, ACL direktori pada Windows. Log tidak menulis stdout/stderr. Error di UI masuk status bar.
- Update preferensi yang beruntun dikoordinasikan agar perubahan terakhir tidak ditimpa save lama. Logger menyerialkan append dalam satu proses; beberapa proses CLI dengan history yang sama belum memakai lock lintas proses. Batas history 32 MiB; file corrupt/penuh dipertahankan dan menghasilkan error, bukan dihapus.

## ScamShield dan Anti-Ghosting

`security.ValidateJobListing` memindai judul, perusahaan, deskripsi, dan teks kompensasi menggunakan aturan lokal. Indikator seperti Telegram-only, biaya pendaftaran di muka, penyewaan rekening bank, dan pay-to-apply menghasilkan RED. WhatsApp interview atau penyebutan kredensial wallet menghasilkan YELLOW. Data entry dengan kompensasi eksplisit per jam di atas USD 250 menghasilkan RED; gaji per tahun dan reward per task tidak diperlakukan sebagai rate per jam.

Risiko lebih tinggi dari engine tetap dipertahankan. Mock dan kurs offline/stale diberi caution. Label hijau `Verified*` hanya berarti tidak ada indikator heuristik yang cocok; detail menjelaskan bahwa ini bukan verifikasi identitas, audit kontrak, atau jaminan keamanan. Heuristik dapat menghasilkan false positive, termasuk teks yang sedang memperingatkan tentang scam.

Saat ScamShield aktif, `A` diblokir untuk RED dan detail memperlihatkan alasannya. Toggle hanya mematikan blokir ScamShield, tidak validasi URL atau Anti-Ghosting. Mock tidak bisa dipakai untuk membuka lamaran. Skema selain HTTP(S), URL berkredensial, dan karakter kontrol ditolak. Browser dipanggil dengan argumen terpisah tanpa shell; helper Windows tidak membuka console tambahan.

Anti-Ghosting menyembunyikan listing dengan `GhostingRate > 0.75`; tepat 0.75 tetap ditampilkan. Nilai tidak diketahui (-1/NaN) tetap tampil. Riwayat tersimpan merupakan catatan tindakan lampau, bukan pencarian baru, sehingga tidak dibuang oleh filter ini.

Teks eksternal dibersihkan dari ANSI/OSC, karakter kontrol, dan karakter arah tak terlihat sebelum dirender atau masuk draf pitch.

## Batas data yang ditampilkan

Win-rate dan skill match langsung berasal dari engine. Badge `Fair` hanya membandingkan nilai dengan `MarketReport.AverageRate` ketika periodenya sama; tidak ada annualisasi, konversi Forex, atau scoring tambahan di UI. Data mock/offline ditandai indicative. Rata-rata keseluruhan sampel bukan penetapan upah layak atau rekomendasi rate individual.

Radar mengelompokkan jumlah lowongan non-mock yang diterima berdasarkan teks region (US, EU, Asia, ID). Bar relatif terhadap jumlah terbanyak dalam sampel itu. ID termasuk Asia; Worldwide/unknown tidak digandakan ke setiap wilayah. Label rate adalah rata-rata sampel pasar dari engine. `EngineContract` belum menyediakan tren/rate per region, sehingga UI tidak menampilkan angka regional buatan. Mode demo mengosongkan bar karena semua listing adalah mock.

Region EU/Asia memakai hasil global engine lalu pengelompokan lokasi untuk tampilan; US/ID diteruskan sebagai filter engine. Kelengkapan lokasi masih bergantung pada sumber.

`BuildPitch` adalah **template lokal**, tanpa API LLM. Draf mencantumkan kebutuhan skill, indikator kecocokan yang tersedia, anggaran listing beserta periodenya, dan placeholder bukti pengalaman serta nama. Tidak mengarang pengalaman, tidak memakai rate pasar yang tidak tersedia, dan tidak mengirim pesan.

Donate tidak memuat alamat sponsor rekaan. Maintainer dapat mengisi `HEXAJOBS_SPONSOR_URL` dengan URL HTTP(S) resminya; jika kosong, view menampilkan pilihan kontribusi nonfinansial.

## Pengujian

```sh
go test -race ./...
go vet ./...
```

Test mencakup rules ScamShield, batas ghosting, struktur pitch, escape-sequence sanitization, argumen launcher Windows/Linux/macOS tanpa membuka browser, append history, preferences, command async, partial results, stale responses, navigasi keyboard, serta clipping semua view dalam tiga bahasa. Backend tests tetap berjalan tanpa modifikasi.

## Kompatibilitas terminal (hasil uji v1.1.1)

- Ukuran teruji otomatis: 80×24, 100×30, 120×40 (`TestAllViewsStayWithinTerminalBounds`, 3 bahasa). Di bawah 80 kolom sidebar bergantian via Tab; di bawah 50×16 tampil permintaan resize. Uji visual manual di terminal asli tetap disarankan tiap rilis.
- `TERM=dumb` dan `NO_COLOR=1`: seluruh test render lolos — Lip Gloss degradasi otomatis tanpa warna; layout dan clipping tidak rusak. Batas dukungan: warna penuh butuh terminal 256-warna/truecolor.
- Wajib terminal interaktif (`/dev/tty`). Tanpa TTY (pipe/redirect), CLI keluar cepat dengan pesan `need an interactive terminal...`; `--version` tetap bisa di mana saja.

Referensi: [Bubble Tea v1](https://github.com/charmbracelet/bubbletea/tree/v1.3.10), [Bubbles](https://github.com/charmbracelet/bubbles/tree/v0.21.0), [Lip Gloss](https://github.com/charmbracelet/lipgloss/tree/v1.1.0).
