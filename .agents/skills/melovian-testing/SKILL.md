---
name: melovian-testing
description: Melovian test taxonomy, naming conventions, commands, mocking patterns, and coverage gates. Use when writing, running, or debugging tests on either side of the repo.
---

# Melovian testing

This repo tests in layers. Pick the smallest layer that proves the behavior, then run the matching command. A bug fix ships with a failing-then-passing test; a feature ships with unit coverage and, when it touches shared contracts, a contract or acceptance test.

## Test file taxonomy (frontend)

Suffix on `*.test.ts` decides which suite picks it up:

| Suffix | Purpose | Example |
|---|---|---|
| `.test.ts` | Plain unit test (212 files) | `queue-ops.test.ts` |
| `.property.test.ts` | fast-check property/invariant tests | `bounded-cache.property.test.ts` |
| `.contract.test.ts` | Cross-boundary contract checks | `api-routes.contract.test.ts` |
| `.oracle.test.ts` | Expected-output oracles and invariants | `album-dedup.oracle.test.ts` |
| `.acceptance.test.ts` | Feature-level flows through the store | `music-store.connect.acceptance.test.ts` |
| `.exploratory.test.ts` | Generative exploration of edge inputs | `album-dedup.exploratory.test.ts` |
| `.leak.test.ts`, `.memory.test.ts` | Leak and memory regression | `bounded-cache.leak.test.ts` |
| `.perf.test.ts` | Performance smoke | `bounded-cache.perf.test.ts` |
| `.regression.test.ts`, `.crash.test.ts` | Bug/crash regression pins | |
| `.smoke.test.ts`, `.mount.test.ts`, `.hygiene.test.ts` | Component mount/markup sanity, repo-wide source scans | `App.mount.test.ts`, `color-tokens.hygiene.test.ts` |

Naming guidance: default to `.test.ts`. Use `.property` for pure functions with invariants, `.oracle` when a second implementation or hand-computed truth exists, `.acceptance` for multi-step flows through `music.svelte.ts`, `.regression`/`.crash` to pin a fixed bug.

## Frontend commands

```bash
cd frontend
pnpm test                  # all suffixes (vitest.config.ts includes everything)
pnpm test:property         # property + contract + leak + memory + exploratory + oracle
pnpm test:acceptance       # acceptance suite
pnpm test:exploratory      # exploratory suite
pnpm test:oracle           # oracle suite
pnpm test:coverage         # unit suite with v8 coverage gates
pnpm test:e2e              # Playwright (spins up demo server)
pnpm test:e2e:smoke        # only @smoke-tagged specs
pnpm test:mutation         # Stryker
```

Coverage floors live in `frontend/vitest.config.ts`: statements 80 / branches 70 / functions 90 / lines 80, applied to a named list of high-value files (search, playback-queue, album-dedup, smart-playlist, bounded-cache, collection, local-search, router match). Adding a file to that `coverage.include` list opts it into the floor.

## Test environment

- jsdom, `testTimeout: 15000` (dynamic imports stall under parallel transform load; keep the timeout).
- `vitest.setup.ts` already stubs `HTMLMediaElement.play/pause/load`, `matchMedia`, `element.animate`, `scrollIntoView`, pointer capture, `ResizeObserver`, and a fresh `localStorage` per test. Do not re-stub these in test files.
- Aliases `$lib` and `@bindings` work in tests, same as the app.

## Mocking patterns

```ts
// HTTP: stub fetch, restore in afterEach
vi.stubGlobal("fetch", vi.fn().mockResolvedValue(response));
afterEach(() => vi.unstubAllGlobals());

// Wails services and runtime: module mocks
vi.mock("@bindings/melovian/services/index.js", () => ({ MediaService: {...} }));
vi.mock("@wailsio/runtime", () => ({ Events: { On: vi.fn() }, Dialogs: {...} }));
```

- `mockReset` resets to a no-op in Vitest 4. Use `mockRestore` when the original implementation matters.
- Property tests use `fast-check` (`fc.record`, `fc.option`, `fc.nat`, domain arbitraries). See `album-dedup.oracle.test.ts` for the house style.
- The api-routes contract test scans `frontend/src` for `/api/...` literals and diffs them against `internal/api/testdata/api-routes.json`. New frontend API calls must land in `api-paths.ts` and the manifest must match the Go routes.
- Persisted state (`runed` `PersistedState`, storage keys): `lib/settings/nav-state.test.ts` toggles the value, asserts the `localStorage` write, then `vi.resetModules()` plus a fresh dynamic import to prove the stored value survives module recreation. Timed UI state like `SaveStatus` uses `vi.useFakeTimers()` and `advanceTimersByTime`. `save-status.test.ts` also covers a stale timer not resetting an in-progress save.

