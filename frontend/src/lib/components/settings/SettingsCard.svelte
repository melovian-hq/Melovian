<script lang="ts">
  interface Props {
    title: string;
    description?: string;
    id?: string;
    class?: string;
    children?: import("svelte").Snippet;
    status?: import("svelte").Snippet;
    footer?: import("svelte").Snippet;
  }

  let {
    title,
    description = "",
    id = undefined,
    class: className = "",
    children,
    status,
    footer,
  }: Props = $props();
</script>

<section {id} class="settings-card {className}">
  <header class="settings-card__header">
    <div class="settings-card__heading">
      <h2 class="settings-card__title">{title}</h2>
      {#if description}
        <p class="settings-card__description">{description}</p>
      {/if}
    </div>
    {@render status?.()}
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
    display: grid;
    grid-template-columns: minmax(0, 1fr);
    gap: var(--jb-space-4);
    padding: var(--jb-space-6) 0;
  }

  .settings-card:first-child {
    padding-top: 0;
  }

  .settings-card__header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: var(--jb-space-3);
    min-width: 0;
  }

  .settings-card__heading {
    min-width: 0;
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
    min-width: 0;
  }

  .settings-card__footer {
    display: flex;
    flex-wrap: wrap;
    gap: var(--jb-space-2);
    padding-top: var(--jb-space-4);
    border-top: 1px solid var(--jb-border);
  }

  @media (min-width: 1024px) {
    .settings-card {
      grid-template-columns: minmax(12rem, 16rem) minmax(0, 1fr);
      column-gap: var(--jb-space-8);
    }

    .settings-card__header {
      grid-row: 1 / -1;
      flex-direction: column;
      justify-content: flex-start;
      gap: var(--jb-space-2);
    }

    .settings-card__footer {
      border-top: none;
      padding-top: 0;
    }
  }
</style>
