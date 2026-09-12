<script lang="ts">
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";
  import SourceIcon from "$lib/components/ui/SourceIcon.svelte";
  import MusicBreadcrumbs from "$lib/components/music/MusicBreadcrumbs.svelte";
  import TrackRow from "$lib/components/music/TrackRow.svelte";
  import VirtualList from "$lib/components/ui/VirtualList.svelte";
  import Spinner from "$lib/components/ui/Spinner.svelte";
  import EmptyState from "$lib/components/ui/EmptyState.svelte";
  import LocalSearchBox from "$lib/components/ui/LocalSearchBox.svelte";
  import Input from "$lib/components/ui/Input.svelte";
  import Button from "$lib/components/ui/Button.svelte";
  import Link from "$lib/router/Link.svelte";
  import { music } from "$lib/config/music.svelte";
  import { sources } from "$lib/features/sources/store.svelte";
  import { filterByLocalSearch } from "$lib/utils/local-search";
  import { formatServerTimestamp } from "$lib/utils/format-date";
  import { formatPlaylistDuration } from "$lib/music/playlist-duration";
  import { stableItemKey } from "$lib/core/collection";
  import { toast } from "$lib/ui/toast.svelte";
  import { createAsyncPage } from "$lib/ui/async-page.svelte";
  import { confirmDialog } from "$lib/ui/confirm.svelte";
  import { exportPlaylistM3U } from "$lib/music/playlist-m3u";
  import { savePlaylistFilesToDevice } from "$lib/music/context-actions";
  import SharePlaylistDialog from "$lib/components/music/SharePlaylistDialog.svelte";
  import type { ServerPlaylist, SubsonicSong } from "$lib/subsonic";

  interface Props {
    playlistId: string;
  }

  let { playlistId }: Props = $props();

  let saving = $state(false);
  let searchQuery = $state("");
  let playlist = $state<ServerPlaylist | null>(null);
  let songs = $state.raw<SubsonicSong[]>([]);
  let renameValue = $state("");
  let editingName = $state(false);

  const canEdit = $derived(sources.hasSubsonicActive);

  const page = createAsyncPage({
    errorMessage: "Failed to load playlist",
    load: async () => {
      const id = playlistId;
      playlist = null;
      songs = [];
      if (!music.libraryReady) {
        throw new Error(music.error ?? "Not connected");
      }
      const result = await music.fetchServerPlaylist(id);
      if (!result) throw new Error("Playlist not found");
      return result;
    },
    apply: (result) => {
      playlist = result.playlist;
      songs = result.songs;
      renameValue = result.playlist.name;
    },
  });

  const filteredSongs = $derived(
    filterByLocalSearch(songs, searchQuery, (track) => [
      track.title,
      track.artist,
      track.album,
    ]),
  );

  const hasSearch = $derived(searchQuery.trim().length > 0);

  const breadcrumbItems = $derived(
    playlist
      ? [
          { label: "Playlists", href: "/music/playlists" },
          { label: playlist.name },
        ]
      : [{ label: "Playlists", href: "/music/playlists" }],
  );

  const createdLabel = $derived(formatServerTimestamp(playlist?.created));
  const changedLabel = $derived(formatServerTimestamp(playlist?.changed));
  const playlistDuration = $derived(
    formatPlaylistDuration(playlist) ??
      formatPlaylistDuration({
        tracks: songs.map((song) => ({
          durationMs: (song.duration ?? 0) * 1000,
        })),
      }),
  );

  function playAll() {
    if (filteredSongs.length === 0) return;
    music.playTracks(filteredSongs, 0);
  }

  function exportPlaylist() {
    if (!playlist) return;
    exportPlaylistM3U(filteredSongs, music.config, playlist.name);
    toast.success("Playlist exported");
  }

  let savingFiles = $state(false);
  let shareOpen = $state(false);

  async function downloadFiles() {
    if (!playlist || savingFiles) return;
    savingFiles = true;
    try {
      await savePlaylistFilesToDevice("server", playlistId, playlist.name);
    } finally {
      savingFiles = false;
    }
  }

  function songIndex(track: SubsonicSong): number {
    return songs.findIndex((song) => song.id === track.id);
  }

  async function withSave(task: () => Promise<void>) {
    if (saving) return;
    saving = true;
    try {
      await task();
      await page.reload();
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Playlist update failed",
      );
    } finally {
      saving = false;
    }
  }

  async function removeTrack(track: SubsonicSong) {
    const index = songIndex(track);
    if (index < 0) return;
    const ok = await confirmDialog.confirm({
      title: "Remove track",
      message: `Remove "${track.title}" from this playlist?`,
      confirmLabel: "Remove",
      danger: true,
    });
    if (!ok) return;
    await withSave(() => music.removeFromServerPlaylist(playlistId, index));
  }

  async function moveTrack(track: SubsonicSong, direction: -1 | 1) {
    const index = songIndex(track);
    const target = index + direction;
    if (index < 0 || target < 0 || target >= songs.length) return;
    const ids = songs.map((song) => song.id);
    await withSave(() =>
      music.moveServerPlaylistSong(playlistId, ids, index, target),
    );
  }

  async function saveRename() {
    const trimmed = renameValue.trim();
    if (!playlist || !trimmed || trimmed === playlist.name) {
      editingName = false;
      renameValue = playlist?.name ?? "";
      return;
    }
    await withSave(async () => {
      await music.renameServerPlaylist(playlistId, trimmed);
      editingName = false;
    });
  }

  async function deletePlaylist() {
    if (!playlist) return;
    const ok = await confirmDialog.confirm({
      title: "Delete server playlist",
      message: `Delete server playlist "${playlist.name}"? This cannot be undone.`,
      confirmLabel: "Delete playlist",
      danger: true,
    });
    if (!ok) return;
    saving = true;
    try {
      await music.deleteServerPlaylist(playlistId);
      toast.success("Playlist deleted");
      import("$lib/router/router.svelte").then(({ router }) => {
        router.navigate("/music/playlists");
      });
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to delete playlist",
      );
    } finally {
      saving = false;
    }
  }
