<script lang="ts">
  import Link from "$lib/router/Link.svelte";
  import PlaylistListCovers from "./PlaylistListCovers.svelte";
  import PlaylistContextMenu from "./PlaylistContextMenu.svelte";
  import { contextMenuPositionFromEvent } from "$lib/components/ui/context-menu";
  import type { HomePlaylistRef } from "$lib/music/home-feed";

  interface Props {
    playlist: HomePlaylistRef;
    hideable?: boolean;
  }

  let { playlist, hideable = false }: Props = $props();
  let menu = $state<{ x: number; y: number } | null>(null);

  function onContextMenu(event: MouseEvent) {
    menu = contextMenuPositionFromEvent(event);
  }
</script>

{#if menu}
  <PlaylistContextMenu
    playlistId={playlist.playlistId}
    name={playlist.name}
    kind={playlist.kind}
    x={menu.x}
    y={menu.y}
    {hideable}
    onclose={() => (menu = null)}
  />
{/if}

<div class="home-playlist-wrap" role="group" oncontextmenu={onContextMenu}>
  <Link href={playlist.href} class="home-playlist">
    <div class="home-playlist__art">
      <PlaylistListCovers
        kind={playlist.kind}
        playlistId={playlist.playlistId}
        seed={playlist.name}
        coverArtIds={playlist.coverArtIds ?? []}
        coverArt={playlist.coverArtId}
        variant="card"
        enrichCovers
      />
    </div>
    <span class="home-playlist__copy">
      <span class="home-playlist__name">{playlist.name}</span>
      <span class="home-playlist__meta">
        {playlist.trackCount.toLocaleString()} tracks
      </span>
    </span>
  </Link>
</div>

<style>
  .home-playlist-wrap {
    min-width: 0;
  }

  :global(a.home-playlist) {
    display: flex;
    flex-direction: column;
    gap: var(--jb-space-3);
    min-width: 0;
    text-decoration: none;
    color: inherit;
    border-radius: var(--jb-radius-lg);
    transition: transform var(--jb-transition);
  }

  :global(a.home-playlist:hover) {
    transform: translateY(-2px);
  }

  :global(a.home-playlist:hover) .home-playlist__art {
    box-shadow: var(--jb-shadow-md);
  }

  :global(a.home-playlist:hover) .home-playlist__name {
    color: var(--jb-music-accent);
  }

  .home-playlist__art {
    aspect-ratio: 1;
    display: grid;
    place-items: stretch;
    border-radius: var(--jb-radius-lg);
    overflow: hidden;
    background: var(--jb-bg-muted);
    transition: box-shadow var(--jb-transition);
  }

  .home-playlist__copy {
    display: flex;
    flex-direction: column;
    gap: 0.125rem;
    min-width: 0;
  }

  .home-playlist__name {
    font-weight: 650;
    font-size: 0.9375rem;
    color: var(--jb-text);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    transition: color var(--jb-transition);
  }

  .home-playlist__meta {
    font-size: 0.8125rem;
    color: var(--jb-text-muted);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  @media (prefers-reduced-motion: reduce) {
    :global(a.home-playlist:hover) {
      transform: none;
    }
  }
</style>
