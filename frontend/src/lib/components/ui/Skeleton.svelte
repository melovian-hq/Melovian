<script lang="ts">
  interface Props {
    class?: string;
    variant?: "block" | "row" | "card" | "avatar" | "hero";
  }

  let { class: className = "", variant = "block" }: Props = $props();
</script>

<div class="skeleton skeleton--{variant} {className}" aria-hidden="true"></div>

<style>
  .skeleton {
    border-radius: var(--jb-radius-md);
    background: linear-gradient(
      90deg,
      var(--jb-bg-muted) 0%,
      var(--jb-surface-hover) 50%,
      var(--jb-bg-muted) 100%
    );
    background-size: 200% 100%;
    animation: skeleton-pulse 1.4s ease-in-out infinite;
  }

  .skeleton--block {
    width: 100%;
    height: 1rem;
  }

  .skeleton--row {
    width: 100%;
    height: 3rem;
  }

  .skeleton--card {
    width: 100%;
    aspect-ratio: 1;
  }

  .skeleton--avatar {
    width: 3rem;
    height: 3rem;
    border-radius: var(--jb-radius-full);
  }

  .skeleton--hero {
    width: 100%;
    height: 10rem;
    border-radius: var(--jb-radius-lg);
  }

  @keyframes skeleton-pulse {
    0% {
      background-position: 100% 0;
    }
    100% {
      background-position: -100% 0;
    }
  }

  @media (prefers-reduced-motion: reduce) {
    .skeleton {
      animation: none;
      background-position: 50% 0;
    }
  }
</style>
