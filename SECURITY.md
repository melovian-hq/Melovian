# Security

Melovian is alpha software. Treat production deploys as early access and keep them updated.

## Supported versions

| Version | Security fixes |
|---------|----------------|
| `0.1.x` (current alpha) | Yes, on a best-effort basis |
| Older / unreleased forks | No |

Only the branch and tags that match the current `0.1.x` line receive patches. There is no LTS track yet.

## How to report a vulnerability

Do not open a public issue for a security problem that could expose credentials, media libraries, or remote code execution.

1. Report privately through your project forge (private issue or security contact), or contact Quad4 Software maintainers directly.
2. Include the Melovian version or git commit, how you run it (desktop, `melovian-server`, or Docker), and steps to reproduce.
3. If you have a patch, attach it. Do not publish the patch until a fix is released or the maintainers say it is safe to disclose.

We aim to acknowledge private reports within 7 days. Fix timing depends on severity and whether a release is already in progress.

## Deploy checklist (server / Docker)

These are the settings that matter most for a multi-user or internet-facing instance:

- Set `MELOVIAN_AUTH_SECRET` to a random string of at least 32 characters.
- Set `MELOVIAN_PUBLIC_URL` to the exact public origin when you sit behind a reverse proxy (needed for streams and OIDC).
- Prefer binding with `MELOVIAN_LISTEN` on a private interface and terminating TLS at the proxy.
- Use `MELOVIAN_ALLOWED_IPS` when only certain networks should reach the API.
- Set `MELOVIAN_TRUST_PROXY=true` only when the proxy strips or overwrites client IP headers correctly. Wrong trust settings let clients spoof IPs past an allowlist.
- Keep OIDC client secrets and Subsonic passwords out of public compose files and git.
- Leave `MELOVIAN_DEBUG_PPROF` unset in production. Profiling endpoints expose process internals.
- Demo mode (`MELOVIAN_DEMO_MODE=true`) is read-only on purpose. Do not mix it with a writeable admin deploy on the same public URL.

Desktop builds store data under the OS data directory (default `~/.local/share/melovian/` on Linux). Protect that directory the same way you protect your music library credentials.

## Scope

In scope: Melovian HTTP API, auth and session handling, share tokens, Subsonic proxy paths, extension install/unzip, Docker image defaults, and desktop IPC that can reach the host filesystem or credentials.

Out of scope for this policy: bugs that only affect your upstream Subsonic or Navidrome server, third-party lyrics providers, or intentionally public demo catalogs.

## Preferential disclosure

If you are unsure whether a finding is security-relevant, still report it privately. We would rather close a false alarm than learn about a real one from a public issue.
