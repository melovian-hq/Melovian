<script lang="ts">
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";
  import { music } from "$lib/config/music.svelte";
  import type { SubsonicAlbum, SubsonicArtist } from "$lib/subsonic";

  interface Props {
    kind: "album" | "artist";
    album?: SubsonicAlbum;
    artist?: SubsonicArtist;
    size?: number;
    class?: string;
    onDark?: boolean;
  }

  let {
    kind,
    album,
    artist,
    size = 18,
    class: className = "",
    onDark = false,
  }: Props = $props();

  const itemId = $derived(kind === "album" ? album?.id : artist?.id);
  const starred = $derived(
    itemId
      ? kind === "album"
        ? music.isFavoriteAlbum(itemId)
        : music.isFavoriteArtist(itemId)
      : false,
  );

  let busy = $state(false);

  async function toggle(event: MouseEvent) {
    event.stopPropagation();
    event.preventDefault();
    if (busy || !music.connected) return;
    busy = true;
    try {
      if (kind === "album" && album) {
        await music.toggleFavoriteAlbum(album);
      } else if (kind === "artist" && artist) {
        await music.toggleFavoriteArtist(artist);
      }
    } finally {
      busy = false;
    }
  }
</script>

<button
  type="button"
  class="star-btn {className}"
  class:star-btn--active={starred}
  class:star-btn--on-dark={onDark}
  disabled={!itemId || busy || !music.connected}
  aria-label={starred ? "Remove from favorites" : "Add to favorites"}
  title={starred ? "Remove from favorites" : "Add to favorites"}
  onclick={toggle}
>
  <MdiIcon name="star" {size} fill={starred ? "currentColor" : "none"} />
</button>

<style>
  .star-btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    border: none;
    border-radius: var(--jb-radius-full);
    background: transparent;
    color: var(--jb-text-subtle);
    cursor: pointer;
    padding: 0;
  }

  .star-btn:hover:not(:disabled) {
    color: var(--jb-text);
  }

  .star-btn--active {
    color: #fbbf24;
  }

  .star-btn--on-dark.star-btn--active {
    color: #fbbf24;
  }

  .star-btn--on-dark.star-btn--active:hover:not(:disabled) {
    color: #fcd34d;
  }

  .star-btn--on-dark {
    color: rgb(255 255 255 / 0.75);
  }

  .star-btn--on-dark:hover:not(:disabled) {
    color: white;
  }

  .star-btn:disabled {
    opacity: 0.45;
    cursor: default;
  }
</style>
