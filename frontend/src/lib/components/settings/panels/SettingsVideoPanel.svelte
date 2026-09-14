<script lang="ts">
  import SettingsCard from "$lib/components/settings/SettingsCard.svelte";
  import SettingsSaveStatus from "$lib/components/settings/SettingsSaveStatus.svelte";
  import SettingsToggleRow from "$lib/components/settings/SettingsToggleRow.svelte";
  import Field from "$lib/components/ui/Field.svelte";
  import Select from "$lib/components/ui/Select.svelte";
  import Spinner from "$lib/components/ui/Spinner.svelte";
  import { useDebounce } from "runed";
  import { getVideoSettings, saveVideoSettings } from "$lib/video/api";
  import { videoFeature } from "$lib/video/feature.svelte";
  import {
    defaultVideoSettings,
    mergeVideoSettings,
    type VideoSearchProvider,
    type VideoSettings,
  } from "$lib/video/ids";
  import { toast } from "$lib/ui/toast.svelte";
  import {
    createSaveStatus,
    saveWithStatus,
  } from "$lib/settings/save-status.svelte";
  import "$lib/settings/settings-page.css";

  const videoStatus = createSaveStatus();

  let settings = $state<VideoSettings>(defaultVideoSettings());
  let loading = $state(true);

  const searchProviderOptions: {
    value: VideoSearchProvider;
    label: string;
  }[] = [
    { value: "invidious", label: "Invidious (primary)" },
    { value: "youtube", label: "YouTube Data API" },
  ];

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

  async function saveSettings() {
    await saveWithStatus(
      videoStatus,
      (err) =>
        err instanceof Error ? err.message : "Could not save video settings",
      async () => {
        settings = mergeVideoSettings(
          await saveVideoSettings({
            enabled: settings.enabled,
            searchProvider: settings.searchProvider,
            invidiousBaseUrl: settings.invidiousBaseUrl.trim(),
            youtubeApiKey: settings.youtubeApiKey.trim(),
          }),
        );
        videoFeature.applySettings(settings);
      },
      (message) => toast.error(message),
    );
  }

  const debouncedSave = useDebounce(() => saveSettings(), 500);

  function queueSave() {
    void debouncedSave().catch(() => {});
  }
</script>

<SettingsCard
  title="Videos"
  description="Local files and music-video search are off until you turn this on. Invidious is the default search path. YouTube Data API is optional."
>
  {#snippet status()}
    <SettingsSaveStatus status={videoStatus} />
  {/snippet}
  {#if loading}
    <div class="settings-loading">
      <Spinner />
    </div>
  {:else}
    <SettingsToggleRow
      label="Enable videos"
      description="Show the Videos page, local video playback, and music-video search on Now Playing."
      checked={settings.enabled}
      onchange={(checked) => {
        settings = { ...settings, enabled: checked };
        void saveSettings();
      }}
    />

    <Field label="Search provider">
      <Select
        value={settings.searchProvider}
        options={searchProviderOptions}
        disabled={!settings.enabled}
        onchange={(searchProvider) => {
          settings = { ...settings, searchProvider };
          void saveSettings();
        }}
      />
    </Field>

    <Field
      label="Invidious instance URL"
      hint="Used for Invidious search and embeds. Example: https://invidious.example.com"
    >
      <input
        type="url"
        class="settings-input"
        bind:value={settings.invidiousBaseUrl}
        oninput={queueSave}
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
        oninput={queueSave}
        placeholder="AIza..."
        autocomplete="off"
        spellcheck="false"
        disabled={!settings.enabled}
      />
    </Field>
  {/if}
</SettingsCard>
