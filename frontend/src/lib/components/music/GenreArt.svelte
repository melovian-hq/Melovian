<script lang="ts">
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";
  import { genreArtUrl, genreIcon } from "$lib/music/genre-art";

  interface Props {
    name: string;
    class?: string;
  }

  let { name, class: className = "" }: Props = $props();

  const src = $derived(genreArtUrl(name));
  const icon = $derived(genreIcon(name));
</script>

<div class="genre-art {className}" aria-hidden="true">
  <img
    class="genre-art__image"
    {src}
    alt=""
    width="44"
    height="44"
    decoding="async"
    loading="lazy"
    draggable="false"
  />
  <MdiIcon name={icon} size={20} class="genre-art__icon" aria-hidden={true} />
</div>

<style>
  .genre-art {
    position: relative;
    width: 2.75rem;
    height: 2.75rem;
    border-radius: var(--jb-radius-sm);
    overflow: hidden;
    flex-shrink: 0;
    box-shadow: inset 0 0 0 1px rgb(255 255 255 / 0.08);
    contain: strict;
  }

  .genre-art__image {
    display: block;
    width: 100%;
    height: 100%;
    object-fit: cover;
  }

  .genre-art :global(.genre-art__icon) {
    position: absolute;
    inset: 0;
    margin: auto;
    color: rgb(255 255 255 / 0.9);
    filter: drop-shadow(0 1px 3px rgb(0 0 0 / 0.6));
    pointer-events: none;
  }
</style>
