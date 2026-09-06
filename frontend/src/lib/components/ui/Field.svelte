<script lang="ts">
  interface Props {
    label: string;
    hint?: string;
    class?: string;
    children?: import("svelte").Snippet;
  }

  let { label, hint = "", class: className = "", children }: Props = $props();
</script>

<label class="field {className}">
  <span class="field__label">{label}</span>
  {@render children?.()}
  {#if hint}
    <span class="field__hint">{hint}</span>
  {/if}
</label>

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
