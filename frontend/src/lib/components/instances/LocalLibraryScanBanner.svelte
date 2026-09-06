<script lang="ts">
  import Spinner from "$lib/components/ui/Spinner.svelte";
  import { localLibraries } from "$lib/features/local-libraries/store.svelte";

  const scanning = $derived(localLibraries.scanningItems);
</script>

{#if scanning.length > 0}
  <div class="scan-banner" role="status" aria-live="polite">
    {#each scanning as item (item.id)}
      <div class="scan-banner__item">
        <Spinner class="scan-banner__spinner" />
        <span class="scan-banner__text">
          Scanning <strong>{item.name}</strong>
          {#if item.scanProgress}
            · {item.scanProgress.phase === "reconciling"
              ? "Reconciling"
              : `${item.scanProgress.processed.toLocaleString()} files`}
          {:else}
            · starting
          {/if}
        </span>
      </div>
    {/each}
  </div>
{/if}

<style>
  .scan-banner {
    position: fixed;
    top: calc(var(--jb-topbar-height) + var(--jb-space-2));
    left: 50%;
    transform: translateX(-50%);
    z-index: 70;
    display: flex;
    flex-direction: column;
    gap: var(--jb-space-2);
    max-width: min(28rem, calc(100vw - 2rem));
    pointer-events: none;
  }

  .scan-banner__item {
    display: flex;
    align-items: center;
    gap: var(--jb-space-3);
    padding: var(--jb-space-2) var(--jb-space-4);
    border-radius: var(--jb-radius-full);
    background: color-mix(in srgb, var(--jb-bg-elevated) 94%, transparent);
    border: 1px solid var(--jb-border);
    box-shadow: var(--jb-shadow-md);
    backdrop-filter: blur(16px);
    font-size: 0.8125rem;
    color: var(--jb-text-muted);
  }

  .scan-banner__item :global(.scan-banner__spinner .spinner__ring) {
    width: 0.875rem;
    height: 0.875rem;
    border-width: 2px;
  }

  .scan-banner__text strong {
    color: var(--jb-text);
    font-weight: 650;
  }

  @media (max-width: 768px) {
    .scan-banner {
      top: calc(var(--jb-topbar-height) + env(safe-area-inset-top, 0px));
      left: var(--jb-space-3);
      right: var(--jb-space-3);
      transform: none;
      max-width: none;
    }
  }
</style>
