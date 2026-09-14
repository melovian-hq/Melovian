<script lang="ts">
  import AppShell from "$lib/components/layout/AppShell.svelte";
  import RouteBoundary from "$lib/router/RouteBoundary.svelte";
  import RouteOutlet from "$lib/router/RouteOutlet.svelte";
  import { outletState } from "$lib/router/outlet-state.svelte";
  import { router, type RouteDefinition } from "$lib/router/router.svelte";

  let { routes }: { routes: RouteDefinition[] } = $props();

  const match = $derived(router.match(routes));
</script>

{#if match}
  <AppShell bare={outletState.bare} content={outletState.content}>
    <RouteBoundary>
      <RouteOutlet
        load={match.route.load}
        component={match.route.component}
        path={match.route.path}
        params={match.match.params}
        content={match.route.content ?? "default"}
        bare={match.route.bare ?? false}
      />
    </RouteBoundary>
  </AppShell>
{/if}
