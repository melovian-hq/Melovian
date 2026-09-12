#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 Quad4 Software
# SPDX-License-Identifier: Apache-2.0
#
# Sync ./VERSION into every file that stamps the app version.
# Run with: task version:sync

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

VERSION="$(tr -d '[:space:]' < VERSION)"
if ! grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+([-+][0-9A-Za-z.-]+)?$' <<<"$VERSION"; then
  echo "version-sync: VERSION must be a bare semver like 0.1.0 (got '$VERSION')" >&2
  exit 1
fi

SEMVER='[0-9]+\.[0-9]+\.[0-9]+[^"]*'
# Numeric triple only; Windows package manifests want four parts (0.1.0 -> 0.1.0.0).
BASE_VERSION="${VERSION%%[-+]*}"
VERSION4="${BASE_VERSION}.0"

# stamp FILE EXPECTED_MATCHES PATTERN REPLACEMENT
# Rewrites every PATTERN match to REPLACEMENT and fails when the match
# count differs, so a renamed or dropped key never goes unnoticed.
stamp() {
  local file=$1 want=$2 pattern=$3 replacement=$4
  local got
  got="$(grep -Ec "$pattern" "$file" || true)"
  if [ "$got" != "$want" ]; then
    echo "version-sync: $file: expected $want match(es) for '$pattern', found $got" >&2
    exit 1
  fi
  sed -i -E "s|$pattern|$replacement|g" "$file"
  echo "version-sync: $file -> $VERSION"
}

# stamp_plist FILE KEY
# Rewrites the <string> on the line after <key>KEY</key> to BASE_VERSION,
# keeping any suffix such as -dev, and fails when the pair is missing.
stamp_plist() {
  local file=$1 key=$2
  local got
  got="$(sed -nE "/<key>${key}<\\/key>/{n;p}" "$file" | grep -cE "<string>${SEMVER}</string>" || true)"
  if [ "$got" != "1" ]; then
    echo "version-sync: $file: expected 1 version string after <key>${key}</key>, found $got" >&2
    exit 1
  fi
  sed -i -E "/<key>${key}<\\/key>/{n;s|<string>[0-9]+\.[0-9]+\.[0-9]+([^<]*)</string>|<string>${BASE_VERSION}\1</string>|}" "$file"
  echo "version-sync: $file (${key}) -> ${BASE_VERSION}"
}

stamp internal/compat/compat.go 1 \
  "^(const defaultVersion = )\"${SEMVER}\"$" \
  "\1\"${VERSION}\""

stamp frontend/package.json 1 \
  "^([[:space:]]*)\"version\": \"${SEMVER}\",$" \
  "\1\"version\": \"${VERSION}\","

stamp build/config.yml 2 \
  "^([[:space:]]*)version: \"${SEMVER}\"$" \
  "\1version: \"${VERSION}\""

stamp build/linux/nfpm/nfpm.yaml 1 \
  "^version: \"${SEMVER}\"$" \
  "version: \"${VERSION}\""

stamp build/windows/info.json 1 \
  "\"file_version\": \"${SEMVER}\"" \
  "\"file_version\": \"${VERSION}\""

stamp build/windows/info.json 1 \
  "\"ProductVersion\": \"${SEMVER}\"" \
  "\"ProductVersion\": \"${VERSION}\""

stamp build/android/app/build.gradle 1 \
  "^([[:space:]]*)versionName \"${SEMVER}\"$" \
  "\1versionName \"${VERSION}\""

stamp build/ios/build.sh 1 \
  "^VERSION=\"${SEMVER}\"$" \
  "VERSION=\"${VERSION}\""

stamp build/ios/build.sh 1 \
  "^BUILD_NUMBER=\"${SEMVER}\"$" \
  "BUILD_NUMBER=\"${VERSION}\""

stamp_plist build/darwin/Info.plist CFBundleShortVersionString
stamp_plist build/darwin/Info.plist CFBundleVersion
stamp_plist build/darwin/Info.dev.plist CFBundleShortVersionString
stamp_plist build/darwin/Info.dev.plist CFBundleVersion
stamp_plist build/ios/Info.plist CFBundleShortVersionString
stamp_plist build/ios/Info.plist CFBundleVersion
stamp_plist build/ios/Info.dev.plist CFBundleShortVersionString
stamp_plist build/ios/Info.dev.plist CFBundleVersion

stamp build/windows/msix/template.xml 1 \
  "( )Version=\"[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+\"" \
  "\1Version=\"${VERSION4}\""

stamp build/windows/msix/app_manifest.xml 1 \
  "( )Version=\"[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+\"" \
  "\1Version=\"${VERSION4}\""

stamp build/windows/nsis/wails_tools.nsh 1 \
  "^([[:space:]]*)!define INFO_PRODUCTVERSION \"${SEMVER}\"$" \
  "\1!define INFO_PRODUCTVERSION \"${VERSION}\""

stamp build/windows/wails.exe.manifest 1 \
  "(name=\"com\.melovian\.app\" version=\")${SEMVER}(\")" \
  "\1${VERSION}\2"

# Example extensions track the app version. Bundled extensions under
# internal/extensions/bundled/ version independently and are not stamped.
stamp extensions/examples/lyrics/melovian-extension.json 1 \
  "^([[:space:]]*)\"version\": \"${SEMVER}\",$" \
  "\1\"version\": \"${VERSION}\","

stamp extensions/examples/lyrics-whisper/melovian-extension.json 1 \
  "^([[:space:]]*)\"version\": \"${SEMVER}\",$" \
  "\1\"version\": \"${VERSION}\","

stamp extensions/examples/metadata/melovian-extension.json 1 \
  "^([[:space:]]*)\"version\": \"${SEMVER}\",$" \
  "\1\"version\": \"${VERSION}\","
