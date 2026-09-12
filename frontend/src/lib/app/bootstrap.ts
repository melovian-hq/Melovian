// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { untrack } from "svelte";
import { MediaQuery } from "svelte/reactivity";
import { music } from "$lib/config/music.svelte";
import { instances } from "$lib/features/instances/store.svelte";
import { localLibraries } from "$lib/features/local-libraries/store.svelte";
import { sources } from "$lib/features/sources/store.svelte";
import { auth } from "$lib/features/auth/store.svelte";
import { eventSocket } from "$lib/core/events/ws.svelte";
import { deviceSync } from "$lib/music/device-sync.svelte";
import { notifications } from "$lib/notifications/notifications.svelte";
import { tasks } from "$lib/tasks/tasks.svelte";
import { router } from "$lib/router/router.svelte";
import { bindNativeMedia } from "$lib/media/native.svelte";
import { bindKeyboardShortcuts } from "$lib/music/keyboard";
import { keyboardHelp } from "$lib/ui/keyboard-help.svelte";
import { commandPalette } from "$lib/ui/command-palette.svelte";
import { keybindFeedback } from "$lib/ui/keybind-feedback.svelte";
import {
  isFakeCatalog,
  isStaticDemo,
  loadRuntimeConfig,
  nativeDesktopAvailable,
} from "$lib/config/runtime";
import { Cap, supports } from "$lib/compat";
import { isSettingsPath } from "$lib/settings/tabs";
import {
  bindDesktopIntegrationSettings,
  bindWindowCloseHandler,
} from "$lib/desktop/window-close";
import {
  applyWindowChromeDocumentState,
  bindNativeTitleBarSettings,
} from "$lib/desktop/window-chrome";
import { bindPlaybackLifecycle } from "$lib/music/playback-lifecycle";
import { connection } from "$lib/music/connection.svelte";
import { toast } from "$lib/ui/toast.svelte";
import { layout, MOBILE_MEDIA } from "$lib/components/layout/layout.svelte";
import { loadExtensions } from "$lib/extensions/registry";

let mobileMedia: MediaQuery | undefined;

// Runs inside an $effect in App.svelte. Reading mobileMedia.current and
// layout.sidebarCollapsed tracks them, so the effect re-runs on viewport or
// collapse changes and no manual listener is needed.
export function syncSidebarWidthEffect(): void {
  if (typeof document === "undefined") return;
  mobileMedia ??= new MediaQuery(MOBILE_MEDIA);

  const mobile = mobileMedia.current;
  layout.setMobileViewport(mobile);
  if (mobile) {
    document.documentElement.style.setProperty(
      "--jb-sidebar-current-width",
      "0px",
    );
    return;
  }
  const styles = getComputedStyle(document.documentElement);
  const width = layout.sidebarCollapsed
    ? styles.getPropertyValue("--jb-sidebar-width-collapsed").trim() || "4.5rem"
    : styles.getPropertyValue("--jb-sidebar-width").trim() || "16rem";
  document.documentElement.style.setProperty(
    "--jb-sidebar-current-width",
    width,
  );
}

export function runInitialBootstrap(
  setBootstrapped: (value: boolean) => void,
): void {
  void (async () => {
    await loadRuntimeConfig();
    if (isFakeCatalog()) {
      // In-memory only. Do not persist so a later real library keeps user prefs.
      music.metadataEnhancementSettings = {
        ...music.metadataEnhancementSettings,
        enabled: true,
        preferServerArtistArt: false,
      };
    }
    await auth.init();
    if (auth.error) {
      setBootstrapped(true);
      return;
    }
    if (!auth.needsAccountLogin) {
      await Promise.all([instances.init(), localLibraries.init()]);
      await sources.refreshStatus();
      await loadExtensions();
    }
    setBootstrapped(true);
  })();
}

export function ensureLibrariesAfterAuth(bootstrapped: boolean): void {
  if (!bootstrapped || auth.loading || auth.needsAccountLogin) return;
  if (instances.ready && localLibraries.ready) return;
  void (async () => {
    await Promise.all([instances.init(), localLibraries.init()]);
    await sources.refreshStatus();
    await loadExtensions();
  })();
}

export function connectRealtimeServices(
  bootstrapped: boolean,
): (() => void) | undefined {
  if (!bootstrapped || auth.needsAccountLogin) return;
  if (isStaticDemo() || !supports(Cap.ws)) return;
  eventSocket.connect();
  deviceSync.start();
  if (auth.enabled && auth.authenticated) {
    void notifications.start();
  } else {
    notifications.stop();
  }
  // untrack: bindEvents reads eventSocket.connected once. Tracking it
  // re-ran this effect, called disconnect(), and permanently killed WS.
  const unbindScans = untrack(() => localLibraries.bindEvents());
  const unbindTasks = untrack(() => tasks.bindEvents());
  return () => {
    unbindScans?.();
    unbindTasks?.();
  };
}

export function publishDevicePlayback(bootstrapped: boolean): void {
  if (!bootstrapped || auth.needsAccountLogin) return;
  void music.playing;
  void music.currentTrack?.id;
  void music.queueIndex;
  void deviceSync.isHost;
  void deviceSync.sessionId;
  const force = deviceSync.isHost && !!deviceSync.sessionId;
  deviceSync.publishPlayback(force);
}

export function loadConnectionSettings(bootstrapped: boolean): void {
  if (!bootstrapped || auth.needsAccountLogin) return;
  void connection.loadRemoteSettings();
}

