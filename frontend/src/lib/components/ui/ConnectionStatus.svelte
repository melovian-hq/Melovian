<script lang="ts">
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";
  import Spinner from "$lib/components/ui/Spinner.svelte";
  import { connection } from "$lib/music/connection.svelte";
  import { music } from "$lib/config/music.svelte";
  import { sources } from "$lib/features/sources/store.svelte";

  interface Props {
    class?: string;
    embedded?: boolean;
  }

  let { class: className = "", embedded = false }: Props = $props();

  let now = $state(Date.now());

  $effect(() => {
    if (!connection.nextRetryAt) return;
    const timer = setInterval(() => {
      now = Date.now();
    }, 500);
    return () => clearInterval(timer);
  });

  const show = $derived(
    music.status.enabled &&
      (!connection.online ||
        connection.phase === "reconnecting" ||
        connection.phase === "retrying" ||
        connection.phase === "self-healing"),
  );

  const icon = $derived.by(() => {
    switch (connection.phase) {
      case "offline":
        return "cloudOff";
      case "reconnecting":
      case "retrying":
        return "refresh";
      case "self-healing":
        return "heartPulse";
      default:
        return "cloudCheck";
    }
  });

  const retryIn = $derived.by(() => {
    if (!connection.nextRetryAt) return null;
    const remaining = Math.max(0, connection.nextRetryAt - now);
    if (remaining < 1000) return "< 1s";
    return `${Math.ceil(remaining / 1000)}s`;
  });

  async function refreshLibrary() {
    if (music.libraryRefreshing || !music.connected) return;
    await music.refreshLibrary();
  }
</script>

{#if music.status.enabled}
  {#if show}
    <button
      type="button"
      class="conn-status conn-status--{connection.phase} {className}"
      class:conn-status--embedded={embedded}
      onclick={() => connection.forceReconnect()}
      title="{connection.statusLabel}{retryIn ? ` · retry in ${retryIn}` : ''}"
      aria-label="{connection.statusLabel}. Click to retry now."
    >
      <MdiIcon
        name={icon}
        size={18}
        class={connection.phase === "reconnecting" ||
        connection.phase === "retrying" ||
        connection.phase === "self-healing"
          ? "conn-status__icon conn-status__icon--spin"
          : "conn-status__icon"}
      />
      <span class="conn-status__label">{connection.statusLabel}</span>
      {#if retryIn && connection.phase !== "self-healing"}
        <span class="conn-status__retry">{retryIn}</span>
      {/if}
    </button>
  {:else if connection.online}
    <button
      type="button"
      class="conn-status conn-status--online {className}"
      class:conn-status--embedded={embedded}
      class:conn-status--refreshing={music.libraryRefreshing}
      title="Connected to {music.serverName}. Click to refresh library."
      aria-label="Connected. Click to refresh library."
      disabled={music.libraryRefreshing || !sources.hasSubsonicActive}
      onclick={() => void refreshLibrary()}
    >
      <span class="conn-status__cloud-wrap">
        <MdiIcon name="cloudCheck" size={18} class="conn-status__cloud" />
        {#if music.libraryRefreshing}
          <span class="conn-status__overlay" aria-hidden="true">
            <Spinner />
          </span>
        {/if}
      </span>
    </button>
  {/if}
{/if}

<style>
  .conn-status {
    display: inline-flex;
    align-items: center;
    gap: 0.375rem;
    padding: 0.375rem 0.625rem;
    border: 1px solid var(--jb-island-border);
    border-radius: var(--jb-radius-full);
    background: var(--jb-island-bg);
    color: var(--jb-text-muted);
    font-size: 0.75rem;
    font-weight: 600;
    backdrop-filter: blur(16px);
    box-shadow: var(--jb-shadow-sm);
    cursor: default;
    max-width: 14rem;
  }

  button.conn-status {
    cursor: pointer;
  }

  button.conn-status:hover {
    border-color: var(--jb-accent);
    color: var(--jb-text);
  }

  button.conn-status:disabled {
    cursor: default;
    opacity: 0.7;
  }

  .conn-status--online {
    padding: 0.4375rem;
    color: var(--jb-success, #16a34a);
    border-color: color-mix(
      in srgb,
      var(--jb-success, #16a34a) 35%,
      var(--jb-border)
    );
    cursor: pointer;
  }

  button.conn-status--online:hover:not(:disabled) {
    border-color: color-mix(
      in srgb,
      var(--jb-success, #16a34a) 55%,
      var(--jb-accent)
    );
  }

  .conn-status__cloud-wrap {
    position: relative;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 1.125rem;
    height: 1.125rem;
  }

  .conn-status--refreshing :global(.conn-status__cloud) {
    filter: blur(2px);
    opacity: 0.4;
  }

  .conn-status__overlay {
    position: absolute;
    inset: -5px;
    display: grid;
    place-content: center;
    pointer-events: none;
  }

  .conn-status__overlay :global(.spinner__ring) {
    width: 1rem;
    height: 1rem;
    border-width: 2px;
  }

  .conn-status--offline {
    color: var(--jb-danger);
    border-color: color-mix(in srgb, var(--jb-danger) 35%, var(--jb-border));
  }

  .conn-status--reconnecting,
  .conn-status--retrying,
  .conn-status--self-healing {
    color: var(--jb-accent);
    border-color: color-mix(in srgb, var(--jb-accent) 35%, var(--jb-border));
  }

  :global(.conn-status__icon--spin) {
    animation: conn-spin 1.2s linear infinite;
  }

  .conn-status__label {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .conn-status__retry {
    opacity: 0.75;
    font-variant-numeric: tabular-nums;
  }

  @keyframes conn-spin {
    to {
      transform: rotate(360deg);
    }
  }

  .conn-status--embedded {
    background: var(--jb-bg-muted);
    border-color: var(--jb-border);
    box-shadow: none;
    backdrop-filter: none;
  }

  @media (max-width: 640px) {
    .conn-status__label,
    .conn-status__retry {
      display: none;
    }

    .conn-status--online:not(.conn-status--embedded) {
      display: none;
    }
  }
</style>
