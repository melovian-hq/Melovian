<!-- SPDX-FileCopyrightText: 2026 Quad4 Software -->
<!-- SPDX-License-Identifier: Apache-2.0 -->
<script lang="ts">
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";
  import { getCompatState, CLIENT_VERSION } from "$lib/compat";
  import { APP_NAME } from "$lib/brand";

  const STORAGE_PREFIX = "mel-compat-mismatch-dismissed:";

  const compat = $derived(getCompatState());

  function dismissKey(serverVersion: string): string {
    return `${STORAGE_PREFIX}${CLIENT_VERSION}|${serverVersion}`;
  }

  const storageKey = $derived(
    compat.versionMismatch && compat.serverVersion
      ? dismissKey(compat.serverVersion)
      : "",
  );

  let sessionDismissed = $state(false);
  let sessionKey = $state("");

  const dismissed = $derived.by(() => {
    if (!storageKey) return false;
    if (sessionDismissed && sessionKey === storageKey) return true;
    try {
      return localStorage.getItem(storageKey) === "1";
    } catch {
      return false;
    }
  });

  function dismiss() {
    if (!storageKey) return;
    sessionDismissed = true;
    sessionKey = storageKey;
    try {
      localStorage.setItem(storageKey, "1");
    } catch {
      // ignore quota / private mode
    }
  }
</script>

{#if compat.blocked}
  <div class="compat-block" role="alert">
    <h1>Version mismatch</h1>
    <p>{compat.blockReason}</p>
    <p class="compat-block__meta">
      Client {CLIENT_VERSION}
      {#if compat.serverVersion}
        · Server {compat.serverVersion}
      {/if}
    </p>
  </div>
{:else if compat.versionMismatch && !dismissed}
  <div class="compat-banner" role="status">
    <p class="compat-banner__text">
      Client {CLIENT_VERSION} and server {compat.serverVersion} differ. If you just
      updated {APP_NAME}, hard-refresh once (Ctrl+Shift+R).
    </p>
    <button
      type="button"
      class="compat-banner__dismiss"
      aria-label="Dismiss version notice"
      onclick={dismiss}
    >
      <MdiIcon name="x" size={16} />
    </button>
  </div>
{/if}

<style>
  .compat-block {
    min-height: 100vh;
    display: grid;
    place-content: center;
    text-align: center;
    gap: var(--jb-space-3);
    padding: var(--jb-space-6);
    background: var(--jb-bg);
    color: var(--jb-text);
  }

  .compat-block__meta {
    color: var(--jb-text-muted);
    font-size: 0.9rem;
  }

  .compat-banner {
    position: sticky;
    top: 0;
    z-index: 40;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 0.75rem;
    padding: 0.5rem 1rem;
    text-align: center;
    font-size: 0.85rem;
    background: color-mix(in srgb, var(--jb-accent) 18%, var(--jb-bg));
    color: var(--jb-text);
    border-bottom: 1px solid var(--jb-border, transparent);
  }

  .compat-banner__text {
    margin: 0;
  }

  .compat-banner__dismiss {
    display: grid;
    place-content: center;
    flex-shrink: 0;
    width: 1.75rem;
    height: 1.75rem;
    border: none;
    border-radius: var(--jb-radius-md);
    background: transparent;
    color: var(--jb-text-muted);
    cursor: pointer;
  }

  .compat-banner__dismiss:hover {
    background: var(--jb-surface-hover);
    color: var(--jb-text);
  }
</style>
