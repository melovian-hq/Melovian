<script lang="ts">
  import Button from "$lib/components/ui/Button.svelte";
  import EmptyState from "$lib/components/ui/EmptyState.svelte";
  import Input from "$lib/components/ui/Input.svelte";
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";
  import Spinner from "$lib/components/ui/Spinner.svelte";
  import { music } from "$lib/config/music.svelte";
  import * as musicApi from "$lib/music/api";
  import type { MusicShare, ShareTrack } from "$lib/music/api";
  import { trackFromOpenStream } from "$lib/music/open-uri";
  import Link from "$lib/router/Link.svelte";
  import { formatDurationMs } from "$lib/subsonic";
  import type { QueueTrack } from "$lib/subsonic/types";
  import { downloadFromUrl, sanitizeFilename } from "$lib/utils/download";
  import { toast } from "$lib/ui/toast.svelte";
  import { createAsyncPage } from "$lib/ui/async-page.svelte";
  import { APP_NAME } from "$lib/brand";
  import { setPageMeta } from "$lib/seo/meta";

  interface Props {
    token: string;
  }

  let { token }: Props = $props();

  let share = $state<MusicShare | null>(null);
  let password = $state("");
  let unlocking = $state(false);
  let downloadBusyId = $state<string | null>(null);

  function shareTrackToQueue(track: ShareTrack): QueueTrack {
    const stream = musicApi.publicShareStreamUrl(token, track.id);
    return {
      ...trackFromOpenStream(stream, track.title || "Track"),
      artist: track.artist,
      album: track.album,
      albumId: track.albumId,
      coverArt: track.coverArtId,
      duration: Math.floor(track.durationMs / 1000),
    };
  }

  const page = createAsyncPage<MusicShare>({
    errorMessage: "Share unavailable",
    onError: () => {
      share = null;
    },
    load: () => {
      const current = token;
      share = null;
      return musicApi.getPublicShare(current);
    },
    apply: (result) => {
      share = result;
    },
  });

  async function unlock(event: Event) {
    event.preventDefault();
    if (!password.trim()) {
      toast.error("Enter the share password");
      return;
    }
    unlocking = true;
    try {
      await musicApi.unlockPublicShare(token, password);
      password = "";
      await page.reload();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Invalid password");
    } finally {
      unlocking = false;
    }
  }

  function playAll() {
    const tracks = share?.tracks ?? [];
    if (tracks.length === 0) return;
    music.playTracks(tracks.map(shareTrackToQueue), 0);
  }

  function playTrack(index: number) {
    const tracks = share?.tracks ?? [];
    if (tracks.length === 0) return;
    music.playTracks(tracks.map(shareTrackToQueue), index);
  }

  async function saveTrack(track: ShareTrack) {
    downloadBusyId = track.id;
    try {
      const filename = sanitizeFilename(
        `${track.artist} - ${track.title}`,
        "track",
      );
      await downloadFromUrl(
        musicApi.publicShareDownloadUrl(token, track.id),
        filename,
        { credentials: "include" },
      );
      toast.success("Download started");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Download failed");
    } finally {
      downloadBusyId = null;
    }
  }

  const title = $derived(
    share?.title || share?.description || "Shared playlist",
  );
  const tracks = $derived(share?.tracks ?? []);
  const unlocked = $derived(
    Boolean(share) && !share?.requiresPassword && !share?.requiresLogin,
  );

  $effect(() => {
    if (!share) return;
    const shareTitle = title.trim() || "Shared playlist";
    const count = tracks.length;
    setPageMeta({
      title: shareTitle,
      description:
        count > 0
          ? `${shareTitle} · ${count} track${count === 1 ? "" : "s"} shared via ${APP_NAME}.`
          : `A playlist shared via ${APP_NAME}.`,
    });
  });
</script>

