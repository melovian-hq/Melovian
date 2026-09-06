<script lang="ts">
  import CoverArt from "$lib/components/ui/CoverArt.svelte";
  import AmbientCoverBackdrop from "$lib/components/ui/AmbientCoverBackdrop.svelte";
  import FavoriteButton from "$lib/components/music/FavoriteButton.svelte";
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";
  import TrackMetaLinks from "$lib/components/music/TrackMetaLinks.svelte";
  import TrackContextMenu from "$lib/components/music/TrackContextMenu.svelte";
  import { contextMenuPositionFromEvent } from "$lib/components/ui/context-menu";
  import { music } from "$lib/config/music.svelte";
  import { layout } from "$lib/components/layout/layout.svelte";
  import { deviceSync } from "$lib/music/device-sync.svelte";
  import {
    trackCoverPaletteKey,
    trackCoverSeed,
  } from "$lib/music/cover-art-fallback";
  import { formatDuration, isInternetRadioTrack } from "$lib/subsonic";
  import {
    COVER_SIZE_NOW_PLAYING,
    COVER_SIZE_PLAYER,
  } from "$lib/music/cover-art-sizes";
  import { trackCoverArtUrl } from "$lib/music/cover-art-prefetch";
  import { decorateTrack } from "$lib/extensions/registry";
  import ProgressSeek from "./ProgressSeek.svelte";

  let trackMenu = $state<{ x: number; y: number } | null>(null);

  const track = $derived(music.currentTrack);
  const trackDecoration = $derived(track ? decorateTrack(track) : {});
  const liveStream = $derived(track ? isInternetRadioTrack(track) : false);
  const image = $derived(
    track
      ? trackCoverArtUrl(music.config, track, COVER_SIZE_NOW_PLAYING)
      : null,
  );
  const previewImage = $derived(
    track ? trackCoverArtUrl(music.config, track, COVER_SIZE_PLAYER) : null,
  );

  function onSeekSeconds(seconds: number) {
    if (deviceSync.dispatchTransport("seek", { positionSec: seconds })) return;
    music.seek(seconds);
  }

  function tvRoot(node: HTMLDivElement) {
    if (typeof node.requestFullscreen === "function") {
      void node.requestFullscreen().catch(() => {});
    }
    return () => {
      if (document.fullscreenElement) {
        void document.exitFullscreen();
      }
    };
  }

  async function leave() {
    layout.exitTvMode();
    if (document.fullscreenElement) {
      try {
        await document.exitFullscreen();
      } catch {
        /* already exited */
      }
    }
  }

  function onFullscreenChange() {
    if (!document.fullscreenElement && layout.tvMode) {
      layout.exitTvMode();
    }
  }

  function onKeydown(event: KeyboardEvent) {
    if (event.code !== "Escape") return;
    event.preventDefault();
    void leave();
  }
</script>

<svelte:document onfullscreenchange={onFullscreenChange} />
<svelte:window onkeydown={onKeydown} />

