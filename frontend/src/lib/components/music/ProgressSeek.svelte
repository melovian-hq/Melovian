<script lang="ts">
  import { untrack } from "svelte";
  import { prefersReducedMotion } from "svelte/motion";
  import type { TrackDecoration } from "$lib/extensions/types";
  import { preloadDecorationImages } from "$lib/extensions/assets";

  type Particle = {
    id: number;
    x: number;
    drift: number;
    hue: number;
    scale: number;
    duration: number;
  };

  interface Props {
    currentTime: number;
    duration: number;
    progressPercent: number;
    playing?: boolean;
    decoration?: TrackDecoration;
    disabled?: boolean;
    ariaLabel?: string;
    class?: string;
    size?: "sm" | "md" | "lg";
    onSeek: (seconds: number) => void;
  }

  let {
    currentTime,
    duration,
    progressPercent,
    playing = false,
    decoration = {},
    disabled = false,
    ariaLabel = "Seek",
    class: className = "",
    size = "md",
    onSeek,
  }: Props = $props();

  let particles = $state<Particle[]>([]);
  let nextId = 0;
  let scrubbing = $state(false);
  let previewSeconds = $state(0);

  const displaySeconds = $derived(scrubbing ? previewSeconds : currentTime);
  const displayProgress = $derived(
    scrubbing
      ? duration > 0
        ? (previewSeconds / duration) * 100
        : 0
      : progressPercent,
  );
  const clamped = $derived(
    Math.min(
      1,
      Math.max(0, Number.isFinite(displayProgress) ? displayProgress / 100 : 0),
    ),
  );
  const hasThumb = $derived(Boolean(decoration.progressThumbUrl));
  const hasParticles = $derived(Boolean(decoration.progressParticleUrl));
  const reducedMotion = $derived(prefersReducedMotion.current);
  const particleUrl = $derived(decoration.progressParticleUrl?.trim() || "");
  const thumbUrl = $derived(decoration.progressThumbUrl?.trim() || "");

  $effect(() => {
    preloadDecorationImages([thumbUrl, particleUrl]);
  });

  function readSeconds(event: Event): number {
    return Number((event.currentTarget as HTMLInputElement).value);
  }

  function handleInput(event: Event) {
    if (disabled) return;
    scrubbing = true;
    previewSeconds = readSeconds(event);
  }

  function commitSeek() {
    if (!scrubbing) return;
    const seconds = previewSeconds;
    scrubbing = false;
    onSeek(seconds);
  }

  function handleChange(event: Event) {
    if (disabled) return;
    previewSeconds = readSeconds(event);
    scrubbing = true;
    commitSeek();
  }

  function handlePointerEnd(event: Event) {
    if (disabled || !scrubbing) return;
    previewSeconds = readSeconds(event);
    commitSeek();
  }

  $effect(() => {
    if (!playing || !hasParticles || reducedMotion || disabled) {
      return;
    }

    let cancelled = false;
    const timeouts: number[] = [];

    function spawnParticle() {
      const id = nextId++;
      const ratio = untrack(() => clamped);
      const particle: Particle = {
        id,
        x: ratio * 100,
        drift: (Math.random() - 0.5) * 48,
        hue: Math.floor(Math.random() * 360),
        scale: 0.7 + Math.random() * 0.7,
        duration: 900 + Math.random() * 500,
      };
      untrack(() => {
        particles = [...particles.slice(-8), particle];
      });
      timeouts.push(
        window.setTimeout(() => {
          if (cancelled) return;
          untrack(() => {
            particles = particles.filter((p) => p.id !== id);
          });
        }, particle.duration),
      );
    }

    spawnParticle();
    const timer = window.setInterval(spawnParticle, 320);
    return () => {
      cancelled = true;
      window.clearInterval(timer);
      for (const timeout of timeouts) window.clearTimeout(timeout);
      untrack(() => {
        particles = [];
      });
    };
  });
</script>

<div
  class="progress-seek progress-seek--{size} {className}"
  class:progress-seek--thumb={hasThumb}
  style:--progress={clamped}
