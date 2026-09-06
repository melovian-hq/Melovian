<script lang="ts">
  import { prefersReducedMotion } from "svelte/motion";
  import { fade, scale } from "svelte/transition";
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";
  import {
    keybindFeedback,
    type KeybindFeedbackPulse,
  } from "$lib/ui/keybind-feedback.svelte";

  const item = $derived(keybindFeedback.current);
  const motion = $derived(prefersReducedMotion.current ? 0 : 1);

  function iconName(pulse: KeybindFeedbackPulse): string {
    switch (pulse.kind) {
      case "seek":
        return pulse.direction === "back" ? "rewind" : "fastForward";
      case "volume": {
        const pct = pulse.volumePct ?? 0;
        if (pct <= 0) return "volumeOff";
        if (pct < 34) return "volumeLow";
        if (pct < 67) return "volumeMedium";
        return "volume2";
      }
      case "play":
        return "play";
      case "pause":
        return "pause";
      case "next":
        return "skipForward";
      case "prev":
        return "skipBack";
    }
  }
</script>

{#if item}
  <div
    class={[
      "keybind-hud",
      item.kind === "seek" && "keybind-hud--seek",
      item.kind === "seek" && item.direction === "back" && "keybind-hud--back",
      item.kind === "seek" &&
        item.direction === "forward" &&
        "keybind-hud--forward",
    ]}
    role="status"
    aria-live="polite"
    out:fade={{ duration: 180 * motion }}
  >
    {#key item.id}
      <div
        class="keybind-hud__card"
        in:scale={{ duration: 140 * motion, start: motion ? 0.88 : 1 }}
      >
        <MdiIcon name={iconName(item)} size={36} aria-hidden="true" />
        <span class="keybind-hud__label">{item.label}</span>
        {#if item.kind === "volume"}
          <div class="keybind-hud__bar" aria-hidden="true">
            <div
              class="keybind-hud__bar-fill"
              style:width={`${item.volumePct ?? 0}%`}
            ></div>
          </div>
        {/if}
      </div>
    {/key}
  </div>
{/if}

<style>
  .keybind-hud {
    position: fixed;
    left: 50%;
    top: 42%;
    transform: translate(-50%, -50%);
    z-index: 90;
    pointer-events: none;
  }

  .keybind-hud--back {
    left: 18%;
  }

  .keybind-hud--forward {
    left: 82%;
  }

  .keybind-hud__card {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 0.35rem;
    min-width: 6.5rem;
    padding: 0.9rem 1.15rem 0.85rem;
    border-radius: var(--jb-radius-xl);
    border: 1px solid var(--jb-border);
    background: var(--jb-island-bg);
    box-shadow: var(--jb-shadow-lg);
    color: var(--jb-text);
    backdrop-filter: blur(10px);
  }

  .keybind-hud--seek .keybind-hud__card {
    min-width: 7.25rem;
  }

  .keybind-hud__label {
    font-size: 1.125rem;
    font-weight: 700;
    letter-spacing: 0.02em;
    line-height: 1.2;
  }

  .keybind-hud__bar {
    width: 100%;
    height: 0.28rem;
    margin-top: 0.15rem;
    border-radius: 999px;
    background: var(--jb-bg-muted);
    overflow: hidden;
  }

  .keybind-hud__bar-fill {
    height: 100%;
    border-radius: inherit;
    background: var(--jb-accent);
  }
</style>
