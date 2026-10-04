# hexajobs.dev CLI

Backend Peran 1 dan TUI Peran 2 terintegrasi. Memerlukan **Go 1.24+** untuk dependensi Charmbracelet. Package `core`, `engine`, `scraper`, dan `models` tetap menggunakan standard library; antarmuka dan aksi pengguna berada di `ui`, `security`, dan `client`.

```sh
go run ./cmd/hexajobs
go run ./cmd/hexajobs --demo
go build -o bin/hexajobs ./cmd/hexajobs
```

Windows: gunakan `go build -o bin/hexajobs.exe ./cmd/hexajobs`. Mode `--demo` memakai contoh bertanda DEMO tanpa request API. `--config <path>` memilih konfigurasi engine; `--version` menampilkan versi. Lihat [panduan TUI](docs/TUI.md) untuk navigasi, keamanan, preferensi, dan riwayat.

## Integrasi dengan Peran 2

Package `internal/models` menyediakan `JobListing`, `MarketReport`, dan kontrak yang diminta:

```go
type EngineContract interface {
    FetchJobs(ctx context.Context, keyword, region, category string) ([]JobListing, error)
    GetMarketPulse(sector string) MarketReport
    PurgeExpiredCache() int
}
```

Contoh inisialisasi dari package dalam modul ini:

```go
cfg, err := core.LoadConfig("")
if err != nil { return err }
backend, err := core.NewEngine(cfg, core.Options{})
if err != nil { return err }
var api models.EngineContract = backend

ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
defer cancel()
jobs, fetchErr := api.FetchJobs(ctx, "golang", "id", "IT")
// jobs tetap dapat dipakai ketika fetchErr berupa *core.FetchError.
// FetchError.Failures memberi rincian kegagalan tiap sumber/cache/forex.
report := api.GetMarketPulse("IT")
_ = jobs
_ = fetchErr
_ = report
```

Kategori: `IT`, `Non-IT`, `Web3`, `Bounty`; kosong atau `all` berarti semua. Pencarian keyword memakai token AND dengan alias skill sederhana; `go` tidak cocok dengan `django`. Region menerima teks lokasi atau kode negara dua huruf; listing `Worldwide` tetap cocok. Kode dua huruf juga memilih negara Adzuna; jika API Adzuna tidak mendukung negara itu, sumber lain tetap dapat menghasilkan data. Region dari teks komunitas dapat tidak lengkap dan bukan jaminan kelayakan pelamar.

`Options.Hub`, `Options.HTTPClient`, `Options.ForexEndpoint`, dan `Options.Now` menyediakan dependency injection untuk test atau adapter tambahan. Pemanggilan `FetchJobs` bersamaan aman; satu batch berjalan per engine, sementara sumber dalam batch memakai worker pool. Pemanggil harus menyediakan context dengan deadline sesuai kebutuhan. `GetMarketPulse` membaca observasi lokal terakhir dan tidak memulai request jaringan.

## Konfigurasi dan cache

`LoadConfig("")` membaca `~/.config/hexajobs/config.json`; jika belum ada, nilai default dipakai. `XDG_CONFIG_HOME` dan `XDG_CACHE_HOME` absolut dihormati, termasuk pada Windows. Cache default berada di `~/.cache/hexajobs/`. Lihat [config.example.json](config.example.json); field yang tidak diisi memakai default. Konfigurasi tidak valid menghasilkan error.

Environment: `ADZUNA_APP_ID`, `ADZUNA_APP_KEY`, `GITHUB_TOKEN`. Token GitHub tidak diserialisasi. `SaveConfig` hanya menulis jika dipanggil secara eksplisit; file konfigurasi dan cache menggunakan mode `0600`, direktori `0700` pada OS yang mendukung permission Unix. Proteksi Windows mengikuti ACL direktori pengguna.

Cache menyimpan listing sebelum personalisasi; skill dan skor dihitung ulang pada setiap fetch. Nama file adalah SHA-256 dari sumber/query, penulisan memakai temporary file dan rename. TTL default seluruh feed **30 menit**; keyword disaring lokal untuk feed penuh sehingga pencarian baru tidak menambah request saat cache masih valid. Cache rusak atau kedaluwarsa menjadi miss. Kegagalan sumber tidak dicache sebagai hasil sukses. Tidak ada polling otomatis; pemanggil mengatur frekuensi pembaruan sesuai kebijakan sumber, termasuk anjuran Remotive hingga empat request sehari.

Auto-purge dijalankan saat startup dan fetch; hanya file cache milik engine dan temporary file miliknya yang berumur **lebih dari 7 hari** yang dihapus. Tidak rekursif dan tidak mengikuti symlink. Untuk proses panjang, pemanggil dapat menjalankan `go backend.RunAutoPurge(ctx, time.Hour)`; context mengendalikan penghentiannya. Tidak ada goroutine latar yang hidup tanpa kendali pemanggil.

## Pipeline sumber

| Jalur | Perilaku |
| --- | --- |
| Remote global | RemoteOK → Remotive → Jobicy; beralih jika error atau feed kosong; mempertahankan hasil parsial dan rincian error |
| Regional | Adzuna halaman pertama, maksimal 50 hasil; mock JSON jika `app_id` kosong; `app_key` wajib jika ID diisi |
| Komunitas | Firebase user `whoishiring`, pilih thread hiring terbaru dari maksimal 30 submission; maksimal 100 komentar tingkat pertama secara default |
| Forum publik | URL RSS 2.0 yang dikonfigurasi dalam `forum_feed_urls`, maksimal 500 item |
| Bounty | GitHub search publik untuk kedua label `bounty` dan `paid-task`, maksimal 100 issue per label, deduplikasi dan pengabaian pull request |
| Web3 | Curated JSON yang dikonfigurasi; mock deterministik bila URL kosong; error feed live tidak diam-diam diganti mock |

