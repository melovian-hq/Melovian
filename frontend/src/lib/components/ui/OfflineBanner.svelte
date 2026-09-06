<script lang="ts">
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";
  import { connection } from "$lib/music/connection.svelte";
  import { music } from "$lib/config/music.svelte";
  import { sources } from "$lib/features/sources/store.svelte";

  const show = $derived(
    music.status.enabled &&
      sources.hasSubsonicActive &&
      !sources.hasUnifiedMode &&
      (!connection.online ||
        connection.phase === "reconnecting" ||
        connection.phase === "retrying"),
  );

  const message = $derived.by(() => {
    if (
      connection.phase === "reconnecting" ||
      connection.phase === "retrying"
    ) {
      return "Connection lost. Retrying in the background. Cached tracks may still play.";
    }
    return "You are offline. Cached tracks may still play.";
  });
</script>

{#if show}
  <div class="offline-banner" role="status" aria-live="polite">
    <MdiIcon name="cloudOff" size={18} />
    <span>{message}</span>
    <button
      type="button"
      class="offline-banner__retry"
      onclick={() => connection.forceReconnect()}
    >
      Retry now
    </button>
  </div>
{/if}

<style>
  .offline-banner {
    display: flex;
    align-items: center;
    gap: var(--jb-space-2);
    flex-wrap: wrap;
    padding: var(--jb-space-2) var(--jb-space-4);
    background: var(--jb-warning-muted, rgba(255, 193, 7, 0.12));
    border-bottom: 1px solid var(--jb-border);
    color: var(--jb-text);
    font-size: 0.875rem;
  }

  .offline-banner__retry {
    margin-left: auto;
    padding: 0.25rem 0.75rem;
    border: 1px solid var(--jb-border);
    border-radius: var(--jb-radius-md);
    background: var(--jb-surface);
    color: var(--jb-text);
    font-size: 0.8125rem;
    font-weight: 600;
    cursor: pointer;
  }

  .offline-banner__retry:hover {
    background: var(--jb-surface-hover);
  }
</style>
