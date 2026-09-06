<script lang="ts">
  import Link from "$lib/router/Link.svelte";
  import EnhancedCoverArt from "$lib/components/ui/EnhancedCoverArt.svelte";
  import CoverArtPlayOverlay from "$lib/components/music/CoverArtPlayOverlay.svelte";
  import { music } from "$lib/config/music.svelte";
  import {
    albumCoverPaletteKey,
    albumCoverSeed,
  } from "$lib/music/cover-art-fallback";
  import { coverArtUrl } from "$lib/subsonic";
  import type { SubsonicAlbum } from "$lib/subsonic";
  import { contextMenuPositionFromEvent } from "$lib/components/ui/context-menu";
  import AlbumContextMenu from "./AlbumContextMenu.svelte";
  import SourceBadge from "$lib/components/ui/SourceBadge.svelte";

  interface Props {
    album: SubsonicAlbum;
    size?: "sm" | "md" | "lg";
    hideable?: boolean;
  }

  let { album, size = "md", hideable = false }: Props = $props();
  let menu = $state<{ x: number; y: number } | null>(null);

  const image = $derived(
    album.coverArt
      ? coverArtUrl(music.config, album.coverArt, size === "lg" ? 400 : 280)
      : null,
  );

  async function playAlbum() {
    const detail = await music.library.getAlbum(album.id).catch(() => null);
    if (!detail?.songs.length) return;
    music.playAlbum(detail.songs, 0);
  }

  function onContextMenu(event: MouseEvent) {
    menu = contextMenuPositionFromEvent(event);
  }
</script>

{#if menu}
  <AlbumContextMenu
    {album}
    x={menu.x}
    y={menu.y}
    {hideable}
    onclose={() => (menu = null)}
  />
{/if}

<div
  class="album-card-wrap album-card-wrap--{size}"
  role="group"
  oncontextmenu={onContextMenu}
>
  <Link href="/music/album/{album.id}" class="album-card album-card--{size}">
    <div class="album-card__art">
      <EnhancedCoverArt
        kind="album"
        entity={{ ...album, name: album.name }}
        src={image}
        seed={albumCoverSeed(album)}
        paletteKey={albumCoverPaletteKey(album)}
        loading="lazy"
      />
      <button
        type="button"
        class="album-card__play"
        aria-label={`Play ${album.name}`}
        onclick={(event) => {
          event.preventDefault();
          event.stopPropagation();
          void playAlbum();
        }}
      >
        <CoverArtPlayOverlay />
      </button>
    </div>
    <div class="album-card__meta">
      <span class="album-card__title-row">
        <span class="album-card__title">{album.name}</span>
        <SourceBadge id={album.id} />
      </span>
      <span class="album-card__artist">{album.artist ?? "Unknown artist"}</span>
      {#if album.year}
        <span class="album-card__year">{album.year}</span>
      {/if}
    </div>
  </Link>
</div>

<style>
  .album-card-wrap {
    min-width: 0;
  }

  :global(a.album-card) {
    display: flex;
    flex-direction: column;
    gap: var(--jb-space-3);
    text-decoration: none;
    color: inherit;
    border-radius: var(--jb-radius-lg);
    transition: transform var(--jb-transition);
  }

  :global(a.album-card:hover) {
    transform: translateY(-2px);
  }

  :global(a.album-card:hover) .album-card__art {
    box-shadow: var(--jb-shadow-md);
  }

  :global(a.album-card:hover) .album-card__title {
    color: var(--jb-music-accent);
  }

  .album-card__art {
    position: relative;
    aspect-ratio: 1;
    border-radius: var(--jb-radius-lg);
    overflow: hidden;
    background: var(--jb-bg-muted);
    transition: box-shadow var(--jb-transition);
  }

  .album-card__art :global(.cover-art) {
    width: 100%;
    height: 100%;
  }

  .album-card__play {
    position: absolute;
    inset: 0;
    border: none;
    padding: 0;
    background: transparent;
    cursor: pointer;
  }

  .album-card__art:hover :global(.cover-play-overlay),
  .album-card__art:focus-within :global(.cover-play-overlay),
  :global(a.album-card:hover) .album-card__art :global(.cover-play-overlay) {
    opacity: 1;
  }

  .album-card__meta {
    display: flex;
    flex-direction: column;
    gap: 0.125rem;
    min-width: 0;
  }

  .album-card__title-row,
  .artist-card__name-row {
    display: flex;
    align-items: center;
    gap: var(--jb-space-2);
    min-width: 0;
  }

  .album-card__title {
    flex: 1;
    min-width: 0;
    font-weight: 700;
    color: var(--jb-text);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    transition: color var(--jb-transition);
  }

  .album-card__artist {
    font-size: 0.8125rem;
    color: var(--jb-text-muted);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .album-card__year {
    font-size: 0.75rem;
    color: var(--jb-text-subtle);
  }

  @media (prefers-reduced-motion: reduce) {
    :global(a.album-card:hover) {
      transform: none;
    }
  }
</style>
