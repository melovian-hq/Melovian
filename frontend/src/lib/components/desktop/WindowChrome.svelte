<script lang="ts">
  import { nativeDesktopAvailable } from "$lib/config/runtime";
  import { StorageKeys } from "$lib/brand";
  import {
    DESKTOP_INTEGRATION_CHANGED_EVENT,
    loadDesktopIntegrationSettings,
  } from "$lib/desktop/desktop-integration-settings";
  import {
    showCustomWindowControls,
    handleTitleBarDoubleClick,
  } from "$lib/desktop/window-chrome";
  import WindowControls from "$lib/components/desktop/WindowControls.svelte";

  interface Props {
    class?: string;
  }

  let { class: className = "" }: Props = $props();

  let settings = $state(loadDesktopIntegrationSettings());

  const visible = $derived(showCustomWindowControls(settings));

  $effect(() => {
    if (!nativeDesktopAvailable()) return;
    const refresh = () => {
      settings = loadDesktopIntegrationSettings();
    };
    const onStorage = (event: StorageEvent) => {
      if (event.key !== StorageKeys.desktopIntegration) return;
      refresh();
    };
    window.addEventListener("storage", onStorage);
    window.addEventListener(DESKTOP_INTEGRATION_CHANGED_EVENT, refresh);
    return () => {
      window.removeEventListener("storage", onStorage);
      window.removeEventListener(DESKTOP_INTEGRATION_CHANGED_EVENT, refresh);
    };
  });
</script>

{#if visible}
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <header
    class="window-chrome jb-drag {className}"
    data-window-chrome
    ondblclick={() => void handleTitleBarDoubleClick()}
  >
    <WindowControls />
  </header>
{/if}

<svelte:window
  on:focus={() => {
    settings = loadDesktopIntegrationSettings();
  }}
/>

<style>
  .window-chrome {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    z-index: 100;
    display: flex;
    align-items: stretch;
    justify-content: flex-end;
    height: var(--jb-window-chrome-height, 2rem);
    pointer-events: auto;
    background: var(--jb-bg);
    border-bottom: 1px solid var(--jb-border);
  }

  .window-chrome :global(.window-controls) {
    position: relative;
    z-index: 1;
  }
</style>
