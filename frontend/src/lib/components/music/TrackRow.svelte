<script lang="ts">
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";
  import EnhancedCoverArt from "$lib/components/ui/EnhancedCoverArt.svelte";
  import { music } from "$lib/config/music.svelte";
  import { trackSelection } from "$lib/music/selection.svelte";
  import {
    trackCoverPaletteKey,
    trackCoverSeed,
  } from "$lib/music/cover-art-fallback";
  import { coverArtUrl, formatDuration } from "$lib/subsonic";
  import PlaylistPicker from "./PlaylistPicker.svelte";
  import NowPlayingBars from "./NowPlayingBars.svelte";
  import TrackQualityBadge from "./TrackQualityBadge.svelte";
  import FavoriteButton from "./FavoriteButton.svelte";
  import CoverArtPlayOverlay from "./CoverArtPlayOverlay.svelte";
  import TrackContextMenu from "./TrackContextMenu.svelte";
  import { contextMenuPositionFromEvent } from "$lib/components/ui/context-menu";
  import type { SubsonicSong } from "$lib/subsonic";

  interface Props {
    track: SubsonicSong;
    index?: number;
    showIndex?: boolean;
    active?: boolean;
    selectable?: boolean;
    subtitle?: string;
    versions?: SubsonicSong[];
    onplay?: () => void;
    onplayversion?: (track: SubsonicSong) => void;
  }

  let {
    track,
    index = 0,
    showIndex = true,
    active = false,
    selectable = false,
    subtitle,
    versions,
    onplay,
    onplayversion,
  }: Props = $props();

  const compact = $derived(!showIndex && !selectable);
  const versionList = $derived(
    versions && versions.length > 1 ? versions : undefined,
  );

  const image = $derived(
    track.coverArt
      ? coverArtUrl(music.config, track.coverArt, 96)
      : track.albumId
        ? coverArtUrl(music.config, track.albumId, 96)
        : null,
  );
  const isCurrent = $derived(
    music.currentTrack?.id === track.id ||
      versionList?.some((version) => version.id === music.currentTrack?.id) ===
        true,
  );
  const selected = $derived(trackSelection.isSelected(track.id));

  let pickerOpen = $state(false);
  let versionsOpen = $state(false);
  let contextMenu = $state<{ x: number; y: number } | null>(null);

  function handlePlay() {
    onplay?.();
  }

  function handleSelect() {
    trackSelection.toggle(track);
  }

  function handlePlayVersion(version: SubsonicSong) {
    versionsOpen = false;
    if (version.id === track.id) {
      onplay?.();
      return;
    }
    onplayversion?.(version);
  }

  function toggleVersions(event: MouseEvent) {
    event.stopPropagation();
    versionsOpen = !versionsOpen;
  }

  function closeVersions() {
    versionsOpen = false;
  }

  function openContextMenu(event: MouseEvent) {
    const pos = contextMenuPositionFromEvent(event);
    if (!pos) return;
    contextMenu = pos;
  }
</script>

