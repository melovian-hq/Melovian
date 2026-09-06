<script lang="ts">
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";
  import EnhancedCoverArt from "$lib/components/ui/EnhancedCoverArt.svelte";
  import { music } from "$lib/config/music.svelte";
  import { deviceSync } from "$lib/music/device-sync.svelte";
  import {
    trackCoverPaletteKey,
    trackCoverSeed,
  } from "$lib/music/cover-art-fallback";
  import TrackQualityBadge from "./TrackQualityBadge.svelte";
  import FavoriteButton from "./FavoriteButton.svelte";
  import TrackMetaLinks from "./TrackMetaLinks.svelte";
  import ListenTogetherChip from "./ListenTogetherChip.svelte";
  import TrackContextMenu from "./TrackContextMenu.svelte";
  import { contextMenuPositionFromEvent } from "$lib/components/ui/context-menu";
  import Link from "$lib/router/Link.svelte";
  import {
    coverArtUrl,
    formatDuration,
    isInternetRadioTrack,
  } from "$lib/subsonic";
  import { isRemoteCacheableTrack } from "$lib/music/track-cache";
  import { decorateTrack } from "$lib/extensions/registry";
  import ProgressSeek from "./ProgressSeek.svelte";
  import { preloadDecorationImages } from "$lib/extensions/assets";
  import { extensionFeatures } from "$lib/extensions/features.svelte";
  import { overlayFade } from "$lib/router/page-motion";
  import { fade } from "svelte/transition";

  const track = $derived(music.currentTrack);
  const trackDecoration = $derived(track ? decorateTrack(track) : {});
  const titlePrefix = $derived(trackDecoration.titlePrefix?.trim() || "");
  const coverOverlay = $derived(trackDecoration.coverOverlayIcon?.trim() || "");
  const coverOverlayIsUrl = $derived(
    coverOverlay.startsWith("/") ||
      coverOverlay.startsWith("http://") ||
      coverOverlay.startsWith("https://"),
  );

  $effect(() => {
    if (coverOverlayIsUrl) preloadDecorationImages([coverOverlay]);
  });

  const liveStream = $derived(track ? isInternetRadioTrack(track) : false);
  const canCacheTrack = $derived(track ? isRemoteCacheableTrack(track) : false);
  const image = $derived(
    track
      ? coverArtUrl(
          music.config,
          track.coverArt ?? track.albumId ?? track.id,
          256,
        )
      : null,
  );
  let trackMenu = $state<{ x: number; y: number } | null>(null);
</script>

