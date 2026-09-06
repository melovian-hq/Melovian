package com.wails.app;

import android.util.Log;
import android.webkit.WebResourceResponse;

import androidx.annotation.NonNull;
import androidx.annotation.Nullable;
import androidx.webkit.WebViewAssetLoader;

/**
 * WailsPathHandler implements WebViewAssetLoader.PathHandler to serve assets
 * from the Go asset server. This allows the WebView to load assets without
 * using a network server, similar to iOS's WKURLSchemeHandler.
 */
public class WailsPathHandler implements WebViewAssetLoader.PathHandler {
    private static final String TAG = "WailsPathHandler";
    private static final boolean DEBUG = BuildConfig.DEBUG;

    private final WailsBridge bridge;

    public WailsPathHandler(WailsBridge bridge) {
        this.bridge = bridge;
    }

    @Nullable
    @Override
    public WebResourceResponse handle(@NonNull String path) {
        if (DEBUG) Log.d(TAG, "Handling path: " + path);

        if (path.isEmpty() || path.equals("/")) {
            path = "/index.html";
        } else if (path.charAt(0) != '/') {
            path = "/" + path;
        }

        return bridge.serveWebResponse(path, "GET", null);
    }
}
