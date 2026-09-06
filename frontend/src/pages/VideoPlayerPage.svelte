<script lang="ts">
  import AppShell from "$lib/components/layout/AppShell.svelte";
  import EmptyState from "$lib/components/ui/EmptyState.svelte";
  import Spinner from "$lib/components/ui/Spinner.svelte";
  import Button from "$lib/components/ui/Button.svelte";
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";
  import Link from "$lib/router/Link.svelte";
  import { parseQuery } from "$lib/router/match";
  import { router } from "$lib/router/router.svelte";
  import {
    getLocalVideo,
    localVideoStreamUrl,
    resolveVideo,
  } from "$lib/video/api";
  import {
    displayVideoTitle,
    invidiousIdFromPlayId,
    isExternalPlayId,
    isInvidiousPlayId,
    isYouTubePlayId,
    youtubeIdFromPlayId,
    type LocalVideo,
  } from "$lib/video/ids";
  import {
    pauseMusicForVideo,
    registerVideoPause,
  } from "$lib/video/playback-gate.svelte";
  import { mediaTrackDownloadUrl } from "$lib/music/api";

  interface Props {
    itemId?: string;
  }

  let { itemId = "" }: Props = $props();

  let loading = $state(true);
  let error = $state<string | null>(null);
  let localVideo = $state<LocalVideo | null>(null);
  let embedUrl = $state<string | null>(null);
  let embedTitle = $state("");
  let embedAuthor = $state("");
  let videoEl = $state<HTMLVideoElement | null>(null);
  let playing = $state(false);
  let heldEmbedUrl = $state<string | null>(null);

  const playId = $derived(decodeURIComponent(itemId || ""));
  const isExternal = $derived(isExternalPlayId(playId));
  const queryTitle = $derived(parseQuery(router.search).title?.trim() ?? "");
  const pageTitle = $derived(
    localVideo
      ? localVideo.title
      : displayVideoTitle(embedTitle || queryTitle, externalVideoId(playId)),
  );
  const localDownloadUrl = $derived(
    localVideo
      ? mediaTrackDownloadUrl(localVideo.id, {
          title: localVideo.title,
          artist: localVideo.artist,
        })
      : "",
  );

  function externalVideoId(id: string): string | null {
    return invidiousIdFromPlayId(id) ?? youtubeIdFromPlayId(id);
  }

  $effect(() => {
    return registerVideoPause(() => {
      const el = videoEl;
      if (el) {
        el.pause();
        playing = false;
      }
      if (embedUrl) {
        heldEmbedUrl = embedUrl;
        embedUrl = null;
      }
    });
  });

  function onLocalPlay() {
    playing = true;
    pauseMusicForVideo();
  }

  function restoreHeldEmbed() {
    if (!heldEmbedUrl) return;
    embedUrl = heldEmbedUrl;
    heldEmbedUrl = null;
    pauseMusicForVideo();
  }

  function focusEmbed() {
    pauseMusicForVideo();
  }

  function downloadLocalVideo() {
    if (!localDownloadUrl) return;
    const anchor = document.createElement("a");
    anchor.href = localDownloadUrl;
    anchor.download = "";
    anchor.rel = "noopener";
    document.body.appendChild(anchor);
    anchor.click();
    anchor.remove();
  }

  $effect(() => {
    const id = playId;
    const hintTitle = queryTitle;
    if (!id) {
      loading = false;
      error = "Missing video id";
      return;
    }
    let cancelled = false;
    loading = true;
    error = null;
    localVideo = null;
    embedUrl = null;
    heldEmbedUrl = null;
    embedTitle = hintTitle;
    embedAuthor = "";

    async function load() {
      try {
        if (isInvidiousPlayId(id)) {
          const videoId = invidiousIdFromPlayId(id);
          if (!videoId) throw new Error("Invalid Invidious video id");
          const resolved = await resolveVideo({
            source: "invidious",
            id: videoId,
            title: hintTitle || undefined,
          });
          if (cancelled) return;
          if (!resolved.embedUrl) {
            throw new Error("No embed URL from Invidious settings");
          }
          embedUrl = resolved.embedUrl;
          embedTitle = resolved.title || hintTitle;
          embedAuthor = resolved.author ?? "";
        } else if (isYouTubePlayId(id)) {
          const videoId = youtubeIdFromPlayId(id);
          if (!videoId) throw new Error("Invalid YouTube video id");
          const resolved = await resolveVideo({
            source: "youtube",
            id: videoId,
            title: hintTitle || undefined,
          });
          if (cancelled) return;
          if (!resolved.embedUrl) {
            throw new Error("No YouTube embed URL");
          }
          embedUrl = resolved.embedUrl;
          embedTitle = resolved.title || hintTitle;
          embedAuthor = resolved.author ?? "";
        } else {
          const video = await getLocalVideo(id);
          if (cancelled) return;
          localVideo = video;
        }
      } catch (err) {
        if (!cancelled) {
          error = err instanceof Error ? err.message : String(err);
        }
      } finally {
        if (!cancelled) loading = false;
      }
    }

    void load();
    return () => {
      cancelled = true;
      releaseLocalVideo();
      embedUrl = null;
      heldEmbedUrl = null;
    };
  });

  function releaseLocalVideo() {
    const el = videoEl;
    if (!el) return;
    el.pause();
    el.removeAttribute("src");
    el.load();
    videoEl = null;
    playing = false;
  }

  function togglePlay() {
    const el = videoEl;
    if (!el) return;
    if (el.paused) {
      void el.play();
    } else {
      el.pause();
    }
  }

  function onKeydown(event: KeyboardEvent) {
    if (event.code !== "Space" && event.code !== "KeyK") return;
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
    if (isExternal || !videoEl) return;
    event.preventDefault();
    togglePlay();
  }

  function goFullscreen() {
    const el = videoEl;
    if (!el) return;
    if (document.fullscreenElement) {
      void document.exitFullscreen();
      return;
    }
    void el.requestFullscreen();
  }
