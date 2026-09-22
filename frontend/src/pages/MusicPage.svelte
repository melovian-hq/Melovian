<script lang="ts">
  import { createSubscriber } from "svelte/reactivity";
  import AlbumCard from "$lib/components/music/AlbumCard.svelte";
  import ArtistCard from "$lib/components/music/ArtistCard.svelte";
  import HomePlaylistCard from "$lib/components/music/HomePlaylistCard.svelte";
  import HomeShelf from "$lib/components/music/HomeShelf.svelte";
  import HomeShortcutGrid from "$lib/components/music/HomeShortcutGrid.svelte";
  import MixCard from "$lib/components/music/MixCard.svelte";
  import LibraryUnavailable from "$lib/components/music/LibraryUnavailable.svelte";
  import HomeTipsBanner from "$lib/components/home/HomeTipsBanner.svelte";
  import EmptyState from "$lib/components/ui/EmptyState.svelte";
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";
  import Skeleton from "$lib/components/ui/Skeleton.svelte";
  import Link from "$lib/router/Link.svelte";
  import { music } from "$lib/config/music.svelte";
  import { libraryUnavailable } from "$lib/music/library-gate";
  import { loadMixDisplay, subscribeMixDisplay } from "$lib/music/mix-display";
  import { fetchMetadataSummary } from "$lib/features/metadata-editor/api";
  import type { MetadataSummary } from "$lib/features/metadata-editor/types";
  import { extensionFeatures } from "$lib/extensions/features.svelte";
  import {
    becauseYouListened,
    buildHomeShortcuts,
    greetingNow,
    homeHasLibraryContent,
    homePlaylists,
    jumpBackInAlbums,
    takeUnusedAlbums,
    type HomeFeedInput,
  } from "$lib/music/home-feed";
  import { toast } from "$lib/ui/toast.svelte";
  import { homeHidden } from "$lib/music/home-hidden.svelte";
  import { instances } from "$lib/features/instances/store.svelte";
  import { localLibraries } from "$lib/features/local-libraries/store.svelte";
  import {
    rejectUnknownAlbums,
    rejectUnknownArtists,
  } from "$lib/music/unknown-metadata";

  const subscribeMixStyle = createSubscriber((update) => {
    return subscribeMixDisplay(() => update());
  });
  const mixDisplayStyle = $derived.by(() => {
    subscribeMixStyle();
    return loadMixDisplay();
  });
  const greeting = greetingNow();

  let metadataSummary = $state<MetadataSummary | null>(null);

  const unavailable = $derived(libraryUnavailable());

  const hideUnknown = $derived(music.hideUnknownMetadata);
  const recentAlbums = $derived(
    rejectUnknownAlbums(music.recentAlbums, hideUnknown),
  );
  const frequentAlbums = $derived(
    rejectUnknownAlbums(music.frequentAlbums, hideUnknown),
  );
  const recommendations = $derived(
    rejectUnknownAlbums(music.recommendations, hideUnknown),
  );
  const favoriteAlbums = $derived(
    rejectUnknownAlbums(music.favoriteAlbums, hideUnknown),
  );
  const favoriteArtists = $derived(
    rejectUnknownArtists(music.favoriteArtists, hideUnknown),
  );

  const feedInput = $derived.by((): HomeFeedInput => ({
    resumeTracks: music.resumeTracks,
    listenHistory: music.listenHistory,
    recentAlbums,
    frequentAlbums,
    recommendations,
    favoriteAlbums,
    favoriteArtists,
    favoriteTrackCount: music.favoriteTracks.length,
    personalMixes: music.personalMixes,
    playlists: music.playlists,
    serverPlaylists: music.serverPlaylists,
  }));

  const shortcuts = $derived(
    buildHomeShortcuts(feedInput).filter(
      (item) => !homeHidden.isShortcutHidden(item),
    ),
  );
  const jumpBackAlbums = $derived(
    homeHidden.filterAlbums(
      rejectUnknownAlbums(
        jumpBackInAlbums(music.resumeTracks, music.listenHistory),
        hideUnknown,
      ),
    ),
  );
  const because = $derived.by(() => {
    const raw = becauseYouListened(
      [...music.resumeTracks, ...music.listenHistory],
      [
        ...favoriteAlbums,
        ...frequentAlbums,
        ...recommendations,
        ...recentAlbums,
      ],
    );
    if (!raw) return null;
    const albums = homeHidden.filterAlbums(raw.albums);
    if (albums.length === 0) return null;
    return { artist: raw.artist, albums };
  });
  const playlists = $derived(
    homeHidden.filterPlaylists(
      homePlaylists(music.playlists, music.serverPlaylists),
    ),
  );
  const visibleMixes = $derived(homeHidden.filterMixes(music.personalMixes));
  const visibleArtists = $derived(homeHidden.filterArtists(favoriteArtists));
  const hasContent = $derived(homeHasLibraryContent(feedInput));
  const loading = $derived(
    !music.libraryReady ||
      music.loading ||
      (music.connected && music.homeFeedSettling && !hasContent),
  );
  const feedSettling = $derived(music.connected && music.homeFeedSettling);

  const laterShelves = $derived.by(() => {
    const used = new Set<string>([...homeHidden.albums]);
    for (const album of jumpBackAlbums) used.add(album.id);
    for (const album of because?.albums ?? []) used.add(album.id);
    return {
      favoriteAlbums: takeUnusedAlbums(favoriteAlbums, used),
      newAlbums: takeUnusedAlbums(recentAlbums, used),
      recommended: takeUnusedAlbums(recommendations, used),
      frequent: takeUnusedAlbums(frequentAlbums, used),
    };
  });

  $effect(() => {
    void instances.activeId;
    homeHidden.reload();
  });

  $effect(() => {
    const libraryId = localLibraries.active?.id;
    if (!libraryId || !extensionFeatures.metadata) {
      metadataSummary = null;
      return;
    }
    void fetchMetadataSummary()
      .then((summary) => {
        if (localLibraries.active?.id === libraryId) {
          metadataSummary = summary;
        }
      })
      .catch(() => {
        if (localLibraries.active?.id === libraryId) {
          metadataSummary = null;
        }
      });
  });

  async function regenerateMixes() {
    try {
      await music.regenerateMixes();
      toast.success("Mixes refreshed");
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to refresh mixes",
      );
    }
  }
