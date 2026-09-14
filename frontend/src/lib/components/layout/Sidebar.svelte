<script lang="ts">
  import Link from "$lib/router/Link.svelte";
  import { APP_NAME } from "$lib/brand";
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";
  import AppLogo from "$lib/components/ui/AppLogo.svelte";
  import InstanceSwitcher from "$lib/components/instances/InstanceSwitcher.svelte";
  import ThemeToggle from "$lib/components/ui/ThemeToggle.svelte";
  import ConnectionStatus from "$lib/components/ui/ConnectionStatus.svelte";
  import { music } from "$lib/config/music.svelte";
  import { instances } from "$lib/features/instances/store.svelte";
  import { localLibraries } from "$lib/features/local-libraries/store.svelte";
  import { auth } from "$lib/features/auth/store.svelte";
  import { videoFeature } from "$lib/video/feature.svelte";
  import { extensionFeatures } from "$lib/extensions/features.svelte";
  import { Cap, supports } from "$lib/compat";
  import { logClientError } from "$lib/core/logger";
  import { layout } from "./layout.svelte";

  interface Props {
    collapsed?: boolean;
  }

  let { collapsed = false }: Props = $props();

  const primaryNav = [
    { href: "/music", label: "Home", icon: "music", cap: Cap.browse },
    {
      href: "/music/now-playing",
      label: "Now playing",
      icon: "disc",
      cap: Cap.playback,
    },
    { href: "/music/search", label: "Search", icon: "search", cap: Cap.browse },
    {
      href: "/music/artists",
      label: "Artists",
      icon: "accountMusic",
      cap: Cap.browse,
    },
    { href: "/music/albums", label: "Albums", icon: "album", cap: Cap.browse },
    { href: "/music/genres", label: "Genres", icon: "tag", cap: Cap.browse },
    {
      href: "/music/playlists",
      label: "Playlists",
      icon: "listMusic",
      cap: Cap.playlists,
    },
    {
      href: "/music/shared",
      label: "Shared with you",
      icon: "share",
      cap: Cap.shared,
    },
    {
      href: "/music/favorites",
      label: "Favorites",
      icon: "star",
      cap: Cap.favorites,
    },
    {
      href: "/music/history",
      label: "History",
      icon: "history",
      cap: Cap.history,
    },
    { href: "/music/videos", label: "Videos", icon: "video", cap: Cap.videos },
  ];

  const visiblePrimaryNav = $derived.by(() => {
    let items = primaryNav.filter((item) => supports(item.cap));
    if (!(auth.enabled && auth.authenticated)) {
      items = items.filter((item) => item.href !== "/music/shared");
    }
    if (!videoFeature.enabled) {
      items = items.filter((item) => item.href !== "/music/videos");
    }
    return items;
  });

  const toolNav = $derived(
    localLibraries.active?.id &&
      extensionFeatures.metadata &&
      supports(Cap.metadataEditor)
      ? [{ href: "/music/metadata", label: "Metadata", icon: "autoFix" }]
      : [],
  );

  const showSettings = $derived(!auth.demoMode && supports(Cap.settings));

  $effect(() => {
    void videoFeature.refresh();
  });

  $effect(() => {
    if (!instances.activeId) return;
    void import("$lib/music/api")
      .then((m) => m.getMusicStatus())
      .then((status) => {
        music.status = status;
      })
      .catch((err) => logClientError(err, "sidebar"));
  });

  function closeOnNavigate() {
    if (window.matchMedia("(max-width: 768px)").matches) {
      layout.closeSidebar();
    }
  }

  const navIconSize = $derived(collapsed ? 22 : 20);
  const collapseIconSize = $derived(collapsed ? 20 : 18);
  const brandLogoSize = $derived(collapsed ? 40 : 32);
</script>

<nav
  class="sidebar jb-no-drag"
  class:sidebar--collapsed={collapsed}
  aria-label="Main navigation"
