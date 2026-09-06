<script lang="ts">
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";
  import CoverArt from "$lib/components/ui/CoverArt.svelte";
  import AmbientCoverBackdrop from "$lib/components/ui/AmbientCoverBackdrop.svelte";
  import AppShell from "$lib/components/layout/AppShell.svelte";
  import AlbumGrid from "$lib/components/music/AlbumGrid.svelte";
  import ArtistCard from "$lib/components/music/ArtistCard.svelte";
  import ArtistContextMenu from "$lib/components/music/ArtistContextMenu.svelte";
  import MusicBreadcrumbs from "$lib/components/music/MusicBreadcrumbs.svelte";
  import StarButton from "$lib/components/music/StarButton.svelte";
  import EmptyState from "$lib/components/ui/EmptyState.svelte";
  import Skeleton from "$lib/components/ui/Skeleton.svelte";
  import { contextMenuPositionFromEvent } from "$lib/components/ui/context-menu";
  import Link from "$lib/router/Link.svelte";
  import { music } from "$lib/config/music.svelte";
  import {
    artistCoverPaletteKey,
    artistCoverSeed,
    coverArtFallbackUrl,
  } from "$lib/music/cover-art-fallback";
  import { coverArtUrl, type SubsonicArtistInfo } from "$lib/subsonic";
  import { resolveMediaUrl } from "$lib/config/runtime";
  import {
    fetchArtistWithCache,
    invalidateArtistDetailCache,
  } from "$lib/subsonic/detail-cache";
  import { fetchArtistInfoWithCache } from "$lib/music/artist-info-cache";
  import {
    enhanceArtistArtwork,
    artworkNeedsEnhancement,
  } from "$lib/music/metadata-enhancement";
  import { extensionFeatures } from "$lib/extensions/features.svelte";
  import {
    hasServerArtistArt,
    resolveServerArtistArtUrl,
  } from "$lib/music/artist-artwork";
  import { stableItemKey } from "$lib/core/collection";
  import { APP_NAME } from "$lib/brand";
  import { setPageMeta } from "$lib/seo/meta";

  interface Props {
    artistId: string;
  }

  let { artistId }: Props = $props();

  let loading = $state(true);
  let infoLoading = $state(false);
  let error = $state<string | null>(null);
  let playingAll = $state(false);
  let bioExpanded = $state(false);
  let data = $state<Awaited<ReturnType<typeof music.library.getArtist>> | null>(
    null,
  );
  let info = $state<SubsonicArtistInfo | null>(null);
  let enhancedPortrait = $state<string | null>(null);
  let artistMenu = $state<{ x: number; y: number } | null>(null);

  $effect(() => {
    const id = artistId;
    const revision = music.libraryRevision;
    let cancelled = false;
    loading = true;
    infoLoading = true;
    error = null;
    info = null;
    data = null;
    enhancedPortrait = null;
    bioExpanded = false;
    artistMenu = null;
    if (revision > 0) invalidateArtistDetailCache(id);
    void (async () => {
      if (!music.libraryReady) {
        if (!cancelled) {
          error = music.error ?? "Not connected";
          loading = false;
          infoLoading = false;
        }
        return;
      }
      try {
        const result = await fetchArtistWithCache(
          music.library,
          id,
          (stale) => {
            if (!cancelled) {
              data = stale;
              loading = false;
            }
          },
        );
        if (!cancelled) {
          data = result;
          loading = false;
        }
      } catch (err) {
        if (!cancelled) {
          error = err instanceof Error ? err.message : "Failed to load artist";
          loading = false;
          infoLoading = false;
        }
        return;
      }

      if (cancelled || !data) return;

      const coverSrc = coverArtUrl(
        music.config,
        data.artist.coverArt ?? data.albums[0]?.coverArt,
        480,
      );

      const hasServerArtistImage = hasServerArtistArt(data.artist);
      const preferServer =
        music.metadataEnhancementSettings.preferServerArtistArt;

      if (
        extensionFeatures.metadata &&
        artworkNeedsEnhancement(coverSrc) &&
        !(preferServer && hasServerArtistImage)
      ) {
        void enhanceArtistArtwork(
          data.artist,
          music.metadataEnhancementSettings,
          { resolvedSrc: coverSrc },
        )
          .then((enhanced) => {
            if (!cancelled) enhancedPortrait = enhanced;
          })
          .catch(() => {});
      }

      try {
        const artistInfo = await fetchArtistInfoWithCache(
          music.library,
          id,
          (stale) => {
            if (!cancelled) {
              info = stale;
              infoLoading = false;
            }
          },
        );
        if (!cancelled) info = artistInfo;
      } finally {
        if (!cancelled) infoLoading = false;
      }
    })();
    return () => {
      cancelled = true;
    };
  });

  const coverArtFallback = $derived(
    data
      ? coverArtUrl(
          music.config,
          data.artist.coverArt ?? data.albums[0]?.coverArt,
          480,
        )
      : null,
  );

  const artistImageUrl = $derived(
    data?.artist.artistImageUrl?.trim()
      ? resolveServerArtistArtUrl(
          music.config,
          { artistImageUrl: data.artist.artistImageUrl },
          600,
          resolveMediaUrl,
        )
      : null,
  );

  const serverPortrait = $derived(artistImageUrl ?? coverArtFallback);

  const remotePortrait = $derived(
    info?.largeImageUrl ?? info?.mediumImageUrl ?? info?.smallImageUrl ?? null,
  );

  // Prefer library/server art. Remote Last.fm-style URLs often 404 and were
  // replacing working portraits with pixel placeholders.
  const preferServerArtistArt = $derived(
    music.metadataEnhancementSettings.preferServerArtistArt,
  );
  const portraitUrl = $derived(
    preferServerArtistArt
      ? (serverPortrait ?? enhancedPortrait)
      : (serverPortrait ?? enhancedPortrait ?? remotePortrait),
  );

  const hasArtistPhoto = $derived(
    Boolean(
      artistImageUrl ??
      enhancedPortrait ??
      (!preferServerArtistArt ? remotePortrait : null),
    ),
  );

  const biography = $derived(info?.biography?.trim() ?? "");
  const showBioToggle = $derived(biography.length > 480);
  const visibleBio = $derived(
    bioExpanded || !showBioToggle
      ? biography
      : `${biography.slice(0, 480).trim()}…`,
  );

  const breadcrumbItems = $derived(
    data ? [{ label: data.artist.name }] : [{ label: "Artist" }],
  );

  $effect(() => {
    if (!data) return;
    const name = data.artist.name?.trim() || "Artist";
    const albumCount = data.albums.length;
    setPageMeta({
      title: name,
      description:
        albumCount > 0
          ? `${name} · ${albumCount} album${albumCount === 1 ? "" : "s"} in ${APP_NAME}.`
          : `Browse ${name} in ${APP_NAME}.`,
      image: portraitUrl ?? undefined,
      type: "profile",
    });
  });

  function onArtistHeroContextMenu(event: MouseEvent) {
    const pos = contextMenuPositionFromEvent(event);
    if (pos) artistMenu = pos;
  }

  async function playAllAlbums(shuffle = false) {
    if (!data || data.albums.length === 0 || playingAll) return;
    playingAll = true;
    try {
      await music.playArtistAlbums(data.albums, shuffle);
    } finally {
      playingAll = false;
    }
  }
