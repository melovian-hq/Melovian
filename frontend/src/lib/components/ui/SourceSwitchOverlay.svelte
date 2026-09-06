<script lang="ts">
  import { fade } from "svelte/transition";
  import Spinner from "$lib/components/ui/Spinner.svelte";
  import { instances } from "$lib/features/instances/store.svelte";
  import { localLibraries } from "$lib/features/local-libraries/store.svelte";
  import { sources } from "$lib/features/sources/store.svelte";
  import { music } from "$lib/config/music.svelte";
  import { overlayFade } from "$lib/router/page-motion";

  const switching = $derived(
    instances.switching ||
      localLibraries.switching ||
      (music.loading && !music.libraryReady && sources.hasAnySource),
  );

  const label = $derived.by(() => {
    if (instances.switching || localLibraries.switching) {
      return `Switching to ${sources.activeLabel}…`;
    }
    return "Loading library…";
  });
</script>

{#if switching}
  <div
    class="source-switch"
    role="status"
    aria-live="polite"
    aria-busy="true"
    transition:fade={overlayFade()}
  >
    <div class="source-switch__card">
      <Spinner />
      <p>{label}</p>
    </div>
  </div>
{/if}

<style>
  .source-switch {
    position: fixed;
    inset: 0;
    z-index: 90;
    display: grid;
    place-content: center;
    background: rgba(0, 0, 0, 0.35);
    pointer-events: all;
  }

  .source-switch__card {
    display: flex;
    align-items: center;
    gap: var(--jb-space-3);
    padding: var(--jb-space-4) var(--jb-space-5);
    border-radius: var(--jb-radius-lg);
    border: 1px solid var(--jb-border);
    background: var(--jb-surface);
    box-shadow: var(--jb-shadow-lg);
    color: var(--jb-text);
    font-weight: 600;
  }

  .source-switch__card p {
    margin: 0;
  }
</style>
