<script lang="ts">
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";
  import SourceIcon from "$lib/components/ui/SourceIcon.svelte";
  import type { SourceKind } from "$lib/icons/server-kind";

  interface Props {
    title: string;
    message?: string;
    icon?: string;
    sourceKind?: SourceKind;
    class?: string;
    embedded?: boolean;
    actions?: import("svelte").Snippet;
  }

  let {
    title,
    message = "",
    icon = "music",
    sourceKind,
    class: className = "",
    embedded = false,
    actions,
  }: Props = $props();
</script>

<div class="empty-state {className}" class:empty-state--embedded={embedded}>
  <div class="empty-state__icon" aria-hidden="true">
    {#if sourceKind}
      <SourceIcon kind={sourceKind} size={28} />
    {:else}
      <MdiIcon name={icon} size={28} />
    {/if}
  </div>
  <h2 class="empty-state__title">{title}</h2>
  {#if message}
    <p class="empty-state__message">{message}</p>
  {/if}
  {#if actions}
    <div class="empty-state__actions">
      {@render actions?.()}
    </div>
  {/if}
</div>

<style>
  .empty-state {
    box-sizing: border-box;
    display: grid;
    justify-items: center;
    text-align: center;
    gap: var(--jb-space-3);
    width: 100%;
    max-width: 100%;
    min-width: 0;
    padding: var(--jb-space-10) var(--jb-space-4);
    border: 1px dashed var(--jb-border);
    border-radius: var(--jb-radius-lg);
    background: var(--jb-bg-subtle);
    overflow-wrap: anywhere;
  }

  .empty-state__icon {
    display: grid;
    place-content: center;
    width: 3rem;
    height: 3rem;
    border-radius: var(--jb-radius-full);
    background: var(--jb-accent-muted);
    color: var(--jb-accent);
  }

  .empty-state__title {
    margin: 0;
    max-width: 100%;
    font-size: 1.125rem;
    overflow-wrap: anywhere;
  }

  .empty-state__message {
    margin: 0;
    width: 100%;
    max-width: min(28rem, 100%);
    min-width: 0;
    color: var(--jb-text-muted);
    line-height: 1.6;
    overflow-wrap: anywhere;
  }

  .empty-state__actions {
    display: flex;
    flex-wrap: wrap;
    justify-content: center;
    align-items: center;
    gap: var(--jb-space-2);
    width: 100%;
    max-width: 100%;
    min-width: 0;
    margin-top: var(--jb-space-2);
  }

  .empty-state__actions :global(a),
  .empty-state__actions :global(.btn) {
    max-width: 100%;
  }

  .empty-state--embedded {
    padding: var(--jb-space-6) var(--jb-space-3);
    border: none;
    border-radius: var(--jb-radius-md);
    background: var(--jb-bg-muted);
  }

  @media (max-width: 768px) {
    .empty-state {
      gap: var(--jb-space-2);
      padding: var(--jb-space-6) var(--jb-space-3);
    }

    .empty-state--embedded {
      padding: var(--jb-space-4) var(--jb-space-2);
    }

    .empty-state__title {
      font-size: 1.0625rem;
    }

    .empty-state__actions {
      flex-direction: column;
      align-items: stretch;
    }

    .empty-state__actions :global(a),
    .empty-state__actions :global(.btn) {
      width: 100%;
      justify-content: center;
      text-align: center;
    }
  }

  @media (max-width: 480px) {
    .empty-state {
      padding: var(--jb-space-5) var(--jb-space-2);
    }
  }
</style>