</script>

<div class="server-playlist-page">
  <MusicBreadcrumbs items={breadcrumbItems} />

  {#if page.loading}
    <div class="server-playlist-page__loading"><Spinner /></div>
  {:else if page.error}
    <p class="server-playlist-page__error">{page.error}</p>
    <Link href="/music/playlists">Back to playlists</Link>
  {:else if playlist}
    <header class="server-playlist-header">
      <div>
        <p class="server-playlist-header__eyebrow">
          <SourceIcon kind="server" size={18} />
          Server playlist
        </p>
        {#if editingName && canEdit}
          <form
            class="server-playlist-header__rename"
            onsubmit={(e) => {
              e.preventDefault();
              void saveRename();
            }}
          >
            <Input bind:value={renameValue} disabled={saving} />
            <Button type="submit" disabled={saving || !renameValue.trim()}>
              Save
            </Button>
            <Button
              type="button"
              variant="ghost"
              disabled={saving}
              onclick={() => {
                editingName = false;
                renameValue = playlist?.name ?? "";
              }}
            >
              Cancel
            </Button>
          </form>
        {:else}
          <div class="server-playlist-header__title-row">
            <h1>{playlist.name}</h1>
            {#if canEdit}
              <button
                type="button"
                class="server-playlist-header__icon-btn"
                onclick={() => {
                  editingName = true;
                  renameValue = playlist?.name ?? "";
                }}
                aria-label="Rename playlist"
                disabled={saving}
              >
                <MdiIcon name="pencil" size={18} />
              </button>
            {/if}
          </div>
        {/if}
        <div class="server-playlist-header__meta">
          <span>{songs.length.toLocaleString()} tracks</span>
          {#if playlistDuration}
            <span>· {playlistDuration}</span>
          {/if}
          {#if playlist.songCount && playlist.songCount !== songs.length}
            <span>· {playlist.songCount.toLocaleString()} on server</span>
          {/if}
          {#if playlist.owner}
            <span>· Owner {playlist.owner}</span>
          {/if}
          {#if playlist.public}
            <span class="server-playlist-header__badge">Public</span>
          {:else}
            <span
              class="server-playlist-header__badge server-playlist-header__badge--private"
              >Private</span
            >
          {/if}
        </div>
        {#if createdLabel || changedLabel}
          <p class="server-playlist-header__dates">
            {#if createdLabel}
              <span>Created {createdLabel}</span>
            {/if}
            {#if changedLabel}
              <span>{createdLabel ? "· " : ""}Updated {changedLabel}</span>
            {/if}
          </p>
        {/if}
      </div>
      <div class="server-playlist-header__actions">
        {#if filteredSongs.length > 0}
          <button
            type="button"
            class="server-playlist-header__shuffle"
            onclick={exportPlaylist}
            disabled={saving}
          >
            <MdiIcon name="export" size={16} />
            Export M3U
          </button>
          <button
            type="button"
            class="server-playlist-header__play"
            onclick={playAll}
            disabled={saving}
          >
            <MdiIcon name="play" size={16} />
            {hasSearch ? "Play matches" : "Play all"}
          </button>
          <button
            type="button"
            class="server-playlist-header__shuffle"
            onclick={() => {
              music.shuffle = true;
              playAll();
            }}
            disabled={saving}
          >
            <MdiIcon name="shuffle" size={16} />
            Shuffle
          </button>
          <button
            type="button"
            class="server-playlist-header__shuffle"
            onclick={() => void downloadFiles()}
            disabled={saving || savingFiles}
          >
            <MdiIcon name="download" size={16} />
            {savingFiles ? "Downloading…" : "Download"}
          </button>
          <button
            type="button"
            class="server-playlist-header__shuffle"
            onclick={() => (shareOpen = true)}
            disabled={saving}
          >
            <MdiIcon name="share" size={16} />
            Share
          </button>
        {/if}
        {#if canEdit}
          <button
            type="button"
            class="server-playlist-header__delete"
            onclick={() => void deletePlaylist()}
            disabled={saving}
          >
            <MdiIcon name="trash2" size={16} />
            Delete
          </button>
        {/if}
      </div>
    </header>

    <LocalSearchBox
      bind:value={searchQuery}
      placeholder="Search tracks"
      disabled={songs.length === 0}
      resultCount={filteredSongs.length}
      totalCount={songs.length}
    />

    {#if songs.length === 0}
      <EmptyState
        title="Empty playlist"
        message={canEdit
          ? "Add tracks from albums or search using the + button."
          : "This server playlist has no tracks."}
        icon="listMusic"
      />
    {:else if filteredSongs.length === 0}
      <EmptyState
        title="No matches"
        message={`No tracks match "${searchQuery.trim()}".`}
        icon="search"
      />
    {:else}
      <div class="track-list-island">
        {#if filteredSongs.length >= 24}
          <VirtualList
            items={filteredSongs}
            itemHeight={58}
            scrollMode="document"
          >
            {#snippet children({ item: track, index })}
              <div class="track-list__row">
                <TrackRow
                  {track}
                  {index}
                  onplay={() => music.playTracks(filteredSongs, index)}
                />
                {#if canEdit}
                  <div class="track-list__actions">
                    <button
                      type="button"
                      class="track-list__action"
                      onclick={() => void moveTrack(track, -1)}
                      disabled={saving || songIndex(track) <= 0}
                      aria-label="Move up"
                    >
                      <MdiIcon name="chevronUp" size={16} />
                    </button>
                    <button
                      type="button"
                      class="track-list__action"
                      onclick={() => void moveTrack(track, 1)}
                      disabled={saving || songIndex(track) >= songs.length - 1}
                      aria-label="Move down"
                    >
                      <MdiIcon name="chevronDown" size={16} />
                    </button>
                    <button
                      type="button"
                      class="track-list__action track-list__action--danger"
                      onclick={() => void removeTrack(track)}
                      disabled={saving}
                      aria-label="Remove track"
                    >
                      <MdiIcon name="trash2" size={16} />
                    </button>
                  </div>
                {/if}
              </div>
            {/snippet}
          </VirtualList>
        {:else}
          {#each filteredSongs as track, i (stableItemKey(track.id, i, track.title))}
            <div class="track-list__row">
              <TrackRow
                {track}
                index={i}
                onplay={() => music.playTracks(filteredSongs, i)}
              />
              {#if canEdit}
                <div class="track-list__actions">
                  <button
                    type="button"
                    class="track-list__action"
                    onclick={() => void moveTrack(track, -1)}
                    disabled={saving || songIndex(track) <= 0}
                    aria-label="Move up"
                  >
                    <MdiIcon name="chevronUp" size={16} />
                  </button>
                  <button
                    type="button"
                    class="track-list__action"
                    onclick={() => void moveTrack(track, 1)}
                    disabled={saving || songIndex(track) >= songs.length - 1}
                    aria-label="Move down"
                  >
                    <MdiIcon name="chevronDown" size={16} />
                  </button>
                  <button
                    type="button"
                    class="track-list__action track-list__action--danger"
                    onclick={() => void removeTrack(track)}
                    disabled={saving}
                    aria-label="Remove track"
                  >
                    <MdiIcon name="trash2" size={16} />
                  </button>
                </div>
              {/if}
            </div>
          {/each}
        {/if}
      </div>
    {/if}
  {/if}
</div>

<SharePlaylistDialog
  open={shareOpen}
  {playlistId}
  playlistName={playlist?.name ?? "Playlist"}
  kind="server"
  onclose={() => (shareOpen = false)}
/>

<style>
  .server-playlist-page {
    display: flex;
    flex-direction: column;
    gap: var(--jb-page-gap);
  }

  .server-playlist-page__loading {
    display: grid;
    place-content: center;
    min-height: 12rem;
  }

  .server-playlist-page__error {
    color: var(--jb-danger);
  }

  .server-playlist-header {
    display: flex;
    flex-wrap: wrap;
    align-items: flex-end;
    justify-content: space-between;
    gap: var(--jb-space-4);
  }

  .server-playlist-header__eyebrow {
    display: inline-flex;
    align-items: center;
    gap: var(--jb-space-2);
    margin: 0 0 var(--jb-space-2);
    font-size: 0.8125rem;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.08em;
    color: var(--jb-text-subtle);
  }

  .server-playlist-header__title-row {
    display: flex;
    align-items: center;
    gap: var(--jb-space-2);
  }

  .server-playlist-header h1 {
    margin: 0 0 var(--jb-space-2);
    font-size: clamp(2rem, 4vw, 2.75rem);
    font-weight: 800;
  }

  .server-playlist-header__rename {
    display: flex;
    flex-wrap: wrap;
    gap: var(--jb-space-2);
    margin-bottom: var(--jb-space-2);
    max-width: 32rem;
  }

  .server-playlist-header__icon-btn {
    border: none;
    background: transparent;
    color: var(--jb-text-subtle);
    padding: var(--jb-space-2);
    border-radius: var(--jb-radius-sm);
    cursor: pointer;
  }

  .server-playlist-header__icon-btn:hover {
    color: var(--jb-text);
    background: var(--jb-surface-hover);
  }

  .server-playlist-header__meta {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.35rem 0.5rem;
    color: var(--jb-text-muted);
    font-size: 0.9375rem;
  }

  .server-playlist-header__badge {
    display: inline-flex;
    align-items: center;
    padding: 0.125rem 0.5rem;
    border-radius: var(--jb-radius-full);
    background: var(--jb-accent-muted);
    color: var(--jb-accent);
    font-size: 0.75rem;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }

  .server-playlist-header__badge--private {
    background: var(--jb-bg-muted);
    color: var(--jb-text-muted);
  }

  .server-playlist-header__dates {
    margin: var(--jb-space-2) 0 0;
    color: var(--jb-text-subtle);
    font-size: 0.8125rem;
  }

  .server-playlist-header__actions {
    display: flex;
    flex-wrap: wrap;
    gap: var(--jb-space-2);
  }

  .server-playlist-header__play,
  .server-playlist-header__shuffle,
  .server-playlist-header__delete {
    display: inline-flex;
    align-items: center;
    gap: var(--jb-space-2);
    padding: 0.5rem 1rem;
    border-radius: var(--jb-radius-full);
    font-weight: 700;
    font-size: 0.875rem;
    cursor: pointer;
  }

  .server-playlist-header__play {
    border: none;
    background: var(--jb-accent);
    color: white;
  }

  .server-playlist-header__shuffle {
    border: 1px solid var(--jb-border);
    background: var(--jb-surface);
    color: var(--jb-text);
  }

  .server-playlist-header__delete {
    border: 1px solid var(--jb-border);
    background: transparent;
    color: var(--jb-danger);
  }

  .server-playlist-header__play:disabled,
  .server-playlist-header__shuffle:disabled,
  .server-playlist-header__delete:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }

  .track-list__row {
    display: grid;
    grid-template-columns: 1fr auto;
    align-items: center;
    gap: var(--jb-space-2);
  }

  .track-list__actions {
    display: inline-flex;
    align-items: center;
    gap: 0.125rem;
  }

  .track-list__action {
    border: none;
    background: transparent;
    color: var(--jb-text-subtle);
    padding: var(--jb-space-2);
    border-radius: var(--jb-radius-sm);
    cursor: pointer;
  }

  .track-list__action:hover:not(:disabled) {
    color: var(--jb-text);
    background: var(--jb-surface-hover);
  }

  .track-list__action--danger:hover:not(:disabled) {
    color: var(--jb-danger);
  }

  .track-list__action:disabled {
    opacity: 0.35;
    cursor: not-allowed;
  }

  @media (max-width: 768px) {
    .server-playlist-header {
      flex-direction: column;
      align-items: stretch;
    }

    .server-playlist-header__actions {
      width: 100%;
      align-items: center;
    }

    .server-playlist-header__play,
    .server-playlist-header__shuffle,
    .server-playlist-header__delete {
      flex: 0 1 auto;
      justify-content: center;
      padding: 0.375rem 0.75rem;
      font-size: 0.8125rem;
      white-space: nowrap;
    }

    .track-list__row {
      grid-template-columns: 1fr;
      gap: var(--jb-space-1);
    }

    .track-list__actions {
      justify-content: flex-end;
      padding-right: var(--jb-space-1);
    }
  }
</style>
