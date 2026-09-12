<script lang="ts">
  import MusicBreadcrumbs from "$lib/components/music/MusicBreadcrumbs.svelte";
  import CollectionHero from "$lib/components/music/CollectionHero.svelte";
  import TrackVirtualList from "$lib/components/music/TrackVirtualList.svelte";
  import TrackSelectionBar from "$lib/components/music/TrackSelectionBar.svelte";
  import AlbumGrid from "$lib/components/music/AlbumGrid.svelte";
  import ArtistGrid from "$lib/components/music/ArtistGrid.svelte";
  import Skeleton from "$lib/components/ui/Skeleton.svelte";
  import EmptyState from "$lib/components/ui/EmptyState.svelte";
  import LocalSearchBox from "$lib/components/ui/LocalSearchBox.svelte";
  import Link from "$lib/router/Link.svelte";
  import ContextMenu from "$lib/components/ui/ContextMenu.svelte";
  import {
    contextMenuPositionFromEvent,
    type ContextMenuEntry,
  } from "$lib/components/ui/context-menu";
  import { music } from "$lib/config/music.svelte";
  import { trackSelection } from "$lib/music/selection.svelte";
  import { filterByLocalSearch } from "$lib/utils/local-search";
  import { formatHumanDuration } from "$lib/music/playlist-duration";
  import { coverArtUrl } from "$lib/subsonic";
  import LibraryUnavailable from "$lib/components/music/LibraryUnavailable.svelte";
  import { libraryUnavailable } from "$lib/music/library-gate";
  import { Tabs } from "bits-ui";

  type FavoritesTab = "tracks" | "albums" | "artists";

  let loading = $state(true);
  let searchQuery = $state("");
  let activeTab = $state<FavoritesTab>("tracks");
  let pageMenu = $state<{ x: number; y: number } | null>(null);

  const unavailable = $derived(libraryUnavailable());

  const favoriteSongs = $derived(
    music.favoriteTracks.map((entry) => music.favoriteToSong(entry)),
  );

  const filteredFavoriteSongs = $derived(
    filterByLocalSearch(favoriteSongs, searchQuery, (track) => [
      track.title,
      track.artist,
      track.album,
    ]),
  );

  const filteredFavoriteAlbums = $derived(
    filterByLocalSearch(music.favoriteAlbums, searchQuery, (album) => [
      album.name,
      album.artist,
    ]),
  );

  const filteredFavoriteArtists = $derived(
    filterByLocalSearch(music.favoriteArtists, searchQuery, (artist) => [
      artist.name,
    ]),
  );

  const totalCount = $derived(
    music.favoriteTracks.length +
      music.favoriteAlbums.length +
      music.favoriteArtists.length,
  );

  const hasSearch = $derived(searchQuery.trim().length > 0);
  const showInitialLoading = $derived(
    !unavailable && loading && totalCount === 0 && !hasSearch,
  );

  const heroCoverSrc = $derived(
    favoriteSongs[0]
      ? coverArtUrl(
          music.config,
          favoriteSongs[0].coverArt ??
            favoriteSongs[0].albumId ??
            favoriteSongs[0].id,
          800,
        )
      : null,
  );

  const tracksDuration = $derived(
    formatHumanDuration(
      filteredFavoriteSongs.reduce(
        (sum, track) => sum + (track.duration ?? 0),
        0,
      ),
    ),
  );

  const heroMeta = $derived.by(() => {
    if (activeTab === "albums") {
      const count = hasSearch
        ? filteredFavoriteAlbums.length
        : music.favoriteAlbums.length;
      return count === 1 ? "1 album" : `${count} albums`;
    }
    if (activeTab === "artists") {
      const count = hasSearch
        ? filteredFavoriteArtists.length
        : music.favoriteArtists.length;
      return count === 1 ? "1 artist" : `${count} artists`;
    }
    const count = hasSearch
      ? filteredFavoriteSongs.length
      : music.favoriteTracks.length;
    const songs = count === 1 ? "1 song" : `${count} songs`;
    return tracksDuration ? `${songs} · ${tracksDuration}` : songs;
  });

  const playDisabled = $derived.by(() => {
    if (activeTab === "albums") return filteredFavoriteAlbums.length === 0;
    if (activeTab === "artists") return filteredFavoriteArtists.length === 0;
    return filteredFavoriteSongs.length === 0;
  });

  $effect(() => {
    if (unavailable) {
      loading = false;
      return;
    }
    if (!music.libraryReady) {
      loading = music.loading;
      return;
    }
    if (totalCount > 0) {
      loading = false;
      return;
    }
    loading = true;
    void music.refreshFavorites().finally(() => {
      loading = false;
    });
  });

  function playCollection() {
    if (activeTab === "albums") {
      void music.playFavoriteAlbums(filteredFavoriteAlbums);
      return;
    }
    if (activeTab === "artists") {
      void music.playFavoriteArtists(filteredFavoriteArtists);
      return;
    }
    if (filteredFavoriteSongs.length === 0) return;
    music.playTracks(filteredFavoriteSongs, 0);
  }

  function shuffleTracks() {
    if (filteredFavoriteSongs.length === 0) return;
    music.shuffle = true;
    music.playTracks(filteredFavoriteSongs, 0);
  }

  function onPageContextMenu(event: MouseEvent) {
    pageMenu = contextMenuPositionFromEvent(event);
  }

  const pageMenuItems = $derived.by((): ContextMenuEntry[] => {
    if (activeTab !== "tracks") {
      return [
        {
          id: "open",
          label: "Open favorites",
          icon: "star",
          onclick: () => {
            activeTab = "tracks";
          },
        },
      ];
    }
    return [
      {
        id: "play",
        label: "Play all songs",
        icon: "play",
        disabled: filteredFavoriteSongs.length === 0,
        onclick: playCollection,
      },
      {
        id: "shuffle",
        label: "Shuffle songs",
        icon: "shuffle",
        disabled: filteredFavoriteSongs.length === 0,
        onclick: shuffleTracks,
      },
      {
        id: "next",
        label: "Play next",
        icon: "playNext",
        disabled: filteredFavoriteSongs.length === 0,
        onclick: () => music.playTracksNext(filteredFavoriteSongs),
      },
      {
        id: "queue",
        label: "Add to queue",
        icon: "queueAdd",
        disabled: filteredFavoriteSongs.length === 0,
        onclick: () => music.addTracksToQueue(filteredFavoriteSongs),
      },
      {
        id: "select",
        label: "Select songs",
        icon: "squareCheck",
        disabled: filteredFavoriteSongs.length === 0,
        onclick: () => trackSelection.enable(),
      },
    ];
  });
