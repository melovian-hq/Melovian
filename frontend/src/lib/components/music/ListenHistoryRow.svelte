<script lang="ts">
  import CoverArt from "$lib/components/ui/CoverArt.svelte";
  import Link from "$lib/router/Link.svelte";
  import { music } from "$lib/config/music.svelte";
  import {
    trackCoverPaletteKey,
    trackCoverSeed,
  } from "$lib/music/cover-art-fallback";
  import {
    formatExactPlayedAt,
    formatPlayedClock,
    formatRelativePlayedAt,
  } from "$lib/music/relative-time";
  import { coverArtUrl, formatDurationMs } from "$lib/subsonic";
  import CoverArtPlayOverlay from "./CoverArtPlayOverlay.svelte";
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";
  import TrackContextMenu from "./TrackContextMenu.svelte";
  import {
    contextMenu,
    contextMenuPositionForTrigger,
    type ContextMenuPosition,
  } from "$lib/components/ui/context-menu";
  import type { ListenEvent } from "$lib/subsonic/types";
  import type { SubsonicSong } from "$lib/subsonic";

  interface Props {
    event: ListenEvent;
    onplay?: () => void;
  }

  let { event, onplay }: Props = $props();
  let menu = $state<{ x: number; y: number } | null>(null);

  const track = $derived({
    id: event.trackId,
    title: event.trackTitle,
    artist: event.artistName,
    album: event.albumTitle,
    albumId: event.albumId,
    coverArt: event.coverArtId,
    duration: Math.floor(event.durationMs / 1000),
  } satisfies SubsonicSong);

  function onContextMenu(pos: ContextMenuPosition) {
    menu = pos;
  }

  const image = $derived(
    coverArtUrl(music.config, track.coverArt ?? track.albumId ?? track.id, 96),
  );
  const isCurrent = $derived(music.currentTrack?.id === track.id);
  const relativeDate = $derived(formatRelativePlayedAt(event.playedAt));
  const playedClock = $derived(formatPlayedClock(event.playedAt));
  const exactDate = $derived(formatExactPlayedAt(event.playedAt));
  const duration = $derived(formatDurationMs(event.durationMs));
</script>

