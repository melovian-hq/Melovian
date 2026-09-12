<script lang="ts">
  import EmptyState from "$lib/components/ui/EmptyState.svelte";
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";
  import AmbientCoverBackdrop from "$lib/components/ui/AmbientCoverBackdrop.svelte";
  import NowPlayingArtHero from "$lib/components/music/NowPlayingArtHero.svelte";
  import NowPlayingQueueTab from "$lib/components/music/NowPlayingQueueTab.svelte";
  import NowPlayingLyricsTab from "$lib/components/music/NowPlayingLyricsTab.svelte";
  import NowPlayingRelatedTab from "$lib/components/music/NowPlayingRelatedTab.svelte";
  import TrackContextMenu from "$lib/components/music/TrackContextMenu.svelte";
  import NowPlayingTvMode from "$lib/components/music/NowPlayingTvMode.svelte";
  import VideoWatchPanel from "$lib/components/music/VideoWatchPanel.svelte";
  import { contextMenuPositionFromEvent } from "$lib/components/ui/context-menu";
  import { music } from "$lib/config/music.svelte";
  import { layout } from "$lib/components/layout/layout.svelte";
  import { videoFeature } from "$lib/video/feature.svelte";
  import { extensionFeatures } from "$lib/extensions/features.svelte";
  import { isInternetRadioTrack } from "$lib/subsonic";
  import {
    COVER_SIZE_NOW_PLAYING,
    COVER_SIZE_PLAYER,
  } from "$lib/music/cover-art-sizes";
  import {
    prefetchNowPlayingCoverArt,
    trackCoverArtUrl,
  } from "$lib/music/cover-art-prefetch";
  import {
    trackCoverPaletteKey,
    trackCoverSeed,
  } from "$lib/music/cover-art-fallback";
  import type { SubsonicSong } from "$lib/subsonic";
  import { syncRelatedTracks } from "$lib/music/now-playing-related";
  import { overlayFade } from "$lib/router/page-motion";
  import { fade } from "svelte/transition";

  type SideTab = "queue" | "related" | "lyrics" | "video";

  let activeTab = $state<SideTab>("queue");
  let relatedTracks = $state<SubsonicSong[]>([]);
  let relatedLoading = $state(false);
  let trackMenu = $state<{
    x: number;
    y: number;
    track: SubsonicSong;
    queueIndex?: number;
  } | null>(null);

  const track = $derived(music.currentTrack);
  const liveStream = $derived(track ? isInternetRadioTrack(track) : false);
  const image = $derived(
    track
      ? trackCoverArtUrl(music.config, track, COVER_SIZE_NOW_PLAYING)
      : null,
  );
  const previewImage = $derived(
    track ? trackCoverArtUrl(music.config, track, COVER_SIZE_PLAYER) : null,
  );

  const tabs = $derived.by(() => {
    if (liveStream) {
      return [
        { id: "queue" as const, label: "Up next" },
        { id: "related" as const, label: "Related" },
      ];
    }
    const base: { id: SideTab; label: string }[] = [
      { id: "queue", label: "Up next" },
      { id: "related", label: "Related" },
    ];
    if (extensionFeatures.lyrics) {
      base.push({ id: "lyrics", label: "Lyrics" });
    }
    if (videoFeature.enabled) {
      base.push({ id: "video", label: "Video" });
    }
    return base;
  });

  $effect(() => {
    if (activeTab === "video" && !videoFeature.enabled) {
      activeTab = "queue";
    }
    if (activeTab === "lyrics" && !extensionFeatures.lyrics) {
      activeTab = "queue";
    }
  });

  $effect(() => {
    if (!track) return;
    prefetchNowPlayingCoverArt(music.config, track);
  });

  $effect(() => {
    if (!track || liveStream || activeTab !== "lyrics") return;
    void music.loadCurrentLyrics();
  });

  $effect(() => {
    return syncRelatedTracks({
      track,
      activeTab,
      library: music.library,
      onTracks: (songs) => {
        relatedTracks = songs;
      },
      onLoading: (loading) => {
        relatedLoading = loading;
      },
    });
  });

  $effect(() => {
    if (!track && layout.tvMode) layout.exitTvMode();
  });

  $effect(() => {
    return () => layout.exitTvMode();
  });

  function onTvShortcut(event: KeyboardEvent) {
    if (event.code !== "KeyT") return;
    if (event.metaKey || event.ctrlKey || event.altKey || event.shiftKey)
      return;
    if (!track || layout.tvMode) return;
    const target = event.target;
    if (target instanceof HTMLElement) {
      const tag = target.tagName;
      if (
        tag === "INPUT" ||
        tag === "TEXTAREA" ||
        tag === "SELECT" ||
        target.isContentEditable
      ) {
        return;
      }
    }
    event.preventDefault();
    layout.enterTvMode();
  }
