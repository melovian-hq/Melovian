<script lang="ts">
  import { prefersReducedMotion } from "svelte/motion";
  import { fly } from "svelte/transition";
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";
  import { toast, type ToastItem, type ToastKind } from "$lib/ui/toast.svelte";
  import { music } from "$lib/config/music.svelte";
  import { layout } from "$lib/components/layout/layout.svelte";
  import { playerBottomInset } from "$lib/music/player-bottom-inset";

  const mobileTop = $derived(layout.isMobileViewport);
  const bottomOffset = $derived(
    mobileTop
      ? undefined
      : playerBottomInset(music.playerVisible, music.playerLayout, {
          mobileNav: layout.isMobileViewport,
        }),
  );
  const topOffset = $derived(
    mobileTop
      ? "calc(var(--jb-space-3) + env(safe-area-inset-top, 0px))"
      : undefined,
  );
  const motion = $derived(prefersReducedMotion.current ? 0 : 1);
  const flyY = $derived(mobileTop ? -12 : 12);

  function iconFor(kind: ToastKind): string {
    switch (kind) {
      case "success":
        return "check";
      case "error":
        return "alertCircle";
      case "warning":
        return "alert";
      default:
        return "info";
    }
  }

  function actionsFor(item: ToastItem): NonNullable<ToastItem["actions"]> {
    if (item.actions && item.actions.length > 0) return item.actions;
    return item.action ? [item.action] : [];
  }

  function runAction(item: ToastItem, onClick: () => void) {
    onClick();
    toast.dismiss(item.id);
  }
</script>

<div
  class="toast-container"
  class:toast-container--top={mobileTop}
  style:bottom={bottomOffset}
  style:top={topOffset}
  aria-live="polite"
>
  {#each toast.items as item (item.id)}
    <div
      class="toast toast--{item.kind}"
      role="status"
      transition:fly={{ y: flyY * motion, duration: 180 * motion }}
      onmouseenter={() => toast.pause(item.id)}
      onmouseleave={() => toast.resume(item.id)}
    >
      <span class="toast__icon" aria-hidden="true">
        <MdiIcon name={iconFor(item.kind)} size={18} />
      </span>
      <div class="toast__body">
        {#if item.title}
          <p class="toast__title">{item.title}</p>
        {/if}
        <p class="toast__message">{item.message}</p>
        {#if actionsFor(item).length > 0}
          <div class="toast__actions">
            {#each actionsFor(item) as action (action.label)}
              <button
                type="button"
                class="toast__action"
                onclick={() => runAction(item, action.onClick)}
              >
                {action.label}
              </button>
            {/each}
          </div>
        {/if}
      </div>
      <button
        type="button"
        class="toast__close"
        aria-label="Dismiss"
        onclick={() => toast.dismiss(item.id)}
      >
        <MdiIcon name="x" size={14} />
      </button>
    </div>
  {/each}
</div>

<style>
  .toast-container {
    position: fixed;
    right: max(var(--jb-space-4), env(safe-area-inset-right, 0px));
    left: auto;
    display: flex;
    flex-direction: column;
    gap: var(--jb-space-2);
    z-index: var(--jb-z-toast);
    pointer-events: none;
    max-width: min(24rem, calc(100vw - var(--jb-space-8)));
    padding-bottom: env(safe-area-inset-bottom, 0px);
  }

  .toast-container--top {
    padding-bottom: 0;
    padding-top: 0;
  }

  .toast {
    display: flex;
    align-items: flex-start;
    gap: var(--jb-space-3);
    padding: 0.75rem 0.875rem;
    border-radius: var(--jb-radius-xl);
    border: 1px solid var(--jb-border);
    border-left-width: 3px;
    background: var(--jb-island-bg);
    backdrop-filter: blur(12px);
    box-shadow: var(--jb-shadow-md);
    pointer-events: auto;
  }

  .toast--success {
    border-left-color: var(--jb-success);
    background: color-mix(in srgb, var(--jb-success) 10%, var(--jb-island-bg));
  }

  .toast--error {
    border-left-color: var(--jb-danger);
    background: color-mix(in srgb, var(--jb-danger) 10%, var(--jb-island-bg));
  }

  .toast--warning {
    border-left-color: var(--jb-warning);
    background: color-mix(in srgb, var(--jb-warning) 10%, var(--jb-island-bg));
  }

  .toast--info {
    border-left-color: var(--jb-accent);
    background: color-mix(in srgb, var(--jb-accent) 8%, var(--jb-island-bg));
  }

  .toast__icon {
    display: grid;
    place-content: center;
    flex-shrink: 0;
    margin-top: 0.1rem;
    color: var(--jb-text-muted);
  }

  .toast--success .toast__icon {
    color: var(--jb-success);
  }

  .toast--error .toast__icon {
    color: var(--jb-danger);
  }

  .toast--warning .toast__icon {
    color: var(--jb-warning);
  }

  .toast--info .toast__icon {
    color: var(--jb-accent);
  }

  .toast__body {
    flex: 1;
    min-width: 0;
    display: grid;
    gap: 0.15rem;
  }

  .toast__title {
    margin: 0;
    font-size: 0.8125rem;
    font-weight: 600;
    line-height: 1.3;
    color: var(--jb-text);
  }

  .toast__message {
    margin: 0;
    font-size: 0.875rem;
    line-height: 1.4;
    color: var(--jb-text);
  }

  .toast__actions {
    display: flex;
    flex-wrap: wrap;
    gap: 0.65rem;
    margin-top: 0.35rem;
  }

  .toast__action {
    padding: 0;
    border: none;
    background: none;
    color: var(--jb-accent);
    font: inherit;
    font-size: 0.8125rem;
    font-weight: 600;
    cursor: pointer;
  }

  .toast__action:hover {
    text-decoration: underline;
  }

  .toast__close {
    display: grid;
    place-content: center;
    padding: 0.125rem;
    border: none;
    border-radius: var(--jb-radius-sm);
    background: none;
    color: var(--jb-text-subtle);
    cursor: pointer;
    flex-shrink: 0;
  }

  .toast__close:hover {
    color: var(--jb-text);
  }

  .toast__close:focus-visible {
    outline: 2px solid var(--jb-focus-ring);
    outline-offset: 2px;
  }

  @media (max-width: 768px) {
    .toast-container {
      right: max(var(--jb-space-3), env(safe-area-inset-right, 0px));
      left: max(var(--jb-space-3), env(safe-area-inset-left, 0px));
      max-width: none;
    }
  }
</style>
