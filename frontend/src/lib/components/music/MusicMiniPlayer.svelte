<script lang="ts">
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";
  import EnhancedCoverArt from "$lib/components/ui/EnhancedCoverArt.svelte";
  import { music } from "$lib/config/music.svelte";
  import { deviceSync } from "$lib/music/device-sync.svelte";
  import {
    trackCoverPaletteKey,
    trackCoverSeed,
  } from "$lib/music/cover-art-fallback";
  import { coverArtUrl, isInternetRadioTrack } from "$lib/subsonic";
  import FavoriteButton from "./FavoriteButton.svelte";
  import TrackMetaLinks from "./TrackMetaLinks.svelte";
  import ListenTogetherChip from "./ListenTogetherChip.svelte";
  import TrackContextMenu from "./TrackContextMenu.svelte";
  import { contextMenuPositionFromEvent } from "$lib/components/ui/context-menu";
  import Link from "$lib/router/Link.svelte";
  import { router } from "$lib/router/router.svelte";
  import { isWailsMobile } from "$lib/config/runtime";
  import { decorateTrack } from "$lib/extensions/registry";
  import ProgressSeek from "./ProgressSeek.svelte";
  import { preloadDecorationImages } from "$lib/extensions/assets";

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
  const image = $derived(
    track
      ? coverArtUrl(
          music.config,
          track.coverArt ?? track.albumId ?? track.id,
          128,
        )
      : null,
  );

  let expanded = $state(false);
  let trackMenu = $state<{ x: number; y: number } | null>(null);

  const touchLayout = $derived(
    isWailsMobile() ||
      (typeof window !== "undefined" &&
        window.matchMedia("(pointer: coarse)").matches),
  );
  const showExpanded = $derived(expanded || touchLayout);
</script>