>
  <input
    type="range"
    class="progress-seek__input"
    min="0"
    max={duration || 100}
    step="0.1"
    value={displaySeconds}
    oninput={handleInput}
    onchange={handleChange}
    onpointerup={handlePointerEnd}
    ontouchend={handlePointerEnd}
    onpointercancel={handlePointerEnd}
    aria-label={ariaLabel}
    {disabled}
  />
  <div class="progress-seek__track" aria-hidden="true">
    <div
      class="progress-seek__fill"
      style:transform="scaleX({clamped})"
      style:background={decoration.progressGradient ?? decoration.progressColor}
    ></div>
  </div>
  {#if hasThumb && thumbUrl}
    <img
      class="progress-seek__thumb"
      src={thumbUrl}
      alt=""
      draggable="false"
      decoding="async"
      loading="eager"
      aria-hidden="true"
    />
  {/if}
  {#if hasParticles && particleUrl}
    <div
      class="progress-seek__particles"
      style:--particle-url="url({particleUrl})"
      aria-hidden="true"
    >
      {#each particles as particle (particle.id)}
        <span
          class="progress-seek__particle"
          style:--x="{particle.x}%"
          style:--drift="{particle.drift}px"
          style:--hue={particle.hue}
          style:--scale={particle.scale}
          style:--dur="{particle.duration}ms"
        ></span>
      {/each}
    </div>
  {/if}
</div>

<style>
  .progress-seek {
    position: relative;
    height: 5px;
    background: var(--jb-bg-muted);
    overflow: visible;
  }

  .progress-seek--sm {
    height: 3px;
  }

  .progress-seek--lg {
    height: 8px;
  }

  .progress-seek--thumb {
    margin-top: 0;
  }

  .progress-seek__input {
    position: absolute;
    inset: 0;
    width: 100%;
    height: 100%;
    margin: 0;
    opacity: 0;
    cursor: pointer;
    z-index: 4;
  }

  .progress-seek__input:disabled {
    cursor: default;
  }

  .progress-seek__track {
    position: absolute;
    inset: 0;
    overflow: hidden;
    border-radius: inherit;
  }

  .progress-seek__fill {
    height: 100%;
    width: 100%;
    transform-origin: left center;
    background: var(
      --progress-fill,
      linear-gradient(
        90deg,
        var(--jb-accent),
        color-mix(in srgb, var(--jb-accent) 65%, #ffffff)
      )
    );
    will-change: transform;
  }

  .progress-seek__thumb {
    position: absolute;
    top: 50%;
    left: calc(var(--progress) * 100%);
    width: 1.75rem;
    height: 1.75rem;
    margin: 0;
    padding: 0;
    border: none;
    border-radius: 2px;
    aspect-ratio: 1;
    object-fit: contain;
    transform: translate(-50%, -50%);
    image-rendering: pixelated;
    pointer-events: none;
    z-index: 3;
    filter: drop-shadow(0 1px 2px rgb(0 0 0 / 0.45));
  }

  .progress-seek--sm .progress-seek__thumb {
    width: 1.25rem;
    height: 1.25rem;
  }

  .progress-seek--lg .progress-seek__thumb {
    width: 2.1rem;
    height: 2.1rem;
  }

  .progress-seek__particles {
    position: absolute;
    inset: 0;
    overflow: visible;
    pointer-events: none;
    z-index: 2;
  }

  .progress-seek__particle {
    position: absolute;
    left: var(--x);
    top: 50%;
    width: 0.85rem;
    height: 0.85rem;
    margin: 0;
    display: block;
    background-image: var(--particle-url);
    background-repeat: no-repeat;
    background-position: center;
    background-size: contain;
    image-rendering: pixelated;
    transform: translate(-50%, -50%) scale(var(--scale));
    filter: brightness(0) saturate(100%) invert(1) sepia(1) saturate(5000%)
      hue-rotate(calc(var(--hue) * 1deg));
    animation: note-spew var(--dur) ease-out forwards;
  }

  .progress-seek--lg .progress-seek__particle {
    width: 1.1rem;
    height: 1.1rem;
  }

  @keyframes note-spew {
    0% {
      opacity: 0.95;
      transform: translate(-50%, -40%) scale(calc(var(--scale) * 0.7))
        translateX(0);
    }
    70% {
      opacity: 0.85;
    }
    100% {
      opacity: 0;
      transform: translate(-50%, -160%) scale(var(--scale))
        translateX(var(--drift));
    }
  }

  @media (prefers-reduced-motion: reduce) {
    .progress-seek__particle {
      animation: none;
      display: none;
    }
  }
</style>
