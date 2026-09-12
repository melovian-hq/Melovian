<script lang="ts">
  import AppShell from "$lib/components/layout/AppShell.svelte";
  import RouteBoundary from "$lib/router/RouteBoundary.svelte";
  import RouteOutlet from "$lib/router/RouteOutlet.svelte";
  import { router, type RouteDefinition } from "$lib/router/router.svelte";

  let { routes }: { routes: RouteDefinition[] } = $props();

  const match = $derived(router.match(routes));
</script>

{#if match}
  <AppShell
    bare={match.route.bare ?? false}
    content={match.route.content ?? "default"}
  >
    <RouteBoundary>
      <RouteOutlet
        load={match.route.load}
        component={match.route.component}
        path={match.route.path}
        params={match.match.params}
      />
    </RouteBoundary>
  </AppShell>
{/if}
