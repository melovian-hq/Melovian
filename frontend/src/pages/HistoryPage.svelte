<script lang="ts">
  import AppShell from "$lib/components/layout/AppShell.svelte";
  import MusicBreadcrumbs from "$lib/components/music/MusicBreadcrumbs.svelte";
  import CollectionHero from "$lib/components/music/CollectionHero.svelte";
  import ListenHistoryList from "$lib/components/music/ListenHistoryList.svelte";
  import Skeleton from "$lib/components/ui/Skeleton.svelte";
  import Spinner from "$lib/components/ui/Spinner.svelte";
  import LocalSearchBox from "$lib/components/ui/LocalSearchBox.svelte";
  import EmptyState from "$lib/components/ui/EmptyState.svelte";
  import LibraryUnavailable from "$lib/components/music/LibraryUnavailable.svelte";
  import { music } from "$lib/config/music.svelte";
  import { instances } from "$lib/features/instances/store.svelte";
  import * as musicApi from "$lib/music/api";
  import { libraryUnavailable } from "$lib/music/library-gate";
  import ContextMenu from "$lib/components/ui/ContextMenu.svelte";
  import {
    contextMenuPositionFromEvent,
    type ContextMenuEntry,
  } from "$lib/components/ui/context-menu";
  import { toast } from "$lib/ui/toast.svelte";
  import { confirmDialog } from "$lib/ui/confirm.svelte";
  import {
    STATS_PERIOD_LABELS,
    statsPeriodForYear,
    statsPeriodYear,
    type StatsPeriod,
  } from "$lib/music/stats-utils";
  import type { ListenEvent } from "$lib/subsonic/types";
  import { coverArtUrl, type SubsonicSong } from "$lib/subsonic";

  const PAGE_SIZE = 100;
  const SEARCH_DEBOUNCE_MS = 250;

  let loading = $state(true);
  let loadingMore = $state(false);
  let error = $state<string | null>(null);
  let events = $state<ListenEvent[]>([]);
  let hasMore = $state(false);
  let period = $state<StatsPeriod>("all");
  let searchQuery = $state("");
  let debouncedSearch = $state("");
  let listenYears = $state<number[]>([]);
  let pageMenu = $state<{ x: number; y: number } | null>(null);

  const unavailable = $derived(libraryUnavailable());
  const hasSearch = $derived(searchQuery.trim().length > 0);
  const showInitialLoading = $derived(
    !unavailable && loading && events.length === 0,
  );

  const scopeLabel = $derived.by(() => {
    const inst = instances.active;
    if (!inst) return "this device";
    const parts = [inst.serverName || inst.serverUrl];
    if (inst.username) parts.push(inst.username);
    return parts.join(" · ");
  });

  const periodLabel = $derived.by(() => {
    const year = statsPeriodYear(period);
    if (year !== null) return String(year);
    if (
      period === "all" ||
      period === "7d" ||
      period === "30d" ||
      period === "90d"
    ) {
      return STATS_PERIOD_LABELS[period];
    }
    return "All time";
  });

  const playTracks = $derived(events.map(eventToSong));

  const heroCoverSrc = $derived(
    events[0]
      ? coverArtUrl(
          music.config,
          events[0].coverArtId || events[0].albumId || events[0].trackId,
          800,
        )
      : null,
  );

  const heroMeta = $derived.by(() => {
    if (events.length === 0) {
      return `${periodLabel} · ${scopeLabel}`;
    }
    const count = hasMore ? `${events.length}+` : String(events.length);
    const plays = events.length === 1 ? "1 play" : `${count} plays`;
    return `${plays} · ${periodLabel} · ${scopeLabel}`;
  });

  $effect(() => {
    const q = searchQuery;
    const timer = setTimeout(() => {
      debouncedSearch = q;
    }, SEARCH_DEBOUNCE_MS);
    return () => clearTimeout(timer);
  });

  $effect(() => {
    if (unavailable) {
      loading = false;
      error = null;
      events = [];
      return;
    }
    if (!music.libraryReady) {
      loading = music.loading;
      error = music.error;
      return;
    }

    const currentPeriod = period;
    const currentSearch = debouncedSearch.trim();
    let cancelled = false;

    loading = true;
    error = null;
    events = [];
    hasMore = false;

    void (async () => {
      try {
        const [page, years] = await Promise.all([
          musicApi.getListenEvents({
            limit: PAGE_SIZE,
            offset: 0,
            period: currentPeriod,
            q: currentSearch || undefined,
          }),
          currentPeriod === "all" && !currentSearch
            ? musicApi.getListenEventYears()
            : Promise.resolve(listenYears),
        ]);
        if (cancelled) return;
        events = page.items;
        hasMore = page.hasMore;
        if (currentPeriod === "all" && !currentSearch) {
          listenYears = years;
        }
      } catch (err) {
        if (!cancelled) {
          error =
            err instanceof Error
              ? err.message
              : "Failed to load listen history";
        }
      } finally {
        if (!cancelled) loading = false;
      }
    })();

    return () => {
      cancelled = true;
    };
  });

  async function loadMore() {
    if (loadingMore || !hasMore || loading) return;
    loadingMore = true;
    try {
      const page = await musicApi.getListenEvents({
        limit: PAGE_SIZE,
        offset: events.length,
        period,
        q: debouncedSearch.trim() || undefined,
      });
      events = [...events, ...page.items];
      hasMore = page.hasMore;
    } catch {
      hasMore = false;
    } finally {
      loadingMore = false;
    }
  }

  function eventToSong(event: ListenEvent): SubsonicSong {
    return {
      id: event.trackId,
      title: event.trackTitle,
      artist: event.artistName,
      album: event.albumTitle,
      albumId: event.albumId,
      coverArt: event.coverArtId,
      duration: Math.floor(event.durationMs / 1000),
    };
  }

  function playAt(index: number) {
    music.playTracks(playTracks, index);
  }

  function playAll() {
    if (playTracks.length === 0) return;
    music.playTracks(playTracks, 0);
  }

  function shuffleAll() {
    if (playTracks.length === 0) return;
    music.shuffle = true;
    music.playTracks(playTracks, 0);
  }

  function onPageContextMenu(event: MouseEvent) {
    pageMenu = contextMenuPositionFromEvent(event);
  }

  async function clearHistory() {
    const ok = await confirmDialog.confirm({
      title: "Clear listening history",
      message: "Clear all listening history? This cannot be undone.",
      confirmLabel: "Clear history",
      danger: true,
    });
    if (!ok) return;
    try {
      await music.clearListenHistory();
      events = [];
      hasMore = false;
      listenYears = [];
      toast.success("History cleared");
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to clear history",
      );
    }
  }

  const pageMenuItems = $derived.by((): ContextMenuEntry[] => [
    {
      id: "play",
      label: "Play all",
      icon: "play",
      disabled: playTracks.length === 0,
      onclick: playAll,
    },
    {
      id: "shuffle",
      label: "Shuffle",
      icon: "shuffle",
      disabled: playTracks.length === 0,
      onclick: shuffleAll,
    },
    {
      id: "next",
      label: "Play next",
      icon: "playNext",
      disabled: playTracks.length === 0,
      onclick: () => music.playTracksNext(playTracks),
    },
    {
      id: "queue",
      label: "Add to queue",
      icon: "queueAdd",
      disabled: playTracks.length === 0,
      onclick: () => music.addTracksToQueue(playTracks),
    },
    { id: "clear-sep", separator: true },
    {
      id: "clear",
      label: "Clear history",
      icon: "trash2",
      danger: true,
      disabled: events.length === 0 && music.listenHistory.length === 0,
      onclick: () => void clearHistory(),
    },
  ]);