## Source scanning tests

Two suites read files with node fs instead of importing them:

- `src/lib/components/svelte-markup.hygiene.test.ts` walks every `.svelte` file and fails on `// SPDX` comments placed before `<script>` (they render as text).
- Both hygiene tests share `listSvelteFiles` from `src/test-fixtures/svelte-files.ts`. New repo-wide source scans should reuse it instead of re-implementing the walk.
- `src/lib/theme/color-tokens.hygiene.test.ts` scans every `.svelte` file for hard-coded colors: hex, `rgb()`/`hsl()`/`oklch()`-family functions, named CSS colors inside `<style>`/`style=`/`style:` contexts, and named SVG paint attributes. Components must use `--jb-*` tokens from `lib/theme/tokens.css`. `transparent`, `currentColor`, and `color-mix()` on token vars stay legal. Pre-existing violations live in an `ALLOWLIST` map keyed by src-relative path with a reason per entry. The test fails on new violations and on allowlisted files that no longer exist. Files that come back clean print a console note so the list can be pruned.

## Backend (Go) tests

- Colocated `*_test.go`. Run with `go test ./internal/... ./services/...`. Root `go test .` needs `frontend/dist` built.
- Fuzz targets exist in `cache`, `compat`, `httputil`, `lyrics`, `metaloader`, `navidrome`, `subsonic`, `melog`, `update`. `go test -fuzz=<name> -fuzztime=5s ./internal/<pkg>` smoke-runs one; `task test:fuzz` runs them all.
- Benchmarks: `go test -bench=. -benchmem -count=1` on `cache`, `localmusic`, `subsonic`, `httputil`, `video` (`task bench`).
- `build/scripts/coverage-go.sh` enforces per-package floors: cache 85%, httputil 50%, lyrics 65%, localmusic 70%, subsonic 50%. Lowering a floor to pass CI is not allowed; raise coverage instead.
- `build/scripts/mutation-go.sh` rewrites Go source mutants and runs the package tests; `cd frontend && pnpm test:mutation` runs Stryker (`task test:mutation` does both).
- `go test -race ./...` runs under `task verify`. `testing/synctest` (stable since Go 1.25) is available for deterministic concurrency tests.

## e2e (Playwright)

- Specs in `frontend/e2e/`. The `webServer` block launches `e2e/helpers/run-demo-server.mjs` (server binary in demo mode) unless `E2E_BASE_URL` points at a running instance.
- Projects: `smoke` (specs tagged `@smoke`) and `chromium` (everything else). Build `bin/melovian-server` first (`go build -tags server -o bin/melovian-server .`) or use `task test:e2e` which builds it.
- Dark color scheme, Desktop Chrome, 60s test timeout, trace on first retry.

## Pre-commit and CI gates

- `lefthook.yml` pre-commit: gofmt check, golangci-lint, prettier --check, eslint on staged files. Skip only in emergencies with `LEFTHOOK=0`.
- The pre-merge gate (`task verify`): bindings drift, format check, golangci-lint, `go test -race ./...`, property suite, eslint, svelte-check, vitest. Each part runs standalone (`bash build/scripts/check-bindings-drift.sh`, `bash build/scripts/check-gofmt.sh`, `pnpm lint`, `pnpm check`, `pnpm test:property`).
- `.github/workflows/ci.yml` runs the same shape. CodeQL, Scorecard, and DeepSource also scan the repo.

## What to run before finishing

| Change scope | Minimum verification |
|---|---|
| Go logic | `go test ./<pkg>/...` (or `./internal/... ./services/...` when broad) |
| Frontend logic | `pnpm test` on the touched suite, `pnpm check` if types changed |
| UI component | `pnpm check` + relevant `.test.ts` |
| Service signature | `wails3 generate bindings -clean=true -ts` + `bash build/scripts/bindings-postprocess.sh`, then `bash build/scripts/check-bindings-drift.sh` |
| API route | update `internal/api/testdata/api-routes.json`, run `pnpm test:property` |
| Anything user-visible | `task verify` (or each gate part) before calling it done |
