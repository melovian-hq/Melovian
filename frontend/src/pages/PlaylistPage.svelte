<script lang="ts">
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";
  import MusicBreadcrumbs from "$lib/components/music/MusicBreadcrumbs.svelte";
  import TrackRow from "$lib/components/music/TrackRow.svelte";
  import VirtualList from "$lib/components/ui/VirtualList.svelte";
  import Button from "$lib/components/ui/Button.svelte";
  import Skeleton from "$lib/components/ui/Skeleton.svelte";
  import EmptyState from "$lib/components/ui/EmptyState.svelte";
  import LocalSearchBox from "$lib/components/ui/LocalSearchBox.svelte";
  import Link from "$lib/router/Link.svelte";
  import PlaylistContextMenu from "$lib/components/music/PlaylistContextMenu.svelte";
  import SharePlaylistDialog from "$lib/components/music/SharePlaylistDialog.svelte";
  import { contextMenuPositionFromEvent } from "$lib/components/ui/context-menu";
  import { music } from "$lib/config/music.svelte";
  import { filterByLocalSearch } from "$lib/utils/local-search";
  import { formatPlaylistDuration } from "$lib/music/playlist-duration";
  import { exportPlaylistM3U } from "$lib/music/playlist-m3u";
  import { toast } from "$lib/ui/toast.svelte";
  import { confirmDialog } from "$lib/ui/confirm.svelte";
  import { savePlaylistFilesToDevice } from "$lib/music/context-actions";
  import { stableItemKey } from "$lib/core/collection";
  import type { PlaylistTrack, SubsonicSong } from "$lib/subsonic";
  import { APP_NAME } from "$lib/brand";
  import { setPageMeta } from "$lib/seo/meta";
  import { createAsyncPage } from "$lib/ui/async-page.svelte";

  interface Props {
    playlistId: string;
  }

  let { playlistId }: Props = $props();

  let searchQuery = $state("");
  let refreshing = $state(false);
  let heroMenu = $state<{ x: number; y: number } | null>(null);
  let playlist = $state<import("$lib/subsonic/types").MusicPlaylist | null>(
    null,
  );

  const page = createAsyncPage({
    errorMessage: "Failed to load playlist",
    load: () => {
      const id = playlistId;
      playlist = null;
      if (!music.libraryReady) {
        throw new Error(music.error ?? "Not connected");
      }
      return import("$lib/music/api").then((m) => m.getPlaylist(id));
    },
    apply: (result) => {
      playlist = result;
    },
  });

  function trackToSong(track: PlaylistTrack): SubsonicSong {
    return {
      id: track.trackId,
      title: track.trackTitle,
      artist: track.artistName,
      album: track.albumTitle,
      albumId: track.albumId,
      coverArt: track.coverArtId,
      duration: Math.floor(track.durationMs / 1000),
    };
  }

  const filteredTracks = $derived(
    filterByLocalSearch(playlist?.tracks ?? [], searchQuery, (track) => [
      track.trackTitle,
      track.artistName,
      track.albumTitle,
    ]),
  );

  const filteredSongs = $derived(filteredTracks.map(trackToSong));

  const hasSearch = $derived(searchQuery.trim().length > 0);

  $effect(() => {
    if (!playlist) return;
    const name = playlist.name?.trim() || "Playlist";
    const count = playlist.tracks?.length ?? 0;
    setPageMeta({
      title: name,
      description:
        count > 0
          ? `${name} · ${count} track${count === 1 ? "" : "s"} in ${APP_NAME}.`
          : `${name} playlist in ${APP_NAME}.`,
    });
  });

  function playAll() {
    if (filteredSongs.length === 0) return;
    music.playTracks(filteredSongs, 0);
  }

  function addPlaylistToQueue() {
    if (filteredSongs.length === 0) return;
    music.addTracksToQueue(filteredSongs);
  }

  function playPlaylistNext() {
    if (filteredSongs.length === 0) return;
    music.playTracksNext(filteredSongs);
  }

  async function removeTrack(trackId: string) {
    const ok = await confirmDialog.confirm({
      title: "Remove track",
      message: "Remove this track from the playlist?",
      confirmLabel: "Remove",
      danger: true,
    });
    if (!ok) return;
    playlist = await import("$lib/music/api").then((m) =>
      m.removeTrackFromPlaylist(playlistId, trackId),
    );
    await music.refreshPlaylists();
  }

  function exportPlaylist() {
    if (!playlist) return;
    exportPlaylistM3U(filteredSongs, music.config, playlist.name);
    toast.success("Playlist exported");
  }

  const isSmart = $derived(playlist?.kind === "smart" && !!playlist?.rulesJson);

  let savingFiles = $state(false);
  let shareOpen = $state(false);

  async function refreshSmart() {
    if (!playlist || !isSmart || refreshing) return;
    refreshing = true;
    try {
      playlist = await music.refreshSmartPlaylist(playlistId);
      toast.success(
        playlist.trackCount === 0
          ? "Smart playlist refreshed (no matches)"
          : `Smart playlist refreshed (${playlist.trackCount} tracks)`,
      );
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to refresh smart playlist",
      );
    } finally {
      refreshing = false;
    }
  }

  async function downloadFiles() {
    if (!playlist || savingFiles) return;
    savingFiles = true;
    try {
      await savePlaylistFilesToDevice("local", playlistId, playlist.name);
    } finally {
      savingFiles = false;
    }
  }

  const playlistDuration = $derived(formatPlaylistDuration(playlist));

  const breadcrumbItems = $derived(
    playlist
      ? [
          { label: "Playlists", href: "/music/playlists" },
          { label: playlist.name },
        ]
      : [
          { label: "Playlists", href: "/music/playlists" },
          { label: "Playlist" },
        ],
  );
