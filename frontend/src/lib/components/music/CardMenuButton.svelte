<script lang="ts">
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";

  interface Props {
    label: string;
    onclick?: (event: MouseEvent) => void;
  }

  let { label, onclick }: Props = $props();
</script>

<button type="button" class="card-menu-btn" aria-label={label} {onclick}>
  <MdiIcon name="dotsHorizontal" size={16} />
</button>

<style>
  .card-menu-btn {
    position: absolute;
    top: var(--jb-space-2);
    right: var(--jb-space-2);
    z-index: 1;
    display: grid;
    place-content: center;
    width: 1.75rem;
    height: 1.75rem;
    padding: 0;
    border: none;
    border-radius: var(--jb-radius-full);
    background: var(--jb-overlay-surface);
    color: var(--jb-overlay-text);
    cursor: pointer;
    opacity: 0;
    transition:
      opacity var(--jb-transition),
      background var(--jb-transition);
  }

  .card-menu-btn:hover {
    background: var(--jb-overlay-surface-hover);
  }

  .card-menu-btn:focus-visible {
    outline: none;
    box-shadow: var(--jb-focus-ring);
  }

  @media (hover: hover) {
    :global(*:hover) > .card-menu-btn,
    :global(*:focus-within) > .card-menu-btn {
      opacity: 1;
    }
  }

  @media (hover: none) {
    .card-menu-btn {
      opacity: 1;
    }
  }
</style>
