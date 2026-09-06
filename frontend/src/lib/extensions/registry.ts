// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { isLocalMusicId } from "$lib/music/library-adapter";
import { isRemoteClient, resolveApiUrl } from "$lib/config/remote-server";
import { extensionFeatures } from "./features.svelte";
import { syncExtensionStyles } from "./styles";
import type {
  DecorateTrackContext,
  ExtensionAPI,
  ExtensionManifest,
  TrackDecoration,
  TrackRule,
} from "./types";
import type { ExtensionListItem } from "./api";

type ScriptHook = (api: ExtensionAPI, ctx: DecorateTrackContext) => void;

const declarativeRules: TrackRule[] = [];
const scriptDecorators: Array<
  (ctx: DecorateTrackContext) => TrackDecoration | null | undefined
> = [];

function contains(haystack: string | undefined, needle: string | undefined) {
  if (!needle) return true;
  return (haystack ?? "").toLowerCase().includes(needle.toLowerCase());
}

function matchesRule(ctx: DecorateTrackContext, match: TrackRule["match"]) {
  if (match.isLocal != null && match.isLocal !== ctx.isLocal) return false;
  if (!contains(ctx.track.title, match.titleContains)) return false;
  if (!contains(ctx.track.artist, match.artistContains)) return false;
  if (!contains(ctx.track.album, match.albumContains)) return false;
  const genre = ctx.genre ?? ctx.track.genre;
  if (!contains(genre, match.genreContains)) return false;
  if (match.titleRegex) {
    try {
      if (!new RegExp(match.titleRegex, "i").test(ctx.track.title ?? "")) {
        return false;
      }
    } catch {
      return false;
    }
  }
  if (match.tagEquals) {
    const tags = ctx.tags ?? [];
    if (
      !tags.some((tag) => tag.toLowerCase() === match.tagEquals!.toLowerCase())
    ) {
      return false;
    }
  }
  if (match.minRating != null && (ctx.rating ?? 0) < match.minRating)
    return false;
  return true;
}

function mergeDecorations(
  base: TrackDecoration,
  next: TrackDecoration,
): TrackDecoration {
  return {
    progressColor: next.progressColor ?? base.progressColor,
    progressGradient: next.progressGradient ?? base.progressGradient,
    progressThumbUrl: next.progressThumbUrl ?? base.progressThumbUrl,
    progressParticleUrl: next.progressParticleUrl ?? base.progressParticleUrl,
    icon: next.icon ?? base.icon,
    iconUrl: next.iconUrl ?? base.iconUrl,
    titlePrefix: next.titlePrefix ?? base.titlePrefix,
    coverOverlayIcon: next.coverOverlayIcon ?? base.coverOverlayIcon,
    playerTheme: next.playerTheme ?? base.playerTheme,
  };
}

export function decorateTrack(
  track: DecorateTrackContext["track"],
  extra: Omit<DecorateTrackContext, "track" | "isLocal"> = {},
): TrackDecoration {
  const ctx: DecorateTrackContext = {
    track,
    isLocal: isLocalMusicId(track.id),
    ...extra,
  };
  let decoration: TrackDecoration = {};
  for (const rule of declarativeRules) {
    if (matchesRule(ctx, rule.match)) {
      decoration = mergeDecorations(decoration, rule.decoration);
    }
  }
  for (const hook of scriptDecorators) {
    const next = hook(ctx);
    if (next) {
      decoration = mergeDecorations(decoration, next);
    }
  }
  return decoration;
}

function createExtensionAPI(): ExtensionAPI {
  return {
    registerTrackRule(rule) {
      declarativeRules.push(rule);
    },
    decorateTrack(ctx) {
      for (const rule of declarativeRules) {
        if (matchesRule(ctx, rule.match)) {
          return rule.decoration;
        }
      }
      return null;
    },
  };
}

async function loadScriptExtension(manifest: ExtensionManifest) {
  if (!manifest.script) return;
  const response = await fetch(
    resolveApiUrl(`/api/extensions/${manifest.id}/script`),
    {
      credentials: isRemoteClient() ? "include" : "same-origin",
    },
  );
  if (!response.ok) return;
  const source = await response.text();
  if (
    /\b(import|require\s*\(|fetch\s*\(|eval\s*\(|Function\s*\(|window\b|document\b)\b/i.test(
      source,
    )
  ) {
    return;
  }
  const api = createExtensionAPI();
  const runner = new Function(
    "api",
    "register",
    `"use strict";\n${source}\n;return typeof register === 'function' ? register(api) : undefined;`,
  ) as (api: ExtensionAPI, register: ScriptHook) => unknown;
  runner(api, (extensionApi, ctx) => {
    const result = extensionApi.decorateTrack(ctx);
    if (result) {
      scriptDecorators.push(() => result);
    }
  });
}

export async function loadExtensions() {
  const response = await fetch(resolveApiUrl("/api/extensions"), {
    credentials: isRemoteClient() ? "include" : "same-origin",
  });
  if (!response.ok) {
    return;
  }
  const payload = (await response.json()) as {
    items?: ExtensionListItem[];
    manifests?: ExtensionManifest[];
  };
  extensionFeatures.applyFromItems(payload.items ?? []);
  declarativeRules.length = 0;
  scriptDecorators.length = 0;
  const manifests = payload.manifests ?? [];
  for (const manifest of manifests) {
    for (const rule of manifest.trackRules ?? []) {
      declarativeRules.push(rule);
    }
    await loadScriptExtension(manifest);
  }
  extensionFeatures.applyAppThemeFromManifests(manifests);
  syncExtensionStyles(manifests);
}

export function resetExtensionsForTests() {
  declarativeRules.length = 0;
  scriptDecorators.length = 0;
  extensionFeatures.resetForTests([]);
  syncExtensionStyles([]);
}

export function registerTrackRuleForTests(rule: TrackRule) {
  declarativeRules.push(rule);
}
