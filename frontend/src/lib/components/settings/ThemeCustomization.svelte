<script lang="ts">
  import Button from "$lib/components/ui/Button.svelte";
  import { APP_NAME } from "$lib/brand";
  import Field from "$lib/components/ui/Field.svelte";
  import { theme, DEFAULT_ACCENT } from "$lib/theme/theme.svelte";
  import { toast } from "$lib/ui/toast.svelte";

  function resetCustomization() {
    theme.resetCustomization();
    toast.success("Theme customization reset");
  }
</script>

<div class="theme-customization">
  <Field
    label="Accent color"
    hint={`Overrides the default ${APP_NAME} red accent on this device.`}
  >
    <div class="theme-customization__accent-row">
      <input
        type="color"
        class="theme-customization__color"
        value={theme.accentColor}
        aria-label="Accent color"
        oninput={(e) =>
          theme.setAccentColor((e.currentTarget as HTMLInputElement).value)}
      />
      <input
        type="text"
        class="theme-customization__hex"
        value={theme.accentColor}
        maxlength="7"
        spellcheck="false"
        aria-label="Accent color hex value"
        oninput={(e) =>
          theme.setAccentColor((e.currentTarget as HTMLInputElement).value)}
      />
      <Button
        variant="ghost"
        size="sm"
        onclick={() => theme.setAccentColor(DEFAULT_ACCENT)}
      >
        Default
      </Button>
    </div>
  </Field>

  <Field
    label="Custom CSS"
    hint="Injected into the app on this device. Use CSS variables such as var(--jb-bg) and var(--jb-accent)."
  >
    <textarea
      class="theme-customization__css"
      rows="8"
      placeholder={`:root {\n  --jb-bg: #0a0a0a;\n}`}
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

  .theme-customization__accent-row {
    display: flex;
    align-items: center;
    gap: var(--jb-space-3);
    flex-wrap: wrap;
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
    flex: 1;
    min-width: 6rem;
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
</style>
