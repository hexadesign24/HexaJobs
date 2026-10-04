#!/bin/sh
# HexaJobs quickstart: prasyarat -> binary -> pasang -> PATH -> bukti.
# Gagal cepat dengan pesan jelas di tiap tahap.
#
# Dari clone lokal:
#   ./scripts/quickstart.sh [--system]
# One-liner (repo public, tanpa token):
#   curl -fsSL https://raw.githubusercontent.com/hexadesign24/HexaJobs/main/hexajobs-cli/scripts/quickstart.sh | sh
# Skeptis curl|sh? Unduh dulu, baca, baru jalankan:
#   curl -fsSLO <url-di-atas>; sh quickstart.sh
#
# Env opsional: GITHUB_TOKEN (menaikkan rate-limit API GitHub),
# HEXAJOBS_VERSION (default v1.1.1), HEXAJOBS_REPO (default hexadesign24/HexaJobs).
set -eu

VERSION="${HEXAJOBS_VERSION:-v1.1.1}"
REPO="${HEXAJOBS_REPO:-hexadesign24/HexaJobs}"
SYSTEM=0
for arg in "$@"; do
  if [ "$arg" = "--system" ]; then SYSTEM=1; fi
done

fail() { echo "quickstart: $*" >&2; exit 1; }

# 1. Syarat ----------------------------------------------------------------
case "$(uname -s)" in
  Linux) ;;
  *) fail "hanya mendukung Linux (terdeteksi $(uname -s)). Windows: lihat README bagian Menjalankan." ;;
esac
case "$(uname -m)" in
  x86_64|amd64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) fail "arsitektur tak didukung: $(uname -m) (hanya amd64/arm64)." ;;
esac
command -v curl >/dev/null 2>&1 || fail "butuh 'curl'. Pasang dulu, mis. sudo apt install curl."
echo "quickstart: Linux/$ARCH, rilis $VERSION"

# 2. Dapatkan binary ---------------------------------------------------------
TMPDIR_QS="$(mktemp -d)"
trap 'rm -rf "$TMPDIR_QS"' EXIT INT TERM
BIN="$TMPDIR_QS/hexajobs"
GOT_BIN=0
# 2a. URL publik dulu (jalan tanpa token setelah repo public).
if curl -fsSL -o "$BIN" "https://github.com/$REPO/releases/download/$VERSION/hexajobs-linux-$ARCH" 2>/dev/null; then
  GOT_BIN=1
# 2b. Repo masih private? Coba via gh CLI (auth ditangani gh).
elif [ -n "${GITHUB_TOKEN:-}" ] && command -v gh >/dev/null 2>&1; then
  echo "quickstart: unduhan publik gagal, coba via gh (repo private)..."
  if GH_TOKEN="$GITHUB_TOKEN" gh release download "$VERSION" \
    -p "hexajobs-linux-$ARCH" -D "$TMPDIR_QS" --repo "$REPO" --clobber >/dev/null 2>&1 \
    && [ -f "$TMPDIR_QS/hexajobs-linux-$ARCH" ]; then
    BIN="$TMPDIR_QS/hexajobs-linux-$ARCH"
    GOT_BIN=1
  fi
fi
if [ "$GOT_BIN" -eq 0 ]; then
  fail "unduhan gagal. Kemungkinan: repo masih private tanpa token (set GITHUB_TOKEN + pasang gh CLI), nama rilis salah ($VERSION), atau offline. Alternatif: make build-linux-$ARCH (butuh Go 1.24+). Lihat docs/TROUBLESHOOTING.md."
fi
chmod 0755 "$BIN"
"$BIN" --version >/dev/null 2>&1 || fail "binary rusak (cek --version gagal)."

# 3. Pasang -------------------------------------------------------------------
if [ "$SYSTEM" -eq 1 ]; then
  DEST="/usr/local/bin/hexajobs"
  install -m 0755 "$BIN" "$DEST" 2>/dev/null \
    || sudo install -m 0755 "$BIN" "$DEST" \
    || fail "tidak bisa tulis ke /usr/local/bin (perlu sudo)."
else
  DEST="$HOME/.local/bin/hexajobs"
  mkdir -p "$HOME/.local/bin"
  install -m 0755 "$BIN" "$DEST"
fi
echo "quickstart: terpasang di $DEST"

# 4. Tuntaskan PATH (bash, idempoten; --system umumnya sudah di PATH) ---------
if [ "$SYSTEM" -eq 0 ]; then
  case ":$PATH:" in
    *":$HOME/.local/bin:"*) ;;
    *)
      if grep -qsF '$HOME/.local/bin' "$HOME/.bashrc" 2>/dev/null; then
        echo "quickstart: \$HOME/.local/bin sudah ada di ~/.bashrc - buka terminal baru atau: source ~/.bashrc"
      else
        printf '\n# added by hexajobs quickstart\nexport PATH="$HOME/.local/bin:$PATH"\n' >> "$HOME/.bashrc"
        echo "quickstart: \$HOME/.local/bin ditambahkan ke ~/.bashrc - buka terminal baru atau: source ~/.bashrc"
      fi
      ;;
  esac
fi

# 5. Bukti berhasil ------------------------------------------------------------
echo "quickstart: verifikasi..."
"$DEST" --version || fail "verifikasi gagal."
echo ""
echo "Berhasil. Langkah berikut (di terminal, butuh terminal interaktif):"
echo "  hexajobs --demo   # coba tanpa network"
echo "  hexajobs          # mode live"