</script>

<div class="playlist-page">
  <MusicBreadcrumbs items={breadcrumbItems} />

  {#if page.loading}
    <div
      class="playlist-page__skeleton"
      role="status"
      aria-busy="true"
      aria-label="Loading playlist"
    >
      <Skeleton variant="hero" class="playlist-page__skeleton-hero" />
      <div class="playlist-page__skeleton-rows">
        {#each Array.from({ length: 8 }) as _, i (i)}
          <Skeleton variant="row" />
        {/each}
      </div>
    </div>
  {:else if page.error || !playlist}
    <EmptyState
      title="Could not load playlist"
      message={page.error ?? "Playlist not found"}
      icon="alertCircle"
    >
      {#snippet actions()}
        <Link href="/music/playlists" class="playlist-page__back"
          >Back to playlists</Link
        >
      {/snippet}
    </EmptyState>
  {:else}
    <header
      class="playlist-hero"
      role="group"
      oncontextmenu={(event) => {
        heroMenu = contextMenuPositionFromEvent(event);
      }}
    >
      <p class="playlist-hero__type">
        {isSmart ? "Smart playlist" : "Playlist"}
      </p>
      <h1>{playlist.name}</h1>
      <p class="playlist-hero__meta">
        {playlist.trackCount ?? playlist.tracks?.length ?? 0} tracks{#if playlistDuration}
          · {playlistDuration}{/if}
      </p>
      <div class="playlist-hero__actions">
        {#if playlist.tracks && playlist.tracks.length > 0}
          <Button size="sm" onclick={playAll}>
            <MdiIcon name="play" size={18} />
            {hasSearch ? "Play matches" : "Play all"}
          </Button>
          <Button size="sm" variant="surface" onclick={playPlaylistNext}>
            <MdiIcon name="playNext" size={16} />
            Play next
          </Button>
          <Button size="sm" variant="surface" onclick={addPlaylistToQueue}>
            <MdiIcon name="queueAdd" size={16} />
            Add to queue
          </Button>
          <Button size="sm" variant="surface" onclick={exportPlaylist}>
            <MdiIcon name="export" size={16} />
            Export M3U
          </Button>
        {/if}
        {#if isSmart}
          <Button
            size="sm"
            variant="surface"
            onclick={refreshSmart}
            disabled={refreshing}
          >
            <MdiIcon name="refresh" size={16} />
            {refreshing ? "Refreshing…" : "Refresh"}
          </Button>
        {/if}
        {#if playlist.tracks && playlist.tracks.length > 0}
          <Button
            size="sm"
            variant="surface"
            onclick={() => void downloadFiles()}
            disabled={savingFiles}
          >
            <MdiIcon name="download" size={16} />
            {savingFiles ? "Downloading…" : "Download"}
          </Button>
        {/if}
        <Button size="sm" variant="surface" onclick={() => (shareOpen = true)}>
          <MdiIcon name="share" size={16} />
          Share
        </Button>
      </div>
    </header>

    <LocalSearchBox
      bind:value={searchQuery}
      placeholder="Search playlist tracks"
      disabled={page.loading && !playlist}
      resultCount={filteredTracks.length}
      totalCount={playlist.tracks?.length ?? 0}
    />

    {#if playlist.tracks && playlist.tracks.length > 0}
      {#if filteredTracks.length === 0}
        <EmptyState
          title="No matches"
          message={`No tracks match "${searchQuery.trim()}".`}
          icon="search"
        />
      {:else}
        <div class="playlist-tracks">
          {#if filteredTracks.length >= 24}
            <VirtualList
              items={filteredTracks}
              itemHeight={58}
              scrollMode="document"
            >
              {#snippet children({ item: track, index })}
                <div class="playlist-tracks__row">
                  <TrackRow
                    track={trackToSong(track)}
                    {index}
                    onplay={() => music.playTracks(filteredSongs, index)}
                  />
                  <button
                    type="button"
                    class="playlist-tracks__remove"
                    onclick={() => void removeTrack(track.trackId)}
                    aria-label="Remove track"
                  >
                    <MdiIcon name="trash2" size={16} />
                  </button>
                </div>
              {/snippet}
            </VirtualList>
          {:else}
            {#each filteredTracks as track, i (stableItemKey(track.trackId, i, track.trackTitle))}
              <div class="playlist-tracks__row">
                <TrackRow
                  track={trackToSong(track)}
                  index={i}
                  onplay={() => music.playTracks(filteredSongs, i)}
                />
                <button
                  type="button"
                  class="playlist-tracks__remove"
                  onclick={() => void removeTrack(track.trackId)}
                  aria-label="Remove track"
                >
                  <MdiIcon name="trash2" size={16} />
                </button>
              </div>
            {/each}
          {/if}
        </div>
      {/if}
    {:else}
      <EmptyState
        title="No tracks yet"
        message="Add songs from any album using the + button."
        icon="listMusic"
      />
    {/if}
  {/if}
</div>

{#if heroMenu && playlist}
  <PlaylistContextMenu
    playlistId={playlist.id}
    name={playlist.name}
    kind="local"
    x={heroMenu.x}
    y={heroMenu.y}
    onExport={exportPlaylist}
    onShare={() => {
      heroMenu = null;
      shareOpen = true;
    }}
    onclose={() => (heroMenu = null)}
  />
{/if}

<SharePlaylistDialog
  open={shareOpen}
  {playlistId}
  playlistName={playlist?.name ?? "Playlist"}
  kind="local"
  onclose={() => (shareOpen = false)}
/>

<style>
  .playlist-page {
    max-width: var(--jb-content-narrow);
  }

  .playlist-page__skeleton {
    display: flex;
    flex-direction: column;
    gap: var(--jb-space-6);
  }

  :global(.playlist-page__skeleton-hero) {
    min-height: 8rem;
  }

  .playlist-page__skeleton-rows {
    display: flex;
    flex-direction: column;
    gap: var(--jb-space-2);
  }

  :global(a.playlist-page__back) {
    color: var(--jb-music-accent);
    font-weight: 600;
    text-decoration: none;
  }

  .playlist-hero {
    margin-bottom: var(--jb-space-8);
  }

  .playlist-hero__type {
    margin: 0 0 var(--jb-space-2);
    font-size: 0.75rem;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.1em;
    color: var(--jb-music-accent);
  }

  .playlist-hero h1 {
    margin: 0 0 var(--jb-space-2);
    font-size: clamp(1.75rem, 4vw, 2.5rem);
    font-weight: 800;
  }

  .playlist-hero__meta {
    margin: 0 0 var(--jb-space-5);
    color: var(--jb-text-muted);
  }

  .playlist-hero__actions {
    display: flex;
    flex-wrap: wrap;
    gap: var(--jb-space-2);
  }

  .playlist-page :global(.local-search-box) {
    margin-bottom: var(--jb-space-6);
  }

  .playlist-tracks {
    display: flex;
    flex-direction: column;
    gap: var(--jb-space-1);
  }

  .playlist-tracks__row {
    display: grid;
    grid-template-columns: 1fr auto;
    align-items: center;
  }

  .playlist-tracks__remove {
    border: none;
    background: transparent;
    color: var(--jb-text-subtle);
    padding: var(--jb-space-2);
    cursor: pointer;
    border-radius: var(--jb-radius-sm);
  }

  .playlist-tracks__remove:hover {
    color: var(--jb-danger);
    background: var(--jb-surface-hover);
  }

  @media (max-width: 768px) {
    .playlist-page {
      max-width: none;
    }

    .playlist-hero__actions {
      width: 100%;
      align-items: center;
    }

    .playlist-hero__actions :global(.btn) {
      flex: 0 1 auto;
      white-space: nowrap;
    }
  }
</style>