</script>

<div class="favorites-page" role="group" oncontextmenu={onPageContextMenu}>
  <MusicBreadcrumbs items={[{ label: "Favorites" }]} />

  <CollectionHero
    typeLabel="Playlist"
    title="Favorites"
    meta={heroMeta}
    tone="favorites"
    icon="star"
    coverSrc={heroCoverSrc}
    coverSeed={favoriteSongs[0]?.id ?? "favorites"}
    {playDisabled}
    onplay={playCollection}
    onshuffle={activeTab === "tracks" ? shuffleTracks : undefined}
    onplaynext={activeTab === "tracks"
      ? () => music.playTracksNext(filteredFavoriteSongs)
      : undefined}
    onqueue={activeTab === "tracks"
      ? () => music.addTracksToQueue(filteredFavoriteSongs)
      : undefined}
    onselect={activeTab === "tracks"
      ? () => trackSelection.enable()
      : undefined}
  />

  <div class="favorites-toolbar">
    <Tabs.Root
      style="display: contents"
      bind:value={
        () => activeTab,
        (value) => {
          activeTab = value as FavoritesTab;
        }
      }
    >
      <Tabs.List class="favorites-tabs" aria-label="Favorite categories">
        <Tabs.Trigger value="tracks" class="favorites-tabs__btn">
          Songs
          <span class="favorites-tabs__count"
            >{music.favoriteTracks.length}</span
          >
        </Tabs.Trigger>
        <Tabs.Trigger value="albums" class="favorites-tabs__btn">
          Albums
          <span class="favorites-tabs__count"
            >{music.favoriteAlbums.length}</span
          >
        </Tabs.Trigger>
        <Tabs.Trigger value="artists" class="favorites-tabs__btn">
          Artists
          <span class="favorites-tabs__count"
            >{music.favoriteArtists.length}</span
          >
        </Tabs.Trigger>
      </Tabs.List>
    </Tabs.Root>

    <LocalSearchBox
      bind:value={searchQuery}
      placeholder="Search in Favorites"
      disabled={loading && totalCount === 0}
      resultCount={activeTab === "tracks"
        ? filteredFavoriteSongs.length
        : activeTab === "albums"
          ? filteredFavoriteAlbums.length
          : filteredFavoriteArtists.length}
      totalCount={activeTab === "tracks"
        ? music.favoriteTracks.length
        : activeTab === "albums"
          ? music.favoriteAlbums.length
          : music.favoriteArtists.length}
    />
  </div>

  {#if showInitialLoading}
    <div
      class="favorites-page__loading"
      class:favorites-page__loading--tracks={activeTab === "tracks"}
      class:favorites-page__loading--albums={activeTab === "albums"}
      class:favorites-page__loading--artists={activeTab === "artists"}
      role="status"
      aria-busy="true"
      aria-label="Loading favorites"
    >
      {#if activeTab === "tracks"}
        {#each Array.from({ length: 8 }) as _, i (i)}
          <Skeleton variant="row" />
        {/each}
      {:else if activeTab === "albums"}
        {#each Array.from({ length: 12 }) as _, i (i)}
          <Skeleton variant="card" />
        {/each}
      {:else}
        {#each Array.from({ length: 12 }) as _, i (i)}
          <Skeleton variant="avatar" class="favorites-skeleton-avatar" />
        {/each}
      {/if}
    </div>
  {:else if unavailable}
    <LibraryUnavailable />
  {:else if totalCount === 0}
    <EmptyState
      title="No favorites yet"
      message="Star a track, album, or artist and it shows up here."
      icon="star"
    >
      {#snippet actions()}
        <Link href="/music" class="favorites-page__back">Browse music</Link>
      {/snippet}
    </EmptyState>
  {:else if activeTab === "tracks"}
    {#if filteredFavoriteSongs.length === 0}
      <EmptyState
        title="No matches"
        message={hasSearch
          ? `No favorite tracks match "${searchQuery.trim()}".`
          : "No favorite tracks yet."}
        icon="search"
      />
    {:else}
      <TrackSelectionBar allTracks={filteredFavoriteSongs} />
      <div class="favorites-tracks">
        <div class="favorites-tracks__head">
          <span>#</span>
          <span>Title</span>
          <span></span>
          <span>Time</span>
        </div>
        <TrackVirtualList
          tracks={filteredFavoriteSongs}
          onplay={(i) => music.playTracks(filteredFavoriteSongs, i)}
          selectable={trackSelection.active}
          lazyThreshold={12}
        />
      </div>
    {/if}
  {:else if activeTab === "albums"}
    {#if filteredFavoriteAlbums.length === 0}
      <EmptyState
        title="No matches"
        message={hasSearch
          ? `No favorite albums match "${searchQuery.trim()}".`
          : "No favorite albums yet."}
        icon="search"
      />
    {:else}
      <AlbumGrid albums={filteredFavoriteAlbums} />
    {/if}
  {:else if filteredFavoriteArtists.length === 0}
    <EmptyState
      title="No matches"
      message={hasSearch
        ? `No favorite artists match "${searchQuery.trim()}".`
        : "No favorite artists yet."}
      icon="search"
    />
  {:else}
    <ArtistGrid artists={filteredFavoriteArtists} lazyThreshold={24} />
  {/if}
</div>

{#if pageMenu}
  <ContextMenu
    x={pageMenu.x}
    y={pageMenu.y}
    items={pageMenuItems}
    label="Favorites actions"
    onclose={() => (pageMenu = null)}
  />
{/if}

<style>
  .favorites-page {
    display: flex;
    flex-direction: column;
    gap: var(--jb-space-6);
  }

  .favorites-toolbar {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    justify-content: space-between;
    gap: var(--jb-space-3);
  }

  :global(.favorites-tabs) {
    display: flex;
    flex-wrap: wrap;
    gap: var(--jb-space-2);
  }

  :global(.favorites-tabs__btn) {
    display: inline-flex;
    align-items: center;
    gap: var(--jb-space-2);
    padding: 0.4rem 0.9rem;
    border: none;
    border-radius: var(--jb-radius-full);
    background: var(--jb-surface);
    color: var(--jb-text-muted);
    font-weight: 600;
    font-size: 0.875rem;
    cursor: pointer;
  }

  :global(.favorites-tabs__btn:hover) {
    background: var(--jb-surface-hover);
    color: var(--jb-text);
  }

  :global(.favorites-tabs__btn[data-state="active"]) {
    background: var(--jb-text);
    color: var(--jb-bg);
  }

  :global(.favorites-tabs__btn[data-state="active"]:hover) {
    background: var(--jb-text);
    color: var(--jb-bg);
  }

  .favorites-tabs__count {
    font-variant-numeric: tabular-nums;
    font-size: 0.75rem;
    opacity: 0.7;
  }

  .favorites-toolbar :global(.local-search-box) {
    flex: 1;
    min-width: 12rem;
    max-width: 22rem;
  }

  .favorites-tracks {
    padding-bottom: var(--jb-space-2);
  }

  .favorites-tracks__head {
    display: grid;
    grid-template-columns: 2rem 1fr auto 4rem;
    gap: var(--jb-space-3);
    padding: var(--jb-space-3) 0;
    border-bottom: 1px solid var(--jb-border);
    font-size: 0.75rem;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    color: var(--jb-text-subtle);
  }

  .favorites-page__loading--tracks {
    display: flex;
    flex-direction: column;
    gap: var(--jb-space-2);
  }

  .favorites-page__loading--albums {
    display: grid;
    grid-template-columns: repeat(
      auto-fill,
      minmax(min(100%, var(--jb-card-grid-min)), 1fr)
    );
    gap: var(--jb-space-5);
  }

  .favorites-page__loading--artists {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(9.5rem, 1fr));
    gap: var(--jb-space-5);
    justify-items: center;
  }

  :global(.favorites-skeleton-avatar) {
    width: 100%;
    max-width: 9.5rem;
    aspect-ratio: 1;
    height: auto;
    border-radius: 50%;
  }

  :global(a.favorites-page__back) {
    color: var(--jb-music-accent);
    font-weight: 600;
    text-decoration: none;
  }

  @media (max-width: 768px) {
    .favorites-toolbar {
      flex-direction: column;
      align-items: stretch;
    }

    :global(.favorites-tabs) {
      overflow-x: auto;
      flex-wrap: nowrap;
      padding-bottom: var(--jb-space-1);
      scrollbar-width: thin;
    }

    :global(.favorites-tabs__btn) {
      flex-shrink: 0;
    }

    .favorites-toolbar :global(.local-search-box) {
      max-width: none;
    }

    .favorites-tracks__head {
      display: none;
    }

    .favorites-page__loading--albums {
      grid-template-columns: repeat(2, minmax(0, 1fr));
      gap: var(--jb-space-4);
    }

    .favorites-page__loading--artists {
      grid-template-columns: repeat(2, minmax(0, 1fr));
      gap: var(--jb-space-4);
    }
  }
</style>