</script>

<svelte:window onkeydown={onKeydown} />

<AppShell fill>
  <div class="video-player-page">
    <div class="video-player-page__bar">
      <Button variant="ghost" onclick={() => router.navigate("/music/videos")}>
        <MdiIcon name="chevronLeft" size={18} />
        Videos
      </Button>
      <Link href="/music/now-playing" class="video-player-page__now">
        Now playing
      </Link>
    </div>

    {#if loading}
      <div class="video-player-page__center">
        <Spinner />
        <p class="video-player-page__loading-label">Loading video…</p>
      </div>
    {:else if error}
      <EmptyState title="Cannot play video" message={error} icon="alert">
        {#snippet actions()}
          <Button
            variant="surface"
            onclick={() => router.navigate("/music/videos")}
          >
            Back to videos
          </Button>
        {/snippet}
      </EmptyState>
    {:else if embedUrl}
      <div class="video-player-page__stage">
        <h1 class="video-player-page__title">{pageTitle}</h1>
        {#if embedAuthor}
          <p class="video-player-page__artist">{embedAuthor}</p>
        {/if}
        <div
          class="video-player-page__frame"
          role="presentation"
          onclick={focusEmbed}
          onfocusin={focusEmbed}
        >
          <iframe
            title={pageTitle}
            src={embedUrl}
            allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; fullscreen"
          ></iframe>
        </div>
      </div>
    {:else if heldEmbedUrl}
      <div class="video-player-page__stage">
        <h1 class="video-player-page__title">{pageTitle}</h1>
        <EmptyState
          title="Video paused for music"
          message="Music started, so the embed was unloaded. Resume the video when you want it back."
          icon="video"
        >
          {#snippet actions()}
            <Button variant="surface" onclick={restoreHeldEmbed}>
              Resume video
            </Button>
          {/snippet}
        </EmptyState>
      </div>
    {:else if localVideo}
      <div class="video-player-page__stage">
        <h1 class="video-player-page__title">{localVideo.title}</h1>
        {#if localVideo.artist}
          <p class="video-player-page__artist">{localVideo.artist}</p>
        {/if}
        <!-- svelte-ignore a11y_media_has_caption -->
        <video
          bind:this={videoEl}
          class="video-player-page__video"
          src={localVideoStreamUrl(localVideo.id)}
          controls
          playsinline
          onplay={onLocalPlay}
          onpause={() => (playing = false)}
        ></video>
        <div class="video-player-page__controls">
          <Button variant="surface" onclick={togglePlay}>
            <MdiIcon name={playing ? "pause" : "play"} size={18} />
            {playing ? "Pause" : "Play"}
          </Button>
          <Button variant="ghost" onclick={goFullscreen}>
            <MdiIcon name="fullscreen" size={18} />
            Fullscreen
          </Button>
          <Button variant="ghost" onclick={downloadLocalVideo}>
            <MdiIcon name="download" size={18} />
            Download
          </Button>
        </div>
      </div>
    {:else}
      <EmptyState
        title="Nothing to play"
        message="Open a local video from Videos or link one from Now Playing."
        icon="video"
      >
        {#snippet actions()}
          <Button
            variant="surface"
            onclick={() => router.navigate("/music/videos")}
          >
            Browse videos
          </Button>
        {/snippet}
      </EmptyState>
    {/if}
  </div>
</AppShell>

<style>
  .video-player-page {
    display: flex;
    flex-direction: column;
    gap: var(--jb-space-4);
    height: 100%;
    min-height: 0;
    max-width: 1100px;
    margin: 0 auto;
    width: 100%;
    padding: var(
        --jb-fill-pad-top,
        calc(var(--jb-window-chrome-offset, 0px) + var(--jb-space-4))
      )
      var(--jb-space-6) var(--jb-space-8);
  }

  .video-player-page__bar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--jb-space-3);
  }

  .video-player-page__bar :global(.video-player-page__now) {
    color: var(--jb-text-muted);
    text-decoration: none;
    font-size: 0.9rem;
  }

  .video-player-page__center {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: var(--jb-space-3);
  }

  .video-player-page__loading-label {
    margin: 0;
    color: var(--jb-text-muted);
    font-size: 0.9rem;
  }

  .video-player-page__stage {
    display: flex;
    flex-direction: column;
    gap: var(--jb-space-3);
    min-height: 0;
    flex: 1;
  }

  .video-player-page__title {
    margin: 0;
    font-size: 1.35rem;
    font-weight: 650;
  }

  .video-player-page__artist {
    margin: 0;
    color: var(--jb-text-muted);
  }

  .video-player-page__video {
    width: 100%;
    max-height: min(70vh, 720px);
    background: #000;
    border-radius: var(--jb-radius-md);
  }

  .video-player-page__frame {
    position: relative;
    width: 100%;
    aspect-ratio: 16 / 9;
    background: #000;
    border-radius: var(--jb-radius-md);
    overflow: hidden;
  }

  .video-player-page__frame iframe {
    position: absolute;
    inset: 0;
    width: 100%;
    height: 100%;
    border: 0;
  }

  .video-player-page__controls {
    display: flex;
    gap: var(--jb-space-2);
    flex-wrap: wrap;
  }
</style>
