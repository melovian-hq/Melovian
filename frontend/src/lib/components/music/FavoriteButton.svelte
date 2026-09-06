<script lang="ts">
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";
  import { music } from "$lib/config/music.svelte";
  import { canMutateInDemo } from "$lib/config/demo-guards";
  import type { SubsonicSong } from "$lib/subsonic";

  interface Props {
    track: SubsonicSong | null;
    size?: number;
    class?: string;
  }

  let { track, size = 16, class: className = "" }: Props = $props();

  const favorited = $derived(track ? music.isFavorite(track.id) : false);
  let busy = $state(false);

  async function toggle(event: MouseEvent) {
    event.stopPropagation();
    event.preventDefault();
    if (!track || busy || !canMutateInDemo()) return;
    busy = true;
    try {
      await music.toggleFavorite(track);
    } finally {
      busy = false;
    }
  }
</script>

<button
  type="button"
  class="favorite-btn {className}"
  class:favorite-btn--active={favorited}
  disabled={!track || busy || !canMutateInDemo()}
  aria-label={favorited ? "Remove from favorites" : "Add to favorites"}
  title={favorited ? "Remove from favorites" : "Add to favorites"}
  onclick={toggle}
>
  <MdiIcon name="star" {size} fill={favorited ? "currentColor" : "none"} />
</button>

<style>
  .favorite-btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    min-width: 2.5rem;
    min-height: 2.5rem;
    border: none;
    border-radius: var(--jb-radius-full);
    background: transparent;
    color: var(--jb-text-subtle);
    cursor: pointer;
    padding: 0;
  }

  .favorite-btn:hover:not(:disabled) {
    color: var(--jb-text);
  }

  .favorite-btn--active {
    color: #fbbf24;
  }

  .favorite-btn:disabled {
    opacity: 0.45;
    cursor: default;
  }
</style>
