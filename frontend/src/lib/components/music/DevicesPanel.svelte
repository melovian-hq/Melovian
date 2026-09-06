<script lang="ts">
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";
  import Button from "$lib/components/ui/Button.svelte";
  import Input from "$lib/components/ui/Input.svelte";
  import CoverArt from "$lib/components/ui/CoverArt.svelte";
  import AmbientCoverBackdrop from "$lib/components/ui/AmbientCoverBackdrop.svelte";
  import DeviceAvatar from "$lib/components/ui/DeviceAvatar.svelte";
  import AvatarStack from "$lib/components/ui/AvatarStack.svelte";
  import { deviceSync } from "$lib/music/device-sync.svelte";
  import { music } from "$lib/config/music.svelte";
  import { coverArtUrl } from "$lib/subsonic";
  import { auth } from "$lib/features/auth/store.svelte";
  import {
    resolveAmbientCoverArt,
    peerStatusLabel,
  } from "$lib/music/devices-panel-display";

  let inviteUsername = $state("");
  let inviteBusy = $state(false);

  const ambientCoverId = $derived(
    resolveAmbientCoverArt({
      localCoverArt:
        music.currentTrack?.coverArt ??
        music.currentTrack?.albumId ??
        music.currentTrack?.id,
      activePeerCoverArt: deviceSync.activePeer?.playback?.coverArt,
    }),
  );
  const ambientSrc = $derived(
    ambientCoverId ? coverArtUrl(music.config, ambientCoverId, 200) : null,
  );
  const isSelfActive = $derived(deviceSync.isSelfActive);
  const linkBusy = $derived(
    deviceSync.linkState === "connecting" || deviceSync.pendingOp !== null,
  );
  const sessionStackMembers = $derived(
    deviceSync.sessionMembers.map((d) => ({
      seed: d.deviceId,
      label: d.username ? `${d.name} (${d.username})` : d.name,
      self: d.deviceId === deviceSync.deviceId,
      active: d.isActivePlayer,
    })),
  );
  const canInvite = $derived(
    auth.enabled && auth.authenticated && deviceSync.isHost,
  );

  function rename() {
    const next = window.prompt("Device name", deviceSync.deviceName);
    if (next != null) deviceSync.rename(next);
  }

  async function copyInvite() {
    inviteBusy = true;
    try {
      await deviceSync.copyInviteToClipboard();
    } finally {
      inviteBusy = false;
    }
  }

  async function inviteUser() {
    const name = inviteUsername.trim();
    if (!name) return;
    inviteBusy = true;
    try {
      const ok = await deviceSync.inviteByUsername(name);
      if (ok) inviteUsername = "";
    } finally {
      inviteBusy = false;
    }
  }
</script>

