// Regenerate src/lib/components/ui/icon-data.ts from icon names used in the
// codebase (the ICONS map in MdiIcon.svelte plus icon="prefix:name" literals)
// and the local @iconify/json sets. Run from frontend/:
//   node scripts/gen-icon-data.cjs
const { readFileSync, readdirSync, writeFileSync } = require("node:fs");
const { join } = require("node:path");

const mdiSrc = readFileSync("src/lib/components/ui/MdiIcon.svelte", "utf8");
const match = mdiSrc.match(
  /const ICONS: Record<string, string> = \{([\s\S]*?)\};/,
);
if (!match) throw new Error("ICONS map not found in MdiIcon.svelte");

const names = new Set(
  [...match[1].matchAll(/(\w+):\s*"([^"]+)"/g)].map((m) => m[2]),
);

// Scan Svelte sources for Icon icon="prefix:name" string literals.
const stack = ["src"];
while (stack.length) {
  const dir = stack.pop();
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const entryPath = join(dir, entry.name);
    if (entry.isDirectory()) stack.push(entryPath);
    else if (entry.name.endsWith(".svelte")) {
      for (const m of readFileSync(entryPath, "utf8").matchAll(
        /icon="([a-z0-9-]+:[a-z0-9-]+)"|iconData\[["']([a-z0-9-]+:[a-z0-9-]+)["']\]/g,
      )) {
        names.add(m[1] ?? m[2]);
      }
    }
  }
}

const sets = {};
const missing = [];
const data = {};
for (const value of [...names].sort()) {
  const [prefix, name] = value.split(":");
  sets[prefix] ??= JSON.parse(
    readFileSync(`node_modules/@iconify/json/json/${prefix}.json`, "utf8"),
  );
  const set = sets[prefix];
  const icon = set.icons[name];
  if (!icon) {
    missing.push(value);
    continue;
  }
  data[value] = {
    body: icon.body,
    width: icon.width ?? set.width,
    height: icon.height ?? set.height,
    ...(icon.top ? { top: icon.top } : {}),
    ...(icon.left ? { left: icon.left } : {}),
    ...(icon.rotate ? { rotate: icon.rotate } : {}),
    ...(icon.hFlip ? { hFlip: icon.hFlip } : {}),
    ...(icon.vFlip ? { vFlip: icon.vFlip } : {}),
  };
}

// Icons that exist only as the opposite direction in their set.
const flipped = { "mdi:resize-bottom-left": "mdi:resize-bottom-right" };
for (const value of missing.slice()) {
  const source = flipped[value];
  if (!source) continue;
  const [prefix, name] = source.split(":");
  const set = sets[prefix];
  const icon = set.icons[name];
  if (!icon) continue;
  data[value] = {
    body: icon.body,
    width: icon.width ?? set.width,
    height: icon.height ?? set.height,
    hFlip: true,
  };
  missing.splice(missing.indexOf(value), 1);
}
if (missing.length) {
  console.warn("icons without local data (runtime API fallback):", missing); // skipcq: JS-0002
}

writeFileSync(
  "src/lib/components/ui/icon-data.ts",
  "// Generated from @iconify/json sets. Regenerate with scripts/gen-icon-data.cjs.\n" +
    'import type { IconifyIcon } from "@iconify/svelte";\n\n' +
    `export const iconData: Record<string, IconifyIcon> = ${JSON.stringify(data, null, 1)};\n`,
);
console.log(`wrote ${Object.keys(data).length} icons`); // skipcq: JS-0002
