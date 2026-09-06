<script lang="ts">
  import { onMount } from "svelte";
  import Button from "$lib/components/ui/Button.svelte";
  import Input from "$lib/components/ui/Input.svelte";
  import SettingsCard from "$lib/components/settings/SettingsCard.svelte";
  import * as musicApi from "$lib/music/api";
  import { extensionFeatures } from "$lib/extensions/features.svelte";
  import { toast } from "$lib/ui/toast.svelte";
  import "$lib/settings/settings-page.css";

  let settings = $state<musicApi.RockskySettings | null>(null);
  let token = $state("");
  let testing = $state(false);
  let saving = $state(false);

  const enabled = $derived(extensionFeatures.isEnabled("rocksky"));

  onMount(() => {
    void load();
  });

  async function load() {
    const next = await musicApi.getRockskySettings().catch(() => null);
    if (!next) {
      settings = { enabled: false, hasToken: false, token: "" };
      return;
    }
    settings = next;
    token = "";
  }

  async function test() {
    const value = token.trim();
    if (!value) return;
    testing = true;
    try {
      const result = await musicApi.testRockskyToken(value);
      if (!result) {
        toast.error("Could not validate token");
        return;
      }
      if (result.ok) {
        toast.success(`Token valid for ${result.userName ?? "Rocksky"}`);
      } else {
        toast.error("Token is not valid");
      }
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Token test failed");
    } finally {
      testing = false;
    }
  }

  async function save() {
    const value = token.trim();
    if (!value) {
      toast.error("Enter a token to save");
      return;
    }
    saving = true;
    try {
      const next = await musicApi.saveRockskySettings({ token: value });
      if (next) {
        settings = next;
        token = "";
        toast.success("Rocksky token saved");
      } else {
        toast.error("Could not save token");
      }
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Could not save token");
    } finally {
      saving = false;
    }
  }
</script>

<SettingsCard title="Rocksky" description="Scrobble listens to Rocksky.">
  <div class="rocksky-settings">
    {#if !enabled}
      <p class="rocksky-settings__note">
        Enable the Rocksky extension in the Extensions tab to start scrobbling.
      </p>
    {/if}
    <label class="rocksky-settings__label" for="rocksky-token">
      API token
    </label>
    <Input
      id="rocksky-token"
      type="password"
      placeholder="Paste your Rocksky API token"
      bind:value={token}
      disabled={!settings}
    />
    {#if settings?.hasToken}
      <p class="rocksky-settings__status">
        A token is already saved. Type above to replace it.
      </p>
    {/if}
    <div class="rocksky-settings__actions">
      <Button
        size="sm"
        disabled={!token.trim() || testing || saving || !settings}
        onclick={() => void test()}
      >
        {testing ? "Testing..." : "Test token"}
      </Button>
      <Button
        size="sm"
        disabled={!token.trim() || saving || testing || !settings}
        onclick={() => void save()}
      >
        {saving ? "Saving..." : "Save token"}
      </Button>
    </div>
  </div>
</SettingsCard>

<style>
  .rocksky-settings {
    display: grid;
    gap: var(--jb-space-3);
    max-width: 32rem;
  }

  .rocksky-settings__note {
    margin: 0;
    font-size: 0.8125rem;
    color: var(--jb-text-muted);
  }

  .rocksky-settings__label {
    font-size: 0.8125rem;
    font-weight: 600;
  }

  .rocksky-settings__status {
    margin: 0;
    font-size: 0.8125rem;
    color: var(--jb-text-muted);
  }

  .rocksky-settings__actions {
    display: flex;
    gap: var(--jb-space-3);
  }
</style>
