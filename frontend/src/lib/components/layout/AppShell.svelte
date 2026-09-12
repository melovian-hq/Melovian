<script lang="ts">
  import Sidebar from "./Sidebar.svelte";
  import TopBar from "./TopBar.svelte";
  import BottomNav from "./BottomNav.svelte";
  import OfflineBanner from "$lib/components/ui/OfflineBanner.svelte";
  import { music } from "$lib/config/music.svelte";
  import { extensionFeatures } from "$lib/extensions/features.svelte";
  import { decorateTrack } from "$lib/extensions/registry";
  import type { RouteContentLayout } from "$lib/router/router.svelte";
  import { layout } from "./layout.svelte";

  interface Props {
    children?: import("svelte").Snippet;
    /** Drop the chrome (sidebar, topbar, bottom nav) for bare routes. */
    bare?: boolean;
    /** Content area layout for the current route. */
    content?: RouteContentLayout;
  }

  let { children, bare = false, content = "default" }: Props = $props();

  const hideChromeForTv = $derived(
    layout.tvMode && Boolean(music.currentTrack),
  );
  const showBottomNav = $derived(
    !bare && layout.isMobileViewport && !hideChromeForTv,
  );
  const sidebarCollapsed = $derived(
    layout.isMobileViewport ? false : layout.sidebarCollapsed,
  );
  const trackDecoration = $derived(
    music.currentTrack ? decorateTrack(music.currentTrack) : {},
  );
  $effect(() => {
    void extensionFeatures.availableThemes;
    extensionFeatures.syncFromTrackDecoration(trackDecoration);
  });

  function onKeydown(event: KeyboardEvent) {
    if (event.key !== "Escape") return;
    if (!layout.sidebarOpen) return;
    layout.closeSidebar();
  }
</script>

<svelte:window onkeydown={onKeydown} />

<div
  class="app-shell"
  class:app-shell--bare={bare}
  class:app-shell--collapsed={layout.sidebarCollapsed &&
    !layout.isMobileViewport}
  class:app-shell--open={layout.sidebarOpen}
