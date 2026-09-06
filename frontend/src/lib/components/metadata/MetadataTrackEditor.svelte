<script lang="ts">
  import Field from "$lib/components/ui/Field.svelte";
  import Input from "$lib/components/ui/Input.svelte";
  import Button from "$lib/components/ui/Button.svelte";
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";
  import Spinner from "$lib/components/ui/Spinner.svelte";
  import CoverArt from "$lib/components/ui/CoverArt.svelte";
  import { localCoverArtUrl } from "$lib/local-music/api";
  import {
    metadataLookupSources,
    metadataLookupSourceLabel,
  } from "$lib/features/metadata-editor/api";
  import {
    buildLookupQuery,
    filenameSuggestionToMatch,
    formStateToUpdate,
    hasFilenameSuggestion,
    isMetadataFormDirty,
    issueLabel,
    trackToFormState,
    type MetadataFormState,
  } from "$lib/features/metadata-editor/utils";
  import type {
    MetadataFilenameSuggestion,
    MetadataLookupMatch,
    MetadataTrack,
    MetadataTrackUpdate,
  } from "$lib/features/metadata-editor/types";

  interface Props {
    track: MetadataTrack;
    saving?: boolean;
    lookupLoading?: boolean;
    matches?: MetadataLookupMatch[];
    filenameSuggestion?: MetadataFilenameSuggestion | null;
    hasPrevious?: boolean;
    hasNext?: boolean;
    onsave?: (update: MetadataTrackUpdate) => void | Promise<void>;
    onlookup?: (query: string, source?: string) => void | Promise<void>;
    onapplymatch?: (match: MetadataLookupMatch) => void | Promise<void>;
    onapplyalbum?: (match: MetadataLookupMatch) => void | Promise<void>;
    onprevious?: () => void;
    onnext?: () => void;
    ondirtychange?: (dirty: boolean) => void;
  }

  let {
    track,
    saving = false,
    lookupLoading = false,
    matches = [],
    filenameSuggestion = null,
    hasPrevious = false,
    hasNext = false,
    onsave,
    onlookup,
    onapplymatch,
    onapplyalbum,
    onprevious,
    onnext,
    ondirtychange,
  }: Props = $props();

  let form = $state<MetadataFormState>({
    title: "",
    artist: "",
    album: "",
    albumArtist: "",
    trackNum: "",
    discNum: "",
    year: "",
    genre: "",
  });
  let lookupQuery = $state("");
  let lookupSource = $state("itunes");

  $effect(() => {
    form = trackToFormState(track);
    lookupQuery = buildLookupQuery(track);
  });

  const coverUrl = $derived(localCoverArtUrl(track.coverArt, 240));
  const dirty = $derived(isMetadataFormDirty(track, form));
  const showFilenameSuggestion = $derived(
    filenameSuggestion && hasFilenameSuggestion(filenameSuggestion),
  );

  $effect(() => {
    ondirtychange?.(dirty);
  });

  function handleSave() {
    onsave?.(formStateToUpdate(form));
  }

  function handleRevert() {
    form = trackToFormState(track);
  }

  function handleLookup() {
    onlookup?.(lookupQuery, lookupSource);
  }

  function handleKeydown(event: KeyboardEvent) {
    if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === "s") {
      event.preventDefault();
      if (!saving && dirty) handleSave();
    }
  }
</script>

<svelte:window onkeydown={handleKeydown} />

