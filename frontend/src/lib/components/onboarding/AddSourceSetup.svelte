<script lang="ts">
  import AppLogo from "$lib/components/ui/AppLogo.svelte";
  import { APP_NAME } from "$lib/brand";
  import Button from "$lib/components/ui/Button.svelte";
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";
  import InstanceForm from "$lib/components/instances/InstanceForm.svelte";
  import LocalLibraryForm from "$lib/components/instances/LocalLibraryForm.svelte";
  import { instances } from "$lib/features/instances/store.svelte";
  import { localLibraries } from "$lib/features/local-libraries/store.svelte";
  import { sources } from "$lib/features/sources/store.svelte";
  import type { InstanceInput } from "$lib/features/instances/types";
  import type { LocalLibraryInput } from "$lib/features/local-libraries/types";
  import { music } from "$lib/config/music.svelte";
  import { isWailsMobile } from "$lib/config/runtime";
  import { toast } from "$lib/ui/toast.svelte";
  import { parseQuery } from "$lib/router/match";
  import { router } from "$lib/router/router.svelte";

  type SetupPath = "pick" | "server" | "local";

  interface Props {
    /** first-run shows welcome copy. add is denser for existing libraries. */
    mode?: "first-run" | "add";
    oncomplete?: () => void;
    oncancel?: () => void;
  }

  let { mode = "first-run", oncomplete, oncancel }: Props = $props();

  const localAvailable = $derived(localLibraries.enabled && !isWailsMobile());

  function initialPath(): SetupPath {
    const source = parseQuery(router.search).source;
    if (source === "local" && !isWailsMobile()) return "local";
    if (source === "server") return "server";
    if (mode === "add" || isWailsMobile()) return "server";
    return "pick";
  }

  let path = $state<SetupPath>(initialPath());
  let testing = $state(false);
  let saving = $state(false);
  let savingLibrary = $state(false);

  $effect(() => {
    if (path !== "local" || localAvailable || !localLibraries.ready) return;
    path = mode === "first-run" ? "pick" : "server";
  });

  async function finishAfterSource() {
    await sources.refreshStatus();
    const ok = await music.connect({ force: true });
    if (!ok) {
      toast.warning("Source saved. Library is still loading.");
    }
    oncomplete?.();
    if (!oncomplete) {
      router.navigate("/music", true);
    }
  }

  async function handleTest(input: InstanceInput) {
    testing = true;
    try {
      const serverName = await instances.test(input);
      toast.success(`Connected to ${serverName}`);
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Connection test failed",
      );
    } finally {
      testing = false;
    }
  }

  async function handleServerSubmit(input: InstanceInput) {
    saving = true;
    try {
      const created = await instances.add(input);
      toast.success(`Connected to ${created.serverName || created.name}`);
      await finishAfterSource();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to add server");
    } finally {
      saving = false;
    }
  }

  async function handleLocalSubmit(input: LocalLibraryInput) {
    savingLibrary = true;
    try {
      const created = await localLibraries.add(input);
      toast.info(`Added ${created.name}. Scanning in background.`);
      await finishAfterSource();
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to add local library",
      );
    } finally {
      savingLibrary = false;
    }
  }

  function backToPick() {
    if (mode === "add" && oncancel) {
      oncancel();
      return;
    }
    if (localAvailable && mode === "first-run") {
      path = "pick";
      return;
    }
    oncancel?.();
  }
</script>

