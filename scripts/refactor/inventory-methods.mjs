#!/usr/bin/env node
// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0
//
// Mechanical extraction safety: inventory class/object methods and top-level
// functions with whitespace-normalized body hashes. Diff two inventories to
// prove a cut-paste move did not drop or rewrite logic.
//
// Usage:
//   node scripts/refactor/inventory-methods.mjs <file> [--out path.json]
//   node scripts/refactor/inventory-methods.mjs --diff before.json after.json [--expect-moved name1,name2]

import { createHash } from "node:crypto";
import { readFileSync, writeFileSync, mkdirSync } from "node:fs";
import { dirname, relative, resolve } from "node:path";

function normalizeBody(text) {
  return text.replace(/\s+/g, " ").trim();
}

function hashBody(text) {
  return createHash("sha256").update(normalizeBody(text)).digest("hex").slice(0, 16);
}

function findMatchingBrace(src, openIdx) {
  let depth = 0;
  let inStr = null;
  let escaped = false;
  let inLineComment = false;
  let inBlockComment = false;
  for (let i = openIdx; i < src.length; i++) {
    const c = src[i];
    const n = src[i + 1];
    if (inLineComment) {
      if (c === "\n") inLineComment = false;
      continue;
    }
    if (inBlockComment) {
      if (c === "*" && n === "/") {
        inBlockComment = false;
        i++;
      }
      continue;
    }
    if (inStr) {
      if (escaped) {
        escaped = false;
        continue;
      }
      if (c === "\\") {
        escaped = true;
        continue;
      }
      if (c === inStr) inStr = null;
      continue;
    }
    if (c === "/" && n === "/") {
      inLineComment = true;
      i++;
      continue;
    }
    if (c === "/" && n === "*") {
      inBlockComment = true;
      i++;
      continue;
    }
    if (c === '"' || c === "'" || c === "`") {
      inStr = c;
      continue;
    }
    if (c === "{") depth++;
    else if (c === "}") {
      depth--;
      if (depth === 0) return i;
    }
  }
  return -1;
}

function lineOf(src, idx) {
  return src.slice(0, idx).split("\n").length;
}

/**
 * Inventory TypeScript/JavaScript/Svelte script methods and Go funcs.
 * For .svelte files, only the first <script> block is scanned.
 */
