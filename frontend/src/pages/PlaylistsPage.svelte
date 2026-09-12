<script lang="ts">
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";
  import SourceIcon from "$lib/components/ui/SourceIcon.svelte";
  import MusicBreadcrumbs from "$lib/components/music/MusicBreadcrumbs.svelte";
  import SmartPlaylistCreator from "$lib/components/music/SmartPlaylistCreator.svelte";
  import PlaylistCollection from "$lib/components/music/PlaylistCollection.svelte";
  import PlaylistsHeader from "$lib/components/music/PlaylistsHeader.svelte";
  import PlaylistKindTabs from "$lib/components/music/PlaylistKindTabs.svelte";
  import PlaylistViewTabs from "$lib/components/music/PlaylistViewTabs.svelte";
  import Button from "$lib/components/ui/Button.svelte";
  import Skeleton from "$lib/components/ui/Skeleton.svelte";
  import LocalSearchBox from "$lib/components/ui/LocalSearchBox.svelte";
  import EmptyState from "$lib/components/ui/EmptyState.svelte";
  import LibraryUnavailable from "$lib/components/music/LibraryUnavailable.svelte";
  import { music } from "$lib/config/music.svelte";
  import { sources } from "$lib/features/sources/store.svelte";
  import { libraryUnavailable } from "$lib/music/library-gate";
  import { filterByLocalSearch } from "$lib/utils/local-search";
  import * as musicApi from "$lib/music/api";
  import {
    createNamedPlaylist,
    createSmartPlaylistForKind,
    deleteLocalPlaylist,
    deleteServerPlaylist,
    exportLocalPlaylist,
    exportServerPlaylist,
    importM3UPlaylist,
  } from "$lib/music/playlist-actions";
  import {
    loadPlaylistKind,
    loadPlaylistView,
    savePlaylistKind,
    PLAYLIST_VIEW_KEY,
    type PlaylistItem,
    type PlaylistKind,
    type PlaylistViewMode,
  } from "$lib/music/playlist-display";
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

  let playlistView = $state<PlaylistViewMode>(loadPlaylistView());

  $effect(() => {
    const saved = loadPlaylistKind();
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
    savePlaylistKind(playlistKind);
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
    await createSmartPlaylistForKind(draft, playlistKind);
  }

  async function createPlaylist() {
    const name = newName.trim();
    if (!name) return;
    creating = true;
    try {
      if (await createNamedPlaylist(playlistKind, name)) {
        newName = "";
      }
    } finally {
      creating = false;
    }
  }

  function exportPlaylist(pl: PlaylistItem) {
    if (playlistKind === "server") {
      void exportServerPlaylist(pl as ServerPlaylist);
    } else {
      void exportLocalPlaylist(pl as MusicPlaylist);
    }
  }

  function removePlaylist(pl: PlaylistItem) {
    if (playlistKind === "server") {
      void deleteServerPlaylist(pl.id, pl.name);
    } else {
      void deleteLocalPlaylist(pl.id);
    }
  }

  function sharePlaylist(pl: PlaylistItem) {
    shareTarget = {
      id: pl.id,
      name: pl.name,
      kind: playlistKind,
    };
  }

  async function importPlaylistFile(file: File) {
    importing = true;
    try {
      await importM3UPlaylist(file, playlistKind === "server" && canUseServer);
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

  <PlaylistsHeader
    {showKindToggle}
    {canUseServer}
    {importing}
    bind:importInput
    {onImportSelected}
  />

  {#if showKindToggle}
    <PlaylistKindTabs bind:value={playlistKind} />
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
        <PlaylistViewTabs bind:value={playlistView} />
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
    {:else}
      <PlaylistCollection
        items={activePlaylists}
        kind={playlistKind}
        view={playlistView}
        onexport={exportPlaylist}
        ondelete={removePlaylist}
        onshare={sharePlaylist}
      />
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

  .playlists-section__empty {
    margin: 0;
    color: var(--jb-text-muted);
    font-size: 0.9375rem;
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

  @media (max-width: 640px) {
    .playlists-page--grid,
    .playlists-page--card,
    .playlists-page--list {
      max-width: none;
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
