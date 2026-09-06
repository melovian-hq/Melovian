<script lang="ts">
  import CoverArt from "$lib/components/ui/CoverArt.svelte";

  interface Props {
    src: string | null;
    seed?: string;
    paletteKey?: string;
    opacity?: number;
    blur?: number;
    saturate?: number;
    scale?: number;
    class?: string;
  }

  let {
    src,
    seed = "",
    paletteKey = seed,
    opacity = 0.35,
    blur = 40,
    saturate = 1.3,
    scale = 1.35,
    class: className = "",
  }: Props = $props();
</script>

{#if src}
  <div
    class="ambient-cover-backdrop {className}"
    style:opacity
    style:--ambient-blur="{blur}px"
    style:--ambient-saturate={saturate}
    style:--ambient-scale={scale}
    aria-hidden="true"
  >
    <CoverArt {src} {seed} {paletteKey} alt="" />
  </div>
{/if}

<style>
  .ambient-cover-backdrop {
    position: absolute;
    inset: -20%;
    z-index: 0;
    pointer-events: none;
    transform: scale(var(--ambient-scale, 1.35));
    filter: blur(var(--ambient-blur, 40px))
      saturate(var(--ambient-saturate, 1.3));
  }

  .ambient-cover-backdrop :global(.cover-art) {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }
</style>
