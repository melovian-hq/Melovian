<script lang="ts">
  import PixelSmile from "$lib/components/ui/PixelSmile.svelte";
  import { profile } from "$lib/profile/profile.svelte";

  interface Props {
    seed: string;
    size?: number;
    label?: string;
    self?: boolean;
    interactive?: boolean;
    class?: string;
    active?: boolean;
  }

  let {
    seed,
    size = 28,
    label,
    self = false,
    interactive = false,
    class: className = "",
    active = false,
  }: Props = $props();

  const customUrl = $derived(
    self && profile.customAvatarUrl ? profile.customAvatarUrl : null,
  );
  const ariaLabel = $derived(label ?? "Device");
</script>

<span
  class={["device-avatar", active && "device-avatar--active", className]}
  style:width="{size}px"
  style:height="{size}px"
  aria-hidden={label ? undefined : true}
  title={label}
>
  {#if customUrl}
    <img
      src={customUrl}
      alt={ariaLabel}
      class="device-avatar__img"
      width={size}
      height={size}
    />
  {:else}
    <PixelSmile
      {seed}
      {size}
      {interactive}
      {label}
      class="device-avatar__face"
    />
  {/if}
</span>

<style>
  .device-avatar {
    position: relative;
    display: inline-flex;
    flex-shrink: 0;
    border-radius: var(--jb-radius-md);
    overflow: hidden;
    box-shadow:
      0 0 0 2px var(--avatar-stack-ring, var(--jb-bg-elevated)),
      var(--jb-shadow-sm);
  }

  .device-avatar--active {
    box-shadow:
      0 0 0 2px var(--avatar-stack-ring, var(--jb-bg-elevated)),
      0 0 0 3px color-mix(in srgb, var(--jb-accent) 70%, transparent);
    animation: device-avatar-pulse 1.4s ease-in-out infinite;
  }

  .device-avatar__img {
    display: block;
    width: 100%;
    height: 100%;
    object-fit: cover;
    border-radius: var(--jb-radius-md);
  }

  .device-avatar :global(.device-avatar__face) {
    box-shadow: none;
    border-radius: var(--jb-radius-md);
  }

  @keyframes device-avatar-pulse {
    0%,
    100% {
      opacity: 1;
    }
    50% {
      opacity: 0.82;
    }
  }

  @media (prefers-reduced-motion: reduce) {
    .device-avatar--active {
      animation: none;
    }
  }
</style>
