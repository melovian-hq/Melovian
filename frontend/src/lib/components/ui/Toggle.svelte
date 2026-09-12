<script lang="ts">
  import { Switch } from "bits-ui";

  interface Props {
    checked?: boolean;
    label?: string;
    ariaLabel?: string;
    disabled?: boolean;
    onchange?: (checked: boolean) => void;
  }

  let {
    checked = $bindable(false),
    label = "",
    ariaLabel,
    disabled = false,
    onchange,
  }: Props = $props();
</script>

<Switch.Root
  bind:checked
  {disabled}
  aria-label={ariaLabel ?? label}
  onCheckedChange={(value) => onchange?.(value)}
  class={checked ? "toggle toggle--on" : "toggle"}
>
  <span class="toggle__track">
    <Switch.Thumb class="toggle__thumb" />
  </span>
  {#if label}
    <span class="toggle__label">{label}</span>
  {/if}
</Switch.Root>

<style>
  :global(.toggle) {
    display: inline-flex;
    align-items: center;
    gap: var(--jb-space-2);
    border: none;
    background: transparent;
    cursor: pointer;
    padding: 0;
    color: var(--jb-text);
    font-size: 0.875rem;
  }

  :global(.toggle:disabled) {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .toggle__track {
    position: relative;
    width: 2.25rem;
    height: 1.25rem;
    border-radius: var(--jb-radius-full);
    background: var(--jb-bg-muted);
    transition: background var(--jb-transition);
  }

  :global(.toggle--on) .toggle__track {
    background: var(--jb-accent);
  }

  :global(.toggle__thumb) {
    position: absolute;
    top: 2px;
    left: 2px;
    width: 1rem;
    height: 1rem;
    border-radius: var(--jb-radius-full);
    background: white;
    transition: transform var(--jb-transition);
    box-shadow: var(--jb-shadow-sm);
  }

  :global(.toggle--on .toggle__thumb) {
    transform: translateX(1rem);
  }

  .toggle__label {
    font-weight: 500;
  }
</style>
