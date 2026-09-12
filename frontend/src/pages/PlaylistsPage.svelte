<script lang="ts">
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";
  import SourceIcon from "$lib/components/ui/SourceIcon.svelte";
  import MusicBreadcrumbs from "$lib/components/music/MusicBreadcrumbs.svelte";
  import PlaylistTile from "$lib/components/music/PlaylistTile.svelte";
  import SmartPlaylistCreator from "$lib/components/music/SmartPlaylistCreator.svelte";
  import Button from "$lib/components/ui/Button.svelte";
  import Skeleton from "$lib/components/ui/Skeleton.svelte";
  import VirtualList from "$lib/components/ui/VirtualList.svelte";
  import LocalSearchBox from "$lib/components/ui/LocalSearchBox.svelte";
  import EmptyState from "$lib/components/ui/EmptyState.svelte";
  import LibraryUnavailable from "$lib/components/music/LibraryUnavailable.svelte";
  import { music } from "$lib/config/music.svelte";
  import { sources } from "$lib/features/sources/store.svelte";
  import { libraryUnavailable } from "$lib/music/library-gate";
  import { filterByLocalSearch } from "$lib/utils/local-search";
  import * as musicApi from "$lib/music/api";
  import {
    exportPlaylistM3U,
    matchM3UEntryToSong,
    parseM3U,
  } from "$lib/music/playlist-m3u";
  import {
    loadPlaylistView,
    PLAYLIST_VIEW_KEY,
    type PlaylistViewMode,
  } from "$lib/music/playlist-display";
  import { toast } from "$lib/ui/toast.svelte";
  import { confirmDialog } from "$lib/ui/confirm.svelte";
  import ContextMenu from "$lib/components/ui/ContextMenu.svelte";
  import {
    contextMenuPositionFromEvent,
    type ContextMenuEntry,
  } from "$lib/components/ui/context-menu";
  import SharePlaylistDialog from "$lib/components/music/SharePlaylistDialog.svelte";
  import { auth } from "$lib/features/auth/store.svelte";
  import Link from "$lib/router/Link.svelte";
  import type { MusicPlaylist, ServerPlaylist } from "$lib/subsonic";
  import type { MusicShare } from "$lib/music/api";
  import type { SmartPlaylistDraft } from "$lib/music/smart-playlist/types";
  import { APP_NAME, StorageKeys } from "$lib/brand";
  import { Tabs } from "bits-ui";

  type PlaylistKind = "server" | "local";

  const VIRTUAL_THRESHOLD = 24;
  const PLAYLIST_ROW_HEIGHT = 84;
  const PLAYLIST_KIND_KEY = StorageKeys.playlistKind;
  const VIEW_OPTIONS: {
    id: PlaylistViewMode;
    label: string;
    icon: "viewList" | "viewGrid" | "viewCard";
  }[] = [
    { id: "list", label: "List", icon: "viewList" },
    { id: "grid", label: "Grid", icon: "viewGrid" },
    { id: "card", label: "Cards", icon: "viewCard" },
  ];

  let loading = $state(true);
  let newName = $state("");
  let creating = $state(false);
  let importing = $state(false);
  let searchQuery = $state("");
  let importInput: HTMLInputElement | undefined = $state();
  let smartPlaylistOpen = $state(false);
  let pageMenu = $state<{ x: number; y: number } | null>(null);
  let smartPlaylistSupported = $state(false);
  let smartPlaylistSupportReason = $state("");
  let shareTarget = $state<{
    id: string;
    name: string;
    kind: PlaylistKind;
  } | null>(null);
  let inboxShares = $state<MusicShare[]>([]);

  const canUseServer = $derived(sources.hasSubsonicActive);
  const canUseLocal = $derived(
    sources.hasLocalActive || music.playlists.length > 0,
  );
  const showKindToggle = $derived(canUseServer && canUseLocal);
  let playlistKind = $state<PlaylistKind>("server");
  const showSmartPlaylist = $derived(
    music.libraryReady &&
      ((playlistKind === "local" && canUseLocal) ||
        (playlistKind === "server" && canUseServer && smartPlaylistSupported)),
  );

  function defaultPlaylistKind(): PlaylistKind {
    if (canUseServer) return "server";
    return "local";
  }

  function loadSavedKind(): PlaylistKind | null {
    try {
      const saved = localStorage.getItem(PLAYLIST_KIND_KEY);
      if (saved === "server" || saved === "local") return saved;
    } catch {
      // ignore
    }
    return null;
  }

  let playlistView = $state<PlaylistViewMode>(loadPlaylistView());

  $effect(() => {
    const saved = loadSavedKind();
    if (saved === "server" && canUseServer) {
      playlistKind = "server";
      return;
    }
    if (saved === "local" && canUseLocal) {
      playlistKind = "local";
      return;
    }
    playlistKind = defaultPlaylistKind();
  });

  $effect(() => {
    if (!showKindToggle) return;
    try {
      localStorage.setItem(PLAYLIST_KIND_KEY, playlistKind);
    } catch {
      // ignore
    }
  });

  $effect(() => {
    if (!canUseServer || playlistKind !== "server") {
      smartPlaylistSupported = false;
      smartPlaylistSupportReason = "";
      return;
    }
    let cancelled = false;
    void musicApi.getSmartPlaylistSupport().then((support) => {
      if (cancelled) return;
      smartPlaylistSupported = support.supported;
      smartPlaylistSupportReason = support.reason ?? "";
    });
    return () => {
      cancelled = true;
    };
  });

  $effect(() => {
    if (!auth.enabled || !auth.authenticated) {
      inboxShares = [];
      return;
    }
    let cancelled = false;
    void musicApi
      .listShareInbox()
      .then((items) => {
        if (!cancelled) inboxShares = items;
      })
      .catch(() => {
        if (!cancelled) inboxShares = [];
      });
    return () => {
      cancelled = true;
    };
  });

  const filteredPlaylists = $derived(
    filterByLocalSearch(music.playlists, searchQuery, (pl) => [pl.name]),
  );

  const filteredServerPlaylists = $derived(
    filterByLocalSearch(music.serverPlaylists, searchQuery, (pl) => [pl.name]),
  );

  const activePlaylists = $derived(
    playlistKind === "server" ? filteredServerPlaylists : filteredPlaylists,
  );

  const totalPlaylistCount = $derived(
    playlistKind === "server"
      ? music.serverPlaylists.length
      : music.playlists.length,
  );

  const useVirtual = $derived(
    playlistView === "list" && activePlaylists.length >= VIRTUAL_THRESHOLD,
  );
  const hasSearch = $derived(searchQuery.trim().length > 0);
  const unavailable = $derived(libraryUnavailable());
  const showInitialLoading = $derived(
    !unavailable && loading && totalPlaylistCount === 0 && !hasSearch,
  );

  let playlistsLoadedRevision = -1;

  $effect(() => {
    if (unavailable) {
      loading = false;
      return;
    }
    if (!music.libraryReady) {
      loading = music.loading;
      return;
    }
    const revision = music.libraryRevision;
    if (playlistsLoadedRevision === revision && totalPlaylistCount > 0) {
      loading = false;
      return;
    }
    const showBlockingLoad = totalPlaylistCount === 0;
    if (showBlockingLoad) loading = true;
    const refreshTasks =
      playlistKind === "server"
        ? [music.refreshServerPlaylists()]
        : [music.refreshPlaylists()];
    void Promise.all(refreshTasks).finally(() => {
      playlistsLoadedRevision = revision;
      loading = false;
    });
  });

  $effect(() => {
    try {
      localStorage.setItem(PLAYLIST_VIEW_KEY, playlistView);
    } catch {
      // ignore
    }
  });

  async function createSmartPlaylist(draft: SmartPlaylistDraft) {
    await music.createSmartPlaylist(draft, playlistKind);
    toast.success(`Smart playlist "${draft.name.trim()}" created`);
  }

  async function createPlaylist() {
    const name = newName.trim();
    if (!name) return;
    creating = true;
    try {
      if (playlistKind === "server") {
        await music.createServerPlaylist(name);
        toast.success("Server playlist created");
      } else {
        await music.createPlaylist(name);
        toast.success("Local playlist created");
      }
      newName = "";
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to create playlist",
      );
    } finally {
      creating = false;
    }
  }

  async function deleteLocalPlaylist(id: string) {
    await musicApi.deletePlaylist(id);
    await music.refreshPlaylists();
  }

  async function deleteServerPlaylist(id: string, name: string) {
    const ok = await confirmDialog.confirm({
      title: "Delete server playlist",
      message: `Delete server playlist "${name}"? This cannot be undone.`,
      confirmLabel: "Delete playlist",
      danger: true,
    });
    if (!ok) return;
    try {
      await music.deleteServerPlaylist(id);
      toast.success("Server playlist deleted");
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to delete server playlist",
      );
    }
  }

  async function exportLocalPlaylist(pl: MusicPlaylist) {
    const full = await musicApi.getPlaylist(pl.id);
    const tracks = (full.tracks ?? []).map((track) => ({
      id: track.trackId,
      title: track.trackTitle,
      artist: track.artistName,
      album: track.albumTitle,
      albumId: track.albumId,
      coverArt: track.coverArtId,
      duration: Math.floor(track.durationMs / 1000),
    }));
    exportPlaylistM3U(tracks, music.config, full.name);
    toast.success("Playlist exported");
  }

  async function exportServerPlaylist(pl: ServerPlaylist) {
    const result = await music.fetchServerPlaylist(pl.id);
    if (!result) {
      toast.error("Failed to load playlist for export");
      return;
    }
    exportPlaylistM3U(result.songs, music.config, result.playlist.name);
    toast.success("Playlist exported");
  }

  async function importPlaylistFile(file: File) {
    importing = true;
    try {
      const text = await file.text();
      const parsed = parseM3U(text);
      const playlistName =
        parsed.name?.trim() ||
        file.name.replace(/\.m3u8?$/i, "").trim() ||
        "Imported playlist";

      const matched: import("$lib/subsonic").SubsonicSong[] = [];
      const seen = new Set<string>();

      for (const entry of parsed.entries) {
        if (entry.trackId) {
          const byId = await music.library
            .getSong(entry.trackId)
            .catch(() => null);
          if (byId && !seen.has(byId.id)) {
            seen.add(byId.id);
            matched.push(byId);
            continue;
          }
        }

        const query = entry.artist
          ? `${entry.artist} ${entry.title}`
          : entry.title;
        const result = await music.library.search3(query, 12).catch(() => ({
          songs: [] as import("$lib/subsonic").SubsonicSong[],
        }));
        const found = matchM3UEntryToSong(entry, result.songs);
        if (found && !seen.has(found.id)) {
          seen.add(found.id);
          matched.push(found);
        }
      }

      if (matched.length === 0) {
        toast.error("No tracks from this playlist matched your library");
        return;
      }

      if (playlistKind === "server" && canUseServer) {
        await music.createServerPlaylist(
          playlistName,
          matched.map((song) => song.id),
        );
      } else {
        const created = await musicApi.createPlaylist(playlistName);
        for (const song of matched) {
          await musicApi.addTrackToPlaylist(created.id, {
            trackId: song.id,
            trackTitle: song.title,
            artistName: song.artist ?? "",
            albumId: song.albumId ?? "",
            albumTitle: song.album ?? "",
            durationMs: (song.duration ?? 0) * 1000,
            coverArtId: song.coverArt ?? song.albumId ?? song.id,
          });
        }
        await music.refreshPlaylists();
      }

      const skipped = parsed.entries.length - matched.length;
      toast.success(
        skipped > 0
          ? `Imported ${matched.length} tracks (${skipped} unmatched)`
          : `Imported ${matched.length} tracks`,
      );
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to import playlist",
      );
    } finally {
      importing = false;
      if (importInput) importInput.value = "";
    }
  }

  function onImportSelected(event: Event) {
    const input = event.currentTarget as HTMLInputElement;
    const file = input.files?.[0];
    if (!file) return;
    void importPlaylistFile(file);
  }

  function onPageContextMenu(event: MouseEvent) {
    pageMenu = contextMenuPositionFromEvent(event);
  }

  const pageMenuItems = $derived.by((): ContextMenuEntry[] => {
    const items: ContextMenuEntry[] = [
      {
        id: "import",
        label: "Import M3U",
        icon: "upload",
        disabled: importing,
        onclick: () => importInput?.click(),
      },
    ];
    if (showSmartPlaylist) {
      items.push({
        id: "smart",
        label: "New smart playlist",
        icon: "slidersHorizontal",
        onclick: () => (smartPlaylistOpen = true),
      });
    }
    return items;
  });
