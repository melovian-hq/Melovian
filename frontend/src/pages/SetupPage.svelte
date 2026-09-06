<script lang="ts">
  import AddSourceSetup from "$lib/components/onboarding/AddSourceSetup.svelte";
  import { sources } from "$lib/features/sources/store.svelte";
  import { router } from "$lib/router/router.svelte";

  const mode = $derived(sources.needsSetup ? "first-run" : "add");

  function oncancel() {
    if (window.history.length > 1) {
      window.history.back();
      return;
    }
    router.navigate(sources.needsSetup ? "/music" : "/settings/servers");
  }
</script>

<div class="setup-page">
  <div class="setup-page__panel">
    <AddSourceSetup {mode} {oncancel} />
  </div>
</div>

<style>
  .setup-page {
    min-height: 100vh;
    min-height: 100dvh;
    display: grid;
    place-content: center;
    overflow-y: auto;
    scroll-padding-bottom: 6rem;
    padding: var(--jb-space-6) var(--jb-space-4);
    padding-top: max(var(--jb-space-6), env(safe-area-inset-top));
    padding-bottom: max(var(--jb-space-6), env(safe-area-inset-bottom));
    background:
      radial-gradient(
        circle at top,
        color-mix(in srgb, var(--jb-accent) 12%, transparent),
        transparent 42%
      ),
      var(--jb-bg);
  }

  .setup-page__panel {
    width: min(100%, 28rem);
    padding: var(--jb-space-6);
    border-radius: var(--jb-radius-xl);
    border: 1px solid var(--jb-border);
    background: var(--jb-surface);
    box-shadow: var(--jb-shadow-lg);
  }

  @media (max-width: 480px) {
    .setup-page {
      place-content: start;
      scroll-padding-bottom: 7rem;
      padding: var(--jb-space-4);
      padding-top: max(var(--jb-space-5), env(safe-area-inset-top));
      padding-bottom: max(var(--jb-space-4), env(safe-area-inset-bottom));
    }

    .setup-page__panel {
      width: 100%;
      padding: var(--jb-space-4);
      border: none;
      border-radius: 0;
      box-shadow: none;
      background: transparent;
    }
  }
</style>
