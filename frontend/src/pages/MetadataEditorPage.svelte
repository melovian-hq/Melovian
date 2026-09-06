<script lang="ts">
  import AppShell from "$lib/components/layout/AppShell.svelte";
  import PageHeader from "$lib/components/ui/PageHeader.svelte";
  import LocalSearchBox from "$lib/components/ui/LocalSearchBox.svelte";
  import EmptyState from "$lib/components/ui/EmptyState.svelte";
  import Spinner from "$lib/components/ui/Spinner.svelte";
  import Button from "$lib/components/ui/Button.svelte";
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";
  import BatchProgress from "$lib/components/ui/BatchProgress.svelte";
  import { mapClientError } from "$lib/ui/client-error";
  import MetadataTrackEditor from "$lib/components/metadata/MetadataTrackEditor.svelte";
  import TrackContextMenu from "$lib/components/music/TrackContextMenu.svelte";
  import ContextMenu from "$lib/components/ui/ContextMenu.svelte";
  import {
    contextMenuPositionFromEvent,
    type ContextMenuEntry,
  } from "$lib/components/ui/context-menu";
  import { router } from "$lib/router/router.svelte";
  import { toast } from "$lib/ui/toast.svelte";
  import { confirmDialog } from "$lib/ui/confirm.svelte";
  import { localLibraries } from "$lib/features/local-libraries/store.svelte";
  import { extensionFeatures } from "$lib/extensions/features.svelte";
  import Link from "$lib/router/Link.svelte";
  import {
    applyMetadataAutofix,
    applyMetadataMatchToAlbum,
    batchAutofixMetadata,
    fetchMetadataSuggestions,
    fetchMetadataSummary,
    fetchMetadataTrack,
    lookupMetadataMatches,
    searchMetadataTracks,
    updateMetadataTrack,
  } from "$lib/features/metadata-editor/api";
  import type {
    MetadataBatchSource,
    MetadataLookupSource,
  } from "$lib/features/metadata-editor/api";
  import { metadataLookupSourceLabel } from "$lib/features/metadata-editor/api";
  import {
    nextTrackId,
    selectedTrackIndex,
    metadataTrackToSong,
  } from "$lib/features/metadata-editor/utils";
  import type {
    MetadataFilenameSuggestion,
    MetadataIssueFilter,
    MetadataLookupMatch,
    MetadataSummary,
    MetadataTrack,
    MetadataTrackUpdate,
  } from "$lib/features/metadata-editor/types";

  const pageSize = 50;

  let query = $state("");
  let issueFilter = $state<MetadataIssueFilter>("any");
  let tracks = $state<MetadataTrack[]>([]);
  let total = $state(0);
  let offset = $state(0);
  let summary = $state<MetadataSummary | null>(null);
  let selectedId = $state<string | null>(null);
  let selectedTrack = $state<MetadataTrack | null>(null);
  let filenameSuggestion = $state<MetadataFilenameSuggestion | null>(null);
  let loading = $state(true);
  let loadingMore = $state(false);
  let saving = $state(false);
  let batchRunning = $state(false);
  let batchProgress = $state<{
    done: number;
    total: number;
    label: string;
  } | null>(null);
  let lookupLoading = $state(false);
  let matches = $state<MetadataLookupMatch[]>([]);
  let loadError = $state<string | null>(null);
  let editorDirty = $state(false);
  let trackMenu = $state<{ x: number; y: number; track: MetadataTrack } | null>(
    null,
  );
  let listMenu = $state<{ x: number; y: number } | null>(null);

  let debounce: ReturnType<typeof setTimeout> | undefined;
  let listRequest = 0;
  let detailRequest = 0;

  const issueFilters: {
    id: MetadataIssueFilter;
    label: string;
    countKey: keyof MetadataSummary | null;
  }[] = [
    { id: "any", label: "Needs attention", countKey: "any" },
    { id: "", label: "All tracks", countKey: "totalTracks" },
    {
      id: "unknown-artist",
      label: "Unknown artist",
      countKey: "unknownArtist",
    },
    { id: "unknown-album", label: "Unknown album", countKey: "unknownAlbum" },
    { id: "missing-title", label: "Missing title", countKey: "missingTitle" },
  ];

  const hasLocalLibrary = $derived(Boolean(localLibraries.active?.id));

  const hasMore = $derived(tracks.length < total);
  const selectedIndex = $derived(selectedTrackIndex(tracks, selectedId));

  function trackIdFromQuery(): string | null {
    if (typeof window === "undefined") return null;
    const value = new URLSearchParams(window.location.search).get("track");
    return value?.trim() ? value.trim() : null;
  }

  async function loadSummary() {
    try {
      summary = await fetchMetadataSummary();
    } catch {
      summary = null;
    }
  }

  async function loadTracks(reset = true) {
    if (!hasLocalLibrary) {
      tracks = [];
      total = 0;
      loading = false;
      return;
    }
    const request = ++listRequest;
    if (reset) {
      loading = true;
      offset = 0;
    } else {
      loadingMore = true;
    }
    loadError = null;
    try {
      const result = await searchMetadataTracks({
        q: query,
        issue: issueFilter,
        limit: pageSize,
        offset: reset ? 0 : offset,
      });
      if (request !== listRequest) return;
      tracks = reset ? result.tracks : [...tracks, ...result.tracks];
      total = result.total;
      offset = tracks.length;

      if (selectedId && !tracks.some((track) => track.id === selectedId)) {
        selectedId = tracks[0]?.id ?? null;
      }
      if (!selectedId && tracks.length > 0) {
        selectedId = tracks[0].id;
      }
      const fromQuery = trackIdFromQuery();
      if (fromQuery && tracks.some((track) => track.id === fromQuery)) {
        selectedId = fromQuery;
      }
    } catch (err) {
      if (request !== listRequest) return;
      loadError = err instanceof Error ? err.message : "Failed to load tracks";
      if (reset) {
        tracks = [];
        total = 0;
      }
    } finally {
      if (request === listRequest) {
        loading = false;
        loadingMore = false;
      }
    }
  }

  async function loadSelectedTrack(id: string | null) {
    if (!id) {
      selectedTrack = null;
      filenameSuggestion = null;
      matches = [];
      return;
    }
    const request = ++detailRequest;
    try {
      const [track, suggestion] = await Promise.all([
        fetchMetadataTrack(id),
        fetchMetadataSuggestions(id),
      ]);
      if (request !== detailRequest) return;
      selectedTrack = track;
      filenameSuggestion = suggestion;
      matches = [];
      editorDirty = false;
    } catch (err) {
      if (request !== detailRequest) return;
      selectedTrack = null;
      filenameSuggestion = null;
      toast.error(err instanceof Error ? err.message : "Failed to load track");
    }
  }

  async function refreshAll() {
    await loadSummary();
    await loadTracks(true);
    if (selectedId) {
      await loadSelectedTrack(selectedId);
    }
  }

  $effect(() => {
    void localLibraries.active?.id;
    void refreshAll();
  });

  $effect(() => {
    void query;
    void issueFilter;
    clearTimeout(debounce);
    debounce = setTimeout(() => {
      void loadTracks(true);
    }, 220);
    return () => clearTimeout(debounce);
  });

  $effect(() => {
    const id = selectedId;
    void loadSelectedTrack(id);
  });

  async function confirmDiscard(): Promise<boolean> {
    if (!editorDirty) return true;
    return confirmDialog.confirm({
      title: "Discard changes",
      message: "Discard unsaved metadata changes?",
      confirmLabel: "Discard",
      danger: true,
    });
  }

  async function selectTrack(id: string) {
    if (!(await confirmDiscard())) return;
    selectedId = id;
  }

  async function navigateTrack(direction: 1 | -1) {
    if (!(await confirmDiscard())) return;
    selectedId = nextTrackId(tracks, selectedId, direction);
  }

  async function advanceAfterSave(savedId: string) {
    const previousIndex = tracks.findIndex((track) => track.id === savedId);
    await loadTracks(true);
    if (tracks.length === 0) {
      selectedId = null;
      return;
    }
    const nextIndex = Math.min(
      previousIndex >= 0 ? previousIndex : 0,
      tracks.length - 1,
    );
    selectedId = tracks[nextIndex]?.id ?? tracks[0]?.id ?? null;
  }

  async function handleSave(update: MetadataTrackUpdate) {
    if (!selectedId) return;
    const savedId = selectedId;
    saving = true;
    try {
      const updated = await updateMetadataTrack(selectedId, update);
      selectedTrack = updated;
      tracks = tracks.map((track) =>
        track.id === updated.id ? updated : track,
      );
      editorDirty = false;
      toast.success("Metadata saved to file");
      await loadSummary();
      if (issueFilter !== "") {
        await advanceAfterSave(savedId);
      }
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to save metadata",
      );
    } finally {
      saving = false;
    }
  }

  async function handleLookup(searchQuery: string, source?: string) {
    if (!selectedTrack) return;
    lookupLoading = true;
    try {
      matches = await lookupMetadataMatches({
        q: searchQuery,
        trackId: selectedTrack.id,
        limit: 8,
        source: source as MetadataLookupSource | undefined,
      });
      if (matches.length === 0) {
        toast.info("No matches found. Try a different lookup search.");
      }
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Lookup failed");
      matches = [];
    } finally {
      lookupLoading = false;
    }
  }

  async function handleApplyMatch(match: MetadataLookupMatch) {
    if (!selectedId) return;
    if (!(await confirmDiscard())) return;
    const savedId = selectedId;
    saving = true;
    try {
      const updated = await applyMetadataAutofix(selectedId, match);
      selectedTrack = updated;
      tracks = tracks.map((track) =>
        track.id === updated.id ? updated : track,
      );
      matches = [];
      editorDirty = false;
      toast.success("Metadata updated from match");
      await loadSummary();
      if (issueFilter !== "") {
        await advanceAfterSave(savedId);
      } else {
        await loadSelectedTrack(selectedId);
      }
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Auto-fix failed");
    } finally {
      saving = false;
    }
  }

  async function handleApplyMatchToAlbum(match: MetadataLookupMatch) {
    if (!selectedId) return;
    if (!(await confirmDiscard())) return;
    const peerLabel = selectedTrack?.album || "this album";
    const ok = await confirmDialog.confirm({
      title: "Apply to album",
      message: `Update album tags for all tracks in "${peerLabel}"? Titles and track numbers stay as they are.`,
      confirmLabel: "Apply to album",
    });
    if (!ok) return;
    saving = true;
    try {
      const result = await applyMetadataMatchToAlbum(selectedId, match);
      toast.success(
        `Updated ${result.updated} track${result.updated === 1 ? "" : "s"} in album`,
      );
      if (result.failed > 0) {
        toast.error(
          `${result.failed} track${result.failed === 1 ? "" : "s"} could not be updated`,
        );
      }
      editorDirty = false;
      matches = [];
      await refreshAll();
    } catch (err) {
      toast.error(mapClientError(err).message);
    } finally {
      saving = false;
    }
  }

  async function handleBatchAutofix(source: MetadataBatchSource) {
    if (tracks.length === 0 || batchRunning) return;
    const label =
      source === "filename"
        ? "filename tags"
        : `${metadataLookupSourceLabel(source)} matches`;
    const ok = await confirmDialog.confirm({
      title: "Batch metadata update",
      message: `Apply ${label} to ${tracks.length} visible tracks?`,
      confirmLabel: "Apply to all",
    });
    if (!ok) return;
    const ids = tracks.map((track) => track.id);
    const chunkSize = 25;
    batchRunning = true;
    batchProgress = {
      done: 0,
      total: ids.length,
      label: `Applying ${label}`,
    };
    let okCount = 0;
    let failCount = 0;
    try {
      for (let i = 0; i < ids.length; i += chunkSize) {
        const chunk = ids.slice(i, i + chunkSize);
        const results = await batchAutofixMetadata({
          trackIds: chunk,
          source,
        });
        okCount += results.filter((result) => result.ok).length;
        failCount += results.filter((result) => !result.ok).length;
        batchProgress = {
          done: Math.min(i + chunk.length, ids.length),
          total: ids.length,
          label: `Applying ${label}`,
        };
      }
      if (okCount > 0) {
        toast.success(`Updated ${okCount} track${okCount === 1 ? "" : "s"}`);
      }
      if (failCount > 0) {
        toast.error(
          `${failCount} track${failCount === 1 ? "" : "s"} could not be updated`,
        );
      }
      await refreshAll();
    } catch (err) {
      toast.error(mapClientError(err).message);
    } finally {
      batchRunning = false;
      batchProgress = null;
    }
  }

  function goToLibraries() {
    router.navigate("/settings/servers#local-library");
  }

  function onTrackContextMenu(event: MouseEvent, track: MetadataTrack) {
    const pos = contextMenuPositionFromEvent(event);
    if (!pos) return;
    trackMenu = { ...pos, track };
  }

  function onListContextMenu(event: MouseEvent) {
    listMenu = contextMenuPositionFromEvent(event);
  }

  const listMenuItems = $derived.by((): ContextMenuEntry[] => [
    {
      id: "filename",
      label: "Fix visible from filename",
      icon: "autoFix",
      disabled: batchRunning || saving || tracks.length === 0,
      onclick: () => void handleBatchAutofix("filename"),
    },
    {
      id: "lookup",
      label: "Apply iTunes matches to visible",
      icon: "search",
      disabled: batchRunning || saving || tracks.length === 0,
      onclick: () => void handleBatchAutofix("itunes"),
    },
    {
      id: "lookup-musicbrainz",
      label: "Apply MusicBrainz matches to visible",
      icon: "search",
      disabled: batchRunning || saving || tracks.length === 0,
      onclick: () => void handleBatchAutofix("musicbrainz"),
    },
    {
      id: "lookup-deezer",
      label: "Apply Deezer matches to visible",
      icon: "search",
      disabled: batchRunning || saving || tracks.length === 0,
      onclick: () => void handleBatchAutofix("deezer"),
    },
    {
      id: "lookup-theaudiodb",
      label: "Apply TheAudioDB matches to visible",
      icon: "search",
      disabled: batchRunning || saving || tracks.length === 0,
      onclick: () => void handleBatchAutofix("theaudiodb"),
    },
    {
      id: "lookup-all",
      label: "Apply best match from all providers",
      icon: "search",
      disabled: batchRunning || saving || tracks.length === 0,
      onclick: () => void handleBatchAutofix("all"),
    },
  ]);