</script>

<div
  class="playlists-page"
  class:playlists-page--list={playlistView === "list"}
  class:playlists-page--grid={playlistView === "grid"}
  class:playlists-page--card={playlistView === "card"}
  role="group"
  oncontextmenu={onPageContextMenu}
>
  <MusicBreadcrumbs items={[{ label: "Playlists" }]} />

  <header class="playlists-header">
    <div>
      <p class="playlists-header__eyebrow">
        <MdiIcon name="listMusic" size={18} />
        Your library
      </p>
      <h1>Playlists</h1>
      <p class="playlists-header__sub">
        {#if showKindToggle}
          Switch between server and local playlists. Imports match tracks from
          your active library.
        {:else if canUseServer}
          Server playlists sync from your Subsonic or Navidrome server.
        {:else}
          Local playlists are stored in {APP_NAME} on this device.
        {/if}
      </p>
    </div>
    <div class="playlists-header__tools">
      <input
        bind:this={importInput}
        type="file"
        accept=".m3u,.m3u8,audio/x-mpegurl"
        class="playlists-header__import-input"
        onchange={onImportSelected}
      />
      <Button
        variant="surface"
        disabled={importing}
        onclick={() => importInput?.click()}
      >
        <MdiIcon name="upload" size={16} />
        Import M3U
      </Button>
    </div>
  </header>

  {#if showKindToggle}
    <Tabs.Root
      style="display: contents"
      bind:value={
        () => playlistKind,
        (value) => {
          playlistKind = value as PlaylistKind;
        }
      }
    >
      <Tabs.List class="playlists-kind" aria-label="Playlist source">
        <Tabs.Trigger value="server" class="playlists-kind__btn">
          <SourceIcon kind="server" size={16} />
          Server
        </Tabs.Trigger>
        <Tabs.Trigger value="local" class="playlists-kind__btn">
          <MdiIcon name="folderOpen" size={16} />
          Local
        </Tabs.Trigger>
      </Tabs.List>
    </Tabs.Root>
  {/if}

  <LocalSearchBox
    bind:value={searchQuery}
    placeholder="Search playlists"
    disabled={loading && totalPlaylistCount === 0}
    resultCount={activePlaylists.length}
    totalCount={totalPlaylistCount}
  />

  {#if inboxShares.length > 0}
    <section class="playlists-section">
      <div class="playlists-section__head">
        <div class="playlists-section__lead">
          <h2 class="playlists-section__title">
            <MdiIcon name="share" size={18} />
            Shared with me
          </h2>
        </div>
      </div>
      <ul class="playlists-collection playlists-collection--list">
        {#each inboxShares as share (share.id)}
          <li class="playlists-inbox__item">
            <Link
              href={`/share/${encodeURIComponent(share.token)}`}
              class="playlists-inbox__link"
            >
              <span class="playlists-inbox__name"
                >{share.title || share.description || "Shared playlist"}</span
              >
              <span class="playlists-inbox__meta"
                >{share.accessMode} · open share</span
              >
            </Link>
          </li>
        {/each}
      </ul>
    </section>
  {/if}

  <section class="playlists-section">
    <div class="playlists-section__head">
      <div class="playlists-section__lead">
        <h2 class="playlists-section__title">
          {#if playlistKind === "server"}
            <SourceIcon kind="server" size={18} />
            Server playlists
          {:else}
            <MdiIcon name="folderOpen" size={18} />
            Local playlists
          {/if}
        </h2>
        <Tabs.Root
          style="display: contents"
          bind:value={
            () => playlistView,
            (value) => {
              playlistView = value as PlaylistViewMode;
            }
          }
        >
          <Tabs.List class="playlists-view" aria-label="Playlist layout">
            {#each VIEW_OPTIONS as option (option.id)}
              <Tabs.Trigger
                value={option.id}
                class="playlists-view__btn"
                aria-label={option.label}
                title={option.label}
              >
                <MdiIcon name={option.icon} size={17} />
              </Tabs.Trigger>
            {/each}
          </Tabs.List>
        </Tabs.Root>
      </div>
      <form
        class="playlists-create playlists-create--inline"
        onsubmit={(e) => {
          e.preventDefault();
          createPlaylist();
        }}
      >
        <input
          bind:value={newName}
          placeholder={playlistKind === "server"
            ? "New server playlist"
            : "New local playlist"}
          autocomplete="off"
        />
        <Button type="submit" disabled={creating || !newName.trim()}>
          <MdiIcon name="plus" size={16} />
          Create
        </Button>
        {#if showSmartPlaylist}
          <Button
            type="button"
            variant="surface"
            onclick={() => (smartPlaylistOpen = true)}
          >
            <MdiIcon name="slidersHorizontal" size={16} />
            Smart
          </Button>
        {/if}
      </form>
      {#if playlistKind === "server" && !smartPlaylistSupported && smartPlaylistSupportReason}
        <p class="playlists-smart-hint">{smartPlaylistSupportReason}</p>
      {/if}
    </div>

    {#if showInitialLoading}
      <div
        class="playlists-page__loading"
        class:playlists-page__loading--list={playlistView === "list"}
        class:playlists-page__loading--grid={playlistView !== "list"}
        role="status"
        aria-busy="true"
        aria-label="Loading playlists"
      >
        {#if playlistView === "list"}
          {#each Array.from({ length: 6 }) as _, i (i)}
            <Skeleton variant="row" />
          {/each}
        {:else}
          {#each Array.from({ length: 8 }) as _, i (i)}
            <Skeleton variant="card" />
          {/each}
        {/if}
      </div>
    {:else if unavailable}
      <LibraryUnavailable />
    {:else if activePlaylists.length === 0 && hasSearch}
      <EmptyState
        title="No matches"
        message={`No playlists match "${searchQuery.trim()}".`}
        icon="search"
      />
    {:else if activePlaylists.length === 0}
      <p class="playlists-section__empty">
        {playlistKind === "server"
          ? "No server playlists yet."
          : "No local playlists yet."}
      </p>
    {:else if useVirtual}
      <VirtualList
        items={activePlaylists}
        itemHeight={PLAYLIST_ROW_HEIGHT}
        scrollMode="document"
        class="playlists-collection playlists-collection--list"
      >
        {#snippet children({ item: pl })}
          <PlaylistTile
            playlist={pl}
            kind={playlistKind}
            view="list"
            onExport={() =>
              playlistKind === "server"
                ? void exportServerPlaylist(pl as ServerPlaylist)
                : void exportLocalPlaylist(pl as MusicPlaylist)}
            onDelete={() =>
              playlistKind === "server"
                ? deleteServerPlaylist(pl.id, pl.name)
                : deleteLocalPlaylist(pl.id)}
            onShare={() =>
              (shareTarget = {
                id: pl.id,
                name: pl.name,
                kind: playlistKind,
              })}
          />
        {/snippet}
      </VirtualList>
    {:else}
      <ul
        class="playlists-collection"
        class:playlists-collection--list={playlistView === "list"}
        class:playlists-collection--grid={playlistView === "grid"}
        class:playlists-collection--card={playlistView === "card"}
      >
        {#each activePlaylists as pl (pl.id)}
          <li class="playlists-collection__item">
            <PlaylistTile
              playlist={pl}
              kind={playlistKind}
              view={playlistView}
              onExport={() =>
                playlistKind === "server"
                  ? void exportServerPlaylist(pl as ServerPlaylist)
                  : void exportLocalPlaylist(pl as MusicPlaylist)}
              onDelete={() =>
                playlistKind === "server"
                  ? deleteServerPlaylist(pl.id, pl.name)
                  : deleteLocalPlaylist(pl.id)}
              onShare={() =>
                (shareTarget = {
                  id: pl.id,
                  name: pl.name,
                  kind: playlistKind,
                })}
            />
          </li>
        {/each}
      </ul>
    {/if}
  </section>
</div>

{#if pageMenu}
  <ContextMenu
    x={pageMenu.x}
    y={pageMenu.y}
    items={pageMenuItems}
    label="Playlist actions"
    onclose={() => (pageMenu = null)}
  />
{/if}

<SmartPlaylistCreator
  open={smartPlaylistOpen}
  onclose={() => (smartPlaylistOpen = false)}
  oncreate={createSmartPlaylist}
/>

{#if shareTarget}
  <SharePlaylistDialog
    open
    playlistId={shareTarget.id}
    playlistName={shareTarget.name}
    kind={shareTarget.kind}
    onclose={() => (shareTarget = null)}
  />
{/if}

<style>
  .playlists-page {
    transition: max-width var(--jb-transition);
  }

  .playlists-page--list {
    max-width: var(--jb-content-narrow);
  }

  .playlists-page--grid,
  .playlists-page--card {
    max-width: var(--jb-content-medium);
  }

  .playlists-header {
    display: flex;
    flex-wrap: wrap;
    align-items: flex-start;
    justify-content: space-between;
    gap: var(--jb-space-4);
    margin-bottom: var(--jb-space-6);
  }

  .playlists-header__tools {
    display: flex;
    align-items: center;
    gap: var(--jb-space-2);
  }

  .playlists-header__import-input {
    display: none;
  }

  :global(.playlists-kind) {
    display: inline-flex;
    gap: var(--jb-space-1);
    padding: 0.2rem;
    margin-bottom: var(--jb-space-6);
    border-radius: var(--jb-radius-full);
    border: 1px solid var(--jb-border);
    background: var(--jb-surface);
  }

  :global(.playlists-kind__btn) {
    display: inline-flex;
    align-items: center;
    gap: var(--jb-space-2);
    border: none;
    background: transparent;
    color: var(--jb-text-muted);
    font-size: 0.875rem;
    font-weight: 650;
    padding: 0.45rem 0.9rem;
    border-radius: var(--jb-radius-full);
    cursor: pointer;
  }

  :global(.playlists-kind__btn[data-state="active"]) {
    background: var(--jb-bg-muted);
    color: var(--jb-text);
    box-shadow: var(--jb-shadow-sm);
  }

  .playlists-section {
    margin-bottom: var(--jb-space-8);
  }

  .playlists-section__head {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    justify-content: space-between;
    gap: var(--jb-space-3);
    margin-bottom: var(--jb-space-5);
  }

  .playlists-section__lead {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--jb-space-3);
  }

  .playlists-section__title {
    display: inline-flex;
    align-items: center;
    gap: var(--jb-space-2);
    margin: 0;
    font-size: 1rem;
    font-weight: 700;
  }

  :global(.playlists-view) {
    display: inline-flex;
    gap: 0.15rem;
    padding: 0.15rem;
    border-radius: var(--jb-radius-full);
    border: 1px solid var(--jb-border);
    background: var(--jb-bg-muted);
  }

  :global(.playlists-view__btn) {
    display: inline-grid;
    place-items: center;
    width: 2rem;
    height: 2rem;
    border: none;
    border-radius: var(--jb-radius-full);
    background: transparent;
    color: var(--jb-text-muted);
    cursor: pointer;
    transition:
      background var(--jb-transition),
      color var(--jb-transition),
      box-shadow var(--jb-transition);
  }

  :global(.playlists-view__btn:hover) {
    color: var(--jb-text);
  }

  :global(.playlists-view__btn[data-state="active"]) {
    background: var(--jb-surface);
    color: var(--jb-music-accent);
    box-shadow: var(--jb-shadow-sm);
  }

  .playlists-section__empty {
    margin: 0;
    color: var(--jb-text-muted);
    font-size: 0.9375rem;
  }

  .playlists-header__eyebrow {
    display: inline-flex;
    align-items: center;
    gap: var(--jb-space-2);
    margin: 0 0 var(--jb-space-2);
    font-size: 0.8125rem;
    font-weight: 600;
    color: var(--jb-music-accent);
    text-transform: uppercase;
    letter-spacing: 0.06em;
  }

  .playlists-header h1 {
    margin: 0 0 var(--jb-space-2);
    font-size: 2rem;
    font-weight: 800;
  }

  .playlists-header__sub {
    margin: 0;
    color: var(--jb-text-muted);
    max-width: 34rem;
  }

  .playlists-create {
    display: flex;
    gap: var(--jb-space-3);
    margin-bottom: var(--jb-space-4);
  }

  .playlists-create--inline {
    margin-bottom: 0;
    flex: 1 1 16rem;
    max-width: 28rem;
  }

  .playlists-smart-hint {
    flex: 1 1 100%;
    margin: 0;
    font-size: 0.8125rem;
    color: var(--jb-text-muted);
  }

  .playlists-page :global(.local-search-box) {
    margin-bottom: var(--jb-space-6);
  }

  .playlists-create input {
    flex: 1;
    padding: 0.625rem 0.875rem;
    border-radius: var(--jb-radius-md);
    border: 1px solid var(--jb-border);
    background: var(--jb-bg);
    color: var(--jb-text);
    font: inherit;
  }

  .playlists-page__loading--list {
    display: flex;
    flex-direction: column;
    gap: var(--jb-space-2);
  }

  .playlists-page__loading--grid {
    display: grid;
    grid-template-columns: repeat(
      auto-fill,
      minmax(min(100%, var(--jb-card-grid-min)), 1fr)
    );
    gap: var(--jb-space-5);
  }

  .playlists-collection {
    list-style: none;
    margin: 0;
    padding: 0;
  }

  .playlists-collection--list {
    display: grid;
    gap: var(--jb-space-2);
  }

  .playlists-collection--grid,
  .playlists-collection--card {
    display: grid;
    grid-template-columns: repeat(
      auto-fill,
      minmax(min(100%, var(--jb-card-grid-min)), 1fr)
    );
    gap: var(--jb-space-5);
  }

  .playlists-collection__item {
    min-width: 0;
  }

  .playlists-inbox__item {
    list-style: none;
  }

  .playlists-inbox__item :global(.playlists-inbox__link) {
    display: flex;
    flex-direction: column;
    gap: 0.15rem;
    padding: var(--jb-space-3) var(--jb-space-4);
    border-radius: var(--jb-radius-md);
    background: var(--jb-surface-2);
    text-decoration: none;
    color: inherit;
  }

  .playlists-inbox__name {
    font-weight: 600;
  }

  .playlists-inbox__meta {
    font-size: 0.85rem;
    color: var(--jb-text-muted);
  }

  :global(.playlists-collection--list.virtual-list--document) {
    display: block;
  }

  :global(.playlists-collection--list .virtual-list__window) {
    display: grid;
    gap: var(--jb-space-2);
  }

  @media (max-width: 640px) {
    .playlists-page--grid,
    .playlists-page--card,
    .playlists-page--list {
      max-width: none;
    }

    .playlists-collection--grid {
      grid-template-columns: repeat(2, minmax(0, 1fr));
      gap: var(--jb-space-3);
    }

    .playlists-collection--card {
      grid-template-columns: repeat(2, minmax(0, 1fr));
      gap: var(--jb-space-3);
    }

    .playlists-page__loading--grid {
      grid-template-columns: repeat(2, minmax(0, 1fr));
      gap: var(--jb-space-3);
    }

    .playlists-section__head {
      align-items: stretch;
    }

    .playlists-create--inline {
      max-width: none;
      flex-wrap: wrap;
    }

    .playlists-section__lead {
      width: 100%;
    }

    .playlists-create--inline :global(.button) {
      flex: 1;
      min-width: 5.5rem;
    }
  }
</style>
