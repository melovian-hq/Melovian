#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 Quad4 Software
# SPDX-License-Identifier: Apache-2.0
# Cross-compile melovian-server for release targets (CGO disabled).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
BIN_DIR="${BIN_DIR:-${ROOT}/bin}"
VERSION="${VERSION:-dev}"
APP_NAME="${APP_NAME:-melovian}"
OUT_DIR="${OUT_DIR:-${BIN_DIR}/server-cross}"

mkdir -p "${OUT_DIR}"

# goos goarch [goarm] [label]
TARGETS=(
  "linux amd64"
  "linux 386"
  "linux arm64"
  "linux arm 6 armv6"
  "linux arm 7 armv7"
  "linux riscv64"
  "freebsd amd64"
  "freebsd arm64"
  "openbsd amd64"
  "openbsd arm64"
  "netbsd amd64"
  "windows amd64"
  "windows arm64"
  "darwin amd64"
  "darwin arm64"
)

build_one() {
  local goos="$1"
  local goarch="$2"
  local goarm="${3:-}"
  local label="${4:-${goarch}}"

  local ext=""
  if [ "${goos}" = "windows" ]; then
    ext=".exe"
  fi

  local binary="${OUT_DIR}/${APP_NAME}-server-${goos}-${label}${ext}"
  echo "Building ${goos}/${label} -> ${binary}"

  local env=(
    CGO_ENABLED=0
    GOOS="${goos}"
    GOARCH="${goarch}"
  )
  if [ -n "${goarm}" ]; then
    env+=(GOARM="${goarm}")
  fi

  env "${env[@]}" go build -tags server -trimpath \
    -ldflags "-s -w -X melovian/internal/compat.Version=${VERSION}" \
    -o "${binary}" \
    "${ROOT}"

  local archive
  if [ "${goos}" = "windows" ]; then
    archive="${BIN_DIR}/${APP_NAME}-${VERSION}-server-${goos}-${label}.zip"
    (
      cd "${OUT_DIR}"
      zip -q -j "${archive}" "$(basename "${binary}")"
    )
  else
    archive="${BIN_DIR}/${APP_NAME}-${VERSION}-server-${goos}-${label}.tar.gz"
    tar -C "${OUT_DIR}" -czf "${archive}" "$(basename "${binary}")"
  fi
  echo "  archived ${archive}"
}

cd "${ROOT}"
for entry in "${TARGETS[@]}"; do
  # shellcheck disable=SC2086
  set -- ${entry}
  build_one "$@"
done

echo "Cross-server builds complete in ${BIN_DIR}"
