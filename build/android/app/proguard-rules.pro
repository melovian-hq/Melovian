# Native JNI entry points used by libwails.so
-keepclasseswithmembernames class * {
    native <methods>;
}

# Wails Android bridge and UI shell
-keep class com.wails.app.MainActivity { *; }
-keep class com.wails.app.WailsBridge { *; }
-keep class com.wails.app.WailsJSBridge { *; }
-keep class com.wails.app.WailsPathHandler { *; }
-keep class com.wails.app.WailsForegroundService { *; }
# Shared MediaSession + Android Auto MediaBrowserService (reflection / binder)
-keep class com.wails.app.MelovianMediaSession { *; }
-keep class com.wails.app.MelovianMediaSession$* { *; }
-keep class com.wails.app.MelovianMediaBrowserService { *; }

# WebView JavaScript interface methods must keep their names
-keepattributes *JavascriptInterface*
-keepclassmembers class com.wails.app.WailsJSBridge {
    @android.webkit.JavascriptInterface <methods>;
}

# Media session / browser callbacks must survive minify (system MediaStyle + Auto)
-keep class android.support.v4.media.** { *; }
-keep class androidx.media.** { *; }
-keepclassmembers class * extends android.support.v4.media.session.MediaSessionCompat$Callback {
    <methods>;
}
-keepclassmembers class * extends androidx.media.MediaBrowserServiceCompat {
    <methods>;
}

# AndroidX WebKit asset loader
-keep class androidx.webkit.** { *; }
-dontwarn androidx.webkit.**

# Biometric and encrypted storage used by the bridge
-keep class androidx.biometric.** { *; }
-keep class androidx.security.crypto.** { *; }
-dontwarn androidx.security.crypto.**

# Material / AppCompat theme resources referenced from manifest
-keep class com.google.android.material.** { *; }
-dontwarn com.google.android.material.**

# Tink / JSR-305 annotations pulled in by security-crypto
-dontwarn javax.annotation.**
-dontwarn javax.annotation.concurrent.**