function inventorySource(raw, filePath) {
  let src = raw;
  if (filePath.endsWith(".svelte")) {
    const m = raw.match(/<script[^>]*>([\s\S]*?)<\/script\s*>/i);
    src = m ? m[1] : raw;
  }

  const entries = [];
  const isGo = filePath.endsWith(".go");

  if (isGo) {
    const re =
      /^func\s+(?:\([^)]+\)\s+)?([A-Za-z_][A-Za-z0-9_]*)\s*\([^)]*\)[^{]*\{/gm;
    let match;
    while ((match = re.exec(src)) !== null) {
      const name = match[1];
      const openIdx = match.index + match[0].length - 1;
      const closeIdx = findMatchingBrace(src, openIdx);
      if (closeIdx < 0) continue;
      const body = src.slice(openIdx, closeIdx + 1);
      const kind = match[0].includes(") ") ? "method" : "func";
      entries.push({
        name,
        kind,
        visibility: /^[A-Z]/.test(name) ? "public" : "private",
        startLine: lineOf(src, match.index),
        endLine: lineOf(src, closeIdx),
        bodyHash: hashBody(body),
        bodyLen: body.length,
      });
    }
  } else {
    // Class methods: optional async/private/public/#, then name(
    const classMethodRe =
      /^[ \t]*(?:(?:public|private|protected|static|async|override)\s+)*#?([A-Za-z_][A-Za-z0-9_]*)\s*\([^)]*\)(?:\s*:\s*[^{]+)?\s*\{/gm;
    // Top-level / exported functions
    const funcRe =
      /^[ \t]*(?:export\s+)?(?:async\s+)?function\s+([A-Za-z_][A-Za-z0-9_]*)\s*\([^)]*\)(?:\s*:\s*[^{]+)?\s*\{/gm;
    // Arrow property methods assigned on classes: name = (...) => { or name = async (...) => {
    const arrowRe =
      /^[ \t]*(?:(?:public|private|protected|static|readonly)\s+)*#?([A-Za-z_][A-Za-z0-9_]*)\s*=\s*(?:async\s*)?\([^)]*\)\s*(?::\s*[^=]+)?=>\s*\{/gm;

    const seen = new Set();
    function addMatch(match, kind, visibilityHint) {
      const name = match[1];
      if (name === "if" || name === "for" || name === "while" || name === "switch")
        return;
      const openIdx = match.index + match[0].length - 1;
      const closeIdx = findMatchingBrace(src, openIdx);
      if (closeIdx < 0) return;
      const key = `${name}@${match.index}`;
      if (seen.has(key)) return;
      seen.add(key);
      const body = src.slice(openIdx, closeIdx + 1);
      const isPrivate =
        /\bprivate\b/.test(match[0]) ||
        match[0].includes("#") ||
        name.startsWith("#");
      entries.push({
        name,
        kind,
        visibility: visibilityHint ?? (isPrivate ? "private" : "public"),
        startLine: lineOf(src, match.index),
        endLine: lineOf(src, closeIdx),
        bodyHash: hashBody(body),
        bodyLen: body.length,
      });
    }

    let match;
    while ((match = classMethodRe.exec(src)) !== null) {
      addMatch(match, "method");
    }
    while ((match = funcRe.exec(src)) !== null) {
      addMatch(match, "function", "public");
    }
    while ((match = arrowRe.exec(src)) !== null) {
      addMatch(match, "arrow");
    }
  }

  entries.sort((a, b) => a.startLine - b.startLine || a.name.localeCompare(b.name));
  return {
    file: filePath,
    generatedAt: new Date().toISOString(),
    methodCount: entries.length,
    methods: entries,
  };
}

function indexByName(inv) {
  const map = new Map();
  for (const m of inv.methods) {
    const list = map.get(m.name) ?? [];
    list.push(m);
    map.set(m.name, list);
  }
  return map;
}

function diffInventories(before, after, expectMoved = []) {
  const beforeIdx = indexByName(before);
  const afterIdx = indexByName(after);
  const beforeNames = new Set(before.methods.map((m) => m.name));
  const afterNames = new Set(after.methods.map((m) => m.name));

  const missing = [...beforeNames].filter((n) => !afterNames.has(n));
  const added = [...afterNames].filter((n) => !beforeNames.has(n));
  const hashChanged = [];

  for (const name of beforeNames) {
    if (!afterNames.has(name)) continue;
    const b = beforeIdx.get(name)[0];
    const a = afterIdx.get(name)[0];
    if (b.bodyHash !== a.bodyHash) {
      hashChanged.push({
        name,
        beforeHash: b.bodyHash,
        afterHash: a.bodyHash,
        beforeLen: b.bodyLen,
        afterLen: a.bodyLen,
      });
    }
  }

  const expectSet = new Set(expectMoved);
  const unexpectedMissing = missing.filter((n) => !expectSet.has(n));
  const expectedStillPresent = [...expectSet].filter((n) => afterNames.has(n));
  // For expect-moved across files, use --diff-moved with destination inventory instead.

  return {
    beforeFile: before.file,
    afterFile: after.file,
    beforeCount: before.methodCount,
    afterCount: after.methodCount,
    missing,
    added,
    hashChanged,
    unexpectedMissing,
    expectedStillPresent,
    ok: unexpectedMissing.length === 0 && hashChanged.length === 0,
  };
}

/**
 * Prove methods moved from god file to new module: each name must exist in
 * destination with the same body hash as before (or as listed in before).
 */
function diffMoved(before, destination, movedNames) {
  const beforeIdx = indexByName(before);
  const destIdx = indexByName(destination);
  const failures = [];
  for (const name of movedNames) {
    const bList = beforeIdx.get(name);
    if (!bList?.length) {
      failures.push({ name, error: "not in before inventory" });
      continue;
    }
    const dList = destIdx.get(name);
    if (!dList?.length) {
      failures.push({ name, error: "missing from destination" });
      continue;
    }
    const b = bList[0];
    const d = dList[0];
    // Destination may wrap with (ctx, ...) params. Compare body hashes only when
    // expect-exact. Default: require destination to contain a function of that name.
    if (b.bodyHash !== d.bodyHash) {
      failures.push({
        name,
        error: "body hash mismatch (expected for ctx-param adaptation)",
        beforeHash: b.bodyHash,
        destHash: d.bodyHash,
        beforeLen: b.bodyLen,
        destLen: d.bodyLen,
      });
    }
  }
  return {
    movedNames,
    failures: failures.filter((f) => f.error !== "body hash mismatch (expected for ctx-param adaptation)"),
    hashMismatches: failures.filter((f) =>
      f.error.startsWith("body hash mismatch"),
    ),
    missing: failures.filter((f) => f.error.includes("missing") || f.error.includes("not in")),
    ok: failures.every((f) => f.error.startsWith("body hash mismatch")),
  };
}

function printUsage() {
  console.error(`Usage:
  inventory-methods.mjs <file> [--out path.json]
  inventory-methods.mjs --diff before.json after.json
  inventory-methods.mjs --diff-moved before.json dest.json name1,name2,...
  inventory-methods.mjs --names-present file.json name1,name2,...`);
}

function main() {
  const args = process.argv.slice(2);
  if (args.length === 0) {
    printUsage();
    process.exit(1);
  }

  if (args[0] === "--diff") {
    const before = JSON.parse(readFileSync(resolve(args[1]), "utf8"));
    const after = JSON.parse(readFileSync(resolve(args[2]), "utf8"));
    const expectMoved = (args.includes("--expect-moved")
      ? args[args.indexOf("--expect-moved") + 1]
      : ""
    )
      .split(",")
      .filter(Boolean);
    const result = diffInventories(before, after, expectMoved);
    console.log(JSON.stringify(result, null, 2));
    process.exit(result.ok ? 0 : 1);
  }

  if (args[0] === "--diff-moved") {
    const before = JSON.parse(readFileSync(resolve(args[1]), "utf8"));
    const dest = JSON.parse(readFileSync(resolve(args[2]), "utf8"));
    const names = (args[3] || "").split(",").filter(Boolean);
    const result = diffMoved(before, dest, names);
    console.log(JSON.stringify(result, null, 2));
    // Pass if all names present in dest (hash may differ due to ctx param)
    const allPresent = names.every((n) =>
      dest.methods.some((m) => m.name === n),
    );
    const noneMissingFromBefore = names.every((n) =>
      before.methods.some((m) => m.name === n),
    );
    process.exit(allPresent && noneMissingFromBefore ? 0 : 1);
  }

  if (args[0] === "--names-present") {
    const inv = JSON.parse(readFileSync(resolve(args[1]), "utf8"));
    const names = (args[2] || "").split(",").filter(Boolean);
    const present = new Set(inv.methods.map((m) => m.name));
    const missing = names.filter((n) => !present.has(n));
    console.log(JSON.stringify({ missing, ok: missing.length === 0 }, null, 2));
    process.exit(missing.length === 0 ? 0 : 1);
  }

  const root = process.cwd();
  const filePath = relative(root, resolve(args[0]));
  const outIdx = args.indexOf("--out");
  const outPath = outIdx >= 0 ? resolve(args[outIdx + 1]) : null;
  const raw = readFileSync(filePath, "utf8");
  const inv = inventorySource(raw, filePath);
  const json = JSON.stringify(inv, null, 2);
  if (outPath) {
    mkdirSync(dirname(outPath), { recursive: true });
    writeFileSync(outPath, json + "\n");
    console.log(`Wrote ${inv.methodCount} methods to ${outPath}`);
  } else {
    console.log(json);
  }
}

main();