>
  <Link
    href="/music"
    class="sidebar__brand"
    onclick={closeOnNavigate}
    aria-label="{APP_NAME} home"
  >
    <AppLogo size={brandLogoSize} />
    <div class="sidebar__brand-copy">
      <span class="sidebar__title">{APP_NAME}</span>
      <span class="sidebar__subtitle">
        {instances.active?.serverName ?? music.serverName}
      </span>
    </div>
  </Link>

  {#if !auth.demoMode}
    <div class="sidebar__instance">
      <InstanceSwitcher compact />
    </div>
  {/if}

  <div class="sidebar__nav">
    {#each visiblePrimaryNav as item (item.href)}
      <Link
        href={item.href}
        class="sidebar__link"
        activeClass="sidebar__link--active"
        onclick={closeOnNavigate}
        aria-label={item.label}
      >
        <MdiIcon name={item.icon} size={navIconSize} />
        <span class="sidebar__label">{item.label}</span>
      </Link>
    {/each}

    {#if toolNav.length > 0}
      <p class="sidebar__section-label">Library tools</p>
      {#each toolNav as item (item.href)}
        <Link
          href={item.href}
          class="sidebar__link"
          activeClass="sidebar__link--active"
          onclick={closeOnNavigate}
          aria-label={item.label}
        >
          <MdiIcon name={item.icon} size={navIconSize} />
          <span class="sidebar__label">{item.label}</span>
        </Link>
      {/each}
    {/if}

    {#if showSettings}
      <Link
        href="/settings/profile"
        class="sidebar__link"
        activeClass="sidebar__link--active"
        matchPrefix="/settings"
        onclick={closeOnNavigate}
        aria-label="Settings"
      >
        <MdiIcon name="settings" size={navIconSize} />
        <span class="sidebar__label">Settings</span>
      </Link>
    {/if}
  </div>

  <div class="sidebar__footer jb-no-drag">
    <div class="sidebar__footer-tools">
      <ConnectionStatus embedded />
      <ThemeToggle embedded />
    </div>
    <button
      type="button"
      class="sidebar__collapse"
      onclick={() => layout.toggleCollapsed()}
      aria-label={collapsed ? "Expand sidebar" : "Collapse sidebar"}
      aria-expanded={!collapsed}
      title={collapsed ? "Expand sidebar" : "Collapse sidebar"}
    >
      <MdiIcon
        name={collapsed ? "panelLeftOpen" : "panelLeftClose"}
        size={collapseIconSize}
      />
    </button>
  </div>
</nav>

<style>
  /* The inner rail keeps a fixed expanded width. The aside clips it while
     animating width, and clip-path mirrors the clip so content never
     repaints outside the rail when the aside allows overflow (custom
     window chrome). Item positions never recompute during the transition. */
  .sidebar {
    position: relative;
    display: flex;
    flex-direction: column;
    width: var(--jb-sidebar-width);
    flex-shrink: 0;
    height: 100%;
    min-height: 0;
    padding: var(--jb-space-4);
    background: var(--jb-bg-elevated);
    gap: var(--jb-space-6);
    clip-path: inset(0 0 0 0);
    transition: clip-path var(--jb-transition);
  }

  .sidebar--collapsed {
    clip-path: inset(
      0 calc(var(--jb-sidebar-width) - var(--jb-sidebar-width-collapsed)) 0 0
    );
  }

  :global(html[data-custom-window-chrome="true"]) .sidebar {
    padding-top: calc(var(--jb-window-chrome-height, 2rem) + var(--jb-space-2));
    gap: var(--jb-space-4);
  }

  .sidebar :global(.sidebar__brand) {
    display: flex;
    align-items: center;
    gap: var(--jb-space-3);
    min-height: 2.5rem;
    flex-shrink: 0;
    color: inherit;
    text-decoration: none;
    border-radius: var(--jb-radius-md);
  }

  .sidebar :global(.sidebar__brand:hover .sidebar__title) {
    color: var(--jb-accent);
  }

  .sidebar :global(.sidebar__brand:focus-visible) {
    outline: 2px solid var(--jb-accent);
    outline-offset: 2px;
  }

  :global(html[data-custom-window-chrome="true"])
    .sidebar
    :global(.sidebar__brand) {
    min-height: 2.5rem;
    padding-inline: 0;
    margin-inline: 0;
    margin-top: 0;
    margin-bottom: 0;
    border-bottom: none;
  }

  .sidebar__brand-copy {
    display: flex;
    flex-direction: column;
    min-width: 0;
    gap: 0.125rem;
    line-height: 1.2;
  }

  .sidebar__instance {
    flex-shrink: 0;
    min-width: 0;
  }

  /* Text and tool rows fade in place instead of being removed, so the
     pinned-width layout above them never reflows mid transition. */
  .sidebar__label,
  .sidebar__brand-copy,
  .sidebar__instance,
  .sidebar__section-label,
  .sidebar__footer-tools {
    transition:
      opacity var(--jb-transition),
      visibility var(--jb-transition);
  }

  .sidebar--collapsed .sidebar__label,
  .sidebar--collapsed .sidebar__brand-copy,
  .sidebar--collapsed .sidebar__instance,
  .sidebar--collapsed .sidebar__section-label,
  .sidebar--collapsed .sidebar__footer-tools {
    opacity: 0;
    visibility: hidden;
  }

  .sidebar__title {
    font-weight: 700;
    color: var(--jb-text);
    line-height: 1.2;
  }

  .sidebar__subtitle {
    font-size: 0.8125rem;
    color: var(--jb-text-muted);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    line-height: 1.2;
  }

  .sidebar__nav {
    display: flex;
    flex-direction: column;
    flex: 1;
    min-height: 0;
    overflow-y: auto;
    gap: var(--jb-space-1);
  }

  .sidebar__section-label {
    margin: var(--jb-space-3) 0 var(--jb-space-1);
    padding: 0 var(--jb-space-3);
    font-size: 0.6875rem;
    font-weight: 700;
    letter-spacing: 0.06em;
    text-transform: uppercase;
    color: var(--jb-text-muted);
  }

  .sidebar :global(.sidebar__link) {
    display: flex;
    align-items: center;
    gap: var(--jb-space-3);
    padding: var(--jb-space-3);
    border-radius: var(--jb-radius-md);
    color: var(--jb-text-muted);
    text-decoration: none;
    transition:
      background var(--jb-transition),
      color var(--jb-transition),
      box-shadow var(--jb-transition);
  }

  .sidebar__label {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .sidebar :global(.sidebar__link:hover) {
    background: var(--jb-surface-hover);
    color: var(--jb-text);
  }

  /* Raised rather than pressed: the active item lifts off the rail with a
     drop shadow, a light top edge, and a thin accent hairline. */
  .sidebar :global(.sidebar__link--active) {
    background: var(--jb-surface-raised);
    color: var(--jb-text);
    box-shadow:
      var(--jb-shadow-sm),
      inset 0 0 0 1px color-mix(in srgb, var(--jb-accent) 28%, var(--jb-border)),
      inset 0 1px 0 color-mix(in srgb, var(--jb-text) 9%, transparent);
  }

  .sidebar :global(.sidebar__link--active:hover) {
    background: var(--jb-surface-raised);
    color: var(--jb-text);
  }

  .sidebar__footer {
    display: none;
    flex-shrink: 0;
    margin-top: auto;
    min-height: calc(var(--jb-space-3) + 2.25rem);
    padding-top: var(--jb-space-3);
    border-top: 1px solid var(--jb-border);
    position: relative;
    z-index: 1;
    align-items: center;
    gap: var(--jb-space-2);
  }

  .sidebar__footer-tools {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: var(--jb-space-2);
    min-width: 0;
    flex: 1;
    /* Keep the centered group clear of the absolutely positioned collapse
       button so it stays optically centered in the sidebar. */
    padding-right: calc(2.25rem + var(--jb-space-2));
  }

  /* The toggle rides the aside edge via transform, staying in sync with the
     width transition instead of jumping between footer slots. */
  .sidebar__collapse {
    position: absolute;
    left: 0;
    bottom: 0;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 2.25rem;
    height: 2.25rem;
    border: none;
    border-radius: var(--jb-radius-md);
    background: transparent;
    color: var(--jb-text-muted);
    cursor: pointer;
    transform: translateX(
      calc(
        var(--jb-sidebar-current-width, var(--jb-sidebar-width)) -
          var(--jb-space-8) - 2.25rem
      )
    );
    transition:
      transform var(--jb-transition),
      background var(--jb-transition),
      color var(--jb-transition);
  }

  .sidebar__collapse:hover {
    background: var(--jb-surface-hover);
    color: var(--jb-text);
  }

  @media (prefers-reduced-motion: reduce) {
    .sidebar,
    .sidebar__collapse,
    .sidebar__label,
    .sidebar__brand-copy,
    .sidebar__instance,
    .sidebar__section-label,
    .sidebar__footer-tools {
      transition: none;
    }
  }

  @media (max-width: 768px) {
    .sidebar {
      width: 100%;
      padding-top: calc(env(safe-area-inset-top, 0px) + var(--jb-space-6));
      padding-bottom: calc(
        env(safe-area-inset-bottom, 0px) + var(--jb-space-4)
      );
    }

    .sidebar__footer {
      display: flex;
    }

    .sidebar__collapse {
      display: none;
    }
  }

  @media (min-width: 769px) {
    .sidebar__footer {
      display: flex;
      align-items: center;
    }
  }
</style>
