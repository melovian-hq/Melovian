<script lang="ts">
  import { Dialog, Select } from "bits-ui";
  import Button from "$lib/components/ui/Button.svelte";
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";
  import Spinner from "$lib/components/ui/Spinner.svelte";
  import {
    SMART_PLAYLIST_FIELDS,
    SMART_PLAYLIST_OPERATORS,
    SMART_PLAYLIST_SORT_OPTIONS,
    defaultOperatorForField,
    getSmartField,
    operatorsForField,
  } from "$lib/music/smart-playlist/fields";
  import {
    addDraftNestedGroup,
    addDraftRule,
    removeDraftGroup,
    removeDraftRule,
    setDraftGroupLogic,
    updateDraftRule,
  } from "$lib/music/smart-playlist/draft";
  import {
    coerceRuleValue,
    createEmptySmartPlaylistDraft,
  } from "$lib/music/smart-playlist/compile";
  import { validateSmartPlaylistDraft } from "$lib/music/smart-playlist/validate";
  import type {
    SmartPlaylistDraft,
    SmartPlaylistGroup,
    SmartPlaylistRule,
    SmartRuleValue,
  } from "$lib/music/smart-playlist/types";

  interface Props {
    open?: boolean;
    onclose?: () => void;
    oncreate?: (draft: SmartPlaylistDraft) => Promise<void>;
  }

  let { open = false, onclose, oncreate }: Props = $props();

  const FIELD_ITEMS = SMART_PLAYLIST_FIELDS.map((field) => ({
    value: field.id,
    label: field.label,
  }));
  const SORT_ITEMS = SMART_PLAYLIST_SORT_OPTIONS.map((option) => ({
    value: option.id as string,
    label: option.label,
  }));
  const PRESENCE_ITEMS = [
    { value: "present", label: "Present" },
    { value: "missing", label: "Missing" },
  ];
  const BOOLEAN_ITEMS = [
    { value: "true", label: "Yes" },
    { value: "false", label: "No" },
  ];

  let draft = $state(createEmptySmartPlaylistDraft());
  let submitting = $state(false);
  let submitError = $state("");

  const validationErrors = $derived(validateSmartPlaylistDraft(draft));
  const errorByPath = $derived(
    Object.fromEntries(
      validationErrors.map((error) => [error.path, error.message]),
    ),
  );

  function resetDraft() {
    draft = createEmptySmartPlaylistDraft();
    submitError = "";
  }

  function closeDialog() {
    resetDraft();
    onclose?.();
  }

  function rulePath(groupId: string, ruleId: string, index: number): string {
    const groupPath = groupPathFor(groupId);
    return `${groupPath}.rules[${index}]`;
  }

  function groupPathFor(groupId: string): string {
    if (draft.root.id === groupId) return "root";
    return findGroupPath(draft.root, groupId, "root") ?? "root";
  }

  function findGroupPath(
    group: SmartPlaylistGroup,
    targetId: string,
    prefix: string,
  ): string | null {
    for (let index = 0; index < group.groups.length; index += 1) {
      const child = group.groups[index]!;
      const childPath = `${prefix}.groups[${index}]`;
      if (child.id === targetId) return childPath;
      const nested = findGroupPath(child, targetId, childPath);
      if (nested) return nested;
    }
    return null;
  }

  function onFieldChange(
    groupId: string,
    rule: SmartPlaylistRule,
    fieldId: string,
  ) {
    const operator = defaultOperatorForField(fieldId);
    draft = updateDraftRule(draft, groupId, rule.id, {
      field: fieldId,
      operator,
      value: defaultValueFor(fieldId, operator),
    });
  }

  function onOperatorChange(
    groupId: string,
    rule: SmartPlaylistRule,
    operator: string,
  ) {
    draft = updateDraftRule(draft, groupId, rule.id, {
      operator,
      value: defaultValueFor(rule.field, operator),
    });
  }

  function defaultValueFor(fieldId: string, operator: string): SmartRuleValue {
    const field = getSmartField(fieldId);
    if (operator === "isMissing" || operator === "isPresent") return true;
    if (operator === "inTheRange") {
      return field?.valueType === "number" ? [0, 0] : ["", ""];
    }
    if (field?.valueType === "boolean") return true;
    if (field?.valueType === "days" || field?.valueType === "number") return 0;
    return "";
  }

  function valueInputType(rule: SmartPlaylistRule): string {
    const field = getSmartField(rule.field);
    if (!field) return "text";
    if (field.valueType === "number" || field.valueType === "days")
      return "number";
    if (field.valueType === "date") return "date";
    return "text";
  }

  function operatorLabel(operator: string): string {
    return SMART_PLAYLIST_OPERATORS[operator]?.label ?? operator;
  }

  function rangeValues(rule: SmartPlaylistRule): [string, string] {
    if (!Array.isArray(rule.value)) return ["", ""];
    return [String(rule.value[0] ?? ""), String(rule.value[1] ?? "")];
  }

  function setRangeValue(
    groupId: string,
    rule: SmartPlaylistRule,
    index: 0 | 1,
    raw: string,
  ) {
    const field = getSmartField(rule.field);
    const current = rangeValues(rule);
    current[index] = raw;
    const value: SmartRuleValue =
      field?.valueType === "range-number"
        ? [Number(current[0]), Number(current[1])]
        : current;
    draft = updateDraftRule(draft, groupId, rule.id, { value });
  }

  async function submit() {
    submitError = "";
    const errors = validateSmartPlaylistDraft(draft);
    if (errors.length > 0) {
      submitError = errors[0]?.message ?? "Fix the highlighted fields";
      return;
    }
    submitting = true;
    try {
      await oncreate?.(draft);
      closeDialog();
    } catch (err) {
      submitError =
        err instanceof Error ? err.message : "Failed to create smart playlist";
    } finally {
      submitting = false;
    }
  }
