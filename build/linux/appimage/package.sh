#!/usr/bin/env bash
set -euo pipefail

: "${APP_NAME:?APP_NAME is required}"
: "${APP_BINARY:?APP_BINARY is required}"
: "${ICON_PATH:?ICON_PATH is required}"
: "${DESKTOP_FILE:?DESKTOP_FILE is required}"
: "${OUTPUT_DIR:?OUTPUT_DIR is required}"
: "${BUILD_DIR:?BUILD_DIR is required}"

APP_BINARY=$(readlink -f "$APP_BINARY")
ICON_PATH=$(readlink -f "$ICON_PATH")
DESKTOP_FILE=$(readlink -f "$DESKTOP_FILE")
OUTPUT_DIR=$(readlink -f "$OUTPUT_DIR")
BUILD_DIR=$(readlink -f "$BUILD_DIR")
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
mkdir -p "$BUILD_DIR" "$OUTPUT_DIR"

case "$(uname -m)" in
    x86_64) ARCH=x86_64 ;;
    aarch64|arm64) ARCH=aarch64 ;;
    *)
        echo "unsupported architecture: $(uname -m)" >&2
        exit 1
        ;;
esac

BINARY_NAME=$(basename "$APP_BINARY")
APP_DIR="${BUILD_DIR}/${APP_NAME}-${ARCH}.AppDir"
APPIMAGE_DESKTOP="${BUILD_DIR}/${APP_NAME}.appimage.desktop"
rm -rf "$APP_DIR"

desktop_field() {
    local key="$1" default="$2"
    local value=""
    value=$(grep -m1 "^${key}=" "$DESKTOP_FILE" 2>/dev/null | cut -d= -f2- || true)
    if [[ -n "$value" ]]; then
        echo "$value"
    else
        echo "$default"
    fi
}

DESKTOP_NAME=$(desktop_field Name "$APP_NAME")
DESKTOP_COMMENT=$(desktop_field Comment "$APP_NAME")
DESKTOP_CATEGORIES=$(desktop_field Categories "AudioVideo;Audio;Music;Player;")
DESKTOP_KEYWORDS=$(desktop_field Keywords "")
DESKTOP_WM_CLASS=$(desktop_field StartupWMClass "$APP_NAME")

cat > "$APPIMAGE_DESKTOP" <<EOF
[Desktop Entry]
Type=Application
Name=${DESKTOP_NAME}
Exec=${APP_NAME} %U
Icon=${APP_NAME}
StartupWMClass=${DESKTOP_WM_CLASS}
Categories=${DESKTOP_CATEGORIES}
Comment=${DESKTOP_COMMENT}
Terminal=false
Keywords=${DESKTOP_KEYWORDS}
Version=1.0
StartupNotify=false
X-AppImage-Name=${APP_NAME}
EOF

USR_BIN="${APP_DIR}/usr/bin"
mkdir -p "$USR_BIN"
cp "$ICON_PATH" "${APP_DIR}/.DirIcon"
ln -sf .DirIcon "${APP_DIR}/${APP_NAME}.png"
cp "$APPIMAGE_DESKTOP" "${APP_DIR}/${APP_NAME}.desktop"

ICONS_SRC="$(cd "${BUILD_DIR}/../.." && pwd)/icons"
if [[ -d "${ICONS_SRC}" ]]; then
    mkdir -p "${APP_DIR}/usr/share/icons/hicolor"
    cp -a "${ICONS_SRC}/." "${APP_DIR}/usr/share/icons/hicolor/"
fi

cd "$BUILD_DIR"

LINUXDEPLOY="linuxdeploy-${ARCH}.AppImage"
RUNTIME="runtime-${ARCH}"

echo "Using FUSE 3 type2 runtime: ${RUNTIME}"

fetch() {
    local url="$1" dest="$2"
    if [[ ! -f "$dest" ]]; then
        wget -q -4 -N "$url" -O "$dest"
    fi
}

fetch "https://github.com/linuxdeploy/linuxdeploy/releases/download/continuous/${LINUXDEPLOY}" "$LINUXDEPLOY"
fetch "https://github.com/AppImage/type2-runtime/releases/download/continuous/${RUNTIME}" "$RUNTIME"
chmod +x "$LINUXDEPLOY" "$RUNTIME"

find_libmpv() {
    local candidate=""
    candidate=$(ldconfig -p 2>/dev/null | awk '/libmpv\.so\.[0-9]/{print $NF; exit}')
    if [[ -n "$candidate" && -f "$candidate" ]]; then
        echo "$candidate"
        return 0
    fi
    for candidate in \
        "/usr/lib/${ARCH}-linux-gnu/libmpv.so.2" \
        "/usr/lib64/libmpv.so.2" \
        "/usr/lib/libmpv.so.2"; do
        if [[ -f "$candidate" ]]; then
            echo "$candidate"
            return 0
        fi
    done
    return 1
}

LIBMPV=$(find_libmpv) || {
    echo "libmpv not found; install libmpv2 or libmpv-dev on the build host" >&2
    exit 1
}
echo "Bundling libmpv from ${LIBMPV}"

chmod +x "${SCRIPT_DIR}/AppRun"

APP_IMAGE_NAME="${APP_NAME}-${ARCH}.AppImage"
export OUTPUT="$APP_IMAGE_NAME"
export LDAI_OUTPUT="$APP_IMAGE_NAME"
export LDAI_RUNTIME_FILE="${BUILD_DIR}/${RUNTIME}"
export NO_STRIP=1

./"${LINUXDEPLOY}" --appimage-extract-and-run \
    --appdir "$APP_DIR" \
    --custom-apprun "${SCRIPT_DIR}/AppRun" \
    --desktop-file "$APPIMAGE_DESKTOP" \
    --icon-file "$ICON_PATH" \
    --icon-filename "${APP_NAME}" \
    --library "$LIBMPV"

cp "$APP_BINARY" "${USR_BIN}/${BINARY_NAME}"
chmod 755 "${USR_BIN}/${BINARY_NAME}"

./"${LINUXDEPLOY}" --appimage-extract-and-run \
    --appdir "$APP_DIR" \
    --output appimage

mv "${BUILD_DIR}/${APP_IMAGE_NAME}" "${OUTPUT_DIR}/"
echo "AppImage created: ${OUTPUT_DIR}/${APP_IMAGE_NAME}"