{#if contextMenu}
  <TrackContextMenu
    {track}
    x={contextMenu.x}
    y={contextMenu.y}
    onclose={() => (contextMenu = null)}
  />
{/if}

<div
  class="track-row"
  class:track-row--compact={compact}
  class:track-row--active={active || isCurrent}
  class:track-row--playing={isCurrent && music.playing}
  class:track-row--selected={selected}
  class:track-row--versions-open={versionsOpen}
  role="button"
  tabindex="0"
  onclick={handlePlay}
  onkeydown={(event) => {
    if (event.key === "Enter" || event.key === " ") {
      event.preventDefault();
      handlePlay();
    }
  }}
  oncontextmenu={openContextMenu}
>
  {#if selectable}
    <button
      type="button"
      class="track-row__check"
      class:track-row__check--on={selected}
      onclick={(event) => {
        event.stopPropagation();
        handleSelect();
      }}
      aria-label={selected ? "Deselect track" : "Select track"}
    >
      {#if selected}
        <MdiIcon name="check" size={14} />
      {/if}
    </button>
  {:else if showIndex}
    <span class="track-row__index">
      {#if isCurrent && music.playing}
        <NowPlayingBars />
      {:else}
        {track.track ?? index + 1}
      {/if}
    </span>
  {/if}

  <div class="track-row__content">
    <div class="track-row__main">
      <div class="track-row__thumb">
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
          loading="lazy"
        />
        <CoverArtPlayOverlay playing={isCurrent && music.playing} />
      </div>
      <span class="track-row__text">
        <span class="track-row__title-row">
          <span class="track-row__title">{track.title}</span>
          {#if !versionList}
            <TrackQualityBadge {track} />
          {/if}
        </span>
        <span class="track-row__artist">{track.artist ?? "Unknown artist"}</span
        >
        {#if subtitle}
          <span class="track-row__subtitle">{subtitle}</span>
        {/if}
      </span>
    </div>

    {#if versionList}
      <button
        type="button"
        class="track-row__versions"
        onclick={(event) => {
          event.stopPropagation();
          toggleVersions(event);
        }}
        aria-expanded={versionsOpen}
        aria-label="{versionList.length} versions"
      >
        {versionList.length} versions
      </button>
    {/if}
  </div>

  <span class="track-row__duration">{formatDuration(track.duration)}</span>

  <div class="track-row__actions">
    <button
      type="button"
      class="track-row__btn"
      onclick={(event) => {
        event.stopPropagation();
        music.playNext(track);
      }}
      aria-label={`Play ${track.title} next`}
    >
      <MdiIcon name="playNext" size={16} />
    </button>

    <button
      type="button"
      class="track-row__btn"
      onclick={(event) => {
        event.stopPropagation();
        music.addToQueue(track);
      }}
      aria-label={`Add ${track.title} to queue`}
    >
      <MdiIcon name="queueAdd" size={16} />
    </button>

    <button
      type="button"
      class="track-row__btn"
      onclick={(event) => {
        event.stopPropagation();
        pickerOpen = true;
      }}
      aria-label="Add to playlist"
    >
      <MdiIcon name="plus" size={16} />
    </button>

    <FavoriteButton {track} size={16} class="track-row__btn" />
  </div>

  {#if versionsOpen && versionList}
    <div
      class="track-row__versions-menu"
      role="presentation"
      onclick={(event) => event.stopPropagation()}
    >
      {#each versionList as version (version.id)}
        <button
          type="button"
          class="track-row__version"
          class:track-row__version--active={music.currentTrack?.id ===
            version.id}
          onclick={() => handlePlayVersion(version)}
        >
          <TrackQualityBadge track={version} size="md" />
          <span class="track-row__version-duration"
            >{formatDuration(version.duration)}</span
          >
          {#if music.currentTrack?.id === version.id && music.playing}
            <NowPlayingBars />
          {:else}
            <MdiIcon name="play" size={14} />
          {/if}
        </button>
      {/each}
    </div>
  {/if}
</div>

{#if versionsOpen}
  <button
    type="button"
    class="track-row__versions-backdrop"
    aria-label="Close versions"
    onclick={closeVersions}
  ></button>
{/if}

<PlaylistPicker
  open={pickerOpen}
  tracks={[track]}
  onclose={() => (pickerOpen = false)}
/>

<style>
  .track-row {
    position: relative;
    display: grid;
    grid-template-columns: 2rem minmax(0, 1fr) auto auto;
    align-items: center;
    content-visibility: auto;
    contain-intrinsic-size: auto 3.875rem;
    gap: var(--jb-space-3);
    padding: var(--jb-space-2) var(--jb-space-3);
    border-radius: var(--jb-radius-md);
    color: var(--jb-text);
    transition: background var(--jb-transition);
    z-index: 0;
    cursor: pointer;
  }

  .track-row--versions-open {
    z-index: 2;
  }

  .track-row--compact {
    grid-template-columns: minmax(0, 1fr) auto auto;
  }

  .track-row__content {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
    min-width: 0;
  }

  .track-row--selected {
    background: color-mix(
      in srgb,
      var(--jb-music-accent-muted) 80%,
      transparent
    );
  }

  .track-row:hover,
  .track-row--active {
    background: var(--jb-surface-hover);
  }

  .track-row:active {
    background: var(--jb-surface-active);
  }

  .track-row:focus-visible {
    outline: none;
    box-shadow: var(--jb-focus-ring);
  }

  .track-row--selected:hover {
    background: var(--jb-music-accent-muted);
  }

  .track-row--playing .track-row__title {
    color: var(--jb-accent);
  }

  .track-row__check {
    display: grid;
    place-items: center;
    width: 1.25rem;
    height: 1.25rem;
    border: 2px solid var(--jb-border-strong);
    border-radius: var(--jb-radius-sm);
    background: transparent;
    color: white;
    cursor: pointer;
    padding: 0;
  }

  .track-row__check--on {
    background: var(--jb-accent);
    border-color: var(--jb-accent);
  }

  .track-row__index {
    text-align: center;
    font-size: 0.875rem;
    color: var(--jb-text-subtle);
    font-variant-numeric: tabular-nums;
  }

  .track-row__main {
    display: flex;
    align-items: center;
    gap: var(--jb-space-3);
    min-width: 0;
    pointer-events: none;
  }

  .track-row__thumb {
    position: relative;
    width: 2.75rem;
    height: 2.75rem;
    border-radius: var(--jb-radius-sm);
    flex-shrink: 0;
    overflow: hidden;
    box-shadow: var(--jb-shadow-sm);
    pointer-events: auto;
  }

  .track-row__thumb:hover :global(.cover-play-overlay),
  .track-row__thumb:focus-within :global(.cover-play-overlay),
  .track-row:hover .track-row__thumb :global(.cover-play-overlay) {
    opacity: 1;
  }

  .track-row__thumb :global(.cover-art) {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }

  .track-row__text {
    display: flex;
    flex-direction: column;
    gap: 0.125rem;
    min-width: 0;
  }

  .track-row__title-row {
    display: flex;
    align-items: center;
    gap: var(--jb-space-2);
    min-width: 0;
  }

  .track-row__title {
    font-weight: 600;
    font-size: 0.9375rem;
    color: var(--jb-text);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .track-row__versions {
    align-self: flex-start;
    border: 1px solid var(--jb-border);
    border-radius: var(--jb-radius-full);
    background: var(--jb-bg-muted);
    color: var(--jb-text-muted);
    font-size: 0.625rem;
    font-weight: 700;
    letter-spacing: 0.04em;
    text-transform: uppercase;
    padding: 0.125rem 0.4375rem;
    cursor: pointer;
  }

  .track-row__versions:hover {
    color: var(--jb-text);
    border-color: var(--jb-border-strong);
  }

  .track-row__versions-menu {
    grid-column: 1 / -1;
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
    margin-top: var(--jb-space-1);
    padding: var(--jb-space-2);
    border-radius: var(--jb-radius-md);
    background: var(--jb-bg-muted);
    border: 1px solid var(--jb-border);
  }

  .track-row__version {
    display: grid;
    grid-template-columns: auto 1fr auto;
    align-items: center;
    gap: var(--jb-space-3);
    width: 100%;
    padding: var(--jb-space-2) var(--jb-space-3);
    border: none;
    border-radius: var(--jb-radius-sm);
    background: transparent;
    color: inherit;
    text-align: left;
    cursor: pointer;
  }

  .track-row__version:hover,
  .track-row__version--active {
    background: var(--jb-surface-hover);
  }

  .track-row__version-duration {
    font-size: 0.8125rem;
    color: var(--jb-text-subtle);
    font-variant-numeric: tabular-nums;
  }

  .track-row__versions-backdrop {
    position: fixed;
    inset: 0;
    z-index: 1;
    border: none;
    background: transparent;
    cursor: default;
    padding: 0;
  }

  .track-row__artist {
    font-size: 0.8125rem;
    color: var(--jb-text-muted);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .track-row__subtitle {
    font-size: 0.75rem;
    color: var(--jb-text-subtle);
  }

  .track-row__duration {
    display: inline-flex;
    align-items: center;
    gap: var(--jb-space-2);
    font-size: 0.8125rem;
    color: var(--jb-text-subtle);
    font-variant-numeric: tabular-nums;
    padding-right: var(--jb-space-2);
  }

  .track-row__actions {
    display: flex;
    align-items: center;
    gap: var(--jb-space-1);
    opacity: 0;
    transition: opacity var(--jb-transition);
    pointer-events: auto;
  }

  .track-row:hover .track-row__actions,
  .track-row--active .track-row__actions,
  .track-row--selected .track-row__actions {
    opacity: 1;
  }

  .track-row__btn {
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
  }

  .track-row__btn:hover {
    background: var(--jb-surface-active);
    color: var(--jb-text);
  }

  :global(.track-row__btn.favorite-btn) {
    width: 2rem;
    height: 2rem;
  }

  @media (max-width: 640px) {
    .track-row__actions {
      opacity: 1;
    }

    .track-row__duration {
      display: none;
    }

    .track-row {
      grid-template-columns: 1.5rem minmax(0, 1fr) auto;
    }
  }
</style>
