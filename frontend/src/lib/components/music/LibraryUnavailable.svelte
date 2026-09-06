<script lang="ts">
  import EmptyState from "$lib/components/ui/EmptyState.svelte";
  import SetupWelcome from "$lib/components/onboarding/SetupWelcome.svelte";
  import Link from "$lib/router/Link.svelte";
  import { music } from "$lib/config/music.svelte";
  import { sources } from "$lib/features/sources/store.svelte";
  import { isWailsMobile } from "$lib/config/runtime";

  interface Props {
    class?: string;
  }

  let { class: className = "" }: Props = $props();

  const setupMessage = $derived(
    isWailsMobile()
      ? "Connect a Subsonic server to browse and play your library."
      : "Connect a Subsonic server or a local folder to browse and play your library.",
  );
</script>

{#if sources.needsSetup}
  <SetupWelcome class={className} />
{:else if music.error}
  <EmptyState
    class={className}
    title="Connection failed"
    message={music.error}
    icon="alertCircle"
  >
    {#snippet actions()}
      <Link href="/setup" class="library-unavailable__link">
        Fix server connection
      </Link>
      <Link href="/settings/servers" class="library-unavailable__link">
        Open source settings
      </Link>
    {/snippet}
  </EmptyState>
{:else if !sources.hasAnySource}
  <EmptyState
    class={className}
    title="No music source"
    message={setupMessage}
    icon="music"
  >
    {#snippet actions()}
      <Link href="/setup" class="library-unavailable__link">Add a server</Link>
      {#if !isWailsMobile()}
        <Link href="/setup" class="library-unavailable__link">
          Add local folder
        </Link>
      {/if}
    {/snippet}
  </EmptyState>
{:else}
  <EmptyState
    class={className}
    title="Library unavailable"
    message="The active source is missing or not ready. Check your server connection or local library in Settings."
    icon="alertCircle"
  >
    {#snippet actions()}
      <Link href="/settings/servers" class="library-unavailable__link">
        Open source settings
      </Link>
    {/snippet}
  </EmptyState>
{/if}

<style>
  :global(.library-unavailable__link) {
    display: inline-flex;
    justify-content: center;
    box-sizing: border-box;
    max-width: 100%;
    font-size: 0.875rem;
    font-weight: 600;
    color: var(--jb-accent);
    text-decoration: none;
    overflow-wrap: anywhere;
  }

  :global(.library-unavailable__link:hover) {
    text-decoration: underline;
  }
</style>
