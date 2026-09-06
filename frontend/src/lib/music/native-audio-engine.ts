// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import type { PlaybackEngine } from "./playback-engine";
import type { EqSettings } from "./eq";
import {
  immersiveAudioToNativeConfig,
  type ImmersiveAudioSettings,
} from "./immersive-audio-settings";

type Handler = () => void;

export const LOAD_SUPERSEDED = "load superseded";

type AudioServiceModule = typeof import("@bindings/melovian/services/index.js");
type AudioServiceType = AudioServiceModule["AudioService"];

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

export class NativeAudioEngine implements PlaybackEngine {
  readonly usesNative = true;

  /** LOAD_ERROR_WINDOW_MS bounds how long a load waits for an early error. */
  private static readonly LOAD_ERROR_WINDOW_MS = 15000;
  /**
   * LOAD_SETTLE_MS is how long to wait for a duration before proceeding to
   * playback anyway. Some backends (libvlc) cannot report duration until
   * playback actually starts, so blocking on it would stall every track.
   */
  private static readonly LOAD_SETTLE_MS = 1200;
  /** STALL_MS marks playback as failed if position does not advance while playing. */
  private static readonly STALL_MS = 7000;

  private currentVolume = 1;
  private loadedUrl = "";
  private _currentTime = 0;
  private _duration = 0;
  private _lastError = "";
  private pollTimer = 0;
  private pollInFlight = false;
  private loadGeneration = 0;
  private loadChain: Promise<void> = Promise.resolve();
  private transportPaused = false;
  private lastAdvanceAt = 0;
  private serviceReady: Promise<AudioServiceType>;
  private service: AudioServiceType | null = null;

  private endedHandlers = new Set<Handler>();
  private timeHandlers = new Set<Handler>();
  private metaHandlers = new Set<Handler>();
  private errorHandlers = new Set<Handler>();

  constructor() {
    this.serviceReady = import("@bindings/melovian/services/index.js").then(
      (mod) => {
        this.service = mod.AudioService;
        return mod.AudioService;
      },
    );
  }

  private callService(
    fn: (svc: AudioServiceType) => void | Promise<void>,
  ): void {
    if (this.service) {
      void fn(this.service);
      return;
    }
    void this.serviceReady.then(fn);
  }

  private async getService(): Promise<AudioServiceType> {
    if (this.service) return this.service;
    return await this.serviceReady;
  }

  private stopPolling() {
    if (this.pollTimer) {
      clearInterval(this.pollTimer);
      this.pollTimer = 0;
    }
    this.pollInFlight = false;
  }

  private startPolling(AudioService: AudioServiceType) {
    this.stopPolling();
    this.pollTimer = window.setInterval(() => {
      if (this.transportPaused || this.pollInFlight) return;
      this.pollInFlight = true;
      void AudioService.GetState()
        .then((state) => {
          if (this.transportPaused) return;
          const prevTime = this._currentTime;
          this._currentTime = state.currentTime;
          if (state.duration > 0) {
            this._duration = state.duration;
          }

          if (Math.abs(this._currentTime - prevTime) > 0.02) {
            this.lastAdvanceAt = Date.now();
            this.timeHandlers.forEach((handler) => handler());
          }

          if (state.ended) {
            void AudioService.ClearEnded();
            this.endedHandlers.forEach((handler) => handler());
            return;
          }
          if (state.error) {
            this._lastError = state.error;
            this.errorHandlers.forEach((handler) => handler());
            return;
          }
          // Playback that never advances is treated as a failure so the caller
          // can fall back to another backend instead of sitting silent.
          if (
            this.lastAdvanceAt > 0 &&
            Date.now() - this.lastAdvanceAt > NativeAudioEngine.STALL_MS
          ) {
            this._lastError =
              this._lastError || "native playback stalled with no audio output";
            this.errorHandlers.forEach((handler) => handler());
            this.stopPolling();
          }
        })
        .finally(() => {
          this.pollInFlight = false;
        });
    }, 500);
  }

  async loadSource(url: string): Promise<void> {
    if (this.loadedUrl === url && this.canPlay()) {
      return;
    }

    const generation = ++this.loadGeneration;
    this.stopPolling();

    let loadError: Error | null = null;
    this.loadChain = this.loadChain.then(async () => {
      if (generation !== this.loadGeneration) return;
      try {
        await this.loadSourceInternal(url, generation);
      } catch (err) {
        if (generation !== this.loadGeneration) return;
        loadError = err instanceof Error ? err : new Error(String(err));
      }
    });
    await this.loadChain;
    if (loadError) throw loadError;
  }

