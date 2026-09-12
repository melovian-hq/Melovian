<script lang="ts">
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";
  import EmptyState from "$lib/components/ui/EmptyState.svelte";
  import Spinner from "$lib/components/ui/Spinner.svelte";
  import LyricsDisplay from "$lib/components/music/LyricsDisplay.svelte";
  import { music } from "$lib/config/music.svelte";
  import { MediaQuery } from "svelte/reactivity";
  import { layout, MOBILE_MEDIA } from "$lib/components/layout/layout.svelte";
  import { playerBottomInset } from "$lib/music/player-bottom-inset";
  import { filterLyricsLines } from "$lib/music/lyrics";
  import {
    clampLyricsPanelPosition,
    clampLyricsPanelSize,
    loadLyricsPanelPosition,
    loadLyricsPanelSize,
    saveLyricsPanelPosition,
    saveLyricsPanelSize,
    type PanelSize,
    type QueuePanelPosition,
  } from "$lib/music/prefs";
  import Link from "$lib/router/Link.svelte";
  import { extensionFeatures } from "$lib/extensions/features.svelte";

  const DEFAULT_PANEL_SIZE: PanelSize = { width: 352, height: 448 };

  const bottomInset = $derived(
    playerBottomInset(music.playerVisible, music.playerLayout, {
      mobileNav: layout.isMobileViewport,
    }),
  );

  let lyricsAutoScroll = $state(true);
  let searchQuery = $state("");
  let panelEl = $state<HTMLElement | null>(null);
  let panelPosition = $state<QueuePanelPosition | null>(
    loadLyricsPanelPosition(),
  );
  let panelSize = $state<PanelSize>(
    loadLyricsPanelSize() ?? DEFAULT_PANEL_SIZE,
  );
  const mobileQuery = new MediaQuery(MOBILE_MEDIA);
  let isMobile = $derived(mobileQuery.current);
  let draggingPanel = $state(false);
  let resizingPanel = $state(false);
  let dragOrigin = { pointerX: 0, pointerY: 0, startX: 0, startY: 0 };
  let resizeStartX = 0;
  let resizeStartY = 0;
  let resizeStartSize: PanelSize = DEFAULT_PANEL_SIZE;

  const displayLyrics = $derived(
    music.currentLyrics
      ? filterLyricsLines(music.currentLyrics, searchQuery)
      : null,
  );

  const isPositioned = $derived(panelPosition !== null && !isMobile);

  function ensurePanelPosition() {
    if (!panelEl || panelPosition || isMobile) return;
    const rect = panelEl.getBoundingClientRect();
    panelPosition = { x: rect.left, y: rect.top };
    panelSize = clampLyricsPanelSize({
      width: rect.width,
      height: rect.height,
    });
  }

  function clampPanelPosition(
    position: QueuePanelPosition,
  ): QueuePanelPosition {
    if (!panelEl) return position;
    return clampLyricsPanelPosition(
      position,
      panelEl.offsetWidth,
      panelEl.offsetHeight,
    );
  }

  function onHeaderPointerDown(event: PointerEvent) {
    if (isMobile || !panelEl) return;
    if (event.button !== 0) return;
    const target = event.target as HTMLElement | null;
    if (target?.closest(".jb-no-drag")) return;

    ensurePanelPosition();
    if (!panelPosition) return;

    draggingPanel = true;
    dragOrigin = {
      pointerX: event.clientX,
      pointerY: event.clientY,
      startX: panelPosition.x,
      startY: panelPosition.y,
    };
    (event.currentTarget as HTMLElement).setPointerCapture(event.pointerId);
    event.preventDefault();
  }

  function onHeaderPointerMove(event: PointerEvent) {
    if (!draggingPanel || !panelPosition) return;
    panelPosition = clampPanelPosition({
      x: dragOrigin.startX + (event.clientX - dragOrigin.pointerX),
      y: dragOrigin.startY + (event.clientY - dragOrigin.pointerY),
    });
  }

  function onHeaderPointerUp(event: PointerEvent) {
    if (!draggingPanel) return;
    draggingPanel = false;
    if (panelPosition) saveLyricsPanelPosition(panelPosition);
    try {
      (event.currentTarget as HTMLElement).releasePointerCapture(
        event.pointerId,
      );
    } catch {
      // Capture may already be released.
    }
  }

  function onResizePointerDown(event: PointerEvent) {
    if (isMobile || !panelEl) return;

    resizingPanel = true;
    resizeStartX = event.clientX;
    resizeStartY = event.clientY;
    resizeStartSize = {
      width: panelEl.offsetWidth,
      height: panelEl.offsetHeight,
    };
    panelSize = { ...resizeStartSize };
    (event.currentTarget as HTMLElement).setPointerCapture(event.pointerId);
    event.stopPropagation();
    event.preventDefault();
  }

  function onResizePointerMove(event: PointerEvent) {
    if (!resizingPanel) return;

    if (isPositioned && panelPosition) {
      const next = clampLyricsPanelSize({
        width: resizeStartSize.width + (event.clientX - resizeStartX),
        height: resizeStartSize.height + (event.clientY - resizeStartY),
      });
      panelSize = next;
      panelPosition = clampPanelPosition(panelPosition);
      return;
    }

    panelSize = clampLyricsPanelSize({
      width: resizeStartSize.width + (resizeStartX - event.clientX),
      height: resizeStartSize.height + (event.clientY - resizeStartY),
    });
  }

  function onResizePointerUp(event: PointerEvent) {
    if (!resizingPanel) return;
    resizingPanel = false;
    saveLyricsPanelSize(panelSize);
    (event.currentTarget as HTMLElement).releasePointerCapture(event.pointerId);
  }

  $effect(() => {
    const onResize = () => {
      panelSize = clampLyricsPanelSize(panelSize);
      if (panelPosition && panelEl) {
        panelPosition = clampPanelPosition(panelPosition);
      }
    };
    window.addEventListener("resize", onResize);

    return () => {
      window.removeEventListener("resize", onResize);
    };
  });
