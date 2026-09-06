<script lang="ts">
  import AppShell from "$lib/components/layout/AppShell.svelte";
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

  let loading = $state(true);
  let searchQuery = $state("");

  const unavailable = $derived(libraryUnavailable());

  const visibleArtists = $derived(
    rejectUnknownArtists(music.allArtists, music.hideUnknownMetadata),
  );

  const filteredArtists = $derived(
    filterByLocalSearch(visibleArtists, searchQuery, (artist) => [artist.name]),
  );

  const hasSearch = $derived(searchQuery.trim().length > 0);
  const showInitialLoading = $derived(
    !unavailable && loading && music.allArtists.length === 0 && !hasSearch,
  );

  $effect(() => {
    if (unavailable) {
      loading = false;
      return;
    }
    if (!music.libraryReady) {
      loading = music.loading;
      return;
    }
    if (music.allArtists.length > 0) {
      loading = false;
      return;
    }
    loading = true;
    void music.loadArtists().finally(() => {
      loading = false;
    });
  });
</script>

<AppShell compactTop>
  <div class="artists-page">
    <PageHeader title="Artists" subtitle="Browse your library by artist." />

    <LocalSearchBox
      bind:value={searchQuery}
      placeholder="Search artists"
      disabled={unavailable || (loading && music.allArtists.length === 0)}
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
</AppShell>

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