<div class="editor">
  <div class="editor__toolbar">
    <div class="editor__nav">
      <Button
        size="sm"
        variant="surface"
        disabled={!hasPrevious}
        onclick={() => onprevious?.()}
        aria-label="Previous track"
      >
        <MdiIcon name="chevronLeft" size={18} />
      </Button>
      <Button
        size="sm"
        variant="surface"
        disabled={!hasNext}
        onclick={() => onnext?.()}
        aria-label="Next track"
      >
        <MdiIcon name="chevronRight" size={18} />
      </Button>
    </div>
    {#if dirty}
      <span class="editor__dirty">Unsaved changes</span>
    {/if}
  </div>

  <div class="editor__hero">
    <CoverArt
      src={coverUrl}
      seed={track.id}
      alt={track.title}
      width={160}
      height={160}
    />
    <div class="editor__hero-copy">
      <h2 class="editor__title">{track.title || "Untitled track"}</h2>
      <p class="editor__path">{track.relPath}</p>
      <p class="editor__format">{track.format.toUpperCase()}</p>
      {#if track.issues.length > 0}
        <div class="editor__issues">
          {#each track.issues as issue (issue)}
            <span class="editor__issue">{issueLabel(issue)}</span>
          {/each}
        </div>
      {/if}
    </div>
  </div>

  {#if showFilenameSuggestion && filenameSuggestion}
    <section class="editor__filename" aria-label="Filename suggestion">
      <div class="editor__filename-copy">
        <strong>From filename</strong>
        <span>
          {filenameSuggestion.artist || "Unknown artist"}
          {#if filenameSuggestion.album}
            · {filenameSuggestion.album}
          {/if}
          · {filenameSuggestion.title}
        </span>
      </div>
      <Button
        size="sm"
        variant="surface"
        disabled={saving}
        onclick={() =>
          onapplymatch?.(filenameSuggestionToMatch(filenameSuggestion))}
      >
        Apply
      </Button>
    </section>
  {/if}

  <div class="editor__grid">
    <Field label="Title">
      <Input bind:value={form.title} autocomplete="off" />
    </Field>
    <Field label="Artist">
      <Input bind:value={form.artist} autocomplete="off" />
    </Field>
    <Field label="Album">
      <Input bind:value={form.album} autocomplete="off" />
    </Field>
    <Field label="Album artist">
      <Input bind:value={form.albumArtist} autocomplete="off" />
    </Field>
    <Field label="Track number">
      <Input bind:value={form.trackNum} autocomplete="off" />
    </Field>
    <Field label="Disc number">
      <Input bind:value={form.discNum} autocomplete="off" />
    </Field>
    <Field label="Year">
      <Input bind:value={form.year} autocomplete="off" />
    </Field>
    <Field label="Genre">
      <Input bind:value={form.genre} autocomplete="off" />
    </Field>
  </div>

  <Field label="Lookup search" hint="Used when finding catalog matches.">
    <Input bind:value={lookupQuery} autocomplete="off" />
  </Field>

  <Field label="Lookup provider">
    <select class="settings-input" bind:value={lookupSource}>
      {#each metadataLookupSources as provider (provider.id)}
        <option value={provider.id}>{provider.label}</option>
      {/each}
    </select>
  </Field>

  <div class="editor__actions">
    <Button onclick={handleSave} disabled={saving || !dirty}>
      {#if saving}
        <Spinner />
      {:else}
        <MdiIcon name="contentSave" size={18} />
      {/if}
      Save to file
    </Button>
    <Button
      variant="surface"
      onclick={handleRevert}
      disabled={!dirty || saving}
    >
      <MdiIcon name="refresh" size={18} />
      Revert
    </Button>
    <Button variant="surface" onclick={handleLookup} disabled={lookupLoading}>
      {#if lookupLoading}
        <Spinner />
      {:else}
        <MdiIcon name="autoFix" size={18} />
      {/if}
      Find matches
    </Button>
  </div>

  {#if lookupLoading}
    <p class="editor__lookup-status">
      Searching {metadataLookupSourceLabel(lookupSource)} catalog...
    </p>
  {:else if matches.length > 0}
    <section class="editor__matches" aria-label="Suggested metadata matches">
      <h3 class="editor__matches-title">Suggested matches</h3>
      <div class="editor__match-list">
        {#each matches as match (match.id)}
          <article class="editor__match">
            {#if match.artworkUrl}
              <img
                class="editor__match-art"
                src={match.artworkUrl}
                alt=""
                loading="lazy"
                width="56"
                height="56"
              />
            {/if}
            <div class="editor__match-copy">
              <strong>{match.title}</strong>
              <span>{match.artist}</span>
              <span class="editor__match-album">{match.album}</span>
              {#if match.year > 0 || match.genre}
                <span class="editor__match-meta">
                  {#if match.year > 0}{match.year}{/if}
                  {#if match.year > 0 && match.genre}
                    ·
                  {/if}
                  {match.genre}
                </span>
              {/if}
            </div>
            <div class="editor__match-actions">
              <Button
                size="sm"
                variant="surface"
                disabled={saving}
                onclick={() => onapplymatch?.(match)}
              >
                Apply
              </Button>
              <Button
                size="sm"
                variant="surface"
                disabled={saving}
                onclick={() => onapplyalbum?.(match)}
              >
                Apply to album
              </Button>
            </div>
          </article>
        {/each}
      </div>
    </section>
  {/if}
</div>

<style>
  .editor {
    display: flex;
    flex-direction: column;
    gap: var(--jb-space-5);
  }

  .editor__toolbar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--jb-space-3);
  }

  .editor__nav {
    display: flex;
    gap: var(--jb-space-2);
  }

  .editor__dirty {
    font-size: 0.8125rem;
    font-weight: 600;
    color: var(--jb-warning);
  }

  .editor__hero {
    display: flex;
    gap: var(--jb-space-4);
    align-items: flex-start;
  }

  .editor__hero-copy {
    min-width: 0;
    flex: 1;
  }

  .editor__title {
    margin: 0;
    font-size: 1.25rem;
    line-height: 1.3;
    color: var(--jb-text);
  }

  .editor__path {
    margin: var(--jb-space-2) 0 0;
    font-size: 0.8125rem;
    color: var(--jb-text-subtle);
    word-break: break-all;
  }

  .editor__format {
    margin: var(--jb-space-1) 0 0;
    font-size: 0.75rem;
    font-weight: 700;
    letter-spacing: 0.04em;
    color: var(--jb-text-muted);
  }

  .editor__issues {
    display: flex;
    flex-wrap: wrap;
    gap: var(--jb-space-2);
    margin-top: var(--jb-space-3);
  }

  .editor__issue {
    padding: 0.125rem 0.5rem;
    border-radius: var(--jb-radius-full);
    background: color-mix(in srgb, var(--jb-warning) 18%, transparent);
    color: var(--jb-warning);
    font-size: 0.75rem;
    font-weight: 600;
    text-transform: capitalize;
  }

  .editor__filename {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--jb-space-3);
    padding: var(--jb-space-3);
    border: 1px dashed var(--jb-border);
    border-radius: var(--jb-radius-lg);
    background: color-mix(in srgb, var(--jb-accent) 6%, transparent);
  }

  .editor__filename-copy {
    display: flex;
    flex-direction: column;
    gap: 0.125rem;
    min-width: 0;
  }

  .editor__filename-copy strong {
    font-size: 0.8125rem;
    color: var(--jb-text);
  }

  .editor__filename-copy span {
    font-size: 0.8125rem;
    color: var(--jb-text-muted);
  }

  .editor__grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: var(--jb-space-4);
  }

  .editor__actions {
    display: flex;
    flex-wrap: wrap;
    gap: var(--jb-space-2);
  }

  .editor__actions :global(.btn) {
    gap: var(--jb-space-2);
  }

  .editor__lookup-status {
    margin: 0;
    font-size: 0.875rem;
    color: var(--jb-text-muted);
  }

  .editor__matches-title {
    margin: 0 0 var(--jb-space-3);
    font-size: 0.9375rem;
    font-weight: 700;
    color: var(--jb-text);
  }

  .editor__match-list {
    display: flex;
    flex-direction: column;
    gap: var(--jb-space-2);
  }

  .editor__match {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--jb-space-3);
    padding: var(--jb-space-3);
    border: 1px solid var(--jb-border);
    border-radius: var(--jb-radius-lg);
    background: var(--jb-surface-raised, var(--jb-surface));
  }

  .editor__match-art {
    width: 56px;
    height: 56px;
    border-radius: var(--jb-radius-md);
    object-fit: cover;
    flex-shrink: 0;
  }

  .editor__match-copy {
    display: flex;
    flex-direction: column;
    gap: 0.125rem;
    min-width: 0;
    flex: 1;
  }

  .editor__match-copy strong {
    color: var(--jb-text);
    font-size: 0.9375rem;
  }

  .editor__match-copy span {
    font-size: 0.8125rem;
    color: var(--jb-text-muted);
  }

  .editor__match-album {
    color: var(--jb-text-subtle) !important;
  }

  .editor__match-meta {
    font-size: 0.75rem !important;
    color: var(--jb-text-subtle) !important;
  }

  .editor__match-actions {
    display: flex;
    flex-direction: column;
    gap: var(--jb-space-2);
    flex-shrink: 0;
  }

  @media (max-width: 720px) {
    .editor__grid {
      grid-template-columns: 1fr;
    }

    .editor__hero {
      flex-direction: column;
      align-items: center;
      text-align: center;
    }
  }
</style>
