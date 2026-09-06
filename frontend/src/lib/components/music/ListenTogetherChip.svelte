<script lang="ts">
  import { deviceSync } from "$lib/music/device-sync.svelte";

  interface Props {
    compact?: boolean;
  }

  let { compact = false }: Props = $props();

  const active = $derived(deviceSync.inListenTogether);
  const label = $derived(
    deviceSync.crossUser
      ? deviceSync.sessionLabel || "Party"
      : deviceSync.sessionLabel || "Listening together",
  );
  const count = $derived(deviceSync.sessionMemberCount);
</script>

{#if active}
  <button
    type="button"
    class={["together-chip", compact && "together-chip--compact"]}
    onclick={() => deviceSync.togglePanel()}
    title={label}
    aria-label={label}
  >
    {#if compact}
      <span class="together-chip__count">{count}</span>
    {:else}
      <span class="together-chip__text">{label}</span>
    {/if}
  </button>
{/if}

<style>
  .together-chip {
    display: inline-flex;
    align-items: center;
    gap: 0.35rem;
    width: fit-content;
    max-width: 100%;
    border: 1px solid color-mix(in srgb, var(--jb-accent) 45%, var(--jb-border));
    border-radius: var(--jb-radius-full);
    background: color-mix(in srgb, var(--jb-accent-muted) 70%, transparent);
    color: var(--jb-accent);
    font-size: 0.6875rem;
    font-weight: 650;
    padding: 0.2rem 0.55rem;
    cursor: pointer;
    line-height: 1.2;
  }

  .together-chip:hover {
    border-color: color-mix(in srgb, var(--jb-accent) 70%, var(--jb-border));
    background: color-mix(in srgb, var(--jb-accent-muted) 90%, transparent);
  }

  .together-chip--compact {
    padding: 0.15rem 0.4rem;
    gap: 0.25rem;
  }

  .together-chip__text {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .together-chip__count {
    font-variant-numeric: tabular-nums;
  }
</style>
