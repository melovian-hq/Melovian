<script lang="ts">
  import SettingsToggleRow from "$lib/components/settings/SettingsToggleRow.svelte";
  import { APP_NAME } from "$lib/brand";
  import { sources } from "$lib/features/sources/store.svelte";
  import { localLibraries } from "$lib/features/local-libraries/store.svelte";
  import {
    getMediaBaseUrl,
    isServerMode,
    loadRuntimeConfig,
    normalizePublicBaseUrl,
  } from "$lib/config/runtime";
  import { fetchWithRetry, apiHeaders } from "$lib/core/http/client";

  let subsonicEnabled = $state(false);
  let transcodingAvailable = $state(false);
  let dlnaEnabled = $state(false);
  let dlnaPort = $state(8200);
  let jukeboxEnabled = $state(false);
  let extensionsDir = $state("");
  let savingMulti = $state(false);

  const publicBase = $derived.by(() => {
    const raw =
      getMediaBaseUrl() ||
      (typeof window !== "undefined" ? window.location.origin : "");
    return normalizePublicBaseUrl(raw);
  });

  const restUrl = $derived(`${publicBase}/rest`);

  const shareBaseUrl = $derived(`${publicBase}/s`);

  $effect(() => {
    void (async () => {
      await loadRuntimeConfig();
      try {
        const response = await fetchWithRetry("/api/config", {
          headers: apiHeaders(),
        });
        if (!response.ok) return;
        const cfg = (await response.json()) as {
          subsonicServer?: { enabled?: boolean };
          transcoding?: { available?: boolean };
          extensions?: { dir?: string };
          dlna?: { enabled?: boolean; port?: number };
          jukebox?: { enabled?: boolean };
        };
        subsonicEnabled = cfg.subsonicServer?.enabled === true;
        transcodingAvailable = cfg.transcoding?.available === true;
        extensionsDir = cfg.extensions?.dir ?? "";
        dlnaEnabled = cfg.dlna?.enabled === true;
        dlnaPort = cfg.dlna?.port ?? 8200;
        jukeboxEnabled = cfg.jukebox?.enabled === true;
      } catch {
        // ignore
      }
    })();
  });

  async function toggleMultiLibrary(enabled: boolean) {
    savingMulti = true;
    try {
      await sources.setMultiLocalLibrary(enabled);
    } finally {
      savingMulti = false;
    }
  }
</script>

{#if localLibraries.enabled}
  <SettingsToggleRow
    label="Merge all local libraries"
    description="Browse every saved local folder in one catalog. Streaming and covers resolve tracks across libraries."
    checked={sources.multiLocalLibrary}
    disabled={savingMulti || localLibraries.items.length < 2}
    onchange={(enabled) => void toggleMultiLibrary(enabled)}
  />
{/if}

{#if subsonicEnabled}
  <div class="music-server-settings__block">
    <p class="music-server-settings__label">Subsonic-compatible API</p>
    <p class="music-server-settings__hint">
      Point other Subsonic clients at this URL. Use your {APP_NAME} account when sign-in
      is enabled, or connect without credentials on trusted desktop installs.
    </p>
    <code class="music-server-settings__url">{restUrl}</code>
    {#if isServerMode()}
      <p class="music-server-settings__meta">
        Server mode exposes the REST API on the same host as the web UI.
      </p>
    {/if}
    {#if transcodingAvailable}
      <p class="music-server-settings__meta">
        ffmpeg is available. Clients can request transcoding with maxBitRate on
        stream requests.
      </p>
    {:else}
      <p class="music-server-settings__meta">
        Install ffmpeg on the host to enable on-the-fly MP3 transcoding for
        remote clients.
      </p>
    {/if}
  </div>
{/if}

{#if subsonicEnabled}
  <div class="music-server-settings__block">
    <p class="music-server-settings__label">Public share links</p>
    <p class="music-server-settings__hint">
      Create shares from the API or Subsonic clients. Public listeners use URLs
      like:
    </p>
    <code class="music-server-settings__url">{shareBaseUrl}/&lt;token&gt;</code>
  </div>
{/if}

{#if jukeboxEnabled}
  <div class="music-server-settings__block">
    <p class="music-server-settings__label">Jukebox mode</p>
    <p class="music-server-settings__hint">
      Control playback on this host from Subsonic jukebox endpoints or
      <code>/api/jukebox/control</code>. Status is at
      <code>/api/jukebox/status</code>.
    </p>
  </div>
{/if}

{#if dlnaEnabled}
  <div class="music-server-settings__block">
    <p class="music-server-settings__label">DLNA media server</p>
    <p class="music-server-settings__hint">
      {APP_NAME} advertises your local library on the LAN for UPnP/DLNA renderers.
      Set <code>MELOVIAN_DLNA_SERVER=true</code> and optionally
      <code>MELOVIAN_DLNA_PORT</code> (default {dlnaPort}).
    </p>
  </div>
{/if}

{#if extensionsDir}
  <p class="music-server-settings__meta">
    Extensions folder: <code>{extensionsDir}</code>
  </p>
{/if}

<style>
  .music-server-settings__block {
    display: grid;
    gap: 0.5rem;
    margin-top: 0.75rem;
  }

  .music-server-settings__label {
    margin: 0;
    font-weight: 600;
  }

  .music-server-settings__hint,
  .music-server-settings__meta {
    margin: 0;
    color: var(--jb-text-muted);
    font-size: 0.9rem;
    line-height: 1.45;
  }

  .music-server-settings__url {
    display: block;
    padding: 0.65rem 0.75rem;
    border-radius: 0.5rem;
    background: var(--jb-surface-2);
    font-size: 0.85rem;
    overflow-x: auto;
  }
</style>
