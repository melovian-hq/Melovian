<script lang="ts">
  interface Props {
    class?: string;
    children?: import("svelte").Snippet;
  }

  let { class: className = "" }: Props = $props();
</script>

<div class="spinner jb-spin {className}" role="status" aria-label="Loading">
  <div class="spinner__ring"></div>
</div>

<style>
  .spinner {
    display: inline-flex;
    align-items: center;
    justify-content: center;
  }

  .spinner__ring {
    width: 1.5rem;
    height: 1.5rem;
    border-radius: var(--jb-radius-full);
    border: 2px solid var(--jb-border);
    border-top-color: var(--jb-accent);
    animation: spin 0.8s linear infinite;
  }

  @keyframes spin {
    to {
      transform: rotate(360deg);
    }
  }

  @media (prefers-reduced-motion: reduce) {
    .spinner__ring {
      animation: none;
    }
  }
</style>
