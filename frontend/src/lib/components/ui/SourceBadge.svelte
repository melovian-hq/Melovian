<script lang="ts">
  import { sources } from "$lib/features/sources/store.svelte";
  import { isLocalMusicId } from "$lib/music/library-adapter";

  interface Props {
    id: string;
    class?: string;
  }

  let { id, class: className = "" }: Props = $props();

  const label = $derived(isLocalMusicId(id) ? "Local" : "Server");
  const show = $derived(sources.hasUnifiedMode);
</script>

{#if show}
  <span class="source-badge source-badge--{label.toLowerCase()} {className}">
    {label}
  </span>
{/if}

<style>
  .source-badge {
    display: inline-flex;
    align-items: center;
    padding: 0.125rem 0.45rem;
    border-radius: var(--jb-radius-full);
    font-size: 0.625rem;
    font-weight: 700;
    letter-spacing: 0.04em;
    text-transform: uppercase;
    line-height: 1.2;
  }

  .source-badge--local {
    background: var(--jb-accent-muted);
    color: var(--jb-accent);
  }

  .source-badge--server {
    background: var(--jb-bg-muted);
    color: var(--jb-text-muted);
    border: 1px solid var(--jb-border);
  }
</style>
