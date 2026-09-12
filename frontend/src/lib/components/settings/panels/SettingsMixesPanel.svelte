<script lang="ts">
  import SettingsCard from "$lib/components/settings/SettingsCard.svelte";
  import { APP_NAME } from "$lib/brand";
  import SettingsToggleRow from "$lib/components/settings/SettingsToggleRow.svelte";
  import Field from "$lib/components/ui/Field.svelte";
  import Select from "$lib/components/ui/Select.svelte";
  import Button from "$lib/components/ui/Button.svelte";
  import { music } from "$lib/config/music.svelte";
  import {
    ALL_MIX_IDS,
    GENRE_SELECTION_LABELS,
    LANGUAGE_BIAS_LABELS,
    MIX_TYPE_LABELS,
    defaultMixSettings,
    formatPreferredLanguages,
    mergeMixSettings,
    parsePreferredLanguages,
    type MixId,
    type MixSettings,
  } from "$lib/music/mix-settings";
  import { toast } from "$lib/ui/toast.svelte";
  import "$lib/settings/settings-page.css";

  let mixSettings = $state<MixSettings>(mergeMixSettings(music.mixSettings));
  let mixSettingsSaving = $state(false);
  let mixRegenerating = $state(false);
  let preferredLanguagesInput = $state(
    formatPreferredLanguages(music.mixSettings.preferredLanguages),
  );

  const languageBiasOptions = Object.entries(LANGUAGE_BIAS_LABELS) as [
    keyof typeof LANGUAGE_BIAS_LABELS,
    string,
  ][];
  const genreSelectionOptions = Object.entries(GENRE_SELECTION_LABELS) as [
    keyof typeof GENRE_SELECTION_LABELS,
    string,
  ][];

  $effect(() => {
    mixSettings = mergeMixSettings(music.mixSettings);
    preferredLanguagesInput = formatPreferredLanguages(
      music.mixSettings.preferredLanguages,
    );
  });

  function toggleMixEnabled(id: MixId, checked: boolean) {
    const current =
      mixSettings.enabledMixIds.length === 0
        ? [...ALL_MIX_IDS]
        : [...mixSettings.enabledMixIds];

    let next: MixId[];
    if (checked) {
      next = current.includes(id) ? current : [...current, id];
    } else {
      next = current.filter((mixId) => mixId !== id);
    }

    mixSettings = {
      ...mixSettings,
      enabledMixIds: next.length === ALL_MIX_IDS.length ? [] : next,
    };
  }

  function isMixTypeEnabled(id: MixId): boolean {
    if (mixSettings.enabledMixIds.length === 0) return true;
    return mixSettings.enabledMixIds.includes(id);
  }

  async function applyMixSettings() {
    mixSettingsSaving = true;
    try {
      const next = mergeMixSettings({
        ...mixSettings,
        preferredLanguages: parsePreferredLanguages(preferredLanguagesInput),
      });
      mixSettings = next;
      preferredLanguagesInput = formatPreferredLanguages(
        next.preferredLanguages,
      );
      music.updateMixSettings(next);
      toast.success("Mix settings saved");
    } finally {
      mixSettingsSaving = false;
    }
  }

  async function regenerateMixes() {
    mixRegenerating = true;
    try {
      const next = mergeMixSettings({
        ...mixSettings,
        preferredLanguages: parsePreferredLanguages(preferredLanguagesInput),
      });
      mixSettings = next;
      music.updateMixSettings(next);
      await music.regenerateMixes();
      toast.success("Mixes regenerated");
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to regenerate mixes",
      );
    } finally {
      mixRegenerating = false;
    }
  }

  function resetMixSettings() {
    mixSettings = defaultMixSettings();
    preferredLanguagesInput = "";
    music.updateMixSettings(mixSettings);
    toast.success("Mix settings reset");
  }
</script>

<SettingsCard
  title="Mix generation"
  description={`Tune how ${APP_NAME} builds your daily mixes. Leave preferred languages empty to infer from listening history.`}
