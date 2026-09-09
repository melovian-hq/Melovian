#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 Quad4 Software
# SPDX-License-Identifier: Apache-2.0
#
# release-extras.sh <tag> [dist-dir]
#
# Post-GoReleaser step for release.yml. It:
#   1. Appends raw-binary sha256 entries (melovian-<tag>-bin-<os>-<arch>) to
#      the checksums manifest so delta-patched binaries can be verified.
#   2. Generates bsdiff deltas from the previous release's server binaries
#      (melovian-<tag>-patch-<os>-<arch>-from-<prev>.bspatch).
#   3. Signs the checksums manifest with UPDATE_SIGNING_KEY (Ed25519,
#      base64) producing melovian-<tag>-checksums.txt.sig.
#   4. Uploads the augmented checksums, the signature, and the patches to
#      the release with --clobber.
#
# Requires: gh (authenticated), bsdiff, tar/unzip. Skips signing when the
# key secret is unset so forks can still cut unsigned releases.

set -euo pipefail

TAG="${1:?usage: release-extras.sh <tag> [dist-dir]}"
DIST="${2:-dist}"
SUMS="${DIST}/melovian-${TAG}-checksums.txt"

mkdir -p "${DIST}"


extract_bin() {
  local archive="$1" outdir="$2"
  mkdir -p "${outdir}"
  case "${archive}" in
    *.zip) unzip -o -q "${archive}" -d "${outdir}" ;;
    *) tar -xzf "${archive}" -C "${outdir}" ;;
  esac
  find "${outdir}" -type f \( -name 'melovian-server' -o -name 'melovian-server.exe' \) | head -1
}

# 1. Checksum coverage for every local release asset GoReleaser did not
# hash (desktop archives come from ./artifacts via extra_files), plus
# raw-binary sha256 entries for delta verification.
for f in "${DIST}"/melovian-"${TAG}"-* artifacts/melovian-"${TAG}"-*; do
  [[ -e "${f}" ]] || continue
  base="$(basename "${f}")"
  case "${base}" in
    *checksums*|*.sig|*.bspatch) continue ;;
  esac
  if grep -q "  ${base}\$" "${SUMS}"; then
    continue
  fi
  hash="$(sha256sum "${f}" | cut -d' ' -f1)"
  echo "${hash}  ${base}" >> "${SUMS}"
done

for archive in "${DIST}"/melovian-"${TAG}"-server-*.tar.gz "${DIST}"/melovian-"${TAG}"-server-*.zip; do
  [[ -e "${archive}" ]] || continue
  base="$(basename "${archive}")"
  suffix="${base#melovian-${TAG}-server-}"
  suffix="${suffix%.tar.gz}"
  suffix="${suffix%.zip}"
  tmp="$(mktemp -d)"
  bin="$(extract_bin "${archive}" "${tmp}")"
  if [[ -z "${bin}" ]]; then
    echo "no binary inside ${base}; skipping bin hash" >&2
    rm -rf "${tmp}"
    continue
  fi
  hash="$(sha256sum "${bin}" | cut -d' ' -f1)"
  echo "${hash}  melovian-${TAG}-bin-${suffix}" >> "${SUMS}"
  rm -rf "${tmp}"
done

# 2. Delta patches against the previous release tag.
PREV_TAG="$(gh release list --limit 20 --json tagName --jq '.[].tagName' | grep -v "^${TAG}$" | head -1 || true)"
if [[ -n "${PREV_TAG}" ]] && command -v bsdiff >/dev/null 2>&1; then
  prev_dir="$(mktemp -d)"
  gh release download "${PREV_TAG}" --pattern "melovian-${PREV_TAG}-server-*" --dir "${prev_dir}" --clobber || true
  for archive in "${DIST}"/melovian-"${TAG}"-server-*.tar.gz; do
    [[ -e "${archive}" ]] || continue
    base="$(basename "${archive}")"
    suffix="${base#melovian-${TAG}-server-}"
    suffix="${suffix%.tar.gz}"
    prev_ext="tar.gz"
    [[ "${suffix}" == windows-* ]] && prev_ext="zip"
    prev_archive="${prev_dir}/melovian-${PREV_TAG}-server-${suffix}.${prev_ext}"
    [[ -f "${prev_archive}" ]] || continue
    old_tmp="$(mktemp -d)"; new_tmp="$(mktemp -d)"
    old_bin="$(extract_bin "${prev_archive}" "${old_tmp}")"
    new_bin="$(extract_bin "${archive}" "${new_tmp}")"
    patch="${DIST}/melovian-${TAG}-patch-${suffix}-from-${PREV_TAG}.bspatch"
    if [[ -n "${old_bin}" && -n "${new_bin}" ]] && bsdiff "${old_bin}" "${new_bin}" "${patch}"; then
      hash="$(sha256sum "${patch}" | cut -d' ' -f1)"
      echo "${hash}  $(basename "${patch}")" >> "${SUMS}"
      echo "delta: ${suffix} ${PREV_TAG} -> ${TAG} ($(stat -c%s "${patch}") bytes)"
    fi
    rm -rf "${old_tmp}" "${new_tmp}"
  done
  rm -rf "${prev_dir}"
else
  echo "no previous tag or bsdiff missing; skipping deltas"
fi

# 3. Sign the checksums manifest when a key is configured.
if [[ -n "${UPDATE_SIGNING_KEY:-}" ]]; then
  UPDATE_SIGNING_KEY="${UPDATE_SIGNING_KEY}" go run ./build/scripts/sign "${SUMS}"
else
  echo "UPDATE_SIGNING_KEY unset; release will ship unsigned checksums" >&2
fi

# 4. Upload extras, clobbering the GoReleaser-uploaded checksums file.
files=("${SUMS}")
[[ -f "${SUMS}.sig" ]] && files+=("${SUMS}.sig")
for p in "${DIST}"/melovian-"${TAG}"-patch-*.bspatch; do
  [[ -e "${p}" ]] && files+=("${p}")
done
gh release upload "${TAG}" "${files[@]}" --clobber