{#if deviceSync.panelOpen}
  <div class="devices-panel" role="dialog" aria-label="Devices">
    {#if ambientSrc && ambientCoverId}
      <AmbientCoverBackdrop src={ambientSrc} seed={ambientCoverId} />
    {/if}

    <div class="devices-panel__shell">
      <header class="devices-panel__head">
        <div class="devices-panel__title">
          <MdiIcon name="monitor" size={16} />
          <h3>Devices</h3>
        </div>
        <button
          type="button"
          class="devices-panel__icon-btn"
          aria-label="Close devices"
          onclick={() => (deviceSync.panelOpen = false)}
        >
          <MdiIcon name="x" size={16} />
        </button>
      </header>

      <div class="devices-panel__body">
        <div
          class="devices-panel__status"
          role="status"
          class:devices-panel__status--error={deviceSync.linkState ===
            "error" || !!deviceSync.lastError}
          class:devices-panel__status--busy={linkBusy}
        >
          <span>{deviceSync.lastError ?? deviceSync.statusLabel}</span>
          {#if deviceSync.linkState !== "connected"}
            <button
              type="button"
              class="devices-panel__chip"
              onclick={() => deviceSync.retryConnect()}
            >
              Retry
            </button>
          {/if}
        </div>

        <div
          class="devices-panel__self"
          class:devices-panel__self--active={isSelfActive}
        >
          <DeviceAvatar
            seed={deviceSync.deviceId}
            label={deviceSync.deviceName}
            self={true}
            active={isSelfActive}
            size={32}
          />
          <div class="devices-panel__self-copy">
            <p class="devices-panel__name">{deviceSync.deviceName}</p>
            <p class="devices-panel__meta">
              This device{isSelfActive
                ? " · Playing"
                : " · Idle"}{deviceSync.inListenTogether
                ? deviceSync.isHost
                  ? " · Hosting"
                  : " · Listening together"
                : ""}
            </p>
          </div>
          <div class="devices-panel__self-actions">
            {#if !isSelfActive}
              <button
                type="button"
                class="devices-panel__chip"
                title="Play on this device"
                onclick={() => deviceSync.takeOver()}
              >
                Play here
              </button>
            {/if}
            <button
              type="button"
              class="devices-panel__icon-btn"
              aria-label="Rename device"
              title="Rename"
              onclick={rename}
            >
              <MdiIcon name="pencil" size={16} />
            </button>
          </div>
        </div>

        {#if deviceSync.peers.length === 0}
          <p class="devices-panel__empty">No other devices connected yet.</p>
        {:else}
          <ul class="devices-panel__list">
            {#each deviceSync.peers as peer (peer.deviceId)}
              {@const coverId = peer.playback?.coverArt}
              {@const coverSrc = coverId
                ? coverArtUrl(music.config, coverId, 64)
                : null}
              <li
                class="devices-panel__item"
                class:devices-panel__item--together={peer.sessionId &&
                  peer.sessionId === deviceSync.sessionId}
              >
                <div class="devices-panel__thumb">
                  <DeviceAvatar
                    seed={peer.deviceId}
                    label={peer.name}
                    active={peer.isActivePlayer}
                    size={36}
                  />
                  {#if coverSrc && coverId}
                    <span class="devices-panel__thumb-badge">
                      <CoverArt src={coverSrc} seed={coverId} alt="" />
                    </span>
                  {/if}
                </div>

                <div class="devices-panel__item-copy">
                  <p class="devices-panel__name">{peer.name}</p>
                  <p class="devices-panel__meta">{peerStatusLabel(peer)}</p>
                </div>

                <div class="devices-panel__actions">
                  {#if !peer.isActivePlayer}
                    <button
                      type="button"
                      class="devices-panel__chip"
                      title="Play on {peer.name}"
                      onclick={() => deviceSync.transferTo(peer.deviceId)}
                    >
                      Transfer
                    </button>
                  {/if}
                  {#if peer.sessionId && !deviceSync.sessionId}
                    <button
                      type="button"
                      class="devices-panel__chip"
                      title="Join listen together"
                      onclick={() =>
                        deviceSync.joinListenTogether(peer.sessionId!)}
                    >
                      Join
                    </button>
                  {/if}
                  {#if deviceSync.isHost && deviceSync.sessionId && peer.sessionId !== deviceSync.sessionId}
                    <button
                      type="button"
                      class="devices-panel__chip"
                      title="Invite {peer.name} to listen together"
                      onclick={() => deviceSync.inviteDevice(peer.deviceId)}
                    >
                      Invite
                    </button>
                  {/if}
                  {#if peer.isActivePlayer}
                    <button
                      type="button"
                      class="devices-panel__icon-btn devices-panel__icon-btn--surface"
                      title="Play on that device"
                      aria-label="Play on that device"
                      onclick={() => deviceSync.remoteCommand("play")}
                    >
                      <MdiIcon name="play" size={16} />
                    </button>
                    <button
                      type="button"
                      class="devices-panel__icon-btn devices-panel__icon-btn--surface"
                      title="Pause that device"
                      aria-label="Pause that device"
                      onclick={() => deviceSync.remoteCommand("pause")}
                    >
                      <MdiIcon name="pause" size={16} />
                    </button>
                    <button
                      type="button"
                      class="devices-panel__icon-btn devices-panel__icon-btn--surface"
                      title="Next on that device"
                      aria-label="Next on that device"
                      onclick={() => deviceSync.remoteCommand("next")}
                    >
                      <MdiIcon name="skipForward" size={16} />
                    </button>
                  {/if}
                </div>
              </li>
            {/each}
          </ul>
        {/if}
      </div>

      <footer
        class="devices-panel__together"
        class:devices-panel__together--live={deviceSync.inListenTogether}
      >
        {#if deviceSync.sessionId}
          <div class="devices-panel__together-copy">
            {#if sessionStackMembers.length > 0}
              <div class="devices-panel__together-stack">
                <AvatarStack members={sessionStackMembers} size={22} max={4} />
              </div>
            {/if}
            <p class="devices-panel__together-title">
              {deviceSync.sessionLabel ||
                (deviceSync.isHost
                  ? "Listening together · waiting"
                  : "Listening together")}
            </p>
            {#if deviceSync.sessionMemberLabel}
              <p class="devices-panel__meta">{deviceSync.sessionMemberLabel}</p>
            {/if}
            {#if deviceSync.isHost && deviceSync.sessionMemberCount < 2}
              <p class="devices-panel__meta">
                {#if deviceSync.peers.some((p) => p.sessionId !== deviceSync.sessionId)}
                  Invite a connected device above
                  {#if canInvite}
                    , or another account by username/link below
                  {/if}.
                {:else if canInvite}
                  Invite another account on this server by username, or copy a
                  link.
                {:else}
                  Waiting for another device to connect.
                {/if}
              </p>
            {/if}
            {#if deviceSync.following}
              <p class="devices-panel__meta">
                Play, pause, skip, and seek go to the host.
              </p>
            {/if}
            {#if canInvite}
              <div class="devices-panel__invite">
                <Input
                  bind:value={inviteUsername}
                  placeholder="Invite by username"
                  id="party-invite-username"
                />
                <Button
                  size="sm"
                  variant="surface"
                  disabled={inviteBusy || !inviteUsername.trim()}
                  onclick={() => void inviteUser()}
                >
                  Invite
                </Button>
                <Button
                  size="sm"
                  variant="ghost"
                  disabled={inviteBusy}
                  onclick={() => void copyInvite()}
                >
                  Copy link
                </Button>
              </div>
            {/if}
            {#if deviceSync.pendingOp === "join-session" || deviceSync.pendingOp === "create-session"}
              <p class="devices-panel__meta">{deviceSync.statusLabel}</p>
            {/if}
          </div>
          <Button
            variant="ghost"
            size="sm"
            onclick={() => deviceSync.leaveListenTogether()}
          >
            Leave
          </Button>
        {:else}
          <Button
            size="sm"
            disabled={deviceSync.pendingOp === "create-session"}
            onclick={() => deviceSync.createListenTogether()}
          >
            {deviceSync.pendingOp === "create-session"
              ? "Starting…"
              : "Start listen together"}
          </Button>
        {/if}
      </footer>
    </div>
  </div>
{/if}

<style>
  .devices-panel {
    position: fixed;
    right: var(--jb-space-4);
    bottom: calc(
      var(--jb-bottom-chrome-height, 5.5rem) +
        env(safe-area-inset-bottom, 0px) + var(--jb-space-2)
    );
    z-index: 60;
    width: min(22rem, calc(100vw - 2rem));
    max-height: min(28rem, 60vh);
    display: flex;
    flex-direction: column;
    border: 1px solid var(--jb-border);
    border-radius: var(--jb-radius-xl);
    background: color-mix(in srgb, var(--jb-bg-elevated) 96%, transparent);
    backdrop-filter: blur(20px);
    box-shadow: var(--jb-shadow-lg);
    overflow: hidden;
    contain: layout paint;
    animation: devices-panel-enter 180ms ease-out;
  }

  .devices-panel__shell {
    position: relative;
    z-index: 1;
    display: flex;
    flex-direction: column;
    min-height: 0;
    max-height: inherit;
    flex: 1;
  }

  .devices-panel__head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--jb-space-3);
    padding: var(--jb-space-3) var(--jb-space-4);
    border-bottom: 1px solid
      color-mix(in srgb, var(--jb-border) 80%, transparent);
    flex-shrink: 0;
  }

  .devices-panel__title {
    display: flex;
    align-items: center;
    gap: var(--jb-space-2);
    color: var(--jb-text);
  }

  .devices-panel__head h3 {
    margin: 0;
    font-size: 0.875rem;
    font-weight: 700;
  }

  .devices-panel__body {
    flex: 1;
    min-height: 0;
    overflow: auto;
    padding: var(--jb-space-3) var(--jb-space-4);
    display: flex;
    flex-direction: column;
    gap: var(--jb-space-3);
  }

  .devices-panel__status {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--jb-space-2);
    font-size: 0.75rem;
    color: var(--jb-text-muted);
  }

  .devices-panel__status--busy {
    color: var(--jb-accent);
  }

  .devices-panel__status--error {
    color: var(--jb-danger, #f87171);
  }

  .devices-panel__self-actions {
    display: flex;
    align-items: center;
    gap: 0.35rem;
    flex-shrink: 0;
  }

  .devices-panel__self {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--jb-space-3);
    padding: var(--jb-space-3);
    border-radius: var(--jb-radius-md);
    border: 1px solid var(--jb-border);
    background: color-mix(in srgb, var(--jb-surface) 72%, transparent);
    --avatar-stack-ring: color-mix(in srgb, var(--jb-surface) 72%, transparent);
  }

  .devices-panel__self--active {
    border-color: color-mix(in srgb, var(--jb-accent) 55%, var(--jb-border));
    box-shadow: inset 0 0 0 1px
      color-mix(in srgb, var(--jb-accent) 35%, transparent);
    background: color-mix(in srgb, var(--jb-accent-muted) 55%, transparent);
  }

  .devices-panel__self-copy,
  .devices-panel__item-copy {
    min-width: 0;
    flex: 1;
  }

  .devices-panel__name {
    margin: 0;
    font-weight: 650;
    font-size: 0.875rem;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .devices-panel__meta {
    margin: 0.125rem 0 0;
    font-size: 0.75rem;
    color: var(--jb-text-muted);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .devices-panel__empty {
    margin: 0;
    font-size: 0.8125rem;
    color: var(--jb-text-muted);
  }

  .devices-panel__list {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: var(--jb-space-2);
  }

  .devices-panel__item {
    display: grid;
    grid-template-columns: auto 1fr;
    grid-template-areas:
      "thumb copy"
      "actions actions";
    align-items: center;
    gap: var(--jb-space-2) var(--jb-space-3);
    padding: var(--jb-space-2);
    border-radius: var(--jb-radius-md);
    background: color-mix(in srgb, var(--jb-surface) 40%, transparent);
  }

  .devices-panel__item--together {
    border: 1px solid color-mix(in srgb, var(--jb-accent) 40%, var(--jb-border));
    background: color-mix(in srgb, var(--jb-accent-muted) 40%, transparent);
  }

  .devices-panel__thumb {
    grid-area: thumb;
    position: relative;
    width: 2.25rem;
    height: 2.25rem;
    flex-shrink: 0;
    --avatar-stack-ring: color-mix(in srgb, var(--jb-surface) 40%, transparent);
  }

  .devices-panel__thumb-badge {
    position: absolute;
    right: -0.3rem;
    top: -0.3rem;
    width: 0.95rem;
    height: 0.95rem;
    border-radius: var(--jb-radius-sm);
    overflow: hidden;
    box-shadow: 0 0 0 2px var(--jb-bg-elevated);
    z-index: 2;
  }

  .devices-panel__thumb-badge :global(.cover-art) {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }

  .devices-panel__item-copy {
    grid-area: copy;
  }

  .devices-panel__actions {
    grid-area: actions;
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.35rem;
    justify-content: flex-end;
  }

  .devices-panel__chip {
    border: 1px solid var(--jb-border);
    border-radius: var(--jb-radius-full);
    background: color-mix(in srgb, var(--jb-surface) 80%, transparent);
    color: var(--jb-text-muted);
    font-size: 0.6875rem;
    font-weight: 600;
    padding: 0.25rem 0.55rem;
    cursor: pointer;
  }

  .devices-panel__chip:hover {
    color: var(--jb-accent);
    border-color: color-mix(in srgb, var(--jb-accent) 50%, var(--jb-border));
  }

  .devices-panel__invite {
    display: grid;
    grid-template-columns: 1fr auto auto;
    gap: 0.35rem;
    margin-top: var(--jb-space-2);
    width: 100%;
  }

  .devices-panel__invite :global(input) {
    min-width: 0;
    font-size: 0.8125rem;
  }

  .devices-panel__icon-btn {
    display: grid;
    place-content: center;
    width: 1.75rem;
    height: 1.75rem;
    border: none;
    border-radius: var(--jb-radius-sm);
    background: transparent;
    color: var(--jb-text-muted);
    cursor: pointer;
    flex-shrink: 0;
  }

  .devices-panel__icon-btn:hover {
    color: var(--jb-text);
    background: var(--jb-accent-muted);
  }

  .devices-panel__icon-btn--surface {
    border: 1px solid var(--jb-border);
    background: color-mix(in srgb, var(--jb-surface) 80%, transparent);
  }

  .devices-panel__together {
    display: flex;
    flex-direction: row;
    align-items: flex-start;
    justify-content: space-between;
    gap: var(--jb-space-2);
    padding: var(--jb-space-3) var(--jb-space-4);
    border-top: 1px solid color-mix(in srgb, var(--jb-border) 80%, transparent);
    background: color-mix(in srgb, var(--jb-bg-elevated) 55%, transparent);
    flex-shrink: 0;
  }

  .devices-panel__together--live {
    border-top-color: color-mix(
      in srgb,
      var(--jb-accent) 40%,
      var(--jb-border)
    );
    background: color-mix(in srgb, var(--jb-accent-muted) 35%, transparent);
  }

  .devices-panel__together-copy {
    min-width: 0;
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 0.35rem;
    --avatar-stack-ring: color-mix(
      in srgb,
      var(--jb-accent-muted) 35%,
      transparent
    );
  }

  .devices-panel__together-stack {
    margin-bottom: 0.1rem;
  }

  .devices-panel__together-title {
    margin: 0;
    font-size: 0.8125rem;
    font-weight: 700;
    color: var(--jb-accent);
  }

  @keyframes devices-panel-enter {
    from {
      opacity: 0;
      transform: translateY(8px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }

  @media (prefers-reduced-motion: reduce) {
    .devices-panel {
      animation: none;
    }
  }
</style>
