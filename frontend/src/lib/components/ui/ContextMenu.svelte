<script lang="ts">
  import { ContextMenu } from "bits-ui";
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";
  import {
    getPlayerBarInsetPx,
    isContextMenuItem,
    type ContextMenuEntry,
    type ContextMenuItem,
  } from "./context-menu";

  interface Props {
    x: number;
    y: number;
    onclose: () => void;
    label?: string;
    items?: ContextMenuEntry[];
  }

  let { x, y, onclose, label = "Actions", items = [] }: Props = $props();

  let open = $state(true);

  // A fresh object per position change so Bits repositions the floating layer
  const anchor = $derived.by(() => {
    const ax = x;
    const ay = y;
    return {
      getBoundingClientRect: (): DOMRect => ({
        x: ax,
        y: ay,
        top: ay,
        right: ax,
        bottom: ay,
        left: ax,
        width: 0,
        height: 0,
        toJSON: () => ({}),
      }),
    };
  });

  const collisionPadding = {
    top: 8,
    right: 8,
    bottom: 8 + getPlayerBarInsetPx(),
    left: 8,
  };

  function handleOpenChange(next: boolean) {
    if (!next) onclose();
  }

  function runItem(item: ContextMenuItem) {
    item.onclick();
  }

  // Reproduce the old backdrop behavior. Any contextmenu outside the menu
  // closes it, even when a consumer stops propagation, so listen on capture.
  $effect(() => {
    function onContextMenu(event: MouseEvent) {
      const target = event.target;
      if (
        target instanceof Element &&
        target.closest(
          "[data-context-menu-content], [data-context-menu-sub-content]",
        )
      ) {
        return;
      }
      event.preventDefault();
      onclose();
    }
    document.addEventListener("contextmenu", onContextMenu, true);
    return () =>
      document.removeEventListener("contextmenu", onContextMenu, true);
  });
</script>

<ContextMenu.Root bind:open onOpenChange={handleOpenChange}>
  <ContextMenu.Portal>
    <ContextMenu.Content
      class="context-menu"
      aria-label={label}
      customAnchor={anchor}
      {collisionPadding}
      side="right"
      align="start"
      sideOffset={0}
      sticky="always"
      preventScroll={false}
    >
      {#snippet child({ props })}
        <div {...props}>
          {#each items as entry (entry.id)}
            {#if isContextMenuItem(entry)}
              <ContextMenu.Item
                class={[
                  "context-menu__item",
                  entry.danger && "context-menu__item--danger",
                ]}
                disabled={entry.disabled}
                textValue={entry.label}
                closeOnSelect={!entry.keepOpen}
                onSelect={() => runItem(entry)}
              >
                {#snippet child({ props: itemProps })}
                  <button
                    type="button"
                    {...itemProps}
                    disabled={entry.disabled}
                  >
                    {#if entry.icon}
                      <MdiIcon name={entry.icon} size={16} />
                    {/if}
                    {entry.label}
                  </button>
                {/snippet}
              </ContextMenu.Item>
            {:else}
              <ContextMenu.Separator class="context-menu__sep">
                {#snippet child({ props: sepProps })}
                  <div {...sepProps}></div>
                {/snippet}
              </ContextMenu.Separator>
            {/if}
          {/each}
        </div>
      {/snippet}
    </ContextMenu.Content>
  </ContextMenu.Portal>
</ContextMenu.Root>

<style>
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
