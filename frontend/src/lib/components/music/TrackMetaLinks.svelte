<script lang="ts">
  import Link from "$lib/router/Link.svelte";
  import { isLocalMusicSource } from "$lib/music/source.svelte";
  import {
    resolveTrackAlbum,
    resolveTrackArtists,
  } from "$lib/music/track-metadata";
  import type { SubsonicSong } from "$lib/subsonic";

  interface Props {
    track: SubsonicSong;
    class?: string;
  }

  let { track, class: className = "" }: Props = $props();

  const artists = $derived(
    resolveTrackArtists(track, { localIds: isLocalMusicSource() }),
  );
  const album = $derived(resolveTrackAlbum(track));
</script>

<span class="track-meta-links {className}">
  {#each artists as artist, index (artist.name + (artist.id ?? ""))}
    {#if index > 0}<span class="track-meta-links__sep">&amp;</span>{/if}
    {#if artist.id}
      <Link class="track-meta-links__link" href="/music/artist/{artist.id}"
        >{artist.name}</Link
      >
    {:else}
      <span class="track-meta-links__text">{artist.name}</span>
    {/if}
  {/each}
  {#if album}
    <span class="track-meta-links__dot">·</span>
    {#if album.id}
      <Link class="track-meta-links__link" href="/music/album/{album.id}"
        >{album.name}</Link
      >
    {:else}
      <span class="track-meta-links__text">{album.name}</span>
    {/if}
  {/if}
</span>

<style>
  .track-meta-links {
    display: inline;
    min-width: 0;
  }

  .track-meta-links__sep,
  .track-meta-links__dot,
  .track-meta-links__text {
    margin: 0 0.2em;
    color: var(--jb-text-muted);
  }

  :global(.track-meta-links__link) {
    color: var(--jb-text-muted);
    text-decoration: none;
  }

  :global(.track-meta-links__link:hover) {
    color: var(--jb-accent);
    text-decoration: underline;
  }
</style>
