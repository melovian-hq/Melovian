<script lang="ts">
  import SettingsCard from "$lib/components/settings/SettingsCard.svelte";
  import { APP_NAME } from "$lib/brand";
  import SettingsSaveStatus from "$lib/components/settings/SettingsSaveStatus.svelte";
  import SettingsToggleRow from "$lib/components/settings/SettingsToggleRow.svelte";
  import Field from "$lib/components/ui/Field.svelte";
  import Button from "$lib/components/ui/Button.svelte";
  import { useDebounce } from "runed";
  import { music } from "$lib/config/music.svelte";
  import {
    defaultLyricsSettings,
    formatLyricsBytes,
    mergeLyricsSettings,
    type LyricsProviderSetting,
    type LyricsSettings,
  } from "$lib/music/lyrics-settings";
  import * as musicApi from "$lib/music/api";
  import { extensionFeatures } from "$lib/extensions/features.svelte";
  import { toast } from "$lib/ui/toast.svelte";
  import { confirmDialog } from "$lib/ui/confirm.svelte";
  import {
    createSaveStatus,
    saveWithStatus,
  } from "$lib/settings/save-status.svelte";
  import "$lib/settings/settings-page.css";

  const lyricsStatus = createSaveStatus();

  let lyricsSettings = $state<LyricsSettings>(defaultLyricsSettings());
  let lyricsResolvedDir = $state("");
  let lyricsDefaultDir = $state("");
  let lyricsTrackCount = $state(0);
  let lyricsUsedBytes = $state(0);
  let lyricsClearing = $state(false);
  let customProviderName = $state("");
  let customProviderURL = $state("");

  $effect(() => {
    if (music.lyricsSettings) {
      lyricsSettings = mergeLyricsSettings(music.lyricsSettings);
      lyricsResolvedDir = music.lyricsSettings.resolvedStorageDir;
      lyricsDefaultDir = music.lyricsSettings.defaultStorageDir;
      lyricsTrackCount = music.lyricsSettings.trackCount;
      lyricsUsedBytes = music.lyricsSettings.usedBytes;
    }
  });

  $effect(() => {
    void music.loadLyricsSettings();
  });

  function toggleLyricsProvider(id: string, checked: boolean) {
    lyricsSettings = {
      ...lyricsSettings,
      providers: lyricsSettings.providers.map((provider) =>
        provider.id === id ? { ...provider, enabled: checked } : provider,
      ),
    };
    void applyLyricsSettings();
  }

  function addCustomLyricsProvider() {
    const name = customProviderName.trim();
    const url = customProviderURL.trim();
    if (!name || !url) {
      toast.error("Custom provider needs a name and URL template");
      return;
    }
    const id = `custom-${Date.now()}`;
    lyricsSettings = {
      ...lyricsSettings,
      providers: [
        ...lyricsSettings.providers,
        { id, name, enabled: true, custom: true, url },
      ],
    };
    customProviderName = "";
    customProviderURL = "";
    void applyLyricsSettings();
  }

  function removeCustomLyricsProvider(id: string) {
    lyricsSettings = {
      ...lyricsSettings,
      providers: lyricsSettings.providers.filter(
        (provider) => provider.id !== id,
      ),
    };
    void applyLyricsSettings();
  }

  async function applyLyricsSettings() {
    await saveWithStatus(
      lyricsStatus,
      (err) =>
        err instanceof Error ? err.message : "Could not save lyrics settings",
      async () => {
        await music.updateLyricsSettings(lyricsSettings);
        if (music.lyricsSettings) {
          lyricsResolvedDir = music.lyricsSettings.resolvedStorageDir;
          lyricsDefaultDir = music.lyricsSettings.defaultStorageDir;
          lyricsTrackCount = music.lyricsSettings.trackCount;
          lyricsUsedBytes = music.lyricsSettings.usedBytes;
        }
      },
      (message) => toast.error(message),
    );
  }

  const debouncedLyricsSave = useDebounce(() => applyLyricsSettings(), 500);

  function queueLyricsSave() {
    void debouncedLyricsSave().catch(() => {});
  }

  async function clearLyricsCache() {
    const ok = await confirmDialog.confirm({
      title: "Clear lyrics cache",
      message:
        "Remove all cached lyrics files? Providers will refetch as needed.",
      confirmLabel: "Clear cache",
      danger: true,
    });
    if (!ok) return;
    lyricsClearing = true;
    try {
      await musicApi.clearLyricsCache();
      await music.loadLyricsSettings();
      if (music.lyricsSettings) {
        lyricsTrackCount = music.lyricsSettings.trackCount;
        lyricsUsedBytes = music.lyricsSettings.usedBytes;
      }
      toast.success("Lyrics cache cleared");
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to clear lyrics cache",
      );
    } finally {
      lyricsClearing = false;
    }
  }

  function lyricsProviderLabel(provider: LyricsProviderSetting): string {
    if (provider.name) return provider.name;
    if (provider.id === "subsonic") return "Navidrome / Subsonic";
    if (provider.id === "lrclib") return "LRCLIB";
    if (provider.id === "lyrics-ovh") return "Lyrics.ovh";
    return provider.id;
  }
