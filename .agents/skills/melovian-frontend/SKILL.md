---
name: melovian-frontend
description: Syntax and API rules for the Melovian frontend stack (Svelte 5 runes, Tailwind v4, bits-ui v2, runed, Vite 8, Vitest 4, TypeScript 6, ESLint 10). Use when writing or editing any frontend code.
---

# Melovian frontend stack

Exact versions live in `frontend/package.json`. This file records the syntax rules that differ from older major versions, so code written here compiles on the first try.

## Svelte 5 (runes only)

New code uses runes. Never mix in Svelte 4 patterns.

```svelte
<script lang="ts">
  let count = $state(0);
  let double = $derived(count * 2);
  let { title, onclose }: { title: string; onclose?: () => void } = $props();

  $effect(() => {
    console.log(count);
    return () => cleanup();
  });
</script>
```

- Props come from `$props()`, not `export let`. A prop only supports `bind:` from the parent if declared `$bindable()`.
- `export let x = $derived(...)` is a compile error in runes mode.
- Content projection uses snippets, not slots: `{#snippet row(item)}...{/snippet}` in the child, `{@render row(x)}` or `children` snippet prop to render.
- Generic snippets are allowed: `{#snippet foo<T>(x: T)}`.
- Attachments `{@attach fn}` replace most `use:` actions. They run in an effect, re-run when reactive deps change, and may return a cleanup.
- `<svelte:boundary>` catches render/effect errors and supports `pending` and `failed` snippets plus `onerror`. Top-level `await` in markup needs `compilerOptions.experimental.async`, which this repo does not enable. Keep async work in `.svelte.ts` modules and effects instead.
- Reactivity outside components lives in `.svelte.ts` files (`music.svelte.ts`, `toast.svelte.ts`, etc.). Plain `.ts` files cannot use runes.
- Reactive state classes from `svelte/reactivity` (`SvelteMap`, `SvelteSet`, `SvelteDate`) are available; `SvelteMap.getOrInsert`/`getOrInsertComputed` exist since 5.57.
- Preprocessing is `vitePreprocess()` in `svelte.config.js`. Under vite-plugin-svelte 7 + Vite 8 it uses Oxc, not esbuild.

## Tailwind CSS v4 (CSS-first)

There is no `tailwind.config.js`. Config lives in CSS.

- `frontend/src/app.css` starts with `@import "tailwindcss"` plus local imports. Do not write `@tailwind base/components/utilities`.
- Design tokens are plain CSS custom properties in `frontend/src/lib/theme/tokens.css` under `@layer theme` (`--jb-*` names, kept from an earlier brand). Utilities come from Tailwind defaults; spacing/colors are not redeclared in `@theme`.
- v3 to v4 syntax rules that matter when editing templates:
  - Important modifier moved to the end: `flex!` not `!flex`, `hover:bg-blue-500!` not `hover:!bg-blue-500`.
  - CSS var shorthand uses parentheses: `bg-(--jb-accent)` not `bg-[--jb-accent]`.
  - Arbitrary values with multiple parts use underscores: `grid-cols-[max-content_auto]`.
  - Stacked variant order is left to right: `*:first:pt-0` not `first:*:pt-0`.
  - Default `border-*` and `divide-*` color is `currentColor`, not `gray-200`. Default `ring` is 1px `currentColor`.
  - `space-y-*`/`space-x-*` now use `margin-bottom`/`margin-inline-end` on `> :not(:last-child)`; do not rely on them over `position: absolute` children.
- `@apply` inside a scoped Svelte `<style>` block needs `@reference "../../app.css"` at the top of the block, because each component style is a separate bundle.
- Automatic content detection covers `src/`. Add `@source` only for out-of-tree files; `@source inline("...")` replaces the v3 `safelist` option.
- New utilities available from 4.1+: `text-shadow-*`, `mask-*`, `pointer-*`/`any-pointer-*` variants, `items-baseline-last`, extra logical-property utilities (`pbs-*`, `pbe-*`, `mbs-*`, `mbe-*`, `inline-*`, `block-*`, `inset-s/e/bs/be-*`), `@container-size`, `zoom-*`, scrollbar utilities. (`ps-*`/`pe-*`/`ms-*`/`me-*`/`start-*`/`end-*` already existed in late v3.)
- The Vite plugin `@tailwindcss/vite` is registered in `frontend/vite.config.ts`; no PostCSS config exists.

## Semantic color tokens (the `--jb-*` contract)

`tokens.css` has two layers per theme: a private `--jb-raw-*` primitive ramp and the semantic `--jb-*` layer. Components may only consume the semantic layer. Never reference `--jb-raw-*` or write literal colors (hex, `rgb()`/`hsl()`/`oklch()`, named colors) in a `.svelte` file. `src/lib/theme/color-tokens.hygiene.test.ts` enforces this and fails on new literals outside its baseline allowlist.