{#if menu}
  <TrackContextMenu
    {track}
    x={menu.x}
    y={menu.y}
    onclose={() => (menu = null)}
  />
{/if}

<div
  class="history-row"
  class:history-row--playing={isCurrent && music.playing}
  role="button"
  tabindex="0"
  onclick={() => onplay?.()}
  onkeydown={(event) => {
    if (event.key === "Enter" || event.key === " ") {
      event.preventDefault();
      onplay?.();
    }
  }}
  use:contextMenu={onContextMenu}
>
  <div class="history-row__main">
    <div class="history-row__thumb">
      <CoverArt
        src={image}
        seed={trackCoverSeed(track)}
        paletteKey={trackCoverPaletteKey(track)}
        loading="lazy"
      />
      <CoverArtPlayOverlay playing={isCurrent && music.playing} />
    </div>
    <span class="history-row__text">
      <span class="history-row__title" title={track.title}>{track.title}</span>
      <span class="history-row__artist" title={track.artist ?? "Unknown artist"}
        >{track.artist ?? "Unknown artist"}</span
      >
    </span>
  </div>

  {#if event.albumId}
    <Link
      href="/music/album/{event.albumId}"
      class="history-row__album"
      onclick={(click: MouseEvent) => click.stopPropagation()}
    >
      {event.albumTitle || "Unknown album"}
    </Link>
  {:else}
    <span class="history-row__album">{event.albumTitle || "Unknown album"}</span
    >
  {/if}

  <span class="history-row__when" title={exactDate} aria-label={exactDate}>
    <span class="history-row__relative">{relativeDate}</span>
    <span class="history-row__clock">{playedClock}</span>
  </span>

  <span class="history-row__duration">{duration}</span>

  <button
    type="button"
    class="history-row__menu"
    aria-label="More actions for {track.title}"
    onclick={(event) => (menu = contextMenuPositionForTrigger(event))}
  >
    <MdiIcon name="dotsHorizontal" size={16} />
  </button>
</div>

<style>
  .history-row {
    display: grid;
    grid-template-columns: minmax(0, 2fr) minmax(0, 1.2fr) 6.5rem 4rem auto;
    align-items: center;
    content-visibility: auto;
    contain-intrinsic-size: auto 3.625rem;
    gap: var(--jb-space-3);
    padding: var(--jb-space-2) var(--jb-space-4);
    border-radius: var(--jb-radius-md);
    color: var(--jb-text);
    transition: background var(--jb-transition);
    cursor: pointer;
  }

  .history-row:hover {
    background: var(--jb-surface-hover);
  }

  .history-row--playing .history-row__title {
    color: var(--jb-active);
  }

  .history-row__main {
    display: flex;
    align-items: center;
    gap: var(--jb-space-3);
    min-width: 0;
    pointer-events: none;
  }

  .history-row__thumb {
    position: relative;
    width: 2.75rem;
    height: 2.75rem;
    border-radius: var(--jb-radius-sm);
    flex-shrink: 0;
    overflow: hidden;
    box-shadow: var(--jb-shadow-sm);
    pointer-events: auto;
  }

  .history-row__thumb:hover :global(.cover-play-overlay),
  .history-row:hover .history-row__thumb :global(.cover-play-overlay) {
    opacity: 1;
  }

  .history-row__thumb :global(.cover-art) {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }

  .history-row__text {
    display: flex;
    flex-direction: column;
    gap: 0.125rem;
    min-width: 0;
  }

  .history-row__title {
    font-weight: 600;
    font-size: 0.9375rem;
    color: var(--jb-text);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .history-row__artist {
    font-size: 0.8125rem;
    color: var(--jb-text-muted);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  :global(a.history-row__album),
  .history-row__album {
    min-width: 0;
    font-size: 0.8125rem;
    color: var(--jb-text-muted);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    text-decoration: none;
  }

  :global(a.history-row__album:hover) {
    color: var(--jb-text);
    text-decoration: underline;
  }

  .history-row__when {
    display: flex;
    flex-direction: column;
    align-items: flex-end;
    gap: 0.125rem;
    min-width: 0;
    cursor: default;
    pointer-events: none;
  }

  .history-row__relative {
    font-size: 0.8125rem;
    color: var(--jb-text-muted);
    white-space: nowrap;
  }

  .history-row__clock {
    font-size: 0.75rem;
    color: var(--jb-text-subtle);
    font-variant-numeric: tabular-nums;
    white-space: nowrap;
  }

  .history-row__duration {
    font-size: 0.8125rem;
    color: var(--jb-text-subtle);
    font-variant-numeric: tabular-nums;
    text-align: right;
    pointer-events: none;
  }

  .history-row__menu {
    display: grid;
    place-content: center;
    width: 1.75rem;
    height: 1.75rem;
    padding: 0;
    border: none;
    border-radius: var(--jb-radius-full);
    background: transparent;
    color: var(--jb-text-subtle);
    cursor: pointer;
    opacity: 0;
    transition:
      opacity var(--jb-transition),
      color var(--jb-transition);
  }

  .history-row__menu:hover {
    color: var(--jb-text);
  }

  .history-row__menu:focus-visible {
    outline: none;
    box-shadow: var(--jb-focus-ring);
  }

  @media (hover: hover) {
    .history-row:hover .history-row__menu,
    .history-row:focus-within .history-row__menu {
      opacity: 1;
    }
  }

  @media (hover: none) {
    .history-row__menu {
      opacity: 1;
    }
  }

  @media (max-width: 768px) {
    .history-row {
      grid-template-columns: minmax(0, 1fr) auto auto;
      padding-inline: var(--jb-space-3);
    }

    :global(a.history-row__album),
    .history-row__album,
    .history-row__duration {
      display: none;
    }
  }
</style>
