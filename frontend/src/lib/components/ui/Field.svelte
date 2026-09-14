<script lang="ts">
  interface Props {
    label: string;
    hint?: string;
    /** Render as a group wrapper (role=group) instead of a label. Use when the
     * field wraps buttons or more than one control, where a label would
     * forward clicks to the first control. */
    group?: boolean;
    class?: string;
    children?: import("svelte").Snippet;
  }

  let {
    label,
    hint = "",
    group = false,
    class: className = "",
    children,
  }: Props = $props();

  const labelId = $props.id();
</script>

{#if group}
  <div class="field {className}" role="group" aria-labelledby={labelId}>
    <span class="field__label" id={labelId}>{label}</span>
    {@render children?.()}
    {#if hint}
      <span class="field__hint">{hint}</span>
    {/if}
  </div>
{:else}
  <label class="field {className}">
    <span class="field__label">{label}</span>
    {@render children?.()}
    {#if hint}
      <span class="field__hint">{hint}</span>
    {/if}
  </label>
{/if}

<style>
  .field {
    display: grid;
    gap: var(--jb-space-2);
    color: var(--jb-text);
  }

  .field__label {
    font-size: 0.875rem;
    font-weight: 600;
  }

  .field__hint {
    font-size: 0.8125rem;
    color: var(--jb-text-subtle);
  }
</style>