- Themes: dark is the default, declared on both `:root` and `[data-theme="dark"]`. Light lives on `[data-theme="light"]`. `theme.svelte.ts` exposes `mode` (`"light" | "dark" | "system"`) and `resolved`. `applyThemeDataset` sets `data-theme` on `<html>` and the `theme` storage key persists the mode.
- Accent presets live in `lib/theme/accent.ts`: aurora (default), ocean, forest, amber, rose, graphite, plus `custom` driven by a hue slider. Non-default presets write `--jb-accent`/`--jb-accent-text` inline on `<html>`. The default preset applies no override so tokens.css keeps control. Storage keys: `theme-accent-preset`, `theme-accent-hue`. The legacy `theme-accent` hex key migrates once into the custom hue.
- Semantic groups: `--jb-bg{,-elevated,-muted,-subtle}`, `--jb-surface{,-raised,-2,-hover,-active}`, `--jb-border{,-strong}`, `--jb-text{,-muted,-subtle}`, `--jb-accent{,-hover,-muted,-text}`, `--jb-music-accent*`, `--jb-{success,warning,danger,info}` plus `*-muted` variants and `--jb-danger-hover`, `--jb-shadow-{sm,md,lg}`, `--jb-island-{bg,border}`, `--jb-pill-{bg,border}`, `--jb-focus-ring`.
- Scrims and over-art chrome are theme-aware too: `--jb-scrim{,-subtle,-strong}` for backdrops, and the overlay family `--jb-overlay-{text,text-muted,shadow,surface,surface-hover,border,chip,chip-hover}` plus `--jb-media-surface` for controls and text painted over artwork, video, or media surfaces. These stay dark/white regardless of theme on purpose. `--jb-favorite{,-hover}` covers star and heart accents.
- Z-index is tokenized: `--jb-z-topbar` (30) up through `--jb-z-toast` (200). Use the scale instead of literal z-index values.
- `transparent`, `currentColor`, `none`, and `color-mix()` built on token vars are always legal.

## Settings panels: auto-save and nav state

- Tab metadata lives in `lib/settings/tabs.ts`. `SETTINGS_TAB_IDS` mirrors `SETTINGS_TABS` order and each tab carries a `tier` of `"recommended"` or `"advanced"`. `visibleSettingsTabs` filters by extension features and server capabilities, `filterSettingsTabs` powers nav search.
- Panels auto-save instead of using a save button. Track each card with `SaveStatus` from `lib/settings/save-status.svelte.ts` (`createSaveStatus()`): call `begin()` before the write, then `saved()` or `failed(message)`. The `saved` state flashes for about 1.5s and returns to `idle`.
- Most saves should go through `saveWithStatus(status, failureMessage, work, onFailure)` in the same module: it runs begin/saved/failed around the work, accepts a fixed message or `(err) => string`, and passes the resolved message to `onFailure` for toasting. Only drop to manual begin/saved/failed when the flow needs mid-save checks (for example `applyNativeBackend` inspecting `music.nativeInitError`).
- Render the indicator with `components/settings/SettingsSaveStatus.svelte` inside the `status` snippet of `SettingsCard`.
- The Advanced nav group collapses via `lib/settings/nav-state.svelte.ts`, persisted at storage key `settings-nav-advanced-collapsed` and defaulting to collapsed. It force-opens while the active tab is advanced without writing the stored preference, so deep links are never hidden.
- `.settings-loading` in `lib/settings/settings-page.css` is the shared pending block for panels still fetching their settings.

## bits-ui v2

UI primitives in `frontend/src/lib/components/ui/` and dialogs/menus elsewhere are bits-ui. Compound components under a namespace:

```svelte
<script lang="ts">
  import { Dialog, Select, Tabs, Slider, Popover, ContextMenu, AlertDialog, Command, Switch } from "bits-ui";
  let open = $state(false);
</script>

<Dialog.Root bind:open>
  <Dialog.Trigger>Open</Dialog.Trigger>
  <Dialog.Portal>
    <Dialog.Overlay />
    <Dialog.Content>
      <Dialog.Title>Title</Dialog.Title>
      <Dialog.Description>Body</Dialog.Description>
    </Dialog.Content>
  </Dialog.Portal>
</Dialog.Root>
```

- State is bound on the `Root`: `bind:open`, `bind:value`, `bind:checked`, `bind:pressed`.
- `asChild` and `let:` directives are gone. To control the rendered element or add transitions, use the `child` snippet plus `forceMount`:

```svelte
<Dialog.Content forceMount>
  {#snippet child({ props, open })}
    {#if open}
      <div {...props} transition:fly={{ y: 8 }}>...</div>
    {/if}
  {/snippet}
</Dialog.Content>
```

- `children` snippets receive state like `{ selected }` on `Select.Item` or `{ thumbs, ticks }` on `Slider.Root`.
- `AlertDialog.Content` must sit inside `AlertDialog.Portal`. `AlertDialog.Action` does not close the dialog by itself; close explicitly.
- Components in use here: `Dialog`, `AlertDialog`, `Select`, `Tabs`, `Slider`, `Popover`, `ContextMenu`, `Command`, `Switch`. Prefer these over hand-rolled overlays so focus trap and aria wiring stay consistent.

