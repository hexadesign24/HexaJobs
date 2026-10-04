#!/bin/sh
# Install hexajobs Linux binary to ~/.local/bin or /usr/local/bin.
# Usage: ./scripts/install-linux.sh [amd64|arm64] [--system]
set -eu

ARCH="${1:-}"
if [ "${ARCH:-}" = "--system" ]; then
  ARCH=""
fi
SYSTEM=0
for arg in "$@"; do
  if [ "$arg" = "--system" ]; then
    SYSTEM=1
  fi
done

if [ -z "$ARCH" ]; then
  case "$(uname -m)" in
    x86_64|amd64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) echo "unsupported architecture: $(uname -m)" >&2; exit 1 ;;
  esac
fi

case "$ARCH" in
  amd64|arm64) ;;
  *) echo "usage: $0 [amd64|arm64] [--system]" >&2; exit 1 ;;
esac

SRC="bin/hexajobs-linux-$ARCH"
if [ ! -f "$SRC" ]; then
  echo "missing $SRC - run 'make build-linux-$ARCH' first" >&2
  exit 1
fi

if [ "$SYSTEM" -eq 1 ]; then
  DEST="/usr/local/bin/hexajobs"
  install -m 0755 "$SRC" "$DEST"
else
  DEST="$HOME/.local/bin/hexajobs"
  mkdir -p "$HOME/.local/bin"
  install -m 0755 "$SRC" "$DEST"
  case ":$PATH:" in
    *":$HOME/.local/bin:"*) ;;
    *)
      # Complete the PATH automatically (bash only): idempotent, one line.
      if grep -qsF '$HOME/.local/bin' "$HOME/.bashrc" 2>/dev/null; then
        echo "note: \$HOME/.local/bin is already in ~/.bashrc - open a new terminal or run: source ~/.bashrc"
      else
        printf '\n# added by hexajobs installer\nexport PATH="$HOME/.local/bin:$PATH"\n' >> "$HOME/.bashrc"
        echo "added \$HOME/.local/bin to PATH in ~/.bashrc - open a new terminal or run: source ~/.bashrc"
      fi
      ;;
  esac
fi

echo "installed $DEST"
"$DEST" --version
