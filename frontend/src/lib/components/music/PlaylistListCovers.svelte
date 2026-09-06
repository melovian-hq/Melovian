<script lang="ts">
  import PlaylistCoverStack from "./PlaylistCoverStack.svelte";
  import {
    enrichServerPlaylistCovers,
    getCachedServerPlaylistCovers,
    loadServerPlaylistCoverIds,
  } from "$lib/music/playlist-covers";

  interface Props {
    kind: "local" | "server";
    playlistId: string;
    seed: string;
    coverArtIds?: string[];
    coverArt?: string;
    variant?: "compact" | "grid" | "card";
    enrichCovers?: boolean;
  }

  let {
    kind,
    playlistId,
    seed,
    coverArtIds = [],
    coverArt,
    variant = "compact",
    enrichCovers = false,
  }: Props = $props();

  let resolvedCovers = $state<string[]>([]);

  $effect(() => {
    if (kind === "local") {
      resolvedCovers = coverArtIds.filter(Boolean).slice(0, 3);
      return;
    }

    const cached = getCachedServerPlaylistCovers(playlistId);
    if (cached?.length) {
      resolvedCovers = cached;
      if (enrichCovers && cached.length < 3) {
        let cancelled = false;
        void enrichServerPlaylistCovers(playlistId, cached).then((ids) => {
          if (!cancelled) resolvedCovers = ids;
        });
        return () => {
          cancelled = true;
        };
      }
      return;
    }

    const fallback = coverArt?.trim();
    if (fallback) {
      resolvedCovers = [fallback];
      if (enrichCovers) {
        let cancelled = false;
        void enrichServerPlaylistCovers(playlistId, [fallback]).then((ids) => {
          if (!cancelled) resolvedCovers = ids;
        });
        return () => {
          cancelled = true;
        };
      }
      return;
    }

    resolvedCovers = [];
    let cancelled = false;
    void loadServerPlaylistCoverIds(playlistId).then((ids) => {
      if (!cancelled) resolvedCovers = ids;
    });
    return () => {
      cancelled = true;
    };
  });
</script>

<PlaylistCoverStack coverArtIds={resolvedCovers} {seed} {variant} />
