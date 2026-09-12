<script lang="ts">
  import { fade, fly } from "svelte/transition";
  import SettingsNav from "$lib/components/settings/SettingsNav.svelte";
  import SettingsProfilePanel from "$lib/components/settings/panels/SettingsProfilePanel.svelte";
  import SettingsGeneralPanel from "$lib/components/settings/panels/SettingsGeneralPanel.svelte";
  import SettingsServersPanel from "$lib/components/settings/panels/SettingsServersPanel.svelte";
  import SettingsPlaybackPanel from "$lib/components/settings/panels/SettingsPlaybackPanel.svelte";
  import SettingsDownloadsPanel from "$lib/components/settings/panels/SettingsDownloadsPanel.svelte";
  import SettingsMixesPanel from "$lib/components/settings/panels/SettingsMixesPanel.svelte";
  import SettingsConnectionPanel from "$lib/components/settings/panels/SettingsConnectionPanel.svelte";
  import SettingsLyricsPanel from "$lib/components/settings/panels/SettingsLyricsPanel.svelte";
  import SettingsVideoPanel from "$lib/components/settings/panels/SettingsVideoPanel.svelte";
  import SettingsRockskyPanel from "$lib/components/settings/panels/SettingsRockskyPanel.svelte";
  import SettingsListenBrainzPanel from "$lib/components/settings/panels/SettingsListenBrainzPanel.svelte";
  import SettingsLastFMPanel from "$lib/components/settings/panels/SettingsLastFMPanel.svelte";
  import SettingsExtensionsPanel from "$lib/components/settings/panels/SettingsExtensionsPanel.svelte";
  import SettingsAboutPanel from "$lib/components/settings/panels/SettingsAboutPanel.svelte";
  import SettingsTasksPanel from "$lib/components/settings/panels/SettingsTasksPanel.svelte";
  import LocalSearchBox from "$lib/components/ui/LocalSearchBox.svelte";
  import EmptyState from "$lib/components/ui/EmptyState.svelte";
  import {
    pinOutgoingPage,
    tabInFly,
    tabOutFade,
  } from "$lib/router/page-motion";
  import { router } from "$lib/router/router.svelte";
  import {
    visibleSettingsTabs,
    filterSettingsTabs,
    isSettingsTabId,
    settingsTabFromPath,
    settingsTabPath,
    type SettingsTabId,
  } from "$lib/settings/tabs";
  import { extensionFeatures } from "$lib/extensions/features.svelte";
  import { supports } from "$lib/compat";
  import "$lib/settings/settings-page.css";

  interface Props {
    tab?: string;
  }

  let { tab: routeTab = "" }: Props = $props();

  let searchQuery = $state("");

  const tabs = $derived(
    visibleSettingsTabs({
      lyrics: extensionFeatures.lyrics,
      rocksky: extensionFeatures.isEnabled("rocksky"),
      listenbrainz: extensionFeatures.isEnabled("listenbrainz"),
      lastfm: extensionFeatures.isEnabled("lastfm"),
      supports,
    }),
  );

  const filteredTabs = $derived(filterSettingsTabs(tabs, searchQuery));

  const tab = $derived.by((): SettingsTabId => {
    const fromPath = settingsTabFromPath(router.pathname);
    const candidate = fromPath || routeTab || "profile";
    return isSettingsTabId(candidate) ? candidate : "profile";
  });

  function goTab(id: SettingsTabId) {
    router.navigate(settingsTabPath(id));
  }

  $effect(() => {
    const fromPath = settingsTabFromPath(router.pathname);
    if (
      router.pathname === "/settings" ||
      (fromPath.length > 0 && !isSettingsTabId(fromPath))
    ) {
      router.navigate(settingsTabPath("profile"), true);
      return;
    }
    if (tab === "lyrics" && !extensionFeatures.lyrics) {
      router.navigate(settingsTabPath("profile"), true);
    }
  });

  $effect(() => {
    if (!searchQuery.trim()) return;
    if (filteredTabs.length === 0) return;
    if (filteredTabs.some((t) => t.id === tab)) return;
    goTab(filteredTabs[0].id);
  });

  function onTabOutroStart(event: Event) {
    const node = event.currentTarget;
    if (node instanceof HTMLElement) {
      pinOutgoingPage(node);
    }
  }
</script>

