<script lang="ts">
  import SettingsCard from "$lib/components/settings/SettingsCard.svelte";
  import { APP_NAME, APP_SLUG } from "$lib/brand";
  import SettingsToggleRow from "$lib/components/settings/SettingsToggleRow.svelte";
  import Field from "$lib/components/ui/Field.svelte";
  import Button from "$lib/components/ui/Button.svelte";
  import { fetchWithRetry, apiHeaders } from "$lib/core/http/client";
  import { ApiPaths } from "$lib/core/http/api-paths";
  import { parseJson } from "$lib/core/http/parse";
  import { runtimeConfigSchema } from "$lib/config/schemas";
  import { auth } from "$lib/features/auth/store.svelte";
  import {
    applyRuntimeSentryConfig,
    disableClientSentry,
  } from "$lib/core/sentry";
  import {
    defaultSentryClientSettings,
    defaultStoredSentrySettings,
    getSentryClientSettings,
    getSentryServerSettings,
    mergeStoredSentrySettings,
    saveSentryClientSettings,
    saveSentryServerSettings,
    sendSentryTestEvent,
    type SentryClientSettings,
    type SentryEnvLocks,
    type StoredSentrySettings,
  } from "$lib/core/sentry-settings";
  import { toast } from "$lib/ui/toast.svelte";
  import "$lib/settings/settings-page.css";

  let sentryServer = $state<StoredSentrySettings>(
    defaultStoredSentrySettings(),
  );
  let sentryEnvLocks = $state<SentryEnvLocks>({});
  let sentryEffectiveEnabled = $state(false);
  let sentryClient = $state<SentryClientSettings>(
    defaultSentryClientSettings(),
  );
  let sentryServerLoading = $state(false);
  let sentryClientLoading = $state(false);
  let sentryServerSaving = $state(false);
  let sentryClientSaving = $state(false);
  let sentryServerAvailable = $state(false);
  let sentryTestSending = $state(false);
  let sentryTestStatus = $state("");

  const canManageServerSentry = $derived(
    !auth.demoMode && (!auth.enabled || auth.authenticated),
  );
  const sentryTestDisabled = $derived(
    sentryTestSending || !sentryEffectiveEnabled || !sentryServerAvailable,
  );

  $effect(() => {
    void loadSentrySettings();
  });

  async function loadSentrySettings() {
    sentryClientLoading = true;
    try {
      sentryClient = await getSentryClientSettings();
    } catch {
      sentryClient = defaultSentryClientSettings();
    } finally {
      sentryClientLoading = false;
    }

    if (!canManageServerSentry) {
      sentryServerAvailable = false;
      return;
    }

    sentryServerLoading = true;
    try {
      const response = await getSentryServerSettings();
      sentryServer = mergeStoredSentrySettings(response.stored);
      sentryEnvLocks = response.envLocks;
      sentryEffectiveEnabled = response.effective.enabled;
      sentryServerAvailable = true;
    } catch {
      sentryServerAvailable = false;
    } finally {
      sentryServerLoading = false;
    }
  }

  async function handleSaveSentryServerSettings() {
    sentryServerSaving = true;
    try {
      const response = await saveSentryServerSettings(sentryServer);
      sentryServer = mergeStoredSentrySettings(response.stored);
      sentryEnvLocks = response.envLocks;
      sentryEffectiveEnabled = response.effective.enabled;
      sentryTestStatus = "";
      toast.success("Server error tracking saved");
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to save error tracking",
      );
    } finally {
      sentryServerSaving = false;
    }
  }

  async function handleSaveSentryClientSettings() {
    sentryClientSaving = true;
    try {
      sentryClient = await saveSentryClientSettings(sentryClient);
      const configResponse = await fetchWithRetry(ApiPaths.config, {
        headers: apiHeaders(),
      });
      if (configResponse.ok) {
        const cfg = await parseJson(
          runtimeConfigSchema,
          configResponse,
          "runtime config",
        );
        if (sentryClient.enabled && cfg.sentry?.clientReporting) {
          applyRuntimeSentryConfig(cfg.sentry);
        } else {
          disableClientSentry();
        }
      }
      toast.success("Client error reporting updated");
    } catch (err) {
      toast.error(
        err instanceof Error
          ? err.message
          : "Failed to save client error reporting",
      );
    } finally {
      sentryClientSaving = false;
    }
  }

  async function handleSendSentryTestEvent() {
    sentryTestSending = true;
    sentryTestStatus = "";
    try {
      const result = await sendSentryTestEvent();
      const detail = result.message
        ? `${result.message} Event id: ${result.eventId}`
        : `Test event sent. Event id: ${result.eventId}`;
      sentryTestStatus = detail;
      toast.success(detail);
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "Failed to send test event";
      sentryTestStatus = message;
      toast.error(message);
    } finally {
      sentryTestSending = false;
    }
  }
</script>

<SettingsCard
  title="Error tracking"
  description="Send backend and client errors to Sentry or GlitchTip. Environment variables override stored values when set."
