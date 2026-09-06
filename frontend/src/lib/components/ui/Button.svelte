<script lang="ts">
  interface Props {
    size?: "sm" | "md" | "lg";
    variant?: "primary" | "ghost" | "surface";
    type?: "button" | "submit" | "reset";
    disabled?: boolean;
    class?: string;
    title?: string;
    "aria-label"?: string;
    onclick?: (event: MouseEvent) => void;
    children?: import("svelte").Snippet;
  }

  let {
    size = "md",
    variant = "primary",
    type = "button",
    disabled = false,
    class: className = "",
    title,
    "aria-label": ariaLabel,
    onclick,
    children,
  }: Props = $props();
</script>

<button
  {type}
  {disabled}
  {onclick}
  {title}
  aria-label={ariaLabel}
  class="btn btn--{variant} btn--{size} {className}"
>
  {@render children?.()}
</button>

<style>
  .btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: var(--jb-space-2);
    border-radius: var(--jb-radius-md);
    font-weight: 600;
    cursor: pointer;
    transition:
      background var(--jb-transition),
      color var(--jb-transition),
      border-color var(--jb-transition),
      transform var(--jb-transition);
    border: 1px solid transparent;
  }

  .btn:disabled {
    opacity: 0.55;
    cursor: not-allowed;
  }

  .btn:focus-visible {
    outline: none;
    box-shadow: var(--jb-focus-ring);
  }

  .btn:not(:disabled):active {
    transform: scale(0.98);
  }

  @media (prefers-reduced-motion: reduce) {
    .btn:not(:disabled):active {
      transform: none;
    }
  }

  .btn--sm {
    padding: 0.375rem 0.75rem;
    font-size: 0.875rem;
  }

  .btn--md {
    padding: 0.5rem 1rem;
    font-size: 0.9375rem;
  }

  .btn--lg {
    padding: 0.75rem 1.25rem;
    font-size: 1rem;
  }

  .btn--primary {
    background: var(--jb-accent);
    color: var(--jb-accent-text);
  }

  .btn--primary:not(:disabled):hover {
    background: var(--jb-accent-hover);
  }

  .btn--ghost {
    background: transparent;
    color: var(--jb-text);
    border-color: var(--jb-border);
  }

  .btn--ghost:not(:disabled):hover {
    background: var(--jb-surface-hover);
  }

  .btn--surface {
    background: var(--jb-surface);
    color: var(--jb-text);
    border-color: var(--jb-border);
  }

  .btn--surface:not(:disabled):hover {
    background: var(--jb-surface-hover);
  }
</style>
