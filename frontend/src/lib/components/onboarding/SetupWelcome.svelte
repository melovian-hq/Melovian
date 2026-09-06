<script lang="ts">
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";
  import Button from "$lib/components/ui/Button.svelte";
  import { isWailsMobile } from "$lib/config/runtime";
  import { APP_NAME, APP_SLUG } from "$lib/brand";
  import { router } from "$lib/router/router.svelte";

  interface Props {
    class?: string;
  }

  let { class: className = "" }: Props = $props();

  const mobile = $derived(isWailsMobile());
</script>

<div class="setup-welcome {className}">
  <div class="setup-welcome__hero">
    <div class="setup-welcome__icon" aria-hidden="true">
      <MdiIcon name="music" size={28} />
    </div>
    <h2 class="setup-welcome__title">Welcome to {APP_NAME}</h2>
    <p class="setup-welcome__lead">
      {mobile
        ? `Connect your ${APP_NAME} server (Docker or ${APP_SLUG}-server), or add a Subsonic server on this device.`
        : "Connect a Subsonic server or index a local music folder to start listening."}
    </p>
  </div>

  <div class="setup-welcome__actions">
    {#if mobile}
      <Button
        size="lg"
        onclick={() => router.navigate("/settings/servers#melovian-host")}
      >
        <MdiIcon name="server" size={18} />
        Connect {APP_NAME} server
      </Button>
      <Button
        size="lg"
        variant="surface"
        onclick={() => router.navigate("/setup")}
      >
        <MdiIcon name="server" size={18} />
        Add Subsonic on this device
      </Button>
    {:else}
      <Button size="lg" onclick={() => router.navigate("/setup")}>
        <MdiIcon name="server" size={18} />
        Set up a music source
      </Button>
      <Button
        size="lg"
        variant="surface"
        onclick={() => router.navigate("/setup?source=local")}
      >
        <MdiIcon name="folderMusic" size={18} />
        Add local folder
      </Button>
    {/if}
  </div>

  <p class="setup-welcome__hint">
    You can change sources later from Settings or the source menu.
  </p>
</div>

<style>
  .setup-welcome {
    box-sizing: border-box;
    display: flex;
    flex-direction: column;
    align-items: stretch;
    gap: var(--jb-space-5);
    width: 100%;
    max-width: 28rem;
    min-width: 0;
    margin: 0 auto;
    padding: var(--jb-space-6) var(--jb-space-4);
    border: 1px dashed var(--jb-border);
    border-radius: var(--jb-radius-lg);
    background: var(--jb-bg-subtle);
    overflow-wrap: anywhere;
  }

  .setup-welcome__hero {
    display: flex;
    flex-direction: column;
    align-items: center;
    text-align: center;
    gap: var(--jb-space-3);
    min-width: 0;
  }

  .setup-welcome__icon {
    display: grid;
    place-content: center;
    width: 3.5rem;
    height: 3.5rem;
    border-radius: var(--jb-radius-full);
    background: var(--jb-accent-muted);
    color: var(--jb-accent);
  }

  .setup-welcome__title {
    margin: 0;
    max-width: 100%;
    font-size: 1.5rem;
  }

  .setup-welcome__lead {
    margin: 0;
    max-width: 100%;
    color: var(--jb-text-muted);
    line-height: 1.6;
  }

  .setup-welcome__actions {
    display: grid;
    gap: var(--jb-space-3);
    min-width: 0;
  }

  .setup-welcome__actions :global(.btn) {
    width: 100%;
    max-width: 100%;
  }

  .setup-welcome__hint {
    margin: 0;
    text-align: center;
    color: var(--jb-text-subtle);
    font-size: 0.8125rem;
    line-height: 1.4;
  }

  @media (max-width: 480px) {
    .setup-welcome {
      gap: var(--jb-space-4);
      padding: var(--jb-space-4) var(--jb-space-3);
      border-style: solid;
    }

    .setup-welcome__title {
      font-size: 1.25rem;
    }

    .setup-welcome__icon {
      width: 3rem;
      height: 3rem;
    }
  }
</style>