</script>

<SettingsCard
  title="Lyrics providers"
  description="Fetch lyrics from your server and free online providers. Synced lines can be clicked to seek during playback."
>
  {#snippet status()}
    <SettingsSaveStatus status={lyricsStatus} />
  {/snippet}
  <SettingsToggleRow
    label="Fetch lyrics automatically when missing"
    checked={lyricsSettings.autoFetch}
    onchange={(checked) => {
      lyricsSettings = { ...lyricsSettings, autoFetch: checked };
      void applyLyricsSettings();
    }}
  />

  <Field
    label="Lyrics storage location"
    hint={`Relative paths are stored under the ${APP_NAME} data directory. Leave empty for the default lyrics folder.`}
  >
    <input
      type="text"
      class="settings-input"
      bind:value={lyricsSettings.storageDir}
      oninput={queueLyricsSave}
      placeholder={lyricsDefaultDir || "lyrics"}
    />
  </Field>

  {#if lyricsResolvedDir}
    <p class="settings-page__meta">Resolved path: {lyricsResolvedDir}</p>
  {/if}

  <p class="settings-page__meta">
    Cached lyrics: {lyricsTrackCount} tracks · {formatLyricsBytes(
      lyricsUsedBytes,
    )}
  </p>

  {#each lyricsSettings.providers as provider (provider.id)}
    <div class="settings-provider-row">
      <SettingsToggleRow
        label={lyricsProviderLabel(provider)}
        checked={provider.enabled}
        onchange={(checked) => toggleLyricsProvider(provider.id, checked)}
      />
      {#if provider.custom}
        <Button
          variant="ghost"
          size="sm"
          onclick={() => removeCustomLyricsProvider(provider.id)}
        >
          Remove
        </Button>
      {/if}
    </div>
    {#if provider.custom && provider.url}
      <p class="settings-page__meta">{provider.url}</p>
    {/if}
  {/each}

  <Field
    label="Custom provider name"
    hint="Shown in settings and used as the provider label."
  >
    <input
      type="text"
      class="settings-input"
      bind:value={customProviderName}
      placeholder="My lyrics API"
    />
  </Field>

  <Field
    label="Custom provider URL"
    hint={"Use {artist}, {title}, {album}, {trackId}, and {duration} placeholders."}
  >
    <input
      type="text"
      class="settings-input"
      bind:value={customProviderURL}
      placeholder={"https://example.com/lyrics?artist={artist}&title={title}"}
    />
  </Field>

  <Button variant="ghost" onclick={addCustomLyricsProvider}>
    Add custom provider
  </Button>

  {#if extensionFeatures.lyricsWhisper}
    <Field
      label="Whisper server URL"
      hint="Base URL of a whisper.cpp-compatible server or transcriptasm host (for example http://127.0.0.1:8080). Used to generate synced lyrics from track audio. Requires the Lyrics Whisper extension."
    >
      <input
        type="text"
        class="settings-input"
        bind:value={lyricsSettings.whisperUrl}
        oninput={queueLyricsSave}
        placeholder="http://127.0.0.1:8080"
      />
    </Field>
    <p class="settings-page__meta">
      Generate synced lyrics from the lyrics panel while a track is playing.
    </p>
  {/if}

  {#snippet footer()}
    <Button
      variant="ghost"
      disabled={lyricsClearing}
      onclick={() => void clearLyricsCache()}
    >
      {lyricsClearing ? "Clearing…" : "Clear lyrics cache"}
    </Button>
  {/snippet}
</SettingsCard>
