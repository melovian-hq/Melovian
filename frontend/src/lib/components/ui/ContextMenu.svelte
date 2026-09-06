<script lang="ts">
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";
  import {
    clampContextMenuPosition,
    isContextMenuItem,
    type ContextMenuEntry,
  } from "./context-menu";

  interface Props {
    x: number;
    y: number;
    onclose: () => void;
    label?: string;
    items?: ContextMenuEntry[];
  }

  let { x, y, onclose, label = "Actions", items = [] }: Props = $props();

  let menuEl = $state<HTMLDivElement | undefined>();
  let didFocus = false;

  function attachMenu(node: HTMLDivElement) {
    menuEl = node;
    const next = clampContextMenuPosition(
      x,
      y,
      node.offsetWidth,
      node.offsetHeight,
    );
    node.style.left = `${next.x}px`;
    node.style.top = `${next.y}px`;
    if (!didFocus) {
      didFocus = true;
      node
        .querySelector<HTMLButtonElement>("[role='menuitem']:not(:disabled)")
        ?.focus();
    }
    return () => {
      if (menuEl === node) menuEl = undefined;
    };
  }

  function onKeydown(event: KeyboardEvent) {
    if (event.key === "Escape") {
      event.preventDefault();
      onclose();
      return;
    }
    if (!menuEl) return;
    const buttons = [
      ...menuEl.querySelectorAll<HTMLButtonElement>(
        "[role='menuitem']:not(:disabled)",
      ),
    ];
    if (buttons.length === 0) return;
    const current = buttons.indexOf(
      document.activeElement as HTMLButtonElement,
    );
    if (event.key === "ArrowDown") {
      event.preventDefault();
      buttons[(current + 1) % buttons.length]?.focus();
    } else if (event.key === "ArrowUp") {
      event.preventDefault();
      buttons[(current - 1 + buttons.length) % buttons.length]?.focus();
    } else if (event.key === "Home") {
      event.preventDefault();
      buttons[0]?.focus();
    } else if (event.key === "End") {
      event.preventDefault();
      buttons[buttons.length - 1]?.focus();
    }
  }

  function runItem(item: ContextMenuEntry) {
    if (!isContextMenuItem(item) || item.disabled) return;
    item.onclick();
    if (!item.keepOpen) onclose();
  }
</script>

<svelte:window onkeydown={onKeydown} />

<div
  class="context-menu-backdrop"
  role="presentation"
  onclick={onclose}
  oncontextmenu={(event) => {
    event.preventDefault();
    onclose();
  }}
></div>
<div
  class="context-menu"
  style:left="{x}px"
  style:top="{y}px"
  role="menu"
  aria-label={label}
  {@attach attachMenu}
>
  {#each items as entry (entry.id)}
    {#if isContextMenuItem(entry)}
      <button
        type="button"
        class={[
          "context-menu__item",
          entry.danger && "context-menu__item--danger",
        ]}
        role="menuitem"
        disabled={entry.disabled}
        onclick={() => runItem(entry)}
      >
        {#if entry.icon}
          <MdiIcon name={entry.icon} size={16} />
        {/if}
        {entry.label}
      </button>
    {:else}
      <div class="context-menu__sep" role="separator"></div>
    {/if}
  {/each}
</div>

<style>
  .context-menu-backdrop {
    position: fixed;
    inset: 0;
    z-index: 90;
  }

  .context-menu {
    position: fixed;
    z-index: 91;
    min-width: 12.5rem;
    padding: var(--jb-space-1);
    border: 1px solid var(--jb-border);
    border-radius: var(--jb-radius-md);
    background: var(--jb-bg-elevated);
    box-shadow: var(--jb-shadow-lg);
    display: flex;
    flex-direction: column;
    gap: 0.125rem;
  }

  .context-menu__item {
    display: flex;
    align-items: center;
    gap: var(--jb-space-2);
    width: 100%;
    padding: 0.55rem 0.75rem;
    border: none;
    border-radius: var(--jb-radius-sm);
    background: transparent;
    color: var(--jb-text);
    font: inherit;
    font-size: 0.875rem;
    font-weight: 600;
    text-align: left;
    cursor: pointer;
  }

  .context-menu__item:hover:not(:disabled) {
    background: var(--jb-surface-hover);
  }

  .context-menu__item:focus-visible {
    outline: 2px solid var(--jb-accent);
    outline-offset: -2px;
  }

  .context-menu__item:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .context-menu__item--danger {
    color: var(--jb-danger);
  }

  .context-menu__sep {
    height: 1px;
    margin: 0.25rem 0.5rem;
    background: var(--jb-border);
  }
</style>
