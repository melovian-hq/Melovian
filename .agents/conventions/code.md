# Code conventions

## License header

Every Go file starts with the SPDX header:

```go
// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0
```

## Comments

- No emdashes in comments or docs. Use a period, comma, or parentheses.
- No semicolons in code comments or doc comments. End one sentence before starting the next.
- Do not add or remove comments unless asked. If you accidentally delete one, put it back.
- Keep comments focused on why something is non-obvious. The code itself is the what.

## Style

- Match the existing style in the file you are editing.
- Small diffs beat drive-by refactors.
- Write compact code. Collapse duplicate else branches, avoid unnecessary nesting, and share abstractions.
- Handle errors at the right boundary. Not every line needs a check.
- Avoid `// TODO` comments unless the user specifically asks for one.
- Do not log or expose secrets, API keys, or credentials.

## Go

- Run `gofmt` or `gofumpt` before committing.
- Check the dependency list in `go.mod` before adding a new package. Prefer the package manager (`go get`) over hand-editing `go.mod`.
- Use build tags for platform-specific files (for example, `//go:build unix`).
- Prefer package-scoped tests like `go test ./internal/...` over `go test .` unless the frontend build is present.

## Svelte and TypeScript

- Use Svelte 5 runes in new code.
- Run `cd frontend && pnpm check` when a change might break types.
- Keep components focused. Route player, queue, and settings logic through `frontend/src/lib/config/music` modules.
- Use `frontend/src/lib/core/http` for fetch, errors, and API paths.
