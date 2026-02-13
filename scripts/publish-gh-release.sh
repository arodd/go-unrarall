#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "usage: $0 <version>" >&2
  echo "example: $0 1.0.0" >&2
  exit 1
fi

raw_version="$1"
version="${raw_version#v}"
tag="v${version}"

if [[ -z "${version}" ]]; then
  echo "version must not be empty" >&2
  exit 1
fi

if ! command -v gh >/dev/null 2>&1; then
  echo "required tool not found: gh" >&2
  exit 1
fi

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
dist_dir="${root_dir}/dist/${tag}"
notes_file="${dist_dir}/RELEASE_NOTES.md"

if [[ ! -d "${dist_dir}" ]]; then
  echo "missing release artifacts directory: ${dist_dir}" >&2
  echo "run ./scripts/build-release.sh ${version} first" >&2
  exit 1
fi

repo="${GH_REPO:-}"
if [[ -z "${repo}" ]]; then
  origin_url="$(git -C "${root_dir}" config --get remote.origin.url || true)"
  if [[ "${origin_url}" =~ github\.com[:/]([^/]+)/([^/.]+)(\.git)?$ ]]; then
    repo="${BASH_REMATCH[1]}/${BASH_REMATCH[2]}"
  fi
fi

if [[ -z "${repo}" ]]; then
  echo "unable to infer GitHub repository from origin." >&2
  echo "set GH_REPO=owner/name and re-run." >&2
  exit 1
fi

if ! gh auth status >/dev/null 2>&1; then
  echo "gh is not authenticated. Run 'gh auth login' first." >&2
  exit 1
fi

assets=(
  "${dist_dir}/go-unrarall_${version}_linux_amd64.tar.gz"
  "${dist_dir}/go-unrarall_${version}_linux_arm64.tar.gz"
  "${dist_dir}/go-unrarall_${version}_darwin_amd64.tar.gz"
  "${dist_dir}/go-unrarall_${version}_darwin_arm64.tar.gz"
  "${dist_dir}/go-unrarall_${version}_windows_amd64.zip"
  "${dist_dir}/SHA256SUMS"
)

for asset in "${assets[@]}"; do
  if [[ ! -f "${asset}" ]]; then
    echo "missing release asset: ${asset}" >&2
    exit 1
  fi
done

if [[ ! -f "${notes_file}" ]]; then
  echo "missing release notes file: ${notes_file}" >&2
  exit 1
fi

if gh release view "${tag}" --repo "${repo}" >/dev/null 2>&1; then
  gh release upload "${tag}" "${assets[@]}" --repo "${repo}" --clobber
else
  gh release create "${tag}" "${assets[@]}" \
    --repo "${repo}" \
    --title "go-unrarall ${tag}" \
    --notes-file "${notes_file}"
fi

echo "github release ${tag} updated in ${repo}"
