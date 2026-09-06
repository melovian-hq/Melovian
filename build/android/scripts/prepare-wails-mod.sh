#!/usr/bin/env bash
# Copy the Wails module out of GOMODCACHE and apply the androidMainOnce overlay.
# Go refuses -overlay replacements under GOMODCACHE, so the Android c-shared
# build uses -modfile with a replace to this writable copy.
set -euo pipefail

root="$(cd "$(dirname "$0")/../../.." && pwd)"
cd "${root}"

# Prefer the module cache (-mod=mod). With a vendor/ tree, the default
# -mod=vendor leaves .Dir empty even when the cache entry exists.
wails_src="$(go list -mod=mod -m -f '{{.Dir}}' github.com/wailsapp/wails/v3 2>/dev/null || true)"
if [ -z "${wails_src}" ] || [ ! -d "${wails_src}" ]; then
  go mod download github.com/wailsapp/wails/v3
  wails_src="$(go list -mod=mod -m -f '{{.Dir}}' github.com/wailsapp/wails/v3)"
fi
wails_local="${root}/build/android/wails-v3"
patch="${root}/build/android/patches/application_android.go"
modfile="${root}/build/android/go.android.mod"
sumfile="${root}/build/android/go.android.sum"

if [ -z "${wails_src}" ] || [ ! -d "${wails_src}" ]; then
  echo "prepare-wails-mod: wails module not in module cache: ${wails_src}" >&2
  exit 1
fi
if [ ! -f "${patch}" ]; then
  echo "prepare-wails-mod: missing ${patch}" >&2
  exit 1
fi

rm -rf "${wails_local}"
mkdir -p "${wails_local}"
cp -a "${wails_src}/." "${wails_local}/"
chmod -R u+w "${wails_local}"
cp "${patch}" "${wails_local}/pkg/application/application_android.go"

{
  cat "${root}/go.mod"
  printf '\nreplace github.com/wailsapp/wails/v3 => ./build/android/wails-v3\n'
} > "${modfile}"
cp "${root}/go.sum" "${sumfile}"
