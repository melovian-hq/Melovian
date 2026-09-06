<script lang="ts">
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";
  import EnhancedCoverArt from "$lib/components/ui/EnhancedCoverArt.svelte";
  import AmbientCoverBackdrop from "$lib/components/ui/AmbientCoverBackdrop.svelte";
  import AppShell from "$lib/components/layout/AppShell.svelte";
  import AlbumContextMenu from "$lib/components/music/AlbumContextMenu.svelte";
  import MusicBreadcrumbs, {
    type MusicBreadcrumb,
  } from "$lib/components/music/MusicBreadcrumbs.svelte";
  import TrackVirtualList from "$lib/components/music/TrackVirtualList.svelte";
  import TrackSelectionBar from "$lib/components/music/TrackSelectionBar.svelte";
  import Button from "$lib/components/ui/Button.svelte";
  import StarButton from "$lib/components/music/StarButton.svelte";
  import EmptyState from "$lib/components/ui/EmptyState.svelte";
  import Skeleton from "$lib/components/ui/Skeleton.svelte";
  import { contextMenuPositionFromEvent } from "$lib/components/ui/context-menu";
  import Link from "$lib/router/Link.svelte";
  import { music } from "$lib/config/music.svelte";
  import { trackSelection } from "$lib/music/selection.svelte";
  import {
    albumCoverPaletteKey,
    albumCoverSeed,
  } from "$lib/music/cover-art-fallback";
  import { coverArtUrl, formatDuration } from "$lib/subsonic";
  import {
    fetchAlbumWithCache,
    invalidateAlbumDetailCache,
  } from "$lib/subsonic/detail-cache";
  import {
    albumPlaybackTracks,
    groupTrackVersions,
  } from "$lib/music/track-versions";
  import type { SubsonicSong } from "$lib/subsonic";

  interface Props {
    albumId: string;
  }

  let { albumId }: Props = $props();

  let loading = $state(true);
  let error = $state<string | null>(null);
  let album = $state<Awaited<ReturnType<typeof music.library.getAlbum>> | null>(
    null,
  );
  let albumMenu = $state<{ x: number; y: number } | null>(null);

  $effect(() => {
    const id = albumId;
    const revision = music.libraryRevision;
    let cancelled = false;
    loading = true;
    error = null;
    album = null;
    albumMenu = null;
    if (revision > 0) invalidateAlbumDetailCache(id);
    void (async () => {
      if (!music.libraryReady) {
        if (!cancelled) {
          error = music.error ?? "Not connected";
          loading = false;
        }
        return;
      }
      try {
        const result = await fetchAlbumWithCache(music.library, id, (stale) => {
          if (!cancelled) {
            album = stale;
            loading = false;
          }
        });
        if (!cancelled) album = result;
      } catch (err) {
        if (!cancelled) {
          error = err instanceof Error ? err.message : "Failed to load album";
        }
      } finally {
        if (!cancelled) loading = false;
      }
    })();
    return () => {
      cancelled = true;
    };
  });

  const coverImage = $derived(
    album
      ? coverArtUrl(music.config, album.album.coverArt ?? album.album.id, 800)
      : null,
  );
  const ambientCoverImage = $derived(
    album
      ? coverArtUrl(music.config, album.album.coverArt ?? album.album.id, 256)
      : null,
  );

  const trackGroups = $derived(album ? groupTrackVersions(album.songs) : []);

  const displayTracks = $derived(trackGroups.map((group) => group.primary));

  const playbackTracks = $derived(
    album ? albumPlaybackTracks(album.songs) : [],
  );

  const totalDuration = $derived(
    playbackTracks.reduce((sum, song) => sum + (song.duration ?? 0), 0),
  );

  const versionCount = $derived(
    album ? album.songs.length - trackGroups.length : 0,
  );

  function playAll(shuffle = false) {
    if (!album) return;
    if (shuffle) music.shuffle = true;
    music.playAlbum(playbackTracks, 0);
  }

  function addAlbumToQueue() {
    if (!album || playbackTracks.length === 0) return;
    music.addTracksToQueue(playbackTracks);
  }

  function playAlbumNext() {
    if (!album || playbackTracks.length === 0) return;
    music.playTracksNext(playbackTracks);
  }

  function playGroup(index: number) {
    if (!album) return;
    music.playAlbum(playbackTracks, index);
  }

  function playVersion(track: SubsonicSong) {
    if (!album) return;
    const groupIndex = trackGroups.findIndex((group) =>
      group.versions.some((version) => version.id === track.id),
    );
    if (groupIndex < 0) return;
    const queue = playbackTracks.map((item, index) =>
      index === groupIndex ? track : item,
    );
    music.playAlbum(queue, groupIndex);
  }

  let savingFiles = $state(false);

  async function downloadFiles() {
    if (!album || savingFiles) return;
    savingFiles = true;
    try {
      const { saveAlbumFilesToDevice } =
        await import("$lib/music/context-actions");
      await saveAlbumFilesToDevice(album.album.id);
    } finally {
      savingFiles = false;
    }
  }

  const breadcrumbItems = $derived.by(() => {
    if (!album) return [{ label: "Album" }];
    const items: MusicBreadcrumb[] = [];
    if (album.album.artistId && album.album.artist) {
      items.push({
        label: album.album.artist,
        href: `/music/artist/${album.album.artistId}`,
      });
    }
    items.push({ label: album.album.name });
    return items;
  });

  function onAlbumHeroContextMenu(event: MouseEvent) {
    const pos = contextMenuPositionFromEvent(event);
    if (pos) albumMenu = pos;
  }
