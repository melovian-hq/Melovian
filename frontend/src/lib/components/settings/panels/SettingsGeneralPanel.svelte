<script lang="ts">
  import SettingsCard from "$lib/components/settings/SettingsCard.svelte";
  import { APP_NAME } from "$lib/brand";
  import SettingsSaveStatus from "$lib/components/settings/SettingsSaveStatus.svelte";
  import SettingsToggleRow from "$lib/components/settings/SettingsToggleRow.svelte";
  import Field from "$lib/components/ui/Field.svelte";
  import Select from "$lib/components/ui/Select.svelte";
  import Button from "$lib/components/ui/Button.svelte";

  import ThemeCustomization from "$lib/components/settings/ThemeCustomization.svelte";
  import AccountSecurity from "$lib/components/settings/AccountSecurity.svelte";
  import SentrySettings from "$lib/components/settings/SentrySettings.svelte";
  import BackupSettings from "$lib/components/settings/BackupSettings.svelte";
  import DesktopIntegrationSettings from "$lib/components/settings/DesktopIntegrationSettings.svelte";
  import GraphicsSettings from "$lib/components/settings/GraphicsSettings.svelte";
  import Spinner from "$lib/components/ui/Spinner.svelte";
  import { fetchWithRetry, apiHeaders } from "$lib/core/http/client";
  import { ApiPaths } from "$lib/core/http/api-paths";
  import { auth } from "$lib/features/auth/store.svelte";
  import { router } from "$lib/router/router.svelte";
  import { music } from "$lib/config/music.svelte";
  import { instances } from "$lib/features/instances/store.svelte";
  import {
    loadMixDisplay,
    saveMixDisplay,
    type MixDisplayStyle,
  } from "$lib/music/mix-display";
  import { mergeMetadataEnhancementSettings } from "$lib/music/metadata-enhancement-settings";
  import { extensionFeatures } from "$lib/extensions/features.svelte";
  import { toast } from "$lib/ui/toast.svelte";
  import { createSaveStatus } from "$lib/settings/save-status.svelte";
  import "$lib/settings/settings-page.css";

  const appearanceStatus = createSaveStatus();
  const libraryStatus = createSaveStatus();
  const metadataStatus = createSaveStatus();

  let dataDir = $state("");
  let loading = $state(true);
  let mixDisplayStyle = $state<MixDisplayStyle>(loadMixDisplay());

  const mixDisplayOptions: { value: MixDisplayStyle; label: string }[] = [
    { value: "cards", label: "Cards" },
    { value: "discs", label: "Discs" },
  ];
  let metadataEnhancementSettings = $derived(
    mergeMetadataEnhancementSettings(music.metadataEnhancementSettings),
  );

  $effect(() => {
    loading = true;
    fetchWithRetry(ApiPaths.config, { headers: apiHeaders() })
      .then((r) => (r.ok ? r.json() : null))
      .then((cfg) => {
        dataDir = cfg?.dataDir ?? "";
      })
      .finally(() => {
        loading = false;
      });
  });

  function saveMetadataEnhancementSettings() {
    metadataStatus.begin();
    try {
      music.updateMetadataEnhancementSettings(metadataEnhancementSettings);
      metadataStatus.saved();
    } catch {
      metadataStatus.failed("Could not save metadata enhancement settings");
      toast.error("Could not save metadata enhancement settings");
    }
  }

  function resetMetadataEnhancementSettings() {
    metadataEnhancementSettings = mergeMetadataEnhancementSettings(null);
    saveMetadataEnhancementSettings();
  }

  function saveMixDisplayStyle(value: MixDisplayStyle) {
    appearanceStatus.begin();
    try {
      mixDisplayStyle = value;
      saveMixDisplay(value);
      appearanceStatus.saved();
    } catch {
      appearanceStatus.failed("Could not save layout settings");
      toast.error("Could not save layout settings");
    }
  }

  function saveHideUnknownMetadata(hide: boolean) {
    libraryStatus.begin();
    try {
      music.setHideUnknownMetadata(hide);
      libraryStatus.saved();
    } catch {
      libraryStatus.failed("Could not save library settings");
      toast.error("Could not save library settings");
    }
  }

  async function handleLogout() {
    await auth.logout();
    music.disconnect();
    await instances.init();
    router.navigate("/account/login", true);
  }
</script>

