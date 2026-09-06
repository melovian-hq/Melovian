// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import type { EqSettings } from "./eq";
import type { ImmersiveAudioSettings } from "./immersive-audio-settings";

type Handler = () => void;

export interface PlaybackEngine {
  readonly usesNative: boolean;
  loadSource(url: string): Promise<void>;
  prepareNext(url: string): void;
  hasPrepared(url: string): boolean;
  activatePrepared(url: string, crossfadeSec?: number): Promise<boolean>;
  prefetch(urls: string[]): void;
  play(): Promise<void>;
  pause(): void;
  get currentTime(): number;
  get duration(): number;
  set currentTime(value: number);
  setVolume(value: number): void;
  applyEq(settings: EqSettings): void;
  applyImmersiveAudio(settings: ImmersiveAudioSettings): void;
  canPlay(): boolean;
  isNetworkError(): boolean;
  isDecodeError(): boolean;
  /** getLastError returns the most recent playback error message, if any. */
  getLastError(): string;
  onEnded(handler: Handler): () => void;
  onTimeUpdate(handler: Handler): () => void;
  onLoadedMetadata(handler: Handler): () => void;
  onError(handler: Handler): () => void;
  destroy(): void;
}

export type { Handler as PlaybackHandler };