## runed

Reactive utility library, pinned `0.37.1`. In use: `PersistedState` (layout, home banner) and `useDebounce` (command palette).

```ts
import { PersistedState, useDebounce, watch, useInterval } from "runed";

const collapsed = new PersistedState("sidebar-collapsed", false, {
  storage: "local",
  syncTabs: true,
});
collapsed.current = true;

const save = useDebounce(() => doSave(), () => 500);
save();                  // schedule
save.cancel();
save.runScheduledNow();
```

- Classes expose `.current` and work in `.svelte.ts` files.
- `useInterval` (not a removed `Interval` class) takes a callback and a reactive delay; returns `{ pause, resume }`.
- `PersistedState` accepts `null` values and has `connect()`/`disconnect()` for tab sync control.
- Other exports worth knowing before hand-rolling: `watch`, `watchOnce`, `Previous`, `Debounced`, `Throttled`, `ElementSize`, `IsInViewport`, `useIntersectionObserver` (`once` option), `useEventListener`, `onClickOutside`, `PressedKeys`, `IsIdle`, `StateHistory`, `FiniteStateMachine`, `resource`, `Context`.
- `runed/kit` exports are SvelteKit-only. This app is a plain SPA; do not import them.

## Vite 8 (Rolldown + Oxc)

- `frontend/vite.config.ts` uses `defineConfig` with `svelte()`, `tailwindcss()`, `wails("./bindings")` plugins.
- Vite 8 is ESM-only and requires Node >=20.19 or >=22.12 (repo uses Node 22).
- `optimizeDeps.esbuildOptions` and the top-level `esbuild` option are deprecated; Oxc handles transforms. The repo does not use them; do not add esbuild-specific options.
- Aliases: `$lib` -> `src/lib`, `@bindings` -> `frontend/bindings`. `extensionAlias` maps `.js` imports to `.ts`.
- Build target `es2022`, sourcemaps off, chunk splitting via `rollupOptions.output.advancedChunks` (Rolldown still accepts `rollupOptions` for now).
- Dev server is pinned to port 9245 (`WAILS_VITE_PORT`), `strictPort: true`, proxying `/api`, `/rest`, `/health`, `/metrics` to `127.0.0.1:17337`.

## Vitest 4

- Unit tests: `frontend/src/**/*.test.ts`, run with `pnpm test`. Separate configs for contract/acceptance/exploratory/oracle suites.
- Breaking changes vs v3 already in effect here: no `basic` reporter, no `vitest.workspace.ts` (use `test.projects`), `mockReset` resets to a no-op (use `mockRestore` to restore an implementation), `getSourceMap`/`ErrorWithDiff`/`UserConfig` removed (use `TestError`, `ViteUserConfig`).
- Mock pattern used for Wails code: `vi.mock("@bindings/melovian/services/index.js", ...)` and `vi.mock("@wailsio/runtime", ...)`.
- Vitest 5 exists but is not adopted. Do not rely on v5 defaults like `clearMocks: true` or `extends: true` for inline projects.

## TypeScript 6

- `tsconfig` uses the `@tsconfig/svelte` base. TS 6 defaults: `strict` on, `module: esnext`, `target` floats to latest ES, `types` defaults to `[]` so ambient `@types` must be listed explicitly.
- `moduleResolution: node10`/`classic`, `module: amd/umd/systemjs`, `outFile`, `downlevelIteration`, and `esModuleInterop: false` are gone.
- `baseUrl` is not required for `paths` anymore.
- TS 7 (native Go port) exists but is not adopted; `~6.0.3` pin is deliberate.
- Check types with `cd frontend && pnpm check` (svelte-check 4). svelte-check 4 needs `typescript` installed as a peer, which it is.

## ESLint 10 + Prettier

- Flat config only (`frontend/eslint.config.js`). No `.eslintrc`, no `eslint-env` comments, no `--ignore-path`.
- `eslint-plugin-svelte` v3 understands runes; legacy-reactivity rules are skipped in runes mode.
- `prettier-plugin-svelte` v4 requires Prettier 3 + Svelte 5; `svelteStrictMode` and `svelteBracketNewLine` options are gone (use `bracketSameLine`).
- Format with `cd frontend && pnpm format`, lint with `cd frontend && pnpm lint`.

## Other frontend deps in use

- `@iconify/svelte` + `@iconify/json` for icons (`<Icon icon="..." />`).
- `@sentry/svelte` for error reporting (init in `frontend/src/lib/core/sentry.ts`).
- `xxhashjs` for hashing (has a local `src/xxhashjs.d.ts` shim).
- `fast-check` for property tests, `axe-core` for a11y checks, `@lhci/cli` for Lighthouse CI, `@stryker-mutator/*` for mutation tests.
