<!-- SPDX-FileCopyrightText: 2026 Quad4 Software -->
<!-- SPDX-License-Identifier: Apache-2.0 -->
<script lang="ts">
  import VirtualList from "$lib/components/ui/VirtualList.svelte";
  import PlaylistTile from "./PlaylistTile.svelte";
  import type {
    PlaylistItem,
    PlaylistKind,
    PlaylistViewMode,
  } from "$lib/music/playlist-display";

  const VIRTUAL_THRESHOLD = 24;
  const PLAYLIST_ROW_HEIGHT = 84;

  interface Props {
    items: PlaylistItem[];
    kind: PlaylistKind;
    view: PlaylistViewMode;
    onexport: (playlist: PlaylistItem) => void;
    ondelete: (playlist: PlaylistItem) => void;
    onshare: (playlist: PlaylistItem) => void;
  }

  let { items, kind, view, onexport, ondelete, onshare }: Props = $props();

  const useVirtual = $derived(
    view === "list" && items.length >= VIRTUAL_THRESHOLD,
  );
</script>

{#if useVirtual}
  <VirtualList
    {items}
    itemHeight={PLAYLIST_ROW_HEIGHT}
    scrollMode="document"
    class="playlists-collection playlists-collection--list"
  >
    {#snippet children({ item: pl })}
      <PlaylistTile
        playlist={pl}
        {kind}
        view="list"
        onExport={() => onexport(pl)}
        onDelete={() => ondelete(pl)}
        onShare={() => onshare(pl)}
      />
    {/snippet}
  </VirtualList>
{:else}
  <ul
    class="playlists-collection"
    class:playlists-collection--list={view === "list"}
    class:playlists-collection--grid={view === "grid"}
    class:playlists-collection--card={view === "card"}
  >
    {#each items as pl (pl.id)}
      <li class="playlists-collection__item">
        <PlaylistTile
          playlist={pl}
          {kind}
          {view}
          onExport={() => onexport(pl)}
          onDelete={() => ondelete(pl)}
          onShare={() => onshare(pl)}
        />
      </li>
    {/each}
  </ul>
{/if}

<style>
  .playlists-collection {
    list-style: none;
    margin: 0;
    padding: 0;
  }

  .playlists-collection--list {
    display: grid;
    gap: var(--jb-space-2);
  }

  .playlists-collection--grid,
  .playlists-collection--card {
    display: grid;
    grid-template-columns: repeat(
      auto-fill,
      minmax(min(100%, var(--jb-card-grid-min)), 1fr)
    );
    gap: var(--jb-space-5);
  }

  .playlists-collection__item {
    min-width: 0;
  }

  :global(.playlists-collection--list.virtual-list--document) {
    display: block;
  }

  :global(.playlists-collection--list .virtual-list__window) {
    display: grid;
    gap: var(--jb-space-2);
  }

  @media (max-width: 640px) {
    .playlists-collection--grid {
      grid-template-columns: repeat(2, minmax(0, 1fr));
      gap: var(--jb-space-3);
    }

    .playlists-collection--card {
      grid-template-columns: repeat(2, minmax(0, 1fr));
      gap: var(--jb-space-3);
    }
  }
</style>
