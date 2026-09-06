# Security conventions

## Secrets and credentials

- Do not commit secrets, API keys, tokens, passwords, or `.env` values.
- Do not log secrets or credentials, even in debug output.
- Do not expose environment variables or credentials in UI strings, error messages, or URLs.

## Defensive scope

- Help with defensive security only. That includes security analysis, detection rules, vulnerability explanations, defensive tools, and security documentation.
- Do not write or improve code that could be used to attack systems, harvest credentials, or bypass protections.
- Do not assist with bulk crawling for SSH keys, browser cookies, or cryptocurrency wallets.

## Dependency and build hygiene

- Check `go.mod` and `package.json` before adding dependencies. Prefer packages the repo already uses.
- Avoid newly published versions. Prefer a version published at least seven days ago.
- Do not use floating ranges like `latest` or `*` in dependency files.
- Do not modify repository security policies, `.npmrc` settings, or CI compliance controls to work around build failures.
