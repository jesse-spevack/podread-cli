#!/bin/sh
# Install script for the podread CLI.
# Usage: curl -fsSL https://podread.app/install.sh | sh
set -e

BASE_URL="${PODREAD_INSTALL_URL:-https://podread.app/cli}"
BINARY_NAME="podread"

detect_os() {
  os="$(uname -s)"
  case "$os" in
    Darwin) echo "darwin" ;;
    Linux)  echo "linux" ;;
    *)
      echo "Error: unsupported operating system: $os" >&2
      echo "podread supports macOS (darwin) and Linux." >&2
      exit 1
      ;;
  esac
}

detect_arch() {
  arch="$(uname -m)"
  case "$arch" in
    x86_64|amd64)  echo "amd64" ;;
    arm64|aarch64) echo "arm64" ;;
    *)
      echo "Error: unsupported architecture: $arch" >&2
      echo "podread supports amd64 and arm64." >&2
      exit 1
      ;;
  esac
}

install_dir() {
  if [ -w /usr/local/bin ]; then
    echo "/usr/local/bin"
  else
    dir="$HOME/.local/bin"
    mkdir -p "$dir"
    echo "$dir"
  fi
}

main() {
  os="$(detect_os)"
  arch="$(detect_arch)"
  dir="$(install_dir)"
  url="${BASE_URL}/${BINARY_NAME}-${os}-${arch}"
  dest="${dir}/${BINARY_NAME}"

  echo "Downloading ${BINARY_NAME} for ${os}/${arch}..."
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL -o "$dest" "$url"
  elif command -v wget >/dev/null 2>&1; then
    wget -qO "$dest" "$url"
  else
    echo "Error: curl or wget is required to download podread." >&2
    exit 1
  fi

  chmod +x "$dest"

  # Verify the binary runs.
  if "$dest" --version >/dev/null 2>&1; then
    echo "podread installed to ${dest}"
    "$dest" --version
  else
    echo "Error: installed binary failed to run." >&2
    echo "You can try manually: ${url}" >&2
    rm -f "$dest"
    exit 1
  fi

  # Warn if install dir is not on PATH.
  case ":${PATH}:" in
    *":${dir}:"*) ;;
    *)
      echo ""
      echo "NOTE: ${dir} is not in your PATH."
      echo "Add it with:  export PATH=\"${dir}:\$PATH\""
      ;;
  esac
}

main
