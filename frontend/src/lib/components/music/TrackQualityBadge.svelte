<script lang="ts">
  import type { SubsonicSong } from "$lib/subsonic/types";
  import { formatTrackQuality, isLosslessQuality } from "$lib/subsonic/quality";
  import { formatImmersiveBadge } from "$lib/music/immersive-audio";

  interface Props {
    track: Partial<
      Pick<
        SubsonicSong,
        | "suffix"
        | "bitRate"
        | "contentType"
        | "transcoded"
        | "channels"
        | "channelCount"
        | "title"
        | "album"
        | "path"
      >
    >;
    size?: "sm" | "md";
  }

  let { track, size = "sm" }: Props = $props();

  const label = $derived(formatTrackQuality(track));
  const immersive = $derived(formatImmersiveBadge(track));
  const lossless = $derived(isLosslessQuality(label));
</script>

{#if label || immersive}
  <span class="quality-group">
    {#if immersive}
      <span
        class="quality quality--{size} quality--immersive"
        title="Immersive or multichannel audio">{immersive}</span
      >
    {/if}
    {#if label}
      <span class="quality quality--{size}" class:quality--lossless={lossless}
        >{label}</span
      >
    {/if}
  </span>
{/if}

<style>
  .quality-group {
    display: inline-flex;
    align-items: center;
    gap: 0.25rem;
  }

  .quality {
    display: inline-flex;
    align-items: center;
    padding: 0.125rem 0.4375rem;
    border-radius: var(--jb-radius-sm);
    font-size: 0.625rem;
    font-weight: 700;
    letter-spacing: 0.04em;
    text-transform: uppercase;
    background: var(--jb-bg-muted);
    color: var(--jb-text-muted);
    border: 1px solid var(--jb-border);
    white-space: nowrap;
  }

  .quality--md {
    font-size: 0.6875rem;
    padding: 0.1875rem 0.5rem;
  }

  .quality--lossless {
    background: color-mix(in srgb, var(--jb-accent-muted) 70%, transparent);
    color: var(--jb-accent);
    border-color: color-mix(in srgb, var(--jb-accent) 35%, var(--jb-border));
  }

  .quality--immersive {
    background: color-mix(in srgb, var(--jb-accent) 18%, transparent);
    color: var(--jb-accent);
    border-color: color-mix(in srgb, var(--jb-accent) 40%, var(--jb-border));
  }
</style>
