<script lang="ts">
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";
  import { music } from "$lib/config/music.svelte";
  import EqPanel from "./EqPanel.svelte";

  let panelEl = $state<HTMLDivElement | null>(null);
  let pos = $state<{ x: number; y: number } | null>(null);
  let dragging = $state(false);
  let dragOrigin = { pointerX: 0, pointerY: 0, startX: 0, startY: 0 };

  const open = $derived(music.eqOpen && music.eqAvailable);

  function close() {
    music.eqOpen = false;
  }

  function onDragStart(event: PointerEvent) {
    if (window.matchMedia("(max-width: 768px)").matches) return;
    if (event.button !== 0) return;
    const target = event.target as HTMLElement | null;
    if (target?.closest(".jb-no-drag")) return;
    const rect = panelEl?.getBoundingClientRect();
    if (!rect) return;
    dragging = true;
    pos = { x: rect.left, y: rect.top };
    dragOrigin = {
      pointerX: event.clientX,
      pointerY: event.clientY,
      startX: rect.left,
      startY: rect.top,
    };
    (event.currentTarget as HTMLElement).setPointerCapture(event.pointerId);
    event.preventDefault();
  }

  function onDragMove(event: PointerEvent) {
    if (!dragging || !pos) return;
    const width = panelEl?.offsetWidth ?? 360;
    const height = panelEl?.offsetHeight ?? 280;
    const x = dragOrigin.startX + (event.clientX - dragOrigin.pointerX);
    const y = dragOrigin.startY + (event.clientY - dragOrigin.pointerY);
    pos = {
      x: Math.min(Math.max(8, x), window.innerWidth - width - 8),
      y: Math.min(Math.max(8, y), window.innerHeight - height - 8),
    };
  }

  function onDragEnd(event: PointerEvent) {
    if (!dragging) return;
    dragging = false;
    try {
      (event.currentTarget as HTMLElement).releasePointerCapture(
        event.pointerId,
      );
    } catch {
      // Capture may already be released.
    }
  }

  function onKeydown(event: KeyboardEvent) {
    if (event.key === "Escape") close();
  }
</script>

<svelte:window onkeydown={onKeydown} />

{#if open}
  <div
    bind:this={panelEl}
    class="eq-float"
    class:eq-float--dragging={dragging}
    class:eq-float--placed={pos !== null}
    style:left={pos ? `${pos.x}px` : undefined}
    style:top={pos ? `${pos.y}px` : undefined}
    role="dialog"
    aria-label="Equalizer"
  >
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <header
      class="eq-float__head"
      onpointerdown={onDragStart}
      onpointermove={onDragMove}
      onpointerup={onDragEnd}
      onpointercancel={onDragEnd}
    >
      <span class="eq-float__grip" aria-hidden="true">
        <MdiIcon name="gripVertical" size={16} />
      </span>
      <h2>Equalizer</h2>
      <button
        type="button"
        class="eq-float__close jb-no-drag"
        aria-label="Close equalizer"
        onclick={close}
      >
        <MdiIcon name="x" size={16} />
      </button>
    </header>
    <div class="eq-float__body">
      <EqPanel />
    </div>
  </div>
{/if}

<style>
  .eq-float {
    position: fixed;
    z-index: 70;
    right: var(--jb-space-4);
    bottom: calc(
      var(--jb-bottom-chrome-height, 6.5rem) +
        env(safe-area-inset-bottom, 0px) + var(--jb-space-2)
    );
    width: min(26rem, calc(100vw - 1.5rem));
    max-height: min(32rem, calc(100vh - 8rem));
    display: flex;
    flex-direction: column;
    border: 1px solid var(--jb-border);
    border-radius: var(--jb-radius-xl);
    background: color-mix(in srgb, var(--jb-bg-elevated) 96%, transparent);
    backdrop-filter: blur(20px);
    box-shadow: var(--jb-shadow-lg);
    overflow: hidden;
  }

  .eq-float--placed {
    right: auto;
    bottom: auto;
  }

  .eq-float--dragging {
    user-select: none;
  }

  .eq-float__head {
    display: flex;
    align-items: center;
    gap: var(--jb-space-2);
    padding: 0.55rem var(--jb-space-3);
    border-bottom: 1px solid var(--jb-border);
    cursor: grab;
    flex-shrink: 0;
  }

  .eq-float--dragging .eq-float__head {
    cursor: grabbing;
  }

  .eq-float__head h2 {
    margin: 0;
    flex: 1;
    font-size: 0.8125rem;
    font-weight: 700;
    letter-spacing: 0.04em;
    text-transform: uppercase;
    color: var(--jb-text-muted);
  }

  .eq-float__grip {
    display: grid;
    place-content: center;
    color: var(--jb-text-subtle);
  }

  .eq-float__close {
    display: grid;
    place-content: center;
    width: 1.75rem;
    height: 1.75rem;
    border: none;
    border-radius: var(--jb-radius-sm);
    background: transparent;
    color: var(--jb-text-muted);
    cursor: pointer;
  }

  .eq-float__close:hover {
    color: var(--jb-text);
    background: var(--jb-accent-muted);
  }

  .eq-float__body {
    min-height: 0;
    overflow: auto;
  }

  .eq-float__body :global(.eq-panel) {
    padding: var(--jb-space-3);
    border-top: none;
    background: transparent;
  }

  .eq-float__body :global(.eq-panel__header) {
    display: none;
  }

  .eq-float__body :global(.eq-panel__faders) {
    display: none;
  }

  .eq-float__body :global(.eq-panel__params) {
    grid-template-columns: 1fr;
    margin-bottom: 0;
  }

  @media (max-width: 768px) {
    .eq-float,
    .eq-float--placed {
      left: 0;
      right: 0;
      top: auto;
      bottom: 0;
      width: 100%;
      max-height: min(70vh, 34rem);
      border-radius: var(--jb-radius-xl) var(--jb-radius-xl) 0 0;
      padding-bottom: env(safe-area-inset-bottom, 0px);
    }

    .eq-float__head {
      cursor: default;
      justify-content: center;
    }

    .eq-float__grip {
      display: none;
    }
  }

  @media (prefers-reduced-motion: reduce) {
    .eq-float {
      backdrop-filter: none;
    }
  }
</style>
