// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { isLocalMusicId } from "$lib/music/library-adapter";
import { ApiPaths } from "$lib/core/http/api-paths";
import { parseJson } from "$lib/core/http/parse";
import { isRemoteClient, resolveApiUrl } from "$lib/config/remote-server";
import { extensionFeatures } from "./features.svelte";
import { extensionsPayloadSchema } from "./schemas";
import { runExtensionScript } from "./sandbox";
import { syncExtensionStyles } from "./styles";
import type {
  DecorateTrackContext,
  ExtensionManifest,
  TrackDecoration,
  TrackRule,
} from "./types";

const declarativeRules: TrackRule[] = [];

function contains(haystack: string | undefined, needle: string | undefined) {
  if (!needle) return true;
  return (haystack ?? "").toLowerCase().includes(needle.toLowerCase());
}

// Reject regexes that can run away on backtracking: overly long
// patterns and the classic quantified-group-with-inner-quantifier
// shape such as (x+)+ or ([a-z]+)*. Not a proof of safety, but it
// removes the catastrophic case cheaply.
const MAX_RULE_REGEX_LENGTH = 200;
const NESTED_QUANTIFIER = /\([^)]*[+*][^)]*\)[+*?{]/;

export function isSafeRuleRegex(pattern: string): boolean {
  return (
    pattern.length <= MAX_RULE_REGEX_LENGTH && !NESTED_QUANTIFIER.test(pattern)
  );
}

function matchesRule(ctx: DecorateTrackContext, match: TrackRule["match"]) {
  if (match.isLocal != null && match.isLocal !== ctx.isLocal) return false;
  if (!contains(ctx.track.title, match.titleContains)) return false;
  if (!contains(ctx.track.artist, match.artistContains)) return false;
  if (!contains(ctx.track.album, match.albumContains)) return false;
  const genre = ctx.genre ?? ctx.track.genre;
  if (!contains(genre, match.genreContains)) return false;
  if (match.titleRegex) {
    if (!isSafeRuleRegex(match.titleRegex)) return false;
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

// Decoration URLs feed <img src> and CSS url(), so anything remote
// would leak listening data and give extension code an egress channel
// the sandbox is designed to deny. Only same-origin paths are kept
// (protocol-relative //host is egress too and is rejected).
const SAME_ORIGIN_PATH = /^\/(?!\/)/;

export function sanitizeDecoration(
  decoration: TrackDecoration,
): TrackDecoration {
  const clean = { ...decoration };
  if (clean.progressThumbUrl && !SAME_ORIGIN_PATH.test(clean.progressThumbUrl))
    delete clean.progressThumbUrl;
  if (
    clean.progressParticleUrl &&
    !SAME_ORIGIN_PATH.test(clean.progressParticleUrl)
  )
    delete clean.progressParticleUrl;
  if (clean.iconUrl && !SAME_ORIGIN_PATH.test(clean.iconUrl))
    delete clean.iconUrl;
  return clean;
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
  return sanitizeDecoration(decoration);
}

async function loadScriptExtension(
  manifest: ExtensionManifest,
  settings: Record<string, unknown>,
) {
  if (!manifest.script) return;
  const response = await fetch(
    resolveApiUrl(ApiPaths.extensionScript(manifest.id)),
    {
      credentials: isRemoteClient() ? "include" : "same-origin",
    },
  );
  if (!response.ok) return;
  const source = await response.text();
  // The static audit stays as a first-pass signal (it drives the
  // scriptSafe flag in the UI) but it is not the boundary. Execution
  // happens in a null-origin sandboxed iframe, which is the boundary.
  if (
    /\b(import|require\s*\(|fetch\s*\(|eval\s*\(|Function\s*\(|window\b|document\b)\b/i.test(
      source,
    )
  ) {
    return;
  }
  const result = await runExtensionScript(
    source,
    settings,
    declarativeRules.slice(),
  );
  for (const rule of result.rules) {
    declarativeRules.push(rule);
  }
}

export async function loadExtensions() {
  const response = await fetch(resolveApiUrl(ApiPaths.extensions), {
    credentials: isRemoteClient() ? "include" : "same-origin",
  });
  if (!response.ok) {
    return;
  }
  const payload = await parseJson(
    extensionsPayloadSchema,
    response,
    "extensions",
  );
  extensionFeatures.applyFromItems(payload.items ?? []);
  declarativeRules.length = 0;
  const settingsById = new Map<string, Record<string, unknown>>();
  for (const item of payload.items ?? []) {
    if (item.settings) settingsById.set(item.id, item.settings);
  }
  const manifests = payload.manifests ?? [];
  for (const manifest of manifests) {
    for (const rule of manifest.trackRules ?? []) {
      declarativeRules.push(rule);
    }
    await loadScriptExtension(manifest, settingsById.get(manifest.id) ?? {});
  }
  extensionFeatures.applyAppThemeFromManifests(manifests);
  syncExtensionStyles(manifests);
}

export function resetExtensionsForTests() {
  declarativeRules.length = 0;
  extensionFeatures.resetForTests([]);
  syncExtensionStyles([]);
}

export function registerTrackRuleForTests(rule: TrackRule) {
  declarativeRules.push(rule);
}
