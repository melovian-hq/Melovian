<script lang="ts">
  import CoverArt from "$lib/components/ui/CoverArt.svelte";
  import { music } from "$lib/config/music.svelte";
  import { extensionFeatures } from "$lib/extensions/features.svelte";
  import {
    artworkNeedsEnhancement,
    enhanceAlbumArtwork,
    enhanceArtistArtwork,
    enhanceTrackArtwork,
  } from "$lib/music/metadata-enhancement";

  interface Props {
    kind: "artist" | "album" | "track";
    entity: {
      id: string;
      name?: string;
      title?: string;
      artist?: string;
      album?: string;
      coverArt?: string;
      artistImageUrl?: string;
      albumId?: string;
    };
    src?: string | null;
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
  }

  let {
    kind,
    entity,
    src = null,
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
  }: Props = $props();

  let enhancedSrc = $state<string | null>(null);
  let primaryFailed = $state(false);
  let enhancedFailed = $state(false);

  $effect(() => {
    void src;
    primaryFailed = false;
    enhancedSrc = null;
    enhancedFailed = false;
  });

  $effect(() => {
    const settings = music.metadataEnhancementSettings;
    const wantsEnhancement = artworkNeedsEnhancement(src, primaryFailed);

    if (!extensionFeatures.metadata || !settings.enabled || !wantsEnhancement) {
      enhancedSrc = null;
      enhancedFailed = false;
      return;
    }

    if (kind === "artist" && !settings.artists) {
      enhancedSrc = null;
      return;
    }
    if (
      kind === "artist" &&
      settings.preferServerArtistArt &&
      (entity.artistImageUrl?.trim() || entity.coverArt?.trim()) &&
      !primaryFailed
    ) {
      enhancedSrc = null;
      return;
    }
    if (kind === "album" && !settings.albums) {
      enhancedSrc = null;
      return;
    }
    if (kind === "track" && !settings.tracks) {
      enhancedSrc = null;
      return;
    }

    let cancelled = false;
    const options = { resolvedSrc: src, primaryLoadFailed: primaryFailed };
    const resolve =
      kind === "artist"
        ? enhanceArtistArtwork(
            {
              id: entity.id,
              name: entity.name ?? entity.title ?? "",
              coverArt: entity.coverArt,
              artistImageUrl: entity.artistImageUrl,
            },
            settings,
            options,
          )
        : kind === "album"
          ? enhanceAlbumArtwork(
              {
                id: entity.id,
                name: entity.name ?? entity.title ?? "",
                artist: entity.artist,
                coverArt: entity.coverArt,
              },
              settings,
              options,
            )
          : enhanceTrackArtwork(
              {
                id: entity.id,
                title: entity.title ?? entity.name ?? "",
                artist: entity.artist,
                album: entity.album,
                coverArt: entity.coverArt,
                albumId: entity.albumId,
              },
              settings,
              options,
            );

    void resolve.then((url) => {
      if (cancelled) return;
      enhancedSrc = url;
      enhancedFailed = false;
    });

    return () => {
      cancelled = true;
    };
  });

  const preferredSrc = $derived(
    enhancedSrc && !enhancedFailed ? enhancedSrc : src,
  );
  const stablePreview = $derived(
    enhancedSrc && !enhancedFailed && src && src !== enhancedSrc ? src : null,
  );
</script>

<CoverArt
  src={preferredSrc}
  previewSrc={stablePreview}
  {seed}
  {paletteKey}
  {alt}
  class={className}
  {width}
  {height}
  {loading}
  {decoding}
  {fetchpriority}
  {draggable}
  onPrimaryFailed={() => {
    if (enhancedSrc && !enhancedFailed) {
      enhancedFailed = true;
      return;
    }
    primaryFailed = true;
  }}
/>