</script>

<AppShell compactTop>
  <div class="history-page" role="group" oncontextmenu={onPageContextMenu}>
    <MusicBreadcrumbs items={[{ label: "History" }]} />

    <CollectionHero
      typeLabel="Collection"
      title="History"
      meta={heroMeta}
      tone="history"
      icon="history"
      coverSrc={heroCoverSrc}
      coverSeed={events[0]?.trackId ?? "history"}
      playDisabled={events.length === 0}
      onplay={playAll}
      onshuffle={shuffleAll}
      onplaynext={() => music.playTracksNext(playTracks)}
      onqueue={() => music.addTracksToQueue(playTracks)}
    />

    <div class="history-toolbar">
      <div
        class="history-toolbar__periods"
        role="tablist"
        aria-label="Time period"
      >
        {#each Object.entries(STATS_PERIOD_LABELS) as [id, label] (id)}
          <button
            type="button"
            role="tab"
            class="history-toolbar__period"
            class:history-toolbar__period--active={period === id}
            aria-selected={period === id}
            onclick={() => {
              period = id as StatsPeriod;
            }}
          >
            {label}
          </button>
        {/each}
        {#each listenYears as year (year)}
          <button
            type="button"
            role="tab"
            class="history-toolbar__period history-toolbar__period--year"
            class:history-toolbar__period--active={period ===
              statsPeriodForYear(year)}
            aria-selected={period === statsPeriodForYear(year)}
            onclick={() => {
              period = statsPeriodForYear(year);
            }}
          >
            {year}
          </button>
        {/each}
      </div>

      <LocalSearchBox
        bind:value={searchQuery}
        placeholder="Search history"
        resultCount={events.length}
      />
    </div>

    {#if unavailable}
      <LibraryUnavailable />
    {:else if error}
      <EmptyState
        title="Could not load history"
        message={error}
        icon="alertCircle"
      />
    {:else if showInitialLoading}
      <div
        class="history-page__loading"
        role="status"
        aria-busy="true"
        aria-label="Loading history"
      >
        {#each Array.from({ length: 8 }) as _, i (i)}
          <Skeleton variant="row" />
        {/each}
      </div>
    {:else if events.length === 0}
      <EmptyState
        title={hasSearch ? "No matches" : "No listening history yet"}
        message={hasSearch
          ? "Try a different search or time range."
          : "Play something and it will show up here."}
        icon="history"
      />
    {:else}
      <div class="history-page__list">
        <div class="history-page__head">
          <span>Title</span>
          <span>Album</span>
          <span>Played</span>
          <span>Time</span>
        </div>
        <ListenHistoryList {events} onplay={playAt} onNearEnd={loadMore} />
        {#if loadingMore}
          <div class="history-page__more">
            <Spinner />
          </div>
        {/if}
      </div>
    {/if}
  </div>
</AppShell>

{#if pageMenu}
  <ContextMenu
    x={pageMenu.x}
    y={pageMenu.y}
    items={pageMenuItems}
    label="History actions"
    onclose={() => (pageMenu = null)}
  />
{/if}

<style>
  .history-page {
    display: flex;
    flex-direction: column;
    gap: var(--jb-space-6);
  }

  .history-toolbar {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    justify-content: space-between;
    gap: var(--jb-space-3);
  }

  .history-toolbar__periods {
    display: flex;
    flex-wrap: wrap;
    gap: var(--jb-space-2);
  }

  .history-toolbar__period {
    border: none;
    border-radius: var(--jb-radius-full);
    background: var(--jb-surface);
    color: var(--jb-text-muted);
    font-size: 0.8125rem;
    font-weight: 600;
    padding: 0.4rem 0.9rem;
    cursor: pointer;
    transition:
      background var(--jb-transition),
      color var(--jb-transition);
  }

  .history-toolbar__period:hover {
    background: var(--jb-surface-hover);
    color: var(--jb-text);
  }

  .history-toolbar__period--active {
    background: var(--jb-text);
    color: var(--jb-bg);
  }

  .history-toolbar__period--active:hover {
    background: var(--jb-text);
    color: var(--jb-bg);
  }

  .history-toolbar__period--year {
    font-variant-numeric: tabular-nums;
  }

  .history-toolbar :global(.local-search-box) {
    flex: 1;
    min-width: 12rem;
    max-width: 22rem;
  }

  .history-page__loading {
    display: flex;
    flex-direction: column;
    gap: var(--jb-space-2);
  }

  .history-page__more {
    display: flex;
    justify-content: center;
    padding: var(--jb-space-4);
  }

  .history-page__list {
    padding-bottom: var(--jb-space-2);
  }

  .history-page__head {
    display: grid;
    grid-template-columns: minmax(0, 2fr) minmax(0, 1.2fr) 6.5rem 4rem;
    gap: var(--jb-space-3);
    padding: var(--jb-space-3) 0;
    border-bottom: 1px solid var(--jb-border);
    font-size: 0.75rem;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    color: var(--jb-text-subtle);
  }

  @media (max-width: 768px) {
    .history-toolbar {
      flex-direction: column;
      align-items: stretch;
    }

    .history-toolbar__periods {
      overflow-x: auto;
      flex-wrap: nowrap;
      padding-bottom: var(--jb-space-1);
      scrollbar-width: thin;
    }

    .history-toolbar__period {
      flex-shrink: 0;
    }

    .history-toolbar :global(.local-search-box) {
      max-width: none;
    }

    .history-page__head {
      display: none;
    }
  }
</style>
