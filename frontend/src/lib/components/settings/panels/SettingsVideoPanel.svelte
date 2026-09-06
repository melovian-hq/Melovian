<script lang="ts">
  import SettingsCard from "$lib/components/settings/SettingsCard.svelte";
  import SettingsToggleRow from "$lib/components/settings/SettingsToggleRow.svelte";
  import Field from "$lib/components/ui/Field.svelte";
  import Button from "$lib/components/ui/Button.svelte";
  import Spinner from "$lib/components/ui/Spinner.svelte";
  import { getVideoSettings, saveVideoSettings } from "$lib/video/api";
  import { videoFeature } from "$lib/video/feature.svelte";
  import {
    defaultVideoSettings,
    mergeVideoSettings,
    type VideoSettings,
  } from "$lib/video/ids";
  import { toast } from "$lib/ui/toast.svelte";
  import "$lib/settings/settings-page.css";

  let settings = $state<VideoSettings>(defaultVideoSettings());
  let loading = $state(true);
  let saving = $state(false);

  $effect(() => {
    let cancelled = false;
    loading = true;
    void getVideoSettings()
      .then((value) => {
        if (!cancelled) settings = mergeVideoSettings(value);
      })
      .catch((err: unknown) => {
        if (!cancelled) {
          toast.error(
            err instanceof Error
              ? err.message
              : "Could not load video settings",
          );
        }
      })
      .finally(() => {
        if (!cancelled) loading = false;
      });
    return () => {
      cancelled = true;
    };
  });

  async function onSave() {
    saving = true;
    try {
      settings = mergeVideoSettings(
        await saveVideoSettings({
          enabled: settings.enabled,
          searchProvider: settings.searchProvider,
          invidiousBaseUrl: settings.invidiousBaseUrl.trim(),
          youtubeApiKey: settings.youtubeApiKey.trim(),
        }),
      );
      videoFeature.applySettings(settings);
      toast.success("Video settings saved");
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Could not save video settings",
      );
    } finally {
      saving = false;
    }
  }
</script>

<SettingsCard
  title="Videos"
  description="Local files and music-video search are off until you turn this on. Invidious is the default search path. YouTube Data API is optional."
>
  {#if loading}
    <div class="settings-panel__loading">
      <Spinner />
    </div>
  {:else}
    <SettingsToggleRow
      label="Enable videos"
      description="Show the Videos page, local video playback, and music-video search on Now Playing."
      checked={settings.enabled}
      onchange={(checked) => {
        settings = { ...settings, enabled: checked };
      }}
    />

    <Field label="Search provider">
      <select
        class="settings-input"
        bind:value={settings.searchProvider}
        disabled={!settings.enabled}
      >
        <option value="invidious">Invidious (primary)</option>
        <option value="youtube">YouTube Data API</option>
      </select>
    </Field>

    <Field
      label="Invidious instance URL"
      hint="Used for Invidious search and embeds. Example: https://invidious.example.com"
    >
      <input
        type="url"
        class="settings-input"
        bind:value={settings.invidiousBaseUrl}
        placeholder="https://"
        autocomplete="off"
        spellcheck="false"
        disabled={!settings.enabled}
      />
    </Field>

    <Field
      label="YouTube Data API key"
      hint="Needed only when the search provider is YouTube. Embeds use the official YouTube player."
    >
      <input
        type="password"
        class="settings-input"
        bind:value={settings.youtubeApiKey}
        placeholder="AIza…"
        autocomplete="off"
        spellcheck="false"
        disabled={!settings.enabled}
      />
    </Field>

    <div class="settings-actions">
      <Button onclick={onSave} disabled={saving}>
        {saving ? "Saving…" : "Save"}
      </Button>
    </div>
  {/if}
</SettingsCard>

<style>
  .settings-panel__loading {
    display: grid;
    place-content: center;
    min-height: 4rem;
  }

  .settings-actions {
    margin-top: var(--jb-space-3);
  }
</style>