</script>

<svelte:window onkeydown={onTvShortcut} />

<div class="now-playing-page">
  {#if !track}
    <EmptyState
      title="Nothing playing"
      message="Start a track from your library to see artwork, queue, lyrics, and related songs here."
      icon="disc"
    />
  {:else}
    <div class="now-playing-page__ambient" aria-hidden="true">
      <AmbientCoverBackdrop
        src={previewImage ?? image}
        seed={trackCoverSeed(track)}
        paletteKey={trackCoverPaletteKey(track)}
        opacity={0.72}
        blur={56}
        saturate={1.55}
        scale={1.45}
      />
      <div class="now-playing-page__wash"></div>
    </div>

    <div class="now-playing-page__toolbar">
      <button
        type="button"
        class="now-playing-page__tool now-playing-page__tool--tv"
        onclick={() => layout.enterTvMode()}
        aria-label="TV mode"
        title="TV mode"
      >
        <MdiIcon name="fullscreen" size={20} />
      </button>
    </div>

    <div class="now-playing-layout">
      <div class="now-playing-main">
        <NowPlayingArtHero
          {track}
          {liveStream}
          {image}
          {previewImage}
          oncontextmenu={(event) => {
            if (!track || liveStream) return;
            const pos = contextMenuPositionFromEvent(event);
            if (!pos) return;
            trackMenu = { ...pos, track };
          }}
        >
          {#snippet actions()}
            {#if videoFeature.enabled}
              <button
                type="button"
                class="now-playing-page__video-btn"
                onclick={() => (activeTab = "video")}
                aria-label="Watch video"
                title="Watch video"
              >
                <MdiIcon name="video" size={20} />
              </button>
            {/if}
          {/snippet}
        </NowPlayingArtHero>
      </div>

      <section class="now-playing-side" aria-label="Queue and related">
        <div
          class="now-playing-side__queue"
          class:now-playing-side__queue--full={liveStream}
        >
          <div
            class="now-playing-side__tabs"
            role="tablist"
            aria-label="Queue panels"
          >
            {#each tabs as tab (tab.id)}
              <button
                type="button"
                role="tab"
                class="now-playing-side__tab"
                class:now-playing-side__tab--active={activeTab === tab.id}
                aria-selected={activeTab === tab.id}
                onclick={() => (activeTab = tab.id)}
              >
                {tab.label}
              </button>
            {/each}
          </div>

          <div class="now-playing-side__panel" role="tabpanel">
            {#key activeTab}
              <div class="now-playing-side__panel-body" in:fade={overlayFade()}>
                {#if activeTab === "queue"}
                  <NowPlayingQueueTab
                    ontrackmenu={(menu) => {
                      trackMenu = menu;
                    }}
                  />
                {:else if activeTab === "lyrics"}
                  <NowPlayingLyricsTab />
                {:else if activeTab === "related"}
                  <NowPlayingRelatedTab {relatedTracks} {relatedLoading} />
                {:else if activeTab === "video"}
                  <VideoWatchPanel {track} embedded />
                {/if}
              </div>
            {/key}
          </div>
        </div>
      </section>
    </div>
  {/if}
</div>

{#if layout.tvMode && track}
  <NowPlayingTvMode />
{/if}

{#if trackMenu}
  <TrackContextMenu
    track={trackMenu.track}
    x={trackMenu.x}
    y={trackMenu.y}
    onclose={() => (trackMenu = null)}
    onRemove={trackMenu.queueIndex != null
      ? () => {
          const index = trackMenu!.queueIndex!;
          music.removeFromQueue(index);
          trackMenu = null;
        }
      : undefined}
  />
{/if}

<style>
  .now-playing-page {
    position: relative;
    isolation: isolate;
    flex: 1 1 auto;
    width: 100%;
    height: 100%;
    min-height: 0;
    min-width: 0;
    overflow: hidden;
    display: flex;
    flex-direction: column;
  }

  .now-playing-page :global(.empty-state:not(.empty-state--embedded)) {
    margin-block: auto;
  }

  .now-playing-page__ambient {
    position: absolute;
    inset: 0;
    z-index: 0;
    pointer-events: none;
    overflow: hidden;
  }

  .now-playing-page__wash {
    position: absolute;
    inset: 0;
    background:
      linear-gradient(
        to bottom,
        color-mix(in srgb, var(--jb-bg) 18%, transparent) 0%,
        color-mix(in srgb, var(--jb-bg) 42%, transparent) 42%,
        color-mix(in srgb, var(--jb-bg) 78%, transparent) 100%
      ),
      radial-gradient(
        120% 80% at 50% 0%,
        transparent 20%,
        color-mix(in srgb, var(--jb-bg) 55%, transparent) 100%
      );
  }

  .now-playing-page__toolbar {
    position: absolute;
    top: calc(var(--jb-fill-pad-top, var(--jb-space-4)) + 0.5rem);
    right: var(--jb-space-6);
    z-index: 2;
    display: flex;
    align-items: center;
    gap: var(--jb-space-2);
  }

  .now-playing-page__tool {
    display: grid;
    place-items: center;
    width: 2.5rem;
    height: 2.5rem;
    border: 1px solid rgb(255 255 255 / 0.18);
    border-radius: var(--jb-radius-full);
    background: rgb(0 0 0 / 0.28);
    color: rgb(255 255 255 / 0.9);
    cursor: pointer;
  }

  .now-playing-page__tool:hover {
    background: rgb(0 0 0 / 0.42);
  }

  .now-playing-page__tool--tv {
    margin-right: var(--jb-space-5);
  }

  .now-playing-page__video-btn {
    display: grid;
    place-items: center;
    width: 2.75rem;
    height: 2.75rem;
    border: 1px solid rgb(255 255 255 / 0.22);
    border-radius: var(--jb-radius-full);
    background: rgb(0 0 0 / 0.28);
    color: rgb(255 255 255 / 0.9);
    cursor: pointer;
  }

  .now-playing-page__video-btn:hover {
    background: rgb(0 0 0 / 0.42);
  }

  .now-playing-layout {
    position: relative;
    z-index: 1;
    flex: 1 1 auto;
    min-height: 0;
    width: 100%;
    height: 100%;
    display: grid;
    grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
    gap: var(--jb-space-6);
    align-items: stretch;
    overflow: hidden;
    padding: var(
        --jb-fill-pad-top,
        calc(var(--jb-window-chrome-offset, 0px) + var(--jb-space-4))
      )
      var(--jb-space-6) var(--jb-space-4);
  }

  .now-playing-main {
    display: flex;
    flex-direction: column;
    min-width: 0;
    min-height: 0;
    height: 100%;
    overflow: hidden;
  }

  .now-playing-main :global(.now-playing-art) {
    flex: 1 1 auto;
    min-height: 0;
    height: 100%;
  }

  .now-playing-side {
    display: flex;
    flex-direction: column;
    align-self: stretch;
    min-height: 0;
    min-width: 0;
    overflow: hidden;
    border: 1px solid color-mix(in srgb, var(--jb-border) 55%, transparent);
    border-radius: var(--jb-radius-xl);
    background: color-mix(in srgb, var(--jb-bg-elevated) 42%, transparent);
    backdrop-filter: blur(16px);
  }

  .now-playing-side__queue {
    flex: 1 1 auto;
    min-height: 0;
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  .now-playing-side__queue--full {
    flex: 1 1 auto;
  }

  .now-playing-side__tabs {
    display: flex;
    flex-shrink: 0;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--jb-space-1);
    padding: var(--jb-space-3) var(--jb-space-4) 0;
    border-bottom: 1px solid var(--jb-border);
    overflow: hidden;
  }

  .now-playing-side__tab {
    position: relative;
    border: none;
    background: transparent;
    color: var(--jb-text-muted);
    font-size: 0.75rem;
    font-weight: 700;
    letter-spacing: 0.08em;
    text-transform: uppercase;
    padding: var(--jb-space-3) var(--jb-space-2);
    cursor: pointer;
    white-space: nowrap;
    border-radius: var(--jb-radius-sm) var(--jb-radius-sm) 0 0;
    transition: color var(--jb-transition);
  }

  .now-playing-side__tab:hover {
    color: var(--jb-text);
  }

  .now-playing-side__tab:focus-visible {
    outline: none;
    box-shadow: var(--jb-focus-ring);
    color: var(--jb-text);
  }

  .now-playing-side__tab--active {
    color: var(--jb-text);
  }

  .now-playing-side__tab--active::after {
    content: "";
    position: absolute;
    left: 0;
    right: 0;
    bottom: 0;
    height: 2px;
    background: var(--jb-text);
  }

  .now-playing-side__panel {
    flex: 1;
    min-height: 0;
    display: flex;
    flex-direction: column;
    padding: var(--jb-space-4);
    overflow: hidden;
  }

  .now-playing-side__panel-body {
    flex: 1;
    min-height: 0;
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  @media (max-width: 960px) {
    .now-playing-layout {
      grid-template-columns: 1fr;
      grid-template-rows: auto minmax(0, 1fr);
      gap: var(--jb-space-4);
      padding-left: var(--jb-space-4);
      padding-right: var(--jb-space-4);
    }

    .now-playing-page__toolbar {
      right: var(--jb-space-4);
    }

    .now-playing-main {
      height: auto;
      overflow: visible;
    }

    .now-playing-main :global(.now-playing-art) {
      height: auto;
      flex: 0 0 auto;
    }

    .now-playing-side {
      min-height: 0;
      border-left: none;
      border-right: none;
      border-radius: var(--jb-radius-xl) var(--jb-radius-xl) 0 0;
      background: color-mix(in srgb, var(--jb-bg) 28%, transparent);
    }

    .now-playing-side__queue {
      flex: 1 1 auto;
      min-height: 0;
    }
  }

  @media (max-width: 768px) {
    .now-playing-page {
      overflow-y: auto;
      overflow-x: hidden;
      height: auto;
      min-height: 100%;
    }

    .now-playing-layout {
      display: flex;
      flex-direction: column;
      height: auto;
      overflow: visible;
      gap: var(--jb-space-4);
      padding-bottom: var(--jb-space-6);
    }

    .now-playing-main {
      flex: 0 0 auto;
      height: auto;
      overflow: visible;
    }

    .now-playing-side {
      flex: 1 1 auto;
      min-height: min(45svh, 24rem);
      overflow: hidden;
    }

    .now-playing-page__tool--tv {
      display: none;
    }
  }

  @media (max-height: 700px) and (min-width: 961px) {
    .now-playing-layout {
      gap: var(--jb-space-4);
    }
  }
</style>
