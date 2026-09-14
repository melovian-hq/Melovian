<script lang="ts">
  import Button from "$lib/components/ui/Button.svelte";
  import Field from "$lib/components/ui/Field.svelte";
  import { theme, type ThemeMode } from "$lib/theme/theme.svelte";
  import {
    ACCENT_PRESETS,
    CUSTOM_ACCENT_ID,
    isHexColor,
    type AccentPreset,
  } from "$lib/theme/accent";
  import {
    DEFAULT_PALETTE_ID,
    palettesForMode,
    RADIUS_STYLES,
    UI_SIZES,
  } from "$lib/theme/palettes";
  import { toast } from "$lib/ui/toast.svelte";

  const modes: { value: ThemeMode; label: string }[] = [
    { value: "dark", label: "Dark" },
    { value: "light", label: "Light" },
    { value: "system", label: "System" },
  ];

  const palettes = $derived(palettesForMode(theme.resolved));

  // Writable derived: tracks the resolved accent but accepts in-progress
  // edits while the user types a hex value.
  let hexDraft = $derived(theme.accentColor);
  const hexInvalid = $derived(
    hexDraft.trim().length > 0 && !isHexColor(hexDraft.trim()),
  );

  function presetSwatchColor(preset: AccentPreset): string {
    return theme.resolved === "dark" ? preset.dark : preset.light;
  }

  function onHexInput(value: string) {
    hexDraft = value;
    const trimmed = value.trim();
    if (isHexColor(trimmed)) theme.setAccentColor(trimmed);
  }

  function resetCustomization() {
    theme.resetCustomization();
    toast.success("Theme customization reset");
  }
</script>

