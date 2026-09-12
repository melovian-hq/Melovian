<script lang="ts">
  import PageHeader from "$lib/components/ui/PageHeader.svelte";
  import EmptyState from "$lib/components/ui/EmptyState.svelte";
  import Skeleton from "$lib/components/ui/Skeleton.svelte";
  import LocalSearchBox from "$lib/components/ui/LocalSearchBox.svelte";
  import ArtistGrid from "$lib/components/music/ArtistGrid.svelte";
  import LibraryUnavailable from "$lib/components/music/LibraryUnavailable.svelte";
  import { music } from "$lib/config/music.svelte";
  import { libraryUnavailable } from "$lib/music/library-gate";
  import { filterByLocalSearch } from "$lib/utils/local-search";
  import { rejectUnknownArtists } from "$lib/music/unknown-metadata";
  import { createAsyncPage } from "$lib/ui/async-page.svelte";

  let searchQuery = $state("");

  const unavailable = $derived(libraryUnavailable());

  const visibleArtists = $derived(
    rejectUnknownArtists(music.allArtists, music.hideUnknownMetadata),
  );

  const filteredArtists = $derived(
    filterByLocalSearch(visibleArtists, searchQuery, (artist) => [artist.name]),
  );

  const hasSearch = $derived(searchQuery.trim().length > 0);

  const page = createAsyncPage({
    load: (run) => {
      if (unavailable) return run.skip();
      if (!music.libraryReady) return run.wait(music.loading);
      if (music.allArtists.length > 0) return run.skip();
      return music.loadArtists();
    },
  });

  const showInitialLoading = $derived(
    !unavailable && page.loading && music.allArtists.length === 0 && !hasSearch,
  );
</script>

<div class="artists-page">
  <PageHeader title="Artists" subtitle="Browse your library by artist." />

  <LocalSearchBox
    bind:value={searchQuery}
    placeholder="Search artists"
    disabled={unavailable || (page.loading && music.allArtists.length === 0)}
    resultCount={filteredArtists.length}
    totalCount={visibleArtists.length}
  />

  {#if showInitialLoading}
    <div class="artists-page__loading">
      {#each Array.from({ length: 12 }) as _, i (i)}
        <Skeleton class="artist-skeleton" />
      {/each}
    </div>
  {:else if unavailable}
    <LibraryUnavailable />
  {:else if visibleArtists.length === 0}
    <EmptyState
      title="No artists"
      message="Your server did not report any artists."
      icon="accountMusic"
    />
  {:else if filteredArtists.length === 0}
    <EmptyState
      title="No matches"
      message={`No artists match "${searchQuery.trim()}".`}
      icon="search"
    />
  {:else}
    <ArtistGrid artists={filteredArtists} lazyThreshold={12} />
  {/if}
</div>

<style>
  .artists-page {
    display: flex;
    flex-direction: column;
    gap: var(--jb-page-gap);
  }

  .artists-page__loading {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(9.5rem, 1fr));
    gap: var(--jb-space-5);
  }

  :global(.artist-skeleton) {
    aspect-ratio: 1;
    border-radius: 50%;
  }

  @media (max-width: 480px) {
    .artists-page__loading {
      grid-template-columns: repeat(2, minmax(0, 1fr));
      gap: var(--jb-space-4);
    }
  }
</style>