</script>

{#if artistMenu && data}
  <ArtistContextMenu
    artist={data.artist}
    x={artistMenu.x}
    y={artistMenu.y}
    onclose={() => (artistMenu = null)}
  />
{/if}

<AppShell fill>
  <div class="artist-page">
    {#if data}
      <div class="artist-page__ambient" aria-hidden="true">
        <AmbientCoverBackdrop
          src={portraitUrl}
          seed={artistCoverSeed(data.artist)}
          paletteKey={artistCoverPaletteKey(data.artist)}
          opacity={0.72}
          blur={56}
          saturate={1.55}
          scale={1.45}
        />
        <div class="artist-page__wash"></div>
      </div>
    {/if}

    <div class="artist-page__body">
      <MusicBreadcrumbs items={breadcrumbItems} />

      {#if loading}
        <div
          class="artist-page__skeleton"
          role="status"
          aria-busy="true"
          aria-label="Loading artist"
        >
          <Skeleton variant="hero" class="artist-page__skeleton-hero" />
          <div class="artist-page__skeleton-cards">
            {#each Array.from({ length: 8 }) as _, i (i)}
              <Skeleton variant="card" />
            {/each}
          </div>
        </div>
      {:else if error || !data}
        <EmptyState
          title="Could not load artist"
          message={error ?? "Artist not found"}
          icon="alertCircle"
        >
          {#snippet actions()}
            <Link href="/music" class="artist-page__back">Back to music</Link>
          {/snippet}
        </EmptyState>
      {:else}
        <header
          class="artist-hero"
          role="group"
          oncontextmenu={onArtistHeroContextMenu}
        >
          <div class="artist-hero__layout">
            <div
              class="artist-hero__portrait-wrap"
              class:artist-hero__portrait-wrap--photo={hasArtistPhoto}
            >
              <CoverArt
                src={portraitUrl}
                seed={artistCoverSeed(data.artist)}
                paletteKey={artistCoverPaletteKey(data.artist)}
                alt={data.artist.name}
                fetchpriority="high"
                loading="eager"
              />
            </div>

            <div class="artist-hero__info">
              <p class="artist-hero__type">Artist</p>
              <div class="artist-hero__title-row">
                <h1>{data.artist.name}</h1>
                <StarButton
                  kind="artist"
                  artist={data.artist}
                  size={22}
                  onDark
                />
              </div>
              <p class="artist-hero__meta">{data.albums.length} albums</p>
              {#if data.albums.length > 0}
                <div class="artist-hero__actions">
                  <button
                    type="button"
                    class="artist-hero__play"
                    disabled={playingAll}
                    onclick={() => void playAllAlbums(false)}
                  >
                    <MdiIcon name="play" size={18} />
                    {playingAll ? "Loading..." : "Play all"}
                  </button>
                  <button
                    type="button"
                    class="artist-hero__shuffle"
                    disabled={playingAll}
                    onclick={() => void playAllAlbums(true)}
                  >
                    <MdiIcon name="shuffle" size={16} />
                    Shuffle
                  </button>
                </div>
              {/if}
            </div>
          </div>

          {#if biography}
            <div class="artist-hero__bio">
              <p class="artist-hero__bio-text">{visibleBio}</p>
              <div class="artist-hero__bio-footer">
                {#if showBioToggle}
                  <button
                    type="button"
                    class="artist-hero__bio-toggle"
                    onclick={() => (bioExpanded = !bioExpanded)}
                  >
                    {bioExpanded ? "Show less" : "Read more"}
                  </button>
                {/if}
                {#if info?.lastFmUrl}
                  <a
                    href={info.lastFmUrl}
                    class="artist-hero__bio-link"
                    target="_blank"
                    rel="noreferrer"
                  >
                    View on Last.fm
                  </a>
                {/if}
              </div>
            </div>
          {/if}
        </header>

        {#if data.albums.length > 0}
          <section class="artist-section">
            <h2 class="artist-page__heading">Albums</h2>
            <AlbumGrid albums={data.albums} size="lg" />
          </section>
        {:else}
          <EmptyState
            title="No albums"
            message="No albums were returned for this artist."
            icon="album"
          />
        {/if}

        {#if infoLoading}
          <section class="artist-section">
            <h2 class="artist-page__heading">Similar artists</h2>
            <div
              class="similar-grid similar-grid--loading"
              role="status"
              aria-busy="true"
              aria-label="Loading similar artists"
            >
              {#each Array.from({ length: 6 }) as _, i (i)}
                <Skeleton variant="avatar" class="artist-similar-skeleton" />
              {/each}
            </div>
          </section>
        {:else if info && info.similarArtists.length > 0}
          <section class="artist-section">
            <h2 class="artist-page__heading">Similar artists</h2>
            <div class="similar-grid">
              {#each info.similarArtists as similar, index (stableItemKey(similar.id, index, similar.name))}
                {#if similar.id}
                  <ArtistCard
                    artist={{
                      id: similar.id,
                      name: similar.name,
                      coverArt: similar.coverArt,
                    }}
                    size="sm"
                  />
                {:else}
                  <div class="similar-card similar-card--static">
                    <div class="similar-card__art">
                      <img
                        src={coverArtFallbackUrl(similar.name, similar.name)}
                        alt=""
                        class="similar-card__art--pixelated"
                      />
                    </div>
                    <span class="similar-card__name">{similar.name}</span>
                  </div>
                {/if}
              {/each}
            </div>
          </section>
        {/if}
      {/if}
    </div>
  </div>
</AppShell>

<style>
  .artist-page {
    position: relative;
    isolation: isolate;
    flex: 1 1 auto;
    min-height: 0;
    min-width: 0;
    overflow: hidden;
    display: flex;
    flex-direction: column;
  }

  .artist-page__ambient {
    position: absolute;
    inset: 0;
    z-index: 0;
    pointer-events: none;
    overflow: hidden;
  }

  .artist-page__wash {
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

  .artist-page__body {
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

  .artist-page__body :global(.music-breadcrumbs) {
    color: rgb(255 255 255 / 0.72);
    margin-bottom: 0;
    flex-shrink: 0;
  }

  .artist-page__body :global(.music-breadcrumbs__current) {
    color: rgb(255 255 255 / 0.92);
  }

  .artist-page__skeleton {
    display: flex;
    flex-direction: column;
    gap: var(--jb-space-6);
  }

  :global(.artist-page__skeleton-hero) {
    min-height: 10rem;
  }

  .artist-page__skeleton-cards {
    display: grid;
    grid-template-columns: repeat(
      auto-fill,
      minmax(min(100%, var(--jb-card-grid-min)), 1fr)
    );
    gap: var(--jb-space-5);
  }

  :global(a.artist-page__back) {
    color: var(--jb-music-accent);
    font-weight: 600;
    text-decoration: none;
  }

  .artist-hero {
    position: relative;
    flex-shrink: 0;
    min-height: 18rem;
  }

  .artist-hero__layout {
    position: relative;
    display: flex;
    flex-wrap: wrap;
    align-items: flex-end;
    gap: var(--jb-space-8);
    padding: clamp(0.5rem, 2vw, 1rem) 0 var(--jb-space-4);
    min-height: 18rem;
  }

  .artist-hero__portrait-wrap {
    flex-shrink: 0;
    width: clamp(9rem, 20vw, 13rem);
    height: clamp(9rem, 20vw, 13rem);
    border-radius: var(--jb-radius-lg);
    overflow: hidden;
    box-shadow:
      0 24px 60px rgb(0 0 0 / 0.55),
      0 0 0 3px rgb(255 255 255 / 0.12);
  }

  .artist-hero__portrait-wrap--photo {
    border-radius: 50%;
  }

  .artist-hero__portrait-wrap :global(.cover-art) {
    width: 100%;
    height: 100%;
  }

  .artist-hero__info {
    flex: 1;
    min-width: min(100%, 16rem);
    color: white;
    padding-bottom: var(--jb-space-2);
  }

  .artist-hero__type {
    margin: 0 0 var(--jb-space-2);
    font-size: 0.75rem;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.12em;
    opacity: 0.75;
  }

  .artist-hero__title-row {
    display: flex;
    align-items: center;
    gap: var(--jb-space-3);
    margin-bottom: var(--jb-space-3);
  }

  .artist-hero h1 {
    margin: 0;
    font-size: clamp(2rem, 5vw, 3.25rem);
    font-weight: 900;
    line-height: 1.05;
    letter-spacing: -0.02em;
    text-shadow: 0 2px 18px rgb(0 0 0 / 0.45);
  }

  .artist-hero__meta {
    margin: 0;
    font-size: 0.9375rem;
    opacity: 0.9;
  }

  .artist-hero__actions {
    display: flex;
    flex-wrap: wrap;
    gap: var(--jb-space-3);
    margin-top: var(--jb-space-5);
  }

  .artist-hero__play,
  .artist-hero__shuffle {
    display: inline-flex;
    align-items: center;
    gap: var(--jb-space-2);
    padding: 0.5rem 1rem;
    border-radius: var(--jb-radius-full);
    font-weight: 700;
    font-size: 0.875rem;
    cursor: pointer;
  }

  .artist-hero__play {
    border: none;
    background: var(--jb-accent);
    color: white;
  }

  .artist-hero__play:disabled,
  .artist-hero__shuffle:disabled {
    opacity: 0.6;
    cursor: default;
  }

  .artist-hero__shuffle {
    border: 1px solid rgb(255 255 255 / 0.25);
    background: rgb(255 255 255 / 0.08);
    color: white;
    backdrop-filter: blur(8px);
  }

  .artist-hero__shuffle:hover:not(:disabled) {
    background: rgb(255 255 255 / 0.15);
  }

  .artist-hero__bio {
    margin-top: var(--jb-space-6);
    padding-top: var(--jb-space-6);
    border-top: 1px solid rgb(255 255 255 / 0.12);
  }

  .artist-hero__bio-text {
    margin: 0;
    line-height: 1.7;
    color: rgb(255 255 255 / 0.78);
    white-space: pre-wrap;
    font-size: 0.9375rem;
  }

  .artist-hero__bio-footer {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--jb-space-4);
    margin-top: var(--jb-space-4);
  }

  .artist-hero__bio-toggle,
  .artist-hero__bio-link {
    border: none;
    background: transparent;
    color: var(--jb-accent);
    font-weight: 600;
    font-size: 0.875rem;
    cursor: pointer;
    text-decoration: none;
  }

  .artist-hero__bio-link:hover {
    text-decoration: underline;
  }

  .artist-section {
    flex-shrink: 0;
    margin-bottom: var(--jb-space-2);
  }

  .artist-page__heading {
    margin: 0 0 var(--jb-space-5);
    font-size: 1.25rem;
    font-weight: 700;
    color: rgb(255 255 255 / 0.92);
    text-shadow: 0 1px 10px rgb(0 0 0 / 0.35);
  }

  .similar-grid {
    display: grid;
    grid-template-columns: repeat(
      auto-fill,
      minmax(min(100%, var(--jb-artist-grid-min)), 1fr)
    );
    gap: var(--jb-space-4);
  }

  .similar-grid--loading {
    display: grid;
    grid-template-columns: repeat(
      auto-fill,
      minmax(min(100%, var(--jb-artist-grid-min)), 1fr)
    );
    gap: var(--jb-space-4);
    justify-items: center;
  }

  :global(.artist-similar-skeleton) {
    width: 100%;
    max-width: 7rem;
    aspect-ratio: 1;
    height: auto;
    border-radius: 50%;
  }

  .similar-card--static {
    display: flex;
    flex-direction: column;
    gap: var(--jb-space-3);
    min-width: 0;
    opacity: 0.75;
  }

  .similar-card--static .similar-card__art {
    aspect-ratio: 1;
    width: 100%;
    border-radius: 50%;
    overflow: hidden;
    background: var(--jb-bg-muted);
    box-shadow: var(--jb-shadow-md);
  }

  .similar-card--static .similar-card__art img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }

  .similar-card__art--pixelated {
    image-rendering: pixelated;
  }

  .similar-card__name {
    font-size: 0.875rem;
    font-weight: 650;
    line-height: 1.3;
    text-align: center;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    color: rgb(255 255 255 / 0.88);
  }

  @media (max-width: 640px) {
    .artist-page__body {
      padding-left: var(--jb-space-4);
      padding-right: var(--jb-space-4);
    }

    .artist-hero__layout {
      flex-direction: column;
      align-items: center;
      text-align: center;
      min-height: auto;
      padding: var(--jb-space-4) 0;
    }

    .artist-hero {
      min-height: auto;
    }

    .artist-hero__actions {
      justify-content: center;
      align-items: center;
    }

    .artist-hero__play,
    .artist-hero__shuffle {
      padding: 0.375rem 0.75rem;
      font-size: 0.8125rem;
      white-space: nowrap;
    }

    .artist-hero__bio-footer {
      justify-content: center;
    }

    .similar-grid {
      grid-template-columns: repeat(2, minmax(0, 1fr));
      gap: var(--jb-space-3);
    }
  }
</style>