<div class="add-source" class:add-source--add={mode === "add"}>
  <div class="add-source__brand">
    <AppLogo size={mode === "first-run" ? 56 : 40} />
  </div>

  {#if path === "pick"}
    <header class="add-source__header">
      <h1>Welcome to {APP_NAME}</h1>
      <p>Pick a music source to start browsing and playing.</p>
    </header>

    <div class="add-source__choices">
      <button
        type="button"
        class="add-source__choice"
        onclick={() => (path = "server")}
      >
        <span class="add-source__choice-icon" aria-hidden="true">
          <MdiIcon name="server" size={24} />
        </span>
        <span class="add-source__choice-copy">
          <strong>Subsonic server</strong>
          <span
            >Navidrome, Jellyfin Subsonic plugin, or any compatible host.</span
          >
        </span>
        <MdiIcon name="chevronRight" size={20} />
      </button>

      {#if localAvailable}
        <button
          type="button"
          class="add-source__choice"
          onclick={() => (path = "local")}
        >
          <span class="add-source__choice-icon" aria-hidden="true">
            <MdiIcon name="folderMusic" size={24} />
          </span>
          <span class="add-source__choice-copy">
            <strong>Local folder</strong>
            <span
              >Index audio on this machine. You can play while it scans.</span
            >
          </span>
          <MdiIcon name="chevronRight" size={20} />
        </button>
      {/if}
    </div>
  {:else if path === "server"}
    <header class="add-source__header">
      {#if localAvailable && mode === "first-run"}
        <button type="button" class="add-source__back" onclick={backToPick}>
          <MdiIcon name="chevronLeft" size={18} />
          Back
        </button>
      {:else if oncancel}
        <button type="button" class="add-source__back" onclick={backToPick}>
          <MdiIcon name="chevronLeft" size={18} />
          Cancel
        </button>
      {/if}
      <h1>
        {mode === "first-run" ? "Connect your server" : "Add a server"}
      </h1>
      <p>Enter the Subsonic API URL and the account you use on that server.</p>
    </header>

    <InstanceForm
      submitLabel={mode === "first-run" ? "Connect and continue" : "Add server"}
      {testing}
      {saving}
      stickyActions
      autofocusUrl
      ontest={handleTest}
      onsubmit={handleServerSubmit}
    />
  {:else}
    <header class="add-source__header">
      <button type="button" class="add-source__back" onclick={backToPick}>
        <MdiIcon name="chevronLeft" size={18} />
        Back
      </button>
      <h1>Add a local library</h1>
      <p>Pick a folder of audio files on this computer.</p>
    </header>

    <LocalLibraryForm
      submitLabel="Add and continue"
      saving={savingLibrary}
      onsubmit={handleLocalSubmit}
    />
  {/if}

  {#if mode === "add" && path === "server" && !oncancel}
    <div class="add-source__footer">
      <Button
        variant="ghost"
        onclick={() => router.navigate("/settings/servers")}
      >
        Manage existing servers
      </Button>
    </div>
  {/if}
</div>

<style>
  .add-source {
    display: flex;
    flex-direction: column;
    gap: var(--jb-space-5);
    width: min(100%, 28rem);
    margin: 0 auto;
  }

  .add-source__brand {
    display: flex;
    justify-content: center;
  }

  .add-source__header {
    display: flex;
    flex-direction: column;
    gap: var(--jb-space-2);
    text-align: center;
  }

  .add-source__header h1 {
    margin: 0;
    font-size: 1.5rem;
    line-height: 1.25;
  }

  .add-source__header p {
    margin: 0;
    color: var(--jb-text-muted);
    line-height: 1.5;
  }

  .add-source__back {
    align-self: flex-start;
    display: inline-flex;
    align-items: center;
    gap: var(--jb-space-1);
    margin-bottom: var(--jb-space-1);
    padding: 0.25rem 0;
    border: none;
    background: transparent;
    color: var(--jb-accent);
    font: inherit;
    font-weight: 600;
    cursor: pointer;
  }

  .add-source__choices {
    display: grid;
    gap: var(--jb-space-3);
  }

  .add-source__choice {
    display: grid;
    grid-template-columns: auto 1fr auto;
    align-items: center;
    gap: var(--jb-space-3);
    width: 100%;
    min-height: 4.5rem;
    padding: var(--jb-space-4);
    border: 1px solid var(--jb-border);
    border-radius: var(--jb-radius-lg);
    background: var(--jb-surface);
    color: var(--jb-text);
    text-align: left;
    cursor: pointer;
    transition:
      border-color var(--jb-transition),
      background var(--jb-transition);
  }

  .add-source__choice:hover,
  .add-source__choice:focus-visible {
    border-color: var(--jb-accent);
    outline: none;
  }

  .add-source__choice-icon {
    display: grid;
    place-content: center;
    width: 2.75rem;
    height: 2.75rem;
    border-radius: var(--jb-radius-full);
    background: var(--jb-accent-muted);
    color: var(--jb-accent);
  }

  .add-source__choice-copy {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
    min-width: 0;
  }

  .add-source__choice-copy strong {
    font-size: 1rem;
  }

  .add-source__choice-copy span {
    color: var(--jb-text-muted);
    font-size: 0.875rem;
    line-height: 1.4;
  }

  .add-source__footer {
    display: flex;
    justify-content: center;
  }

  @media (max-width: 480px) {
    .add-source__header h1 {
      font-size: 1.35rem;
    }
  }
</style>
