<script lang="ts">
  import Link from "$lib/router/Link.svelte";
  import EnhancedCoverArt from "$lib/components/ui/EnhancedCoverArt.svelte";
  import { music } from "$lib/config/music.svelte";
  import { resolveMediaUrl } from "$lib/config/runtime";
  import {
    artistCoverPaletteKey,
    artistCoverSeed,
  } from "$lib/music/cover-art-fallback";
  import {
    prefetchArtistInfo,
    resolveServerArtistArtUrl,
  } from "$lib/music/artist-media";
  import type { SubsonicArtist } from "$lib/subsonic";
  import { contextMenuPositionFromEvent } from "$lib/components/ui/context-menu";
  import ArtistContextMenu from "./ArtistContextMenu.svelte";
  import SourceBadge from "$lib/components/ui/SourceBadge.svelte";

  interface Props {
    artist: SubsonicArtist;
    size?: "sm" | "md";
    hideable?: boolean;
  }

  let { artist, size = "md", hideable = false }: Props = $props();
  let menu = $state<{ x: number; y: number } | null>(null);

  function onContextMenu(event: MouseEvent) {
    menu = contextMenuPositionFromEvent(event);
  }

  const image = $derived(
    resolveServerArtistArtUrl(
      music.config,
      artist,
      size === "sm" ? 160 : 240,
      resolveMediaUrl,
    ),
  );
</script>

{#if menu}
  <ArtistContextMenu
    {artist}
    x={menu.x}
    y={menu.y}
    {hideable}
    onclose={() => (menu = null)}
  />
{/if}

<div class="artist-card-wrap" role="group" oncontextmenu={onContextMenu}>
  <Link
    href="/music/artist/{artist.id}"
    class="artist-card artist-card--{size}"
    onmouseenter={() => prefetchArtistInfo(music.library, artist.id)}
  >
    <div class="artist-card__art">
      <EnhancedCoverArt
        kind="artist"
        entity={artist}
        src={image}
        seed={artistCoverSeed(artist)}
        paletteKey={artistCoverPaletteKey(artist)}
        loading="lazy"
      />
    </div>
    <div class="artist-card__meta">
      <span class="artist-card__name-row">
        <span class="artist-card__name">{artist.name}</span>
        <SourceBadge id={artist.id} />
      </span>
      {#if artist.albumCount}
        <span class="artist-card__count"
          >{artist.albumCount.toLocaleString()} albums</span
        >
      {/if}
    </div>
  </Link>
</div>

<style>
  .artist-card-wrap {
    min-width: 0;
  }

  :global(a.artist-card) {
    display: flex;
    flex-direction: column;
    gap: var(--jb-space-3);
    text-decoration: none;
    color: inherit;
    contain: layout style;
    border-radius: var(--jb-radius-lg);
    transition: transform var(--jb-transition);
  }

  :global(a.artist-card:hover) {
    transform: translateY(-2px);
  }

  :global(a.artist-card:hover) .artist-card__art {
    box-shadow: var(--jb-shadow-md);
  }

  :global(a.artist-card:hover) .artist-card__name {
    color: var(--jb-music-accent);
  }

  .artist-card__art {
    aspect-ratio: 1;
    border-radius: 50%;
    overflow: hidden;
    background: var(--jb-bg-muted);
    transition: box-shadow var(--jb-transition);
  }

  .artist-card__art :global(.cover-art) {
    width: 100%;
    height: 100%;
  }

  .artist-card__meta {
    display: flex;
    flex-direction: column;
    gap: 0.125rem;
    min-width: 0;
    text-align: center;
  }

  .artist-card__name-row {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: var(--jb-space-2);
    min-width: 0;
    width: 100%;
  }

  .artist-card__name {
    flex: 1;
    min-width: 0;
    font-weight: 650;
    font-size: 0.9375rem;
    color: var(--jb-text);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    transition: color var(--jb-transition);
  }

  :global(a.artist-card--sm) .artist-card__name {
    font-size: 0.875rem;
  }

  .artist-card__count {
    font-size: 0.8125rem;
    color: var(--jb-text-muted);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  @media (prefers-reduced-motion: reduce) {
    :global(a.artist-card:hover) {
      transform: none;
    }
  }
</style>
