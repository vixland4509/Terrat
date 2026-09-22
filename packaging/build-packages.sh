#!/usr/bin/env bash
set -euo pipefail

VERSION="${1:-${VERSION:-0.2.3}}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
DIST_DIR="${ROOT_DIR}/dist"

mkdir -p "${DIST_DIR}"

if command -v nfpm >/dev/null 2>&1; then
    NFPM="nfpm"
elif [ -x "${HOME}/go/bin/nfpm" ]; then
    NFPM="${HOME}/go/bin/nfpm"
else
    echo "Installing nfpm for packaging..."
    go install github.com/goreleaser/nfpm/v2/cmd/nfpm@latest
    NFPM="${HOME}/go/bin/nfpm"
fi

echo "Building Linux distribution packages for v${VERSION}..."

for ARCH in amd64 arm64; do
    BIN_SRC="${DIST_DIR}/terrat-linux-${ARCH}"
    echo "Compiling Linux binary for ${ARCH}..."
    GOTOOLCHAIN=local CGO_ENABLED=0 GOOS=linux GOARCH="${ARCH}" go build -ldflags="-s -w -X main.Version=${VERSION}" -o "${BIN_SRC}" "${ROOT_DIR}/main.go"

    NFPM_CONFIG=$(mktemp --suffix=.yaml)
    cat << EOF > "${NFPM_CONFIG}"
name: "terrat"
arch: "${ARCH}"
platform: "linux"
version: "${VERSION}"
section: "utils"
priority: "optional"
maintainer: "Nightland4509 <nightlandcompany@proton.me>"
description: "Lightweight pure Go high-performance terminal emulator"
homepage: "https://github.com/vixland4509/Terrat"
license: "MIT"
contents:
  - src: ${BIN_SRC}
    dst: /usr/bin/terrat
    file_info:
      mode: 0755
  - src: ${ROOT_DIR}/packaging/terrat.desktop
    dst: /usr/share/applications/terrat.desktop
    file_info:
      mode: 0644
  - src: ${ROOT_DIR}/packaging/terrat.desktop
    dst: /usr/share/applications/terraterminal.desktop
    file_info:
      mode: 0644
  - src: ${ROOT_DIR}/icon.png
    dst: /usr/share/icons/hicolor/512x512/apps/terrat.png
    file_info:
      mode: 0644
  - src: ${ROOT_DIR}/icon.png
    dst: /usr/share/icons/hicolor/512x512/apps/terraterminal.png
    file_info:
      mode: 0644
  - src: ${ROOT_DIR}/LICENSE
    dst: /usr/share/licenses/terrat/LICENSE
    file_info:
      mode: 0644
scripts:
  postinstall: ${ROOT_DIR}/packaging/postinstall.sh
  postremove: ${ROOT_DIR}/packaging/postremove.sh
EOF

    echo "Packaging .deb for ${ARCH}..."
    "${NFPM}" package -f "${NFPM_CONFIG}" --packager deb --target "${DIST_DIR}/"

    echo "Packaging .rpm for ${ARCH}..."
    "${NFPM}" package -f "${NFPM_CONFIG}" --packager rpm --target "${DIST_DIR}/"

    echo "Packaging Arch Linux (.pkg.tar.zst) for ${ARCH}..."
    "${NFPM}" package -f "${NFPM_CONFIG}" --packager archlinux --target "${DIST_DIR}/"

    rm -f "${NFPM_CONFIG}"

    echo "Packaging Linux portable tarball for ${ARCH}..."
    PORTABLE_DIR=$(mktemp -d)
    TARGET_SUBDIR="${PORTABLE_DIR}/terrat-v${VERSION}-linux-${ARCH}"
    mkdir -p "${TARGET_SUBDIR}"
    cp "${BIN_SRC}" "${TARGET_SUBDIR}/terrat"
    chmod 755 "${TARGET_SUBDIR}/terrat"
    cp "${ROOT_DIR}/packaging/terrat.desktop" "${TARGET_SUBDIR}/"
    cp "${ROOT_DIR}/icon.png" "${TARGET_SUBDIR}/"
    cp "${ROOT_DIR}/packaging/portable-install.sh" "${TARGET_SUBDIR}/install.sh"
    chmod +x "${TARGET_SUBDIR}/install.sh"
    cp "${ROOT_DIR}/LICENSE" "${TARGET_SUBDIR}/"

    tar -czf "${DIST_DIR}/terrat-v${VERSION}-linux-${ARCH}.tar.gz" -C "${PORTABLE_DIR}" "terrat-v${VERSION}-linux-${ARCH}"
    rm -rf "${PORTABLE_DIR}"
done

echo "Building Windows portable bundles for v${VERSION}..."
for ARCH in amd64 arm64; do
    WIN_BIN="${DIST_DIR}/terrat-windows-${ARCH}.exe"
    echo "Compiling Windows binary for ${ARCH}..."
    GOTOOLCHAIN=local CGO_ENABLED=0 GOOS=windows GOARCH="${ARCH}" go build -ldflags="-s -w -H=windowsgui -X main.Version=${VERSION}" -o "${WIN_BIN}" "${ROOT_DIR}/main.go"

    echo "Packaging Windows portable zip for ${ARCH}..."
    WIN_PORTABLE=$(mktemp -d)
    WIN_TARGET="${WIN_PORTABLE}/terrat-v${VERSION}-windows-${ARCH}"
    mkdir -p "${WIN_TARGET}"
    cp "${WIN_BIN}" "${WIN_TARGET}/terrat.exe"
    cp "${ROOT_DIR}/icon.png" "${WIN_TARGET}/"
    cp "${ROOT_DIR}/LICENSE" "${WIN_TARGET}/"

    cat << 'CFG' > "${WIN_TARGET}/config.json"
{
  "theme": "auto",
  "font_size": 13.0,
  "opacity": 0.95,
  "ghost_text": true,
  "diagnostics": true,
  "sanitize_paste": true,
  "bracketed_paste": true
}
CFG

    cat << 'RME' > "${WIN_TARGET}/README.txt"
TerraTerminal (Terrat) - Portable Edition
=========================================
A lightweight, frameless minimalist terminal emulator for Windows and Linux.

Usage:
  - Double-click terrat.exe to launch.
  - Automatically detects Git Bash, PowerShell, and WSL.
  - Edit config.json in this directory to customize fonts, colors, and behavior.

Keyboard Shortcuts:
  - Ctrl+Shift+T       : New Tab
  - Ctrl+Shift+W       : Close Tab
  - Ctrl+Tab / Shift   : Next / Previous Tab
  - Ctrl+Shift+C / V   : Copy / Paste
  - Ctrl+Shift+F       : Search text
  - Ctrl+Shift+D       : Toggle Diagnostics & Typo suggestions
  - Ctrl+Shift+Up/Down : Line scroll
  - Shift+PageUp/Down  : Page scroll
  - Ctrl+Plus / Minus  : Zoom in / Zoom out
  - Ctrl+0             : Reset zoom
RME

    (cd "${WIN_PORTABLE}" && zip -q -r "${DIST_DIR}/terrat-v${VERSION}-windows-${ARCH}.zip" "terrat-v${VERSION}-windows-${ARCH}")
    rm -rf "${WIN_PORTABLE}"
done

echo "Generating SHA-256 checksums..."
(
    cd "${DIST_DIR}"
    rm -f checksums.txt
    sha256sum *.deb *.rpm *.pkg.tar.zst *.tar.gz *.zip > checksums.txt
)

echo "Release assets successfully built in ${DIST_DIR}:"
ls -lh "${DIST_DIR}"
