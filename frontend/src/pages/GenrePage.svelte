<script lang="ts">
  import PageHeader from "$lib/components/ui/PageHeader.svelte";
  import EmptyState from "$lib/components/ui/EmptyState.svelte";
  import Skeleton from "$lib/components/ui/Skeleton.svelte";
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";
  import TrackVirtualList from "$lib/components/music/TrackVirtualList.svelte";
  import TrackSelectionBar from "$lib/components/music/TrackSelectionBar.svelte";
  import LocalSearchBox from "$lib/components/ui/LocalSearchBox.svelte";
  import ContextMenu from "$lib/components/ui/ContextMenu.svelte";
  import {
    contextMenuPositionFromEvent,
    type ContextMenuEntry,
  } from "$lib/components/ui/context-menu";
  import { trackSelection } from "$lib/music/selection.svelte";
  import { music } from "$lib/config/music.svelte";
  import { filterByLocalSearch } from "$lib/utils/local-search";
  import type { SubsonicSong } from "$lib/subsonic";
  import { rejectUnknownTracks } from "$lib/music/unknown-metadata";
  import { createPagedList } from "$lib/ui/async-page.svelte";
  import { APP_NAME } from "$lib/brand";
  import { setPageMeta } from "$lib/seo/meta";

  interface Props {
    genre: string;
  }

  let { genre }: Props = $props();

  const name = $derived(decodeURIComponent(genre));

  $effect(() => {
    const genreName = name.trim() || "Genre";
    setPageMeta({
      title: genreName,
      description: `Browse ${genreName} tracks in ${APP_NAME}.`,
    });
  });

  const PAGE_SIZE = 200;
  let searchQuery = $state("");
  let pageMenu = $state<{ x: number; y: number } | null>(null);

  const list = createPagedList<SubsonicSong>({
    errorMessage: "Failed to load genre tracks",
    load: async (offset) => {
      const current = name;
      if (offset === 0) list.items = [];
      if (!music.libraryReady) {
        throw new Error(music.error ?? "Not connected");
      }
      const page = await music.getGenreSongs(current, PAGE_SIZE, offset);
      return { items: page, hasMore: page.length === PAGE_SIZE };
    },
  });

  const visibleSongs = $derived(
    rejectUnknownTracks(list.items, music.hideUnknownMetadata),
  );

  const filteredSongs = $derived(
    filterByLocalSearch(visibleSongs, searchQuery, (track) => [
      track.title,
      track.artist,
      track.album,
    ]),
  );
  const hasSearch = $derived(searchQuery.trim().length > 0);

  function addVisibleToQueue() {
    if (filteredSongs.length === 0) return;
    music.addTracksToQueue(filteredSongs);
  }

  function playVisibleNext() {
    if (filteredSongs.length === 0) return;
    music.playTracksNext(filteredSongs);
  }
  const showInitialLoading = $derived(
    list.loading && list.items.length === 0 && !hasSearch,
  );

  function onPageContextMenu(event: MouseEvent) {
    pageMenu = contextMenuPositionFromEvent(event);
  }

  function playGenreNow(shuffle = false) {
    void music.playGenre(name, shuffle);
  }

  const pageMenuItems = $derived.by((): ContextMenuEntry[] => [
    {
      id: "play",
      label: "Play now",
      icon: "play",
      disabled: filteredSongs.length === 0,
      onclick: () => playGenreNow(false),
    },
    {
      id: "shuffle",
      label: "Shuffle",
      icon: "shuffle",
      disabled: filteredSongs.length === 0,
      onclick: () => playGenreNow(true),
    },
    {
      id: "queue",
      label: "Add to queue",
      icon: "queueAdd",
      disabled: filteredSongs.length === 0,
      onclick: () => void music.addGenreToQueue(name),
    },
    {
      id: "next",
      label: "Play next",
      icon: "playNext",
      disabled: filteredSongs.length === 0,
      onclick: () => void music.playGenreNext(name),
    },
    {
      id: "select",
      label: "Select tracks",
      icon: "squareCheck",
      disabled: filteredSongs.length === 0,
      onclick: () => trackSelection.enable(),
    },
  ]);
</script>

