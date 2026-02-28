#!/bin/sh
# Install script for the podread CLI.
# Usage: curl -fsSL https://raw.githubusercontent.com/jesse-spevack/podread-cli/main/install.sh | sh
set -e

REPO="jesse-spevack/podread-cli"
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

latest_version() {
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed 's/.*"tag_name": *"//;s/".*//'
  elif command -v wget >/dev/null 2>&1; then
    wget -qO- "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed 's/.*"tag_name": *"//;s/".*//'
  else
    echo "Error: curl or wget is required to install podread." >&2
    exit 1
  fi
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

download() {
  url="$1"
  dest="$2"
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL -o "$dest" "$url"
  elif command -v wget >/dev/null 2>&1; then
    wget -qO "$dest" "$url"
  fi
}

main() {
  os="$(detect_os)"
  arch="$(detect_arch)"
  dir="$(install_dir)"

  echo "Detecting latest podread release..."
  version="$(latest_version)"
  if [ -z "$version" ]; then
    echo "Error: could not determine latest release version." >&2
    echo "Check https://github.com/${REPO}/releases for available versions." >&2
    exit 1
  fi

  # GoReleaser archive naming: podread_<os>_<arch>.tar.gz
  archive="podread_${os}_${arch}.tar.gz"
  url="https://github.com/${REPO}/releases/download/${version}/${archive}"

  echo "Downloading ${BINARY_NAME} ${version} for ${os}/${arch}..."

  tmpdir="$(mktemp -d)"
  trap 'rm -rf "$tmpdir"' EXIT

  download "$url" "${tmpdir}/${archive}"

  # Extract binary from tar.gz
  tar -xzf "${tmpdir}/${archive}" -C "$tmpdir"

  dest="${dir}/${BINARY_NAME}"
  mv "${tmpdir}/${BINARY_NAME}" "$dest"
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
