<script lang="ts">
  import ErrorFallback from "$lib/components/ui/ErrorFallback.svelte";
  import { logClientError } from "$lib/core/logger";

  interface Props {
    children: import("svelte").Snippet;
  }

  let { children }: Props = $props();

  function onerror(error: unknown) {
    logClientError(error, "route");
  }
</script>

<svelte:boundary {onerror}>
  {@render children()}
  {#snippet failed(error, reset)}
    <ErrorFallback {error} onretry={reset} />
  {/snippet}
</svelte:boundary>