</script>

<aside
  bind:this={panelEl}
  class="lyrics-sidebar"
  class:lyrics-sidebar--positioned={isPositioned}
  class:lyrics-sidebar--sized={!isMobile}
  class:lyrics-sidebar--dragging={draggingPanel}
  class:lyrics-sidebar--resizing={resizingPanel}
  style:bottom={isMobile ? bottomInset : undefined}
  style:left={isPositioned ? `${panelPosition?.x}px` : undefined}
  style:top={isPositioned ? `${panelPosition?.y}px` : undefined}
  style:width={!isMobile ? `${panelSize.width}px` : undefined}
  style:height={!isMobile ? `${panelSize.height}px` : undefined}
  aria-label="Lyrics"
>
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <header
    class="lyrics-sidebar__header"
    class:lyrics-sidebar__header--draggable={!isMobile}
    aria-label={!isMobile ? "Move lyrics panel" : undefined}
    onpointerdown={!isMobile ? onHeaderPointerDown : undefined}
    onpointermove={!isMobile ? onHeaderPointerMove : undefined}
    onpointerup={!isMobile ? onHeaderPointerUp : undefined}
    onpointercancel={!isMobile ? onHeaderPointerUp : undefined}
  >
    <div
      class="lyrics-sidebar__heading"
      class:lyrics-sidebar__drag-handle={!isMobile}
      class:lyrics-sidebar__drag-handle--enabled={!isMobile}
    >
      <h2 class="lyrics-sidebar__title">Lyrics</h2>
      {#if music.currentLyrics}
        <p class="lyrics-sidebar__meta">
          {music.currentLyrics.title}
          {#if music.currentLyrics.artist}
            · {music.currentLyrics.artist}
          {/if}
        </p>
      {/if}
    </div>
    <div class="lyrics-sidebar__actions jb-no-drag">
      {#if music.currentTrack && !music.lyricsLoading}
        <button
          type="button"
          class="lyrics-sidebar__icon-btn"
          title="Refetch lyrics"
          aria-label="Refetch lyrics"
          disabled={music.lyricsFetching}
          onclick={() => void music.fetchCurrentLyrics()}
        >
          <MdiIcon name="refresh" size={16} />
        </button>
      {/if}
      {#if music.currentLyrics}
        {#if music.currentLyrics.synced}
          <button
            type="button"
            class="lyrics-sidebar__icon-btn"
            class:lyrics-sidebar__icon-btn--active={lyricsAutoScroll}
            title={lyricsAutoScroll
              ? "Disable auto-scroll"
              : "Enable auto-scroll"}
            aria-label={lyricsAutoScroll
              ? "Disable auto-scroll"
              : "Enable auto-scroll"}
            aria-pressed={lyricsAutoScroll}
            onclick={() => (lyricsAutoScroll = !lyricsAutoScroll)}
          >
            <MdiIcon
              name={lyricsAutoScroll ? "scrollFollowActive" : "scrollFollow"}
              size={16}
            />
          </button>
        {/if}
        <button
          type="button"
          class="lyrics-sidebar__icon-btn"
          title="Export lyrics"
          aria-label="Export lyrics"
          onclick={() =>
            import("$lib/music/lyrics-export").then(({ exportLyrics }) =>
              exportLyrics(music.currentLyrics!, music.currentTrack?.title),
            )}
        >
          <MdiIcon name="export" size={16} />
        </button>
      {/if}
      <Link href="/music/lyrics" class="lyrics-sidebar__expand">Open page</Link>
      <button
        type="button"
        class="lyrics-sidebar__close"
        aria-label="Close lyrics"
        onclick={() => music.toggleLyricsPanel()}
      >
        <MdiIcon name="x" size={16} />
      </button>
    </div>
  </header>

  <div class="lyrics-sidebar__body">
    {#if music.lyricsLoading}
      <div class="lyrics-sidebar__loading"><Spinner /></div>
    {:else if !music.currentTrack}
      <EmptyState
        title="Nothing playing"
        message="Start a track to load lyrics here."
        icon="disc"
        embedded
      />
    {:else if !music.currentLyrics}
      <EmptyState
        title="No lyrics found"
        message="This track has no lyrics yet. Fetch from enabled providers or add lyrics on your server."
        icon="lyrics"
        embedded
      />
      <button
        type="button"
        class="lyrics-sidebar__fetch"
        disabled={music.lyricsFetching}
        onclick={() => void music.fetchCurrentLyrics()}
      >
        {music.lyricsFetching ? "Fetching lyrics…" : "Fetch lyrics"}
      </button>
      {#if extensionFeatures.lyricsWhisper}
        <button
          type="button"
          class="lyrics-sidebar__fetch"
          disabled={music.lyricsFetching}
          onclick={() => void music.generateWhisperLyrics()}
        >
          {music.lyricsFetching
            ? "Transcribing…"
            : "Generate synced lyrics with Whisper"}
        </button>
      {/if}
    {:else}
      {#if music.currentLyrics.synced}
        <p class="lyrics-sidebar__hint">Synced lyrics</p>
      {/if}
      <label class="lyrics-sidebar__search">
        <MdiIcon name="search" size={15} />
        <input
          type="search"
          placeholder="Search lyrics"
          bind:value={searchQuery}
          autocomplete="off"
        />
      </label>
      <div class="lyrics-sidebar__scroll">
        {#if displayLyrics && displayLyrics.lines.length === 0}
          <p class="lyrics-sidebar__empty">No matching lines.</p>
        {:else if displayLyrics}
          <LyricsDisplay
            lyrics={displayLyrics}
            currentTimeMs={music.currentTime * 1000}
            playing={music.playing}
            autoScroll={lyricsAutoScroll &&
              music.playing &&
              music.currentLyrics.synced}
            onSeek={(startMs) => music.seekToLyricLine(startMs)}
          />
        {/if}
      </div>
    {/if}
  </div>

  {#if !isMobile}
    <button
      type="button"
      class="lyrics-sidebar__resize-handle jb-no-drag"
      class:lyrics-sidebar__resize-handle--se={isPositioned}
      aria-label="Resize lyrics panel"
      title="Drag to resize"
      onpointerdown={onResizePointerDown}
      onpointermove={onResizePointerMove}
      onpointerup={onResizePointerUp}
      onpointercancel={onResizePointerUp}
    >
      <MdiIcon
        name={isPositioned ? "resizeBottomRight" : "resizeBottomLeft"}
        size={14}
      />
    </button>
  {/if}
</aside>

<style>
  .lyrics-sidebar {
    position: fixed;
    top: calc(var(--jb-topbar-height) + env(safe-area-inset-top, 0px));
    right: 0;
    z-index: 64;
    width: min(22rem, 100vw);
    max-height: min(calc(100vh - var(--jb-topbar-height) - 7rem), 28rem);
    display: flex;
    flex-direction: column;
    border-left: 1px solid var(--jb-border);
    background: color-mix(in srgb, var(--jb-bg-elevated) 96%, transparent);
    backdrop-filter: blur(16px);
    box-shadow: var(--jb-shadow-lg);
    color: var(--jb-text);
  }

  .lyrics-sidebar--sized {
    max-height: none;
  }

  .lyrics-sidebar--positioned {
    right: auto;
    max-height: none;
    border: 1px solid var(--jb-border);
    border-radius: var(--jb-radius-lg);
  }

  .lyrics-sidebar--dragging,
  .lyrics-sidebar--resizing {
    user-select: none;
  }

  .lyrics-sidebar__header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: var(--jb-space-3);
    padding: var(--jb-space-4);
    border-bottom: 1px solid var(--jb-border);
    flex-shrink: 0;
  }

  .lyrics-sidebar__header--draggable {
    touch-action: none;
    user-select: none;
  }

  .lyrics-sidebar__drag-handle {
    flex: 1;
    min-width: 0;
    touch-action: none;
    user-select: none;
  }

  .lyrics-sidebar__drag-handle--enabled {
    cursor: grab;
  }

  .lyrics-sidebar--dragging .lyrics-sidebar__header--draggable,
  .lyrics-sidebar--dragging .lyrics-sidebar__drag-handle--enabled {
    cursor: grabbing;
  }

  .lyrics-sidebar__heading {
    min-width: 0;
    border: none;
    background: none;
    padding: 0;
    margin: 0;
    text-align: left;
    font: inherit;
    color: inherit;
  }

  .lyrics-sidebar__title {
    margin: 0;
    font-size: 0.875rem;
    font-weight: 700;
    letter-spacing: 0.06em;
    text-transform: uppercase;
    color: var(--jb-text-subtle);
  }

  .lyrics-sidebar__meta {
    margin: var(--jb-space-1) 0 0;
    font-size: 0.8125rem;
    color: var(--jb-text-muted);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .lyrics-sidebar__actions {
    display: flex;
    align-items: center;
    gap: var(--jb-space-2);
    flex-shrink: 0;
  }

  :global(a.lyrics-sidebar__expand) {
    font-size: 0.8125rem;
    font-weight: 600;
    color: var(--jb-accent);
    text-decoration: none;
  }

  .lyrics-sidebar__icon-btn {
    display: inline-grid;
    place-items: center;
    width: 1.75rem;
    height: 1.75rem;
    border: 1px solid var(--jb-border);
    border-radius: var(--jb-radius-sm);
    background: var(--jb-surface);
    color: var(--jb-text-muted);
    cursor: pointer;
  }

  .lyrics-sidebar__icon-btn:hover:not(:disabled) {
    color: var(--jb-text);
    background: var(--jb-surface-hover);
  }

  .lyrics-sidebar__icon-btn--active {
    color: var(--jb-accent);
    border-color: color-mix(in srgb, var(--jb-accent) 45%, var(--jb-border));
    background: color-mix(in srgb, var(--jb-accent) 12%, var(--jb-surface));
  }

  .lyrics-sidebar__icon-btn:disabled {
    opacity: 0.55;
    cursor: not-allowed;
  }

  .lyrics-sidebar__close {
    display: grid;
    place-content: center;
    width: 1.75rem;
    height: 1.75rem;
    border: none;
    border-radius: var(--jb-radius-full);
    background: transparent;
    color: var(--jb-text-muted);
    cursor: pointer;
  }

  .lyrics-sidebar__close:hover {
    background: var(--jb-surface-hover);
    color: var(--jb-text);
  }

  .lyrics-sidebar__search {
    display: flex;
    align-items: center;
    gap: var(--jb-space-2);
    flex-shrink: 0;
    margin-bottom: var(--jb-space-3);
    padding-bottom: var(--jb-space-2);
    border: none;
    border-bottom: 1px solid var(--jb-border);
    color: var(--jb-text-subtle);
  }

  .lyrics-sidebar__search input {
    flex: 1;
    min-width: 0;
    border: none;
    outline: none;
    background: transparent;
    color: var(--jb-text);
    font: inherit;
    font-size: 0.8125rem;
  }

  .lyrics-sidebar__search input::-webkit-search-cancel-button {
    display: none;
  }

  .lyrics-sidebar__body {
    flex: 1;
    min-height: 0;
    display: flex;
    flex-direction: column;
    padding: var(--jb-space-4);
    overflow: hidden;
  }

  .lyrics-sidebar__scroll {
    flex: 1;
    min-height: 0;
    overflow-y: auto;
    padding-right: var(--jb-space-1);
  }

  .lyrics-sidebar__loading {
    flex: 1;
    display: grid;
    place-content: center;
    min-height: 8rem;
  }

  .lyrics-sidebar__empty {
    margin: 0;
    color: var(--jb-text-muted);
    text-align: center;
  }

  .lyrics-sidebar__hint {
    margin: 0 0 var(--jb-space-3);
    font-size: 0.6875rem;
    color: var(--jb-text-subtle);
    text-transform: uppercase;
    letter-spacing: 0.08em;
    font-weight: 700;
    flex-shrink: 0;
  }

  .lyrics-sidebar__fetch {
    align-self: center;
    margin-top: var(--jb-space-3);
    border: 1px solid var(--jb-border);
    border-radius: var(--jb-radius-full);
    background: var(--jb-surface);
    color: var(--jb-text);
    font-size: 0.8125rem;
    font-weight: 600;
    padding: 0.5rem 1rem;
    cursor: pointer;
  }

  .lyrics-sidebar__fetch:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }

  .lyrics-sidebar__resize-handle {
    position: absolute;
    left: 0;
    bottom: 0;
    display: grid;
    place-items: center;
    width: 1.5rem;
    height: 1.5rem;
    padding: 0;
    border: none;
    border-radius: var(--jb-radius-sm) 0 0 0;
    cursor: nesw-resize;
    touch-action: none;
    background: transparent;
    color: var(--jb-text-muted);
  }

  .lyrics-sidebar__resize-handle--se {
    left: auto;
    right: 0;
    border-radius: 0 var(--jb-radius-sm) 0 0;
    cursor: nwse-resize;
  }

  .lyrics-sidebar__resize-handle:hover,
  .lyrics-sidebar--resizing .lyrics-sidebar__resize-handle {
    color: var(--jb-accent);
    background: color-mix(in srgb, var(--jb-accent) 12%, transparent);
  }

  @media (max-width: 768px) {
    .lyrics-sidebar {
      left: 0;
      width: auto;
      top: auto;
      bottom: auto;
      max-height: min(42vh, 24rem);
      border-left: none;
      border-top: 1px solid var(--jb-border);
      border-radius: var(--jb-radius-xl) var(--jb-radius-xl) 0 0;
    }
  }
</style>
