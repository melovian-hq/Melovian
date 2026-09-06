<script lang="ts">
  import { generatePixelSmile } from "$lib/utils/pixel-smile";
  import { APP_SLUG } from "$lib/brand";

  interface Props {
    seed?: string;
    size?: number;
    class?: string;
    variant?: number;
    interactive?: boolean;
    label?: string;
    onvariantchange?: (variant: number) => void;
  }

  let {
    seed = APP_SLUG,
    size = 32,
    class: className = "",
    variant: controlledVariant,
    interactive = true,
    label,
    onvariantchange,
  }: Props = $props();

  let internalVariant = $state(0);
  let lastTapAt = 0;
  let lastRegenAt = 0;

  const variant = $derived(controlledVariant ?? internalVariant);
  const effectiveSeed = $derived(variant === 0 ? seed : `${seed}#${variant}`);
  const smile = $derived(generatePixelSmile(effectiveSeed));
  const ariaLabel = $derived(label ?? "Pixel face");

  function regenerate() {
    const now = Date.now();
    if (now - lastRegenAt < 400) return;
    lastRegenAt = now;
    const next = variant + 1;
    if (onvariantchange) {
      onvariantchange(next);
      return;
    }
    internalVariant = next;
  }

  function handleDblClick(event: MouseEvent) {
    event.stopPropagation();
    regenerate();
  }

  function handleTouchEnd(event: TouchEvent) {
    const now = Date.now();
    if (now - lastTapAt < 350) {
      regenerate();
      lastTapAt = 0;
      event.preventDefault();
      return;
    }
    lastTapAt = now;
  }
</script>

{#if interactive}
  <button
    type="button"
    class={["pixel-smile", className]}
    style:width="{size}px"
    style:height="{size}px"
    style:--grid-size={smile.gridSize}
    title="Double-click for a new face"
    aria-label="{ariaLabel}. Double-click or double-tap for a new face."
    ondblclick={handleDblClick}
    ontouchend={handleTouchEnd}
  >
    {#each smile.cells as color, index (index)}
      <span class="pixel-smile__cell" style:background={color}></span>
    {/each}
  </button>
{:else}
  <span
    role="img"
    aria-label={ariaLabel}
    class={["pixel-smile", "pixel-smile--static", className]}
    style:width="{size}px"
    style:height="{size}px"
    style:--grid-size={smile.gridSize}
  >
    {#each smile.cells as color, index (index)}
      <span class="pixel-smile__cell" style:background={color}></span>
    {/each}
  </span>
{/if}

<style>
  .pixel-smile {
    display: grid;
    grid-template-columns: repeat(var(--grid-size), 1fr);
    border-radius: var(--jb-radius-md);
    overflow: hidden;
    box-shadow: var(--jb-shadow-sm);
    flex-shrink: 0;
    image-rendering: pixelated;
    padding: 0;
    border: none;
    cursor: pointer;
    background: transparent;
    touch-action: manipulation;
  }

  .pixel-smile--static {
    cursor: default;
    touch-action: auto;
  }

  .pixel-smile__cell {
    aspect-ratio: 1;
    pointer-events: none;
  }
</style>
