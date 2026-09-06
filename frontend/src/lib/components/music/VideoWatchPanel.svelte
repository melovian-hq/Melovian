<script lang="ts">
  import Button from "$lib/components/ui/Button.svelte";
  import EmptyState from "$lib/components/ui/EmptyState.svelte";
  import Field from "$lib/components/ui/Field.svelte";
  import Spinner from "$lib/components/ui/Spinner.svelte";
  import { toast } from "$lib/ui/toast.svelte";
  import {
    deleteTrackVideoLink,
    getTrackVideoLink,
    resolveVideo,
    saveTrackVideoLink,
    searchVideos,
  } from "$lib/video/api";
  import { videoFeature } from "$lib/video/feature.svelte";
  import {
    buildEmbedUrl,
    displayVideoTitle,
    musicVideoSearchQuery,
    searchReady,
    type TrackVideoLink,
    type VideoSearchHit,
    type VideoSearchProvider,
    type VideoSettings,
    type VideoSource,
  } from "$lib/video/ids";
  import type { SubsonicSong } from "$lib/subsonic";
  import {
    pauseMusicForVideo,
    registerVideoPause,
  } from "$lib/video/playback-gate.svelte";

  interface Props {
    track: SubsonicSong;
    /** Compact layout for the Now Playing side panel. */
    embedded?: boolean;
  }

  let { track, embedded = false }: Props = $props();

  let link = $state<TrackVideoLink | null>(null);
  let loadingLink = $state(true);
  let settings = $state<VideoSettings | null>(null);
  let searching = $state(false);
  let searchQuery = $state("");
  let results = $state<VideoSearchHit[]>([]);
  let searchError = $state<string | null>(null);
  let searched = $state(false);
  let lastProvider = $state<VideoSearchProvider>("invidious");
  let embedUrl = $state<string | null>(null);
  let embedTitle = $state("");
  let embedAuthor = $state("");
  let resolving = $state(false);
  let resolveError = $state<string | null>(null);
  let heldEmbedUrl = $state<string | null>(null);

  const enabled = $derived(Boolean(settings?.enabled));
  const canSearch = $derived(settings ? searchReady(settings) : false);
  const linkedTitle = $derived(
    link ? displayVideoTitle(link.title, link.videoId) : "",
  );

  $effect(() => {
    return registerVideoPause(() => {
      if (!embedUrl) return;
      heldEmbedUrl = embedUrl;
      embedUrl = null;
    });
  });

  function restoreHeldEmbed() {
    if (!heldEmbedUrl) return;
    embedUrl = heldEmbedUrl;
    heldEmbedUrl = null;
    pauseMusicForVideo();
  }

  function focusPlayer() {
    pauseMusicForVideo();
  }

  $effect(() => {
    const trackId = track.id;
    const title = track.title;
    const artist = track.artist;
    const query = musicVideoSearchQuery({ title, artist });
    let cancelled = false;
    const stillCurrent = () => !cancelled && track.id === trackId;
    loadingLink = true;
    results = [];
    searchError = null;
    searched = false;
    searchQuery = query;
    embedUrl = null;
    embedTitle = "";
    embedAuthor = "";
    resolveError = null;
    heldEmbedUrl = null;

    void videoFeature
      .refresh()
      .then(async (merged) => {
        if (!stillCurrent()) return;
        settings = merged;
        if (!merged.enabled) {
          link = null;
          return;
        }
        const existing = await getTrackVideoLink(trackId);
        if (!stillCurrent()) return;
        link = existing;
        if (existing) {
          await loadEmbed(existing, merged, trackId);
          return;
        }
        if (searchReady(merged)) {
          await runSearch(query, true, trackId);
        }
      })
      .catch(() => {
        if (!stillCurrent()) return;
        settings = videoFeature.settings;
        link = null;
      })
      .finally(() => {
        if (stillCurrent()) loadingLink = false;
      });

    return () => {
      cancelled = true;
    };
  });

  async function loadEmbed(
    saved: TrackVideoLink,
    cfg: VideoSettings,
    forTrackId = track.id,
  ) {
    resolving = true;
    resolveError = null;
    try {
      const localEmbed = buildEmbedUrl(cfg, saved.source, saved.videoId);
      const knownTitle = displayVideoTitle(saved.title, saved.videoId);
      if (localEmbed && knownTitle !== "Music video") {
        if (track.id !== forTrackId) return;
        embedUrl = localEmbed;
        embedTitle = knownTitle;
        embedAuthor = "";
        return;
      }

      const resolved = await resolveVideo({
        source: saved.source,
        id: saved.videoId,
        title: saved.title,
      });
      if (track.id !== forTrackId) return;
      embedUrl = resolved.embedUrl ?? localEmbed;
      embedTitle = displayVideoTitle(
        resolved.title || saved.title,
        saved.videoId,
      );
      embedAuthor = resolved.author ?? "";
      if (!embedUrl) {
        resolveError = "No embed URL from video settings";
      }
    } catch (err) {
      if (track.id !== forTrackId) return;
      embedUrl = null;
      resolveError = err instanceof Error ? err.message : String(err);
    } finally {
      if (track.id === forTrackId) resolving = false;
    }
  }

  async function runSearch(
    q = searchQuery,
    fromAuto = false,
    forTrackId = track.id,
  ) {
    const query = q.trim();
    if (!query || !settings || !searchReady(settings)) return;
    searching = true;
    searchError = null;
    searched = true;
    try {
      const payload = await searchVideos(query);
      if (track.id !== forTrackId) return;
      results = payload.results;
      lastProvider = payload.provider;
      if (
        fromAuto &&
        results.length > 0 &&
        !link &&
        looksLikeMusicVideo(results[0], track)
      ) {
        await pickResult(results[0], false, forTrackId);
      }
    } catch (err) {
      if (track.id !== forTrackId) return;
      results = [];
      searchError = err instanceof Error ? err.message : String(err);
    } finally {
      if (track.id === forTrackId) searching = false;
    }
  }

  function looksLikeMusicVideo(
    hit: VideoSearchHit,
    song: SubsonicSong,
  ): boolean {
    const hay = `${hit.title} ${hit.author ?? ""}`.toLowerCase();
    const title = (song.title ?? "").trim().toLowerCase();
    if (!title) return false;
    if (!hay.includes(title)) return false;
    const artist = (song.artist ?? "").trim().toLowerCase();
    if (artist && !hay.includes(artist.split(/\s+/)[0] ?? artist)) {
      return false;
    }
    return true;
  }

  async function pickResult(
    hit: VideoSearchHit,
    playAfter = true,
    forTrackId = track.id,
  ) {
    if (track.id !== forTrackId) return;
    const source: VideoSource =
      lastProvider === "youtube" ? "youtube" : "invidious";
    try {
      const saved = await saveTrackVideoLink(forTrackId, {
        source,
        videoId: hit.id,
        title: hit.title,
      });
      if (track.id !== forTrackId) return;
      link = saved;
      results = [];
      searched = false;
      if (settings) await loadEmbed(saved, settings, forTrackId);
      if (playAfter && track.id === forTrackId && !embedUrl) {
        toast.error("Linked, but embed failed to load");
      }
    } catch (err) {
      if (track.id !== forTrackId) return;
      toast.error(err instanceof Error ? err.message : "Could not save link");
    }
  }

  async function clearLink() {
    const forTrackId = track.id;
    try {
      await deleteTrackVideoLink(forTrackId);
      if (track.id !== forTrackId) return;
      link = null;
      embedUrl = null;
      heldEmbedUrl = null;
      embedTitle = "";
      embedAuthor = "";
      resolveError = null;
      toast.success("Video link cleared");
      if (canSearch) {
        await runSearch(musicVideoSearchQuery(track), false, forTrackId);
      }
    } catch (err) {
      if (track.id !== forTrackId) return;
      toast.error(err instanceof Error ? err.message : "Could not clear link");
    }
  }
