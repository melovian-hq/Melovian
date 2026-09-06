<script lang="ts">
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";
  import EnhancedCoverArt from "$lib/components/ui/EnhancedCoverArt.svelte";
  import ProgressSeek from "./ProgressSeek.svelte";
  import { music } from "$lib/config/music.svelte";
  import { deviceSync } from "$lib/music/device-sync.svelte";
  import {
    trackCoverPaletteKey,
    trackCoverSeed,
  } from "$lib/music/cover-art-fallback";
  import {
    coverArtUrl,
    formatDuration,
    isInternetRadioTrack,
  } from "$lib/subsonic";
  import Link from "$lib/router/Link.svelte";
  import { overlayFade } from "$lib/router/page-motion";
  import { fade } from "svelte/transition";

  const track = $derived(music.currentTrack);
  const liveStream = $derived(track ? isInternetRadioTrack(track) : false);
  const image = $derived(
    track
      ? coverArtUrl(
          music.config,
          track.coverArt ?? track.albumId ?? track.id,
          128,
        )
      : null,
  );
</script>

{#if track && music.playerLayout === "full" && music.playerVisible}
  <div class="slim-player" role="group" aria-label="Now playing">
    <ProgressSeek
      class="slim-player__progress"
      size="sm"
      currentTime={music.currentTime}
      duration={music.duration}
      progressPercent={music.smoothProgress}
      playing={music.playing}
      disabled={liveStream}
      onSeek={(seconds) => {
        if (deviceSync.dispatchTransport("seek", { positionSec: seconds }))
          return;
        music.seek(seconds);
      }}
    />

    <div class="slim-player__row">
      <Link
        href="/music/now-playing"
        class="slim-player__track"
        title="Now playing"
      >
        {#key track.id}
          <span class="slim-player__swap" in:fade={overlayFade()}>
            <span class="slim-player__art">
              <EnhancedCoverArt
                kind="track"
                entity={{
                  id: track.id,
                  title: track.title,
                  artist: track.artist,
                  album: track.album,
                  coverArt: track.coverArt,
                  albumId: track.albumId,
                }}
                src={image}
                seed={trackCoverSeed(track)}
                paletteKey={trackCoverPaletteKey(track)}
              />
            </span>
            <span class="slim-player__meta">
              <span class="slim-player__title">{track.title}</span>
              {#if track.artist}
                <span class="slim-player__artist">{track.artist}</span>
              {/if}
            </span>
          </span>
        {/key}
      </Link>

      <span class="slim-player__time">
        {#if liveStream}
          Live
        {:else}
          {formatDuration(Math.floor(music.currentTime))} / {formatDuration(
            Math.floor(music.duration),
          )}
        {/if}
      </span>

      <div class="slim-player__controls">
        <button
          type="button"
          class="slim-player__btn slim-player__btn--play"
          onclick={() => {
            if (!deviceSync.dispatchTransport("toggle")) music.togglePlay();
          }}
          aria-label={music.playing ? "Pause" : "Play"}
        >
          {#if music.playing}
            <MdiIcon name="pause" size={22} />
          {:else}
            <MdiIcon name="play" size={22} />
          {/if}
        </button>
        <button
          type="button"
          class="slim-player__btn"
          onclick={() => {
            if (!deviceSync.dispatchTransport("next")) music.next();
          }}
          aria-label="Next"
        >
          <MdiIcon name="skipForward" size={22} />
        </button>
      </div>
    </div>
  </div>
{/if}

{#if track && music.playerLayout === "dismissed" && !music.onPlayRoute}
  <button
    type="button"
    class="slim-player-fab"
    class:slim-player-fab--spinning={music.playing}
    onclick={() => music.restorePlayer()}
    aria-label="Show player"
  >
    {#if image}
      <img src={image} alt="" />
    {/if}
  </button>
{/if}

<style>
  .slim-player {
    position: fixed;
    left: 0;
    right: 0;
    bottom: calc(
      var(--jb-bottom-nav-height) + env(safe-area-inset-bottom, 0px)
    );
    z-index: 50;
    display: flex;
    flex-direction: column;
    align-items: stretch;
    gap: 0;
    height: var(--jb-slim-player-height);
    padding: 0;
    border-top: 1px solid var(--jb-border);
    background: color-mix(in srgb, var(--jb-bg-elevated) 96%, transparent);
    backdrop-filter: blur(20px);
    color: var(--jb-text);
  }

  .slim-player :global(.slim-player__progress) {
    flex-shrink: 0;
    border-radius: 0;
  }

  .slim-player__row {
    flex: 1;
    min-height: 0;
    display: flex;
    align-items: center;
    gap: var(--jb-space-2);
    padding: 0 var(--jb-space-2) 0 var(--jb-space-3);
  }

  .slim-player :global(.slim-player__track) {
    flex: 1;
    min-width: 0;
    display: flex;
    align-items: center;
    gap: var(--jb-space-3);
    text-decoration: none;
    color: inherit;
  }

  .slim-player__art {
    flex-shrink: 0;
    width: 2.25rem;
    height: 2.25rem;
    border-radius: var(--jb-radius-sm);
    overflow: hidden;
    background: var(--jb-bg-muted);
  }

  .slim-player__art :global(img),
  .slim-player__art :global(canvas),
  .slim-player__art :global(*) {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }

  .slim-player__meta {
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 0.1rem;
  }

  .slim-player__title {
    font-size: 0.8125rem;
    font-weight: 600;
    line-height: 1.2;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .slim-player__artist {
    font-size: 0.6875rem;
    color: var(--jb-text-muted);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .slim-player__time {
    flex-shrink: 0;
    font-size: 0.6875rem;
    color: var(--jb-text-subtle);
    font-variant-numeric: tabular-nums;
    white-space: nowrap;
  }

  .slim-player__controls {
    flex-shrink: 0;
    display: flex;
    align-items: center;
    gap: var(--jb-space-1);
  }

  .slim-player__swap {
    display: flex;
    align-items: center;
    gap: var(--jb-space-3);
    min-width: 0;
    flex: 1;
  }

  .slim-player__btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 2.5rem;
    height: 2.5rem;
    border: none;
    border-radius: var(--jb-radius-full);
    background: transparent;
    color: var(--jb-text);
    cursor: pointer;
    -webkit-tap-highlight-color: transparent;
    transition:
      background var(--jb-transition),
      color var(--jb-transition);
  }

  .slim-player__btn:hover,
  .slim-player__btn:active {
    background: var(--jb-surface-hover);
  }

  .slim-player__btn--play {
    background: var(--jb-accent);
    color: var(--jb-accent-text);
  }

  .slim-player__btn--play:hover,
  .slim-player__btn--play:active {
    background: var(--jb-accent-hover);
    color: var(--jb-accent-text);
  }

  .slim-player-fab {
    position: fixed;
    right: var(--jb-space-4);
    bottom: calc(
      var(--jb-bottom-nav-height) + env(safe-area-inset-bottom, 0px) +
        var(--jb-space-3)
    );
    z-index: 55;
    width: 3.25rem;
    height: 3.25rem;
    border: 2px solid var(--jb-border);
    border-radius: var(--jb-radius-full);
    padding: 0;
    overflow: hidden;
    cursor: pointer;
    box-shadow: var(--jb-shadow-lg);
    background: var(--jb-surface);
  }

  .slim-player-fab img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }

  .slim-player-fab--spinning img {
    animation: slim-disc-spin 8s linear infinite;
  }

  @keyframes slim-disc-spin {
    to {
      transform: rotate(360deg);
    }
  }
</style>