{#if track}
  <div class="tv-mode" {@attach tvRoot}>
    <div class="tv-mode__ambient" aria-hidden="true">
      <AmbientCoverBackdrop
        src={image ?? previewImage}
        seed={trackCoverSeed(track)}
        paletteKey={trackCoverPaletteKey(track)}
        opacity={0.82}
        blur={64}
        saturate={1.65}
        scale={1.5}
      />
      <div class="tv-mode__wash"></div>
      <div class="tv-mode__vignette"></div>
    </div>

    <button
      type="button"
      class="tv-mode__exit"
      onclick={() => void leave()}
      aria-label="Exit TV mode"
      title="Exit TV mode"
    >
      <MdiIcon name="fullscreenExit" size={22} />
    </button>

    <div class="tv-mode__stage">
      <div
        class="tv-mode__art"
        role="img"
        aria-label="Album artwork"
        oncontextmenu={(event) => {
          if (liveStream) return;
          const pos = contextMenuPositionFromEvent(event);
          if (!pos) return;
          trackMenu = pos;
        }}
      >
        <CoverArt
          src={image}
          previewSrc={previewImage}
          seed={trackCoverSeed(track)}
          paletteKey={trackCoverPaletteKey(track)}
          loading="eager"
          fetchpriority="high"
        />
      </div>

      <div class="tv-mode__meta">
        <h1 class="tv-mode__title">{track.title}</h1>
        <TrackMetaLinks {track} class="tv-mode__artist" />
        {#if track.year}
          <p class="tv-mode__year">{track.year}</p>
        {/if}
        {#if !liveStream}
          <FavoriteButton {track} size={24} class="tv-mode__fav" />
        {/if}
      </div>

      <div class="tv-mode__transport">
        {#if !liveStream}
          <div class="tv-mode__progress">
            <span>{formatDuration(Math.floor(music.currentTime))}</span>
            <ProgressSeek
              class="tv-mode__progress-seek"
              size="lg"
              currentTime={music.currentTime}
              duration={music.duration}
              progressPercent={music.smoothProgress}
              playing={music.playing}
              decoration={trackDecoration}
              onSeek={onSeekSeconds}
            />
            <span>{formatDuration(Math.floor(music.duration))}</span>
          </div>
        {:else}
          <p class="tv-mode__live">Live</p>
        {/if}

        <div class="tv-mode__controls">
          <button
            type="button"
            class="tv-mode__ctrl"
            onclick={() => {
              if (!deviceSync.dispatchTransport("prev")) music.previous();
            }}
            aria-label="Previous"
          >
            <MdiIcon name="skipBack" size={30} />
          </button>
          <button
            type="button"
            class="tv-mode__ctrl tv-mode__ctrl--play"
            onclick={() => {
              if (!deviceSync.dispatchTransport("toggle")) music.togglePlay();
            }}
            aria-label={music.playing ? "Pause" : "Play"}
          >
            {#if music.playing}
              <MdiIcon name="pause" size={36} />
            {:else}
              <MdiIcon name="play" size={36} />
            {/if}
          </button>
          <button
            type="button"
            class="tv-mode__ctrl"
            onclick={() => {
              if (!deviceSync.dispatchTransport("next")) music.next();
            }}
            aria-label="Next"
          >
            <MdiIcon name="skipForward" size={30} />
          </button>
        </div>
      </div>
    </div>

    <p class="tv-mode__hint">Esc to exit</p>
  </div>
{/if}

{#if trackMenu && track && !liveStream}
  <TrackContextMenu
    {track}
    x={trackMenu.x}
    y={trackMenu.y}
    onclose={() => (trackMenu = null)}
  />
{/if}

<style>
  .tv-mode {
    position: fixed;
    inset: 0;
    z-index: 200;
    display: flex;
    align-items: center;
    justify-content: center;
    background: #050505;
    color: white;
  }

  .tv-mode__ambient {
    position: absolute;
    inset: 0;
    overflow: hidden;
    pointer-events: none;
  }

  .tv-mode__wash {
    position: absolute;
    inset: 0;
    background: linear-gradient(
      to bottom,
      rgb(0 0 0 / 0.22) 0%,
      rgb(0 0 0 / 0.48) 48%,
      rgb(0 0 0 / 0.78) 100%
    );
  }

  .tv-mode__vignette {
    position: absolute;
    inset: 0;
    background: radial-gradient(
      ellipse 72% 68% at 50% 42%,
      transparent 28%,
      rgb(0 0 0 / 0.55) 100%
    );
  }

  .tv-mode__exit {
    position: absolute;
    top: max(1.5rem, env(safe-area-inset-top, 0px));
    right: max(1.5rem, env(safe-area-inset-right, 0px));
    z-index: 2;
    display: grid;
    place-items: center;
    width: 3rem;
    height: 3rem;
    border: 1px solid rgb(255 255 255 / 0.2);
    border-radius: var(--jb-radius-full);
    background: rgb(255 255 255 / 0.08);
    backdrop-filter: blur(12px);
    color: rgb(255 255 255 / 0.92);
    cursor: pointer;
  }

  .tv-mode__exit:hover {
    background: rgb(255 255 255 / 0.16);
  }

  .tv-mode__stage {
    position: relative;
    z-index: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: clamp(1.25rem, 3.5vh, 2.5rem);
    width: min(44rem, calc(100vw - 3rem));
    padding: clamp(2rem, 6vh, 3.5rem) 1.25rem 4rem;
  }

  .tv-mode__art {
    width: min(100%, 36rem, 70vh);
    aspect-ratio: 1;
    border-radius: calc(var(--jb-radius-xl) + 0.25rem);
    overflow: hidden;
    transform: scale(1.02);
    box-shadow:
      0 40px 100px rgb(0 0 0 / 0.65),
      0 12px 36px rgb(0 0 0 / 0.4),
      inset 0 0 0 1px rgb(255 255 255 / 0.08);
  }

  .tv-mode__art :global(.cover-art) {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }

  .tv-mode__meta {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 0.45rem;
    text-align: center;
    width: 100%;
  }

  .tv-mode__title {
    margin: 0;
    font-size: clamp(1.85rem, 4.5vw, 3rem);
    font-weight: 800;
    letter-spacing: -0.03em;
    line-height: 1.12;
    text-shadow: 0 2px 22px rgb(0 0 0 / 0.55);
  }

  :global(.tv-mode__artist.track-meta-links) {
    justify-content: center;
    font-size: clamp(1.05rem, 2.2vw, 1.25rem);
    color: rgb(255 255 255 / 0.84);
  }

  .tv-mode__year {
    margin: 0.2rem 0 0;
    font-size: 0.9375rem;
    color: rgb(255 255 255 / 0.55);
    font-variant-numeric: tabular-nums;
  }

  :global(.tv-mode__fav.favorite-btn) {
    width: 2.75rem;
    height: 2.75rem;
    margin-top: 0.25rem;
    color: rgb(255 255 255 / 0.9);
  }

  .tv-mode__transport {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 1.25rem;
    width: min(100%, 34rem);
  }

  .tv-mode__progress {
    display: grid;
    grid-template-columns: 3.5rem 1fr 3.5rem;
    align-items: center;
    gap: 0.85rem;
    width: 100%;
    font-size: 0.875rem;
    font-variant-numeric: tabular-nums;
    color: rgb(255 255 255 / 0.72);
  }

  :global(.tv-mode__progress-seek) {
    flex: 1;
    min-width: 0;
    background: rgb(255 255 255 / 0.22);
    border-radius: var(--jb-radius-full);
    --progress-fill: linear-gradient(
      90deg,
      rgb(255 255 255 / 0.95),
      rgb(255 255 255 / 0.7)
    );
  }

  .tv-mode__live {
    margin: 0;
    font-weight: 700;
    letter-spacing: 0.12em;
    text-transform: uppercase;
    font-size: 0.75rem;
    color: rgb(255 255 255 / 0.7);
  }

  .tv-mode__controls {
    display: flex;
    align-items: center;
    gap: 1.5rem;
  }

  .tv-mode__ctrl {
    display: grid;
    place-items: center;
    width: 3.25rem;
    height: 3.25rem;
    border: none;
    border-radius: var(--jb-radius-full);
    background: transparent;
    color: white;
    cursor: pointer;
  }

  .tv-mode__ctrl--play {
    width: 4.75rem;
    height: 4.75rem;
    border: 1px solid rgb(255 255 255 / 0.22);
    background: rgb(255 255 255 / 0.14);
    backdrop-filter: blur(16px);
    box-shadow: 0 10px 28px rgb(0 0 0 / 0.35);
  }

  .tv-mode__ctrl:hover {
    background: rgb(255 255 255 / 0.12);
  }

  .tv-mode__ctrl--play:hover {
    background: rgb(255 255 255 / 0.22);
  }

  .tv-mode__hint {
    position: absolute;
    bottom: max(1.25rem, env(safe-area-inset-bottom, 0px));
    left: 50%;
    z-index: 1;
    margin: 0;
    transform: translateX(-50%);
    font-size: 0.75rem;
    letter-spacing: 0.04em;
    color: rgb(255 255 255 / 0.42);
  }
</style>
