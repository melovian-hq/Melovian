<script lang="ts">
  interface Props {
    title: string;
    subtitle?: string;
    class?: string;
    actions?: import("svelte").Snippet;
  }

  let {
    title,
    subtitle = "",
    class: className = "",
    actions,
  }: Props = $props();
</script>

<header class="page-header {className}">
  <div class="page-header__copy">
    <h1 class="page-header__title">{title}</h1>
    {#if subtitle}
      <p class="page-header__subtitle">{subtitle}</p>
    {/if}
  </div>
  {#if actions}
    <div class="page-header__actions">
      {@render actions()}
    </div>
  {/if}
</header>

<style>
  .page-header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: var(--jb-space-4);
    margin-bottom: var(--jb-space-6);
  }

  .page-header__title {
    margin: 0;
    font-size: 1.75rem;
    line-height: 1.2;
    color: var(--jb-text);
  }

  .page-header__subtitle {
    margin: var(--jb-space-2) 0 0;
    color: var(--jb-text-muted);
    line-height: 1.6;
  }

  .page-header__actions {
    display: flex;
    gap: var(--jb-space-2);
    flex-shrink: 0;
  }

  @media (max-width: 640px) {
    .page-header {
      flex-direction: column;
      align-items: stretch;
      margin-bottom: var(--jb-space-4);
    }

    .page-header__title {
      font-size: 1.375rem;
    }

    .page-header__subtitle {
      font-size: 0.9375rem;
    }

    .page-header__actions {
      flex-wrap: wrap;
    }
  }
</style>
