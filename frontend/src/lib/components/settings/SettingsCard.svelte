<script lang="ts">
  interface Props {
    title: string;
    description?: string;
    id?: string;
    class?: string;
    children?: import("svelte").Snippet;
    footer?: import("svelte").Snippet;
  }

  let {
    title,
    description = "",
    id = undefined,
    class: className = "",
    children,
    footer,
  }: Props = $props();
</script>

<section {id} class="settings-card {className}">
  <header class="settings-card__header">
    <h2 class="settings-card__title">{title}</h2>
    {#if description}
      <p class="settings-card__description">{description}</p>
    {/if}
  </header>
  <div class="settings-card__body">
    {@render children?.()}
  </div>
  {#if footer}
    <footer class="settings-card__footer">
      {@render footer()}
    </footer>
  {/if}
</section>

<style>
  .settings-card {
    border-radius: var(--jb-radius-xl);
    border: 1px solid var(--jb-border);
    background: var(--jb-surface);
    box-shadow: var(--jb-shadow-sm);
    padding: var(--jb-space-5);
  }

  .settings-card__header {
    margin-bottom: var(--jb-space-4);
  }

  .settings-card__title {
    margin: 0;
    font-size: 0.9375rem;
    font-weight: 700;
    color: var(--jb-text);
  }

  .settings-card__description {
    margin: var(--jb-space-1) 0 0;
    font-size: 0.875rem;
    line-height: 1.55;
    color: var(--jb-text-muted);
  }

  .settings-card__body {
    display: flex;
    flex-direction: column;
    gap: var(--jb-space-4);
  }

  .settings-card__footer {
    display: flex;
    flex-wrap: wrap;
    gap: var(--jb-space-2);
    margin-top: var(--jb-space-4);
    padding-top: var(--jb-space-4);
    border-top: 1px solid var(--jb-border);
  }
</style>
