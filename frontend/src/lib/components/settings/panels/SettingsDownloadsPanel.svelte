<script lang="ts">
  import SettingsCard from "$lib/components/settings/SettingsCard.svelte";
  import SettingsToggleRow from "$lib/components/settings/SettingsToggleRow.svelte";
  import Field from "$lib/components/ui/Field.svelte";
  import Button from "$lib/components/ui/Button.svelte";
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";
  import { apiHeaders } from "$lib/core/http/client";
  import { DOWNLOADS_ZIP_BASENAME } from "$lib/brand";
  import { music } from "$lib/config/music.svelte";
  import {
    formatBytes,
    mergeCacheSettings,
    CACHE_STRATEGY_LABELS,
    type CacheSettings,
    type CacheStrategy,
  } from "$lib/music/cache-settings";
  import * as musicApi from "$lib/music/api";
  import { downloadFromUrl } from "$lib/utils/download";
  import { toast } from "$lib/ui/toast.svelte";
  import { confirmDialog } from "$lib/ui/confirm.svelte";
  import "$lib/settings/settings-page.css";

  let cacheSettings = $state<CacheSettings>(
    mergeCacheSettings(music.cacheSettings),
  );
  let cacheUsedBytes = $state(music.cacheUsedBytes);
  let cacheTrackCount = $state(music.cacheTrackCount);
  let cacheSaving = $state(false);
  let cacheClearing = $state(false);
  let cacheExporting = $state(false);
  let cacheRevealing = $state(false);
  let downloadDir = $state<string | null>(null);

  const cacheStrategies = Object.entries(CACHE_STRATEGY_LABELS) as [
    CacheStrategy,
    string,
  ][];

  $effect(() => {
    cacheSettings = mergeCacheSettings(music.cacheSettings);
    cacheUsedBytes = music.cacheUsedBytes;
    cacheTrackCount = music.cacheTrackCount;
    void musicApi.getDownloadDir().then((path) => {
      downloadDir = path;
    });
  });

  function setCacheStrategy(value: string) {
    if (!(value in CACHE_STRATEGY_LABELS)) return;
    cacheSettings = {
      ...cacheSettings,
      strategy: value as CacheStrategy,
    };
  }

  function cacheLimitGb(): number {
    return cacheSettings.limitBytes / (1024 * 1024 * 1024);
  }

  function setCacheLimitGb(raw: string) {
    const value = Number.parseFloat(raw);
    if (!Number.isFinite(value) || value <= 0) return;
    cacheSettings = {
      ...cacheSettings,
      limitBytes: Math.round(value * 1024 * 1024 * 1024),
    };
  }

  async function applyCacheSettings() {
    cacheSaving = true;
    try {
      await music.updateCacheSettings(cacheSettings);
      toast.success("Download settings saved");
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to save cache settings",
      );
    } finally {
      cacheSaving = false;
    }
  }

  async function clearCache() {
    const ok = await confirmDialog.confirm({
      title: "Delete all downloads",
      message:
        "Delete every cached download on this device? This cannot be undone.",
      confirmLabel: "Delete all",
      danger: true,
    });
    if (!ok) return;
    cacheClearing = true;
    try {
      await music.clearDownloadCache();
      toast.success("All downloads deleted");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to clear cache");
    } finally {
      cacheClearing = false;
    }
  }

  async function openDownloadFolder() {
    cacheRevealing = true;
    try {
      await musicApi.revealDownloadDir();
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to open downloads folder",
      );
    } finally {
      cacheRevealing = false;
    }
  }

  async function exportCachedTracks() {
    cacheExporting = true;
    try {
      await downloadFromUrl(
        musicApi.downloadExportAllUrl(),
        `${DOWNLOADS_ZIP_BASENAME}.zip`,
        { headers: apiHeaders() },
      );
      toast.success("Downloads exported");
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to export downloads",
      );
    } finally {
      cacheExporting = false;
    }
  }
</script>

<SettingsCard
  title="Downloads and cache"
  description="Save tracks on this device for smoother playback when the connection drops."
>
  <SettingsToggleRow
    label="Download and cache songs"
    description="Keep downloaded tracks available for offline playback."
    checked={cacheSettings.enabled}
    onchange={(checked) => {
      cacheSettings = { ...cacheSettings, enabled: checked };
    }}
  />

  <Field
    label="Caching strategy"
    hint="Controls which tracks are downloaded in the background."
  >
    <select
      class="settings-input"
      value={cacheSettings.strategy}
      onchange={(e) => setCacheStrategy(e.currentTarget.value)}
    >
      {#each cacheStrategies as [value, label] (value)}
        <option {value}>{label}</option>
      {/each}
    </select>
  </Field>

  <Field
    label="Download limit (GB)"
    hint="Oldest cached tracks are removed automatically when the limit is reached."
  >
    <input
      type="number"
      class="settings-input"
      min="0.5"
      step="0.5"
      value={cacheLimitGb()}
      oninput={(e) => setCacheLimitGb(e.currentTarget.value)}
    />
  </Field>

  <p class="settings-page__meta">
    {formatBytes(cacheUsedBytes)} used of {formatBytes(
      cacheSettings.limitBytes,
    )}
    · {cacheTrackCount} tracks downloaded
  </p>

  {#if downloadDir}
    <p class="settings-page__meta">
      <code class="settings-page__path">{downloadDir}</code>
    </p>
  {/if}

  {#snippet footer()}
    <Button disabled={cacheSaving} onclick={() => void applyCacheSettings()}>
      Save download settings
    </Button>
    <Button
      variant="ghost"
      disabled={cacheRevealing}
      onclick={() => void openDownloadFolder()}
    >
      <MdiIcon name="folderOpen" size={16} />
      Open folder
    </Button>
    <Button
      variant="ghost"
      disabled={cacheExporting || cacheTrackCount === 0}
      onclick={() => void exportCachedTracks()}
    >
      <MdiIcon name="export" size={16} />
      {cacheExporting ? "Exporting…" : "Export downloads"}
    </Button>
    <Button
      variant="ghost"
      disabled={cacheClearing || cacheTrackCount === 0}
      onclick={() => void clearCache()}
    >
      Delete all downloads
    </Button>
  {/snippet}
</SettingsCard>
