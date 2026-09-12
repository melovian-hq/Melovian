<!-- SPDX-FileCopyrightText: 2026 Quad4 Software -->
<!-- SPDX-License-Identifier: Apache-2.0 -->
<script lang="ts">
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";
  import SmartPlaylistRuleRow from "./SmartPlaylistRuleRow.svelte";
  import {
    addDraftNestedGroup,
    addDraftRule,
    removeDraftGroup,
    removeDraftRule,
    setDraftGroupLogic,
    updateDraftRule,
  } from "$lib/music/smart-playlist/draft";
  import { rulePath } from "$lib/music/smart-playlist/editor";
  import type {
    SmartPlaylistDraft,
    SmartPlaylistGroup,
  } from "$lib/music/smart-playlist/types";

  interface Props {
    draft: SmartPlaylistDraft;
    group: SmartPlaylistGroup;
    parentId: string | null;
    depth: number;
    errorByPath: Record<string, string>;
    onchange: (draft: SmartPlaylistDraft) => void;
  }

  let { draft, group, parentId, depth, errorByPath, onchange }: Props =
    $props();
</script>

{#snippet groupEditor(
  current: SmartPlaylistGroup,
  currentParentId: string | null,
  currentDepth: number,
)}
  <section
    class="smart-group"
    class:smart-group--nested={currentDepth > 0}
    style:--smart-depth={currentDepth}
  >
    <header class="smart-group__head">
      <div class="smart-group__logic" role="group" aria-label="Match rules">
        <button
          type="button"
          class:smart-group__logic-btn--active={current.logic === "all"}
          onclick={() => onchange(setDraftGroupLogic(draft, current.id, "all"))}
        >
          Match all (AND)
        </button>
        <button
          type="button"
          class:smart-group__logic-btn--active={current.logic === "any"}
          onclick={() => onchange(setDraftGroupLogic(draft, current.id, "any"))}
        >
          Match any (OR)
        </button>
      </div>
      <div class="smart-group__actions">
        <button
          type="button"
          class="smart-group__text-btn"
          onclick={() => onchange(addDraftRule(draft, current.id))}
        >
          <MdiIcon name="plus" size={14} />
          Rule
        </button>
        <button
          type="button"
          class="smart-group__text-btn"
          onclick={() => onchange(addDraftNestedGroup(draft, current.id))}
        >
          <MdiIcon name="plus" size={14} />
          Group
        </button>
        {#if currentParentId}
          <button
            type="button"
            class="smart-group__text-btn smart-group__text-btn--danger"
            onclick={() =>
              onchange(removeDraftGroup(draft, currentParentId, current.id))}
          >
            <MdiIcon name="trash2" size={14} />
            Remove group
          </button>
        {/if}
      </div>
    </header>

    <div class="smart-group__rules">
      {#each current.rules as rule, index (rule.id)}
        {@const path = rulePath(draft.root, current.id, index)}
        <SmartPlaylistRuleRow
          {rule}
          error={errorByPath[path]}
          onpatch={(patch) =>
            onchange(updateDraftRule(draft, current.id, rule.id, patch))}
          onremove={() => onchange(removeDraftRule(draft, current.id, rule.id))}
        />
      {/each}
    </div>

    {#each current.groups as child (child.id)}
      {@render groupEditor(child, current.id, currentDepth + 1)}
    {/each}
  </section>
{/snippet}

{@render groupEditor(group, parentId, depth)}

<style>
  .smart-group {
    display: grid;
    gap: var(--jb-space-3);
    padding: var(--jb-space-3);
    border-radius: var(--jb-radius-lg);
    border: 1px solid var(--jb-border);
    background: var(--jb-surface);
  }

  .smart-group--nested {
    margin-left: calc(var(--smart-depth, 0) * 0.35rem);
    background: color-mix(in srgb, var(--jb-bg-muted) 55%, var(--jb-surface));
  }

  .smart-group__head {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    justify-content: space-between;
    gap: var(--jb-space-2);
  }

  .smart-group__logic {
    display: inline-flex;
    gap: 0.15rem;
    padding: 0.15rem;
    border-radius: var(--jb-radius-full);
    background: var(--jb-bg-muted);
  }

  .smart-group__logic button {
    border: none;
    background: transparent;
    color: var(--jb-text-muted);
    font-size: 0.75rem;
    font-weight: 650;
    padding: 0.35rem 0.7rem;
    border-radius: var(--jb-radius-full);
    cursor: pointer;
  }

  .smart-group__logic-btn--active {
    background: var(--jb-bg-elevated) !important;
    color: var(--jb-music-accent) !important;
    box-shadow: var(--jb-shadow-sm);
  }

  .smart-group__actions {
    display: flex;
    flex-wrap: wrap;
    gap: var(--jb-space-2);
  }

  .smart-group__text-btn {
    display: inline-flex;
    align-items: center;
    gap: 0.2rem;
    border: none;
    background: transparent;
    color: var(--jb-text-muted);
    font-size: 0.8125rem;
    font-weight: 600;
    cursor: pointer;
    padding: 0.2rem 0.35rem;
  }

  .smart-group__text-btn:hover {
    color: var(--jb-text);
  }

  .smart-group__text-btn--danger:hover {
    color: var(--jb-danger);
  }

  .smart-group__rules {
    display: grid;
    gap: var(--jb-space-2);
  }
</style>