</script>

{#if albumMenu && album}
  <AlbumContextMenu
    album={album.album}
    x={albumMenu.x}
    y={albumMenu.y}
    onclose={() => (albumMenu = null)}
  />
{/if}

<AppShell fill>
  <div class="album-page">
    {#if album}
      <div class="album-page__ambient" aria-hidden="true">
        <AmbientCoverBackdrop
          src={ambientCoverImage}
          seed={albumCoverSeed(album.album)}
          paletteKey={albumCoverPaletteKey(album.album)}
          opacity={0.72}
          blur={56}
          saturate={1.55}
          scale={1.45}
        />
        <div class="album-page__wash"></div>
      </div>
    {/if}

    <div class="album-page__body">
      <MusicBreadcrumbs items={breadcrumbItems} />

      {#if loading}
        <div
          class="album-page__skeleton"
          role="status"
          aria-busy="true"
          aria-label="Loading album"
        >
          <Skeleton variant="hero" class="album-page__skeleton-hero" />
          <div class="album-page__skeleton-rows">
            {#each Array.from({ length: 8 }) as _, i (i)}
              <Skeleton variant="row" />
            {/each}
          </div>
        </div>
      {:else if error || !album}
        <EmptyState
          title="Could not load album"
          message={error ?? "Album not found"}
          icon="alertCircle"
        >
          {#snippet actions()}
            <Link href="/music" class="album-page__back">Back to music</Link>
          {/snippet}
        </EmptyState>
      {:else}
        <header
          class="album-hero"
          role="group"
          oncontextmenu={onAlbumHeroContextMenu}
        >
          <div class="album-hero__layout">
            <div class="album-hero__cover-wrap">
              <EnhancedCoverArt
                kind="album"
                entity={{
                  id: album.album.id,
                  name: album.album.name,
                  artist: album.album.artist,
                  coverArt: album.album.coverArt,
                }}
                src={coverImage}
                seed={albumCoverSeed(album.album)}
                paletteKey={albumCoverPaletteKey(album.album)}
                alt={album.album.name}
                fetchpriority="high"
                loading="eager"
              />
            </div>

            <div class="album-hero__info">
              <p class="album-hero__type">Album</p>
              <div class="album-hero__title-row">
                <h1>{album.album.name}</h1>
                <StarButton kind="album" album={album.album} size={22} onDark />
              </div>
              <p class="album-hero__meta">
                {#if album.album.artistId}
                  <Link href="/music/artist/{album.album.artistId}"
                    >{album.album.artist}</Link
                  >
                {:else}
                  {album.album.artist ?? "Unknown artist"}
                {/if}
                {#if album.album.year}
                  <span class="album-hero__dot">·</span> {album.album.year}
                {/if}
                <span class="album-hero__dot">·</span>
                {trackGroups.length} tracks
                {#if versionCount > 0}
                  <span class="album-hero__dot">·</span>
                  {versionCount} alternate versions
                {/if}
                <span class="album-hero__dot">·</span>
                {formatDuration(totalDuration)}
              </p>
              {#if album.album.genre}
                <p class="album-hero__genre">{album.album.genre}</p>
              {/if}

              <div class="album-hero__actions">
                <Button size="sm" onclick={() => playAll(false)}>
                  <MdiIcon name="play" size={18} />
                  Play
                </Button>
                <button
                  type="button"
                  class="album-hero__secondary"
                  onclick={() => playAll(true)}
                >
                  <MdiIcon name="shuffle" size={16} />
                  Shuffle
                </button>
                <button
                  type="button"
                  class="album-hero__secondary"
                  onclick={playAlbumNext}
                >
                  <MdiIcon name="playNext" size={16} />
                  Play next
                </button>
                <button
                  type="button"
                  class="album-hero__secondary"
                  onclick={addAlbumToQueue}
                >
                  <MdiIcon name="queueAdd" size={16} />
                  Add to queue
                </button>
                <button
                  type="button"
                  class="album-hero__secondary"
                  onclick={() => trackSelection.enable()}
                >
                  <MdiIcon name="squareCheck" size={16} />
                  Select
                </button>
                <button
                  type="button"
                  class="album-hero__secondary"
                  onclick={() => void downloadFiles()}
                  disabled={savingFiles || !album?.songs?.length}
                >
                  <MdiIcon name="download" size={16} />
                  {savingFiles ? "Downloading…" : "Download"}
                </button>
              </div>
            </div>
          </div>
        </header>

        <TrackSelectionBar allTracks={playbackTracks} />

        {#if displayTracks.length === 0}
          <EmptyState
            title="No tracks"
            message="This album has no songs to play."
            icon="album"
          />
        {:else}
          <div class="album-tracks">
            <div class="album-tracks__head">
              <span>#</span>
              <span>Title</span>
              <span></span>
              <span>Duration</span>
            </div>
            <TrackVirtualList
              tracks={displayTracks}
              versionsFor={(index) => trackGroups[index]?.versions}
              onplay={playGroup}
              onplayversion={playVersion}
              selectable={trackSelection.active}
              lazyThreshold={12}
              itemHeight={62}
            />
          </div>
        {/if}
      {/if}
    </div>
  </div>
</AppShell>

<style>
  .album-page {
    position: relative;
    isolation: isolate;
    flex: 1 1 auto;
    min-height: 0;
    min-width: 0;
    overflow: hidden;
    display: flex;
    flex-direction: column;
  }

  .album-page__ambient {
    position: absolute;
    inset: 0;
    z-index: 0;
    pointer-events: none;
    overflow: hidden;
  }

  .album-page__wash {
    position: absolute;
    inset: 0;
    background:
      linear-gradient(
        to bottom,
        color-mix(in srgb, var(--jb-bg) 18%, transparent) 0%,
        color-mix(in srgb, var(--jb-bg) 42%, transparent) 42%,
        color-mix(in srgb, var(--jb-bg) 78%, transparent) 100%
      ),
      radial-gradient(
        120% 80% at 50% 0%,
        transparent 20%,
        color-mix(in srgb, var(--jb-bg) 55%, transparent) 100%
      );
  }

  .album-page__body {
    position: relative;
    z-index: 1;
    flex: 1 1 auto;
    min-height: 0;
    overflow-x: hidden;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
    gap: var(--jb-space-5);
    padding: var(
        --jb-fill-pad-top,
        calc(var(--jb-window-chrome-offset, 0px) + var(--jb-space-4))
      )
      var(--jb-space-6) var(--jb-space-8);
  }

  .album-page__body :global(.music-breadcrumbs) {
    color: rgb(255 255 255 / 0.72);
    margin-bottom: 0;
  }

  .album-page__body :global(.music-breadcrumbs__current) {
    color: rgb(255 255 255 / 0.92);
  }

  .album-page__skeleton {
    display: flex;
    flex-direction: column;
    gap: var(--jb-space-6);
  }

  :global(.album-page__skeleton-hero) {
    min-height: 10rem;
  }

  .album-page__skeleton-rows {
    display: flex;
    flex-direction: column;
    gap: var(--jb-space-2);
  }

  :global(a.album-page__back) {
    color: var(--jb-music-accent);
    font-weight: 600;
    text-decoration: none;
  }

  .album-hero {
    position: relative;
    flex-shrink: 0;
    min-height: 18rem;
  }

  .album-hero__layout {
    position: relative;
    display: flex;
    flex-wrap: wrap;
    align-items: flex-end;
    gap: var(--jb-space-8);
    padding: clamp(0.5rem, 2vw, 1rem) 0 var(--jb-space-4);
    min-height: 18rem;
  }

  .album-hero__cover-wrap {
    flex-shrink: 0;
    width: clamp(10rem, 22vw, 16rem);
    height: clamp(10rem, 22vw, 16rem);
    border-radius: var(--jb-radius-lg);
    overflow: hidden;
    box-shadow: 0 24px 60px rgb(0 0 0 / 0.55);
  }

  .album-hero__cover-wrap :global(.cover-art) {
    width: 100%;
    height: 100%;
  }

  .album-hero__info {
    flex: 1;
    min-width: min(100%, 16rem);
    color: white;
    padding-bottom: var(--jb-space-2);
  }

  .album-hero__type {
    margin: 0 0 var(--jb-space-2);
    font-size: 0.75rem;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.12em;
    opacity: 0.75;
  }

  .album-hero__title-row {
    display: flex;
    align-items: center;
    gap: var(--jb-space-3);
    margin-bottom: var(--jb-space-3);
  }

  .album-hero h1 {
    margin: 0;
    font-size: clamp(2rem, 5vw, 3.25rem);
    font-weight: 900;
    line-height: 1.05;
    letter-spacing: -0.02em;
    text-shadow: 0 2px 18px rgb(0 0 0 / 0.45);
  }

  .album-hero__meta {
    margin: 0 0 var(--jb-space-2);
    font-size: 0.9375rem;
    opacity: 0.9;
    line-height: 1.5;
  }

  .album-hero__meta :global(a) {
    color: white;
    font-weight: 700;
    text-decoration: none;
  }

  .album-hero__meta :global(a:hover) {
    text-decoration: underline;
  }

  .album-hero__dot {
    opacity: 0.5;
    margin-inline: 0.25rem;
  }

  .album-hero__genre {
    margin: 0 0 var(--jb-space-5);
    font-size: 0.8125rem;
    opacity: 0.7;
    text-transform: uppercase;
    letter-spacing: 0.06em;
  }

  .album-hero__actions {
    display: flex;
    flex-wrap: wrap;
    gap: var(--jb-space-3);
    align-items: center;
  }

  .album-hero__secondary {
    display: inline-flex;
    align-items: center;
    gap: var(--jb-space-2);
    padding: 0.5rem 1rem;
    border: 1px solid rgb(255 255 255 / 0.25);
    border-radius: var(--jb-radius-full);
    background: rgb(255 255 255 / 0.08);
    color: white;
    font-weight: 600;
    font-size: 0.875rem;
    cursor: pointer;
    backdrop-filter: blur(8px);
  }

  .album-hero__secondary:hover {
    background: rgb(255 255 255 / 0.15);
  }

  .album-hero__secondary:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .album-tracks {
    padding-bottom: var(--jb-space-2);
  }

  .album-tracks__head {
    display: grid;
    grid-template-columns: 2rem 1fr auto 4rem;
    gap: var(--jb-space-3);
    padding: var(--jb-space-3) 0;
    border-bottom: 1px solid rgb(255 255 255 / 0.12);
    font-size: 0.75rem;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    color: rgb(255 255 255 / 0.55);
  }

  @media (max-width: 640px) {
    .album-page__body {
      padding-left: var(--jb-space-4);
      padding-right: var(--jb-space-4);
    }

    .album-hero__layout {
      flex-direction: column;
      align-items: center;
      text-align: center;
      min-height: auto;
      padding: var(--jb-space-4) 0;
    }

    .album-hero {
      min-height: auto;
    }

    .album-hero__actions {
      justify-content: center;
      align-items: center;
    }

    .album-hero__secondary {
      padding: 0.375rem 0.75rem;
      font-size: 0.8125rem;
      white-space: nowrap;
    }

    .album-tracks__head {
      display: none;
    }
  }
</style>
