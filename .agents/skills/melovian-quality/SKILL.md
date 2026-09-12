---
name: melovian-quality
description: Melovian code quality bar and definition of done. Use when writing code, reviewing a diff before finishing, or deciding how much structure a change needs.
---

# Melovian quality bar

The conventions in `.agents/conventions/` are the rules. This skill is the judgement layer: how to apply them and what "done" means.

## Before writing code

- Read the neighboring code first. Match its imports, naming, error style, and abstraction level. A correct change in a foreign style is still wrong here.
- Check `go.mod` / `package.json` before reaching for a library. Prefer what the repo already uses: `runed` utilities before hand-rolled reactive helpers, `bits-ui` primitives before custom overlays, `httputil` helpers before raw `http.Error`.
- Plan the smallest diff that solves the problem. Drive-by refactors, unrelated renames, and reformatting files you did not need to touch all count against the change.

## While writing

- Errors get handled at the right boundary, not every line. Propagate in Go with context (`fmt.Errorf("...: %w", err)`); show once in the UI through `toast`/`confirm` helpers rather than per-call-site alert logic.
- Compact over clever. Collapse duplicate branches, avoid nesting, share existing abstractions. If a helper exists (`bounded-cache`, `detail-cache`, `local-search`), use it.
- No `// TODO` unless the user asked. No dead code, no commented-out blocks, no unused params kept "for later".
- State lives in the right layer: playback in `config/music/*-ops.ts`, feature state in `features/*/store.svelte.ts`, persistence in `internal/store/`. Do not bolt new state onto a component when a store owns that concern.
- Desktop-only code stays behind lazy `await import("@bindings/...")` so server/web mode never loads it.
- User-visible strings follow `.agents/rules/prose.md`: no emdashes, no intensifiers, no filler. Toast and dialog text names the concrete failure ("Could not reach server at 192.168.1.5:4533"), not "Something went wrong".

## Go specifics

- SPDX header on every new file: `// SPDX-FileCopyrightText: 2026 Quad4 Software` / `// SPDX-License-Identifier: Apache-2.0`.
- `gofmt` before commit (lefthook enforces it). golangci-lint runs errcheck, govet (all but fieldalignment/shadow), ineffassign, staticcheck, unused.
- Platform code uses build tags (`//go:build unix`, `//go:build server`) and per-OS files (`*_linux.go`, `*_darwin.go`, `*_windows.go`, `*_stub.go`).
- Package-scoped tests over root tests. Race-clean under `go test -race`.

## Svelte/TS specifics

- Runes only in new code (`$state`, `$derived`, `$effect`, `$props`, `$bindable`, snippets). See `melovian-frontend` for syntax rules.
- Type boundaries: types at module edges (`types.ts` per feature dir), inference inside.
- `svelte-check` (`pnpm check`) must pass. Do not silence with `any` or `@ts-ignore`.

## Definition of done

A change is done when all of these hold:

1. The smallest relevant test passes (see `melovian-testing` for which suite covers the change).
2. Bug fixes landed with a test that failed before the fix.
3. `git diff` shows only what the change needs. No stray formatting, no leftover debug logging, no commented code.
4. New Go service signatures came with regenerated `frontend/bindings/` in the same change.
5. User-facing text passes the prose rules in `.agents/rules/prose.md`.
6. No secrets, tokens, or `.env` values in the diff.
7. For broad changes: `task verify` ran clean, not just `task test`.

## Self-review questions

Before declaring done, re-read the diff and answer:

- Does every changed line trace back to the request?
- What input breaks this? (empty list, offline server, missing tag, duplicate id, unmounted component)
- Is the error path as good as the happy path?
- Would a maintainer reading this file in a year know why this code exists, without the PR?
- Did the change add an abstraction? If yes, does something already in `lib/core/` or `internal/httputil/` do it?

## Reviewing someone else's diff

- Check correctness before style. Style nits that violate `.agents/conventions/` still get called out, but say which rule, not a preference.
- Verify claims against code: run the test, grep the caller, read the generated binding. Do not approve on plausibility.
- Flag missing tests for changed behavior, especially on `config/music/*-ops.ts` and `internal/api/` handlers, which carry the most coverage obligations.
