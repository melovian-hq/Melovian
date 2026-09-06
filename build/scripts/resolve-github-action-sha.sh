#!/usr/bin/env bash
set -euo pipefail

resolve_tag_sha() {
  local repo="$1"
  local tag="$2"
  local data sha type

  data="$(curl -fsSL "https://api.github.com/repos/${repo}/git/ref/tags/${tag}")"
  sha="$(printf '%s' "$data" | jq -r '.object.sha')"
  type="$(printf '%s' "$data" | jq -r '.object.type')"
  if [ "$type" = "tag" ]; then
    sha="$(curl -fsSL "https://api.github.com/repos/${repo}/git/tags/${sha}" | jq -r '.object.sha')"
  fi
  if [ -z "$sha" ] || [ "$sha" = "null" ]; then
    sha="$(curl -fsSL "https://api.github.com/repos/${repo}/commits/${tag}" | jq -r '.sha')"
  fi
  printf '%s %s\n' "$tag" "$sha"
}

echo "Latest GitHub Action release SHAs"
echo

for entry in \
  "actions/checkout" \
  "actions/setup-go" \
  "actions/setup-node" \
  "pnpm/action-setup" \
  "go-task/setup-task"; do
  latest="$(curl -fsSL "https://api.github.com/repos/${entry}/releases/latest" | jq -r '.tag_name')"
  read -r tag sha < <(resolve_tag_sha "$entry" "$latest")
  printf '%-24s %-10s %s\n' "$entry" "$tag" "$sha"
done