</script>

{#snippet groupEditor(
  group: SmartPlaylistGroup,
  parentId: string | null,
  depth: number,
)}
  <section
    class="smart-group"
    class:smart-group--nested={depth > 0}
    style:--smart-depth={depth}
  >
    <header class="smart-group__head">
      <div class="smart-group__logic" role="group" aria-label="Match rules">
        <button
          type="button"
          class:smart-group__logic-btn--active={group.logic === "all"}
          onclick={() => (draft = setDraftGroupLogic(draft, group.id, "all"))}
        >
          Match all (AND)
        </button>
        <button
          type="button"
          class:smart-group__logic-btn--active={group.logic === "any"}
          onclick={() => (draft = setDraftGroupLogic(draft, group.id, "any"))}
        >
          Match any (OR)
        </button>
      </div>
      <div class="smart-group__actions">
        <button
          type="button"
          class="smart-group__text-btn"
          onclick={() => (draft = addDraftRule(draft, group.id))}
        >
          <MdiIcon name="plus" size={14} />
          Rule
        </button>
        <button
          type="button"
          class="smart-group__text-btn"
          onclick={() => (draft = addDraftNestedGroup(draft, group.id))}
        >
          <MdiIcon name="plus" size={14} />
          Group
        </button>
        {#if parentId}
          <button
            type="button"
            class="smart-group__text-btn smart-group__text-btn--danger"
            onclick={() =>
              (draft = removeDraftGroup(draft, parentId, group.id))}
          >
            <MdiIcon name="trash2" size={14} />
            Remove group
          </button>
        {/if}
      </div>
    </header>

    <div class="smart-group__rules">
      {#each group.rules as rule, index (rule.id)}
        {@const path = rulePath(group.id, rule.id, index)}
        {@const field = getSmartField(rule.field)}
        {@const operatorItems = operatorsForField(rule.field).map(
          (operator) => ({
            value: operator,
            label: operatorLabel(operator),
          }),
        )}
        <div
          class="smart-rule"
          class:smart-rule--invalid={Boolean(errorByPath[path])}
        >
          <Select.Root
            type="single"
            items={FIELD_ITEMS}
            value={rule.field}
            onValueChange={(value) => onFieldChange(group.id, rule, value)}
          >
            <Select.Trigger
              class="smart-rule__field smart-select-trigger"
              aria-label="Rule field"
            >
              <Select.Value />
              <MdiIcon name="chevronDown" size={14} />
            </Select.Trigger>
            <Select.Portal>
              <Select.Content
                class="smart-creator__select-content"
                sideOffset={4}
              >
                <Select.Viewport>
                  {#each FIELD_ITEMS as option (option.value)}
                    <Select.Item
                      class="smart-creator__select-item"
                      value={option.value}
                      label={option.label}
                    >
                      {#snippet children({ selected })}
                        {option.label}
                        {#if selected}
                          <MdiIcon name="check" size={14} />
                        {/if}
                      {/snippet}
                    </Select.Item>
                  {/each}
                </Select.Viewport>
              </Select.Content>
            </Select.Portal>
          </Select.Root>

          <Select.Root
            type="single"
            items={operatorItems}
            value={rule.operator}
            onValueChange={(value) => onOperatorChange(group.id, rule, value)}
          >
            <Select.Trigger
              class="smart-rule__operator smart-select-trigger"
              aria-label="Rule operator"
            >
              <Select.Value />
              <MdiIcon name="chevronDown" size={14} />
            </Select.Trigger>
            <Select.Portal>
              <Select.Content
                class="smart-creator__select-content"
                sideOffset={4}
              >
                <Select.Viewport>
                  {#each operatorItems as option (option.value)}
                    <Select.Item
                      class="smart-creator__select-item"
                      value={option.value}
                      label={option.label}
                    >
                      {#snippet children({ selected })}
                        {option.label}
                        {#if selected}
                          <MdiIcon name="check" size={14} />
                        {/if}
                      {/snippet}
                    </Select.Item>
                  {/each}
                </Select.Viewport>
              </Select.Content>
            </Select.Portal>
          </Select.Root>

          {#if rule.operator === "inTheRange"}
            {@const range = rangeValues(rule)}
            <div class="smart-rule__range">
              <input
                type={valueInputType(rule)}
                value={range[0]}
                placeholder="From"
                oninput={(event) =>
                  setRangeValue(
                    group.id,
                    rule,
                    0,
                    (event.currentTarget as HTMLInputElement).value,
                  )}
              />
              <span>to</span>
              <input
                type={valueInputType(rule)}
                value={range[1]}
                placeholder="To"
                oninput={(event) =>
                  setRangeValue(
                    group.id,
                    rule,
                    1,
                    (event.currentTarget as HTMLInputElement).value,
                  )}
              />
            </div>
          {:else if rule.operator === "isMissing" || rule.operator === "isPresent"}
            <Select.Root
              type="single"
              items={PRESENCE_ITEMS}
              value={rule.value === true ? "present" : "missing"}
              onValueChange={(value) =>
                (draft = updateDraftRule(draft, group.id, rule.id, {
                  value: value === "present",
                }))}
            >
              <Select.Trigger
                class="smart-rule__value smart-select-trigger"
                aria-label="Rule value"
              >
                <Select.Value />
                <MdiIcon name="chevronDown" size={14} />
              </Select.Trigger>
              <Select.Portal>
                <Select.Content
                  class="smart-creator__select-content"
                  sideOffset={4}
                >
                  <Select.Viewport>
                    {#each PRESENCE_ITEMS as option (option.value)}
                      <Select.Item
                        class="smart-creator__select-item"
                        value={option.value}
                        label={option.label}
                      >
                        {#snippet children({ selected })}
                          {option.label}
                          {#if selected}
                            <MdiIcon name="check" size={14} />
                          {/if}
                        {/snippet}
                      </Select.Item>
                    {/each}
                  </Select.Viewport>
                </Select.Content>
              </Select.Portal>
            </Select.Root>
          {:else if field?.valueType === "boolean"}
            <Select.Root
              type="single"
              items={BOOLEAN_ITEMS}
              value={rule.value === true ? "true" : "false"}
              onValueChange={(value) =>
                (draft = updateDraftRule(draft, group.id, rule.id, {
                  value: value === "true",
                }))}
            >
              <Select.Trigger
                class="smart-rule__value smart-select-trigger"
                aria-label="Rule value"
              >
                <Select.Value />
                <MdiIcon name="chevronDown" size={14} />
              </Select.Trigger>
              <Select.Portal>
                <Select.Content
                  class="smart-creator__select-content"
                  sideOffset={4}
                >
                  <Select.Viewport>
                    {#each BOOLEAN_ITEMS as option (option.value)}
                      <Select.Item
                        class="smart-creator__select-item"
                        value={option.value}
                        label={option.label}
                      >
                        {#snippet children({ selected })}
                          {option.label}
                          {#if selected}
                            <MdiIcon name="check" size={14} />
                          {/if}
                        {/snippet}
                      </Select.Item>
                    {/each}
                  </Select.Viewport>
                </Select.Content>
              </Select.Portal>
            </Select.Root>
          {:else}
            <input
              class="smart-rule__value"
              type={valueInputType(rule)}
              value={String(rule.value ?? "")}
              placeholder={rule.field === "title" &&
              rule.operator === "contains"
                ? "Text or /regex/"
                : "Value"}
              oninput={(event) =>
                (draft = updateDraftRule(draft, group.id, rule.id, {
                  value: coerceRuleValue(
                    (event.currentTarget as HTMLInputElement).value,
                    rule.field,
                    rule.operator,
                  ),
                }))}
            />
          {/if}

          <button
            type="button"
            class="smart-rule__remove"
            aria-label="Remove rule"
            onclick={() => (draft = removeDraftRule(draft, group.id, rule.id))}
          >
            <MdiIcon name="trash2" size={16} />
          </button>

          {#if errorByPath[path]}
            <p class="smart-rule__error">{errorByPath[path]}</p>
          {/if}
        </div>
      {/each}
    </div>

    {#each group.groups as child (child.id)}
      {@render groupEditor(child, group.id, depth + 1)}
    {/each}
  </section>
{/snippet}

<Dialog.Root
  {open}
  onOpenChange={(next) => {
    if (!next) closeDialog();
  }}
>
  <Dialog.Portal>
    <Dialog.Overlay class="smart-creator__backdrop" />
    <Dialog.Content class="smart-creator">
      <header class="smart-creator__header">
        <div>
          <Dialog.Title id="smart-creator-title">
            {#snippet child({ props })}
              <h2 {...props}>Smart playlist</h2>
            {/snippet}
          </Dialog.Title>
          <Dialog.Description>
            {#snippet child({ props })}
              <p {...props}>
                Build dynamic playlists with AND/OR rules from your library.
              </p>
            {/snippet}
          </Dialog.Description>
        </div>
        <Dialog.Close
          type="button"
          class="smart-creator__close"
          aria-label="Close"
        >
          <MdiIcon name="x" size={18} />
        </Dialog.Close>
      </header>

      <div class="smart-creator__body">
        <div class="smart-creator__grid">
          <label class="smart-field">
            <span>Name</span>
            <input
              bind:value={draft.name}
              placeholder="Evening jazz"
              autocomplete="off"
            />
            {#if errorByPath.name}
              <span class="smart-field__error">{errorByPath.name}</span>
            {/if}
          </label>

          <label class="smart-field">
            <span>Sort</span>
            <Select.Root
              type="single"
              items={SORT_ITEMS}
              bind:value={draft.sort}
            >
              <Select.Trigger
                class="smart-field__select smart-select-trigger"
                aria-label="Sort"
              >
                <Select.Value />
                <MdiIcon name="chevronDown" size={14} />
              </Select.Trigger>
              <Select.Portal>
                <Select.Content
                  class="smart-creator__select-content"
                  sideOffset={4}
                >
                  <Select.Viewport>
                    {#each SORT_ITEMS as option (option.value)}
                      <Select.Item
                        class="smart-creator__select-item"
                        value={option.value}
                        label={option.label}
                      >
                        {#snippet children({ selected })}
                          {option.label}
                          {#if selected}
                            <MdiIcon name="check" size={14} />
                          {/if}
                        {/snippet}
                      </Select.Item>
                    {/each}
                  </Select.Viewport>
                </Select.Content>
              </Select.Portal>
            </Select.Root>
          </label>

          <label class="smart-field">
            <span>Track limit</span>
            <input
              type="number"
              min="1"
              value={draft.limit ?? ""}
              oninput={(event) => {
                const raw = (event.currentTarget as HTMLInputElement).value;
                draft.limit = raw === "" ? null : Number(raw);
                if (draft.limit !== null) draft.limitPercent = null;
              }}
            />
            {#if errorByPath.limit}
              <span class="smart-field__error">{errorByPath.limit}</span>
            {/if}
          </label>

          <label class="smart-field smart-field--checkbox">
            <input type="checkbox" bind:checked={draft.public} />
            <span>Share publicly on server</span>
          </label>
        </div>

        <label class="smart-field">
          <span>Comment</span>
          <input
            bind:value={draft.comment}
            placeholder="Optional description"
            autocomplete="off"
          />
        </label>

        <div class="smart-creator__rules">
          <h3>Rules</h3>
          {@render groupEditor(draft.root, null, 0)}
        </div>

        {#if submitError}
          <p class="smart-creator__submit-error">{submitError}</p>
        {/if}
      </div>

      <footer class="smart-creator__footer">
        <Button variant="surface" onclick={closeDialog} disabled={submitting}>
          Cancel
        </Button>
        <Button onclick={() => void submit()} disabled={submitting}>
          {#if submitting}
            <Spinner />
            Creating...
          {:else}
            <MdiIcon name="plus" size={16} />
            Create smart playlist
          {/if}
        </Button>
      </footer>
    </Dialog.Content>
  </Dialog.Portal>
</Dialog.Root>

<style>
  :global(.smart-creator__backdrop) {
    position: fixed;
    inset: 0;
    border: none;
    background: rgb(0 0 0 / 0.5);
    z-index: 80;
    cursor: pointer;
  }

  :global(.smart-creator) {
    position: fixed;
    top: 50%;
    left: 50%;
    transform: translate(-50%, -50%);
    z-index: 81;
    width: min(44rem, calc(100vw - 1.5rem));
    max-height: min(44rem, 92vh);
    display: flex;
    flex-direction: column;
    border-radius: var(--jb-radius-xl);
    background: var(--jb-bg-elevated);
    border: 1px solid var(--jb-border);
    box-shadow: var(--jb-shadow-lg);
    overflow: hidden;
  }

  .smart-creator__header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: var(--jb-space-4);
    padding: var(--jb-space-5);
    border-bottom: 1px solid var(--jb-border);
  }

  .smart-creator__header h2 {
    margin: 0 0 0.25rem;
    font-size: 1.25rem;
  }

  .smart-creator__header p {
    margin: 0;
    color: var(--jb-text-muted);
    font-size: 0.875rem;
  }

  :global(.smart-creator__close) {
    border: none;
    background: transparent;
    color: var(--jb-text-muted);
    cursor: pointer;
    padding: 0.35rem;
    border-radius: var(--jb-radius-md);
  }

  :global(.smart-creator__close:hover) {
    color: var(--jb-text);
    background: var(--jb-surface-hover);
  }

  .smart-creator__body {
    padding: var(--jb-space-5);
    overflow: auto;
    display: grid;
    gap: var(--jb-space-4);
  }

  .smart-creator__grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: var(--jb-space-3);
  }

  .smart-field {
    display: grid;
    gap: 0.35rem;
    font-size: 0.8125rem;
    color: var(--jb-text-muted);
  }

  .smart-field input,
  :global(.smart-field__select) {
    width: 100%;
    padding: 0.55rem 0.7rem;
    border-radius: var(--jb-radius-md);
    border: 1px solid var(--jb-border);
    background: var(--jb-bg);
    color: var(--jb-text);
    font: inherit;
  }

  .smart-field--checkbox {
    display: flex;
    align-items: center;
    gap: var(--jb-space-2);
    align-self: end;
    color: var(--jb-text);
  }

  .smart-field--checkbox input {
    width: auto;
  }

  .smart-field__error,
  .smart-rule__error,
  .smart-creator__submit-error {
    margin: 0;
    color: var(--jb-danger);
    font-size: 0.75rem;
  }

  .smart-creator__rules h3 {
    margin: 0 0 var(--jb-space-3);
    font-size: 0.9375rem;
  }

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

  .smart-rule {
    display: grid;
    grid-template-columns:
      minmax(7rem, 1fr) minmax(7rem, 1fr) minmax(8rem, 1.4fr)
      auto;
    gap: var(--jb-space-2);
    align-items: start;
    padding: var(--jb-space-2);
    border-radius: var(--jb-radius-md);
    background: var(--jb-bg);
    border: 1px solid transparent;
  }

  .smart-rule--invalid {
    border-color: color-mix(in srgb, var(--jb-danger) 35%, transparent);
  }

  :global(.smart-rule__field),
  :global(.smart-rule__operator),
  :global(.smart-rule__value),
  .smart-rule__range input {
    width: 100%;
    padding: 0.45rem 0.55rem;
    border-radius: var(--jb-radius-md);
    border: 1px solid var(--jb-border);
    background: var(--jb-bg-elevated);
    color: var(--jb-text);
    font: inherit;
    font-size: 0.8125rem;
  }

  :global(.smart-select-trigger) {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 0.35rem;
    text-align: left;
    cursor: pointer;
  }

  :global(.smart-select-trigger svg) {
    flex-shrink: 0;
    color: var(--jb-text-subtle);
  }

  :global(.smart-creator__select-content) {
    z-index: 100;
    min-width: var(--bits-select-anchor-width);
    padding: var(--jb-space-1);
    border: 1px solid var(--jb-border);
    border-radius: var(--jb-radius-md);
    background: var(--jb-bg-elevated);
    box-shadow: var(--jb-shadow-lg);
    display: flex;
    flex-direction: column;
    gap: 0.125rem;
  }

  :global(.smart-creator__select-item) {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--jb-space-2);
    width: 100%;
    padding: 0.45rem 0.55rem;
    border-radius: var(--jb-radius-sm);
    color: var(--jb-text);
    font-size: 0.8125rem;
    cursor: pointer;
    user-select: none;
  }

  :global(.smart-creator__select-item[data-highlighted]) {
    background: var(--jb-surface-hover);
  }

  :global(.smart-creator__select-item[data-disabled]) {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .smart-rule__range {
    display: grid;
    grid-template-columns: 1fr auto 1fr;
    gap: 0.35rem;
    align-items: center;
    font-size: 0.75rem;
    color: var(--jb-text-muted);
  }

  .smart-rule__error {
    grid-column: 1 / -1;
  }

  .smart-rule__remove {
    border: none;
    background: transparent;
    color: var(--jb-text-subtle);
    cursor: pointer;
    padding: 0.35rem;
    border-radius: var(--jb-radius-md);
  }

  .smart-rule__remove:hover {
    color: var(--jb-danger);
  }

  .smart-creator__footer {
    display: flex;
    justify-content: flex-end;
    gap: var(--jb-space-2);
    padding: var(--jb-space-4) var(--jb-space-5);
    border-top: 1px solid var(--jb-border);
    background: color-mix(in srgb, var(--jb-bg-muted) 35%, transparent);
  }

  @media (max-width: 720px) {
    .smart-creator__grid {
      grid-template-columns: 1fr;
    }

    .smart-rule {
      grid-template-columns: 1fr;
    }
  }

  @media (max-width: 480px) {
    :global(.smart-creator) {
      width: calc(100vw - 0.75rem);
      max-height: 94dvh;
      border-radius: var(--jb-radius-lg);
    }

    .smart-creator__header,
    .smart-creator__body,
    .smart-creator__footer {
      padding-inline: var(--jb-space-4);
    }

    .smart-creator__footer {
      flex-wrap: wrap;
    }

    .smart-creator__footer :global(.button) {
      flex: 1;
      min-width: 8rem;
    }
  }
</style>