export function initSourceConnection(
  bootstrapped: boolean,
): (() => void) | undefined {
  if (!bootstrapped || auth.needsAccountLogin || sources.needsSetup) return;
  if (sources.hasSubsonicActive && !sources.hasUnifiedMode) {
    connection.init(
      () => music.connect({ quiet: true, authEnabled: auth.enabled }),
      () => music.pingServer(),
      () => music.selfHeal(),
      {
        onReconnected: () => {
          toast.success("Back online");
        },
        onDisconnected: () => {
          toast.warning("Connection lost, retrying…");
        },
      },
    );
    return () => connection.dispose();
  }
  if (sources.hasLocalActive && !sources.hasUnifiedMode) {
    void (async () => {
      const ok = await music.connect({
        quiet: true,
        authEnabled: auth.enabled,
      });
      if (!ok) {
        await music.connect({
          quiet: true,
          authEnabled: auth.enabled,
          force: true,
        });
      }
    })();
    return;
  }
  if (sources.hasUnifiedMode) {
    void music.connect({ quiet: true, authEnabled: auth.enabled });
  }
}

export function syncRoutePath(): void {
  music.setRoutePath(router.pathname);
}

export function bindNativeMediaEffect(
  bootstrapped: boolean,
): (() => void) | undefined {
  if (!bootstrapped) return;
  if (!nativeDesktopAvailable()) return;
  return bindNativeMedia();
}

export function bindPlaybackLifecycleEffect(
  bootstrapped: boolean,
): (() => void) | undefined {
  if (!bootstrapped) return;
  return bindPlaybackLifecycle();
}

export function bindWindowChromeEffect(
  bootstrapped: boolean,
): (() => void) | undefined {
  if (!bootstrapped) return;
  if (!nativeDesktopAvailable()) return;
  applyWindowChromeDocumentState();
  return bindNativeTitleBarSettings();
}

export function bindDesktopIntegrationEffect(
  bootstrapped: boolean,
): (() => void) | undefined {
  if (!bootstrapped) return;
  if (!nativeDesktopAvailable()) return;
  return bindDesktopIntegrationSettings();
}

export function bindWindowCloseEffect(
  bootstrapped: boolean,
): (() => void) | undefined {
  if (!bootstrapped) return;
  if (!nativeDesktopAvailable()) return;
  return bindWindowCloseHandler();
}

export function bindAndroidMediaActions(
  bootstrapped: boolean,
): (() => void) | undefined {
  if (!bootstrapped) return;
  if (typeof window === "undefined") return;

  const onNativeMediaAction = (event: Event) => {
    const action = (event as CustomEvent<{ action?: string }>).detail?.action;
    if (action === "play") {
      if (!music.playing) void music.togglePlay();
    } else if (action === "pause") {
      if (music.playing) music.pause();
    } else if (action === "toggle") {
      void music.togglePlay();
    } else if (action === "next") {
      music.next();
    } else if (action === "previous") {
      music.previous();
    } else if (action?.startsWith("seek:")) {
      const seconds = Number(action.slice(5));
      if (Number.isFinite(seconds)) music.seek(seconds);
    } else if (action?.startsWith("play-queue-index:")) {
      const index = Number(action.slice("play-queue-index:".length));
      if (Number.isInteger(index) && index >= 0) {
        music.playQueueIndex(index);
      }
    }
  };

  window.addEventListener("native-media-action", onNativeMediaAction);
  window.addEventListener("android-media-action", onNativeMediaAction);
  window.addEventListener("ios-media-action", onNativeMediaAction);
  return () => {
    window.removeEventListener("native-media-action", onNativeMediaAction);
    window.removeEventListener("android-media-action", onNativeMediaAction);
    window.removeEventListener("ios-media-action", onNativeMediaAction);
  };
}

export function bindKeyboardShortcutsEffect(
  bootstrapped: boolean,
): (() => void) | undefined {
  if (!bootstrapped || auth.needsAccountLogin) return;
  return bindKeyboardShortcuts(
    {
      togglePlay: () => {
        keybindFeedback.playback(!music.playing);
        void music.togglePlay();
      },
      next: () => {
        keybindFeedback.skip("next");
        music.next();
      },
      previous: () => {
        keybindFeedback.skip("prev");
        music.previous();
      },
      adjustVolume: (delta) => {
        music.adjustVolume(delta);
        keybindFeedback.volume(music.volume);
      },
      seekBy: (seconds) => {
        keybindFeedback.seek(seconds);
        music.seekBy(seconds);
      },
      toggleHelp: () => keyboardHelp.toggle(),
      closeHelp: () => keyboardHelp.close(),
      togglePalette: () => {
        keyboardHelp.close();
        commandPalette.toggle();
      },
      closePalette: () => commandPalette.close(),
      toggleQueue: () => music.toggleQueue(),
    },
    () => keyboardHelp.open,
    () => commandPalette.open,
  );
}

export function syncAuthRouting(bootstrapped: boolean): void {
  if (!bootstrapped || auth.loading) return;
  if (auth.needsAccountLogin) {
    if (router.pathname !== "/account/login") {
      router.navigate("/account/login", true);
    }
    return;
  }
  if (instances.loading || localLibraries.loading) return;
  if (auth.demoMode && router.pathname === "/account/login") {
    router.navigate("/music", true);
    return;
  }
  if (auth.demoMode && isSettingsPath(router.pathname)) {
    router.navigate("/music", true);
    return;
  }
  if (router.pathname === "/account/login") {
    router.navigate("/music", true);
    return;
  }
  if (router.pathname === "/login" || router.pathname === "/") {
    router.navigate("/music", true);
  }
  if (router.pathname === "/settings") {
    router.navigate("/settings/profile", true);
  }
  if (router.pathname === "/instances") {
    router.navigate("/settings/servers", true);
  }
}
