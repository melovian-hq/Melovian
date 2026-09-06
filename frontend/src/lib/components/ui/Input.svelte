<script lang="ts">
  type InputMode =
    | "none"
    | "text"
    | "tel"
    | "url"
    | "email"
    | "numeric"
    | "decimal"
    | "search";
  type EnterKeyHint =
    "enter" | "done" | "go" | "next" | "previous" | "search" | "send";
  type AutoCapitalize =
    "off" | "none" | "on" | "sentences" | "words" | "characters";

  interface Props {
    value?: string;
    type?: string;
    placeholder?: string;
    autocomplete?: string;
    disabled?: boolean;
    class?: string;
    id?: string;
    name?: string;
    required?: boolean;
    inputmode?: InputMode;
    enterkeyhint?: EnterKeyHint;
    autocapitalize?: AutoCapitalize;
    spellcheck?: boolean;
    autofocus?: boolean;
    onblur?: (event: FocusEvent) => void;
    oninput?: (event: Event) => void;
  }

  let {
    value = $bindable(""),
    type = "text",
    placeholder = "",
    autocomplete,
    disabled = false,
    class: className = "",
    id,
    name,
    required = false,
    inputmode,
    enterkeyhint,
    autocapitalize,
    spellcheck,
    autofocus = false,
    onblur,
    oninput,
  }: Props = $props();

  function focusIfNeeded(node: HTMLInputElement) {
    if (!autofocus) return;
    queueMicrotask(() => node.focus());
  }
</script>

<input
  {id}
  {name}
  {type}
  {placeholder}
  autocomplete={autocomplete as HTMLInputElement["autocomplete"]}
  {disabled}
  {required}
  {inputmode}
  {enterkeyhint}
  {autocapitalize}
  {spellcheck}
  {onblur}
  {oninput}
  bind:value
  class="input {className}"
  use:focusIfNeeded
/>

<style>
  .input {
    width: 100%;
    min-height: 2.75rem;
    padding: var(--jb-space-3);
    border: 1px solid var(--jb-border);
    border-radius: var(--jb-radius-md);
    background: var(--jb-surface);
    color: var(--jb-text);
    font: inherit;
    font-size: 1rem;
  }

  .input:focus {
    outline: none;
    box-shadow: var(--jb-focus-ring);
    border-color: var(--jb-accent);
  }

  .input:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }
</style>