<div class="settings-page">
  <header class="settings-page__header">
    <div class="settings-page__heading">
      <h1>Settings</h1>
      <p class="settings-page__subtitle">
        Sources, playback, and personalization.
      </p>
    </div>
    <LocalSearchBox
      bind:value={searchQuery}
      placeholder="Search settings"
      resultCount={filteredTabs.length}
      totalCount={tabs.length}
      class="settings-page__search"
    />
  </header>

  {#if filteredTabs.length === 0}
    <EmptyState
      title="No matching settings"
      message={`No settings match "${searchQuery.trim()}".`}
      icon="search"
    />
  {:else}
    <div class="settings-layout">
      <SettingsNav tabs={filteredTabs} active={tab} onselect={goTab} />

      <div class="settings-layout__content">
        {#key tab}
          <div
            class="settings-layout__pane"
            in:fly={tabInFly()}
            out:fade={tabOutFade()}
            onoutrostart={onTabOutroStart}
          >
            {#if tab === "profile"}
              <SettingsProfilePanel />
            {:else if tab === "general"}
              <SettingsGeneralPanel />
            {:else if tab === "servers"}
              <SettingsServersPanel />
            {:else if tab === "playback"}
              <SettingsPlaybackPanel />
            {:else if tab === "downloads"}
              <SettingsDownloadsPanel />
            {:else if tab === "mixes"}
              <SettingsMixesPanel />
            {:else if tab === "connection"}
              <SettingsConnectionPanel />
            {:else if tab === "lyrics" && extensionFeatures.lyrics}
              <SettingsLyricsPanel />
            {:else if tab === "video"}
              <SettingsVideoPanel />
            {:else if tab === "rocksky" && extensionFeatures.isEnabled("rocksky")}
              <SettingsRockskyPanel />
            {:else if tab === "listenbrainz" && extensionFeatures.isEnabled("listenbrainz")}
              <SettingsListenBrainzPanel />
            {:else if tab === "lastfm" && extensionFeatures.isEnabled("lastfm")}
              <SettingsLastFMPanel />
            {:else if tab === "extensions"}
              <SettingsExtensionsPanel />
            {:else if tab === "about"}
              <SettingsAboutPanel />
            {:else if tab === "tasks"}
              <SettingsTasksPanel />
            {/if}
          </div>
        {/key}
      </div>
    </div>
  {/if}
</div>

<style>
  .settings-page {
    display: flex;
    flex-direction: column;
    gap: var(--jb-space-4);
    max-width: var(--jb-content-max);
    margin: 0 auto;
    width: 100%;
  }

  .settings-layout {
    display: flex;
    flex-direction: column;
    gap: 0;
    width: 100%;
  }

  .settings-layout__content {
    position: relative;
    min-width: 0;
    display: flex;
    flex-direction: column;
    padding: var(--jb-space-5) 0;
    overflow: hidden;
  }

  .settings-layout__pane {
    width: 100%;
    display: flex;
    flex-direction: column;
  }

  .settings-layout__content :global(.settings-card) {
    border: none;
    border-radius: 0;
    box-shadow: none;
    background: transparent;
    padding: var(--jb-space-5) 0 0;
    margin: 0;
  }

  .settings-layout__content :global(.settings-card:first-child) {
    padding-top: 0;
  }

  .settings-layout__content :global(.settings-card + .settings-card) {
    border-top: 1px solid var(--jb-border);
    margin-top: var(--jb-space-5);
  }

  @media (min-width: 1024px) {
    .settings-layout {
      flex-direction: row;
      align-items: flex-start;
      gap: var(--jb-space-8);
    }

    .settings-layout__content {
      flex: 1;
      padding: 0;
    }
  }

  @media (min-width: 1400px) {
    .settings-layout__pane {
      display: grid;
      grid-template-columns: repeat(2, minmax(0, 1fr));
      gap: var(--jb-space-5);
      align-items: start;
    }

    .settings-layout__content :global(.settings-card) {
      border: 1px solid var(--jb-border);
      border-radius: var(--jb-radius-xl);
      background: var(--jb-surface);
      padding: var(--jb-space-5);
      box-shadow: var(--jb-shadow-sm);
      margin: 0;
    }

    .settings-layout__content :global(.settings-card + .settings-card) {
      border-top: 1px solid var(--jb-border);
      margin-top: 0;
    }
  }

  @media (min-width: 1600px) {
    .settings-page {
      max-width: none;
    }
  }

  @media (max-width: 768px) {
    .settings-page {
      gap: var(--jb-space-3);
      max-width: none;
    }

    .settings-layout__content {
      padding: var(--jb-space-3) 0 var(--jb-space-4);
    }

    .settings-layout__content :global(.settings-card) {
      padding-top: var(--jb-space-4);
    }

    .settings-layout__content :global(.settings-card + .settings-card) {
      margin-top: var(--jb-space-4);
    }
  }
</style>