Semua request memakai `User-Agent: hexajobs-cli/1.0`, context, timeout, batas response 16 MiB, dan semaphore HTTP bersama. Circuit breaker terbuka setelah tiga kegagalan, cooldown 60 detik, dengan satu probe pemulihan. HTTP 429/403 langsung membuka circuit; `Retry-After` dihormati sampai batas 24 jam. Pembatalan pemanggil tidak dihitung sebagai kegagalan sumber. Redirect lintas host yang membawa kredensial dan downgrade HTTPS ditolak. Pesan error transport tidak memuat URL berkredensial.

`Platform` dan URL listing asal dipertahankan agar Peran 2 dapat memberi atribusi sumber. Mock diberi `IsMock=true` dan platform `adzuna-mock` atau `web3-mock`; tanggal sampel tetap, tidak direkayasa menjadi tanggal hari ini.

## Kompensasi, risiko, dan skor

`CompensationUSD` menyimpan midpoint rentang dalam USD **dengan periode asal**, ditandai `CompensationPeriod`: `hour`, `day`, `week`, `month`, `year`, `task`, atau `unknown`. Nol berarti tidak diketahui/tidak dapat dikonversi. `CompensationRaw` mempertahankan teks sumber. Parser memerlukan mata uang eksplisit; dolar tanpa prefiks dianggap USD sesuai konvensi feed Remotive. Parser free text bersifat heuristik, bukan parser seluruh format gaji global.

Endpoint Frankfurter yang diminta digunakan untuk kurs USD; ini kurs referensi harian, bukan tick pasar intraday. Kurs dicache 30 menit. Pada kegagalan, gunakan kurs terakhir atau tabel offline ilustratif: 1 USD = 0.92 EUR, 150 JPY, 16,000 IDR, 0.79 GBP. `ForexSource` menandai `native`, `frankfurter`, `frankfurter-stale`, `offline`, atau `unsupported`. Mata uang lain hanya dikonversi jika tersedia dalam response live; token crypto tidak diasumsikan setara USD. Reward testnet yang tidak guaranteed tetap bernilai tak diketahui.

Skor deterministik: **60% Jaccard skill + 15% cosine frekuensi token + 10% kompensasi terhadap target + 10% freshness + 5% bukti respons**, lalu diskon risiko (`YELLOW` × 0.85, `RED` × 0.5), dan pemetaan ke **5–98**. Cosine memakai token dan alias, bukan embedding atau model bahasa. Normalisasi gaji menggunakan 8 jam/hari, 40 jam/minggu, 2,080 jam/tahun. Reward per task tidak diasumsikan memiliki durasi kerja tertentu. `SkillsMatch` berada di [0,1], kosong menghasilkan 0. `WinRate` adalah **heuristik peringkat, bukan probabilitas diterima yang terkalibrasi**.

`GhostingRate=-1` berarti tidak ada bukti; sumber yang tersedia tidak memberi statistik respons pelamar sehingga engine tidak membuat angka ghosting. `GREEN` Web3 hanya berarti feed menyatakan audit passed dan reward guaranteed; bukan audit mandiri atau jaminan keamanan. Audit gagal atau kewajiban deposit menghasilkan `RED`, ketidakpastian menghasilkan `YELLOW`.

## Market pulse

Agregasi membandingkan jumlah publikasi dalam 7 hari terakhir dengan 7 hari sebelumnya. `DemandGrowth` adalah perubahan persentase; baseline nol menghasilkan 0 dan `STABLE` karena data tidak cukup. Sentimen bullish di atas +5%, bearish di bawah −5%. `AverageRate` memakai USD/jam untuk gaji yang dapat dinormalisasi; bila hanya ada task, USD/task. Keduanya tidak dicampur dalam satu rata-rata. `TopSkills` memuat maksimal lima skill dengan urutan deterministik.

Data dibatasi 14 hari dan 20,000 listing unik, tanpa mock, tanggal masa depan, atau tanggal yang hilang. Snapshot disimpan di cache dan dipulihkan lintas sesi. Deskripsi lengkap tidak disimpan dalam history pasar. Laporan mewakili **sampel yang pernah diambil engine**; pagination terbatas, perubahan cakupan sumber, dan filter upstream dapat memengaruhi tren. Snapshot tidak menjamin listing masih terbuka. Cache history juga tunduk pada auto-purge 7 hari tanpa aktivitas.

## Verifikasi

```sh
go test ./... -cover
go test -race ./...
go vet ./...
```

Seluruh unit/integration test memakai server `httptest`, clock terkontrol, dan direktori sementara; tidak memerlukan kredensial atau koneksi ke API publik. Test meliputi kontrak, schema API, cache/TTL/purge, konkurensi, pembatalan, circuit breaker, failover, konversi valuta, similarity, batas skor, risiko Web3, serta agregasi historis. Race detector pada Windows memerlukan compiler C yang kompatibel.

## Referensi API

- [RemoteOK API dan atribusi](https://remoteok.com/api)
- [Remotive: schema, atribusi, dan frekuensi request](https://github.com/remotive-com/remote-jobs-api)
- [Jobicy API](https://jobicy.com/jobs-rss-feed)
- [Adzuna Search](https://developer.adzuna.com/docs/search)
- [Hacker News Firebase API](https://github.com/HackerNews/API)
- [GitHub Search Issues](https://docs.github.com/en/rest/search/search#search-issues-and-pull-requests)
- [Frankfurter v1](https://frankfurter.dev/v1/)

Skema Web3 contoh tersedia di [web3-feed.example.json](web3-feed.example.json).
