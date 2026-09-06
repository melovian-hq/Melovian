<script lang="ts">
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";
  import Link from "$lib/router/Link.svelte";
  import PlaylistListCovers from "./PlaylistListCovers.svelte";
  import { localPlaylistCoverIds } from "$lib/music/playlist-covers";
  import {
    coverStackVariant,
    playlistHref,
    playlistMetaParts,
    type PlaylistItem,
    type PlaylistKind,
    type PlaylistViewMode,
  } from "$lib/music/playlist-display";
  import type { MusicPlaylist, ServerPlaylist } from "$lib/subsonic";
  import { contextMenuPositionFromEvent } from "$lib/components/ui/context-menu";
  import PlaylistContextMenu from "./PlaylistContextMenu.svelte";

  interface Props {
    playlist: PlaylistItem;
    kind: PlaylistKind;
    view: PlaylistViewMode;
    onExport: () => void;
    onDelete: () => void;
    onShare?: () => void;
  }

  let { playlist, kind, view, onExport, onDelete, onShare }: Props = $props();
  let menu = $state<{ x: number; y: number } | null>(null);

  function onContextMenu(event: MouseEvent) {
    menu = contextMenuPositionFromEvent(event);
  }

  const href = $derived(playlistHref(kind, playlist.id));
  const stackVariant = $derived(coverStackVariant(view));
  const enrichCovers = $derived(view !== "list");
  const metaParts = $derived(playlistMetaParts(playlist, kind));
  const localCovers = $derived(
    kind === "local" ? localPlaylistCoverIds(playlist as MusicPlaylist) : [],
  );
  const serverCover = $derived(
    kind === "server" ? (playlist as ServerPlaylist).coverArt : undefined,
  );
  const isSmart = $derived(
    kind === "local" && (playlist as MusicPlaylist).kind === "smart",
  );
</script>

<article
  class="playlist-tile playlist-tile--{view}"
  oncontextmenu={onContextMenu}