<div class="theme-customization">
  <Field
    label="Appearance"
    group
    hint="Dark keeps the low-glare studio look. System follows your OS setting."
  >
    <div
      class="theme-customization__modes"
      role="group"
      aria-label="Appearance"
    >
      {#each modes as mode (mode.value)}
        <button
          type="button"
          class="theme-customization__mode"
          class:theme-customization__mode--active={theme.mode === mode.value}
          aria-pressed={theme.mode === mode.value}
          onclick={() => theme.setMode(mode.value)}
        >
          {mode.label}
        </button>
      {/each}
    </div>
  </Field>

  <Field
    label="Palette"
    group
    hint="Surface palette for the {theme.resolved} appearance."
  >
    <div
      class="theme-customization__swatches"
      role="group"
      aria-label="Palette"
    >
      <button
        type="button"
        class="theme-customization__swatch"
        class:theme-customization__swatch--active={theme.palette ===
          DEFAULT_PALETTE_ID}
        aria-pressed={theme.palette === DEFAULT_PALETTE_ID}
        onclick={() => theme.setPalette(DEFAULT_PALETTE_ID)}
      >
        <span class="theme-customization__dot theme-customization__dot--split"
        ></span>
        <span class="theme-customization__swatch-label">Default</span>
      </button>
      {#each palettes as palette (palette.id)}
        <button
          type="button"
          class="theme-customization__swatch"
          class:theme-customization__swatch--active={theme.palette ===
            palette.id}
          aria-pressed={theme.palette === palette.id}
          onclick={() => theme.setPalette(palette.id)}
        >
          <span
            class="theme-customization__dot"
            style="background: {palette.bg}; border-color: {palette.surface}"
          ></span>
          <span class="theme-customization__swatch-label">{palette.label}</span>
        </button>
      {/each}
    </div>
  </Field>

  <Field
    label="Accent"
    group
    hint="Used for buttons, links, and highlights on this device."
  >
    <div class="theme-customization__swatches" role="group" aria-label="Accent">
      {#each ACCENT_PRESETS as preset (preset.id)}
        <button
          type="button"
          class="theme-customization__swatch"
          class:theme-customization__swatch--active={theme.accentPreset ===
            preset.id}
          aria-pressed={theme.accentPreset === preset.id}
          title={preset.label}
          onclick={() => theme.setAccentPreset(preset.id)}
        >
          <span
            class="theme-customization__dot"
            style="background: {presetSwatchColor(preset)}"
          ></span>
          <span class="theme-customization__swatch-label">{preset.label}</span>
        </button>
      {/each}
      <button
        type="button"
        class="theme-customization__swatch"
        class:theme-customization__swatch--active={theme.accentPreset ===
          CUSTOM_ACCENT_ID}
        aria-pressed={theme.accentPreset === CUSTOM_ACCENT_ID}
        title="Custom hue"
        onclick={() => theme.setAccentPreset(CUSTOM_ACCENT_ID)}
      >
        <span
          class="theme-customization__dot"
          style="background: {theme.accentColor}"
        ></span>
        <span class="theme-customization__swatch-label">Custom</span>
      </button>
    </div>
  </Field>

  {#if theme.accentPreset === CUSTOM_ACCENT_ID}
    <Field
      label="Custom accent"
      group
      hint="Pick a hue. Shades are derived to stay readable in both themes."
    >
      <div class="theme-customization__accent-row">
        <input
          type="range"
          class="theme-customization__hue"
          min="0"
          max="360"
          step="1"
          value={theme.customHue}
          aria-label="Accent hue"
          oninput={(e) =>
            theme.setCustomHue(
              Number((e.currentTarget as HTMLInputElement).value),
            )}
        />
        <input
          type="color"
          class="theme-customization__color"
          value={theme.accentColor}
          aria-label="Accent color picker"
          oninput={(e) =>
            theme.setAccentColor((e.currentTarget as HTMLInputElement).value)}
        />
        <input
          type="text"
          class="theme-customization__hex"
          class:theme-customization__hex--invalid={hexInvalid}
          value={hexDraft}
          maxlength="7"
          spellcheck="false"
          aria-label="Accent color hex value"
          aria-invalid={hexInvalid}
          oninput={(e) =>
            onHexInput((e.currentTarget as HTMLInputElement).value)}
        />
        {#if hexInvalid}
          <span class="theme-customization__hex-hint">Use #rrggbb</span>
        {/if}
      </div>
    </Field>
  {/if}

  <Field
    label="Interface size"
    group
    hint="Scales text and controls across the app on this device."
  >
    <div
      class="theme-customization__modes"
      role="group"
      aria-label="Interface size"
    >
      {#each UI_SIZES as size (size.id)}
        <button
          type="button"
          class="theme-customization__mode"
          class:theme-customization__mode--active={theme.uiSize === size.id}
          aria-pressed={theme.uiSize === size.id}
          onclick={() => theme.setUiSize(size.id)}
        >
          {size.label}
        </button>
      {/each}
    </div>
  </Field>

  <Field
    label="Corners"
    group
    hint="Corner radius on cards, buttons, and panels."
  >
    <div class="theme-customization__modes" role="group" aria-label="Corners">
      {#each RADIUS_STYLES as style (style.id)}
        <button
          type="button"
          class="theme-customization__mode"
          class:theme-customization__mode--active={theme.radiusStyle ===
            style.id}
          aria-pressed={theme.radiusStyle === style.id}
          onclick={() => theme.setRadiusStyle(style.id)}
        >
          {style.label}
        </button>
      {/each}
    </div>
  </Field>

  <Field
    label="Custom CSS"
    hint="Injected into the app on this device. Use CSS variables such as var(--jb-bg) and var(--jb-accent)."
  >
    <textarea
      class="theme-customization__css"
      rows="8"
      placeholder={`:root {\n  --jb-bg: #0a0a0f;\n}`}
      value={theme.customCss}
      oninput={(e) =>
        theme.setCustomCss((e.currentTarget as HTMLTextAreaElement).value)}
    ></textarea>
  </Field>

  <div class="theme-customization__actions">
    <Button variant="ghost" size="sm" onclick={resetCustomization}>
      Reset customization
    </Button>
  </div>
</div>

<style>
  .theme-customization {
    display: grid;
    gap: var(--jb-space-4);
  }

  .theme-customization__modes {
    display: inline-flex;
    gap: var(--jb-space-1);
    padding: var(--jb-space-1);
    border: 1px solid var(--jb-border);
    border-radius: var(--jb-radius-full);
    background: var(--jb-bg-muted);
    width: fit-content;
  }

  .theme-customization__mode {
    padding: var(--jb-space-1) var(--jb-space-4);
    border: none;
    border-radius: var(--jb-radius-full);
    background: transparent;
    color: var(--jb-text-muted);
    font: inherit;
    font-size: 0.8125rem;
    font-weight: 600;
    cursor: pointer;
    transition:
      background var(--jb-transition),
      color var(--jb-transition);
  }

  .theme-customization__mode:hover {
    color: var(--jb-text);
  }

  .theme-customization__mode--active {
    background: var(--jb-accent);
    color: var(--jb-accent-text);
  }

  .theme-customization__swatches {
    display: flex;
    flex-wrap: wrap;
    gap: var(--jb-space-2);
  }

  .theme-customization__swatch {
    display: inline-flex;
    align-items: center;
    gap: var(--jb-space-2);
    padding: var(--jb-space-2) var(--jb-space-3);
    border: 1px solid var(--jb-border);
    border-radius: var(--jb-radius-full);
    background: var(--jb-surface);
    color: var(--jb-text-muted);
    font: inherit;
    font-size: 0.8125rem;
    cursor: pointer;
    transition:
      border-color var(--jb-transition),
      color var(--jb-transition);
  }

  .theme-customization__swatch:hover {
    border-color: var(--jb-border-strong);
    color: var(--jb-text);
  }

  .theme-customization__swatch--active {
    border-color: var(--jb-accent);
    color: var(--jb-text);
    box-shadow: var(--jb-focus-ring);
  }

  .theme-customization__dot {
    width: 0.875rem;
    height: 0.875rem;
    border-radius: var(--jb-radius-full);
    border: 1px solid var(--jb-border);
    flex-shrink: 0;
  }

  .theme-customization__dot--split {
    background: linear-gradient(
      135deg,
      var(--jb-bg) 0%,
      var(--jb-bg) 49%,
      var(--jb-accent) 50%,
      var(--jb-accent) 100%
    );
  }

  .theme-customization__accent-row {
    display: flex;
    align-items: center;
    gap: var(--jb-space-3);
    flex-wrap: wrap;
  }

  .theme-customization__hue {
    flex: 1;
    min-width: 10rem;
    height: 0.625rem;
    appearance: none;
    border-radius: var(--jb-radius-full);
    border: 1px solid var(--jb-border);
    background: linear-gradient(
      90deg,
      hsl(0 84% 68%),
      hsl(60 84% 68%),
      hsl(120 84% 68%),
      hsl(180 84% 68%),
      hsl(240 84% 68%),
      hsl(300 84% 68%),
      hsl(360 84% 68%)
    );
    cursor: pointer;
  }

  .theme-customization__hue::-webkit-slider-thumb {
    appearance: none;
    width: 1.125rem;
    height: 1.125rem;
    border-radius: var(--jb-radius-full);
    background: var(--jb-accent);
    border: 2px solid var(--jb-accent-text);
  }

  .theme-customization__hue::-moz-range-thumb {
    width: 1.125rem;
    height: 1.125rem;
    border-radius: var(--jb-radius-full);
    background: var(--jb-accent);
    border: 2px solid var(--jb-accent-text);
  }

  .theme-customization__color {
    width: 2.75rem;
    height: 2.75rem;
    padding: 0;
    border: 1px solid var(--jb-border);
    border-radius: var(--jb-radius-md);
    background: transparent;
    cursor: pointer;
  }

  .theme-customization__hex {
    width: 6.5rem;
    padding: var(--jb-space-3);
    border: 1px solid var(--jb-border);
    border-radius: var(--jb-radius-md);
    background: var(--jb-surface);
    color: var(--jb-text);
    font: inherit;
    font-size: 0.875rem;
    font-family: var(--jb-font-mono);
  }

  .theme-customization__hex:focus {
    outline: none;
    box-shadow: var(--jb-focus-ring);
    border-color: var(--jb-accent);
  }

  .theme-customization__hex--invalid {
    border-color: var(--jb-danger);
  }

  .theme-customization__hex-hint {
    color: var(--jb-text-subtle);
    font-size: 0.8125rem;
  }

  .theme-customization__css {
    width: 100%;
    min-height: 10rem;
    padding: var(--jb-space-3);
    border: 1px solid var(--jb-border);
    border-radius: var(--jb-radius-md);
    background: var(--jb-surface);
    color: var(--jb-text);
    font-family: var(--jb-font-mono);
    font-size: 0.8125rem;
    line-height: 1.5;
    resize: vertical;
  }

  .theme-customization__css:focus {
    outline: none;
    box-shadow: var(--jb-focus-ring);
    border-color: var(--jb-accent);
  }

  .theme-customization__actions {
    display: flex;
    justify-content: flex-start;
  }

  @media (prefers-reduced-motion: reduce) {
    .theme-customization__mode,
    .theme-customization__swatch {
      transition: none;
    }
  }
</style>