>
  <Field
    label="Language bias"
    hint="Prefer or restrict mixes toward your main languages."
  >
    <Select
      bind:value={mixSettings.languageBias}
      options={languageBiasOptions.map(([value, label]) => ({ value, label }))}
    />
  </Field>

  <Field
    label="Preferred languages"
    hint="ISO-style codes, comma-separated. Example: ru, en"
  >
    <input
      type="text"
      class="settings-input"
      bind:value={preferredLanguagesInput}
      placeholder="ru, en"
    />
  </Field>

  <Field
    label="Genre mix selection"
    hint="Personal uses albums from your top artists. Library uses server genre size."
  >
    <Select
      bind:value={mixSettings.genreSelection}
      options={genreSelectionOptions.map(([value, label]) => ({
        value,
        label,
      }))}
    />
  </Field>

  <Field
    label="Discover freshness (days)"
    hint="Tracks played within this window are excluded from Discover."
  >
    <input
      type="number"
      class="settings-input"
      min="7"
      max="90"
      bind:value={mixSettings.discoverRecentDays}
    />
  </Field>

  <Field
    label="Throwback window (days)"
    hint="Tracks not played for at least this long qualify for Throwback."
  >
    <input
      type="number"
      class="settings-input"
      min="14"
      max="365"
      bind:value={mixSettings.throwbackDays}
    />
  </Field>

  <Field
    label="Deep cut play count max"
    hint="Deep Cuts prefer tracks played at most this many times."
  >
    <input
      type="number"
      class="settings-input"
      min="0"
      max="10"
      bind:value={mixSettings.deepCutMaxPlayCount}
    />
  </Field>

  <Field
    label="Minimum tracks per mix"
    hint="Mixes with fewer tracks are skipped. Spotify uses about 30."
  >
    <input
      type="number"
      class="settings-input"
      min="5"
      max="50"
      bind:value={mixSettings.minTracksPerMix}
    />
  </Field>

  <Field label="Max tracks per mix">
    <input
      type="number"
      class="settings-input"
      min="10"
      max="100"
      bind:value={mixSettings.maxTracksPerMix}
    />
  </Field>

  <Field
    label="Flow album spacing"
    hint="Minimum gap between songs from the same album in mixes and radio."
  >
    <input
      type="number"
      class="settings-input"
      min="2"
      max="8"
      bind:value={mixSettings.flowAlbumLookback}
    />
  </Field>

  <Field
    label="Personal radio recency (hours)"
    hint="Skip tracks played within this window when seeding radio."
  >
    <input
      type="number"
      class="settings-input"
      min="0"
      max="24"
      bind:value={mixSettings.personalRadioRecencyHours}
    />
  </Field>

  <Field
    label="Personal radio cold start (plays)"
    hint="Below this play count, radio blends more random discovery."
  >
    <input
      type="number"
      class="settings-input"
      min="1"
      max="50"
      bind:value={mixSettings.personalRadioColdStartPlays}
    />
  </Field>

  <Field
    label="Radio explore bonus"
    hint="Higher values favor discovery over familiar artists (0 to 2)."
  >
    <input
      type="number"
      class="settings-input"
      min="0"
      max="2"
      step="0.1"
      bind:value={mixSettings.radioExploreBonus}
    />
  </Field>

  <SettingsToggleRow
    label="Avoid repeating tracks across mixes"
    checked={mixSettings.crossMixDedup}
    onchange={(checked) => {
      mixSettings = { ...mixSettings, crossMixDedup: checked };
    }}
  />

  <SettingsToggleRow
    label="Alternate long and short tracks in mix order"
    checked={mixSettings.durationPacing}
    onchange={(checked) => {
      mixSettings = { ...mixSettings, durationPacing: checked };
    }}
  />

  {#snippet footer()}
    <Button
      disabled={mixSettingsSaving}
      onclick={() => void applyMixSettings()}
    >
      Save mix settings
    </Button>
    <Button
      variant="ghost"
      disabled={mixRegenerating || !music.connected}
      onclick={() => void regenerateMixes()}
    >
      {mixRegenerating ? "Regenerating..." : "Regenerate all mixes"}
    </Button>
    <Button variant="ghost" onclick={resetMixSettings}>
      Reset mix defaults
    </Button>
  {/snippet}
</SettingsCard>

<SettingsCard
  title="Enabled mix types"
  description={`Choose which personalized mixes ${APP_NAME} generates.`}
>
  <div class="settings-grid settings-grid--mix-types">
    {#each ALL_MIX_IDS as mixId (mixId)}
      <SettingsToggleRow
        label={MIX_TYPE_LABELS[mixId]}
        checked={isMixTypeEnabled(mixId)}
        onchange={(checked) => toggleMixEnabled(mixId, checked)}
      />
    {/each}
  </div>
</SettingsCard>
