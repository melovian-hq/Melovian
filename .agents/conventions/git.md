# Git conventions

## Rules

- Never update git config.
- Never use `-i` or other interactive flags. Pipes and one-shot commands only.
- Do not push unless the user explicitly asks.
- Do not commit if no changes exist.
- Do not commit secrets, `.env` values, or credentials.

## Making a commit

1. Run `git status`, `git diff`, and `git log` in parallel.
2. Draft a concise commit message that says why the change exists, not what it does.
3. Commit with a here-doc so the message can be reviewed before it lands:

```bash
git commit -m "$(cat <<'EOF'
Your commit message here.
EOF
)"
```

4. If pre-commit hooks modify files and the commit fails, stage the changes and retry.

## Pull requests

Use `gh` for all GitHub operations. Review every commit on the branch, not just the latest. Do not force-push or rewrite history.
