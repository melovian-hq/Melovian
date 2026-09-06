#!/usr/bin/env python3
"""Patch Wails iOS webview to register melovianMedia and merge into overlay.json."""

from __future__ import annotations

import json
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
VENDOR_WV = (
    ROOT
    / "vendor/github.com/wailsapp/wails/v3/pkg/application/webview_window_ios.m"
)
OUT_WV = ROOT / "build/ios/patches/webview_window_ios.m"
OVERLAY = ROOT / "build/ios/xcode/overlay.json"

IMPORT_MARKER = '#import "mobile_features_ios_internal.h"'

HANDLER_IMPL = r'''
// MARK: - MelovianMediaHandler (Now Playing / CarPlay remote commands)
@interface MelovianMediaHandler : NSObject <WKScriptMessageHandler>
@end
@implementation MelovianMediaHandler
- (void)userContentController:(WKUserContentController *)userContentController
      didReceiveScriptMessage:(WKScriptMessage *)message {
    id body = message.body;
    NSDictionary *dict = nil;
    if ([body isKindOfClass:[NSDictionary class]]) {
        dict = (NSDictionary *)body;
    } else if ([body isKindOfClass:[NSString class]]) {
        NSData *data = [(NSString *)body dataUsingEncoding:NSUTF8StringEncoding];
        id obj = data ? [NSJSONSerialization JSONObjectWithData:data options:0 error:nil] : nil;
        if ([obj isKindOfClass:[NSDictionary class]]) dict = (NSDictionary *)obj;
    }
    if (!dict) return;
    NSString *type = [dict[@"type"] isKindOfClass:[NSString class]] ? dict[@"type"] : @"setMediaState";
    NSError *err = nil;
    NSData *jsonData = [NSJSONSerialization dataWithJSONObject:dict options:0 error:&err];
    if (err || !jsonData) return;
    NSString *json = [[NSString alloc] initWithData:jsonData encoding:NSUTF8StringEncoding];
    if ([type isEqualToString:@"setMediaQueue"]) {
        melovian_nowplaying_set_queue([json UTF8String]);
    } else {
        melovian_nowplaying_set_state([json UTF8String]);
    }
}
@end
'''

USER_SCRIPT = r'''
    // Melovian Now Playing bridge (window.wails.setMediaState / setMediaQueue)
    {
        NSString *bridgeJS =
            @"(function(){"
             "if(!window.wails)window.wails={};"
             "function post(o){"
             "try{window.webkit.messageHandlers.melovianMedia.postMessage(o);}catch(e){}"
             "}"
             "window.wails.setMediaState=function(s){"
             "var o;try{o=typeof s==='string'?JSON.parse(s):s;}catch(e){o={active:true,playing:false};}"
             "o.type='setMediaState';post(o);"
             "};"
             "window.wails.setMediaPlaying=function(p){"
             "post({type:'setMediaState',active:true,playing:!!p});"
             "};"
             "window.wails.setMediaQueue=function(s){"
             "var o;try{o=typeof s==='string'?JSON.parse(s):s;}catch(e){o={index:-1,tracks:[]};}"
             "o.type='setMediaQueue';post(o);"
             "};"
             "})();";
        WKUserScript *script = [[WKUserScript alloc]
            initWithSource:bridgeJS
            injectionTime:WKUserScriptInjectionTimeAtDocumentStart
            forMainFrameOnly:YES];
        [config.userContentController addUserScript:script];
        static MelovianMediaHandler *jjMediaHandler;
        if (!jjMediaHandler) jjMediaHandler = [[MelovianMediaHandler alloc] init];
        [config.userContentController addScriptMessageHandler:jjMediaHandler name:@"melovianMedia"];
        melovian_nowplaying_init();
    }
'''


def patch_webview(src: str) -> str:
    if "MelovianMediaHandler" in src:
        return src
    if IMPORT_MARKER not in src:
        raise SystemExit("import marker missing in webview_window_ios.m")
    # Keep original import line; add Now Playing header after includes block
    src = src.replace(
        IMPORT_MARKER,
        '#import "mobile_features_ios_internal.h"\n'
        '// MelovianNowPlaying.h is compiled from build/ios at link time.\n'
        'void melovian_nowplaying_init(void);\n'
        'void melovian_nowplaying_set_state(const char *json);\n'
        'void melovian_nowplaying_set_queue(const char *json);',
        1,
    )
    # Insert handler class before WailsViewController implementation
    marker = "// MARK: - WailsViewController"
    if marker not in src:
        raise SystemExit("WailsViewController marker missing")
    src = src.replace(marker, HANDLER_IMPL + "\n" + marker, 1)

    inject_after = (
        '[config.userContentController addScriptMessageHandler:self.messageHandler name:@"wails"];'
    )
    if inject_after not in src:
        raise SystemExit("script message handler registration missing")
    src = src.replace(inject_after, inject_after + "\n" + USER_SCRIPT, 1)
    return src


def main() -> None:
    if not VENDOR_WV.is_file():
        raise SystemExit(f"missing {VENDOR_WV}")
    OUT_WV.parent.mkdir(parents=True, exist_ok=True)
    patched = patch_webview(VENDOR_WV.read_text())
    OUT_WV.write_text(patched)

    vendor_key = str(VENDOR_WV.resolve())
    overlay: dict = {"Replace": {}}
    if OVERLAY.is_file():
        overlay = json.loads(OVERLAY.read_text())
        if not isinstance(overlay.get("Replace"), dict):
            overlay["Replace"] = {}
    overlay["Replace"][vendor_key] = str(OUT_WV.resolve())
    # Also map short vendor-relative path variants go might use
    rel = "vendor/github.com/wailsapp/wails/v3/pkg/application/webview_window_ios.m"
    overlay["Replace"][str((ROOT / rel).resolve())] = str(OUT_WV.resolve())
    OVERLAY.parent.mkdir(parents=True, exist_ok=True)
    OVERLAY.write_text(json.dumps(overlay, indent=2) + "\n")
    print(f"patched {OUT_WV}")
    print(f"updated {OVERLAY}")


if __name__ == "__main__":
    main()
