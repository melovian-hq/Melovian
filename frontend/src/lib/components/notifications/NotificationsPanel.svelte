<script lang="ts">
  import { Popover } from "bits-ui";
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";
  import Spinner from "$lib/components/ui/Spinner.svelte";
  import { notifications } from "$lib/notifications/notifications.svelte";
  import type { AppNotification } from "$lib/notifications/api";

  function kindIcon(kind: string): string {
    switch (kind) {
      case "share_received":
        return "share";
      case "party_invite":
      case "party_joined":
      case "party_left":
      case "party_ended":
        return "headphones";
      default:
        return "bell";
    }
  }

  function formatWhen(iso: string): string {
    const t = Date.parse(iso);
    if (!Number.isFinite(t)) return "";
    const delta = Date.now() - t;
    const mins = Math.floor(delta / 60000);
    if (mins < 1) return "just now";
    if (mins < 60) return `${mins}m`;
    const hours = Math.floor(mins / 60);
    if (hours < 24) return `${hours}h`;
    const days = Math.floor(hours / 24);
    return `${days}d`;
  }

  function onItemClick(item: AppNotification) {
    if (item.kind === "party_invite") {
      void notifications.acceptPartyInvite(item);
      return;
    }
    void notifications.openNotification(item);
  }

  // The bell trigger lives in TopBar and toggles the store itself. If its click
  // also counted as an outside interaction the popover would close on pointerup
  // and immediately reopen on click, so it is excluded here.
  function onInteractOutside(event: PointerEvent) {
    const target = event.target as HTMLElement | null;
    if (target?.closest(".topbar__bell")) event.preventDefault();
  }

  function preventAutoFocus(event: Event) {
    event.preventDefault();
  }
</script>