>
  <Link {href} class="playlist-tile__link">
    <div class="playlist-tile__art">
      <PlaylistListCovers
        {kind}
        playlistId={playlist.id}
        seed={playlist.name}
        coverArtIds={localCovers}
        coverArt={serverCover}
        variant={stackVariant}
        {enrichCovers}
      />
    </div>

    <div class="playlist-tile__body">
      <h3 class="playlist-tile__name">
        {playlist.name}
        {#if isSmart}
          <span class="playlist-tile__smart">Smart</span>
        {/if}
      </h3>
      <p class="playlist-tile__meta">
        {#each metaParts as part, index (part)}
          {#if index > 0}<span class="playlist-tile__dot">·</span>{/if}
          <span>{part}</span>
        {/each}
      </p>
    </div>
  </Link>

  <div class="playlist-tile__actions">
    <button
      type="button"
      class="playlist-tile__action"
      aria-label="Export playlist"
      onclick={onExport}
    >
      <MdiIcon name="export" size={16} />
    </button>
    <button
      type="button"
      class="playlist-tile__action playlist-tile__action--danger"
      aria-label="Delete playlist"
      onclick={onDelete}
    >
      <MdiIcon name="trash2" size={16} />
    </button>
  </div>
</article>

{#if menu}
  <PlaylistContextMenu
    playlistId={playlist.id}
    name={playlist.name}
    {kind}
    x={menu.x}
    y={menu.y}
    {onExport}
    {onDelete}
    onShare={onShare
      ? () => {
          menu = null;
          onShare();
        }
      : undefined}
    onclose={() => (menu = null)}
  />
{/if}

<style>
  .playlist-tile {
    position: relative;
    border-radius: var(--jb-radius-lg);
    background: var(--jb-surface);
    border: 1px solid var(--jb-border);
    overflow: hidden;
    transition:
      transform var(--jb-transition),
      box-shadow var(--jb-transition),
      border-color var(--jb-transition);
  }

  .playlist-tile--list:hover,
  .playlist-tile--grid:hover {
    border-color: color-mix(
      in srgb,
      var(--jb-music-accent) 28%,
      var(--jb-border)
    );
    box-shadow: var(--jb-shadow-md);
  }

  .playlist-tile--list {
    display: flex;
    align-items: stretch;
  }

  .playlist-tile--list:hover {
    transform: none;
    box-shadow: none;
  }

  .playlist-tile--grid:hover {
    transform: translateY(-2px);
  }

  .playlist-tile--card {
    background: transparent;
    border: none;
    overflow: visible;
  }

  :global(a.playlist-tile__link) {
    flex: 1;
    display: flex;
    min-width: 0;
    text-decoration: none;
    color: inherit;
  }

  .playlist-tile--list :global(a.playlist-tile__link) {
    align-items: center;
    gap: var(--jb-space-4);
    padding: var(--jb-space-4) var(--jb-space-5);
  }

  .playlist-tile--list :global(a.playlist-tile__link:hover) {
    background: var(--jb-surface-hover);
  }

  .playlist-tile--grid :global(a.playlist-tile__link),
  .playlist-tile--card :global(a.playlist-tile__link) {
    flex-direction: column;
  }

  .playlist-tile__art {
    position: relative;
    flex: 0 0 auto;
  }

  .playlist-tile--grid .playlist-tile__art {
    aspect-ratio: 1;
    width: 100%;
    display: grid;
    place-items: stretch;
    overflow: hidden;
    padding: 0;
    background: var(--jb-bg-muted);
  }

  .playlist-tile--card .playlist-tile__art {
    aspect-ratio: 1;
    width: 100%;
    display: grid;
    place-items: stretch;
    overflow: hidden;
    padding: 0;
    border-radius: var(--jb-radius-lg);
    background: var(--jb-bg-muted);
    box-shadow: var(--jb-shadow-md);
    transition:
      transform var(--jb-transition),
      box-shadow var(--jb-transition);
  }

  .playlist-tile--card:hover .playlist-tile__art,
  .playlist-tile--card:focus-within .playlist-tile__art {
    transform: translateY(-3px);
    box-shadow: var(--jb-shadow-lg);
  }

  .playlist-tile__body {
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 0.2rem;
  }

  .playlist-tile--grid .playlist-tile__body {
    padding: var(--jb-space-3) var(--jb-space-4) var(--jb-space-4);
    text-align: left;
    min-width: 0;
  }

  .playlist-tile--card :global(a.playlist-tile__link) {
    gap: var(--jb-space-3);
  }

  .playlist-tile--card .playlist-tile__body {
    padding: 0;
    text-align: left;
    background: transparent;
  }

  .playlist-tile__name {
    margin: 0;
    font-size: 0.98rem;
    font-weight: 700;
    line-height: 1.25;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .playlist-tile__smart {
    margin-left: 0.4rem;
    padding: 0.1rem 0.4rem;
    border-radius: 0.25rem;
    font-size: 0.7rem;
    font-weight: 600;
    letter-spacing: 0.02em;
    text-transform: uppercase;
    color: var(--jb-text-muted);
    background: color-mix(in srgb, var(--jb-text-muted) 16%, transparent);
    vertical-align: middle;
  }

  .playlist-tile--card .playlist-tile__name {
    font-size: 0.9375rem;
    transition: color var(--jb-transition);
  }

  .playlist-tile--card:hover .playlist-tile__name {
    color: var(--jb-music-accent);
  }

  .playlist-tile__meta {
    margin: 0;
    font-size: 0.8125rem;
    color: var(--jb-text-muted);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .playlist-tile__dot {
    margin: 0 0.28rem;
    opacity: 0.65;
  }

  .playlist-tile__actions {
    display: flex;
    align-items: center;
    flex-shrink: 0;
  }

  .playlist-tile--grid .playlist-tile__actions,
  .playlist-tile--card .playlist-tile__actions {
    position: absolute;
    z-index: 2;
    top: var(--jb-space-2);
    right: var(--jb-space-2);
    gap: 0.15rem;
    padding: 0.15rem;
    border-radius: var(--jb-radius-full);
    background: color-mix(in srgb, var(--jb-bg) 72%, transparent);
    border: 1px solid color-mix(in srgb, var(--jb-border) 80%, transparent);
    backdrop-filter: blur(10px);
    opacity: 0;
    transform: translateY(-4px);
    transition:
      opacity var(--jb-transition),
      transform var(--jb-transition);
  }

  .playlist-tile--grid:hover .playlist-tile__actions,
  .playlist-tile--grid:focus-within .playlist-tile__actions,
  .playlist-tile--card:hover .playlist-tile__actions,
  .playlist-tile--card:focus-within .playlist-tile__actions {
    opacity: 1;
    transform: translateY(0);
  }

  .playlist-tile__action {
    border: none;
    background: transparent;
    color: var(--jb-text-subtle);
    padding: var(--jb-space-3);
    cursor: pointer;
    border-radius: var(--jb-radius-md);
    transition: color var(--jb-transition);
  }

  .playlist-tile--grid .playlist-tile__action,
  .playlist-tile--card .playlist-tile__action {
    padding: 0.4rem;
  }

  .playlist-tile__action:hover {
    color: var(--jb-text);
  }

  .playlist-tile__action--danger:hover {
    color: var(--jb-danger);
  }
</style>
