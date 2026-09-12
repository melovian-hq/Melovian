<script lang="ts">
  import PageHeader from "$lib/components/ui/PageHeader.svelte";
  import Section from "$lib/components/ui/Section.svelte";
  import Input from "$lib/components/ui/Input.svelte";
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";
  import EmptyState from "$lib/components/ui/EmptyState.svelte";
  import Skeleton from "$lib/components/ui/Skeleton.svelte";
  import Spinner from "$lib/components/ui/Spinner.svelte";
  import AlbumGrid from "$lib/components/music/AlbumGrid.svelte";
  import ArtistGrid from "$lib/components/music/ArtistGrid.svelte";
  import TrackVirtualList from "$lib/components/music/TrackVirtualList.svelte";
  import LibraryUnavailable from "$lib/components/music/LibraryUnavailable.svelte";
  import { music } from "$lib/config/music.svelte";
  import { sources } from "$lib/features/sources/store.svelte";
  import { libraryUnavailable } from "$lib/music/library-gate";
  import { isLocalMusicId } from "$lib/music/library-adapter";
  import { mapClientError } from "$lib/ui/client-error";
  import type { SubsonicSearchResult, SubsonicSong } from "$lib/subsonic";
  import {
    rejectUnknownAlbums,
    rejectUnknownArtists,
    rejectUnknownTracks,
  } from "$lib/music/unknown-metadata";
  import {
    addSearchHistory,
    loadSearchHistory,
    saveSearchHistory,
  } from "$lib/music/search-history";
  import { confirmDialog } from "$lib/ui/confirm.svelte";
  import ContextMenu from "$lib/components/ui/ContextMenu.svelte";
  import {
    contextMenuPositionFromEvent,
    type ContextMenuEntry,
  } from "$lib/components/ui/context-menu";

  let query = $state("");
  let searching = $state(false);
  let didYouMean = $state<string | null>(null);
  let history = $state<string[]>(loadSearchHistory());
  let result = $state<SubsonicSearchResult>({
    artists: [],
    albums: [],
    songs: [],
  });
  let similar = $state<SubsonicSong[]>([]);
  let similarLoading = $state(false);
  let lastQuery = $state("");
  let searchError = $state<string | null>(null);
  let pageMenu = $state<{ x: number; y: number } | null>(null);
  let sourceFilter = $state<"all" | "local" | "server">("all");

  const unavailable = $derived(libraryUnavailable());
  const showSourceFilter = $derived(sources.hasUnifiedMode);

  let debounce: ReturnType<typeof setTimeout> | undefined;
  let searchRequest = 0;
  let similarRequest = 0;

  async function run(term: string) {
    const trimmed = term.trim();
    if (!trimmed) {
      lastQuery = "";
      result = { artists: [], albums: [], songs: [] };
      similar = [];
      similarLoading = false;
      didYouMean = null;
      searchError = null;
      searching = false;
      return;
    }
    if (trimmed === lastQuery) return;
    lastQuery = trimmed;
    const request = ++searchRequest;
    searching = true;
    searchError = null;
    similar = [];
    similarLoading = false;
    try {
      const res = await music.searchAll(trimmed);
      if (request !== searchRequest) return;
      result = res.result;
      didYouMean = res.suggestion;
      history = addSearchHistory(trimmed, history);
      saveSearchHistory(history);
      const topSong = res.result.songs[0];
      if (topSong) {
        const similarReq = ++similarRequest;
        similarLoading = true;
        void music
          .searchSimilarTracks(topSong.id)
          .then((tracks) => {
            if (similarReq !== similarRequest) return;
            similar = tracks;
          })
          .catch(() => {
            if (similarReq !== similarRequest) return;
            similar = [];
          })
          .finally(() => {
            if (similarReq === similarRequest) similarLoading = false;
          });
      }
    } catch (err) {
      if (request !== searchRequest) return;
      searchError = mapClientError(err).message;
      result = { artists: [], albums: [], songs: [] };
      similar = [];
      didYouMean = null;
    } finally {
      if (request === searchRequest) searching = false;
    }
  }

  $effect(() => {
    const term = query.trim();
    clearTimeout(debounce);
    // Single characters rarely yield useful matches. Wait for more input but
    // search immediately once the field is cleared.
    const delay = term.length === 0 ? 0 : term.length < 2 ? 450 : 240;
    debounce = setTimeout(() => void run(query), delay);
    return () => clearTimeout(debounce);
  });

  function applySuggestion(term: string) {
    query = term;
  }

  function applyHistory(term: string) {
    query = term;
  }

  async function clearHistory() {
    const ok = await confirmDialog.confirm({
      title: "Clear search history",
      message: "Remove all recent searches from this device?",
      confirmLabel: "Clear history",
      danger: true,
    });
    if (!ok) return;
    history = [];
    saveSearchHistory(history);
  }

  const showHistory = $derived(
    query.trim().length === 0 && history.length > 0 && !searching,
  );

  function matchesSource(id: string): boolean {
    if (!showSourceFilter || sourceFilter === "all") return true;
    const local = isLocalMusicId(id);
    return sourceFilter === "local" ? local : !local;
  }

  const visibleArtists = $derived(
    rejectUnknownArtists(result.artists, music.hideUnknownMetadata).filter(
      (artist) => matchesSource(artist.id),
    ),
  );
  const visibleAlbums = $derived(
    rejectUnknownAlbums(result.albums, music.hideUnknownMetadata).filter(
      (album) => matchesSource(album.id),
    ),
  );
  const visibleSongs = $derived(
    rejectUnknownTracks(result.songs, music.hideUnknownMetadata).filter(
      (song) => matchesSource(song.id),
    ),
  );
  const visibleSimilar = $derived(
    rejectUnknownTracks(similar, music.hideUnknownMetadata).filter((song) =>
      matchesSource(song.id),
    ),
  );

  const hasResults = $derived(
    visibleArtists.length > 0 ||
      visibleAlbums.length > 0 ||
      visibleSongs.length > 0,
  );

  const showEmpty = $derived(
    lastQuery.length > 0 && !searching && !hasResults && !searchError,
  );

  function onPageContextMenu(event: MouseEvent) {
    pageMenu = contextMenuPositionFromEvent(event);
  }

  const pageMenuItems = $derived.by((): ContextMenuEntry[] => {
    if (showHistory && history.length > 0) {
      return [
        {
          id: "clear-history",
          label: "Clear recent searches",
          icon: "trash2",
          danger: true,
          onclick: () => void clearHistory(),
        },
      ];
    }
    if (lastQuery.length > 0 && hasResults) {
      return [
        {
          id: "play-songs",
          label: "Play all songs",
          icon: "play",
          disabled: visibleSongs.length === 0,
          onclick: () => music.playTracks(visibleSongs, 0),
        },
        {
          id: "queue-songs",
          label: "Queue all songs",
          icon: "queueAdd",
          disabled: visibleSongs.length === 0,
          onclick: () => music.addTracksToQueue(visibleSongs),
        },
      ];
    }
    return [];
  });
