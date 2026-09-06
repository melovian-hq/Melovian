#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 Quad4 Software
# SPDX-License-Identifier: Apache-2.0
#
# Build a matching frontend + melovian-server binary.
# Stamps both sides with the same version so the compat banner stays quiet.

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

VERSION="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo 0.1.0)}"
BUILDDATE="${BUILDDATE:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}"
OUT="${OUT:-bin/melovian-server}"
GOOS="${GOOS:-linux}"
GOARCH="${GOARCH:-amd64}"

mkdir -p bin
printf '%s\n' "$VERSION" > frontend/.build-version

echo "Building frontend (VITE_APP_VERSION=${VERSION})"
(
  cd frontend
  VITE_APP_VERSION="$VERSION" pnpm build
)

echo "Building server ${OUT} (${GOOS}/${GOARCH}, version=${VERSION})"
CGO_ENABLED=0 GOOS="$GOOS" GOARCH="$GOARCH" go build -tags server \
  -ldflags "-s -w -X melovian/internal/compat.Version=${VERSION} -X melovian/internal/compat.BuildDate=${BUILDDATE}" \
  -o "$OUT" .

echo "Done: ${OUT}"
echo "  version=${VERSION}"
echo "  buildDate=${BUILDDATE}"