  private async loadSourceInternal(
    url: string,
    generation: number,
  ): Promise<void> {
    const AudioService = await this.getService();
    if (generation !== this.loadGeneration) {
      throw new Error(LOAD_SUPERSEDED);
    }

    this._lastError = "";
    this._currentTime = 0;
    this._duration = 0;
    await AudioService.Pause();
    if (generation !== this.loadGeneration) {
      throw new Error(LOAD_SUPERSEDED);
    }
    await AudioService.LoadURL(url);
    if (generation !== this.loadGeneration) {
      throw new Error(LOAD_SUPERSEDED);
    }
    await AudioService.SetVolume(this.currentVolume);

    const started = Date.now();
    const deadline = started + NativeAudioEngine.LOAD_ERROR_WINDOW_MS;
    while (Date.now() < deadline) {
      if (generation !== this.loadGeneration) {
        throw new Error(LOAD_SUPERSEDED);
      }
      const state = await AudioService.GetState();
      if (state.error) {
        this._lastError = state.error;
        throw new Error(state.error);
      }
      if (state.duration > 0) {
        this._duration = state.duration;
        this._currentTime = state.currentTime;
        this.loadedUrl = url;
        this.metaHandlers.forEach((handler) => handler());
        return;
      }
      if (Date.now() - started >= NativeAudioEngine.LOAD_SETTLE_MS) {
        this.loadedUrl = url;
        this.metaHandlers.forEach((handler) => handler());
        return;
      }
      await sleep(100);
    }

    this.loadedUrl = url;
    this.metaHandlers.forEach((handler) => handler());
  }

  private preparedUrl = "";

  prepareNext(url: string): void {
    if (!url || url === this.loadedUrl) {
      this.preparedUrl = "";
      this.callService((svc) => {
        if (typeof svc.ClearPrepared === "function") {
          void svc.ClearPrepared();
        }
      });
      return;
    }
    if (this.preparedUrl === url) return;
    this.preparedUrl = url;
    this.callService((svc) => {
      if (typeof svc.PrepareURL !== "function") {
        this.preparedUrl = "";
        return;
      }
      void svc.PrepareURL(url).catch(() => {
        if (this.preparedUrl === url) this.preparedUrl = "";
      });
    });
  }

  hasPrepared(url: string): boolean {
    return url !== "" && this.preparedUrl === url;
  }

  async activatePrepared(url: string, crossfadeSec = 0): Promise<boolean> {
    if (!this.hasPrepared(url)) return false;
    const AudioService = await this.getService();
    if (typeof AudioService.ActivatePrepared !== "function") {
      this.preparedUrl = "";
      return false;
    }
    try {
      if (typeof AudioService.HasPrepared === "function") {
        const ready = await AudioService.HasPrepared(url);
        if (!ready) {
          this.preparedUrl = "";
          return false;
        }
      }
      await AudioService.ActivatePrepared(crossfadeSec);
      this.loadedUrl = url;
      this.preparedUrl = "";
      this.transportPaused = false;
      this.lastAdvanceAt = Date.now();
      this.startPolling(AudioService);
      this.metaHandlers.forEach((handler) => handler());
      return true;
    } catch {
      this.preparedUrl = "";
      return false;
    }
  }

  prefetch(_urls: string[]): void {}

  async play(): Promise<void> {
    this.transportPaused = false;
    this.lastAdvanceAt = Date.now();
    const AudioService = await this.getService();
    await AudioService.Play();
    this.startPolling(AudioService);
  }

  pause(): void {
    this.transportPaused = true;
    this.stopPolling();
    this.callService((svc) => svc.Pause());
  }

  get currentTime(): number {
    return this._currentTime;
  }

  get duration(): number {
    return this._duration;
  }

  set currentTime(value: number) {
    this._currentTime = value;
    this.callService((svc) => svc.Seek(value));
  }

  setVolume(value: number): void {
    this.currentVolume = value;
    this.callService((svc) => svc.SetVolume(value));
  }

  applyEq(_settings: EqSettings): void {}

  applyImmersiveAudio(settings: ImmersiveAudioSettings): void {
    const config = immersiveAudioToNativeConfig(settings);
    this.callService((svc) => {
      if (typeof svc.SetAudioOutputConfig === "function") {
        void svc.SetAudioOutputConfig(config);
      }
    });
  }

  canPlay(): boolean {
    return this.loadedUrl !== "";
  }

  isNetworkError(): boolean {
    return false;
  }

  isDecodeError(): boolean {
    return false;
  }

  getLastError(): string {
    return this._lastError;
  }

  onEnded(handler: Handler): () => void {
    this.endedHandlers.add(handler);
    return () => this.endedHandlers.delete(handler);
  }

  onTimeUpdate(handler: Handler): () => void {
    this.timeHandlers.add(handler);
    return () => this.timeHandlers.delete(handler);
  }

  onLoadedMetadata(handler: Handler): () => void {
    this.metaHandlers.add(handler);
    return () => this.metaHandlers.delete(handler);
  }

  onError(handler: Handler): () => void {
    this.errorHandlers.add(handler);
    return () => this.errorHandlers.delete(handler);
  }

  destroy(): void {
    this.loadGeneration += 1;
    this.stopPolling();
    this.pause();
    this.loadedUrl = "";
    this.preparedUrl = "";
    this.callService((svc) => {
      if (typeof svc.ClearPrepared === "function") {
        void svc.ClearPrepared();
      }
    });
    this.endedHandlers.clear();
    this.timeHandlers.clear();
    this.metaHandlers.clear();
    this.errorHandlers.clear();
  }
}
