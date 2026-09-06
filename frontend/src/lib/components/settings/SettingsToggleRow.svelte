<script lang="ts">
  import Toggle from "$lib/components/ui/Toggle.svelte";

  interface Props {
    label: string;
    description?: string;
    checked: boolean;
    disabled?: boolean;
    onchange: (checked: boolean) => void;
  }

  let {
    label,
    description = "",
    checked,
    disabled = false,
    onchange,
  }: Props = $props();
</script>

<div class="settings-toggle-row">
  <div class="settings-toggle-row__copy">
    <p class="settings-toggle-row__label">{label}</p>
    {#if description}
      <p class="settings-toggle-row__description">{description}</p>
    {/if}
  </div>
  <Toggle {checked} {disabled} ariaLabel={label} {onchange} />
</div>

<style>
  .settings-toggle-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--jb-space-4);
  }

  .settings-toggle-row__copy {
    min-width: 0;
  }

  .settings-toggle-row__label {
    margin: 0;
    font-size: 0.875rem;
    font-weight: 600;
    color: var(--jb-text);
  }

  .settings-toggle-row__description {
    margin: 0.25rem 0 0;
    font-size: 0.8125rem;
    line-height: 1.5;
    color: var(--jb-text-muted);
  }
</style>
