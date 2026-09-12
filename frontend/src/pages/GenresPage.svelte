<script lang="ts">
  import PageHeader from "$lib/components/ui/PageHeader.svelte";
  import EmptyState from "$lib/components/ui/EmptyState.svelte";
  import Skeleton from "$lib/components/ui/Skeleton.svelte";
  import GenreArt from "$lib/components/music/GenreArt.svelte";
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";
  import Link from "$lib/router/Link.svelte";
  import Button from "$lib/components/ui/Button.svelte";
  import LocalSearchBox from "$lib/components/ui/LocalSearchBox.svelte";
  import { music } from "$lib/config/music.svelte";
  import { filterByLocalSearch } from "$lib/utils/local-search";
  import { stableItemKey } from "$lib/core/collection";
  import GenreContextMenu from "$lib/components/music/GenreContextMenu.svelte";
  import LibraryUnavailable from "$lib/components/music/LibraryUnavailable.svelte";
  import { contextMenuPositionFromEvent } from "$lib/components/ui/context-menu";
  import { libraryUnavailable } from "$lib/music/library-gate";
  import { createAsyncPage } from "$lib/ui/async-page.svelte";

  const GENRE_BATCH = 60;

  let visibleCount = $state(GENRE_BATCH);
  let searchQuery = $state("");
  let genreMenu = $state<{ x: number; y: number; name: string } | null>(null);

  const unavailable = $derived(libraryUnavailable());

  const filteredGenres = $derived(
    filterByLocalSearch(music.genres, searchQuery, (genre) => [genre.name]),
  );

  const hasSearch = $derived(searchQuery.trim().length > 0);
  const visibleGenres = $derived(
    hasSearch ? filteredGenres : filteredGenres.slice(0, visibleCount),
  );
  const hasMoreGenres = $derived(
    !hasSearch && visibleCount < filteredGenres.length,
  );
  const page = createAsyncPage({
    load: (run) => {
      if (unavailable) return run.skip();
      if (!music.libraryReady) return run.wait(music.loading);
      if (music.genres.length > 0) return run.skip();
      return music.loadGenres();
    },
  });

  const showInitialLoading = $derived(
    !unavailable && page.loading && music.genres.length === 0 && !hasSearch,
  );

  $effect(() => {
    if (!hasSearch && music.genres.length >= 0) {
      visibleCount = GENRE_BATCH;
    }
  });

  function onGenreContextMenu(event: MouseEvent, name: string) {
    const pos = contextMenuPositionFromEvent(event);
    if (!pos) return;
    genreMenu = { ...pos, name };
  }
</script>

<div class="genres-page">
  <PageHeader title="Genres" subtitle="Browse and play by genre." />

  <LocalSearchBox
    bind:value={searchQuery}
    placeholder="Search genres"
    disabled={unavailable || (page.loading && music.genres.length === 0)}
    resultCount={filteredGenres.length}
    totalCount={music.genres.length}
  />

  {#if showInitialLoading}
    <div class="genre-grid">
      {#each Array.from({ length: 12 }) as _, i (i)}
        <Skeleton class="genre-skeleton" />
      {/each}
    </div>
  {:else if unavailable}
    <LibraryUnavailable />
  {:else if music.genres.length === 0}
    <EmptyState
      title="No genres"
      message="Your server did not report any genres."
      icon="tag"
    />
  {:else if filteredGenres.length === 0}
    <EmptyState
      title="No matches"
      message={`No genres match "${searchQuery.trim()}".`}
      icon="search"
    />
  {:else}
    <div class="genre-grid">
      {#each visibleGenres as genre, index (stableItemKey(genre.name, index, "genre"))}
        <div
          class="genre-card"
          role="group"
          oncontextmenu={(event) => onGenreContextMenu(event, genre.name)}
        >
          <Link
            href="/music/genre/{encodeURIComponent(genre.name)}"
            class="genre-card__main"
          >
            <GenreArt name={genre.name} />
            <span class="genre-card__name" title={genre.name}>{genre.name}</span
            >
            {#if genre.songCount}
              <span class="genre-card__count"
                >{genre.songCount.toLocaleString()}</span
              >
            {/if}
          </Link>
          <button
            type="button"
            class="genre-card__queue"
            aria-label={`Add ${genre.name} to queue`}
            onclick={() => void music.addGenreToQueue(genre.name)}
          >
            <MdiIcon name="queueAdd" size={16} />
          </button>
          <button
            type="button"
            class="genre-card__play"
            aria-label={`Play ${genre.name}`}
            onclick={() => void music.playGenre(genre.name)}
          >
            <MdiIcon name="play" size={18} />
          </button>
        </div>
      {/each}
    </div>
    {#if hasMoreGenres}
      <div class="genres-page__more">
        <Button
          variant="ghost"
          onclick={() => {
            visibleCount = Math.min(
              visibleCount + GENRE_BATCH,
              filteredGenres.length,
            );
          }}
        >
          Show more ({filteredGenres.length - visibleCount} remaining)
        </Button>
      </div>
    {/if}
  {/if}
</div>

{#if genreMenu}
  <GenreContextMenu
    name={genreMenu.name}
    x={genreMenu.x}
    y={genreMenu.y}
    onclose={() => (genreMenu = null)}
  />
{/if}

<style>
  .genres-page {
    display: flex;
    flex-direction: column;
    gap: var(--jb-space-4);
  }

  .genre-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(17rem, 1fr));
    gap: var(--jb-space-3);
  }

  :global(.genre-skeleton) {
    height: 4.25rem;
    border-radius: var(--jb-radius-lg);
  }

  .genre-card {
    display: flex;
    align-items: center;
    border: 1px solid var(--jb-border);
    border-radius: var(--jb-radius-lg);
    background: var(--jb-surface);
    overflow: hidden;
    transition: border-color var(--jb-transition);
    content-visibility: auto;
    contain-intrinsic-size: auto 4.25rem;
  }

  .genre-card:hover {
    border-color: var(--jb-accent);
  }

  :global(a.genre-card__main) {
    display: flex;
    align-items: center;
    gap: var(--jb-space-2);
    flex: 1;
    min-width: 0;
    padding: var(--jb-space-3) var(--jb-space-4);
    color: var(--jb-text);
    text-decoration: none;
    font-weight: 600;
  }

  .genre-card__name {
    flex: 1;
    min-width: 0;
    overflow: hidden;
    display: -webkit-box;
    -webkit-line-clamp: 2;
    line-clamp: 2;
    -webkit-box-orient: vertical;
    white-space: normal;
    line-height: 1.25;
  }

  .genre-card__count {
    font-size: 0.75rem;
    color: var(--jb-text-muted);
    font-variant-numeric: tabular-nums;
  }

  .genre-card__play {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 2.75rem;
    align-self: stretch;
    border: none;
    border-left: 1px solid var(--jb-border);
    background: transparent;
    color: var(--jb-accent);
    cursor: pointer;
    transition: background var(--jb-transition);
  }

  .genre-card__queue {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 2.5rem;
    align-self: stretch;
    border: none;
    border-left: 1px solid var(--jb-border);
    background: transparent;
    color: var(--jb-text-muted);
    cursor: pointer;
    transition: background var(--jb-transition);
  }

  .genre-card__queue:hover {
    color: var(--jb-accent);
    background: var(--jb-accent-muted);
  }

  .genre-card__play:hover {
    background: var(--jb-accent-muted);
  }

  .genres-page__more {
    display: flex;
    justify-content: center;
    padding-top: var(--jb-space-2);
  }

  @media (max-width: 768px) {
    .genre-grid {
      grid-template-columns: 1fr;
    }
  }
</style>
