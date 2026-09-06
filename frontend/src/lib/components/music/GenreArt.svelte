<script lang="ts">
  import { genreArtUrl, genreInitial } from "$lib/music/genre-art";

  interface Props {
    name: string;
    class?: string;
  }

  let { name, class: className = "" }: Props = $props();

  const src = $derived(genreArtUrl(name));
  const initial = $derived(genreInitial(name));
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
  <span class="genre-art__initial">{initial}</span>
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
    image-rendering: pixelated;
  }

  .genre-art__initial {
    position: absolute;
    inset: 0;
    display: grid;
    place-content: center;
    font-size: 0.875rem;
    font-weight: 800;
    color: rgb(255 255 255 / 0.85);
    text-shadow: 0 1px 3px rgb(0 0 0 / 0.6);
    pointer-events: none;
  }
</style>
