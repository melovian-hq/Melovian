<script lang="ts">
  import { routes } from "./routes";
  import { APP_DESCRIPTION, APP_NAME } from "$lib/brand";
  import { music } from "$lib/config/music.svelte";
  import { auth } from "$lib/features/auth/store.svelte";
  import { router } from "$lib/router/router.svelte";
  import KeyboardHelp from "$lib/components/ui/KeyboardHelp.svelte";
  import KeybindFeedback from "$lib/components/ui/KeybindFeedback.svelte";
  import CommandPalette from "$lib/components/ui/CommandPalette.svelte";
  import WindowChrome from "$lib/components/desktop/WindowChrome.svelte";
  import MusicPlayer from "$lib/components/music/MusicPlayer.svelte";
  import LocalLibraryScanBanner from "$lib/components/instances/LocalLibraryScanBanner.svelte";
  import ToastContainer from "$lib/components/ui/ToastContainer.svelte";
  import ConfirmDialog from "$lib/components/ui/ConfirmDialog.svelte";
  import SourceSwitchOverlay from "$lib/components/ui/SourceSwitchOverlay.svelte";
  import ClosePrompt from "$lib/components/desktop/ClosePrompt.svelte";
  import Spinner from "$lib/components/ui/Spinner.svelte";
  import CompatBanner from "$lib/components/ui/CompatBanner.svelte";
  import RouteOutlet from "$lib/router/RouteOutlet.svelte";
  import RouteBoundary from "$lib/router/RouteBoundary.svelte";
  import { getCompatState } from "$lib/compat";
  import {
    bindAndroidMediaActions,
    bindDesktopIntegrationEffect,
    bindKeyboardShortcutsEffect,
    bindNativeMediaEffect,
    bindPlaybackLifecycleEffect,
    bindWindowChromeEffect,
    bindWindowCloseEffect,
    connectRealtimeServices,
    ensureLibrariesAfterAuth,
    initSourceConnection,
    loadConnectionSettings,
    publishDevicePlayback,
    runInitialBootstrap,
    syncAuthRouting,
    syncRoutePath,
    syncSidebarWidthEffect,
  } from "$lib/app/bootstrap";
  import { layout } from "$lib/components/layout/layout.svelte";
  import { applyPageMeta, metaFromRoute } from "$lib/seo/meta";

  let bootstrapped = $state(false);

  $effect(() => {
    return syncSidebarWidthEffect();
  });

  $effect(() => {
    runInitialBootstrap((value) => {
      bootstrapped = value;
    });
  });

  $effect(() => {
    ensureLibrariesAfterAuth(bootstrapped);
  });

  $effect(() => {
    return connectRealtimeServices(bootstrapped);
  });

  $effect(() => {
    publishDevicePlayback(bootstrapped);
  });

  $effect(() => {
    loadConnectionSettings(bootstrapped);
  });

  $effect(() => {
    return initSourceConnection(bootstrapped);
  });

  $effect(() => {
    syncRoutePath();
  });

  $effect(() => {
    return bindNativeMediaEffect(bootstrapped);
  });

  $effect(() => {
    return bindPlaybackLifecycleEffect(bootstrapped);
  });

  $effect(() => {
    return bindWindowChromeEffect(bootstrapped);
  });

  $effect(() => {
    return bindDesktopIntegrationEffect(bootstrapped);
  });

  $effect(() => {
    return bindWindowCloseEffect(bootstrapped);
  });

  $effect(() => {
    return bindAndroidMediaActions(bootstrapped);
  });

  $effect(() => {
    return bindKeyboardShortcutsEffect(bootstrapped);
  });

  $effect(() => {
    syncAuthRouting(bootstrapped);
  });

  const match = $derived(router.match(routes));

  $effect(() => {
    const current = match;
    const path = `${router.pathname}${router.search}`;
    if (!current) {
      applyPageMeta({
        title: "Not found",
        description: APP_DESCRIPTION,
        path,
        index: false,
      });
      return;
    }
    applyPageMeta(metaFromRoute(current.route, path));
  });

  const playerChrome = $derived(music.playerVisible);
  const playerMini = $derived(
    music.playerLayout === "mini" && music.playerVisible,
  );
  const showShell = $derived(
    !auth.needsAccountLogin || router.pathname === "/account/login",
  );
  const showMobileNav = $derived(
    showShell &&
      layout.isMobileViewport &&
      !(layout.tvMode && Boolean(music.currentTrack)),
  );
  const nowPlayingMobile = $derived(
    layout.isMobileViewport && music.onNowPlayingRoute,
  );
  const compatBlocked = $derived(bootstrapped && getCompatState().blocked);
</script>

{#if !bootstrapped || auth.loading}
  <div class="boot">
    <Spinner />
  </div>
{:else if auth.error && !auth.statusLoaded}
  <div class="boot boot--error">
    <p>Could not reach the {APP_NAME} server.</p>
    <p class="boot__detail">{auth.error}</p>
    <button
      type="button"
      class="boot__retry"
      onclick={() => {
        bootstrapped = false;
        runInitialBootstrap((value) => {
          bootstrapped = value;
        });
      }}
    >
      Retry
    </button>
  </div>
{:else if compatBlocked}
  <CompatBanner />
{:else if match}
  <CompatBanner />
  <WindowChrome />
  <div
    class="app-root"
    class:app-root--player={playerChrome && showShell}
    class:app-root--mini-player={playerMini && showShell}
    class:app-root--mobile-nav={showMobileNav}
    class:app-root--now-playing-mobile={nowPlayingMobile}
  >
    <RouteBoundary>
      <RouteOutlet
        load={match.route.load}
        component={match.route.component}
        path={match.route.path}
        params={match.match.params}
      />
    </RouteBoundary>
  </div>
{:else}
  <div class="not-found">
    <h1>404</h1>
    <p>That page does not exist.</p>
    <a href="/music">Go home</a>
  </div>
{/if}

{#if showShell}
  <LocalLibraryScanBanner />
  <MusicPlayer />
{/if}
<KeyboardHelp />
<KeybindFeedback />
<CommandPalette />
<ClosePrompt />
<ConfirmDialog />
<SourceSwitchOverlay />
<ToastContainer />

<style>
  .app-root {
    flex: 1;
    min-height: 0;
    display: flex;
    flex-direction: column;
  }

  .boot {
    min-height: 100vh;
    display: grid;
    place-content: center;
    background: var(--jb-bg);
  }

  .boot--error {
    gap: var(--jb-space-3);
    padding: var(--jb-space-6);
    text-align: center;
    color: var(--jb-text);
  }

  .boot__detail {
    margin: 0;
    color: var(--jb-text-muted);
    font-size: 0.875rem;
  }

  .boot__retry {
    justify-self: center;
    border: 1px solid var(--jb-border);
    border-radius: var(--jb-radius-full);
    background: var(--jb-surface);
    color: var(--jb-text);
    font: inherit;
    font-weight: 600;
    padding: 0.5rem 1.25rem;
    cursor: pointer;
  }

  .not-found {
    min-height: 100vh;
    display: grid;
    place-content: center;
    text-align: center;
    gap: var(--jb-space-3);
    background: var(--jb-bg);
    color: var(--jb-text);
  }

  .not-found a {
    color: var(--jb-accent);
  }
</style>
