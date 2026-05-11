#!/bin/sh
# easyrice installer
#
# Environment variables:
#   EASYRICE_INSTALL_DIR      Install directory (default: $HOME/.local/bin)
#   EASYRICE_VERSION          Version to install (default: latest)
#   EASYRICE_RELEASE_BASE_URL Base URL for release assets
#                             (default: https://github.com/guneet-xyz/easyrice/releases/download)
#                             Override for mirrors or local testing.
set -eu

REPO="guneet-xyz/easyrice"
BINARY="easyrice"
SYMLINK="rice"
INSTALL_DIR="${EASYRICE_INSTALL_DIR:-$HOME/.local/bin}"
VERSION="${EASYRICE_VERSION:-latest}"
RELEASE_BASE_URL="${EASYRICE_RELEASE_BASE_URL:-https://github.com/$REPO/releases/download}"

info() { printf "\033[1;34m==>\033[0m %s\n" "$1"; }
warn() { printf "\033[1;33mwarn:\033[0m %s\n" "$1" >&2; }
err()  { printf "\033[1;31merror:\033[0m %s\n" "$1" >&2; exit 1; }

prompt_yes() {
  printf "%s" "$1"
  IFS= read -r answer || answer=""
  case "$answer" in
    [Yy]|[Yy][Ee][Ss]) return 0 ;;
    *) return 1 ;;
  esac
}

for cmd in curl uname mktemp; do
  command -v "$cmd" >/dev/null 2>&1 || err "$cmd is required but not installed"
done

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
  linux|darwin) ;;
  msys*|mingw*|cygwin*) OS="windows" ;;
  *) err "Unsupported OS: $OS" ;;
esac

ARCH=$(uname -m)
case "$ARCH" in
  x86_64|amd64)  ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) err "Unsupported architecture: $ARCH" ;;
esac

EXT=""
[ "$OS" = "windows" ] && EXT=".exe"

if [ "$VERSION" = "latest" ]; then
  info "Resolving latest version..."
  VERSION=$(curl -fsSL -o /dev/null -w '%{url_effective}' \
    "https://github.com/$REPO/releases/latest" | sed 's|.*/tag/||')
  [ -n "$VERSION" ] || err "Could not determine latest version"
else
  case "$VERSION" in
    v*) ;;
    *)  VERSION="v$VERSION" ;;
  esac
fi

if [ -x "$INSTALL_DIR/$BINARY" ]; then
  CURRENT=$("$INSTALL_DIR/$BINARY" version 2>/dev/null || echo "unknown")
  printf "\033[1;33m%s is already installed (%s).\033[0m\n" "$BINARY" "$CURRENT"
  if [ -t 0 ]; then
    if ! prompt_yes "Reinstall $VERSION? [y/N] "; then
      info "Cancelled."
      exit 0
    fi
  fi
fi

ASSET="easyrice-${VERSION}-${OS}-${ARCH}${EXT}"
URL="${RELEASE_BASE_URL}/${VERSION}/${ASSET}"
SUMS_URL="${RELEASE_BASE_URL}/${VERSION}/checksums.txt"

TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

info "Downloading $ASSET..."
if ! curl -fsSL -o "$TMPDIR/$ASSET" "$URL"; then
  err "Failed to download $URL (no prebuilt binary for ${OS}/${ARCH} in ${VERSION}?)"
fi

info "Downloading checksums..."
if curl -fsSL -o "$TMPDIR/checksums.txt" "$SUMS_URL"; then
  info "Verifying checksum..."
  if command -v sha256sum >/dev/null 2>&1; then
    SHASUM="sha256sum"
  elif command -v shasum >/dev/null 2>&1; then
    SHASUM="shasum -a 256"
  else
    warn "No sha256sum/shasum found; skipping checksum verification"
    SHASUM=""
  fi

  if [ -n "$SHASUM" ]; then
    EXPECTED=$(grep " ${ASSET}\$" "$TMPDIR/checksums.txt" | awk '{print $1}')
    [ -n "$EXPECTED" ] || err "Checksum for $ASSET not found in checksums.txt"
    ACTUAL=$($SHASUM "$TMPDIR/$ASSET" | awk '{print $1}')
    if [ "$EXPECTED" != "$ACTUAL" ]; then
      err "Checksum mismatch! expected=$EXPECTED actual=$ACTUAL"
    fi
    info "Checksum OK"
  fi
else
  warn "Could not fetch checksums.txt; skipping verification"
fi

mkdir -p "$INSTALL_DIR"
mv "$TMPDIR/$ASSET" "$INSTALL_DIR/${BINARY}${EXT}"
chmod +x "$INSTALL_DIR/${BINARY}${EXT}"
info "Installed $BINARY $VERSION to $INSTALL_DIR/${BINARY}${EXT}"

ln -sf "$INSTALL_DIR/${BINARY}${EXT}" "$INSTALL_DIR/${SYMLINK}${EXT}"
info "Created symlink $SYMLINK -> $BINARY in $INSTALL_DIR"

if echo ":$PATH:" | grep -q ":$INSTALL_DIR:"; then
  info "$INSTALL_DIR is already in your PATH. You're all set."
  exit 0
fi

printf "\n\033[1;33m%s is not in your PATH.\033[0m\n" "$INSTALL_DIR"
if [ ! -t 0 ]; then
  printf "Add it manually by appending to your shell profile:\n  export PATH=\"%s:\$PATH\"\n" "$INSTALL_DIR"
  exit 0
fi

if prompt_yes "Add it to your shell profile? [y/N] "; then
  SHELL_NAME=$(basename "${SHELL:-}")
  case "$SHELL_NAME" in
    zsh)  PROFILE="$HOME/.zshrc" ;;
    bash)
      if [ -f "$HOME/.bash_profile" ]; then
        PROFILE="$HOME/.bash_profile"
      else
        PROFILE="$HOME/.bashrc"
      fi
      ;;
    fish) PROFILE="$HOME/.config/fish/config.fish" ;;
    *)    PROFILE="$HOME/.profile" ;;
  esac

  if [ "$SHELL_NAME" = "fish" ]; then
    LINE="set -gx PATH $INSTALL_DIR \$PATH"
  else
    LINE="export PATH=\"$INSTALL_DIR:\$PATH\""
  fi

  mkdir -p "$(dirname "$PROFILE")"
  {
    echo ""
    echo "# Added by easyrice installer"
    echo "$LINE"
  } >> "$PROFILE"

  info "Added to $PROFILE. Run 'source $PROFILE' or open a new shell."
else
  printf "Add this to your shell profile manually:\n  export PATH=\"%s:\$PATH\"\n" "$INSTALL_DIR"
fi
