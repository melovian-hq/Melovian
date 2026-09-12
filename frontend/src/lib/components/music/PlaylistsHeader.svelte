<!-- SPDX-FileCopyrightText: 2026 Quad4 Software -->
<!-- SPDX-License-Identifier: Apache-2.0 -->
<script lang="ts">
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";
  import Button from "$lib/components/ui/Button.svelte";
  import { APP_NAME } from "$lib/brand";

  interface Props {
    showKindToggle: boolean;
    canUseServer: boolean;
    importing: boolean;
    importInput?: HTMLInputElement;
    onImportSelected: (event: Event) => void;
  }

  let {
    showKindToggle,
    canUseServer,
    importing,
    importInput = $bindable(),
    onImportSelected,
  }: Props = $props();
</script>

<header class="playlists-header">
  <div>
    <p class="playlists-header__eyebrow">
      <MdiIcon name="listMusic" size={18} />
      Your library
    </p>
    <h1>Playlists</h1>
    <p class="playlists-header__sub">
      {#if showKindToggle}
        Switch between server and local playlists. Imports match tracks from
        your active library.
      {:else if canUseServer}
        Server playlists sync from your Subsonic or Navidrome server.
      {:else}
        Local playlists are stored in {APP_NAME} on this device.
      {/if}
    </p>
  </div>
  <div class="playlists-header__tools">
    <input
      bind:this={importInput}
      type="file"
      accept=".m3u,.m3u8,audio/x-mpegurl"
      class="playlists-header__import-input"
      onchange={onImportSelected}
    />
    <Button
      variant="surface"
      disabled={importing}
      onclick={() => importInput?.click()}
    >
      <MdiIcon name="upload" size={16} />
      Import M3U
    </Button>
  </div>
</header>

<style>
  .playlists-header {
    display: flex;
    flex-wrap: wrap;
    align-items: flex-start;
    justify-content: space-between;
    gap: var(--jb-space-4);
    margin-bottom: var(--jb-space-6);
  }

  .playlists-header__tools {
    display: flex;
    align-items: center;
    gap: var(--jb-space-2);
  }

  .playlists-header__import-input {
    display: none;
  }

  .playlists-header__eyebrow {
    display: inline-flex;
    align-items: center;
    gap: var(--jb-space-2);
    margin: 0 0 var(--jb-space-2);
    font-size: 0.8125rem;
    font-weight: 600;
    color: var(--jb-music-accent);
    text-transform: uppercase;
    letter-spacing: 0.06em;
  }

  .playlists-header h1 {
    margin: 0 0 var(--jb-space-2);
    font-size: 2rem;
    font-weight: 800;
  }

  .playlists-header__sub {
    margin: 0;
    color: var(--jb-text-muted);
    max-width: 34rem;
  }
</style>
