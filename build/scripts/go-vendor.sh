#!/usr/bin/env bash
# Vendor Go modules into ./vendor.
# Wails v3 ships go:embed paths for WebView2Loader.dll that are missing from
# the published module zip. Copy the DLLs from third_party before go mod vendor.
set -euo pipefail

root="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$root"

dll_src="$root/third_party/wails-webview2loader"
if [[ ! -f "$dll_src/arm64/WebView2Loader.dll" || ! -f "$dll_src/x64/WebView2Loader.dll" || ! -f "$dll_src/x86/WebView2Loader.dll" ]]; then
  echo "missing third_party/wails-webview2loader/{arm64,x64,x86}/WebView2Loader.dll" >&2
  exit 1
fi

wails_ver="$(go list -m -f '{{.Version}}' github.com/wailsapp/wails/v3)"
modcache="$(go env GOMODCACHE)/github.com/wailsapp/wails/v3@${wails_ver}/internal/webview2/webviewloader"

if [[ ! -d "$modcache" ]]; then
  go mod download github.com/wailsapp/wails/v3@"$wails_ver"
fi

chmod -R u+w "$modcache"
mkdir -p "$modcache/arm64" "$modcache/x64" "$modcache/x86"
cp -f "$dll_src/arm64/WebView2Loader.dll" "$modcache/arm64/"
cp -f "$dll_src/x64/WebView2Loader.dll" "$modcache/x64/"
cp -f "$dll_src/x86/WebView2Loader.dll" "$modcache/x86/"

rm -rf vendor
go mod vendor

echo "vendored modules into ./vendor (wails ${wails_ver} WebView2Loader embeds patched)"
