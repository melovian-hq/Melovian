<script lang="ts">
  import { coverArtFallbackUrl } from "$lib/music/cover-art-fallback";
  import { isCoverArtPrefetched } from "$lib/music/cover-art-prefetch";

  interface Props {
    src?: string | null;
    previewSrc?: string | null;
    seed: string;
    paletteKey?: string;
    alt?: string;
    class?: string;
    width?: number | string;
    height?: number | string;
    loading?: "lazy" | "eager";
    decoding?: "async" | "sync" | "auto";
    fetchpriority?: "high" | "low" | "auto";
    draggable?: boolean;
    onPrimaryFailed?: () => void;
  }

  let {
    src = null,
    previewSrc = null,
    seed,
    paletteKey = seed,
    alt = "",
    class: className = "",
    width,
    height,
    loading = "lazy",
    decoding = "async",
    fetchpriority,
    draggable = false,
    onPrimaryFailed,
  }: Props = $props();

  let failed = $state(false);
  let fullReady = $state(false);
  let preferPreview = $state(false);

  const fallback = $derived(coverArtFallbackUrl(seed, paletteKey));
  const hasPreview = $derived(Boolean(previewSrc && previewSrc !== src));
  const usePreview = $derived(
    Boolean(hasPreview && src && !failed && (preferPreview || !fullReady)),
  );
  const displaySrc = $derived(
    usePreview
      ? previewSrc
      : src && !failed
        ? src
        : hasPreview && preferPreview
          ? previewSrc
          : fallback,
  );
  const pixelated = $derived(!displaySrc || displaySrc === fallback);

  $effect(() => {
    void src;
    void previewSrc;
    failed = false;
    fullReady = false;
    preferPreview = false;

    if (!src || !previewSrc || previewSrc === src) {
      fullReady = true;
      return;
    }

    if (isCoverArtPrefetched(src)) {
      fullReady = true;
      return;
    }

    const img = new Image();
    img.decoding = "async";
    if (fetchpriority) {
      img.fetchPriority = fetchpriority;
    }
    img.onload = () => {
      fullReady = true;
      preferPreview = false;
    };
    img.onerror = () => {
      // Keep the working preview visible. Never jump straight to pixel art.
      preferPreview = true;
      fullReady = false;
      onPrimaryFailed?.();
    };
    img.src = src;

    return () => {
      img.onload = null;
      img.onerror = null;
      img.src = "";
    };
  });

  function handleError() {
    if (hasPreview && displaySrc === src) {
      preferPreview = true;
      fullReady = false;
      onPrimaryFailed?.();
      return;
    }
    if (hasPreview && displaySrc === previewSrc) {
      failed = true;
      return;
    }
    if (!failed) {
      failed = true;
      onPrimaryFailed?.();
    }
  }
</script>

<img
  src={displaySrc}
  {alt}
  {width}
  {height}
  {loading}
  {decoding}
  {draggable}
  {fetchpriority}
  class="cover-art {className}"
  class:cover-art--pixelated={pixelated}
  onerror={handleError}
/>

<style>
  .cover-art {
    display: block;
    object-fit: cover;
    max-width: 100%;
  }

  .cover-art--pixelated {
    image-rendering: pixelated;
  }
</style>
