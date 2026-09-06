<script lang="ts">
  import AppLogo from "$lib/components/ui/AppLogo.svelte";

  interface Props {
    error: unknown;
    onretry?: () => void;
  }

  let { error, onretry }: Props = $props();

  const message = $derived(
    error instanceof Error
      ? error.message || "Something went wrong"
      : typeof error === "string"
        ? error
        : "Something went wrong",
  );
</script>

<div class="error-fallback">
  <AppLogo size={48} />
  <p class="error-fallback__eyebrow">Page error</p>
  <h1 class="error-fallback__title">This page hit a problem</h1>
  <p class="error-fallback__message">{message}</p>
  <div class="error-fallback__actions">
    {#if onretry}
      <button type="button" class="error-fallback__retry" onclick={onretry}>
        Try again
      </button>
    {/if}
    <a href="/music" class="error-fallback__home">Back to music</a>
  </div>
</div>

<style>
  .error-fallback {
    min-height: 16rem;
    display: grid;
    place-content: center;
    justify-items: center;
    gap: var(--jb-space-3);
    padding: var(--jb-space-8);
    text-align: center;
  }

  .error-fallback__eyebrow {
    margin: 0;
    color: var(--jb-danger, #f87171);
    font-size: 0.75rem;
    font-weight: 700;
    letter-spacing: 0.08em;
    text-transform: uppercase;
  }

  .error-fallback__title {
    margin: 0;
    font-size: 1.5rem;
    font-weight: 800;
  }

  .error-fallback__message {
    margin: 0;
    max-width: 32rem;
    color: var(--jb-text-muted);
    line-height: 1.6;
  }

  .error-fallback__actions {
    display: flex;
    flex-wrap: wrap;
    justify-content: center;
    gap: var(--jb-space-3);
    margin-top: var(--jb-space-2);
  }

  .error-fallback__retry,
  .error-fallback__home {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    padding: 0.625rem 1rem;
    border-radius: var(--jb-radius-full);
    font-weight: 650;
    text-decoration: none;
  }

  .error-fallback__retry {
    border: none;
    background: var(--jb-accent);
    color: white;
    cursor: pointer;
  }

  .error-fallback__home {
    border: 1px solid var(--jb-border);
    color: var(--jb-text);
  }
</style>
