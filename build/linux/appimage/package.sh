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

cp "$APP_BINARY" "${USR_BIN}/${BINARY_NAME}"
chmod 755 "${USR_BIN}/${BINARY_NAME}"

# WebKitGTK subprocesses (WebKitWebProcess, WebKitNetworkProcess, ...) are
# helper executables looked up at the libexec dir compiled into
# libwebkitgtk, not linked libraries, so linuxdeploy never copies them.
# Bundle them before the deploy pass so their own library dependencies are
# resolved too.
WEBKIT_LIBEXEC=""
for candidate in \
    "/usr/lib/${ARCH}-linux-gnu/webkitgtk-6.0" \
    "/usr/lib/${ARCH}-linux-gnu/webkit2gtk-4.1" \
    "/usr/lib/webkitgtk-6.0" \
    "/usr/lib/webkit2gtk-4.1" \
    "/usr/libexec/webkitgtk-6.0" \
    "/usr/libexec/webkit2gtk-4.1"; do
    if [[ -d "$candidate" ]]; then
        WEBKIT_LIBEXEC="$candidate"
        break
    fi
done

LINUXDEPLOY_EXES=(--executable "${USR_BIN}/${BINARY_NAME}")
if [[ -n "$WEBKIT_LIBEXEC" ]]; then
    echo "Bundling WebKitGTK helpers from ${WEBKIT_LIBEXEC}"
    WEBKIT_DEST="${APP_DIR}/usr/libexec/$(basename "$WEBKIT_LIBEXEC")"
    mkdir -p "$WEBKIT_DEST"
    cp -a "$WEBKIT_LIBEXEC/." "$WEBKIT_DEST/"
    for helper in "$WEBKIT_DEST"/*; do
        if [[ -f "$helper" && ! -L "$helper" && -x "$helper" ]]; then
            LINUXDEPLOY_EXES+=(--executable "$helper")
        fi
    done
else
    echo "warning: no WebKitGTK libexec dir found; the AppImage webview will not work" >&2
fi

./"${LINUXDEPLOY}" --appimage-extract-and-run \
    --appdir "$APP_DIR" \
    --custom-apprun "${SCRIPT_DIR}/AppRun" \
    --desktop-file "$APPIMAGE_DESKTOP" \
    --icon-file "$ICON_PATH" \
    --icon-filename "${APP_NAME}" \
    --library "$LIBMPV" \
    "${LINUXDEPLOY_EXES[@]}"

if [[ -n "$WEBKIT_LIBEXEC" ]]; then
    # Release builds of WebKitGTK ignore WEBKIT_EXEC_PATH, so rewrite the
    # compiled-in libexec path inside the bundled libraries and helpers.
    # AppRun symlinks this fixed location to the bundled helper directory.
    # The replacement must not exceed the original string length because it
    # is written in place over NUL terminated path strings.
    WEBKIT_LINK="/tmp/.melovian-webkit"
    if [[ ${#WEBKIT_LINK} -gt ${#WEBKIT_LIBEXEC} ]]; then
        echo "warning: cannot relocate webkit libexec ${WEBKIT_LIBEXEC}; path too long" >&2
    else
        for target in "$APP_DIR"/usr/lib/libwebkitgtk-*.so.* "$APP_DIR"/usr/lib/libwebkit2gtk-*.so.* "$APP_DIR"/usr/lib/libjavascriptcoregtk-*.so.* "$WEBKIT_DEST"/*; do
            if [[ -f "$target" && ! -L "$target" ]]; then
                OLD="$WEBKIT_LIBEXEC" NEW="$WEBKIT_LINK" perl -0777 -i -pe '
                    my ($old, $new) = ($ENV{OLD}, $ENV{NEW});
                    s/\Q$old\E([^\0]*)\0/
                        my $s = "$new$1\0";
                        $s . "\0" x (length($&) - length($s))
                    /ge;
                ' "$target"
            fi
        done
    fi
fi

./"${LINUXDEPLOY}" --appimage-extract-and-run \
    --appdir "$APP_DIR" \
    --output appimage

mv "${BUILD_DIR}/${APP_IMAGE_NAME}" "${OUTPUT_DIR}/"
echo "AppImage created: ${OUTPUT_DIR}/${APP_IMAGE_NAME}"
