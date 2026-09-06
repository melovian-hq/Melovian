<script lang="ts">
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";
  import AvatarStack from "$lib/components/ui/AvatarStack.svelte";
  import NotificationsPanel from "$lib/components/notifications/NotificationsPanel.svelte";
  import { auth } from "$lib/features/auth/store.svelte";
  import { notifications } from "$lib/notifications/notifications.svelte";
  import { deviceSync } from "$lib/music/device-sync.svelte";
  import { nativeDesktopAvailable } from "$lib/config/runtime";
  import { handleTitleBarDoubleClick } from "$lib/desktop/window-chrome";
  import { layout } from "./layout.svelte";

  interface Props {
    class?: string;
    actions?: import("svelte").Snippet;
  }

  let { class: className = "", actions }: Props = $props();

  const showBell = $derived(auth.enabled && auth.authenticated);
  const unreadLabel = $derived(
    notifications.unread > 99 ? "99+" : String(notifications.unread),
  );
  const sessionStackMembers = $derived(
    deviceSync.sessionMembers.map((d) => ({
      seed: d.deviceId,
      label: d.username || d.name,
      self: d.deviceId === deviceSync.deviceId,
      active: d.isActivePlayer,
    })),
  );
  const showPresence = $derived(
    deviceSync.inListenTogether && sessionStackMembers.length > 0,
  );
  const presenceLabel = $derived(
    deviceSync.sessionLabel || "Listening together",
  );
</script>

<header class="topbar {className}">
  <div class="topbar__island topbar__menu jb-no-drag">
    <button
      type="button"
      class="topbar__menu-btn"
      onclick={() => layout.toggleSidebar()}
      aria-label={layout.sidebarOpen ? "Close navigation" : "Open navigation"}
      aria-expanded={layout.sidebarOpen}
    >
      <MdiIcon name="menu" size={18} />
    </button>
  </div>

  <div
    class="topbar__drag jb-drag"
    aria-hidden="true"
    ondblclick={() => {
      if (nativeDesktopAvailable()) void handleTitleBarDoubleClick();
    }}
  ></div>

  <div class="topbar__island topbar__actions jb-no-drag">
    {#if showPresence}
      <button
        type="button"
        class="topbar__presence"
        aria-label={presenceLabel}
        title={presenceLabel}
        aria-expanded={deviceSync.panelOpen}
        onclick={() => deviceSync.togglePanel()}
      >
        <AvatarStack
          members={sessionStackMembers}
          size={22}
          max={4}
          showTitles={auth.enabled}
        />
      </button>
    {/if}
    {#if showBell}
      <button
        type="button"
        class="topbar__bell"
        aria-label={notifications.unread > 0
          ? `Notifications, ${notifications.unread} unread`
          : "Notifications"}
        aria-expanded={notifications.panelOpen}
        onclick={() => notifications.togglePanel()}
      >
        <MdiIcon
          name={notifications.unread > 0 ? "bellBadge" : "bell"}
          size={18}
        />
        {#if notifications.unread > 0}
          <span class="topbar__bell-count">{unreadLabel}</span>
        {/if}
      </button>
    {/if}
    {#if actions}
      {@render actions()}
    {/if}
  </div>
</header>

{#if showBell}
  <NotificationsPanel />
{/if}

<style>
  .topbar {
    position: fixed;
    top: calc(
      var(--jb-window-chrome-offset, 0px) + var(--jb-space-4) +
        env(safe-area-inset-top, 0px)
    );
    left: calc(var(--jb-sidebar-current-width) + var(--jb-space-4));
    right: calc(var(--jb-space-4) + var(--jb-window-controls-inset, 0px));
    display: flex;
    align-items: start;
    justify-content: flex-end;
    gap: var(--jb-space-3);
    z-index: 30;
    pointer-events: none;
    transition: left var(--jb-transition);
  }

  .topbar__drag {
    flex: 1;
    min-width: 0;
    align-self: stretch;
    min-height: 2.75rem;
    pointer-events: auto;
  }

  :global(html:not([data-custom-window-chrome="true"])) .topbar__drag {
    display: none;
  }

  .topbar__menu {
    display: none;
  }

  .topbar__island {
    pointer-events: auto;
  }

  .topbar__menu-btn,
  .topbar__bell,
  .topbar__presence {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    height: 2.75rem;
    border: 1px solid var(--jb-island-border);
    border-radius: var(--jb-radius-full);
    background: var(--jb-island-bg);
    color: var(--jb-text);
    cursor: pointer;
    backdrop-filter: blur(16px);
    box-shadow: var(--jb-shadow-md);
    position: relative;
  }

  .topbar__menu-btn,
  .topbar__bell {
    width: 2.75rem;
  }

  .topbar__presence {
    width: auto;
    min-width: 2.75rem;
    padding: 0 0.45rem;
    --avatar-stack-ring: var(--jb-island-bg);
  }

  .topbar__bell-count {
    position: absolute;
    top: -0.15rem;
    right: -0.15rem;
    min-width: 1.1rem;
    height: 1.1rem;
    padding: 0 0.25rem;
    border-radius: var(--jb-radius-full);
    background: var(--jb-accent);
    color: var(--jb-accent-text);
    font-size: 0.625rem;
    font-weight: 700;
    line-height: 1.1rem;
    text-align: center;
  }

  .topbar__actions {
    display: flex;
    align-items: center;
    gap: var(--jb-space-2);
  }

  @media (max-width: 768px) {
    .topbar {
      top: calc(
        var(--jb-window-chrome-offset, 0px) + var(--jb-space-2) +
          env(safe-area-inset-top, 0px)
      );
      left: var(--jb-space-3);
      right: calc(var(--jb-space-3) + var(--jb-window-controls-inset, 0px));
      gap: var(--jb-space-2);
      justify-content: flex-end;
    }

    .topbar__drag {
      display: none;
    }

    /* Bottom nav replaces the hamburger on mobile. */
    .topbar__menu {
      display: none;
    }

    .topbar__menu-btn,
    .topbar__bell,
    .topbar__presence {
      height: 2.5rem;
      border-radius: var(--jb-radius-md);
      border-color: var(--jb-border);
      background: var(--jb-surface);
      backdrop-filter: none;
      box-shadow: none;
    }

    .topbar__menu-btn,
    .topbar__bell {
      width: 2.5rem;
    }

    .topbar__presence {
      min-width: 2.5rem;
      padding: 0 0.35rem;
      --avatar-stack-ring: var(--jb-surface);
    }

    .topbar__actions {
      gap: var(--jb-space-1);
    }
  }
</style>