</script>

<div class="search-page" role="group" oncontextmenu={onPageContextMenu}>
  <PageHeader title="Search" subtitle="Find artists, albums, and tracks." />

  {#if unavailable}
    <LibraryUnavailable />
  {:else}
    <div class="search-box">
      <MdiIcon name="search" size={20} />
      <Input bind:value={query} placeholder="Search your library" />
      {#if searching}
        <Spinner />
      {/if}
    </div>

    {#if showSourceFilter}
      <div class="search-source" role="tablist" aria-label="Source filter">
        {#each [{ id: "all", label: "All sources" }, { id: "local", label: "Local" }, { id: "server", label: "Server" }] as option (option.id)}
          <button
            type="button"
            role="tab"
            class="search-source__btn"
            class:search-source__btn--active={sourceFilter === option.id}
            aria-selected={sourceFilter === option.id}
            onclick={() => {
              sourceFilter = option.id as "all" | "local" | "server";
            }}
          >
            {option.label}
          </button>
        {/each}
      </div>
    {/if}

    {#if showHistory}
      <div class="search-history" aria-label="Recent searches">
        <div class="search-history__head">
          <span class="search-history__label">Recent searches</span>
          <button
            type="button"
            class="search-history__clear"
            onclick={() => void clearHistory()}
          >
            Clear
          </button>
        </div>
        {#each history as term (term)}
          <button
            type="button"
            class="search-history__item"
            onclick={() => applyHistory(term)}
          >
            {term}
          </button>
        {/each}
      </div>
    {/if}

    {#if searching && lastQuery && !hasResults}
      <div
        class="search-skeleton"
        role="status"
        aria-busy="true"
        aria-label="Searching"
      >
        <div class="search-skeleton__section">
          <Skeleton class="search-skeleton__heading" />
          <div class="search-skeleton__avatars">
            {#each Array.from({ length: 4 }) as _, i (i)}
              <Skeleton variant="avatar" class="search-skeleton__avatar" />
            {/each}
          </div>
        </div>
        <div class="search-skeleton__section">
          <Skeleton class="search-skeleton__heading" />
          <div class="search-skeleton__cards">
            {#each Array.from({ length: 4 }) as _, i (i)}
              <Skeleton variant="card" />
            {/each}
          </div>
        </div>
        <div class="search-skeleton__section">
          <Skeleton class="search-skeleton__heading" />
          <div class="search-skeleton__rows">
            {#each Array.from({ length: 5 }) as _, i (i)}
              <Skeleton variant="row" />
            {/each}
          </div>
        </div>
      </div>
    {/if}

    {#if searching && lastQuery && hasResults}
      <p class="search-status" aria-live="polite">
        Searching for "{lastQuery}"…
      </p>
    {/if}

    {#if didYouMean}
      <p class="did-you-mean">
        Did you mean
        <button type="button" onclick={() => applySuggestion(didYouMean!)}>
          {didYouMean}
        </button>?
      </p>
    {/if}

    {#if searchError && !searching}
      <EmptyState
        title="Search failed"
        message={searchError}
        icon="alertCircle"
      />
    {/if}

    {#if showEmpty}
      <EmptyState
        title="No matches"
        message={`Nothing found for "${lastQuery}".`}
        icon="search"
      />
    {/if}

    <div class="search-results" class:search-results--pending={searching}>
      {#if visibleArtists.length > 0}
        <Section title="Artists">
          <ArtistGrid artists={visibleArtists} size="sm" />
        </Section>
      {/if}

      {#if visibleAlbums.length > 0}
        <Section title="Albums">
          <AlbumGrid albums={visibleAlbums} />
        </Section>
      {/if}

      {#if visibleSongs.length > 0}
        <Section title="Songs">
          <div class="track-list-island">
            <TrackVirtualList
              tracks={visibleSongs}
              onplay={(i) => music.playTracks(visibleSongs, i)}
              lazyThreshold={12}
            />
          </div>
        </Section>
      {/if}

      {#if similarLoading && visibleSimilar.length === 0}
        <Section title="Similar tracks">
          <div
            class="search-skeleton__rows"
            role="status"
            aria-busy="true"
            aria-label="Loading similar tracks"
          >
            {#each Array.from({ length: 4 }) as _, i (i)}
              <Skeleton variant="row" />
            {/each}
          </div>
        </Section>
      {/if}

      {#if visibleSimilar.length > 0}
        <Section title="Similar tracks">
          <div class="track-list-island">
            <TrackVirtualList
              tracks={visibleSimilar}
              onplay={(i) => music.playTracks(visibleSimilar, i)}
              lazyThreshold={12}
            />
          </div>
        </Section>
      {/if}
    </div>
  {/if}
</div>

{#if pageMenu && pageMenuItems.length > 0}
  <ContextMenu
    x={pageMenu.x}
    y={pageMenu.y}
    items={pageMenuItems}
    label="Search actions"
    onclose={() => (pageMenu = null)}
  />
{/if}

<style>
  .search-page {
    display: flex;
    flex-direction: column;
    gap: var(--jb-space-6);
  }

  .search-box {
    display: flex;
    align-items: center;
    gap: var(--jb-space-3);
    padding: 0 var(--jb-space-4);
    border: 1px solid var(--jb-border);
    border-radius: var(--jb-radius-full);
    background: var(--jb-surface);
    color: var(--jb-text-muted);
  }

  .search-box :global(.input) {
    border: none;
    background: transparent;
    padding-inline: 0;
  }

  .search-box :global(.input:focus) {
    box-shadow: none;
  }

  .search-source {
    display: flex;
    flex-wrap: wrap;
    gap: var(--jb-space-2);
  }

  .search-source__btn {
    padding: 0.35rem 0.8rem;
    border-radius: var(--jb-radius-full);
    border: 1px solid var(--jb-border);
    background: var(--jb-surface);
    color: var(--jb-text-muted);
    font: inherit;
    font-size: 0.8125rem;
    font-weight: 600;
    cursor: pointer;
  }

  .search-source__btn--active {
    border-color: var(--jb-music-accent);
    background: var(--jb-music-accent-muted);
    color: var(--jb-music-accent);
  }

  .search-history {
    display: flex;
    flex-direction: column;
    gap: var(--jb-space-2);
  }

  .search-history__head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--jb-space-2);
  }

  .search-history__label {
    font-size: 0.8125rem;
    font-weight: 600;
    color: var(--jb-text-muted);
  }

  .search-history__clear {
    border: none;
    background: transparent;
    color: var(--jb-text-muted);
    font-size: 0.8125rem;
    font-weight: 600;
    cursor: pointer;
    padding: 0;
  }

  .search-history__clear:hover {
    color: var(--jb-accent);
  }

  .search-history__item {
    border: none;
    background: transparent;
    color: var(--jb-text-muted);
    font-size: 0.875rem;
    cursor: pointer;
    padding: 0;
  }

  .search-history__item:hover {
    color: var(--jb-accent);
  }

  .did-you-mean {
    margin: 0;
    color: var(--jb-text-muted);
  }

  .did-you-mean button {
    border: none;
    background: transparent;
    color: var(--jb-accent);
    font-weight: 700;
    cursor: pointer;
    padding: 0;
  }

  .search-status {
    font-size: 0.8125rem;
    color: var(--jb-text-muted);
  }

  .search-skeleton {
    display: flex;
    flex-direction: column;
    gap: var(--jb-space-6);
  }

  .search-skeleton__section {
    display: flex;
    flex-direction: column;
    gap: var(--jb-space-4);
  }

  :global(.search-skeleton__heading) {
    width: 7rem;
    height: 1.25rem;
  }

  .search-skeleton__avatars {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(5.5rem, 1fr));
    gap: var(--jb-space-4);
    justify-items: center;
  }

  :global(.search-skeleton__avatar) {
    width: 4.5rem;
    height: 4.5rem;
  }

  .search-skeleton__cards {
    display: grid;
    grid-template-columns: repeat(
      auto-fill,
      minmax(min(100%, var(--jb-card-grid-min)), 1fr)
    );
    gap: var(--jb-space-4);
  }

  .search-skeleton__rows {
    display: flex;
    flex-direction: column;
    gap: var(--jb-space-2);
  }

  .search-results {
    display: flex;
    flex-direction: column;
    gap: var(--jb-space-6);
    transition: opacity 0.15s ease;
  }

  .search-results--pending {
    opacity: 0.72;
  }

  @media (max-width: 768px) {
    .search-page {
      gap: var(--jb-space-4);
    }

    .search-box {
      padding: 0 var(--jb-space-3);
    }
  }
</style>
