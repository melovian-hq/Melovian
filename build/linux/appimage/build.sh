#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
ROOT_DIR=$(cd "${SCRIPT_DIR}/../../.." && pwd)

export APP_NAME="${APP_NAME:-melovian}"
export APP_BINARY="${APP_BINARY:-${ROOT_DIR}/bin/${APP_NAME}}"
export ICON_PATH="${ICON_PATH:-${ROOT_DIR}/build/appicon.png}"
export DESKTOP_FILE="${DESKTOP_FILE:-${ROOT_DIR}/build/linux/${APP_NAME}.desktop}"
export OUTPUT_DIR="${OUTPUT_DIR:-${ROOT_DIR}/bin}"
export BUILD_DIR="${BUILD_DIR:-${ROOT_DIR}/build/linux/appimage/build}"

exec "${SCRIPT_DIR}/package.sh"