>
  {#if sentryClientLoading}
    <p class="settings-page__meta">Loading client settings...</p>
  {:else}
    <SettingsToggleRow
      label="Report errors from this device"
      description="When enabled, frontend crashes and unhandled errors are sent to your configured tracker."
      checked={sentryClient.enabled}
      onchange={(enabled) => {
        sentryClient = { ...sentryClient, enabled };
      }}
    />
    <Button
      variant="primary"
      disabled={sentryClientSaving}
      onclick={() => void handleSaveSentryClientSettings()}
    >
      Save client reporting
    </Button>
  {/if}

  {#if canManageServerSentry}
    <hr class="settings-page__divider" />
    {#if sentryServerLoading}
      <p class="settings-page__meta">Loading server settings...</p>
    {:else if sentryServerAvailable}
      <SettingsToggleRow
        label="Enable server error tracking"
        description={`Report Go panics, HTTP 5xx responses, and client errors received by this ${APP_NAME} instance.`}
        checked={sentryServer.enabled}
        disabled={sentryEnvLocks.dsn === true}
        onchange={(enabled) => {
          sentryServer = { ...sentryServer, enabled };
        }}
      />
      <Field
        label="Backend DSN"
        hint={sentryEnvLocks.dsn
          ? "Locked by environment variable."
          : "Sentry-compatible DSN for the Go backend."}
      >
        <input
          class="settings-input"
          type="url"
          value={sentryServer.dsn ?? ""}
          disabled={sentryEnvLocks.dsn === true || !sentryServer.enabled}
          oninput={(e) => {
            sentryServer = {
              ...sentryServer,
              dsn: (e.currentTarget as HTMLInputElement).value,
            };
          }}
        />
      </Field>
      <Field
        label="Frontend DSN"
        hint={sentryEnvLocks.frontendDsn
          ? "Locked by environment variable."
          : "Optional separate DSN for browsers. Leave empty to reuse the backend DSN."}
      >
        <input
          class="settings-input"
          type="url"
          value={sentryServer.frontendDsn ?? ""}
          disabled={sentryEnvLocks.frontendDsn === true ||
            !sentryServer.enabled}
          oninput={(e) => {
            sentryServer = {
              ...sentryServer,
              frontendDsn: (e.currentTarget as HTMLInputElement).value,
            };
          }}
        />
      </Field>
      <Field
        label="Environment"
        hint={sentryEnvLocks.environment
          ? "Locked by environment variable."
          : "Optional environment tag, for example production or staging."}
      >
        <input
          class="settings-input"
          type="text"
          value={sentryServer.environment ?? ""}
          disabled={sentryEnvLocks.environment === true ||
            !sentryServer.enabled}
          oninput={(e) => {
            sentryServer = {
              ...sentryServer,
              environment: (e.currentTarget as HTMLInputElement).value,
            };
          }}
        />
      </Field>
      <Field
        label="Release"
        hint={sentryEnvLocks.release
          ? "Locked by environment variable."
          : `Optional release name, for example ${APP_SLUG}@0.1.0.`}
      >
        <input
          class="settings-input"
          type="text"
          value={sentryServer.release ?? ""}
          disabled={sentryEnvLocks.release === true || !sentryServer.enabled}
          oninput={(e) => {
            sentryServer = {
              ...sentryServer,
              release: (e.currentTarget as HTMLInputElement).value,
            };
          }}
        />
      </Field>
      <Field
        label="Trace sample rate"
        hint={sentryEnvLocks.tracesSampleRate
          ? "Locked by environment variable."
          : "Performance trace sampling from 0 (off) to 1 (all requests)."}
      >
        <input
          class="settings-input"
          type="number"
          min="0"
          max="1"
          step="0.05"
          value={sentryServer.tracesSampleRate ?? 0}
          disabled={sentryEnvLocks.tracesSampleRate === true ||
            !sentryServer.enabled}
          oninput={(e) => {
            const value = Number((e.currentTarget as HTMLInputElement).value);
            sentryServer = {
              ...sentryServer,
              tracesSampleRate: Number.isFinite(value) ? value : 0,
            };
          }}
        />
      </Field>
      <SettingsToggleRow
        label="Allow client reporting"
        description="When off, browsers will not receive a frontend DSN even if one is configured."
        checked={sentryServer.clientReportingAllowed}
        disabled={!sentryServer.enabled && !sentryEffectiveEnabled}
        onchange={(clientReportingAllowed) => {
          sentryServer = {
            ...sentryServer,
            clientReportingAllowed,
          };
        }}
      />
      <div class="settings-page__backup-buttons">
        <Button
          variant="primary"
          disabled={sentryServerSaving}
          onclick={() => void handleSaveSentryServerSettings()}
        >
          Save server error tracking
        </Button>
        <Button
          variant="ghost"
          disabled={sentryTestDisabled}
          onclick={() => void handleSendSentryTestEvent()}
        >
          {sentryTestSending ? "Sending…" : "Send test event"}
        </Button>
      </div>
      <p class="settings-page__meta">
        Backend tracking: {sentryEffectiveEnabled ? "active" : "inactive"}
      </p>
      {#if sentryTestStatus}
        <p class="settings-page__meta" role="status">{sentryTestStatus}</p>
      {/if}
      <p class="settings-page__meta">
        Save server settings before testing. The test button stays disabled
        until backend tracking is active.
      </p>
    {:else}
      <p class="settings-page__meta">
        Server error tracking settings are not available on this instance.
      </p>
    {/if}
  {/if}
</SettingsCard>