</script>

<div class="music-home">
  <header class="home-head">
    <p class="home-head__eyebrow">{music.serverName}</p>
    <div class="home-head__row">
      <h1 class="home-head__title">{greeting}</h1>
    </div>
  </header>

  {#if extensionFeatures.metadata && metadataSummary && metadataSummary.any > 0}
    <div class="metadata-alert">
      <MdiIcon name="autoFix" size={18} />
      <p>
        {metadataSummary.any.toLocaleString()} local tracks need metadata attention.
      </p>
      <Link href="/music/metadata" class="metadata-alert__link">
        Open metadata editor
      </Link>
    </div>
  {/if}

  {#if hasContent}
    <HomeTipsBanner />
  {/if}

  {#if unavailable && !loading}
    <LibraryUnavailable />
  {:else if loading}
    <div
      class="home-loading"
      role="status"
      aria-busy="true"
      aria-label="Loading library"
    >
      <div class="skeleton-shortcuts">
        {#each Array.from({ length: 4 }) as _, i (i)}
          <Skeleton class="skeleton-shortcut" />
        {/each}
      </div>
      <div class="skeleton-shelf">
        {#each Array.from({ length: 6 }) as _, i (i)}
          <Skeleton class="skeleton-album" />
        {/each}
      </div>
    </div>
  {:else if !hasContent}
    <EmptyState
      title="Nothing to play yet"
      message="Your library is empty. Try another source or add more music."
      icon="music"
    >
      {#snippet actions()}
        <Link href="/settings/servers" class="home-empty-link">
          Source settings
        </Link>
      {/snippet}
    </EmptyState>
  {:else}
    {#if feedSettling}
      <div class="skeleton-shortcuts" aria-hidden="true">
        {#each Array.from({ length: 8 }) as _, i (i)}
          <Skeleton class="skeleton-shortcut" />
        {/each}
      </div>
      <div class="home-shelves-pending" aria-hidden="true">
        {@render pendingShelf()}
        {@render pendingShelf()}
      </div>
    {:else}
      <HomeShortcutGrid {shortcuts} />
      {#if jumpBackAlbums.length > 0}
        <HomeShelf title="Jump back in" href="/music/history">
          {#each jumpBackAlbums as album (album.id)}
            <AlbumCard {album} size="sm" hideable />
          {/each}
        </HomeShelf>
      {/if}

      {#if visibleMixes.length > 0}
        <HomeShelf title="Made for you">
          {#snippet action()}
            <button
              type="button"
              class="mix-refresh-btn"
              disabled={music.mixesRegenerating}
              onclick={() => void regenerateMixes()}
            >
              {music.mixesRegenerating ? "Refreshing..." : "Refresh mixes"}
            </button>
          {/snippet}
          {#each visibleMixes as mix (mix.id)}
            <MixCard {mix} displayStyle={mixDisplayStyle} hideable />
          {/each}
        </HomeShelf>
      {/if}

      {#if because}
        <HomeShelf title="Because you listened to {because.artist}">
          {#each because.albums as album (album.id)}
            <AlbumCard {album} size="sm" hideable />
          {/each}
        </HomeShelf>
      {/if}

      {#if playlists.length > 0}
        <HomeShelf title="Your playlists" href="/music/playlists">
          {#each playlists as playlist (playlist.id)}
            <HomePlaylistCard {playlist} hideable />
          {/each}
        </HomeShelf>
      {/if}

      {#if visibleArtists.length > 0}
        <HomeShelf title="Your favorite artists" href="/music/favorites">
          {#each visibleArtists.slice(0, 12) as artist (artist.id)}
            <ArtistCard {artist} size="sm" hideable />
          {/each}
        </HomeShelf>
      {/if}

      {#if laterShelves.favoriteAlbums.length > 0}
        <HomeShelf title="Favorite albums" href="/music/favorites">
          {#each laterShelves.favoriteAlbums as album (album.id)}
            <AlbumCard {album} size="sm" hideable />
          {/each}
        </HomeShelf>
      {/if}

      {#if laterShelves.newAlbums.length > 0}
        <HomeShelf title="New in your library" href="/music/albums">
          {#each laterShelves.newAlbums as album (album.id)}
            <AlbumCard {album} size="sm" hideable />
          {/each}
        </HomeShelf>
      {/if}

      {#if laterShelves.recommended.length > 0}
        <HomeShelf title="Recommended for you">
          {#each laterShelves.recommended as album (album.id)}
            <AlbumCard {album} size="sm" hideable />
          {/each}
        </HomeShelf>
      {/if}

      {#if laterShelves.frequent.length > 0}
        <HomeShelf title="Popular on your server">
          {#each laterShelves.frequent as album (album.id)}
            <AlbumCard {album} size="sm" hideable />
          {/each}
        </HomeShelf>
      {/if}
    {/if}
  {/if}
</div>

{#snippet pendingShelf()}
  <section class="home-shelf-pending" aria-hidden="true">
    <Skeleton class="skeleton-shelf-heading" />
    <div class="home-shelf-pending__items">
      {#each Array.from({ length: 6 }) as _, i (i)}
        <div class="home-shelf-pending__card">
          <Skeleton class="skeleton-album" />
          <Skeleton class="skeleton-line" />
          <Skeleton class="skeleton-line skeleton-line--short" />
        </div>
      {/each}
    </div>
  </section>
{/snippet}

<style>
  .music-home {
    display: flex;
    flex-direction: column;
    gap: var(--jb-space-8);
    min-width: 0;
    overflow-x: hidden;
  }

  .home-head {
    display: flex;
    flex-direction: column;
    gap: var(--jb-space-1);
  }

  .home-head__eyebrow {
    margin: 0;
    font-size: 0.8125rem;
    font-weight: 600;
    color: var(--jb-text-muted);
  }

  .home-head__title {
    margin: 0;
    font-size: clamp(1.75rem, 4vw, 2.5rem);
    font-weight: 800;
    letter-spacing: -0.03em;
  }

  .home-head__row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--jb-space-3);
    flex-wrap: wrap;
    /* Keep the Customize button clear of the fixed TopBar islands when
       this row scrolls under them. */
    padding-right: var(--jb-topbar-islands-inset, 0px);
  }

  .metadata-alert {
    display: flex;
    align-items: center;
    gap: var(--jb-space-3);
    flex-wrap: wrap;
    padding: var(--jb-space-3) var(--jb-space-4);
    border-radius: var(--jb-radius-md);
    border: 1px solid var(--jb-border);
    background: var(--jb-bg-muted);
    color: var(--jb-text);
    font-size: 0.875rem;
  }

  .metadata-alert p {
    margin: 0;
    flex: 1;
    min-width: 12rem;
  }

  :global(.metadata-alert__link) {
    font-weight: 600;
    color: var(--jb-accent);
    text-decoration: none;
  }

  :global(.metadata-alert__link:hover) {
    text-decoration: underline;
  }

  .mix-refresh-btn {
    padding: 0.375rem 0.875rem;
    border: none;
    background: transparent;
    color: var(--jb-text-muted);
    font-size: 0.8125rem;
    font-weight: 700;
    letter-spacing: 0.04em;
    text-transform: uppercase;
    cursor: pointer;
  }

  .mix-refresh-btn:hover:not(:disabled) {
    color: var(--jb-text);
  }

  .mix-refresh-btn:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }

  .home-loading {
    display: flex;
    flex-direction: column;
    gap: var(--jb-space-8);
    width: 100%;
    min-width: 0;
  }

  .skeleton-shortcuts {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 2px;
    overflow: hidden;
    border-radius: var(--jb-radius-md);
  }

  .skeleton-shelf {
    display: grid;
    grid-template-columns: repeat(
      auto-fill,
      minmax(min(100%, var(--jb-card-grid-min)), 1fr)
    );
    gap: var(--jb-space-4);
  }

  :global(.skeleton-shortcut) {
    height: 4rem;
    border-radius: 0;
  }

  :global(.skeleton-album) {
    min-width: 0;
    aspect-ratio: 1;
    border-radius: var(--jb-radius-lg);
  }

  /* Stand-in for shelves that arrive when the feed settles. Shelves mount
     atomically below the painted shortcuts so late personalization cannot
     shift visible content. */
  .home-shelves-pending {
    display: flex;
    flex-direction: column;
    gap: var(--jb-space-8);
  }

  .home-shelf-pending {
    display: flex;
    flex-direction: column;
    gap: var(--jb-space-3);
    min-width: 0;
    overflow-x: hidden;
  }

  .home-shelf-pending__items {
    display: grid;
    grid-template-columns: repeat(
      auto-fill,
      minmax(min(100%, var(--jb-card-grid-min)), 1fr)
    );
    gap: var(--jb-space-4);
    overflow: hidden;
  }

  .home-shelf-pending__card {
    display: grid;
    gap: var(--jb-space-2);
    align-content: start;
  }

  :global(.skeleton-line) {
    height: 0.8rem;
    border-radius: var(--jb-radius-sm);
  }

  :global(.skeleton-line--short) {
    width: 60%;
  }

  :global(.skeleton-shelf-heading) {
    height: 1.5rem;
    width: 11rem;
    border-radius: var(--jb-radius-sm);
  }

  :global(a.home-empty-link) {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    box-sizing: border-box;
    max-width: 100%;
    padding: 0.5rem 1rem;
    border-radius: var(--jb-radius-full);
    background: var(--jb-accent);
    color: var(--jb-accent-text);
    font-weight: 700;
    text-decoration: none;
  }

  @media (min-width: 900px) {
    .skeleton-shortcuts {
      grid-template-columns: repeat(4, minmax(0, 1fr));
    }
  }

  @media (max-width: 640px) {
    .music-home {
      gap: var(--jb-space-6);
    }
  }
</style>