<div class="genre-page" role="group" oncontextmenu={onPageContextMenu}>
  <PageHeader
    title={name}
    subtitle={list.loading && list.items.length === 0
      ? "Loading..."
      : hasSearch
        ? `${filteredSongs.length} of ${visibleSongs.length} tracks`
        : `${visibleSongs.length} tracks`}
  >
    {#snippet actions()}
      <button
        type="button"
        class="genre-action"
        disabled={filteredSongs.length === 0}
        onclick={() => music.playTracks(filteredSongs, 0)}
      >
        <MdiIcon name="play" size={18} /> Play
      </button>
      <button
        type="button"
        class="genre-action genre-action--ghost"
        disabled={filteredSongs.length === 0}
        onclick={() => void playVisibleNext()}
      >
        <MdiIcon name="playNext" size={16} /> Play next
      </button>
      <button
        type="button"
        class="genre-action genre-action--ghost"
        disabled={filteredSongs.length === 0}
        onclick={addVisibleToQueue}
      >
        <MdiIcon name="queueAdd" size={16} /> Add to queue
      </button>
      <button
        type="button"
        class="genre-action genre-action--ghost"
        disabled={filteredSongs.length === 0}
        onclick={() => {
          music.shuffle = true;
          music.playTracks(filteredSongs, 0);
        }}
      >
        <MdiIcon name="shuffle" size={16} /> Shuffle
      </button>
    {/snippet}
  </PageHeader>

  <LocalSearchBox
    bind:value={searchQuery}
    placeholder="Search tracks in this genre"
    disabled={list.loading && list.items.length === 0}
    resultCount={filteredSongs.length}
    totalCount={visibleSongs.length}
  />

  {#if showInitialLoading}
    <div class="track-list-island">
      {#each Array.from({ length: 8 }) as _, i (i)}
        <Skeleton class="track-skeleton" />
      {/each}
    </div>
  {:else if list.error && list.items.length === 0}
    <EmptyState
      title="Could not load genre"
      message={list.error}
      icon="alertCircle"
    />
  {:else if visibleSongs.length === 0}
    <EmptyState
      title="No tracks"
      message={`No songs found for ${name}.`}
      icon="tag"
    />
  {:else if filteredSongs.length === 0}
    <EmptyState
      title="No matches"
      message={`No tracks match "${searchQuery.trim()}".`}
      icon="search"
    />
  {:else}
    <TrackSelectionBar allTracks={filteredSongs} />
    <div class="track-list-island">
      <TrackVirtualList
        tracks={filteredSongs}
        onplay={(i) => music.playTracks(filteredSongs, i)}
        selectable={trackSelection.active}
        onNearEnd={hasSearch ? undefined : list.loadMore}
        lazyThreshold={12}
      />
    </div>
    {#if !hasSearch && list.hasMore && !list.loadingMore}
      <p class="load-more-hint">Scroll down to load more tracks</p>
    {:else if list.loadingMore}
      <p class="load-more-hint">Loading more...</p>
    {/if}
  {/if}
</div>

{#if pageMenu}
  <ContextMenu
    x={pageMenu.x}
    y={pageMenu.y}
    items={pageMenuItems}
    label="Genre actions"
    onclose={() => (pageMenu = null)}
  />
{/if}

<style>
  .genre-page {
    display: flex;
    flex-direction: column;
    gap: var(--jb-space-4);
  }

  .genre-action {
    display: inline-flex;
    align-items: center;
    gap: var(--jb-space-2);
    padding: 0.5rem 1rem;
    border: none;
    border-radius: var(--jb-radius-full);
    background: var(--jb-accent);
    color: var(--jb-accent-text);
    font-weight: 700;
    font-size: 0.875rem;
    cursor: pointer;
  }

  .genre-action--ghost {
    background: var(--jb-surface);
    color: var(--jb-text);
    border: 1px solid var(--jb-border);
  }

  .genre-action:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  :global(.track-skeleton) {
    height: 3.5rem;
    border-radius: var(--jb-radius-md);
  }

  .load-more-hint {
    align-self: center;
    margin: var(--jb-space-2) 0 0;
    font-size: 0.8125rem;
    color: var(--jb-text-subtle);
  }

  @media (max-width: 768px) {
    .genre-page :global(.page-header__actions) {
      width: 100%;
      align-items: center;
    }

    .genre-action {
      flex: 0 1 auto;
      justify-content: center;
      padding: 0.375rem 0.75rem;
      font-size: 0.8125rem;
      white-space: nowrap;
    }
  }
</style>
