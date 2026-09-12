<script lang="ts">
  import PageHeader from "$lib/components/ui/PageHeader.svelte";
  import EmptyState from "$lib/components/ui/EmptyState.svelte";
  import Skeleton from "$lib/components/ui/Skeleton.svelte";
  import Spinner from "$lib/components/ui/Spinner.svelte";
  import Button from "$lib/components/ui/Button.svelte";
  import LocalSearchBox from "$lib/components/ui/LocalSearchBox.svelte";
  import AlbumGrid from "$lib/components/music/AlbumGrid.svelte";
  import LibraryUnavailable from "$lib/components/music/LibraryUnavailable.svelte";
  import { music } from "$lib/config/music.svelte";
  import { libraryUnavailable } from "$lib/music/library-gate";
  import { type SubsonicAlbum } from "$lib/subsonic";
  import { filterByLocalSearch } from "$lib/utils/local-search";
  import { rejectUnknownAlbums } from "$lib/music/unknown-metadata";
  import { Tabs } from "bits-ui";

  type AlbumSort = "alphabeticalByName" | "newest" | "frequent";

  const SORT_OPTIONS: { id: AlbumSort; label: string }[] = [
    { id: "alphabeticalByName", label: "A-Z" },
    { id: "newest", label: "Newest" },
    { id: "frequent", label: "Most played" },
  ];

  const ALBUM_PAGE_SIZE = 200;

  let loading = $state(true);
  let loadingMore = $state(false);
  let hasMore = $state(true);
  let sort = $state<AlbumSort>("alphabeticalByName");
  let albums = $state.raw<SubsonicAlbum[]>([]);
  let searchQuery = $state("");

  const unavailable = $derived(libraryUnavailable());

  const visibleAlbums = $derived(
    rejectUnknownAlbums(albums, music.hideUnknownMetadata),
  );

  const filteredAlbums = $derived(
    filterByLocalSearch(visibleAlbums, searchQuery, (album) => [
      album.name,
      album.artist,
    ]),
  );

  const hasSearch = $derived(searchQuery.trim().length > 0);
  const showInitialLoading = $derived(
    !unavailable && loading && albums.length === 0 && !hasSearch,
  );

  async function loadMoreAlbums() {
    if (unavailable || loading || loadingMore || !hasMore || hasSearch) return;
    loadingMore = true;
    try {
      const items = await music.library.getAlbumList2(
        sort,
        ALBUM_PAGE_SIZE,
        albums.length,
      );
      if (items.length < ALBUM_PAGE_SIZE) {
        hasMore = false;
      }
      if (items.length > 0) {
        albums = [...albums, ...items];
      }
    } catch {
      hasMore = false;
    } finally {
      loadingMore = false;
    }
  }

  $effect(() => {
    const activeSort = sort;
    if (unavailable) {
      loading = false;
      albums = [];
      return;
    }
    if (!music.libraryReady) {
      loading = music.loading;
      return;
    }
    let cancelled = false;
    loading = true;
    hasMore = true;
    albums = [];
    void music.library
      .getAlbumList2(activeSort, ALBUM_PAGE_SIZE, 0)
      .then((items) => {
        if (cancelled) return;
        albums = items;
        hasMore = items.length >= ALBUM_PAGE_SIZE;
      })
      .catch(() => {
        if (!cancelled) albums = [];
      })
      .finally(() => {
        if (!cancelled) loading = false;
      });
    return () => {
      cancelled = true;
    };
  });
</script>

<div class="albums-page">
  <div class="albums-page__intro">
    <PageHeader title="Albums" subtitle="Browse albums from your server." />

    <Tabs.Root
      style="display: contents"
      bind:value={
        () => sort,
        (value) => {
          sort = value as AlbumSort;
        }
      }
    >
      <Tabs.List class="albums-page__sort" aria-label="Album sort">
        {#each SORT_OPTIONS as option (option.id)}
          <Tabs.Trigger
            value={option.id}
            class="albums-page__sort-btn"
            disabled={unavailable}
          >
            {option.label}
          </Tabs.Trigger>
        {/each}
      </Tabs.List>
    </Tabs.Root>
  </div>

  <LocalSearchBox
    bind:value={searchQuery}
    placeholder="Search albums"
    disabled={unavailable || (loading && albums.length === 0)}
    resultCount={filteredAlbums.length}
    totalCount={visibleAlbums.length}
  />

  {#if showInitialLoading}
    <div
      class="albums-page__loading"
      role="status"
      aria-busy="true"
      aria-label="Loading albums"
    >
      {#each Array.from({ length: 12 }) as _, i (i)}
        <Skeleton variant="card" />
      {/each}
    </div>
  {:else if unavailable}
    <LibraryUnavailable />
  {:else if visibleAlbums.length === 0}
    <EmptyState
      title="No albums"
      message="Your server did not return any albums for this view."
      icon="album"
    />
  {:else if filteredAlbums.length === 0}
    <EmptyState
      title="No matches"
      message={`No albums match "${searchQuery.trim()}".`}
      icon="search"
    />
  {:else}
    <AlbumGrid
      albums={filteredAlbums}
      size="md"
      lazyThreshold={18}
      onNearEnd={() => void loadMoreAlbums()}
    />
    {#if !hasSearch && hasMore}
      <div class="albums-page__more">
        {#if loadingMore}
          <Spinner />
        {:else}
          <Button variant="surface" onclick={() => void loadMoreAlbums()}>
            Load more albums
          </Button>
        {/if}
      </div>
    {/if}
  {/if}
</div>

<style>
  .albums-page {
    display: flex;
    flex-direction: column;
    gap: var(--jb-page-gap);
  }

  .albums-page__intro {
    display: flex;
    flex-direction: column;
    gap: var(--jb-space-3);
  }

  .albums-page__intro :global(.page-header) {
    margin-bottom: 0;
  }

  :global(.albums-page__sort) {
    display: flex;
    flex-wrap: wrap;
    gap: var(--jb-space-2);
  }

  :global(.albums-page__sort-btn) {
    padding: 0.4rem 0.85rem;
    border-radius: var(--jb-radius-full);
    border: 1px solid var(--jb-border);
    background: var(--jb-surface);
    color: var(--jb-text-muted);
    font: inherit;
    font-weight: 600;
    cursor: pointer;
  }

  :global(.albums-page__sort-btn[data-state="active"]) {
    border-color: var(--jb-music-accent);
    background: var(--jb-music-accent-muted);
    color: var(--jb-music-accent);
  }

  .albums-page__loading {
    display: grid;
    grid-template-columns: repeat(
      auto-fill,
      minmax(min(100%, var(--jb-card-grid-min)), 1fr)
    );
    gap: var(--jb-space-5);
  }

  .albums-page__more {
    display: flex;
    justify-content: center;
    padding-top: var(--jb-space-2);
  }

  @media (max-width: 768px) {
    :global(.albums-page__sort) {
      overflow-x: auto;
      flex-wrap: nowrap;
      padding-bottom: var(--jb-space-1);
      scrollbar-width: thin;
    }

    :global(.albums-page__sort-btn) {
      flex-shrink: 0;
    }
  }

  @media (max-width: 480px) {
    .albums-page__loading {
      grid-template-columns: repeat(2, minmax(0, 1fr));
      gap: var(--jb-space-4);
    }
  }
</style>
