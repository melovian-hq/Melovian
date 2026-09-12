<!-- SPDX-FileCopyrightText: 2026 Quad4 Software -->
<!-- SPDX-License-Identifier: Apache-2.0 -->
<script lang="ts">
  import { Select } from "bits-ui";
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";
  import {
    defaultOperatorForField,
    getSmartField,
    operatorsForField,
  } from "$lib/music/smart-playlist/fields";
  import { coerceRuleValue } from "$lib/music/smart-playlist/compile";
  import {
    BOOLEAN_ITEMS,
    FIELD_ITEMS,
    PRESENCE_ITEMS,
    defaultValueFor,
    nextRangeValue,
    operatorLabel,
    rangeValues,
    valueInputType,
  } from "$lib/music/smart-playlist/editor";
  import type { SmartPlaylistRule } from "$lib/music/smart-playlist/types";

  interface Props {
    rule: SmartPlaylistRule;
    error?: string;
    onpatch: (patch: Partial<SmartPlaylistRule>) => void;
    onremove: () => void;
  }

  let { rule, error, onpatch, onremove }: Props = $props();

  const field = $derived(getSmartField(rule.field));
  const operatorItems = $derived(
    operatorsForField(rule.field).map((operator) => ({
      value: operator,
      label: operatorLabel(operator),
    })),
  );

  function onFieldChange(fieldId: string) {
    const operator = defaultOperatorForField(fieldId);
    onpatch({
      field: fieldId,
      operator,
      value: defaultValueFor(fieldId, operator),
    });
  }

  function onOperatorChange(operator: string) {
    onpatch({
      operator,
      value: defaultValueFor(rule.field, operator),
    });
  }
</script>

<div class="smart-rule" class:smart-rule--invalid={Boolean(error)}>
  <Select.Root
    type="single"
    items={FIELD_ITEMS}
    value={rule.field}
    onValueChange={(value) => onFieldChange(value)}
  >
    <Select.Trigger
      class="smart-rule__field smart-select-trigger"
      aria-label="Rule field"
    >
      <Select.Value />
      <MdiIcon name="chevronDown" size={14} />
    </Select.Trigger>
    <Select.Portal>
      <Select.Content class="smart-creator__select-content" sideOffset={4}>
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
    onValueChange={(value) => onOperatorChange(value)}
  >
    <Select.Trigger
      class="smart-rule__operator smart-select-trigger"
      aria-label="Rule operator"
    >
      <Select.Value />
      <MdiIcon name="chevronDown" size={14} />
    </Select.Trigger>
    <Select.Portal>
      <Select.Content class="smart-creator__select-content" sideOffset={4}>
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
          onpatch({
            value: nextRangeValue(
              rule,
              0,
              (event.currentTarget as HTMLInputElement).value,
            ),
          })}
      />
      <span>to</span>
      <input
        type={valueInputType(rule)}
        value={range[1]}
        placeholder="To"
        oninput={(event) =>
          onpatch({
            value: nextRangeValue(
              rule,
              1,
              (event.currentTarget as HTMLInputElement).value,
            ),
          })}
      />
    </div>
  {:else if rule.operator === "isMissing" || rule.operator === "isPresent"}
    <Select.Root
      type="single"
      items={PRESENCE_ITEMS}
      value={rule.value === true ? "present" : "missing"}
      onValueChange={(value) => onpatch({ value: value === "present" })}
    >
      <Select.Trigger
        class="smart-rule__value smart-select-trigger"
        aria-label="Rule value"
      >
        <Select.Value />
        <MdiIcon name="chevronDown" size={14} />
      </Select.Trigger>
      <Select.Portal>
        <Select.Content class="smart-creator__select-content" sideOffset={4}>
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
      onValueChange={(value) => onpatch({ value: value === "true" })}
    >
      <Select.Trigger
        class="smart-rule__value smart-select-trigger"
        aria-label="Rule value"
      >
        <Select.Value />
        <MdiIcon name="chevronDown" size={14} />
      </Select.Trigger>
      <Select.Portal>
        <Select.Content class="smart-creator__select-content" sideOffset={4}>
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
      placeholder={rule.field === "title" && rule.operator === "contains"
        ? "Text or /regex/"
        : "Value"}
      oninput={(event) =>
        onpatch({
          value: coerceRuleValue(
            (event.currentTarget as HTMLInputElement).value,
            rule.field,
            rule.operator,
          ),
        })}
    />
  {/if}

  <button
    type="button"
    class="smart-rule__remove"
    aria-label="Remove rule"
    onclick={onremove}
  >
    <MdiIcon name="trash2" size={16} />
  </button>

  {#if error}
    <p class="smart-rule__error">{error}</p>
  {/if}
</div>

<style>
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

  .smart-rule__range {
    display: grid;
    grid-template-columns: 1fr auto 1fr;
    gap: 0.35rem;
    align-items: center;
    font-size: 0.75rem;
    color: var(--jb-text-muted);
  }

  .smart-rule__error {
    margin: 0;
    color: var(--jb-danger);
    font-size: 0.75rem;
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

  @media (max-width: 720px) {
    .smart-rule {
      grid-template-columns: 1fr;
    }
  }
</style>