<Popover.Root bind:open={notifications.panelOpen}>
  {#if notifications.panelOpen}
    <Popover.Portal>
      <div class="notif-root">
        <button
          type="button"
          class="notif-backdrop"
          aria-label="Close notifications"
          onclick={() => notifications.closePanel()}
        ></button>
        <Popover.ContentStatic
          class="notif-panel"
          aria-label="Notifications"
          trapFocus={false}
          {onInteractOutside}
          onOpenAutoFocus={preventAutoFocus}
          onCloseAutoFocus={preventAutoFocus}
        >
          {#snippet child({ props })}
            <div {...props}>
              <header class="notif-panel__head">
                <div class="notif-panel__title">
                  <MdiIcon name="bell" size={16} />
                  <h3>Notifications</h3>
                  {#if notifications.unread > 0}
                    <span class="notif-panel__badge"
                      >{notifications.unread}</span
                    >
                  {/if}
                </div>
                <div class="notif-panel__actions">
                  {#if notifications.unread > 0}
                    <button
                      type="button"
                      class="notif-panel__text-btn"
                      onclick={() => void notifications.markAllRead()}
                    >
                      Mark all read
                    </button>
                  {/if}
                  <button
                    type="button"
                    class="notif-panel__icon-btn"
                    aria-label="Close"
                    onclick={() => notifications.closePanel()}
                  >
                    <MdiIcon name="x" size={16} />
                  </button>
                </div>
              </header>

              <div class="notif-panel__body">
                {#if notifications.loading && notifications.items.length === 0}
                  <div class="notif-panel__empty">
                    <Spinner />
                  </div>
                {:else if notifications.items.length === 0}
                  <div class="notif-panel__empty">
                    <p>No notifications yet</p>
                  </div>
                {:else}
                  <ul class="notif-list">
                    {#each notifications.items as item (item.id)}
                      <li>
                        <div
                          class="notif-item"
                          class:notif-item--unread={!item.read}
                        >
                          <button
                            type="button"
                            class="notif-item__main"
                            onclick={() => onItemClick(item)}
                          >
                            <span class="notif-item__icon" aria-hidden="true">
                              <MdiIcon name={kindIcon(item.kind)} size={16} />
                            </span>
                            <span class="notif-item__copy">
                              <span class="notif-item__title">{item.title}</span
                              >
                              {#if item.body}
                                <span class="notif-item__body">{item.body}</span
                                >
                              {/if}
                            </span>
                            <span class="notif-item__when"
                              >{formatWhen(item.createdAt)}</span
                            >
                          </button>
                          <button
                            type="button"
                            class="notif-item__delete"
                            aria-label="Dismiss"
                            onclick={() => void notifications.remove(item.id)}
                          >
                            <MdiIcon name="x" size={12} />
                          </button>
                        </div>
                      </li>
                    {/each}
                  </ul>
                {/if}
              </div>
            </div>
          {/snippet}
        </Popover.ContentStatic>
      </div>
    </Popover.Portal>
  {/if}
</Popover.Root>

<style>
  .notif-root {
    position: fixed;
    inset: 0;
    z-index: 120;
    pointer-events: none;
  }

  .notif-backdrop {
    position: absolute;
    inset: 0;
    border: none;
    background: transparent;
    pointer-events: auto;
    cursor: default;
  }

  .notif-panel {
    position: absolute;
    top: calc(
      var(--jb-window-chrome-offset, 0px) + var(--jb-space-4) + 3.25rem +
        env(safe-area-inset-top, 0px)
    );
    right: calc(var(--jb-space-4) + var(--jb-window-controls-inset, 0px));
    width: min(22rem, calc(100vw - var(--jb-space-6)));
    max-height: min(28rem, calc(100vh - 6rem));
    display: flex;
    flex-direction: column;
    border: 1px solid var(--jb-border);
    border-radius: var(--jb-radius-xl);
    background: var(--jb-island-bg);
    backdrop-filter: blur(16px);
    box-shadow: var(--jb-shadow-lg);
    pointer-events: auto;
    overflow: hidden;
  }

  .notif-panel__head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--jb-space-2);
    padding: var(--jb-space-3) var(--jb-space-4);
    border-bottom: 1px solid var(--jb-border);
  }

  .notif-panel__title {
    display: flex;
    align-items: center;
    gap: var(--jb-space-2);
    min-width: 0;
  }

  .notif-panel__title h3 {
    margin: 0;
    font-size: 0.9375rem;
    font-weight: 600;
  }

  .notif-panel__badge {
    display: inline-grid;
    place-content: center;
    min-width: 1.25rem;
    height: 1.25rem;
    padding: 0 0.35rem;
    border-radius: var(--jb-radius-full);
    background: var(--jb-accent);
    color: var(--jb-accent-text);
    font-size: 0.6875rem;
    font-weight: 700;
  }

  .notif-panel__actions {
    display: flex;
    align-items: center;
    gap: var(--jb-space-1);
  }

  .notif-panel__text-btn {
    border: none;
    background: none;
    color: var(--jb-accent);
    font: inherit;
    font-size: 0.75rem;
    font-weight: 600;
    cursor: pointer;
  }

  .notif-panel__icon-btn {
    display: grid;
    place-content: center;
    width: 1.75rem;
    height: 1.75rem;
    border: none;
    border-radius: var(--jb-radius-md);
    background: transparent;
    color: var(--jb-text-subtle);
    cursor: pointer;
  }

  .notif-panel__body {
    overflow: auto;
    min-height: 8rem;
  }

  .notif-panel__empty {
    display: grid;
    place-content: center;
    gap: var(--jb-space-2);
    min-height: 8rem;
    color: var(--jb-text-muted);
    font-size: 0.875rem;
  }

  .notif-list {
    list-style: none;
    margin: 0;
    padding: 0;
  }

  .notif-item {
    display: grid;
    grid-template-columns: 1fr auto;
    align-items: start;
    gap: var(--jb-space-2);
    border-bottom: 1px solid var(--jb-border);
  }

  .notif-item:hover {
    background: color-mix(in srgb, var(--jb-surface) 70%, transparent);
  }

  .notif-item--unread {
    background: color-mix(in srgb, var(--jb-accent) 6%, transparent);
  }

  .notif-item__main {
    display: grid;
    grid-template-columns: auto 1fr auto;
    gap: var(--jb-space-3);
    width: 100%;
    padding: var(--jb-space-3) 0 var(--jb-space-3) var(--jb-space-4);
    border: none;
    background: transparent;
    color: inherit;
    text-align: left;
    font: inherit;
    cursor: pointer;
  }

  .notif-item__icon {
    display: grid;
    place-content: center;
    width: 1.75rem;
    height: 1.75rem;
    border-radius: var(--jb-radius-full);
    background: var(--jb-surface);
    color: var(--jb-text-muted);
  }

  .notif-item__copy {
    display: grid;
    gap: 0.15rem;
    min-width: 0;
  }

  .notif-item__title {
    font-size: 0.8125rem;
    font-weight: 600;
    color: var(--jb-text);
  }

  .notif-item__body {
    font-size: 0.75rem;
    color: var(--jb-text-muted);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .notif-item__when {
    font-size: 0.6875rem;
    color: var(--jb-text-subtle);
    padding-top: 0.15rem;
  }

  .notif-item__delete {
    display: grid;
    place-content: center;
    width: 1.75rem;
    height: 1.75rem;
    margin: var(--jb-space-3) var(--jb-space-3) 0 0;
    border: none;
    border-radius: var(--jb-radius-sm);
    background: transparent;
    color: var(--jb-text-subtle);
    cursor: pointer;
    opacity: 0;
  }

  .notif-item:hover .notif-item__delete {
    opacity: 1;
  }

  @media (max-width: 768px) {
    .notif-panel {
      top: calc(
        var(--jb-window-chrome-offset, 0px) + var(--jb-space-2) + 3rem +
          env(safe-area-inset-top, 0px)
      );
      right: var(--jb-space-3);
      left: var(--jb-space-3);
      width: auto;
    }

    .notif-item__delete {
      opacity: 1;
    }
  }
</style>
