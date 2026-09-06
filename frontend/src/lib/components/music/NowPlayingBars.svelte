<script lang="ts">
  interface Props {
    size?: "sm" | "md" | "lg";
  }

  let { size = "sm" }: Props = $props();
</script>

<span class="now-playing-bars now-playing-bars--{size} jb-motion" aria-hidden="true">
  <span class="now-playing-bars__bar"></span>
  <span class="now-playing-bars__bar"></span>
  <span class="now-playing-bars__bar"></span>
</span>

<style>
  .now-playing-bars {
    display: inline-flex;
    align-items: flex-end;
    gap: 2px;
    flex-shrink: 0;
  }

  .now-playing-bars--sm {
    width: 0.875rem;
    height: 0.875rem;
  }

  .now-playing-bars--md {
    width: 1.125rem;
    height: 1.125rem;
  }

  .now-playing-bars--lg {
    width: 1.75rem;
    height: 1.75rem;
    gap: 3px;
  }

  .now-playing-bars__bar {
    display: block;
    width: 3px;
    border-radius: 1px;
    background: var(--now-playing-bar-color, var(--jb-accent));
    animation: bar-bounce 0.9s ease-in-out infinite;
  }

  .now-playing-bars--lg .now-playing-bars__bar {
    width: 4px;
    border-radius: 2px;
  }

  .now-playing-bars__bar:nth-child(1) {
    height: 40%;
    animation-delay: 0s;
  }

  .now-playing-bars__bar:nth-child(2) {
    height: 100%;
    animation-delay: 0.15s;
  }

  .now-playing-bars__bar:nth-child(3) {
    height: 65%;
    animation-delay: 0.3s;
  }

  @keyframes bar-bounce {
    0%,
    100% {
      transform: scaleY(0.45);
      opacity: 0.65;
    }
    50% {
      transform: scaleY(1);
      opacity: 1;
    }
  }

  @media (prefers-reduced-motion: reduce) {
    .now-playing-bars__bar {
      animation: none;
      transform: scaleY(0.7);
      opacity: 0.85;
    }
  }
</style>
