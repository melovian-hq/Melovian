<script lang="ts">
  import Icon from "@iconify/svelte";
  import { iconData } from "$lib/components/ui/icon-data";
  import { nativeDesktopAvailable } from "$lib/config/runtime";
  import {
    loadDesktopIntegrationSettings,
    type DesktopIntegrationSettings,
  } from "$lib/desktop/desktop-integration-settings";
  import { handleWindowCloseRequest } from "$lib/desktop/window-close";

  let settings = $state<DesktopIntegrationSettings>(
    loadDesktopIntegrationSettings(),
  );

  const showControls = $derived(
    nativeDesktopAvailable() && !settings.nativeTitleBar,
  );

  $effect(() => {
    if (!nativeDesktopAvailable()) return;
    const refresh = () => {
      settings = loadDesktopIntegrationSettings();
    };
    const onStorage = (event: StorageEvent) => {
      if (event.key !== "mel-desktop-integration") return;
      refresh();
    };
    window.addEventListener("storage", onStorage);
    window.addEventListener("mel-desktop-integration-changed", refresh);
    return () => {
      window.removeEventListener("storage", onStorage);
      window.removeEventListener("mel-desktop-integration-changed", refresh);
    };
  });

  async function minimize() {
    const { MediaService } =
      await import("@bindings/melovian/services/index.js");
    await MediaService.MinimizeMainWindow();
  }

  async function toggleMaximize() {
    const { MediaService } =
      await import("@bindings/melovian/services/index.js");
    await MediaService.ToggleMaximizeMainWindow();
  }

  async function closeWindow() {
    await handleWindowCloseRequest();
  }
</script>

{#if showControls}
  <div class="window-controls jb-no-drag" data-window-controls>
    <button
      type="button"
      class="window-controls__btn"
      onclick={() => void minimize()}
      aria-label="Minimize"
    >
      <Icon icon={iconData["lucide:minus"]} width={14} height={14} />
    </button>
    <button
      type="button"
      class="window-controls__btn"
      onclick={() => void toggleMaximize()}
      aria-label="Maximize"
    >
      <Icon icon={iconData["lucide:square"]} width={12} height={12} />
    </button>
    <button
      type="button"
      class="window-controls__btn window-controls__btn--close"
      onclick={() => void closeWindow()}
      aria-label="Close"
    >
      <Icon icon={iconData["lucide:x"]} width={14} height={14} />
    </button>
  </div>
{/if}

<svelte:window
  on:focus={() => {
    settings = loadDesktopIntegrationSettings();
  }}
/>

<style>
  .window-controls {
    display: inline-flex;
    align-items: stretch;
    height: 100%;
    flex-shrink: 0;
    pointer-events: auto;
    position: relative;
    z-index: 2;
    background: var(--jb-bg);
    border-bottom: 1px solid var(--jb-border);
  }

  .window-controls__btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 2.75rem;
    border: none;
    background: transparent;
    color: var(--jb-text-muted);
    cursor: pointer;
    transition:
      background-color var(--jb-transition),
      color var(--jb-transition);
  }

  .window-controls__btn:hover {
    background: color-mix(in srgb, var(--jb-text) 8%, transparent);
    color: var(--jb-text);
  }

  .window-controls__btn--close:hover {
    background: #e81123;
    color: white;
  }
</style>
