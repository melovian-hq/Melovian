<script lang="ts">
  import SettingsCard from "$lib/components/settings/SettingsCard.svelte";
  import { APP_NAME } from "$lib/brand";
  import SettingsToggleRow from "$lib/components/settings/SettingsToggleRow.svelte";
  import Field from "$lib/components/ui/Field.svelte";
  import Button from "$lib/components/ui/Button.svelte";
  import ThemeToggle from "$lib/components/ui/ThemeToggle.svelte";
  import ThemeCustomization from "$lib/components/settings/ThemeCustomization.svelte";
  import AccountSecurity from "$lib/components/settings/AccountSecurity.svelte";
  import SentrySettings from "$lib/components/settings/SentrySettings.svelte";
  import BackupSettings from "$lib/components/settings/BackupSettings.svelte";
  import DesktopIntegrationSettings from "$lib/components/settings/DesktopIntegrationSettings.svelte";
  import GraphicsSettings from "$lib/components/settings/GraphicsSettings.svelte";
  import Spinner from "$lib/components/ui/Spinner.svelte";
  import { fetchWithRetry, apiHeaders } from "$lib/core/http/client";
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
  import "$lib/settings/settings-page.css";

  let dataDir = $state("");
  let loading = $state(true);
  let mixDisplayStyle = $state<MixDisplayStyle>(loadMixDisplay());
  let metadataEnhancementSettings = $derived(
    mergeMetadataEnhancementSettings(music.metadataEnhancementSettings),
  );

  $effect(() => {
    loading = true;
    fetchWithRetry("/api/config", { headers: apiHeaders() })
      .then((r) => (r.ok ? r.json() : null))
      .then((cfg) => {
        dataDir = cfg?.dataDir ?? "";
      })
      .finally(() => {
        loading = false;
      });
  });

  function saveMetadataEnhancementSettings() {
    music.updateMetadataEnhancementSettings(metadataEnhancementSettings);
    toast.success("Metadata enhancement settings saved");
  }

  function resetMetadataEnhancementSettings() {
    metadataEnhancementSettings = mergeMetadataEnhancementSettings(null);
    music.updateMetadataEnhancementSettings(metadataEnhancementSettings);
    toast.success("Metadata enhancement settings reset");
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
    <Button variant="ghost" onclick={() => void handleLogout()}>Sign out</Button
    >
  </SettingsCard>
  <AccountSecurity />
{/if}

<SettingsCard
  title="Appearance"
  description="Choose light, dark, or system theme for this device."
>
  <ThemeToggle embedded />
  <ThemeCustomization />
  <Field
    label="Made for you layout"
    hint="Cards show play and shuffle. Discs show circular covers with play on hover."
  >
    <select
      class="settings-input"
      value={mixDisplayStyle}
      onchange={(e) => {
        const value = (e.currentTarget as HTMLSelectElement)
          .value as MixDisplayStyle;
        mixDisplayStyle = value;
        saveMixDisplay(value);
      }}
    >
      <option value="cards">Cards</option>
      <option value="discs">Discs</option>
    </select>
  </Field>
</SettingsCard>

<SettingsCard
  title="Library"
  description="Control which albums, artists, and tracks show up in browse views."
>
  <SettingsToggleRow
    label="Hide unknown artists and albums"
    description="Hide items tagged Unknown Artist or Unknown Album on home, albums, artists, search, favorites, genres, and history."
    checked={music.hideUnknownMetadata}
    onchange={(hide) => music.setHideUnknownMetadata(hide)}
  />
</SettingsCard>

{#if extensionFeatures.metadata}
  <SettingsCard
    title="Metadata enhancement"
    description="Fill in missing artist photos, album art, and track artwork from iTunes when your library does not provide them. Matches are verified by name before use."
  >
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
    <Button variant="ghost" onclick={resetMetadataEnhancementSettings}>
      Reset defaults
    </Button>
  </SettingsCard>
{/if}

<SentrySettings />

<SettingsCard
  title="Storage"
  description="Local database and preferences path on this device."
>
  {#if loading}
    <div class="settings-page__loading">
      <Spinner />
    </div>
  {:else}
    <code class="settings-page__path">{dataDir}</code>
  {/if}
</SettingsCard>

<BackupSettings />

<DesktopIntegrationSettings />
<GraphicsSettings />

<style>
  .settings-page__loading {
    display: grid;
    place-content: center;
    min-height: 3rem;
  }
</style>