>
  {#if !bare}
    {#if layout.sidebarOpen}
      <button
        type="button"
        class="app-shell__backdrop"
        onclick={() => layout.closeSidebar()}
        aria-label="Close navigation"
      ></button>
    {/if}

    <aside class="app-shell__sidebar jb-no-drag">
      <Sidebar collapsed={sidebarCollapsed} />
    </aside>
  {/if}

  <div class="app-shell__main">
    {#if !bare && !hideChromeForTv}
      <TopBar />
    {/if}
    {#if !bare}
      <OfflineBanner />
    {/if}
    <main
      class="app-shell__content"
      class:app-shell__content--compact={!bare && content === "compact"}
      class:app-shell__content--fill={!bare && content === "fill"}
      class:app-shell__content--bare={bare}
    >
      {@render children?.()}
    </main>
  </div>

  {#if showBottomNav}
    <BottomNav />
  {/if}
</div>

<style>
  .app-shell {
    --jb-sidebar-current-width: var(--jb-sidebar-width);
    flex: 1;
    display: flex;
    width: 100%;
    height: 100%;
    min-height: 0;
    overflow: hidden;
    background: var(--jb-bg);
    padding: env(safe-area-inset-top, 0px) env(safe-area-inset-right, 0px)
      env(safe-area-inset-bottom, 0px) env(safe-area-inset-left, 0px);
  }

  .app-shell--collapsed {
    --jb-sidebar-current-width: var(--jb-sidebar-width-collapsed);
  }

  .app-shell__sidebar {
    flex-shrink: 0;
    width: var(--jb-sidebar-current-width);
    height: 100%;
    min-height: 0;
    overflow: hidden;
    transition: width var(--jb-transition);
    z-index: 55;
  }

  :global(html[data-custom-window-chrome="true"]) .app-shell__sidebar {
    overflow: visible;
  }

  .app-shell__main {
    flex: 1;
    min-width: 0;
    min-height: 0;
    display: flex;
    flex-direction: column;
    overflow: hidden;
    color: var(--jb-text);
  }

  .app-shell__content {
    --jb-page-pad-x: var(--jb-space-6);
    --jb-page-pad-top: calc(var(--jb-topbar-height) + var(--jb-space-2));
    flex: 1;
    min-height: 0;
    overflow-x: hidden;
    overflow-y: auto;
    overscroll-behavior: contain;
    -webkit-overflow-scrolling: touch;
    padding: var(--jb-page-pad-top) var(--jb-page-pad-x) var(--jb-space-8);
    width: 100%;
    color: var(--jb-text);
  }

  .app-shell__content--compact {
    --jb-page-pad-top: calc(
      var(--jb-window-chrome-offset, 0px) + var(--jb-space-4)
    );
    padding-top: var(--jb-page-pad-top);
  }

  .app-shell__content--fill {
    /* Match compactTop: TopBar floats top-right, so content can start higher. */
    --jb-fill-pad-top: calc(
      var(--jb-window-chrome-offset, 0px) + var(--jb-space-4)
    );
    overflow: hidden;
    display: flex;
    flex-direction: column;
    padding-top: 0;
    padding-left: 0;
    padding-right: 0;
    padding-bottom: 0;
  }

  .app-shell__content--fill > :global(*) {
    flex: 1 1 auto;
    min-height: 0;
    min-width: 0;
    width: 100%;
    height: 100%;
  }

  .app-shell__content--bare {
    padding: 0;
  }

  .app-shell__backdrop {
    display: none;
  }

  @media (max-width: 768px) {
    .app-shell {
      --jb-sidebar-current-width: 0px;
      /* Bottom nav owns safe-area inset. Avoid double padding. */
      padding-bottom: 0;
    }

    .app-shell--collapsed {
      --jb-sidebar-current-width: 0px;
    }

    .app-shell__sidebar {
      position: fixed;
      top: 0;
      left: 0;
      bottom: 0;
      width: min(var(--jb-sidebar-width), 85vw);
      transform: translateX(-100%);
      transition: transform var(--jb-transition);
    }

    .app-shell--open .app-shell__sidebar {
      transform: translateX(0);
    }

    .app-shell__backdrop {
      display: block;
      position: fixed;
      inset: 0;
      border: none;
      background: rgb(0 0 0 / 0.35);
      backdrop-filter: blur(2px);
      z-index: 35;
      cursor: pointer;
    }

    .app-shell__content {
      --jb-page-pad-x: var(--jb-space-4);
      --jb-page-pad-top: calc(var(--jb-topbar-height) + var(--jb-space-2));
      padding-top: var(--jb-page-pad-top);
      padding-left: var(--jb-page-pad-x);
      padding-right: var(--jb-page-pad-x);
    }

    .app-shell__content--compact {
      --jb-page-pad-top: calc(var(--jb-window-chrome-offset, 0px) + 3.25rem);
      padding-top: var(--jb-page-pad-top);
    }

    .app-shell__content--fill {
      --jb-fill-pad-top: calc(var(--jb-window-chrome-offset, 0px) + 3.25rem);
      padding-top: 0;
      padding-left: 0;
      padding-right: 0;
      padding-bottom: 0;
    }

    .app-shell__content--bare {
      padding: 0;
    }
  }

  @media (max-width: 480px) {
    .app-shell__content {
      --jb-page-pad-x: var(--jb-space-3);
      --jb-page-pad-top: calc(var(--jb-topbar-height) + var(--jb-space-3));
      padding-top: var(--jb-page-pad-top);
      padding-left: var(--jb-page-pad-x);
      padding-right: var(--jb-page-pad-x);
    }

    .app-shell__content--compact {
      --jb-page-pad-top: calc(var(--jb-window-chrome-offset, 0px) + 3.25rem);
      padding-top: var(--jb-page-pad-top);
    }

    .app-shell__content--fill {
      --jb-fill-pad-top: calc(var(--jb-window-chrome-offset, 0px) + 3.25rem);
      padding-top: 0;
      padding-left: 0;
      padding-right: 0;
      padding-bottom: 0;
    }

    .app-shell__content--bare {
      padding: 0;
    }
  }
</style>