{#if auth.enabled && auth.user}
  <SettingsCard
    title="Account"
    description={`Signed in to this ${APP_NAME} instance.`}
  >
    <p class="settings-page__account">
      Signed in as <strong>{auth.user.username}</strong>
    </p>
    {#snippet footer()}
      <Button variant="ghost" onclick={() => void handleLogout()}>
        Sign out
      </Button>
    {/snippet}
  </SettingsCard>
  <AccountSecurity />
{/if}

<SettingsCard
  title="Appearance"
  description="Choose light, dark, or system theme for this device."
>
  {#snippet status()}
    <SettingsSaveStatus status={appearanceStatus} />
  {/snippet}
  <ThemeCustomization />
  <Field
    label="Made for you layout"
    hint="Cards show play and shuffle. Discs show circular covers with play on hover."
  >
    <Select
      value={mixDisplayStyle}
      options={mixDisplayOptions}
      onchange={saveMixDisplayStyle}
    />
  </Field>
</SettingsCard>

<SettingsCard
  title="Library"
  description="Control which albums, artists, and tracks show up in browse views."
>
  {#snippet status()}
    <SettingsSaveStatus status={libraryStatus} />
  {/snippet}
  <SettingsToggleRow
    label="Hide unknown artists and albums"
    description="Hide items tagged Unknown Artist or Unknown Album on home, albums, artists, search, favorites, genres, and history."
    checked={music.hideUnknownMetadata}
    onchange={saveHideUnknownMetadata}
  />
</SettingsCard>

{#if extensionFeatures.metadata}
  <SettingsCard
    title="Metadata enhancement"
    description="Fill in missing artist photos, album art, and track artwork from iTunes when your library does not provide them. Matches are verified by name before use."
  >
    {#snippet status()}
      <SettingsSaveStatus status={metadataStatus} />
    {/snippet}
    <SettingsToggleRow
      label="Prefer Navidrome / Subsonic artist artwork"
      description="Use artist images from your music server instead of iTunes. Turn off to allow external artist photos when the server has none."
      checked={metadataEnhancementSettings.preferServerArtistArt}
      onchange={(preferServerArtistArt) => {
        metadataEnhancementSettings = {
          ...metadataEnhancementSettings,
          preferServerArtistArt,
        };
        saveMetadataEnhancementSettings();
      }}
    />
    <SettingsToggleRow
      label="Enable metadata enhancement"
      description="Look up missing artwork from external sources."
      checked={metadataEnhancementSettings.enabled}
      onchange={(enabled) => {
        metadataEnhancementSettings = {
          ...metadataEnhancementSettings,
          enabled,
        };
        saveMetadataEnhancementSettings();
      }}
    />
    <SettingsToggleRow
      label="Artist photos"
      description="Enhance artists that have no cover art from the server."
      checked={metadataEnhancementSettings.artists}
      disabled={!metadataEnhancementSettings.enabled}
      onchange={(artists) => {
        metadataEnhancementSettings = {
          ...metadataEnhancementSettings,
          artists,
        };
        saveMetadataEnhancementSettings();
      }}
    />
    <SettingsToggleRow
      label="Album artwork"
      description="Enhance albums missing cover art."
      checked={metadataEnhancementSettings.albums}
      disabled={!metadataEnhancementSettings.enabled}
      onchange={(albums) => {
        metadataEnhancementSettings = {
          ...metadataEnhancementSettings,
          albums,
        };
        saveMetadataEnhancementSettings();
      }}
    />
    <SettingsToggleRow
      label="Track artwork"
      description="Enhance tracks missing cover art."
      checked={metadataEnhancementSettings.tracks}
      disabled={!metadataEnhancementSettings.enabled}
      onchange={(tracks) => {
        metadataEnhancementSettings = {
          ...metadataEnhancementSettings,
          tracks,
        };
        saveMetadataEnhancementSettings();
      }}
    />
    {#snippet footer()}
      <Button variant="ghost" onclick={resetMetadataEnhancementSettings}>
        Reset defaults
      </Button>
    {/snippet}
  </SettingsCard>
{/if}

<SentrySettings />

<SettingsCard
  title="Storage"
  description="Local database and preferences path on this device."
>
  {#if loading}
    <div class="settings-loading">
      <Spinner />
    </div>
  {:else}
    <code class="settings-page__path">{dataDir}</code>
  {/if}
</SettingsCard>

<BackupSettings />

<DesktopIntegrationSettings />
<GraphicsSettings />