{#if track && music.playerLayout === "full" && music.playerVisible}
  <div
    class="player"
    class:player--together={deviceSync.inListenTogether}
    role="group"
    oncontextmenu={(event) => {
      if (liveStream) return;
      trackMenu = contextMenuPositionFromEvent(event);
    }}
  >
    <ProgressSeek
      class="player__progress"
      size="md"
      currentTime={music.currentTime}
      duration={music.duration}
      progressPercent={music.smoothProgress}
      playing={music.playing}
      decoration={trackDecoration}
      disabled={liveStream}
      onSeek={(seconds) => {
        if (deviceSync.dispatchTransport("seek", { positionSec: seconds }))
          return;
        music.seek(seconds);
      }}
    />

    <div class="player__body">
      <div class="player__track">
        {#key track.id}
          <div class="player__track-swap" in:fade={overlayFade()}>
            <Link
              href="/music/now-playing"
              class="player__art"
              title="Now playing"
            >
              <EnhancedCoverArt
                kind="track"
                entity={track
                  ? {
                      id: track.id,
                      title: track.title,
                      artist: track.artist,
                      album: track.album,
                      coverArt: track.coverArt,
                      albumId: track.albumId,
                    }
                  : { id: "track" }}
                src={image}
                seed={track ? trackCoverSeed(track) : "track"}
                paletteKey={track ? trackCoverPaletteKey(track) : "track"}
              />
              {#if coverOverlay}
                {#if coverOverlayIsUrl}
                  <img
                    class="player__cover-overlay"
                    src={coverOverlay}
                    alt=""
                    aria-hidden="true"
                  />
                {:else}
                  <span class="player__cover-overlay" aria-hidden="true"
                    >{coverOverlay}</span
                  >
                {/if}
              {/if}
            </Link>
            <div class="player__meta">
              <span class="player__title-row">
                {#if titlePrefix}
                  <span class="player__title-prefix" aria-hidden="true"
                    >{titlePrefix}</span
                  >
                {/if}
                <span class="player__title">{track.title}</span>
                <TrackQualityBadge {track} size="md" />
              </span>
              <TrackMetaLinks {track} class="player__artist" />
              <ListenTogetherChip />
            </div>
          </div>
        {/key}
        {#if !liveStream}
          <div class="player__track-actions">
            <FavoriteButton {track} size={18} />
          </div>
        {/if}
      </div>

      <div class="player__controls">
        <button
          type="button"
          class="player__btn player__btn--ghost"
          class:player__btn--active={music.shuffle}
          onclick={() => (music.shuffle = !music.shuffle)}
          aria-label="Shuffle"
        >
          <MdiIcon name="shuffle" size={16} />
        </button>
        <button
          type="button"
          class="player__btn player__btn--ghost"
          class:player__btn--active={music.repeat !== "off"}
          onclick={() => music.cycleRepeat()}
          aria-label={music.repeat === "one"
            ? "Repeat one"
            : music.repeat === "all"
              ? "Repeat all"
              : "Repeat off"}
          title={music.repeat === "one"
            ? "Repeat one"
            : music.repeat === "all"
              ? "Repeat all"
              : "Repeat off"}
        >
          <MdiIcon
            name={music.repeat === "one" ? "repeatOnce" : "repeat"}
            size={16}
          />
        </button>
        <button
          type="button"
          class="player__btn"
          onclick={() => {
            if (!deviceSync.dispatchTransport("prev")) music.previous();
          }}
          aria-label="Previous"
        >
          <MdiIcon name="skipBack" size={20} />
        </button>
        <button
          type="button"
          class="player__btn player__btn--play"
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
          class="player__btn"
          onclick={() => {
            if (!deviceSync.dispatchTransport("next")) music.next();
          }}
          aria-label="Next"
        >
          <MdiIcon name="skipForward" size={20} />
        </button>
        <button
          type="button"
          class="player__btn player__btn--ghost"
          class:player__btn--busy={music.continuousBusy === "internet"}
          disabled={music.continuousBusy !== "off"}
          onclick={() => void music.playRandomInternetRadio()}
          aria-label="Internet radio"
          title={music.continuousBusy === "internet"
            ? "Tuning internet radio…"
            : "Play a random internet radio station"}
        >
          {#if music.continuousBusy === "internet"}
            <span class="player__busy" aria-hidden="true"></span>
          {:else}
            <MdiIcon name="signal" size={16} />
          {/if}
        </button>
        <span class="player__time">
          {#if liveStream}
            Live
          {:else}
            {formatDuration(Math.floor(music.currentTime))} / {formatDuration(
              Math.floor(music.duration),
            )}
          {/if}
        </span>
      </div>

      <div class="player__extras">
        <button
          type="button"
          class="player__btn player__btn--ghost player__btn--devices"
          class:player__btn--active={deviceSync.panelOpen ||
            deviceSync.inListenTogether}
          onclick={() => deviceSync.togglePanel()}
          aria-label={deviceSync.inListenTogether
            ? deviceSync.sessionLabel
            : "Devices"}
          title={deviceSync.inListenTogether
            ? deviceSync.sessionLabel
            : "Devices, handoff, and listen together"}
        >
          <MdiIcon name="monitor" size={16} />
        </button>
        <button
          type="button"
          class="player__btn player__btn--ghost"
          class:player__btn--active={music.queueOpen}
          onclick={() => music.toggleQueue()}
          aria-label="Queue"
        >
          <MdiIcon name="listMusic" size={16} />
        </button>
        {#if canCacheTrack}
          <button
            type="button"
            class="player__btn player__btn--ghost"
            class:player__btn--active={track && music.isDownloaded(track.id)}
            onclick={() => track && void music.toggleDownload(track)}
            aria-label={track && music.isDownloaded(track.id)
              ? "Remove offline download"
              : "Download for offline"}
            title={track && music.isDownloaded(track.id)
              ? "Downloaded for offline"
              : "Download for offline"}
          >
            <MdiIcon
              name={track && music.isDownloaded(track.id)
                ? "downloadDone"
                : "download"}
              size={16}
            />
          </button>
        {/if}
        {#if extensionFeatures.lyrics}
          <button
            type="button"
            class="player__btn player__btn--ghost"
            class:player__btn--active={music.lyricsOpen}
            class:player__btn--disabled={liveStream}
            onclick={() => music.toggleLyricsPanel()}
            aria-label="Lyrics"
            disabled={liveStream}
            title={liveStream
              ? "Lyrics unavailable for live streams"
              : "Lyrics"}
          >
            <MdiIcon name="lyrics" size={16} />
          </button>
        {/if}
        <button
          type="button"
          class="player__btn player__btn--ghost"
          class:player__btn--active={music.eq.enabled || music.eqOpen}
          class:player__btn--disabled={!music.eqAvailable}
          onclick={() => music.toggleEqPanel()}
          aria-label="Equalizer"
          disabled={!music.eqAvailable}
          title={music.eqAvailable
            ? "Equalizer"
            : "Equalizer unavailable with native playback"}
        >
          <MdiIcon name="slidersHorizontal" size={16} />
        </button>
        <div class="player__volume">
          <MdiIcon name="volume2" size={16} />
          <input
            type="range"
            min="0"
            max="1"
            step="0.01"
            value={music.volume}
            oninput={(e) =>
              music.setVolume(
                Number((e.currentTarget as HTMLInputElement).value),
              )}
            aria-label="Volume"
          />
        </div>
        <button
          type="button"
          class="player__btn player__btn--ghost"
          onclick={() => music.dismissPlayer()}
          aria-label="Hide player"
        >
          <MdiIcon name="chevronDown" size={16} />
        </button>
      </div>
    </div>
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

{#if track && music.playerLayout === "dismissed" && !music.onPlayRoute}
  <button
    type="button"
    class="player-fab"
    class:player-fab--spinning={music.playing}
    onclick={() => music.restorePlayer()}
    aria-label="Show player"
  >
    {#if image}
      <img src={image} alt="" />
    {/if}
    {#if music.playing}
      <span class="player-fab__pulse"></span>
    {/if}
  </button>
{/if}

<style>
  .player {
    position: fixed;
    bottom: 0;
    left: var(--jb-sidebar-current-width, 0);
    right: 0;
    z-index: 50;
    color: var(--jb-text);
    background: color-mix(in srgb, var(--jb-bg-elevated) 94%, transparent);
    backdrop-filter: blur(24px);
    border-top: 1px solid var(--jb-border);
    box-shadow: 0 -12px 40px rgb(0 0 0 / 0.18);
    transition: left var(--jb-transition);
    padding-bottom: env(safe-area-inset-bottom, 0px);
    overflow: visible;
  }

  .player--together {
    border-top-color: color-mix(
      in srgb,
      var(--jb-accent) 55%,
      var(--jb-border)
    );
    box-shadow:
      0 -12px 40px rgb(0 0 0 / 0.18),
      0 -1px 0 color-mix(in srgb, var(--jb-accent) 40%, transparent);
  }

  :global(.player__progress) {
    position: relative;
    z-index: 2;
    overflow: visible;
  }

  .player__body {
    position: relative;
    z-index: 1;
    display: grid;
    grid-template-columns: 1fr auto 1fr;
    align-items: center;
    gap: var(--jb-space-4);
    padding: var(--jb-space-3) var(--jb-space-6);
  }

  .player__track {
    display: flex;
    align-items: center;
    gap: var(--jb-space-3);
    min-width: 0;
  }

  .player__track-swap {
    display: flex;
    align-items: center;
    gap: var(--jb-space-3);
    min-width: 0;
    flex: 1;
  }

  :global(a.player__art) {
    width: 3.75rem;
    height: 3.75rem;
    border-radius: var(--jb-radius-md);
    overflow: hidden;
    box-shadow: var(--jb-shadow-md);
    flex-shrink: 0;
    text-decoration: none;
    transition: transform var(--jb-transition);
    display: block;
    position: relative;
  }

  :global(a.player__art:hover) {
    transform: scale(1.04);
  }

  :global(a.player__art .cover-art) {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }

  :global(.player__cover-overlay) {
    position: absolute;
    right: 0.15rem;
    bottom: 0.15rem;
    width: 1.1rem;
    height: 1.1rem;
    object-fit: contain;
    image-rendering: pixelated;
    pointer-events: none;
    filter: drop-shadow(1px 1px 0 #000);
    font-size: 0.75rem;
    line-height: 1;
    display: grid;
    place-items: center;
  }

  .player__title-prefix {
    flex-shrink: 0;
    color: var(--jb-text-muted);
    font-size: 0.875rem;
    line-height: 1;
  }

  .player__meta {
    display: flex;
    flex-direction: column;
    gap: 0.2rem;
    min-width: 0;
  }

  .player__track-actions {
    margin-left: auto;
    display: inline-flex;
    align-items: center;
    flex-shrink: 0;
  }

  .player__title-row {
    display: flex;
    align-items: center;
    gap: var(--jb-space-2);
    min-width: 0;
  }

  .player__title {
    font-weight: 700;
    font-size: 0.9375rem;
    color: var(--jb-text);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  :global(.player__artist.track-meta-links) {
    font-size: 0.8125rem;
    color: var(--jb-text-muted);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .player__controls,
  .player__extras {
    display: flex;
    align-items: center;
    gap: var(--jb-space-1);
  }

  .player__extras {
    justify-self: end;
  }

  .player__btn {
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
    transition:
      background var(--jb-transition),
      color var(--jb-transition);
  }

  .player__btn:hover,
  .player__btn:active {
    background: var(--jb-surface-hover);
  }

  .player__btn--ghost {
    color: var(--jb-text-muted);
    width: 2rem;
    height: 2rem;
  }

  .player__btn--active {
    color: var(--jb-accent);
  }

  .player__btn--devices {
    position: relative;
  }

  .player__btn--disabled {
    opacity: 0.35;
    cursor: not-allowed;
  }

  .player__btn:disabled {
    opacity: 0.55;
    cursor: wait;
  }

  .player__btn--busy {
    color: var(--jb-accent);
  }

  .player__busy {
    width: 0.9rem;
    height: 0.9rem;
    border-radius: var(--jb-radius-full);
    border: 2px solid color-mix(in srgb, var(--jb-accent) 35%, transparent);
    border-top-color: var(--jb-accent);
    animation: player-spin 0.7s linear infinite;
  }

  @keyframes player-spin {
    to {
      transform: rotate(360deg);
    }
  }

  .player__btn--play {
    width: 3rem;
    height: 3rem;
    background: var(--jb-accent);
    color: white;
  }

  .player__btn--play:hover,
  .player__btn--play:active {
    background: var(--jb-accent-hover);
  }

  .player__time {
    font-size: 0.75rem;
    color: var(--jb-text-subtle);
    font-variant-numeric: tabular-nums;
    margin-left: var(--jb-space-2);
    white-space: nowrap;
  }

  .player__volume {
    display: flex;
    align-items: center;
    gap: var(--jb-space-2);
    color: var(--jb-text-muted);
  }

  .player__volume input[type="range"] {
    width: 5rem;
    accent-color: var(--jb-accent);
  }

  .player-fab {
    position: fixed;
    bottom: var(--jb-space-5);
    right: var(--jb-space-5);
    z-index: 55;
    width: 3.5rem;
    height: 3.5rem;
    border: 2px solid var(--jb-border);
    border-radius: var(--jb-radius-full);
    padding: 0;
    overflow: hidden;
    cursor: pointer;
    box-shadow: var(--jb-shadow-lg);
    background: var(--jb-surface);
  }

  .player-fab img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }

  .player-fab--spinning img {
    animation: disc-spin 8s linear infinite;
  }

  @keyframes disc-spin {
    from {
      transform: rotate(0deg);
    }
    to {
      transform: rotate(360deg);
    }
  }

  .player-fab__pulse {
    position: absolute;
    inset: -4px;
    border: 2px solid var(--jb-accent);
    border-radius: var(--jb-radius-full);
    animation: fab-pulse 1.2s ease-out infinite;
  }

  @keyframes fab-pulse {
    from {
      opacity: 1;
      transform: scale(1);
    }
    to {
      opacity: 0;
      transform: scale(1.15);
    }
  }

  @media (max-width: 768px) {
    .player {
      left: 0;
    }

    .player__body {
      grid-template-columns: minmax(0, 1fr) auto;
      gap: var(--jb-space-2);
      padding: var(--jb-space-2) var(--jb-space-3);
    }

    .player__track {
      grid-column: 1 / -1;
    }

    .player__extras,
    .player__time {
      display: none;
    }

    .player__controls {
      justify-content: center;
      grid-column: 1 / -1;
    }

    :global(a.player__art) {
      width: 3rem;
      height: 3rem;
    }

    .player__title {
      font-size: 0.875rem;
    }

    :global(.player__artist.track-meta-links) {
      font-size: 0.8125rem;
    }
  }

  @media (max-height: 700px), (max-width: 840px) {
    .player__body {
      grid-template-columns: minmax(0, 1fr) auto;
      gap: var(--jb-space-2);
      padding: var(--jb-space-2) var(--jb-space-3);
    }

    .player__track {
      gap: var(--jb-space-2);
    }

    :global(a.player__art) {
      width: 2.75rem;
      height: 2.75rem;
    }

    .player__title {
      font-size: 0.8125rem;
    }

    .player__extras {
      display: none;
    }

    .player__time {
      display: none;
    }

    .player__volume {
      width: 4.5rem;
    }
  }

  @media (max-width: 480px) {
    .player__body {
      grid-template-columns: 1fr;
    }

    .player__volume {
      display: none;
    }
  }
</style>
