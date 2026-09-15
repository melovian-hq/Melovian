<script lang="ts">
  import type { Component } from "svelte";
  import { fade } from "svelte/transition";
  import Spinner from "$lib/components/ui/Spinner.svelte";
  import ErrorFallback from "$lib/components/ui/ErrorFallback.svelte";
  import type { RouteComponent, RouteLoader } from "./router.svelte";
  import { routePropsFromMatch } from "./route-props";
  import {
    captureViewSnapshot,
    pageInFly,
    pageOutFade,
    pinOutgoingPage,
    resetPageScroll,
    routeViewKey,
    type PinnedViewSnapshot,
  } from "./page-motion";
  import { outletState } from "./outlet-state.svelte";
  import type { RouteContentLayout } from "./router.svelte";
  import { attemptStaleReload, isStaleChunkError } from "$lib/ui/stale-chunk";
  import { APP_NAME } from "$lib/brand";

  interface Props {
    load?: RouteLoader;
    component?: RouteComponent;
    path: string;
    params: Record<string, string>;
    content?: RouteContentLayout;
    bare?: boolean;
  }

  let {
    load,
    component,
    path,
    params,
    content = "default",
    bare = false,
  }: Props = $props();

  let Page = $state<Component | null>(null);
  let shownPath = $state("");
  let shownParams = $state<Record<string, string>>({});
  let coldLoading = $state(true);
  let loadError = $state<string | null>(null);
  let staleError = $state(false);
  let refreshing = $state(false);
  let retryToken = $state(0);
  let outletEl = $state<HTMLDivElement | null>(null);
  let viewSnapshot: PinnedViewSnapshot | null = null;

  const viewKey = $derived(routeViewKey(shownPath, shownParams));
  const pageProps: Record<string, string> = $derived(
    routePropsFromMatch(shownPath, shownParams),
  );

  function captureView() {
    const el = outletEl;
    if (el) viewSnapshot = captureViewSnapshot(el);
  }

  // Keep a fresh geometry snapshot of the stable view. Route commits can
  // shift the outlet (content padding flips between compact and fill), and
  // onoutrostart reads this to pin the leaving page where it last painted.
  // ResizeObserver covers sidebar, window, and chrome resizes between
  // commits; the viewKey rAF below covers commits that do not resize.
  $effect(() => {
    const el = outletEl;
    if (!el) return;
    captureView();
    if (typeof ResizeObserver === "undefined") return;
    const observer = new ResizeObserver(captureView);
    observer.observe(el);
    return () => observer.disconnect();
  });

  // The shell scroll container is shared across routes. Reset it when a new
  // view commits. onoutrostart already resets during crossfades, but zero
  // duration transitions and environments without WAAPI never fire it.
  $effect(() => {
    void viewKey;
    const el = outletEl;
    if (!el) return;
    const raf = requestAnimationFrame(() => {
      resetPageScroll(el);
      captureView();
    });
    return () => cancelAnimationFrame(raf);
  });

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
    const activeContent = content;
    const activeBare = bare;
    void retryToken;
    let cancelled = false;

    loadError = null;
    staleError = false;

    const commit = (next: Component) => {
      Page = next;
      shownPath = activePath;
      shownParams = activeParams;
      // The shell only re-lays-out once the incoming page is ready, so the
      // old view keeps its compact/fill padding while a lazy chunk loads.
      outletState.content = activeContent;
      outletState.bare = activeBare;
      coldLoading = false;
    };

    const nextKey = routeViewKey(activePath, activeParams);
    if (Page !== null && routeViewKey(shownPath, shownParams) === nextKey) {
      if (shownPath !== activePath || !paramsEqual(shownParams, activeParams)) {
        shownPath = activePath;
        shownParams = activeParams;
      }
      if (activeComponent && Page !== activeComponent) {
        commit(activeComponent);
      } else {
        coldLoading = false;
      }
      return;
    }

    if (activeComponent) {
      commit(activeComponent);
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
        commit(mod.default);
      })
      .catch((err) => {
        if (cancelled) return;
        coldLoading = false;
        staleError = isStaleChunkError(err);
        if (staleError && attemptStaleReload() === "reloading") {
          refreshing = true;
          return;
        }
        loadError = staleError
          ? `${APP_NAME} was updated in the background. Refresh to load the latest version.`
          : err instanceof Error
            ? err.message
            : "Failed to load page";
      });

    return () => {
      cancelled = true;
    };
  });

  function retryLoad() {
    // Retrying a stale chunk re-imports the same missing URL, so the
    // recovery path is a reload rather than another import attempt.
    if (staleError) {
      window.location.reload();
      return;
    }
    retryToken += 1;
  }

  function onPageOutroStart(event: Event) {
    const node = event.currentTarget;
    if (node instanceof HTMLElement) {
      pinOutgoingPage(node, viewSnapshot);
      resetPageScroll(node);
    }
  }
</script>

{#if refreshing}
  <div class="route-loading">
    <Spinner />
    <p class="route-refreshing">Updating {APP_NAME}...</p>
  </div>
{:else if coldLoading && !Page}
  <div class="route-loading"><Spinner /></div>
{:else if loadError}
  <!-- A failed lazy load replaces the stale page so the retry affordance is
       visible. Page is kept in state, so a successful retry swaps straight
       back in instead of falling through to the cold spinner. -->
  <ErrorFallback error={loadError} onretry={retryLoad} />
{:else if Page}
  <div class="route-outlet" bind:this={outletEl}>
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
    /* Opaque so crossfades composite cleanly. Without a page background the
       outgoing page stays visible through transparent regions of the
       incoming one and both pages' content overlaps mid transition. */
    background: var(--jb-bg);
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
    justify-items: center;
    gap: 0.75rem;
  }

  .route-refreshing {
    margin: 0;
    font-size: 0.875rem;
    color: var(--jb-text-muted);
  }
</style>
