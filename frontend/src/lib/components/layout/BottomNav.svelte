<script lang="ts">
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";
  import Link from "$lib/router/Link.svelte";
  import { router } from "$lib/router/router.svelte";
  import { layout } from "./layout.svelte";
  import { bottomNavActiveTab, type BottomNavTab } from "./bottom-nav-active";

  const active = $derived(
    bottomNavActiveTab(router.pathname, layout.sidebarOpen),
  );

  const tabs: {
    id: BottomNavTab;
    label: string;
    href?: string;
    icon: string;
  }[] = [
    { id: "home", label: "Home", href: "/music", icon: "music" },
    { id: "search", label: "Search", href: "/music/search", icon: "search" },
    { id: "library", label: "Library", href: "/music/albums", icon: "album" },
    {
      id: "playlists",
      label: "Playlists",
      href: "/music/playlists",
      icon: "listMusic",
    },
    { id: "more", label: "More", icon: "dotsHorizontal" },
  ];

  function onMore() {
    layout.toggleSidebar();
  }

  function onTabNavigate() {
    if (layout.sidebarOpen) layout.closeSidebar();
  }
</script>

<nav class="bottom-nav jb-no-drag" aria-label="Primary">
  {#each tabs as tab (tab.id)}
    {#if tab.href}
      <Link
        href={tab.href}
        class="bottom-nav__item{active === tab.id
          ? ' bottom-nav__item--active'
          : ''}"
        aria-current={active === tab.id ? "page" : undefined}
        onclick={onTabNavigate}
      >
        <MdiIcon name={tab.icon} size={22} />
        <span class="bottom-nav__label">{tab.label}</span>
      </Link>
    {:else}
      <button
        type="button"
        class="bottom-nav__item"
        class:bottom-nav__item--active={active === tab.id}
        aria-current={active === tab.id ? "true" : undefined}
        aria-expanded={layout.sidebarOpen}
        aria-label="More navigation"
        onclick={onMore}
      >
        <MdiIcon name={tab.icon} size={22} />
        <span class="bottom-nav__label">{tab.label}</span>
      </button>
    {/if}
  {/each}
</nav>

<style>
  .bottom-nav {
    position: fixed;
    left: 0;
    right: 0;
    bottom: 0;
    z-index: 45;
    display: flex;
    align-items: stretch;
    justify-content: space-around;
    height: calc(
      var(--jb-bottom-nav-height) + env(safe-area-inset-bottom, 0px)
    );
    padding-bottom: env(safe-area-inset-bottom, 0px);
    border-top: 1px solid var(--jb-border);
    background: color-mix(in srgb, var(--jb-bg-elevated) 96%, transparent);
    backdrop-filter: blur(20px);
    box-shadow: 0 -8px 24px rgb(0 0 0 / 0.12);
  }

  .bottom-nav :global(.bottom-nav__item),
  .bottom-nav__item {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 0.15rem;
    min-width: 0;
    min-height: 2.75rem;
    padding: var(--jb-space-1) var(--jb-space-1);
    border: none;
    border-radius: var(--jb-radius-md);
    background: transparent;
    color: var(--jb-text-subtle);
    text-decoration: none;
    font: inherit;
    cursor: pointer;
    -webkit-tap-highlight-color: transparent;
    transition:
      color var(--jb-transition),
      background var(--jb-transition);
  }

  .bottom-nav :global(.bottom-nav__item:hover),
  .bottom-nav__item:hover {
    color: var(--jb-text);
    background: var(--jb-surface-hover);
  }

  .bottom-nav :global(.bottom-nav__item:active),
  .bottom-nav__item:active {
    background: var(--jb-surface-active);
  }

  .bottom-nav :global(.bottom-nav__item--active),
  .bottom-nav__item--active {
    color: var(--jb-accent);
  }

  .bottom-nav :global(.bottom-nav__item--active:hover),
  .bottom-nav__item--active:hover {
    color: var(--jb-accent);
    background: color-mix(in srgb, var(--jb-accent) 12%, transparent);
  }

  .bottom-nav__label {
    font-size: 0.625rem;
    font-weight: 600;
    line-height: 1;
    letter-spacing: 0.01em;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    max-width: 100%;
  }
</style>
