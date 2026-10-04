# Penyiapan Go untuk HexaJobs

HexaJobs (`hexajobs-cli`) membutuhkan **Go 1.24+** (`go.mod` menetapkan
`go 1.24.0` untuk dependensi Bubble Tea/Bubbles/Lip Gloss).

## 1. Cek instalasi

```sh
go version   # harusnya: go version go1.24.0 atau lebih baru
```

Jika muncul `go: perintah tidak ditemukan`, lanjut ke langkah 2.

## 2. Opsi A — pakai SDK lokal yang sudah ada (mesin ini)

Mesin ini sudah memiliki SDK Go 1.27.0 di
`$HOME/go1.27.0.linux-amd64/go/`, hanya belum masuk PATH:

```sh
# Coba sementara di shell saat ini:
export PATH="$HOME/go1.27.0.linux-amd64/go/bin:$PATH"
go version

# Permanen (tambah ke ~/.bashrc bila belum ada):
grep -q 'go1.27.0.linux-amd64/go/bin' ~/.bashrc \
  || echo 'export PATH="$HOME/go1.27.0.linux-amd64/go/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc
go version
```

## 3. Opsi B — instalasi fresh (mesin lain)

Pilih salah satu:

```sh
# Debian/Ubuntu (versi distro bisa lebih lama dari 1.24 — cek dulu):
sudo apt update && sudo apt install golang-go
go version

# Linuxbrew:
/home/syfu/.linuxbrew/bin/brew install go

# Tarball resmi go.dev (contoh Go 1.24, amd64):
curl -LO https://go.dev/dl/go1.24.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.24.0.linux-amd64.tar.gz
echo 'export PATH="/usr/local/go/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc
```

## 4. Variabel lingkungan (umumnya default sudah cukup)

| Variabel     | Default                              | Keterangan                    |
| ------------ | ------------------------------------ | ----------------------------- |
| `GOPATH`     | `~/go`                               | Workspace modul & binary      |
| `GOCACHE`    | `~/.cache/go-build`                  | Cache kompilasi               |
| `GOTOOLCHAIN`| `auto`                               | Unduh toolchain bila `go.mod` meminta versi lebih baru |
| `GOPROXY`    | `https://proxy.golang.org,direct`    | Ganti ke `direct`/`off` bila proxy diblokir |

Lihat nilai aktif: `go env GOPATH GOCACHE GOTOOLCHAIN GOPROXY`.

## 5. Verifikasi terhadap repo

```sh
go version                 # >= go1.24.0
go vet ./...               # harus bersih, tanpa output
go test ./... -count=1     # semua paket OK
```

Jalankan dari direktori `hexajobs-cli/`.

## 6. Troubleshooting

| Gejala | Penyebab umum | Solusi |
| ------ | ------------- | ------ |
| `go: perintah tidak ditemukan` | `bin/` Go belum di PATH | Ulangi langkah 2, pastikan `source ~/.bashrc` atau buka terminal baru |
| PATH duplikat berkali-kali | Baris export ditambah berulang | Cek `~/.bashrc`, hapus duplikat; pola `grep -q ... \|\| echo ...` di atas aman idempoten |
| `permission denied` saat `go build` | Direktori output milik root / read-only | Build ke direktori milik user (`bin/` repo) atau perbaiki kepemilikan |
| `GOTOOLCHAIN` gagal mengunduh | Offline / proxy diblokir | `go env -w GOTOOLCHAIN=local` lalu pakai toolchain lokal yang memenuhi `go.mod` |
| `GOPROXY ... 403/404` | Jaringan/proxy membatasi | `go env -w GOPROXY=direct` atau isi module cache manual |
| `go test` gagal unduh modul | Belum pernah `go mod download` | `go mod download` sekali saat online, lalu test offline dengan `GOFLAGS=-mod=mod` |
| Cache penuh / corrupt | `GOCACHE` rusak | `go clean -cache` lalu ulangi build |
