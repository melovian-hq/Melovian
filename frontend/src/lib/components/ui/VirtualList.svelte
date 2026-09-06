<script lang="ts" generics="T">
  import { findScrollParent, offsetWithin } from "$lib/core/dom/scroll-parent";
  import { virtualWindow } from "./virtual-list-window";

  interface Props<T> {
    items: T[];
    itemHeight?: number;
    overscan?: number;
    maxHeight?: string;
    scrollMode?: "internal" | "document";
    class?: string;
    onNearEnd?: () => void;
    nearEndThreshold?: number;
    itemKey?: (item: T, index: number) => string | number;
    children: import("svelte").Snippet<[{ item: T; index: number }]>;
  }

  let {
    items,
    itemHeight = 58,
    overscan = 6,
    maxHeight = "none",
    scrollMode = "internal",
    class: className = "",
    onNearEnd,
    nearEndThreshold = 8,
    itemKey = (_item, index) => index,
    children,
  }: Props<T> = $props();

  let scrollEl = $state<HTMLElement | null>(null);
  let anchorEl = $state<HTMLElement | null>(null);
  let scrollTop = $state(0);
  let viewportHeight = $state(0);
  let listOffsetTop = $state(0);

  let scrollRaf = 0;
  let cachedScrollParent: HTMLElement | null = null;

  const isDocumentScroll = $derived(
    scrollMode === "document" ||
      (maxHeight === "none" && scrollMode !== "internal"),
  );

  const windowed = $derived(
    virtualWindow({
      itemCount: items.length,
      itemHeight,
      scrollTop,
      viewportHeight,
      overscan,
    }),
  );
  const totalHeight = $derived(windowed.totalHeight);
  const startIndex = $derived(windowed.startIndex);
  const endIndex = $derived(windowed.endIndex);
  const offsetY = $derived(windowed.offsetY);
  const visibleItems = $derived(items.slice(startIndex, endIndex));

  function maybeNearEnd() {
    if (
      onNearEnd &&
      items.length > 0 &&
      endIndex >= items.length - nearEndThreshold
    ) {
      onNearEnd();
    }
  }

  function clampInternalScroll() {
    if (!scrollEl || isDocumentScroll) return;
    const clamped = windowed.scrollTop;
    if (clamped !== scrollTop) {
      scrollTop = clamped;
    }
    if (scrollEl.scrollTop !== clamped) {
      scrollEl.scrollTop = clamped;
    }
  }

  function syncDocumentScrollTop() {
    if (!cachedScrollParent) return;
    const next = Math.max(0, cachedScrollParent.scrollTop - listOffsetTop);
    if (next === scrollTop) return;
    scrollTop = next;
    maybeNearEnd();
  }

  function scheduleDocumentScroll() {
    if (scrollRaf) return;
    scrollRaf = requestAnimationFrame(() => {
      scrollRaf = 0;
      syncDocumentScrollTop();
    });
  }

  function measureDocumentLayout() {
    if (!anchorEl) return;
    cachedScrollParent = findScrollParent(anchorEl);
    scrollEl = cachedScrollParent;
    listOffsetTop = offsetWithin(anchorEl, cachedScrollParent);
    viewportHeight = cachedScrollParent.clientHeight;
    syncDocumentScrollTop();
  }

  function onInternalScroll() {
    if (!scrollEl) return;
    scheduleInternalScroll();
  }

  function syncInternalScrollTop() {
    if (!scrollEl) return;
    const next = scrollEl.scrollTop;
    if (next === scrollTop) return;
    scrollTop = next;
    maybeNearEnd();
  }

  function scheduleInternalScroll() {
    if (scrollRaf) return;
    scrollRaf = requestAnimationFrame(() => {
      scrollRaf = 0;
      syncInternalScrollTop();
    });
  }

  function mountInternal(node: HTMLElement) {
    scrollEl = node;
    const measure = () => {
      const next = node.clientHeight;
      if (next > 0) {
        viewportHeight = next;
        clampInternalScroll();
        return;
      }
      const fallback = Math.min(items.length * itemHeight, 360);
      viewportHeight = Math.max(fallback, itemHeight * 4);
      clampInternalScroll();
    };
    measure();
    scrollTop = node.scrollTop;

    const ro = new ResizeObserver(() => {
      measure();
    });
    ro.observe(node);
    requestAnimationFrame(measure);

    return {
      destroy() {
        ro.disconnect();
        if (scrollRaf) cancelAnimationFrame(scrollRaf);
        if (scrollEl === node) scrollEl = null;
      },
    };
  }

  function mountDocument(node: HTMLElement) {
    anchorEl = node;
    measureDocumentLayout();

    const onScroll = () => scheduleDocumentScroll();
    const ro = new ResizeObserver(() => measureDocumentLayout());
    const scrollParent = cachedScrollParent ?? findScrollParent(node);

    scrollParent.addEventListener("scroll", onScroll, { passive: true });
    ro.observe(node);
    ro.observe(scrollParent);

    return {
      destroy() {
        scrollParent.removeEventListener("scroll", onScroll);
        ro.disconnect();
        if (scrollRaf) cancelAnimationFrame(scrollRaf);
        if (anchorEl === node) {
          anchorEl = null;
          cachedScrollParent = null;
        }
      },
    };
  }

  function clampDocumentScroll() {
    if (!cachedScrollParent || !isDocumentScroll) return;
    const maxScroll = Math.max(0, totalHeight - viewportHeight);
    const relative = Math.max(0, cachedScrollParent.scrollTop - listOffsetTop);
    if (relative > maxScroll) {
      cachedScrollParent.scrollTop = listOffsetTop + maxScroll;
      scrollTop = maxScroll;
      return;
    }
    if (scrollTop > maxScroll) {
      scrollTop = maxScroll;
    }
  }

  $effect(() => {
    void items.length;
    void totalHeight;
    void viewportHeight;
    if (isDocumentScroll) {
      clampDocumentScroll();
      return;
    }
    clampInternalScroll();
  });
</script>

{#if isDocumentScroll}
  <div
    class="virtual-list virtual-list--document {className}"
    style:height="{totalHeight}px"
    use:mountDocument
  >
    <div class="virtual-list__window" style:transform="translateY({offsetY}px)">
      {#each visibleItems as item, localIndex (itemKey(item, startIndex + localIndex))}
        {@render children({ item, index: startIndex + localIndex })}
      {/each}
    </div>
  </div>
{:else}
  <div
    class="virtual-list {className}"
    style:max-height={maxHeight === "none" ? undefined : maxHeight}
    style:height={maxHeight === "100%"
      ? "100%"
      : maxHeight !== "none"
        ? maxHeight
        : undefined}
    use:mountInternal
    onscroll={onInternalScroll}
  >
    <div class="virtual-list__spacer" style:height="{totalHeight}px">
      <div
        class="virtual-list__window"
        style:transform="translateY({offsetY}px)"
      >
        {#each visibleItems as item, localIndex (itemKey(item, startIndex + localIndex))}
          {@render children({ item, index: startIndex + localIndex })}
        {/each}
      </div>
    </div>
  </div>
{/if}

<style>
  .virtual-list {
    overflow-y: auto;
    overflow-x: hidden;
    contain: layout style;
  }

  .virtual-list--document {
    position: relative;
    overflow: visible;
    contain: layout style;
  }

  .virtual-list__spacer {
    position: relative;
    width: 100%;
  }

  .virtual-list__window {
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    will-change: transform;
  }

  .virtual-list--document .virtual-list__window {
    width: 100%;
  }
</style>