{#if track && music.playerLayout === "mini" && music.playerVisible}
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div
    class="mini-player"
    class:mini-player--expanded={showExpanded}
    class:mini-player--touch={touchLayout}
    class:mini-player--together={deviceSync.inListenTogether}
    onmouseenter={() => (expanded = true)}
    onmouseleave={() => (expanded = false)}
    onfocusin={() => (expanded = true)}
    onfocusout={(e) => {
      if (!e.currentTarget.contains(e.relatedTarget as Node)) expanded = false;
    }}
    oncontextmenu={(event) => {
      if (!track || liveStream) return;
      trackMenu = contextMenuPositionFromEvent(event);
    }}
  >
    {#if showExpanded}
      <ProgressSeek
        class="mini-player__progress"
        size="sm"
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

      <div class="mini-player__body">
        <Link
          href="/music/now-playing"
          class="mini-player__art"
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
                class="mini-player__cover-overlay"
                src={coverOverlay}
                alt=""
                aria-hidden="true"
              />
            {:else}
              <span class="mini-player__cover-overlay" aria-hidden="true"
                >{coverOverlay}</span
              >
            {/if}
          {/if}
        </Link>

        <div class="mini-player__copy">
          <div
            class="mini-player__meta"
            role="link"
            tabindex="0"
            onclick={() => router.navigate("/music/now-playing")}
            onkeydown={(e) => {
              if (e.key === "Enter" || e.key === " ") {
                e.preventDefault();
                router.navigate("/music/now-playing");
              }
            }}
          >
            <span class="mini-player__title-row">
              {#if titlePrefix}
                <span class="mini-player__title-prefix" aria-hidden="true"
                  >{titlePrefix}</span
                >
              {/if}
              <span class="mini-player__title">{track.title}</span>
            </span>
            <TrackMetaLinks {track} class="mini-player__artist" />
          </div>
          <ListenTogetherChip compact />
        </div>

        <div class="mini-player__controls">
          <button
            type="button"
            class="mini-player__btn mini-player__btn--play"
            onclick={() => {
              if (!deviceSync.dispatchTransport("toggle")) music.togglePlay();
            }}
            aria-label={music.playing ? "Pause" : "Play"}
          >
            {#if music.playing}
              <MdiIcon name="pause" size={18} />
            {:else}
              <MdiIcon name="play" size={18} />
            {/if}
          </button>
          <button
            type="button"
            class="mini-player__btn"
            onclick={() => {
              if (!deviceSync.dispatchTransport("next")) music.next();
            }}
            aria-label="Next"
          >
            <MdiIcon name="skipForward" size={16} />
          </button>
          <Link
            href="/music/now-playing"
            class="mini-player__btn"
            title="Now playing"
          >
            <MdiIcon name="chevronUp" size={16} />
          </Link>
          {#if !liveStream}
            <FavoriteButton {track} size={15} class="mini-player__favorite" />
          {/if}
          <button
            type="button"
            class="mini-player__btn mini-player__btn--close"
            onclick={() => music.dismissPlayer()}
            aria-label="Close player"
          >
            <MdiIcon name="x" size={16} />
          </button>
        </div>
      </div>
    {:else}
      <button
        type="button"
        class="mini-player__peek"
        onclick={() => (expanded = true)}
        aria-label="Expand player"
      >
        <span class="mini-player__peek-art-wrap">
          <span
            class="mini-player__disc-inner"
            class:mini-player__disc-inner--spin={music.playing}
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
          </span>
          {#if music.playing}
            <span class="mini-player__disc-ring"></span>
          {/if}
        </span>
        <span class="mini-player__peek-meta">
          <span class="mini-player__peek-title">{track.title}</span>
          <span class="mini-player__peek-subtitle">
            {#if deviceSync.inListenTogether}
              {deviceSync.sessionLabel}
            {:else}
              {track.artist}
            {/if}
          </span>
        </span>
        <MdiIcon name="chevronUp" size={16} />
      </button>
    {/if}
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
  .mini-player {
    position: fixed;
    bottom: calc(var(--jb-space-5) + env(safe-area-inset-bottom, 0px));
    right: var(--jb-space-5);
    z-index: 60;
    transition:
      width 0.28s cubic-bezier(0.4, 0, 0.2, 1),
      height 0.28s cubic-bezier(0.4, 0, 0.2, 1),
      border-radius 0.28s cubic-bezier(0.4, 0, 0.2, 1);
  }

  .mini-player--expanded {
    width: min(22rem, calc(100vw - 2rem));
    border-radius: var(--jb-radius-xl);
    overflow: visible;
    color: var(--jb-text);
    background: color-mix(in srgb, var(--jb-bg-elevated) 94%, transparent);
    backdrop-filter: blur(24px);
    border: 1px solid var(--jb-border);
    box-shadow: var(--jb-shadow-lg);
  }

  .mini-player--expanded :global(.mini-player__progress) {
    position: relative;
    z-index: 2;
    border-radius: var(--jb-radius-xl) var(--jb-radius-xl) 0 0;
    overflow: visible;
  }

  .mini-player--together.mini-player--expanded,
  .mini-player--together .mini-player__peek {
    border-color: color-mix(in srgb, var(--jb-accent) 50%, var(--jb-border));
  }

  .mini-player__peek {
    display: inline-flex;
    align-items: center;
    gap: var(--jb-space-3);
    width: min(20rem, calc(100vw - 2rem));
    min-height: 4.5rem;
    padding: 0.5rem 0.75rem;
    border: 1px solid var(--jb-border);
    border-radius: var(--jb-radius-xl);
    background: color-mix(in srgb, var(--jb-bg-elevated) 96%, transparent);
    backdrop-filter: blur(16px);
    color: var(--jb-text);
    box-shadow: var(--jb-shadow-lg);
    cursor: pointer;
  }

  .mini-player__peek-art-wrap {
    position: relative;
    width: 3.25rem;
    height: 3.25rem;
    flex-shrink: 0;
    border-radius: var(--jb-radius-full);
    overflow: hidden;
    border: 2px solid var(--jb-border);
    background: var(--jb-surface);
  }

  .mini-player__disc-inner {
    display: block;
    width: 100%;
    height: 100%;
  }

  .mini-player__disc-inner :global(.cover-art) {
    width: 100%;
    height: 100%;
    border-radius: 50%;
  }

  .mini-player__disc-inner--spin {
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

  .mini-player__disc-ring {
    position: absolute;
    inset: -3px;
    border: 2px solid var(--jb-accent);
    border-radius: var(--jb-radius-full);
    animation: disc-ring 1.4s ease-out infinite;
  }

  .mini-player__peek-meta {
    display: flex;
    flex: 1;
    min-width: 0;
    flex-direction: column;
    align-items: flex-start;
    gap: 0.1rem;
  }

  .mini-player__peek-title {
    width: 100%;
    font-size: 0.875rem;
    font-weight: 650;
    color: var(--jb-text);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    text-align: left;
  }

  .mini-player__peek-subtitle {
    width: 100%;
    font-size: 0.75rem;
    color: var(--jb-text-muted);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    text-align: left;
  }

  @keyframes disc-ring {
    from {
      opacity: 0.9;
      transform: scale(1);
    }
    to {
      opacity: 0;
      transform: scale(1.12);
    }
  }

  :global(.mini-player__progress) {
    border-radius: 0;
  }

  .mini-player__body {
    position: relative;
    z-index: 1;
    display: flex;
    align-items: center;
    gap: var(--jb-space-3);
    padding: var(--jb-space-3);
    border-radius: 0 0 var(--jb-radius-xl) var(--jb-radius-xl);
    overflow: hidden;
  }

  :global(a.mini-player__art) {
    width: 3rem;
    height: 3rem;
    border-radius: var(--jb-radius-md);
    flex-shrink: 0;
    overflow: hidden;
    box-shadow: var(--jb-shadow-sm);
    text-decoration: none;
    display: block;
    position: relative;
  }

  :global(a.mini-player__art .cover-art) {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }

  :global(.mini-player__cover-overlay) {
    position: absolute;
    right: 0.1rem;
    bottom: 0.1rem;
    width: 0.9rem;
    height: 0.9rem;
    object-fit: contain;
    image-rendering: pixelated;
    pointer-events: none;
    filter: drop-shadow(1px 1px 0 #000);
    font-size: 0.65rem;
    line-height: 1;
    display: grid;
    place-items: center;
  }

  .mini-player__title-prefix {
    flex-shrink: 0;
    color: var(--jb-text-muted);
    font-size: 0.8rem;
    line-height: 1;
  }

  .mini-player__copy {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: 0.25rem;
  }

  .mini-player__meta {
    flex: 1;
    min-width: 0;
    width: 100%;
    border: none;
    background: transparent;
    text-align: left;
    color: var(--jb-text);
    cursor: pointer;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: 0.125rem;
  }

  .mini-player__title-row {
    display: flex;
    align-items: center;
    gap: var(--jb-space-2);
    min-width: 0;
  }

  .mini-player__title {
    flex: 1;
    min-width: 0;
    font-weight: 650;
    font-size: 0.875rem;
    color: var(--jb-text);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .mini-player :global(.mini-player__favorite) {
    flex-shrink: 0;
    position: relative;
    z-index: 3;
    width: 1.75rem;
    height: 1.75rem;
  }

  :global(.mini-player__artist.track-meta-links) {
    font-size: 0.75rem;
    color: var(--jb-text-muted);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .mini-player__controls {
    display: flex;
    align-items: center;
    gap: 0.125rem;
    flex-shrink: 0;
  }

  .mini-player__btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 2rem;
    height: 2rem;
    border: none;
    border-radius: var(--jb-radius-full);
    background: transparent;
    color: var(--jb-text-muted);
    cursor: pointer;
    text-decoration: none;
  }

  .mini-player__btn:hover {
    background: var(--jb-surface-hover);
    color: var(--jb-text);
  }

  .mini-player__btn--play {
    width: 2.25rem;
    height: 2.25rem;
    background: var(--jb-accent);
    color: white;
  }

  .mini-player__btn--play:hover {
    background: var(--jb-accent-hover);
    color: white;
  }

  .mini-player__btn--close:hover {
    color: var(--jb-danger);
  }

  .mini-player--touch .mini-player__btn {
    width: 2.5rem;
    height: 2.5rem;
  }

  .mini-player--touch .mini-player__btn--play {
    width: 2.75rem;
    height: 2.75rem;
  }

  @media (max-width: 480px) {
    .mini-player {
      right: var(--jb-space-3);
      bottom: calc(var(--jb-space-3) + env(safe-area-inset-bottom, 0px));
    }

    .mini-player__peek {
      width: calc(100vw - (var(--jb-space-3) * 2));
      border-radius: var(--jb-radius-lg);
      box-shadow: var(--jb-shadow-sm);
    }

    .mini-player--expanded {
      left: var(--jb-space-3);
      width: auto;
      border-radius: var(--jb-radius-lg);
      box-shadow: var(--jb-shadow-sm);
      backdrop-filter: none;
    }
  }

  @media (max-width: 768px) {
    .mini-player--expanded {
      background: var(--jb-bg-elevated);
      backdrop-filter: none;
    }
  }
</style>
