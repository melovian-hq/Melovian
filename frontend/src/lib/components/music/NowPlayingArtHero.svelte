<script lang="ts">
  import CoverArt from "$lib/components/ui/CoverArt.svelte";
  import TrackMetaLinks from "$lib/components/music/TrackMetaLinks.svelte";
  import TrackQualityBadge from "$lib/components/music/TrackQualityBadge.svelte";
  import FavoriteButton from "$lib/components/music/FavoriteButton.svelte";
  import ListenTogetherChip from "$lib/components/music/ListenTogetherChip.svelte";
  import {
    trackCoverPaletteKey,
    trackCoverSeed,
  } from "$lib/music/cover-art-fallback";
  import type { SubsonicSong } from "$lib/subsonic";

  interface Props {
    track: SubsonicSong;
    liveStream: boolean;
    image: string | null;
    previewImage: string | null;
    oncontextmenu?: (event: MouseEvent) => void;
    /** Extra actions under the favorite control (e.g. Watch video). */
    actions?: import("svelte").Snippet;
  }

  let {
    track,
    liveStream,
    image,
    previewImage,
    oncontextmenu,
    actions,
  }: Props = $props();
</script>

<section class="now-playing-art" aria-label="Current track" {oncontextmenu}>
  <div class="now-playing-art__frame">
    <CoverArt
      src={image}
      previewSrc={previewImage}
      seed={trackCoverSeed(track)}
      paletteKey={trackCoverPaletteKey(track)}
      loading="eager"
      fetchpriority="high"
    />
  </div>

  <div class="now-playing-art__meta">
    <h1 class="now-playing-art__title">{track.title}</h1>
    <TrackMetaLinks {track} class="now-playing-art__artist" />
    <ListenTogetherChip />

    <div class="now-playing-art__details">
      <TrackQualityBadge {track} size="md" />
      {#if track.year}
        <span class="now-playing-art__year">{track.year}</span>
      {/if}
    </div>

    {#if !liveStream}
      <div class="now-playing-art__actions">
        <FavoriteButton {track} size={22} class="now-playing-art__fav" />
        {#if actions}
          {@render actions()}
        {/if}
      </div>
    {/if}
  </div>
</section>

<style>
  .now-playing-art {
    position: relative;
    z-index: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: var(--jb-space-5);
    flex: 1 1 auto;
    min-height: 0;
    min-width: 0;
    height: 100%;
    overflow: hidden;
    container-type: size;
    padding: var(--jb-space-5) var(--jb-space-4) var(--jb-space-3);
  }

  .now-playing-art__frame {
    position: relative;
    width: min(100%, 22rem, calc(100cqh - 9rem));
    aspect-ratio: 1;
    border-radius: var(--jb-radius-xl);
    overflow: hidden;
    box-shadow: var(--jb-shadow-lg);
    background: var(--jb-bg-muted);
  }

  .now-playing-art__frame :global(.cover-art) {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }

  .now-playing-art__meta {
    width: min(100%, 26rem);
    text-align: center;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--jb-space-2);
  }

  .now-playing-art__title {
    margin: 0;
    font-size: clamp(1.35rem, 4.2vw, 2.1rem);
    font-weight: 800;
    letter-spacing: -0.02em;
    line-height: 1.2;
    color: rgb(255 255 255 / 0.96);
    text-shadow: 0 1px 12px rgb(0 0 0 / 0.45);
    overflow-wrap: anywhere;
  }

  :global(.now-playing-art__artist.track-meta-links) {
    justify-content: center;
    font-size: 0.9375rem;
    color: rgb(255 255 255 / 0.82);
  }

  .now-playing-art__actions {
    display: flex;
    align-items: center;
    justify-content: center;
    flex-wrap: wrap;
    gap: var(--jb-space-2);
  }

  :global(.now-playing-art__fav.favorite-btn) {
    width: 2.75rem;
    height: 2.75rem;
    color: rgb(255 255 255 / 0.88);
  }

  .now-playing-art__details {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: var(--jb-space-3);
    margin-top: var(--jb-space-1);
  }

  .now-playing-art__year {
    font-size: 0.8125rem;
    color: rgb(255 255 255 / 0.65);
    font-variant-numeric: tabular-nums;
  }

  @media (max-width: 960px) {
    .now-playing-art {
      container-type: normal;
      padding: var(--jb-space-3) var(--jb-space-3) var(--jb-space-2);
      gap: var(--jb-space-3);
    }

    .now-playing-art__frame {
      width: min(70vw, 16rem);
      max-height: min(40svh, 16rem);
      aspect-ratio: 1;
    }

    .now-playing-art__title {
      font-size: clamp(1.1rem, 4vw, 1.5rem);
    }
  }

  @media (max-width: 768px) {
    .now-playing-art__frame {
      width: min(70vw, 16rem);
      max-height: min(40svh, 16rem);
    }
  }

  @media (max-height: 700px) and (min-width: 961px) {
    .now-playing-art__frame {
      width: min(100%, 14rem, calc(100cqh - 8rem));
    }
  }
</style>
