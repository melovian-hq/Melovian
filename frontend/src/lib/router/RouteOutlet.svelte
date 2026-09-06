<script lang="ts">
  import type { Component } from "svelte";
  import { fade } from "svelte/transition";
  import Spinner from "$lib/components/ui/Spinner.svelte";
  import ErrorFallback from "$lib/components/ui/ErrorFallback.svelte";
  import type { RouteComponent, RouteLoader } from "./router.svelte";
  import { routePropsFromMatch } from "./route-props";
  import {
    pageInFly,
    pageOutFade,
    pinOutgoingPage,
    routeViewKey,
  } from "./page-motion";

  interface Props {
    load?: RouteLoader;
    component?: RouteComponent;
    path: string;
    params: Record<string, string>;
  }

  let { load, component, path, params }: Props = $props();

  let Page = $state<Component | null>(null);
  let shownPath = $state("");
  let shownParams = $state<Record<string, string>>({});
  let coldLoading = $state(true);
  let loadError = $state<string | null>(null);
  let retryToken = $state(0);

  const viewKey = $derived(routeViewKey(shownPath, shownParams));
  const pageProps: Record<string, string> = $derived(
    routePropsFromMatch(shownPath, shownParams),
  );

  function paramsEqual(
    a: Record<string, string>,
    b: Record<string, string>,
  ): boolean {
    const keys = Object.keys(a);
    if (keys.length !== Object.keys(b).length) return false;
    return keys.every((key) => a[key] === b[key]);
  }

  $effect(() => {
    const activePath = path;
    const activeParams = { ...params };
    const activeLoad = load;
    const activeComponent = component;
    void retryToken;
    let cancelled = false;

    loadError = null;

    const nextKey = routeViewKey(activePath, activeParams);
    if (Page !== null && routeViewKey(shownPath, shownParams) === nextKey) {
      if (shownPath !== activePath || !paramsEqual(shownParams, activeParams)) {
        shownPath = activePath;
        shownParams = activeParams;
      }
      if (activeComponent && Page !== activeComponent) {
        Page = activeComponent;
      }
      coldLoading = false;
      return;
    }

    if (activeComponent) {
      Page = activeComponent;
      shownPath = activePath;
      shownParams = activeParams;
      coldLoading = false;
      return;
    }

    if (!activeLoad) {
      return;
    }

    if (Page === null) {
      coldLoading = true;
    }

    void activeLoad()
      .then((mod) => {
        if (cancelled) return;
        Page = mod.default;
        shownPath = activePath;
        shownParams = activeParams;
        coldLoading = false;
      })
      .catch((err) => {
        if (cancelled) return;
        loadError = err instanceof Error ? err.message : "Failed to load page";
        coldLoading = false;
      });

    return () => {
      cancelled = true;
    };
  });

  function retryLoad() {
    retryToken += 1;
  }

  function onPageOutroStart(event: Event) {
    const node = event.currentTarget;
    if (node instanceof HTMLElement) {
      pinOutgoingPage(node);
    }
  }
</script>

{#if coldLoading && !Page}
  <div class="route-loading"><Spinner /></div>
{:else if loadError && !Page}
  <ErrorFallback error={loadError} onretry={retryLoad} />
{:else if Page}
  <div class="route-outlet">
    {#key viewKey}
      <div
        class="route-page"
        in:fade={pageInFly()}
        out:fade={pageOutFade()}
        onoutrostart={onPageOutroStart}
      >
        <Page {...pageProps} />
      </div>
    {/key}
  </div>
{/if}

<style>
  .route-outlet {
    position: relative;
    flex: 1;
    min-height: 0;
    display: flex;
    flex-direction: column;
  }

  .route-page {
    flex: 1;
    min-height: 0;
    display: flex;
    flex-direction: column;
    width: 100%;
  }

  .route-page > :global(*) {
    flex: 1 1 auto;
    min-height: 0;
    min-width: 0;
  }

  .route-loading {
    min-height: 12rem;
    display: grid;
    place-content: center;
  }
</style>
