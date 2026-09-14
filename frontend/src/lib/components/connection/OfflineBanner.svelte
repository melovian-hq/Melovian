<script lang="ts">
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";
  import { connection } from "$lib/music/connection.svelte";
  import { music } from "$lib/config/music.svelte";
  import { instances } from "$lib/features/instances/store.svelte";
  import { isDemoMode, isStaticDemo } from "$lib/config/runtime";

  function hostLabel(serverUrl: string): string {
    if (!serverUrl) return "";
    try {
      return new URL(serverUrl).host;
    } catch {
      return serverUrl.replace(/^https?:\/\//, "").replace(/\/+$/, "");
    }
  }

  const serverLabel = $derived.by(() => {
    const active = instances.active;
    const name = active?.serverName || active?.name || music.serverName;
    const host = hostLabel(active?.serverUrl ?? "");
    return host ? `${name} at ${host}` : name;
  });

  const retrying = $derived(
    connection.phase === "reconnecting" || connection.phase === "retrying",
  );

  const show = $derived(
    !isStaticDemo() &&
      !isDemoMode() &&
      music.status.enabled &&
      connection.managed &&
      !connection.online,
  );

  const message = $derived.by(() => {
    const base = retrying
      ? `Lost connection to ${serverLabel}. Retrying automatically.`
      : `Lost connection to ${serverLabel}.`;
    if (music.resumePendingOnReconnect) {
      return `${base} Playback is paused and will resume on reconnect.`;
    }
    return base;
  });

  // If the banner unmounts while it holds focus (Retry succeeded, reconnect
  // landed), move focus to a stable target instead of dropping it to body.
  let focusInside = false;
  $effect(() => {
    if (!show && focusInside) {
      focusInside = false;
      document.querySelector<HTMLElement>(".topbar__menu-btn")?.focus();
    }
  });
</script>

{#if show}
  <div
    class="offline-banner"
    role="status"
    aria-live="polite"
    onfocusin={() => (focusInside = true)}
    onfocusout={(e) => {
      if (!e.currentTarget.contains(e.relatedTarget as Node | null)) {
        focusInside = false;
      }
    }}
  >
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
    background: var(--jb-warning-muted);
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
