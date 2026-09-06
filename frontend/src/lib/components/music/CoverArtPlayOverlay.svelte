<script lang="ts">
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";

  interface Props {
    playing?: boolean;
    size?: number;
  }

  let { playing = false, size = 18 }: Props = $props();
</script>

<span class="cover-play-overlay" aria-hidden="true">
  <span class="cover-play-overlay__backdrop"></span>
  {#if playing}
    <MdiIcon name="pause" {size} />
  {:else}
    <MdiIcon name="play" {size} />
  {/if}
</span>

<style>
  .cover-play-overlay {
    position: absolute;
    inset: 0;
    display: grid;
    place-content: center;
    color: #fff;
    opacity: 0;
    transition: opacity var(--jb-transition);
    pointer-events: none;
  }

  .cover-play-overlay__backdrop {
    position: absolute;
    inset: 0;
    background: color-mix(in srgb, #000 28%, transparent);
    backdrop-filter: blur(1px);
  }

  .cover-play-overlay :global(svg) {
    position: relative;
    z-index: 1;
  }
</style>