</script>

{#if loadingLink}
  <div
    class="video-watch video-watch--loading"
    class:video-watch--embedded={embedded}
  >
    <Spinner />
    <p class="video-watch__loading-label">Looking up video…</p>
  </div>
{:else if !enabled}
  <EmptyState
    title="Videos are off"
    message="Turn on videos in Settings → Video to watch music videos here."
    icon="video"
    {embedded}
  />
{:else}
  <div class="video-watch" class:video-watch--embedded={embedded}>
    {#if resolving}
      <div class="video-watch--loading">
        <Spinner />
        <p class="video-watch__loading-label">Loading player…</p>
      </div>
    {:else if embedUrl && link}
      <div class="video-watch__player">
        <p class="video-watch__linked-title">{embedTitle || linkedTitle}</p>
        {#if embedAuthor}
          <p class="video-watch__author">{embedAuthor}</p>
        {/if}
        <div
          class="video-watch__frame"
          role="presentation"
          onclick={focusPlayer}
          onfocusin={focusPlayer}
        >
          <iframe
            title={embedTitle || linkedTitle || "Music video"}
            src={embedUrl}
            allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; fullscreen"
          ></iframe>
        </div>
        <div class="video-watch__row">
          <Button variant="ghost" onclick={() => void clearLink()}>
            Clear link
          </Button>
          <Button
            variant="ghost"
            onclick={() => void runSearch(musicVideoSearchQuery(track), false)}
          >
            Find another
          </Button>
        </div>
      </div>
    {:else if heldEmbedUrl && link}
      <div class="video-watch__player">
        <p class="video-watch__linked-title">{embedTitle || linkedTitle}</p>
        <EmptyState
          title="Video paused for music"
          message="Music started, so the embed was unloaded. Resume the video when you want it back."
          icon="video"
          embedded
        >
          {#snippet actions()}
            <Button variant="surface" onclick={restoreHeldEmbed}>
              Resume video
            </Button>
          {/snippet}
        </EmptyState>
      </div>
    {:else if resolveError && link}
      <EmptyState
        title="Cannot play video"
        message={resolveError}
        icon="alert"
        embedded
      >
        {#snippet actions()}
          <Button
            variant="surface"
            onclick={() => link && settings && void loadEmbed(link, settings)}
          >
            Retry
          </Button>
          <Button variant="ghost" onclick={() => void clearLink()}>
            Find another
          </Button>
        {/snippet}
      </EmptyState>
    {:else}
      <div class="video-watch__search">
        <Field label="Search music videos">
          <div class="video-watch__search-row">
            <input
              class="video-watch__input"
              bind:value={searchQuery}
              onkeydown={(event) => {
                if (event.key === "Enter") void runSearch();
              }}
            />
            <Button
              onclick={() => void runSearch()}
              disabled={searching || !searchQuery.trim() || !canSearch}
            >
              {searching ? "Searching…" : "Search"}
            </Button>
          </div>
        </Field>

        {#if !canSearch}
          <EmptyState
            title="Search not configured"
            message="Set an Invidious URL or YouTube API key in Settings → Video."
            icon="video"
            embedded
          />
        {:else if searching}
          <div class="video-watch--loading">
            <Spinner />
            <p class="video-watch__loading-label">Searching…</p>
          </div>
        {:else if searchError}
          <EmptyState
            title="Search failed"
            message={searchError}
            icon="alert"
            embedded
          />
        {:else if searched && results.length === 0}
          <EmptyState
            title="No results"
            message="Try a shorter query or a different provider in Settings → Video."
            icon="video"
            embedded
          />
        {:else if results.length > 0}
          <ul class="video-watch__results">
            {#each results as hit (hit.id)}
              <li>
                <button
                  type="button"
                  class="video-watch__result"
                  onclick={() => void pickResult(hit)}
                >
                  {#if hit.thumbnail}
                    <img src={hit.thumbnail} alt="" loading="lazy" />
                  {/if}
                  <span>
                    <strong>{displayVideoTitle(hit.title, hit.id)}</strong>
                    {#if hit.author}
                      <span class="video-watch__author">{hit.author}</span>
                    {/if}
                  </span>
                </button>
              </li>
            {/each}
          </ul>
        {:else}
          <EmptyState
            title="No video linked"
            message="Search runs automatically for this track when video search is ready."
            icon="video"
            embedded
          />
        {/if}
      </div>
    {/if}
  </div>
{/if}

<style>
  .video-watch {
    display: flex;
    flex-direction: column;
    gap: var(--jb-space-3);
    min-height: 0;
    flex: 1 1 auto;
  }

  .video-watch--embedded {
    flex: 1 1 auto;
  }

  .video-watch--loading {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: var(--jb-space-2);
    padding: var(--jb-space-6) var(--jb-space-2);
    flex: 1;
  }

  .video-watch__loading-label {
    margin: 0;
    color: var(--jb-text-muted);
    font-size: 0.875rem;
  }

  .video-watch__player {
    display: flex;
    flex-direction: column;
    gap: var(--jb-space-2);
    min-height: 0;
    flex: 1;
  }

  .video-watch__linked-title {
    margin: 0;
    font-size: 1rem;
    font-weight: 650;
    line-height: 1.3;
    overflow-wrap: anywhere;
  }

  .video-watch__frame {
    position: relative;
    width: 100%;
    aspect-ratio: 16 / 9;
    background: #000;
    border-radius: var(--jb-radius-md);
    overflow: hidden;
    flex: 0 0 auto;
  }

  .video-watch__frame iframe {
    position: absolute;
    inset: 0;
    width: 100%;
    height: 100%;
    border: 0;
  }

  .video-watch__row {
    display: flex;
    flex-wrap: wrap;
    gap: var(--jb-space-2);
    align-items: center;
  }

  .video-watch__search {
    display: flex;
    flex-direction: column;
    gap: var(--jb-space-2);
    min-height: 0;
    flex: 1;
  }

  .video-watch__search-row {
    display: flex;
    gap: var(--jb-space-2);
  }

  .video-watch__input {
    flex: 1;
    min-width: 0;
    padding: 0.5rem 0.75rem;
    border-radius: var(--jb-radius-sm);
    border: 1px solid var(--jb-border);
    background: var(--jb-surface);
    color: inherit;
  }

  .video-watch__results {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: var(--jb-space-1);
    max-height: min(50vh, 420px);
    overflow: auto;
  }

  .video-watch__result {
    display: flex;
    gap: var(--jb-space-3);
    align-items: center;
    width: 100%;
    text-align: left;
    padding: var(--jb-space-2);
    border: 0;
    border-radius: var(--jb-radius-sm);
    background: transparent;
    color: inherit;
    cursor: pointer;
  }

  .video-watch__result:hover {
    background: color-mix(in srgb, var(--jb-accent) 12%, transparent);
  }

  .video-watch__result img {
    width: 96px;
    height: 54px;
    object-fit: cover;
    border-radius: 4px;
    flex-shrink: 0;
  }

  .video-watch__result span {
    display: flex;
    flex-direction: column;
    gap: 0.15rem;
    min-width: 0;
  }

  .video-watch__author {
    margin: 0;
    color: var(--jb-text-muted);
    font-size: 0.85rem;
  }
</style>
