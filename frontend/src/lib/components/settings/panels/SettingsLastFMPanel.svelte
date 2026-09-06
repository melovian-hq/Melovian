<script lang="ts">
  import { onMount } from "svelte";
  import Button from "$lib/components/ui/Button.svelte";
  import Input from "$lib/components/ui/Input.svelte";
  import SettingsCard from "$lib/components/settings/SettingsCard.svelte";
  import * as musicApi from "$lib/music/api";
  import { extensionFeatures } from "$lib/extensions/features.svelte";
  import { toast } from "$lib/ui/toast.svelte";
  import "$lib/settings/settings-page.css";

  let settings = $state<musicApi.LastFMSettings | null>(null);
  let apiKey = $state("");
  let apiSecret = $state("");
  let sessionKey = $state("");
  let endpoint = $state("");
  let testing = $state(false);
  let saving = $state(false);

  const enabled = $derived(extensionFeatures.isEnabled("lastfm"));

  onMount(() => {
    void load();
  });

  async function load() {
    const next = await musicApi.getLastFMSettings().catch(() => null);
    if (!next) {
      settings = {
        enabled: false,
        hasToken: false,
        apiKey: "",
        apiSecret: "",
        sessionKey: "",
        endpoint: "",
      };
      return;
    }
    settings = next;
    apiKey = next.apiKey;
    apiSecret = "";
    sessionKey = "";
    endpoint = next.endpoint;
  }

  function canTest() {
    return (
      apiKey.trim() &&
      apiSecret.trim() &&
      sessionKey.trim() &&
      !testing &&
      !saving &&
      settings
    );
  }

  function canSave() {
    return (
      apiKey.trim() &&
      apiSecret.trim() &&
      sessionKey.trim() &&
      !saving &&
      !testing &&
      settings
    );
  }

  async function test() {
    if (!canTest()) return;
    testing = true;
    try {
      const result = await musicApi.testLastFMToken(
        apiKey.trim(),
        apiSecret.trim(),
        sessionKey.trim(),
        endpoint.trim() || "https://ws.audioscrobbler.com/2.0/",
      );
      if (!result) {
        toast.error("Could not validate credentials");
        return;
      }
      if (result.ok) {
        toast.success(`Credentials valid for ${result.userName ?? "Last.fm"}`);
      } else {
        toast.error("Credentials are not valid");
      }
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Credential test failed",
      );
    } finally {
      testing = false;
    }
  }

  async function save() {
    if (!canSave()) return;
    saving = true;
    try {
      const next = await musicApi.saveLastFMSettings({
        apiKey: apiKey.trim(),
        apiSecret: apiSecret.trim(),
        sessionKey: sessionKey.trim(),
        endpoint: endpoint.trim() || "https://ws.audioscrobbler.com/2.0/",
      });
      if (next) {
        settings = next;
        apiSecret = "";
        sessionKey = "";
        toast.success("Last.fm settings saved");
      } else {
        toast.error("Could not save settings");
      }
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Could not save settings",
      );
    } finally {
      saving = false;
    }
  }
</script>

<SettingsCard title="Last.fm" description="Scrobble listens to Last.fm.">
  <div class="lastfm-settings">
    {#if !enabled}
      <p class="lastfm-settings__note">
        Enable the Last.fm extension in the Extensions tab to start scrobbling.
      </p>
    {/if}
    <label class="lastfm-settings__label" for="lastfm-endpoint">
      Endpoint
    </label>
    <Input
      id="lastfm-endpoint"
      type="url"
      placeholder="https://ws.audioscrobbler.com/2.0/"
      bind:value={endpoint}
      disabled={!settings}
    />
    <label class="lastfm-settings__label" for="lastfm-api-key"> API key </label>
    <Input
      id="lastfm-api-key"
      type="text"
      placeholder="Your Last.fm API key"
      bind:value={apiKey}
      disabled={!settings}
    />
    <label class="lastfm-settings__label" for="lastfm-api-secret">
      API secret
    </label>
    <Input
      id="lastfm-api-secret"
      type="password"
      placeholder="Your Last.fm API secret"
      bind:value={apiSecret}
      disabled={!settings}
    />
    <label class="lastfm-settings__label" for="lastfm-session-key">
      Session key
    </label>
    <Input
      id="lastfm-session-key"
      type="password"
      placeholder="Your Last.fm session key"
      bind:value={sessionKey}
      disabled={!settings}
    />
    {#if settings?.hasToken}
      <p class="lastfm-settings__status">
        Credentials are already saved. Type above to replace them.
      </p>
    {/if}
    <div class="lastfm-settings__actions">
      <Button size="sm" disabled={!canTest()} onclick={() => void test()}>
        {testing ? "Testing..." : "Test credentials"}
      </Button>
      <Button size="sm" disabled={!canSave()} onclick={() => void save()}>
        {saving ? "Saving..." : "Save credentials"}
      </Button>
    </div>
  </div>
</SettingsCard>

<style>
  .lastfm-settings {
    display: grid;
    gap: var(--jb-space-3);
    max-width: 32rem;
  }

  .lastfm-settings__note {
    margin: 0;
    font-size: 0.8125rem;
    color: var(--jb-text-muted);
  }

  .lastfm-settings__label {
    font-size: 0.8125rem;
    font-weight: 600;
  }

  .lastfm-settings__status {
    margin: 0;
    font-size: 0.8125rem;
    color: var(--jb-text-muted);
  }

  .lastfm-settings__actions {
    display: flex;
    gap: var(--jb-space-3);
  }
</style>