</script>

<AppShell compactTop>
  {#if !extensionFeatures.metadata}
    <EmptyState
      title="Metadata extension is off"
      message="Turn on Metadata under Settings, Extensions to edit local tags and enhance artwork."
      icon="autoFix"
    >
      {#snippet actions()}
        <Link href="/settings/extensions">Open Extensions</Link>
      {/snippet}
    </EmptyState>
  {:else}
    <PageHeader
      title="Metadata editor"
      subtitle="Search local tracks, fix tags manually, or apply matches from iTunes, MusicBrainz, Deezer, or TheAudioDB."
    >
      {#snippet actions()}
        {#if hasLocalLibrary && tracks.length > 0}
          <Button
            variant="surface"
            disabled={batchRunning || saving}
            onclick={() => handleBatchAutofix("filename")}
          >
            {#if batchRunning}
              <Spinner />
            {:else}
              <MdiIcon name="autoFix" size={18} />
            {/if}
            Fix visible from filename
          </Button>
        {/if}
      {/snippet}
    </PageHeader>

    {#if batchProgress}
      <BatchProgress
        done={batchProgress.done}
        total={batchProgress.total}
        label={batchProgress.label}
      />
    {/if}

    {#if summary}
      <div class="metadata-page__summary">
        <div class="metadata-page__stat">
          <span class="metadata-page__stat-value">{summary.any}</span>
          <span class="metadata-page__stat-label">Need attention</span>
        </div>
        <div class="metadata-page__stat">
          <span class="metadata-page__stat-value">{summary.unknownArtist}</span>
          <span class="metadata-page__stat-label">Unknown artist</span>
        </div>
        <div class="metadata-page__stat">
          <span class="metadata-page__stat-value">{summary.unknownAlbum}</span>
          <span class="metadata-page__stat-label">Unknown album</span>
        </div>
        <div class="metadata-page__stat">
          <span class="metadata-page__stat-value">{summary.totalTracks}</span>
          <span class="metadata-page__stat-label">Total tracks</span>
        </div>
      </div>
    {/if}

    {#if !hasLocalLibrary}
      <EmptyState
        title="No active local library"
        message="Add a local library in Settings, then come back here to clean up tags."
        icon="folderMusic"
      >
        {#snippet actions()}
          <Button onclick={goToLibraries}>Open library settings</Button>
        {/snippet}
      </EmptyState>
    {:else}
      <div class="metadata-page">
        <aside
          class="metadata-page__list-panel"
          role="group"
          oncontextmenu={onListContextMenu}
        >
          <div class="metadata-page__filters">
            {#each issueFilters as filter (filter.id)}
              <button
                type="button"
                class="metadata-page__filter"
                class:metadata-page__filter--active={issueFilter === filter.id}
                onclick={() => {
                  void (async () => {
                    if (!(await confirmDiscard())) return;
                    issueFilter = filter.id;
                  })();
                }}
              >
                {filter.label}
                {#if summary && filter.countKey}
                  <span class="metadata-page__filter-count">
                    {summary[filter.countKey]}
                  </span>
                {/if}
              </button>
            {/each}
          </div>

          <LocalSearchBox
            bind:value={query}
            placeholder="Search by title, artist, album, or path"
            resultCount={tracks.length}
            totalCount={total}
          />

          {#if loading}
            <div class="metadata-page__loading">
              <Spinner />
            </div>
          {:else if loadError}
            <EmptyState
              title="Could not load tracks"
              message={loadError}
              icon="alertCircle"
            />
          {:else if tracks.length === 0}
            <EmptyState
              title="No tracks found"
              message="Try another search term or filter."
              icon="search"
            />
          {:else}
            <ul class="metadata-page__tracks" aria-label="Tracks">
              {#each tracks as track (track.id)}
                <li>
                  <button
                    type="button"
                    class="metadata-page__track"
                    class:metadata-page__track--active={track.id === selectedId}
                    onclick={() => void selectTrack(track.id)}
                    oncontextmenu={(event) => onTrackContextMenu(event, track)}
                  >
                    <span class="metadata-page__track-title">{track.title}</span
                    >
                    <span class="metadata-page__track-meta">
                      {track.artist || "Unknown artist"}
                      {#if track.album}
                        · {track.album}
                      {/if}
                    </span>
                    {#if track.issues.length > 0}
                      <span class="metadata-page__track-issues">
                        {track.issues.length} issue{track.issues.length === 1
                          ? ""
                          : "s"}
                      </span>
                    {/if}
                  </button>
                </li>
              {/each}
            </ul>
            {#if hasMore}
              <div class="metadata-page__more">
                <Button
                  variant="surface"
                  disabled={loadingMore}
                  onclick={() => loadTracks(false)}
                >
                  {#if loadingMore}
                    <Spinner />
                  {/if}
                  Load more
                </Button>
              </div>
            {/if}
          {/if}
        </aside>

        <section class="metadata-page__editor-panel">
          {#if selectedTrack}
            <MetadataTrackEditor
              track={selectedTrack}
              {saving}
              {lookupLoading}
              {matches}
              {filenameSuggestion}
              hasPrevious={selectedIndex > 0}
              hasNext={selectedIndex >= 0 && selectedIndex < tracks.length - 1}
              onsave={handleSave}
              onlookup={handleLookup}
              onapplymatch={handleApplyMatch}
              onapplyalbum={handleApplyMatchToAlbum}
              onprevious={() => navigateTrack(-1)}
              onnext={() => navigateTrack(1)}
              ondirtychange={(dirty) => {
                editorDirty = dirty;
              }}
            />
          {:else if !loading}
            <EmptyState
              title="Select a track"
              message="Pick a track from the list to edit its metadata."
              icon="tag"
            />
          {/if}
        </section>
      </div>
    {/if}
  {/if}
</AppShell>

{#if trackMenu}
  <TrackContextMenu
    track={metadataTrackToSong(trackMenu.track)}
    x={trackMenu.x}
    y={trackMenu.y}
    onclose={() => (trackMenu = null)}
  />
{/if}

{#if listMenu}
  <ContextMenu
    x={listMenu.x}
    y={listMenu.y}
    items={listMenuItems}
    label="Batch metadata actions"
    onclose={() => (listMenu = null)}
  />
{/if}

<style>
  .metadata-page__summary {
    display: grid;
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: var(--jb-space-3);
    margin-bottom: var(--jb-space-5);
  }

  .metadata-page__stat {
    display: flex;
    flex-direction: column;
    gap: 0.125rem;
    padding: var(--jb-space-4);
    border: 1px solid var(--jb-border);
    border-radius: var(--jb-radius-lg);
    background: var(--jb-surface);
  }

  .metadata-page__stat-value {
    font-size: 1.5rem;
    font-weight: 700;
    font-variant-numeric: tabular-nums;
    color: var(--jb-text);
  }

  .metadata-page__stat-label {
    font-size: 0.8125rem;
    color: var(--jb-text-muted);
  }

  .metadata-page {
    display: grid;
    grid-template-columns: minmax(280px, 360px) minmax(0, 1fr);
    gap: var(--jb-space-5);
    align-items: start;
  }

  .metadata-page__list-panel,
  .metadata-page__editor-panel {
    border: 1px solid var(--jb-border);
    border-radius: var(--jb-radius-xl);
    background: var(--jb-surface);
    padding: var(--jb-space-4);
    box-shadow: var(--jb-shadow-sm);
  }

  .metadata-page__filters {
    display: flex;
    flex-wrap: wrap;
    gap: var(--jb-space-2);
    margin-bottom: var(--jb-space-4);
  }

  .metadata-page__filter {
    display: inline-flex;
    align-items: center;
    gap: var(--jb-space-2);
    padding: 0.375rem 0.75rem;
    border: 1px solid var(--jb-border);
    border-radius: var(--jb-radius-full);
    background: transparent;
    color: var(--jb-text-muted);
    font-size: 0.8125rem;
    font-weight: 600;
    cursor: pointer;
  }

  .metadata-page__filter--active {
    border-color: var(--jb-accent);
    background: color-mix(in srgb, var(--jb-accent) 12%, transparent);
    color: var(--jb-accent);
  }

  .metadata-page__filter-count {
    min-width: 1.25rem;
    padding: 0 0.375rem;
    border-radius: var(--jb-radius-full);
    background: color-mix(in srgb, currentColor 12%, transparent);
    font-size: 0.75rem;
    font-variant-numeric: tabular-nums;
    text-align: center;
  }

  .metadata-page__loading,
  .metadata-page__more {
    display: flex;
    justify-content: center;
    padding: var(--jb-space-4) 0;
  }

  .metadata-page__tracks {
    list-style: none;
    margin: var(--jb-space-4) 0 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: var(--jb-space-2);
    max-height: min(70vh, 720px);
    overflow: auto;
  }

  .metadata-page__track {
    width: 100%;
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: 0.125rem;
    padding: var(--jb-space-3);
    border: 1px solid transparent;
    border-radius: var(--jb-radius-lg);
    background: transparent;
    text-align: left;
    cursor: pointer;
    color: var(--jb-text);
  }

  .metadata-page__track:hover,
  .metadata-page__track--active {
    border-color: var(--jb-border);
    background: color-mix(in srgb, var(--jb-accent) 8%, transparent);
  }

  .metadata-page__track-title {
    font-size: 0.9375rem;
    font-weight: 600;
  }

  .metadata-page__track-meta {
    font-size: 0.8125rem;
    color: var(--jb-text-muted);
  }

  .metadata-page__track-issues {
    margin-top: 0.125rem;
    font-size: 0.75rem;
    font-weight: 600;
    color: var(--jb-warning);
  }

  @media (max-width: 960px) {
    .metadata-page__summary {
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }

    .metadata-page {
      grid-template-columns: 1fr;
    }

    .metadata-page__tracks {
      max-height: 320px;
    }
  }
</style>