<div class="share-page">
  {#if page.loading}
    <div class="share-page__state"><Spinner /></div>
  {:else if page.error}
    <EmptyState title="Share unavailable" message={page.error} icon="share">
      {#snippet actions()}
        <Link href="/music">Back to music</Link>
      {/snippet}
    </EmptyState>
  {:else if share?.requiresPassword}
    <section class="share-page__gate">
      <h1>Password required</h1>
      <p>This shared playlist is protected by a password.</p>
      <form class="share-page__password" onsubmit={unlock}>
        <Input
          type="password"
          bind:value={password}
          placeholder="Password"
          autocomplete="current-password"
        />
        <Button type="submit" disabled={unlocking}>
          {unlocking ? "Unlocking…" : "Unlock"}
        </Button>
      </form>
    </section>
  {:else if share?.requiresLogin}
    <EmptyState
      title="Sign in required"
      message="This shared playlist is only available to signed-in accounts."
      icon="lock"
    >
      {#snippet actions()}
        <Link href="/account/login">Sign in</Link>
      {/snippet}
    </EmptyState>
  {:else if unlocked}
    <header class="share-page__header">
      <p class="share-page__eyebrow">Shared playlist</p>
      <h1>{title}</h1>
      <p class="share-page__meta">
        {tracks.length.toLocaleString()}
        {tracks.length === 1 ? "track" : "tracks"}
      </p>
      <div class="share-page__actions">
        <Button type="button" disabled={tracks.length === 0} onclick={playAll}>
          <MdiIcon name="play" size={18} />
          Play all
        </Button>
      </div>
    </header>

    {#if tracks.length === 0}
      <EmptyState
        title="No tracks"
        message="This share has no playable tracks."
        icon="listMusic"
      />
    {:else}
      <ul class="share-page__tracks">
        {#each tracks as track, index (track.id)}
          <li class="share-page__track">
            <button
              type="button"
              class="share-page__track-main"
              onclick={() => playTrack(index)}
            >
              <span class="share-page__track-index">{index + 1}</span>
              <span class="share-page__track-info">
                <span class="share-page__track-title">{track.title}</span>
                <span class="share-page__track-artist"
                  >{track.artist || "Unknown artist"}</span
                >
              </span>
              <span class="share-page__track-duration"
                >{formatDurationMs(track.durationMs)}</span
              >
            </button>
            <Button
              type="button"
              variant="ghost"
              size="sm"
              disabled={downloadBusyId === track.id}
              aria-label="Save {track.title} to device"
              onclick={() => void saveTrack(track)}
            >
              <MdiIcon name="download" size={16} />
              Save
            </Button>
          </li>
        {/each}
      </ul>
    {/if}
  {/if}
</div>

<style>
  .share-page {
    display: flex;
    flex-direction: column;
    gap: var(--jb-space-5);
    padding: var(--jb-space-4) 0 var(--jb-space-8);
    max-width: 48rem;
  }

  .share-page__state {
    display: grid;
    place-items: center;
    min-height: 12rem;
  }

  .share-page__gate {
    display: flex;
    flex-direction: column;
    gap: var(--jb-space-3);
    max-width: 24rem;
  }

  .share-page__gate h1 {
    margin: 0;
    font-size: 1.5rem;
  }

  .share-page__gate p {
    margin: 0;
    color: var(--jb-text-muted);
  }

  .share-page__password {
    display: flex;
    gap: var(--jb-space-2);
    align-items: center;
  }

  .share-page__header h1 {
    margin: var(--jb-space-2) 0;
    font-size: 1.75rem;
  }

  .share-page__eyebrow {
    margin: 0;
    font-size: 0.8125rem;
    font-weight: 600;
    color: var(--jb-text-muted);
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }

  .share-page__meta {
    margin: 0 0 var(--jb-space-3);
    color: var(--jb-text-muted);
  }

  .share-page__actions {
    display: flex;
    gap: var(--jb-space-2);
  }

  .share-page__tracks {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: var(--jb-space-1);
  }

  .share-page__track {
    display: flex;
    align-items: center;
    gap: var(--jb-space-2);
    border-radius: var(--jb-radius-md);
    border: 1px solid transparent;
  }

  .share-page__track:hover {
    background: var(--jb-surface-hover);
  }

  .share-page__track-main {
    flex: 1;
    min-width: 0;
    display: grid;
    grid-template-columns: 2rem 1fr auto;
    align-items: center;
    gap: var(--jb-space-3);
    padding: 0.625rem 0.75rem;
    border: none;
    background: transparent;
    color: inherit;
    text-align: left;
    cursor: pointer;
  }

  .share-page__track-index {
    color: var(--jb-text-muted);
    font-variant-numeric: tabular-nums;
    font-size: 0.8125rem;
  }

  .share-page__track-info {
    display: flex;
    flex-direction: column;
    gap: 0.125rem;
    min-width: 0;
  }

  .share-page__track-title {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-weight: 600;
  }

  .share-page__track-artist {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    color: var(--jb-text-muted);
    font-size: 0.8125rem;
  }

  .share-page__track-duration {
    color: var(--jb-text-muted);
    font-variant-numeric: tabular-nums;
    font-size: 0.8125rem;
  }
</style>
