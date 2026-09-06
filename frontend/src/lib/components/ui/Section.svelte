<script lang="ts">
  interface Props {
    title: string;
    class?: string;
    children?: import("svelte").Snippet;
    action?: import("svelte").Snippet;
  }

  let { title, class: className = "", children, action }: Props = $props();
</script>

<section class="section {className}">
  <header class="section__header">
    <h2 class="section__title">{title}</h2>
    {#if action}
      <div class="section__action">
        {@render action()}
      </div>
    {/if}
  </header>
  <div class="section__body">
    {@render children?.()}
  </div>
</section>

<style>
  .section {
    display: flex;
    flex-direction: column;
    gap: var(--jb-space-3);
    margin-bottom: var(--jb-page-gap);
  }

  .section__header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--jb-space-4);
  }

  .section__title {
    margin: 0;
    font-size: 1.125rem;
    font-weight: 650;
    letter-spacing: -0.01em;
    color: var(--jb-text);
  }

  .section__body {
    display: flex;
    flex-direction: column;
    gap: var(--jb-space-3);
  }

  @media (max-width: 640px) {
    .section {
      margin-bottom: var(--jb-space-5);
    }

    .section__header {
      flex-direction: column;
      align-items: stretch;
      gap: var(--jb-space-2);
    }

    .section__title {
      font-size: 1rem;
    }
  }
</style>
